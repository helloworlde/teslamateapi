package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// TeslaMateAPICarsStatsLifetimeV2 returns since-ownership aggregates over
// drives, charges, and derived parking sessions in a single SQL roundtrip.
//
// `since` is the earliest drive start_date for the car (proxy for
// "ownership begins"). All distance / range fields are converted to user's
// `unit_of_length`; energies stay in kWh; temperatures honour
// `unit_of_temperature`.
//
// Parking aggregates are computed from the same drive-pair window the
// /parkings endpoint uses, so the numbers reconcile.
//
// All aggregates land in one round-trip via parallel CTEs (`d`, `ch`, `dp`,
// `pk`, `up`, `cm`); each CTE walks one source table at most once. The
// outer SELECT anchors on a single-row VALUES so the response is well-formed
// even when the car has no completed drives / charges / parkings yet.
//
// @Summary      Lifetime stats
// @Description  Since-ownership aggregates over drives, charges, parkings, firmware updates, plus static car metadata.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path  int  true  "TeslaMate cars.id"
// @Success      200  {object}  dto.V2LifetimeResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/stats/lifetime [get]
func (h *Handler) StatsLifetime(c *gin.Context) {

	const handler = "TeslaMateAPICarsStatsLifetimeV2"
	var ErrMsg = "Unable to load lifetime stats."

	CarID, ok := respond.RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}

	var (
		CarName                                   NullString
		Since                                     NullString
		recordedDays                              int
		avgDailyDistance, avgMonthlyDistance      float64
		drives                                    dto.V2DrivesAgg
		charges                                   dto.V2ChargesAgg
		parkings                                  dto.V2ParkingsAgg
		updates                                   dto.V2UpdatesAgg
		carMeta                                   dto.V2CarMeta
		UnitsLength, UnitsTemperature             string
	)

	tzName := h.tz.String()

	// One CTE per source table — Postgres flattens trivial CTEs since v12.
	// Anchored on a single-row VALUES so empty cars still produce a well-formed
	// response (otherwise the empty `d` CTE would yield zero rows and the
	// handler would surface ErrNoRows).
	query := `
		WITH d AS (
			SELECT
				MIN(start_date) AS since,
				MAX(start_date) AS last_drive_date,
				MAX(end_km) AS current_odometer,
				COUNT(*) AS cnt,
				COUNT(DISTINCT date_trunc('day', start_date AT TIME ZONE $2)) AS active_days,
				COALESCE(SUM(distance), 0) AS total_km,
				COALESCE(SUM(duration_min), 0) AS total_dur,
				COALESCE(MAX(speed_max), 0) AS max_speed,
				COALESCE(SUM(distance) / NULLIF(SUM(duration_min), 0) * 60, 0) AS avg_speed,
				COALESCE(AVG(distance), 0) AS avg_dist_per_drive,
				COALESCE(AVG(duration_min), 0) AS avg_dur_per_drive,
				COALESCE(MAX(distance), 0) AS longest_km,
				COALESCE(MIN(distance) FILTER (WHERE distance > 0), 0) AS shortest_km,
				COALESCE(MAX(duration_min), 0) AS longest_dur,
				COALESCE(MAX(power_max), 0) AS peak_drive_power,
				COALESCE(-MIN(power_min), 0) AS max_regen_power,
				AVG(outside_temp_avg) AS avg_outside_temp,
				AVG(inside_temp_avg) AS avg_inside_temp,
				COALESCE(SUM(
					CASE WHEN start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency
					ELSE 0 END
				), 0) AS total_kwh,
				CASE WHEN SUM(distance) > 0 THEN
					SUM(
						CASE WHEN start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
						THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency
						ELSE 0 END
					) / NULLIF(SUM(distance), 0) * 1000
				ELSE 0 END AS avg_consumption,
				MIN(
					CASE WHEN distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency / distance * 1000
					ELSE NULL END
				) AS best_consumption,
				MAX(
					CASE WHEN distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency / distance * 1000
					ELSE NULL END
				) AS worst_consumption
			FROM drives
			LEFT JOIN cars ON cars.id = drives.car_id
			WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL
			GROUP BY cars.id
		),
		ch AS (
			SELECT
				COUNT(*) AS cnt,
				COALESCE(SUM(charge_energy_added), 0) AS total_added,
				COALESCE(SUM(GREATEST(charge_energy_used, charge_energy_added)), 0) AS total_used,
				COALESCE(SUM(cost), 0) AS total_cost,
				CASE WHEN SUM(charge_energy_added) > 0
					THEN SUM(cost) / NULLIF(SUM(charge_energy_added), 0)
					ELSE 0 END AS avg_cost_per_kwh,
				COALESCE(AVG(charge_energy_added), 0) AS avg_energy_per_session,
				COALESCE(AVG(duration_min), 0) AS avg_duration_min,
				COUNT(*) FILTER (WHERE fast_present) AS fast_count,
				COUNT(*) FILTER (WHERE NOT fast_present) AS ac_count,
				COALESCE(SUM(charge_energy_added) FILTER (WHERE fast_present), 0) AS fast_energy,
				COALESCE(SUM(charge_energy_added) FILTER (WHERE NOT fast_present), 0) AS ac_energy,
				COALESCE(SUM(charge_energy_added) FILTER (WHERE geofence_id IS NOT NULL), 0) AS geofenced_energy,
				COALESCE(SUM(charge_energy_added) FILTER (WHERE geofence_id IS NULL), 0) AS non_geofenced_energy,
				COALESCE(SUM(charge_energy_added) FILTER (WHERE fast_present AND has_free_supercharging), 0) AS free_sc_energy,
				COALESCE(MAX(peak_power), 0) AS peak_power,
				COALESCE(MAX(peak_voltage), 0) AS peak_voltage,
				COALESCE(MAX(duration_min), 0) AS longest_session_dur,
				COALESCE(MAX(charge_energy_added), 0) AS largest_session_energy,
				COALESCE(MAX(cost), 0) AS max_session_cost,
				COALESCE(AVG(cost) FILTER (WHERE cost IS NOT NULL), 0) AS avg_session_cost,
				COALESCE(AVG(peak_power) FILTER (WHERE NOT fast_present), 0) AS avg_power_ac,
				COALESCE(AVG(peak_power) FILTER (WHERE fast_present), 0) AS avg_power_dc,
				COALESCE(MAX(peak_power) FILTER (WHERE NOT fast_present), 0) AS max_power_ac,
				COALESCE(MAX(peak_power) FILTER (WHERE fast_present), 0) AS max_power_dc,
				MIN(start_battery_level) AS min_start_lvl,
				MAX(end_battery_level) AS max_end_lvl,
				COUNT(DISTINCT COALESCE(geofence_id::text, address_id::text)) AS distinct_locations,
				MIN(start_date) AS first_charge_date,
				MAX(start_date) AS last_charge_date
			FROM (
				SELECT
					cp.charge_energy_added,
					cp.charge_energy_used,
					cp.cost,
					cp.duration_min,
					cp.start_battery_level,
					cp.end_battery_level,
					cp.geofence_id,
					cp.address_id,
					cp.start_date,
					COALESCE(cs.free_supercharging, false) AS has_free_supercharging,
					EXISTS(SELECT 1 FROM charges c WHERE c.charging_process_id = cp.id AND c.fast_charger_present) AS fast_present,
					(SELECT MAX(charger_power) FROM charges c WHERE c.charging_process_id = cp.id) AS peak_power,
					(SELECT MAX(charger_voltage) FROM charges c WHERE c.charging_process_id = cp.id) AS peak_voltage
				FROM charging_processes cp
				LEFT JOIN car_settings cs ON cs.id = cp.car_id
				WHERE cp.car_id = $1 AND cp.end_date IS NOT NULL
			) sub
		),
		dp AS (
			SELECT
				d.id AS drive_id,
				d.end_date AS park_start,
				LEAD(d.start_date) OVER w AS park_end,
				d.end_position_id,
				LEAD(d.start_position_id) OVER w AS next_start_position_id
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			WINDOW w AS (PARTITION BY d.car_id ORDER BY d.start_date ASC)
		),
		pk AS (
			SELECT
				COUNT(*) AS cnt,
				COALESCE(SUM(park_dur)::int, 0) AS total_dur,
				COALESCE(MAX(park_dur)::int, 0) AS longest_dur,
				COALESCE(AVG(park_dur), 0) AS avg_dur,
				COALESCE(SUM(
					CASE
						WHEN sp.rated_battery_range_km IS NOT NULL AND ep.rated_battery_range_km IS NOT NULL
						AND NOT EXISTS(SELECT 1 FROM charging_processes cp
							WHERE cp.car_id = $1 AND cp.start_date >= dp.park_start
							AND (dp.park_end IS NULL OR cp.start_date < dp.park_end))
						THEN GREATEST(sp.rated_battery_range_km - ep.rated_battery_range_km, 0) * cars.efficiency
						ELSE 0
					END
				), 0) AS total_drop
			FROM dp
			CROSS JOIN LATERAL (
				SELECT COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60,
				                EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60) AS park_dur
			) durs
			LEFT JOIN cars ON cars.id = $1
			LEFT JOIN positions sp ON sp.id = dp.end_position_id
			LEFT JOIN positions ep ON ep.id = dp.next_start_position_id
		),
		up AS (
			SELECT
				COUNT(*) AS cnt,
				(array_agg(version ORDER BY start_date ASC) FILTER (WHERE version IS NOT NULL))[1] AS first_version,
				(array_agg(version ORDER BY start_date DESC) FILTER (WHERE version IS NOT NULL))[1] AS latest_version,
				MAX(start_date) AS latest_update_date
			FROM updates
			WHERE car_id = $1 AND end_date IS NOT NULL
		),
		cm AS (
			SELECT vin, model, trim_badging, exterior_color, wheel_type, spoiler_type, efficiency, inserted_at
			FROM cars WHERE id = $1
		),
		rd AS (
			SELECT
				COUNT(DISTINCT day) AS recorded_days,
				COUNT(DISTINCT date_trunc('month', day)) AS recorded_months
			FROM (
				SELECT date_trunc('day', start_date AT TIME ZONE $2) AS day
				FROM drives WHERE car_id = $1 AND end_date IS NOT NULL
				UNION
				SELECT date_trunc('day', start_date AT TIME ZONE $2)
				FROM charging_processes WHERE car_id = $1 AND end_date IS NOT NULL
			) days
		)
		SELECT
			(SELECT name FROM cars WHERE id = $1),
			d.since,
			COALESCE(rd.recorded_days, 0),
			CASE WHEN COALESCE(rd.recorded_days, 0) > 0
				THEN COALESCE(d.total_km, 0) / rd.recorded_days
				ELSE 0 END AS avg_daily_distance,
			CASE WHEN COALESCE(rd.recorded_months, 0) > 0
				THEN COALESCE(d.total_km, 0) / rd.recorded_months
				ELSE 0 END AS avg_monthly_distance,
			COALESCE(d.cnt, 0), COALESCE(d.total_km, 0), COALESCE(d.total_dur, 0), COALESCE(d.total_kwh, 0),
			COALESCE(d.avg_consumption, 0), COALESCE(d.best_consumption, 0), COALESCE(d.worst_consumption, 0),
			COALESCE(d.longest_km, 0), COALESCE(d.shortest_km, 0),
			COALESCE(d.longest_dur, 0),
			COALESCE(d.max_speed, 0), COALESCE(d.avg_speed, 0),
			COALESCE(d.peak_drive_power, 0)::int,
			COALESCE(d.avg_dist_per_drive, 0), COALESCE(d.avg_dur_per_drive, 0),
			COALESCE(d.max_regen_power, 0)::int,
			d.avg_outside_temp, d.avg_inside_temp,
			COALESCE(d.active_days, 0), d.last_drive_date, COALESCE(d.current_odometer, 0),
			COALESCE(ch.cnt, 0), COALESCE(ch.total_added, 0), COALESCE(ch.total_used, 0), COALESCE(ch.total_cost, 0),
			COALESCE(ch.avg_cost_per_kwh, 0), COALESCE(ch.avg_energy_per_session, 0), COALESCE(ch.avg_duration_min, 0),
			COALESCE(ch.fast_count, 0), COALESCE(ch.fast_energy, 0),
			COALESCE(ch.ac_count, 0), COALESCE(ch.ac_energy, 0),
			COALESCE(ch.geofenced_energy, 0), COALESCE(ch.non_geofenced_energy, 0),
			COALESCE(ch.free_sc_energy, 0),
			COALESCE(ch.peak_power, 0), COALESCE(ch.peak_voltage, 0),
			COALESCE(ch.longest_session_dur, 0), COALESCE(ch.largest_session_energy, 0),
			COALESCE(ch.max_session_cost, 0), COALESCE(ch.avg_session_cost, 0),
			COALESCE(ch.avg_power_ac, 0), COALESCE(ch.avg_power_dc, 0),
			COALESCE(ch.max_power_ac, 0)::int, COALESCE(ch.max_power_dc, 0)::int,
			ch.min_start_lvl, ch.max_end_lvl,
			COALESCE(ch.distinct_locations, 0),
			ch.first_charge_date, ch.last_charge_date,
			COALESCE(pk.cnt, 0), COALESCE(pk.total_dur, 0), COALESCE(pk.avg_dur, 0), COALESCE(pk.longest_dur, 0), COALESCE(pk.total_drop, 0),
			COALESCE(up.cnt, 0), up.first_version, up.latest_version, up.latest_update_date,
			cm.vin, cm.model, cm.trim_badging, cm.exterior_color, cm.wheel_type, cm.spoiler_type, cm.efficiency, cm.inserted_at,
			(SELECT unit_of_length FROM settings LIMIT 1),
			(SELECT unit_of_temperature FROM settings LIMIT 1)
		FROM (VALUES (1)) anchor(_)
		LEFT JOIN d ON true
		LEFT JOIN ch ON true
		LEFT JOIN pk ON true
		LEFT JOIN up ON true
		LEFT JOIN cm ON true
		LEFT JOIN rd ON true;`

	row := h.db.QueryRowContext(c.Request.Context(), query, CarID, tzName)
	err := row.Scan(
		&CarName,
		&Since,
		&recordedDays, &avgDailyDistance, &avgMonthlyDistance,
		// drives
		&drives.Count, &drives.TotalDistance, &drives.TotalDurationMin, &drives.TotalEnergyConsumedKWh,
		&drives.AvgConsumption, &drives.BestConsumption, &drives.WorstConsumption,
		&drives.LongestDistance, &drives.ShortestDistance,
		&drives.LongestDurationMin,
		&drives.MaxSpeed, &drives.AvgSpeed,
		&drives.PeakDrivePowerKW,
		&drives.AvgDistancePerDrive, &drives.AvgDurationPerDrive,
		&drives.MaxRegenPower,
		&drives.AvgOutsideTemp, &drives.AvgInsideTemp,
		&drives.ActiveDays, &drives.LastDriveDate, &drives.CurrentOdometer,
		// charges
		&charges.Count, &charges.TotalEnergyAddedKWh, &charges.TotalEnergyUsedKWh, &charges.TotalCost,
		&charges.AvgCostPerKWh, &charges.AvgEnergyPerSession, &charges.AvgDurationMin,
		&charges.FastChargeCount, &charges.FastChargeEnergyKWh,
		&charges.ACChargeCount, &charges.ACChargeEnergyKWh,
		&charges.GeofencedChargeEnergyKWh, &charges.NonGeofencedChargeEnergyKWh,
		&charges.FreeSuperchargingKWh,
		&charges.PeakPowerMaxKW, &charges.PeakVoltageMax,
		&charges.LongestSessionDurationMin, &charges.LargestSessionEnergyKWh,
		&charges.MaxSessionCost, &charges.AvgSessionCost,
		&charges.AvgPowerACKW, &charges.AvgPowerDCKW,
		&charges.MaxPowerACKW, &charges.MaxPowerDCKW,
		&charges.MinStartBatteryLevel, &charges.MaxEndBatteryLevel,
		&charges.DistinctChargeLocations,
		&charges.FirstChargeDate, &charges.LastChargeDate,
		// parkings
		&parkings.Count, &parkings.TotalDurationMin, &parkings.AvgDurationMin, &parkings.LongestParkingMin, &parkings.TotalEnergyDropKWh,
		// updates
		&updates.Count, &updates.FirstVersion, &updates.LatestVersion, &updates.LatestUpdateDate,
		// car meta
		&carMeta.Vin, &carMeta.Model, &carMeta.TrimBadging, &carMeta.ExteriorColor, &carMeta.WheelType, &carMeta.SpoilerType, &carMeta.Efficiency, &carMeta.InsertedAt,
		// units
		&UnitsLength, &UnitsTemperature,
	)
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	// Unit conversion. avg_consumption / best_consumption are Wh/(distance);
	// converting from Wh/km to Wh/mi means *dividing* by 1.609344 (energy
	// stays the same, distance units stretch). convert.KilometersToMiles
	// handles the distance direction (multiplies by 0.62137...) — perfect.
	if UnitsLength == "mi" {
		drives.TotalDistance = convert.KilometersToMiles(drives.TotalDistance)
		drives.LongestDistance = convert.KilometersToMiles(drives.LongestDistance)
		drives.ShortestDistance = convert.KilometersToMiles(drives.ShortestDistance)
		drives.AvgDistancePerDrive = convert.KilometersToMiles(drives.AvgDistancePerDrive)
		drives.CurrentOdometer = convert.KilometersToMiles(drives.CurrentOdometer)
		drives.MaxSpeed = convert.KilometersToMilesInteger(drives.MaxSpeed)
		drives.AvgSpeed = convert.KilometersToMiles(drives.AvgSpeed)
		drives.AvgConsumption = drives.AvgConsumption / 0.62137119223733
		drives.BestConsumption = drives.BestConsumption / 0.62137119223733
		drives.WorstConsumption = drives.WorstConsumption / 0.62137119223733
		avgDailyDistance = convert.KilometersToMiles(avgDailyDistance)
		avgMonthlyDistance = convert.KilometersToMiles(avgMonthlyDistance)
	}
	if UnitsTemperature == "F" {
		if drives.AvgOutsideTemp.Valid {
			drives.AvgOutsideTemp.Float64 = convert.CelsiusToFahrenheit(drives.AvgOutsideTemp.Float64)
		}
		if drives.AvgInsideTemp.Valid {
			drives.AvgInsideTemp.Float64 = convert.CelsiusToFahrenheit(drives.AvgInsideTemp.Float64)
		}
	}

	if len(Since) > 0 {
		Since = NullString(h.timeInTZ(string(Since)))
	}
	if len(drives.LastDriveDate) > 0 {
		drives.LastDriveDate = NullString(h.timeInTZ(string(drives.LastDriveDate)))
	}
	if len(charges.FirstChargeDate) > 0 {
		charges.FirstChargeDate = NullString(h.timeInTZ(string(charges.FirstChargeDate)))
	}
	if len(charges.LastChargeDate) > 0 {
		charges.LastChargeDate = NullString(h.timeInTZ(string(charges.LastChargeDate)))
	}
	if len(updates.LatestUpdateDate) > 0 {
		updates.LatestUpdateDate = NullString(h.timeInTZ(string(updates.LatestUpdateDate)))
	}
	if len(carMeta.InsertedAt) > 0 {
		carMeta.InsertedAt = NullString(h.timeInTZ(string(carMeta.InsertedAt)))
	}

	respond.HandleSuccess(c, handler, dto.V2LifetimeResponse{
		Data: dto.V2Lifetime{
			Car:                dto.Car{CarID: CarID, CarName: CarName},
			CarMeta:            carMeta,
			Since:              Since,
			RecordedDays:       recordedDays,
			AvgDailyDistance:   avgDailyDistance,
			AvgMonthlyDistance: avgMonthlyDistance,
			Drives:             drives,
			Charges:            charges,
			Parkings:           parkings,
			Updates:            updates,
			Units: dto.TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	})
}
