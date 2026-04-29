package main

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func TeslaMateAPICarsDashboardV2(c *gin.Context) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid dashboard range", map[string]any{"reason": err.Error()})
		return
	}
	warnings := []any{}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsDashboardV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	driveSummary, err := fetchDriveHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load dashboard drive summary", map[string]any{"reason": err.Error()})
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load dashboard charge summary", map[string]any{"reason": err.Error()})
		return
	}
	statistics, err := fetchStatisticsSummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, ctx.UnitsTemperature, driveSummary, chargeSummary)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load dashboard statistics", map[string]any{"reason": err.Error()})
		return
	}
	calendarItems, calendarSummary, calendarWarnings, err := fetchUnifiedCalendar(ctx.CarID, startUTC, endUTC, "day", false, false)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load dashboard calendar", map[string]any{"reason": err.Error()})
		return
	}
	warnings = append(warnings, calendarWarnings...)
	current, currentErr := fetchDashboardCurrentSnapshot(ctx.CarID, ctx.UnitsLength, ctx.UnitsTemperature)
	if currentErr != nil {
		warnings = append(warnings, nonFatalWarning("current_snapshot_unavailable", "failed to load current vehicle snapshot", nil, currentErr))
		current = map[string]any{}
	}
	recentDrives, recentDriveErr := fetchDashboardRecentDrives(ctx.CarID, ctx.UnitsLength, ctx.UnitsTemperature, 5)
	if recentDriveErr != nil {
		warnings = append(warnings, nonFatalWarning("recent_drives_unavailable", "failed to load recent drives", nil, recentDriveErr))
		recentDrives = []map[string]any{}
	}
	recentCharges, recentChargeErr := fetchDashboardRecentCharges(ctx.CarID, ctx.UnitsLength, ctx.UnitsTemperature, 5)
	if recentChargeErr != nil {
		warnings = append(warnings, nonFatalWarning("recent_charges_unavailable", "failed to load recent charges", nil, recentChargeErr))
		recentCharges = []map[string]any{}
	}
	recentUpdates, updateErr := fetchDashboardRecentUpdates(ctx.CarID, 3)
	if updateErr != nil {
		warnings = append(warnings, nonFatalWarning("recent_updates_unavailable", "failed to load recent updates", nil, updateErr))
		recentUpdates = []map[string]any{}
	}
	series, seriesWarnings := fetchDashboardMetricSeries(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	warnings = append(warnings, seriesWarnings...)
	distributions, distributionWarnings := fetchDashboardDistributions(ctx.CarID, startUTC, endUTC)
	warnings = append(warnings, distributionWarnings...)
	insights, insightWarnings := buildSimpleInsights(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, nil, 6)
	warnings = append(warnings, insightWarnings...)
	data := map[string]any{
		"car_id":     ctx.CarID,
		"range":      buildRangeDTO(dr),
		"current":    current,
		"statistics": statistics,
		"calendar": map[string]any{
			"bucket":  "day",
			"summary": calendarSummary,
			"items":   calendarItems,
		},
		"series":         series,
		"distributions":  distributions,
		"insights":       insights,
		"recent_drives":  recentDrives,
		"recent_charges": recentCharges,
		"recent_updates": recentUpdates,
	}
	writeV1Object(c, data, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func fetchDashboardCurrentSnapshot(carID int, unitsLength, unitsTemperature string) (map[string]any, error) {
	query := `
		WITH latest_position AS (
			SELECT date, latitude, longitude, speed, power, odometer, battery_level, usable_battery_level,
				rated_battery_range_km, ideal_battery_range_km, outside_temp, inside_temp, elevation
			FROM positions
			WHERE car_id = $1
			ORDER BY date DESC
			LIMIT 1
		),
		latest_state AS (
			SELECT state::text, start_date, end_date
			FROM states
			WHERE car_id = $1
			ORDER BY start_date DESC
			LIMIT 1
		),
		latest_charge AS (
			SELECT id, start_date, end_date, charge_energy_added, charge_energy_used, cost,
				start_battery_level, end_battery_level
			FROM charging_processes
			WHERE car_id = $1
			ORDER BY end_date IS NULL DESC, start_date DESC
			LIMIT 1
		)
		SELECT
			(SELECT date FROM latest_position),
			(SELECT latitude FROM latest_position),
			(SELECT longitude FROM latest_position),
			(SELECT speed FROM latest_position),
			(SELECT power FROM latest_position),
			(SELECT odometer FROM latest_position),
			(SELECT battery_level FROM latest_position),
			(SELECT usable_battery_level FROM latest_position),
			(SELECT rated_battery_range_km FROM latest_position),
			(SELECT ideal_battery_range_km FROM latest_position),
			(SELECT outside_temp FROM latest_position),
			(SELECT inside_temp FROM latest_position),
			(SELECT elevation FROM latest_position),
			(SELECT state FROM latest_state),
			(SELECT start_date FROM latest_state),
			(SELECT end_date FROM latest_state),
			(SELECT id FROM latest_charge),
			(SELECT start_date FROM latest_charge),
			(SELECT end_date FROM latest_charge),
			(SELECT charge_energy_added FROM latest_charge),
			(SELECT charge_energy_used FROM latest_charge),
			(SELECT cost FROM latest_charge),
			(SELECT start_battery_level FROM latest_charge),
			(SELECT end_battery_level FROM latest_charge)`
	var (
		positionDate                                          sql.NullString
		latitude, longitude, odometer, ratedRange, idealRange sql.NullFloat64
		outsideTemp, insideTemp, elevation                    sql.NullFloat64
		speed, power, batteryLevel, usableBatteryLevel        sql.NullInt64
		state, stateStart, stateEnd                           sql.NullString
		chargeID, chargeStartBattery, chargeEndBattery        sql.NullInt64
		chargeStart, chargeEnd                                sql.NullString
		chargeEnergyAdded, chargeEnergyUsed, chargeCost       sql.NullFloat64
	)
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(queryCtx, query, carID).Scan(
		&positionDate,
		&latitude,
		&longitude,
		&speed,
		&power,
		&odometer,
		&batteryLevel,
		&usableBatteryLevel,
		&ratedRange,
		&idealRange,
		&outsideTemp,
		&insideTemp,
		&elevation,
		&state,
		&stateStart,
		&stateEnd,
		&chargeID,
		&chargeStart,
		&chargeEnd,
		&chargeEnergyAdded,
		&chargeEnergyUsed,
		&chargeCost,
		&chargeStartBattery,
		&chargeEndBattery,
	); err != nil {
		return nil, err
	}
	if strings.EqualFold(unitsLength, "mi") {
		odometer = kilometersToMilesSqlNullFloat64(odometer)
		ratedRange = kilometersToMilesSqlNullFloat64(ratedRange)
		idealRange = kilometersToMilesSqlNullFloat64(idealRange)
		speed = kilometersToMilesSqlNullInt64(speed)
	}
	if strings.EqualFold(unitsTemperature, "f") {
		if outsideTemp.Valid {
			outsideTemp.Float64 = celsiusToFahrenheit(outsideTemp.Float64)
		}
		if insideTemp.Valid {
			insideTemp.Float64 = celsiusToFahrenheit(insideTemp.Float64)
		}
	}
	isCharging := chargeID.Valid && !chargeEnd.Valid
	return map[string]any{
		"position": map[string]any{
			"time":                 timeZoneStringPointer(positionDate),
			"latitude":             floatPointer(latitude),
			"longitude":            floatPointer(longitude),
			"speed":                intPointer(speed),
			"power":                intPointer(power),
			"odometer":             floatPointer(odometer),
			"battery_level":        intPointer(batteryLevel),
			"usable_battery_level": intPointer(usableBatteryLevel),
			"rated_range":          floatPointer(ratedRange),
			"ideal_range":          floatPointer(idealRange),
			"outside_temperature":  floatPointer(outsideTemp),
			"inside_temperature":   floatPointer(insideTemp),
			"elevation":            floatPointer(elevation),
		},
		"state": map[string]any{
			"name":       stringPointer(state),
			"start_time": timeZoneStringPointer(stateStart),
			"end_time":   timeZoneStringPointer(stateEnd),
		},
		"charge": map[string]any{
			"charge_id":           intPointer(chargeID),
			"is_charging":         isCharging,
			"start_time":          timeZoneStringPointer(chargeStart),
			"end_time":            timeZoneStringPointer(chargeEnd),
			"energy_added":        floatPointer(chargeEnergyAdded),
			"energy_used":         floatPointer(chargeEnergyUsed),
			"cost":                floatPointer(chargeCost),
			"start_battery_level": intPointer(chargeStartBattery),
			"end_battery_level":   intPointer(chargeEndBattery),
		},
	}, nil
}

func fetchDashboardRecentDrives(carID int, unitsLength, unitsTemperature string, limit int) ([]map[string]any, error) {
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, `
		SELECT drives.id, drives.start_date, drives.end_date,
			COALESCE(start_geofence.name, CONCAT_WS(', ', COALESCE(start_address.name, NULLIF(CONCAT_WS(' ', start_address.road, start_address.house_number), '')), start_address.city)) AS start_address,
			COALESCE(end_geofence.name, CONCAT_WS(', ', COALESCE(end_address.name, NULLIF(CONCAT_WS(' ', end_address.road, end_address.house_number), '')), end_address.city)) AS end_address,
			GREATEST(COALESCE(drives.distance, 0), 0)::float8,
			GREATEST(COALESCE(drives.duration_min, 0), 0)::int,
			drives.speed_max,
			CASE WHEN drives.duration_min > 0 THEN drives.distance / (drives.duration_min::float8 / 60.0) ELSE NULL END::float8,
			CASE WHEN (drives.start_rated_range_km - drives.end_rated_range_km) > 0 THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency ELSE NULL END::float8,
			CASE WHEN drives.distance > 0 AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0 THEN ((drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / drives.distance) * 1000 ELSE NULL END::float8,
			drives.outside_temp_avg
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		LEFT JOIN addresses start_address ON start_address.id = drives.start_address_id
		LEFT JOIN addresses end_address ON end_address.id = drives.end_address_id
		LEFT JOIN geofences start_geofence ON start_geofence.id = drives.start_geofence_id
		LEFT JOIN geofences end_geofence ON end_geofence.id = drives.end_geofence_id
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL
		ORDER BY drives.start_date DESC
		LIMIT $2`, carID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var (
			id                  int
			startDate, endDate  string
			startAddress        sql.NullString
			endAddress          sql.NullString
			distance            float64
			duration            int
			maxSpeed            sql.NullInt64
			avgSpeed            sql.NullFloat64
			energy, consumption sql.NullFloat64
			outsideTemp         sql.NullFloat64
		)
		if err := rows.Scan(&id, &startDate, &endDate, &startAddress, &endAddress, &distance, &duration, &maxSpeed, &avgSpeed, &energy, &consumption, &outsideTemp); err != nil {
			return nil, err
		}
		if strings.EqualFold(unitsLength, "mi") {
			distance = kilometersToMiles(distance)
			maxSpeed = kilometersToMilesSqlNullInt64(maxSpeed)
			avgSpeed = kilometersToMilesSqlNullFloat64(avgSpeed)
			consumption = whPerKmToWhPerMiNull(consumption)
		}
		if strings.EqualFold(unitsTemperature, "f") && outsideTemp.Valid {
			outsideTemp.Float64 = celsiusToFahrenheit(outsideTemp.Float64)
		}
		items = append(items, map[string]any{
			"drive_id":            id,
			"start_time":          getTimeInTimeZone(startDate),
			"end_time":            getTimeInTimeZone(endDate),
			"start_address":       stringPointer(startAddress),
			"end_address":         stringPointer(endAddress),
			"distance":            distance,
			"duration_seconds":    duration * 60,
			"max_speed":           intPointer(maxSpeed),
			"average_speed":       floatPointer(avgSpeed),
			"energy_used":         floatPointer(energy),
			"consumption_net":     floatPointer(consumption),
			"outside_temperature": floatPointer(outsideTemp),
		})
	}
	return items, rows.Err()
}

func fetchDashboardRecentCharges(carID int, unitsLength, unitsTemperature string, limit int) ([]map[string]any, error) {
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, `
		SELECT charging_processes.id, charging_processes.start_date, charging_processes.end_date,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, NULLIF(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS location,
			GREATEST(COALESCE(charging_processes.duration_min, 0), 0)::int,
			charging_processes.charge_energy_added,
			charging_processes.charge_energy_used,
			charging_processes.cost,
			charging_processes.start_battery_level,
			charging_processes.end_battery_level,
			charging_processes.start_rated_range_km,
			charging_processes.end_rated_range_km,
			charging_processes.outside_temp_avg,
			NULLIF(MAX(charges.fast_charger_type), '') AS charger_type
		FROM charging_processes
		LEFT JOIN addresses address ON address.id = charging_processes.address_id
		LEFT JOIN geofences geofence ON geofence.id = charging_processes.geofence_id
		LEFT JOIN charges ON charges.charging_process_id = charging_processes.id
		WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL
		GROUP BY charging_processes.id, address.name, address.road, address.house_number, address.city, geofence.name
		ORDER BY charging_processes.start_date DESC
		LIMIT $2`, carID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var (
			id                                int
			startDate, endDate                string
			location, chargerType             sql.NullString
			duration                          int
			energyAdded, energyUsed, cost     sql.NullFloat64
			startBattery, endBattery          sql.NullInt64
			startRange, endRange, outsideTemp sql.NullFloat64
		)
		if err := rows.Scan(&id, &startDate, &endDate, &location, &duration, &energyAdded, &energyUsed, &cost, &startBattery, &endBattery, &startRange, &endRange, &outsideTemp, &chargerType); err != nil {
			return nil, err
		}
		if strings.EqualFold(unitsLength, "mi") {
			startRange = kilometersToMilesSqlNullFloat64(startRange)
			endRange = kilometersToMilesSqlNullFloat64(endRange)
		}
		if strings.EqualFold(unitsTemperature, "f") && outsideTemp.Valid {
			outsideTemp.Float64 = celsiusToFahrenheit(outsideTemp.Float64)
		}
		efficiency := sql.NullFloat64{}
		if energyAdded.Valid && energyUsed.Valid && energyUsed.Float64 > 0 {
			efficiency.Valid = true
			efficiency.Float64 = energyAdded.Float64 / energyUsed.Float64
		}
		items = append(items, map[string]any{
			"charge_id":           id,
			"start_time":          getTimeInTimeZone(startDate),
			"end_time":            getTimeInTimeZone(endDate),
			"location":            stringPointer(location),
			"charger_type":        stringPointer(chargerType),
			"duration_seconds":    duration * 60,
			"energy_added":        floatPointer(energyAdded),
			"energy_used":         floatPointer(energyUsed),
			"cost":                floatPointer(cost),
			"charging_efficiency": floatPointer(efficiency),
			"start_battery_level": intPointer(startBattery),
			"end_battery_level":   intPointer(endBattery),
			"start_rated_range":   floatPointer(startRange),
			"end_rated_range":     floatPointer(endRange),
			"outside_temperature": floatPointer(outsideTemp),
		})
	}
	return items, rows.Err()
}

func fetchDashboardRecentUpdates(carID int, limit int) ([]map[string]any, error) {
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, `
		SELECT id, start_date, end_date, version
		FROM updates
		WHERE car_id = $1 AND version IS NOT NULL
		ORDER BY start_date DESC
		LIMIT $2`, carID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id int
		var startDate string
		var endDate sql.NullString
		var version sql.NullString
		if err := rows.Scan(&id, &startDate, &endDate, &version); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"update_id":  id,
			"start_time": getTimeInTimeZone(startDate),
			"end_time":   timeZoneStringPointer(endDate),
			"version":    stringPointer(version),
		})
	}
	return items, rows.Err()
}
