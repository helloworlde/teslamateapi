package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsStatsLifetimeV2 returns since-ownership aggregates over
// drives, charges, and derived parking sessions in a single SQL roundtrip.
//
// `since` is the earliest drive start_date for the car (proxy for
// "ownership begins"). All distance / range fields are converted to user's
// `unit_of_length`; energies stay in kWh; temperatures already aren't
// surfaced here.
//
// Parking aggregates are computed from the same drive-pair window the
// /parkings endpoint uses, so the numbers reconcile.
//
// @Summary      Lifetime stats
// @Description  Since-ownership aggregates over drives, charges, and derived parking sessions.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path  int  true  "TeslaMate cars.id"
// @Success      200  {object}  dto.V2LifetimeResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/stats/lifetime [get]
func TeslaMateAPICarsStatsLifetimeV2(c *gin.Context) {

	const handler = "TeslaMateAPICarsStatsLifetimeV2"
	var ErrMsg = "Unable to load lifetime stats."

	CarID, ok := v2RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}

	type DrivesAgg struct {
		Count                  int     `json:"count"`
		TotalDistance          float64 `json:"total_distance"`
		TotalDurationMin       int     `json:"total_duration_min"`
		TotalEnergyConsumedKWh float64 `json:"total_energy_consumed_kwh"`
		AvgConsumption         float64 `json:"avg_consumption"`        // Wh per (km|mi)
		BestConsumption        float64 `json:"best_consumption"`
		LongestDistance        float64 `json:"longest_distance"`
		MaxSpeed               int     `json:"max_speed"`
	}
	type ChargesAgg struct {
		Count                int     `json:"count"`
		TotalEnergyAddedKWh  float64 `json:"total_energy_added_kwh"`
		TotalEnergyUsedKWh   float64 `json:"total_energy_used_kwh"`
		TotalCost            float64 `json:"total_cost"`
		FastChargeCount      int     `json:"fast_charge_count"`
		FastChargeEnergyKWh  float64 `json:"fast_charge_energy_kwh"`
		PeakPowerMaxKW       int     `json:"peak_power_max_kw"`
	}
	type ParkingsAgg struct {
		Count               int     `json:"count"`
		TotalDurationMin    int     `json:"total_duration_min"`
		TotalEnergyDropKWh  float64 `json:"total_vampire_drain_kwh"`
	}
	type Car struct {
		CarID   int        `json:"car_id"`
		CarName NullString `json:"car_name"`
	}
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`
		UnitsTemperature string `json:"unit_of_temperature"`
	}
	type Lifetime struct {
		Car      Car            `json:"car"`
		Since    NullString     `json:"since"`
		Drives   DrivesAgg      `json:"drives"`
		Charges  ChargesAgg     `json:"charges"`
		Parkings ParkingsAgg    `json:"parkings"`
		Units    TeslaMateUnits `json:"units"`
	}
	type JSONData struct {
		Data Lifetime `json:"data"`
	}

	var (
		CarName                       NullString
		Since                         NullString
		drives                        DrivesAgg
		charges                       ChargesAgg
		parkings                      ParkingsAgg
		UnitsLength, UnitsTemperature string
	)

	// One CTE per aggregate. Postgres flattens trivial CTEs since v12.
	// The outer SELECT anchors on a single-row VALUES so the response is well-formed
	// even when the car has no completed drives / charges / parkings yet (otherwise
	// the empty `d` CTE would yield zero rows and the handler would return ErrNoRows).
	query := `
		WITH d AS (
			SELECT
				MIN(start_date) AS since,
				COUNT(*) AS cnt,
				COALESCE(SUM(distance), 0) AS total_km,
				COALESCE(SUM(duration_min), 0) AS total_dur,
				COALESCE(MAX(speed_max), 0) AS max_speed,
				COALESCE(MAX(distance), 0) AS longest_km,
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
				) AS best_consumption
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
				COUNT(*) FILTER (WHERE fast_present) AS fast_count,
				COALESCE(SUM(charge_energy_added) FILTER (WHERE fast_present), 0) AS fast_energy,
				COALESCE(MAX(peak_power), 0) AS peak_power
			FROM (
				SELECT
					cp.charge_energy_added,
					cp.charge_energy_used,
					cp.cost,
					EXISTS(SELECT 1 FROM charges c WHERE c.charging_process_id = cp.id AND c.fast_charger_present) AS fast_present,
					(SELECT MAX(charger_power) FROM charges c WHERE c.charging_process_id = cp.id) AS peak_power
				FROM charging_processes cp
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
				COALESCE(SUM(
					COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60, EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60)
				)::int, 0) AS total_dur,
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
			LEFT JOIN cars ON cars.id = $1
			LEFT JOIN positions sp ON sp.id = dp.end_position_id
			LEFT JOIN positions ep ON ep.id = dp.next_start_position_id
		)
		SELECT
			(SELECT name FROM cars WHERE id = $1),
			d.since,
			COALESCE(d.cnt, 0), COALESCE(d.total_km, 0), COALESCE(d.total_dur, 0), COALESCE(d.total_kwh, 0),
			COALESCE(d.avg_consumption, 0), COALESCE(d.best_consumption, 0), COALESCE(d.longest_km, 0), COALESCE(d.max_speed, 0),
			COALESCE(ch.cnt, 0), COALESCE(ch.total_added, 0), COALESCE(ch.total_used, 0), COALESCE(ch.total_cost, 0),
			COALESCE(ch.fast_count, 0), COALESCE(ch.fast_energy, 0), COALESCE(ch.peak_power, 0),
			COALESCE(pk.cnt, 0), COALESCE(pk.total_dur, 0), COALESCE(pk.total_drop, 0),
			(SELECT unit_of_length FROM settings LIMIT 1),
			(SELECT unit_of_temperature FROM settings LIMIT 1)
		FROM (VALUES (1)) anchor(_)
		LEFT JOIN d ON true
		LEFT JOIN ch ON true
		LEFT JOIN pk ON true;`

	row := db.QueryRowContext(c.Request.Context(), query, CarID)
	err := row.Scan(
		&CarName,
		&Since,
		&drives.Count, &drives.TotalDistance, &drives.TotalDurationMin, &drives.TotalEnergyConsumedKWh, &drives.AvgConsumption, &drives.BestConsumption, &drives.LongestDistance, &drives.MaxSpeed,
		&charges.Count, &charges.TotalEnergyAddedKWh, &charges.TotalEnergyUsedKWh, &charges.TotalCost, &charges.FastChargeCount, &charges.FastChargeEnergyKWh, &charges.PeakPowerMaxKW,
		&parkings.Count, &parkings.TotalDurationMin, &parkings.TotalEnergyDropKWh,
		&UnitsLength,
		&UnitsTemperature,
	)
	if err != nil {
		v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	// Unit conversion. avg_consumption / best_consumption are Wh/(distance);
	// converting from Wh/km to Wh/mi means *dividing* by 1.609344 (energy
	// stays the same, distance units stretch). kilometersToMiles handles
	// that direction (multiplies by 0.62137...) — perfect.
	if UnitsLength == "mi" {
		drives.TotalDistance = kilometersToMiles(drives.TotalDistance)
		drives.LongestDistance = kilometersToMiles(drives.LongestDistance)
		drives.MaxSpeed = kilometersToMilesInteger(drives.MaxSpeed)
		drives.AvgConsumption = drives.AvgConsumption / 0.62137119223733 // Wh/km → Wh/mi means *more* per unit, since 1 mi > 1 km
		drives.BestConsumption = drives.BestConsumption / 0.62137119223733
	}

	if len(Since) > 0 {
		Since = NullString(getTimeInTimeZone(string(Since)))
	}

	TeslaMateAPIHandleSuccessResponse(c, handler, JSONData{
		Data: Lifetime{
			Car:      Car{CarID: CarID, CarName: CarName},
			Since:    Since,
			Drives:   drives,
			Charges:  charges,
			Parkings: parkings,
			Units: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	})
}
