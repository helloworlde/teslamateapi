package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func TeslaMateAPICarsLocationsV2(c *gin.Context) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid locations range", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsLocationsV2")
	if !ok {
		return
	}
	limit := 100
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeV1Error(c, http.StatusBadRequest, "invalid_limit", "limit must be positive integer", nil)
			return
		}
		if parsed < limit {
			limit = parsed
		}
	}
	startUTC, endUTC := dbTimeRange(dr)
	locations, summary, err := fetchLocationsSummary(ctx.CarID, startUTC, endUTC, limit)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load locations", map[string]any{"reason": err.Error()})
		return
	}
	writeV1Object(c, map[string]any{
		"car_id":    ctx.CarID,
		"range":     buildRangeDTO(dr),
		"summary":   summary,
		"locations": locations,
	}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"))
}

func fetchLocationsSummary(carID int, startUTC, endUTC string, limit int) ([]map[string]any, map[string]any, error) {
	query := `
		WITH location_events AS (
			SELECT
				COALESCE(
					'geofence:' || start_geofence.id::text,
					'address:' || start_address.id::text,
					'coord:' || ROUND(start_position.latitude::numeric, 4)::text || ',' || ROUND(start_position.longitude::numeric, 4)::text,
					'unknown'
				) AS location_key,
				COALESCE(start_geofence.name, CONCAT_WS(', ', COALESCE(start_address.name, NULLIF(CONCAT_WS(' ', start_address.road, start_address.house_number), '')), start_address.city), 'Unknown') AS location,
				start_position.latitude,
				start_position.longitude,
				1::int AS drive_start_count,
				0::int AS drive_end_count,
				0::int AS charge_count,
				0::float8 AS charge_energy_kwh,
				NULL::float8 AS charge_cost,
				drives.start_date AS last_seen
			FROM drives
			LEFT JOIN addresses start_address ON start_address.id = drives.start_address_id
			LEFT JOIN geofences start_geofence ON start_geofence.id = drives.start_geofence_id
			LEFT JOIN positions start_position ON start_position.id = drives.start_position_id
			WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
			UNION ALL
			SELECT
				COALESCE(
					'geofence:' || end_geofence.id::text,
					'address:' || end_address.id::text,
					'coord:' || ROUND(end_position.latitude::numeric, 4)::text || ',' || ROUND(end_position.longitude::numeric, 4)::text,
					'unknown'
				) AS location_key,
				COALESCE(end_geofence.name, CONCAT_WS(', ', COALESCE(end_address.name, NULLIF(CONCAT_WS(' ', end_address.road, end_address.house_number), '')), end_address.city), 'Unknown') AS location,
				end_position.latitude,
				end_position.longitude,
				0::int AS drive_start_count,
				1::int AS drive_end_count,
				0::int AS charge_count,
				0::float8 AS charge_energy_kwh,
				NULL::float8 AS charge_cost,
				drives.end_date AS last_seen
			FROM drives
			LEFT JOIN addresses end_address ON end_address.id = drives.end_address_id
			LEFT JOIN geofences end_geofence ON end_geofence.id = drives.end_geofence_id
			LEFT JOIN positions end_position ON end_position.id = drives.end_position_id
			WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
			UNION ALL
			SELECT
				COALESCE(
					'geofence:' || geofence.id::text,
					'address:' || address.id::text,
					'coord:' || ROUND(position.latitude::numeric, 4)::text || ',' || ROUND(position.longitude::numeric, 4)::text,
					'unknown'
				) AS location_key,
				COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, NULLIF(CONCAT_WS(' ', address.road, address.house_number), '')), address.city), 'Unknown') AS location,
				position.latitude,
				position.longitude,
				0::int AS drive_start_count,
				0::int AS drive_end_count,
				1::int AS charge_count,
				GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0)::float8 AS charge_energy_kwh,
				charging_processes.cost::float8 AS charge_cost,
				charging_processes.start_date AS last_seen
			FROM charging_processes
			LEFT JOIN addresses address ON address.id = charging_processes.address_id
			LEFT JOIN geofences geofence ON geofence.id = charging_processes.geofence_id
			LEFT JOIN positions position ON position.id = charging_processes.position_id
			WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date < $3
		),
		location_agg AS (
			SELECT
				location_key,
				NULLIF(location, '') AS location,
				AVG(latitude)::float8 AS latitude,
				AVG(longitude)::float8 AS longitude,
				SUM(drive_start_count)::int AS drive_start_count,
				SUM(drive_end_count)::int AS drive_end_count,
				SUM(charge_count)::int AS charge_count,
				SUM(charge_energy_kwh)::float8 AS charge_energy_kwh,
				NULLIF(SUM(CASE WHEN charge_cost > 0 THEN charge_cost ELSE 0 END), 0)::float8 AS charge_cost,
				MAX(last_seen) AS last_seen
			FROM location_events
			GROUP BY location_key, NULLIF(location, '')
		)
		SELECT location, latitude, longitude, drive_start_count, drive_end_count, charge_count,
			charge_energy_kwh, charge_cost, last_seen,
			COUNT(*) OVER()::int AS total_locations
		FROM location_agg
		WHERE location IS NOT NULL
		ORDER BY last_seen DESC, (drive_start_count + drive_end_count + charge_count) DESC
		LIMIT $4`
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, carID, startUTC, endUTC, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	locations := make([]map[string]any, 0)
	summary := map[string]any{
		"location_count":        0,
		"returned_count":        0,
		"drive_location_count":  0,
		"charge_location_count": 0,
		"drive_start_count":     0,
		"drive_end_count":       0,
		"charge_count":          0,
		"charge_energy_kwh":     0.0,
		"charge_cost":           nil,
	}
	totalCost := 0.0
	hasCost := false
	for rows.Next() {
		var (
			location                          string
			latitude, longitude               sql.NullFloat64
			driveStart, driveEnd, chargeCount int
			chargeEnergy, chargeCost          sql.NullFloat64
			lastSeen                          sql.NullString
			totalLocations                    int
		)
		if err := rows.Scan(&location, &latitude, &longitude, &driveStart, &driveEnd, &chargeCount, &chargeEnergy, &chargeCost, &lastSeen, &totalLocations); err != nil {
			return nil, nil, err
		}
		summary["location_count"] = totalLocations
		summary["drive_start_count"] = summary["drive_start_count"].(int) + driveStart
		summary["drive_end_count"] = summary["drive_end_count"].(int) + driveEnd
		summary["charge_count"] = summary["charge_count"].(int) + chargeCount
		if driveStart+driveEnd > 0 {
			summary["drive_location_count"] = summary["drive_location_count"].(int) + 1
		}
		if chargeCount > 0 {
			summary["charge_location_count"] = summary["charge_location_count"].(int) + 1
		}
		if chargeEnergy.Valid {
			summary["charge_energy_kwh"] = summary["charge_energy_kwh"].(float64) + chargeEnergy.Float64
		}
		if chargeCost.Valid {
			totalCost += chargeCost.Float64
			hasCost = true
		}
		locations = append(locations, map[string]any{
			"name":              location,
			"latitude":          floatPointer(latitude),
			"longitude":         floatPointer(longitude),
			"drive_start_count": driveStart,
			"drive_end_count":   driveEnd,
			"drive_count":       driveStart + driveEnd,
			"charge_count":      chargeCount,
			"charge_energy_kwh": floatPointer(chargeEnergy),
			"charge_cost":       floatPointer(chargeCost),
			"total_event_count": driveStart + driveEnd + chargeCount,
			"last_seen":         timeZoneStringPointer(lastSeen),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	summary["returned_count"] = len(locations)
	if hasCost {
		summary["charge_cost"] = totalCost
	}
	return locations, summary, nil
}
