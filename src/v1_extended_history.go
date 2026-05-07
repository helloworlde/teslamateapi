package main

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TeslaMateAPICarsDrivesHistoryV2 返回分页行程会话列表。
// @Summary 行程会话
// @Description 扩展接口 v2：返回分页行程会话。
// @Tags 扩展 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Param limit query int false "每页数量"
// @Param offset query int false "偏移量"
// @Success 200 {object} HistoryListV2Envelope
// @Failure 400,404,500 {object} v1ErrorEnvelope
// @Router /v2/cars/{CarID}/drives [get]
func TeslaMateAPICarsDrivesHistoryV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsDrivesHistoryV2")
	if !ok {
		return
	}
	offset, limit, err := parseOffsetLimit(c, 20, 100)
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_pagination", err.Error(), nil)
		return
	}
	dr, err := parseDateRangeStrictOrDefault(c, "custom")
	startUTC, endUTC := "", ""
	if err == nil && (dr.Period == "custom" || !dr.Start.IsZero()) {
		startUTC, endUTC = dbTimeRange(dr)
	}

	drives, total, err := fetchDrivesHistory(ctx.CarID, ctx.UnitsLength, ctx.UnitsTemperature, startUTC, endUTC, offset, limit)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load drives", map[string]any{"reason": err.Error()})
		return
	}
	writeV1List(c, drives, v1Pagination{Limit: limit, Offset: offset, Total: total}, buildV1MetaFromCar(ctx, appUsersTimezone.String()))
}

// TeslaMateAPICarsChargesHistoryV2 返回分页充电会话列表。
// @Summary 充电会话
// @Description 扩展接口 v2：返回分页充电会话。
// @Tags 扩展 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Param limit query int false "每页数量"
// @Param offset query int false "偏移量"
// @Success 200 {object} HistoryListV2Envelope
// @Failure 400,404,500 {object} v1ErrorEnvelope
// @Router /v2/cars/{CarID}/charges [get]
func TeslaMateAPICarsChargesHistoryV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsChargesHistoryV2")
	if !ok {
		return
	}
	offset, limit, err := parseOffsetLimit(c, 20, 100)
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_pagination", err.Error(), nil)
		return
	}
	dr, err := parseDateRangeStrictOrDefault(c, "custom")
	startUTC, endUTC := "", ""
	if err == nil && (dr.Period == "custom" || !dr.Start.IsZero()) {
		startUTC, endUTC = dbTimeRange(dr)
	}

	charges, total, err := fetchChargesHistory(ctx.CarID, ctx.UnitsLength, ctx.UnitsTemperature, startUTC, endUTC, offset, limit)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load charges", map[string]any{"reason": err.Error()})
		return
	}
	writeV1List(c, charges, v1Pagination{Limit: limit, Offset: offset, Total: total}, buildV1MetaFromCar(ctx, appUsersTimezone.String()))
}

func fetchDrivesHistory(carID int, unitsLength, unitsTemperature, startUTC, endUTC string, offset, limit int) ([]map[string]any, int, error) {
	whereClause := "d.car_id = $1 AND d.end_date IS NOT NULL"
	args := []any{carID}
	idx := 2
	if startUTC != "" {
		whereClause += " AND d.start_date >= $" + itoa(idx)
		args = append(args, startUTC)
		idx++
	}
	if endUTC != "" {
		whereClause += " AND d.end_date < $" + itoa(idx)
		args = append(args, endUTC)
		idx++
	}
	args = append(args, appUsersTimezone.String(), limit, offset)
	tzIdx, limitIdx, offsetIdx := idx, idx+1, idx+2

	query := `
		SELECT
			d.id,
			TO_CHAR(d.start_date AT TIME ZONE $` + itoa(tzIdx) + `, 'YYYY-MM-DD"T"HH24:MI:SSof') AS started_at,
			TO_CHAR(d.end_date   AT TIME ZONE $` + itoa(tzIdx) + `, 'YYYY-MM-DD"T"HH24:MI:SSof') AS ended_at,
			d.duration_min,
			d.distance,
			CASE WHEN d.duration_min > 0 THEN d.distance / (d.duration_min / 60.0) ELSE NULL END AS avg_speed,
			d.speed_max,
			CASE WHEN (d.start_rated_range_km - d.end_rated_range_km) > 0
				THEN (d.start_rated_range_km - d.end_rated_range_km) * c.efficiency
				ELSE NULL END AS energy_used_kwh,
			CASE WHEN d.distance > 0 AND (d.start_rated_range_km - d.end_rated_range_km) > 0
				THEN (d.start_rated_range_km - d.end_rated_range_km) * c.efficiency / d.distance * 1000.0
				ELSE NULL END AS efficiency_wh_km,
			p_start.battery_level AS start_soc,
			p_end.battery_level   AS end_soc,
			d.outside_temp_avg,
			COALESCE(g_start.name,
				NULLIF(CONCAT_WS(', ', NULLIF(CONCAT_WS(' ', a_start.road, a_start.house_number), ''), a_start.city), ''),
				'Unknown') AS start_location,
			COALESCE(g_end.name,
				NULLIF(CONCAT_WS(', ', NULLIF(CONCAT_WS(' ', a_end.road, a_end.house_number), ''), a_end.city), ''),
				'Unknown') AS end_location,
			COUNT(*) OVER() AS total_count
		FROM drives d
		LEFT JOIN cars c               ON c.id = d.car_id
		LEFT JOIN positions p_start    ON p_start.id = d.start_position_id
		LEFT JOIN positions p_end      ON p_end.id   = d.end_position_id
		LEFT JOIN geofences g_start    ON g_start.id = d.start_geofence_id
		LEFT JOIN addresses a_start    ON a_start.id = d.start_address_id
		LEFT JOIN geofences g_end      ON g_end.id   = d.end_geofence_id
		LEFT JOIN addresses a_end      ON a_end.id   = d.end_address_id
		WHERE ` + whereClause + `
		ORDER BY d.start_date DESC
		LIMIT $` + itoa(limitIdx) + ` OFFSET $` + itoa(offsetIdx)

	qCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(qCtx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	imperial := strings.EqualFold(unitsLength, "mi")
	fahrenheit := strings.EqualFold(unitsTemperature, "f")
	distUnit := "km"
	speedUnit := "km/h"
	consumptionUnit := "Wh/km"
	if imperial {
		distUnit = "mi"
		speedUnit = "mph"
		consumptionUnit = "Wh/mi"
	}
	tempUnit := "°C"
	if fahrenheit {
		tempUnit = "°F"
	}
	_ = distUnit
	_ = speedUnit
	_ = consumptionUnit
	_ = tempUnit

	result := make([]map[string]any, 0)
	total := 0
	for rows.Next() {
		var (
			id                                      int
			startedAt, endedAt                      sql.NullString
			durationMin                             sql.NullInt64
			distance, avgSpeed                      sql.NullFloat64
			speedMax                                sql.NullInt64
			energyUsed, efficiencyWhKm, outsideTemp sql.NullFloat64
			startSOC, endSOC                        sql.NullInt64
			startLocation, endLocation              sql.NullString
			totalCount                              int
		)
		if err := rows.Scan(&id, &startedAt, &endedAt, &durationMin, &distance, &avgSpeed,
			&speedMax, &energyUsed, &efficiencyWhKm, &startSOC, &endSOC,
			&outsideTemp, &startLocation, &endLocation, &totalCount); err != nil {
			return nil, 0, err
		}
		total = totalCount

		distVal := floatOrNil(distance)
		avgSpeedVal := floatOrNil(avgSpeed)
		speedMaxVal := intOrNil(speedMax)
		effVal := floatOrNil(efficiencyWhKm)
		tempVal := floatOrNil(outsideTemp)

		if imperial {
			if v, ok := distVal.(float64); ok {
				distVal = kilometersToMiles(v)
			}
			if v, ok := avgSpeedVal.(float64); ok {
				avgSpeedVal = kilometersToMiles(v)
			}
			if v, ok := speedMaxVal.(int); ok {
				speedMaxVal = kilometersToMilesInteger(v)
			}
			if v, ok := effVal.(float64); ok {
				effVal = whPerKmToWhPerMi(v)
			}
		}
		if fahrenheit {
			if v, ok := tempVal.(float64); ok {
				tempVal = celsiusToFahrenheit(v)
			}
		}

		result = append(result, map[string]any{
			"id":              id,
			"started_at":      nullStringVal(startedAt),
			"ended_at":        nullStringVal(endedAt),
			"duration_min":    intOrNil(durationMin),
			"distance":        distVal,
			"avg_speed":       avgSpeedVal,
			"max_speed":       speedMaxVal,
			"energy_used_kwh": floatOrNil(energyUsed),
			"efficiency":      effVal,
			"start_soc":       intOrNil(startSOC),
			"end_soc":         intOrNil(endSOC),
			"outside_temp":    tempVal,
			"start_location":  nullStringVal(startLocation),
			"end_location":    nullStringVal(endLocation),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func fetchChargesHistory(carID int, unitsLength, _ string, startUTC, endUTC string, offset, limit int) ([]map[string]any, int, error) {
	whereClause := "cp.car_id = $1 AND cp.end_date IS NOT NULL"
	args := []any{carID}
	idx := 2
	if startUTC != "" {
		whereClause += " AND cp.start_date >= $" + itoa(idx)
		args = append(args, startUTC)
		idx++
	}
	if endUTC != "" {
		whereClause += " AND cp.end_date < $" + itoa(idx)
		args = append(args, endUTC)
		idx++
	}
	args = append(args, appUsersTimezone.String(), limit, offset)
	tzIdx, limitIdx, offsetIdx := idx, idx+1, idx+2

	query := `
		SELECT
			cp.id,
			TO_CHAR(cp.start_date AT TIME ZONE $` + itoa(tzIdx) + `, 'YYYY-MM-DD"T"HH24:MI:SSof') AS started_at,
			TO_CHAR(cp.end_date   AT TIME ZONE $` + itoa(tzIdx) + `, 'YYYY-MM-DD"T"HH24:MI:SSof') AS ended_at,
			EXTRACT(EPOCH FROM (cp.end_date - cp.start_date))::int / 60 AS duration_min,
			cp.charge_energy_added,
			cp.charge_energy_used,
			CASE WHEN NULLIF(cp.charge_energy_used, 0) IS NOT NULL
				THEN cp.charge_energy_added / cp.charge_energy_used ELSE NULL END AS efficiency,
			cp.cost,
			MAX(ch.charger_power)  AS max_power_kw,
			AVG(NULLIF(ch.charger_power, 0)) AS avg_power_kw,
			cp.start_battery_level AS start_soc,
			cp.end_battery_level   AS end_soc,
			COALESCE(g.name,
				NULLIF(CONCAT_WS(', ', NULLIF(CONCAT_WS(' ', a.road, a.house_number), ''), a.city), ''),
				'Unknown') AS location,
			COUNT(*) OVER() AS total_count
		FROM charging_processes cp
		LEFT JOIN charges  ch ON ch.charging_process_id = cp.id
		LEFT JOIN geofences g ON g.id = cp.geofence_id
		LEFT JOIN addresses a ON a.id = cp.address_id
		WHERE ` + whereClause + `
		GROUP BY cp.id, g.name, a.road, a.house_number, a.city
		ORDER BY cp.start_date DESC
		LIMIT $` + itoa(limitIdx) + ` OFFSET $` + itoa(offsetIdx)

	qCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(qCtx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := make([]map[string]any, 0)
	total := 0
	for rows.Next() {
		var (
			id                                        int
			startedAt, endedAt                        sql.NullString
			durationMin                               sql.NullInt64
			energyAdded, energyUsed, efficiency, cost sql.NullFloat64
			maxPower, avgPower                        sql.NullFloat64
			startSOC, endSOC                          sql.NullInt64
			location                                  sql.NullString
			totalCount                                int
		)
		if err := rows.Scan(&id, &startedAt, &endedAt, &durationMin,
			&energyAdded, &energyUsed, &efficiency, &cost,
			&maxPower, &avgPower, &startSOC, &endSOC,
			&location, &totalCount); err != nil {
			return nil, 0, err
		}
		total = totalCount
		result = append(result, map[string]any{
			"id":               id,
			"started_at":       nullStringVal(startedAt),
			"ended_at":         nullStringVal(endedAt),
			"duration_min":     intOrNil(durationMin),
			"energy_added_kwh": floatOrNil(energyAdded),
			"energy_input_kwh": floatOrNil(energyUsed),
			"efficiency":       floatOrNil(efficiency),
			"cost":             floatOrNil(cost),
			"max_power_kw":     floatOrNil(maxPower),
			"avg_power_kw":     floatOrNil(avgPower),
			"start_soc":        intOrNil(startSOC),
			"end_soc":          intOrNil(endSOC),
			"location":         nullStringVal(location),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

// itoa 将 int 转换为字符串，用于构造 SQL 占位符。
func itoa(n int) string {
	const digits = "0123456789"
	if n < 10 {
		return string(digits[n])
	}
	buf := make([]byte, 0, 4)
	for n > 0 {
		buf = append([]byte{digits[n%10]}, buf...)
		n /= 10
	}
	return string(buf)
}
