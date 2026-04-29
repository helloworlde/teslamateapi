package main

import (
	"database/sql"
	"strings"
)

func fetchDriveDetailsPayload(carID int, driveID int, unitsLength, unitsTemperature string) (map[string]any, error) {
	query := `
		SELECT
			drives.id,
			drives.start_date,
			drives.end_date,
			COALESCE(start_geofence.name, CONCAT_WS(', ', COALESCE(start_address.name, NULLIF(CONCAT_WS(' ', start_address.road, start_address.house_number), '')), start_address.city)) AS start_address,
			COALESCE(end_geofence.name, CONCAT_WS(', ', COALESCE(end_address.name, NULLIF(CONCAT_WS(' ', end_address.road, end_address.house_number), '')), end_address.city)) AS end_address,
			GREATEST(COALESCE(drives.distance, 0), 0)::float8 AS distance,
			GREATEST(COALESCE(drives.duration_min, 0), 0)::int AS duration_min,
			COALESCE(drives.speed_max, 0)::int AS speed_max,
			CASE WHEN drives.duration_min > 0 THEN (GREATEST(COALESCE(drives.distance, 0), 0) / (drives.duration_min::float8 / 60.0)) ELSE NULL END AS speed_avg,
			CASE WHEN (drives.start_rated_range_km - drives.end_rated_range_km) > 0 THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency ELSE NULL END AS energy_used,
			CASE WHEN GREATEST(COALESCE(drives.distance, 0), 0) > 0 AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0 THEN ((drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / NULLIF(drives.distance, 0)) * 1000 ELSE NULL END AS consumption_net,
			drives.outside_temp_avg,
			COALESCE(start_position.elevation, end_position.elevation)::float8 AS elevation,
			start_position.odometer,
			end_position.odometer,
			COALESCE(start_position.usable_battery_level, start_position.battery_level)::int AS start_battery_level,
			COALESCE(end_position.usable_battery_level, end_position.battery_level)::int AS end_battery_level
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		LEFT JOIN addresses start_address ON start_address.id = drives.start_address_id
		LEFT JOIN addresses end_address ON end_address.id = drives.end_address_id
		LEFT JOIN geofences start_geofence ON start_geofence.id = drives.start_geofence_id
		LEFT JOIN geofences end_geofence ON end_geofence.id = drives.end_geofence_id
		LEFT JOIN positions start_position ON start_position.id = drives.start_position_id
		LEFT JOIN positions end_position ON end_position.id = drives.end_position_id
		WHERE drives.car_id = $1 AND drives.id = $2 AND drives.end_date IS NOT NULL`
	var (
		id             int
		startDate      string
		endDate        string
		startAddress   string
		endAddress     string
		distance       float64
		durationMin    int
		speedMax       int
		speedAvg       sql.NullFloat64
		energyUsed     sql.NullFloat64
		consumptionNet sql.NullFloat64
		outsideTemp    sql.NullFloat64
		elevation      sql.NullFloat64
		startOdometer  sql.NullFloat64
		endOdometer    sql.NullFloat64
		startBattery   int
		endBattery     int
	)
	if err := db.QueryRow(query, carID, driveID).Scan(&id, &startDate, &endDate, &startAddress, &endAddress, &distance, &durationMin, &speedMax, &speedAvg, &energyUsed, &consumptionNet, &outsideTemp, &elevation, &startOdometer, &endOdometer, &startBattery, &endBattery); err != nil {
		return nil, err
	}
	if strings.EqualFold(unitsLength, "mi") {
		distance = kilometersToMiles(distance)
		speedMax = int(kilometersToMiles(float64(speedMax)))
		speedAvg = kmhToMphNull(speedAvg)
		consumptionNet = whPerKmToWhPerMiNull(consumptionNet)
		startOdometer = kilometersToMilesSqlNullFloat64(startOdometer)
		endOdometer = kilometersToMilesSqlNullFloat64(endOdometer)
	}
	if strings.EqualFold(unitsTemperature, "f") && outsideTemp.Valid {
		outsideTemp.Float64 = celsiusToFahrenheit(outsideTemp.Float64)
	}
	positions, truncated, err := fetchDrivePositions(driveID, 2000)
	if err != nil {
		return nil, err
	}
	contextBefore, contextAfter, _ := fetchDriveParkingContexts(carID, startDate, endDate)
	return map[string]any{
		"drive_id":            id,
		"start_time":          getTimeInTimeZone(startDate),
		"end_time":            getTimeInTimeZone(endDate),
		"start_address":       startAddress,
		"end_address":         endAddress,
		"distance":            distance,
		"duration_minutes":    durationMin,
		"average_speed":       floatPointer(speedAvg),
		"max_speed":           speedMax,
		"energy_used":         floatPointer(energyUsed),
		"consumption_net":     floatPointer(consumptionNet),
		"outside_temperature": floatPointer(outsideTemp),
		"elevation":           floatPointer(elevation),
		"start_odometer":      floatPointer(startOdometer),
		"end_odometer":        floatPointer(endOdometer),
		"start_battery_level": startBattery,
		"end_battery_level":   endBattery,
		"route":               positions,
		"route_meta":          map[string]any{"truncated": truncated, "point_limit": 2000},
		"parking_before":      contextBefore,
		"parking_after":       contextAfter,
	}, nil
}

func fetchDrivePositions(driveID int, limit int) ([]map[string]any, bool, error) {
	rows, err := db.Query(`
		SELECT date, latitude, longitude, speed, power, odometer
		FROM positions
		WHERE drive_id = $1 AND latitude IS NOT NULL AND longitude IS NOT NULL
		ORDER BY date ASC
		LIMIT $2`, driveID, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var date string
		var lat, lng float64
		var speed sql.NullInt64
		var power sql.NullInt64
		var odometer sql.NullFloat64
		if err := rows.Scan(&date, &lat, &lng, &speed, &power, &odometer); err != nil {
			return nil, false, err
		}
		items = append(items, map[string]any{
			"time":      getTimeInTimeZone(date),
			"latitude":  lat,
			"longitude": lng,
			"speed":     intPointer(speed),
			"power":     intPointer(power),
			"odometer":  floatPointer(odometer),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	truncated := len(items) > limit
	if truncated {
		items = items[:limit]
	}
	return items, truncated, nil
}

func fetchDriveParkingContexts(carID int, startDate, endDate string) (map[string]any, map[string]any, error) {
	fetchOne := func(query string, dateValue string) (map[string]any, error) {
		var id int
		var state string
		var start string
		var end sql.NullString
		var duration int
		err := db.QueryRow(query, carID, dateValue).Scan(&id, &state, &start, &end, &duration)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		return map[string]any{"state_id": id, "state": state, "start_time": getTimeInTimeZone(start), "end_time": timeZoneStringPointer(end), "duration_minutes": duration}, nil
	}
	before, err := fetchOne(`
		SELECT id, state::text, start_date, end_date,
			GREATEST(COALESCE(EXTRACT(EPOCH FROM (COALESCE(end_date, NOW() AT TIME ZONE 'UTC') - start_date)) / 60, 0), 0)::int
		FROM states
		WHERE car_id = $1 AND COALESCE(end_date, NOW() AT TIME ZONE 'UTC') <= $2
		ORDER BY COALESCE(end_date, NOW() AT TIME ZONE 'UTC') DESC
		LIMIT 1`, startDate)
	if err != nil {
		return nil, nil, err
	}
	after, err := fetchOne(`
		SELECT id, state::text, start_date, end_date,
			GREATEST(COALESCE(EXTRACT(EPOCH FROM (COALESCE(end_date, NOW() AT TIME ZONE 'UTC') - start_date)) / 60, 0), 0)::int
		FROM states
		WHERE car_id = $1 AND start_date >= $2
		ORDER BY start_date ASC
		LIMIT 1`, endDate)
	return before, after, err
}

func fetchChargeDetailsPayload(carID int, chargeID int, unitsLength, unitsTemperature string) (map[string]any, error) {
	query := `
		SELECT charging_processes.id, charging_processes.start_date, charging_processes.end_date,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, NULLIF(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS address,
			GREATEST(COALESCE(charging_processes.duration_min, 0), 0)::int,
			charging_processes.start_battery_level,
			charging_processes.end_battery_level,
			GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0)::float8,
			GREATEST(COALESCE(charging_processes.charge_energy_used, charging_processes.charge_energy_added, 0), 0)::float8,
			charging_processes.cost,
			charging_processes.outside_temp_avg,
			(SELECT NULLIF(MAX(charges.fast_charger_brand), '') FROM charges WHERE charges.charging_process_id = charging_processes.id),
			(SELECT NULLIF(MAX(charges.fast_charger_type), '') FROM charges WHERE charges.charging_process_id = charging_processes.id)
		FROM charging_processes
		LEFT JOIN addresses address ON address.id = charging_processes.address_id
		LEFT JOIN geofences geofence ON geofence.id = charging_processes.geofence_id
		WHERE charging_processes.car_id = $1 AND charging_processes.id = $2 AND charging_processes.end_date IS NOT NULL`
	var (
		id           int
		startDate    string
		endDate      string
		address      string
		duration     int
		startBattery int
		endBattery   int
		energyAdded  float64
		energyUsed   float64
		cost         sql.NullFloat64
		outsideTemp  sql.NullFloat64
		fastBrand    sql.NullString
		fastType     sql.NullString
	)
	if err := db.QueryRow(query, carID, chargeID).Scan(&id, &startDate, &endDate, &address, &duration, &startBattery, &endBattery, &energyAdded, &energyUsed, &cost, &outsideTemp, &fastBrand, &fastType); err != nil {
		return nil, err
	}
	if strings.EqualFold(unitsTemperature, "f") && outsideTemp.Valid {
		outsideTemp.Float64 = celsiusToFahrenheit(outsideTemp.Float64)
	}
	curve, maxPower, avgPower, err := fetchChargeCurveSeries(chargeID)
	if err != nil {
		return nil, err
	}
	current, _, _, _, carEfficiency, err := fetchChargeIntervalCurrentCharge(carID, chargeID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	var breakdown any = map[string]any{}
	if err == nil && current.StartDate.Valid {
		previous, prevErr := fetchPreviousChargeIntervalCharge(carID, current.StartDate.String)
		if prevErr == nil {
			interval, energyBreakdown, calcErr := makeChargeIntervalSummary(current, previous, carID, carEfficiency, unitsLength)
			if calcErr == nil {
				breakdown = map[string]any{"interval": interval, "energy_breakdown": energyBreakdown, "previous_charge_id": previous.ID}
			}
		}
	}
	chargingEfficiency := 0.0
	if energyUsed > 0 && energyAdded > 0 {
		chargingEfficiency = energyAdded / energyUsed
	}
	chargerType := "AC"
	if fastType.Valid && strings.TrimSpace(fastType.String) != "" {
		chargerType = fastType.String
	}
	return map[string]any{
		"charge_id":           id,
		"start_time":          getTimeInTimeZone(startDate),
		"end_time":            getTimeInTimeZone(endDate),
		"duration_minutes":    duration,
		"location":            address,
		"charger_type":        chargerType,
		"charger_brand":       stringPointer(fastBrand),
		"start_soc":           startBattery,
		"end_soc":             endBattery,
		"energy_added":        energyAdded,
		"energy_used":         energyUsed,
		"cost":                floatPointer(cost),
		"max_power":           maxPower,
		"average_power":       avgPower,
		"charging_efficiency": chargingEfficiency,
		"outside_temperature": floatPointer(outsideTemp),
		"charge_curve":        curve,
		"interval_usage":      breakdown,
	}, nil
}

func fetchChargeCurveSeries(chargeID int) ([]map[string]any, *float64, *float64, error) {
	rows, err := db.Query(`
		SELECT date, charger_power, battery_level, charge_energy_added
		FROM charges
		WHERE charging_process_id = $1
		ORDER BY date ASC`, chargeID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	curve := []map[string]any{}
	var totalPower float64
	var count float64
	var maxPower float64
	for rows.Next() {
		var date string
		var chargerPower sql.NullFloat64
		var batteryLevel sql.NullInt64
		var energyAdded sql.NullFloat64
		if err := rows.Scan(&date, &chargerPower, &batteryLevel, &energyAdded); err != nil {
			return nil, nil, nil, err
		}
		if chargerPower.Valid {
			totalPower += chargerPower.Float64
			count++
			if chargerPower.Float64 > maxPower {
				maxPower = chargerPower.Float64
			}
		}
		curve = append(curve, map[string]any{"time": getTimeInTimeZone(date), "power": floatPointer(chargerPower), "battery_level": intPointer(batteryLevel), "energy_added": floatPointer(energyAdded)})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	var avgPower *float64
	var maxPowerPtr *float64
	if count > 0 {
		avg := totalPower / count
		avgPower = &avg
		maxPowerPtr = &maxPower
	}
	return curve, maxPowerPtr, avgPower, nil
}
