package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	aggregatecache "github.com/tobiasehlert/teslamateapi/src/internal/aggregatecache"
)

type metricDef struct {
	Key       string
	Name      string
	Unit      string
	Scope     string
	ChartType string
}

func TeslaMateAPICarsDriveSeriesV2(c *gin.Context) {
	writeScopedSeries(c, "drives", []string{"distance", "speed", "max_speed", "motor_power", "regen_power", "elevation", "outside_temp", "efficiency", "energy", "regeneration"})
}

func TeslaMateAPICarsChargeSeriesV2(c *gin.Context) {
	writeScopedSeries(c, "charges", []string{"energy", "power", "cost", "start_soc", "end_soc"})
}

func TeslaMateAPICarsBatterySeriesV2(c *gin.Context) {
	writeScopedSeries(c, "battery", []string{"soc", "range"})
}

func TeslaMateAPICarsStateSeriesV2(c *gin.Context) {
	writeScopedSeries(c, "states", []string{"duration", "vampire_drain"})
}

func writeScopedSeries(c *gin.Context, scope string, defaultMetrics []string) {
	// series 返回“指标元数据 + 合并后的时间点”，前端可以直接按 time 渲染多指标图表。
	bucket := strings.ToLower(strings.TrimSpace(c.DefaultQuery("bucket", "day")))
	metrics := parseCSV(c.Query("metrics"))
	if len(metrics) == 0 {
		metrics = defaultMetrics
	}
	switch bucket {
	case "raw", "hour", "day", "week", "month", "year":
	default:
		writeV1Error(c, http.StatusBadRequest, "invalid_bucket", "bucket must be raw|hour|day|week|month|year", nil)
		return
	}
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid series range", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsSeriesV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	defs := make([]metricDef, 0, len(metrics))
	values := make(map[string][]map[string]any, len(metrics))
	metadata := make([]map[string]any, 0, len(metrics))
	for _, metric := range metrics {
		def, ok := metricDefinition(scope, metric)
		if !ok {
			writeV1Error(c, http.StatusBadRequest, "unsupported_metric", "unsupported series metric", map[string]any{"scope": scope, "metric": metric})
			return
		}
		points, err := fetchMetricSeries(ctx.CarID, scope, metric, bucket, startUTC, endUTC, ctx.UnitsLength)
		if err != nil {
			writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load series metric", map[string]any{"scope": scope, "metric": metric, "reason": err.Error()})
			return
		}
		def.Unit = metricUnit(def.Key, scope, ctx.UnitsLength)
		defs = append(defs, def)
		values[def.Key] = points
		metadata = append(metadata, map[string]any{
			"metric":     def.Key,
			"name":       def.Name,
			"unit":       def.Unit,
			"chart_type": def.ChartType,
		})
	}
	writeV1Object(c, map[string]any{
		"car_id":  ctx.CarID,
		"scope":   scope,
		"bucket":  bucket,
		"range":   buildRangeDTO(dr),
		"metrics": metadata,
		"points":  mergeMetricSeries(defs, values),
	}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"))
}

func mergeMetricSeries(defs []metricDef, values map[string][]map[string]any) []map[string]any {
	// 将多条单指标序列合并成同一时间点的宽表结构：
	// {"time": "...", "distance": 12.3, "speed": 34.5}
	byTime := make(map[string]map[string]any)
	times := make([]string, 0)
	for _, def := range defs {
		for _, point := range values[def.Key] {
			t, _ := point["time"].(string)
			if t == "" {
				continue
			}
			row, ok := byTime[t]
			if !ok {
				row = map[string]any{"time": t}
				byTime[t] = row
				times = append(times, t)
			}
			row[def.Key] = point["value"]
		}
	}
	sort.SliceStable(times, func(i, j int) bool { return times[i] > times[j] })
	out := make([]map[string]any, 0, len(times))
	for _, t := range times {
		out = append(out, byTime[t])
	}
	return out
}

func metricDefinition(scope, metric string) (metricDef, bool) {
	defs := map[string]map[string]metricDef{
		"drives": {
			"distance":     {Key: "distance", Name: "drive_distance", Unit: "km", Scope: "drives", ChartType: "bar"},
			"efficiency":   {Key: "efficiency", Name: "drive_efficiency", Unit: "Wh/km", Scope: "drives", ChartType: "line"},
			"speed":        {Key: "speed", Name: "drive_average_speed", Unit: "km/h", Scope: "drives", ChartType: "line"},
			"max_speed":    {Key: "max_speed", Name: "drive_max_speed", Unit: "km/h", Scope: "drives", ChartType: "line"},
			"motor_power":  {Key: "motor_power", Name: "drive_motor_power", Unit: "kW", Scope: "drives", ChartType: "line"},
			"regen_power":  {Key: "regen_power", Name: "drive_regen_power", Unit: "kW", Scope: "drives", ChartType: "line"},
			"elevation":    {Key: "elevation", Name: "drive_elevation", Unit: "m", Scope: "drives", ChartType: "line"},
			"outside_temp": {Key: "outside_temp", Name: "drive_outside_temperature", Unit: "C", Scope: "drives", ChartType: "line"},
			"inside_temp":  {Key: "inside_temp", Name: "drive_inside_temperature", Unit: "C", Scope: "drives", ChartType: "line"},
			"energy":       {Key: "energy", Name: "drive_energy", Unit: "kWh", Scope: "drives", ChartType: "area"},
			"regeneration": {Key: "regeneration", Name: "drive_regeneration", Unit: "kWh", Scope: "drives", ChartType: "area"},
		},
		"charges": {
			"energy":    {Key: "energy", Name: "charge_energy", Unit: "kWh", Scope: "charges", ChartType: "bar"},
			"power":     {Key: "power", Name: "charge_average_power", Unit: "kW", Scope: "charges", ChartType: "line"},
			"cost":      {Key: "cost", Name: "charge_cost", Unit: "currency", Scope: "charges", ChartType: "bar"},
			"start_soc": {Key: "start_soc", Name: "charge_start_soc", Unit: "%", Scope: "charges", ChartType: "line"},
			"end_soc":   {Key: "end_soc", Name: "charge_end_soc", Unit: "%", Scope: "charges", ChartType: "line"},
			"soc":       {Key: "end_soc", Name: "charge_end_soc", Unit: "%", Scope: "charges", ChartType: "line"},
		},
		"battery": {
			"soc":   {Key: "soc", Name: "battery_soc", Unit: "%", Scope: "battery", ChartType: "line"},
			"range": {Key: "range", Name: "battery_rated_range", Unit: "km", Scope: "battery", ChartType: "line"},
		},
		"states": {
			"duration":      {Key: "duration", Name: "state_duration", Unit: "min", Scope: "states", ChartType: "bar"},
			"vampire_drain": {Key: "vampire_drain", Name: "vampire_drain", Unit: "kWh", Scope: "states", ChartType: "bar"},
		},
	}
	scopeDefs, ok := defs[scope]
	if !ok {
		return metricDef{}, false
	}
	def, ok := scopeDefs[metric]
	return def, ok
}

func metricUnit(metric, scope, unitsLength string) string {
	length := "km"
	speed := "km/h"
	consumption := "Wh/km"
	if strings.EqualFold(unitsLength, "mi") {
		length = "mi"
		speed = "mph"
		consumption = "Wh/mi"
	}
	switch metric {
	case "distance", "range":
		return length
	case "speed", "max_speed":
		return speed
	case "efficiency":
		return consumption
	case "energy", "regeneration", "vampire_drain":
		return "kWh"
	case "cost":
		return "currency"
	case "power", "motor_power", "regen_power":
		return "kW"
	case "soc", "start_soc", "end_soc":
		return "%"
	case "duration":
		return "min"
	case "elevation":
		return "m"
	case "outside_temp", "inside_temp":
		return "C"
	default:
		if scope == "battery" {
			return "%"
		}
		return "count"
	}
}

func convertMetricSeriesUnits(points []map[string]any, metric, unitsLength string) []map[string]any {
	if !strings.EqualFold(unitsLength, "mi") {
		return points
	}
	for _, point := range points {
		raw, ok := asFloat64(point["value"])
		if !ok {
			continue
		}
		switch metric {
		case "distance", "range", "speed", "max_speed":
			point["value"] = kilometersToMiles(raw)
		case "efficiency":
			point["value"] = whPerKmToWhPerMi(raw)
		}
	}
	return points
}

func localBucketTime(value string) string {
	if value == "" {
		return ""
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", value, appUsersTimezone); err == nil {
		return t.Format(time.RFC3339)
	}
	if t, err := time.ParseInLocation("2006-01-02", value, appUsersTimezone); err == nil {
		return t.Format(time.RFC3339)
	}
	return value
}

func fetchMetricSeries(carID int, scope, metric, bucket, startUTC, endUTC, unitsLength string) ([]map[string]any, error) {
	// 历史时序数据按指标独立缓存，避免一次请求中重复扫描相同时间范围。
	key := aggregatecache.Key("series", carID, scope, metric, bucket, startUTC, endUTC, unitsLength, appUsersTimezone.String())
	return aggregatecache.Value(key, aggregatecache.TTL(endUTC), func() ([]map[string]any, error) {
		return fetchMetricSeriesUncached(carID, scope, metric, bucket, startUTC, endUTC, unitsLength)
	})
}

func bucketExpression(bucket, column string) string {
	// raw 对外表示“尽量细”，当前落到 hour，避免直接返回海量 position 原始点。
	if bucket == "raw" {
		bucket = "hour"
	}
	trunc := "day"
	switch bucket {
	case "hour", "week", "month", "year":
		trunc = bucket
	}
	return fmt.Sprintf("date_trunc('%s', timezone($4, %s))", trunc, column)
}

func fetchMetricSeriesUncached(carID int, scope, metric, bucket, startUTC, endUTC, unitsLength string) ([]map[string]any, error) {
	// SQL 统一规则：所有时间桶先按环境时区转换，再 date_trunc；过滤条件使用 UTC 半开区间。
	eventBucketExpr := bucketExpression(bucket, "start_date")
	positionBucketExpr := bucketExpression(bucket, "positions.date")
	var query string
	switch metric {
	case "distance":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, SUM(distance)::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "efficiency":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, CASE WHEN SUM(distance) > 0 THEN SUM(CASE WHEN (start_rated_range_km-end_rated_range_km) > 0 THEN (start_rated_range_km-end_rated_range_km) * cars.efficiency ELSE 0 END) / SUM(distance) * 1000.0 ELSE NULL END::float8 FROM drives LEFT JOIN cars ON cars.id = drives.car_id WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "energy":
		if scope == "charges" {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, SUM(charge_energy_added)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
		} else {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, SUM(CASE WHEN (start_rated_range_km-end_rated_range_km) > 0 THEN (start_rated_range_km-end_rated_range_km) * cars.efficiency ELSE 0 END)::float8 FROM drives LEFT JOIN cars ON cars.id = drives.car_id WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
		}
	case "cost":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, NULLIF(SUM(CASE WHEN cost > 0 THEN cost ELSE 0 END),0)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "speed":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, CASE WHEN SUM(duration_min) > 0 THEN SUM(distance)/(SUM(duration_min)/60.0) ELSE NULL END::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "max_speed":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, MAX(speed_max)::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "motor_power":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, MAX(NULLIF(power_max, 0))::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "regen_power":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, ABS(MIN(CASE WHEN power_min < 0 THEN power_min ELSE NULL END))::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "outside_temp":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(outside_temp_avg)::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "inside_temp":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(positions.inside_temp)::float8 FROM drives INNER JOIN positions ON positions.drive_id = drives.id WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3 AND positions.inside_temp IS NOT NULL GROUP BY t ORDER BY t DESC`, positionBucketExpr)
	case "elevation":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(positions.elevation)::float8 FROM drives INNER JOIN positions ON positions.drive_id = drives.id WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3 AND positions.elevation IS NOT NULL GROUP BY t ORDER BY t DESC`, positionBucketExpr)
	case "power":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(NULLIF(charges.charger_power,0))::float8 FROM charging_processes LEFT JOIN charges ON charges.charging_process_id = charging_processes.id WHERE charging_processes.car_id=$1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "start_soc":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(start_battery_level)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
	case "soc", "end_soc":
		if scope == "charges" {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(end_battery_level)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t DESC`, eventBucketExpr)
		} else {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(positions.battery_level)::float8 FROM positions WHERE positions.car_id=$1 AND positions.date >= $2 AND positions.date < $3 GROUP BY t ORDER BY t DESC`, positionBucketExpr)
		}
	case "range":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(positions.rated_battery_range_km)::float8 FROM positions WHERE positions.car_id=$1 AND positions.date >= $2 AND positions.date < $3 AND positions.rated_battery_range_km IS NOT NULL GROUP BY t ORDER BY t DESC`, positionBucketExpr)
	case "regeneration":
		query = fmt.Sprintf(`WITH regen AS (
			-- 使用 position.power 的负功率样本估算动能回收；过滤异常采样间隔避免尖峰污染
			SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t,
				ABS(LEAST(COALESCE(positions.power, 0)::float8, 0)) AS pkw,
				EXTRACT(EPOCH FROM (positions.date - LAG(positions.date) OVER (PARTITION BY drives.id ORDER BY positions.id))) AS ds
			FROM drives INNER JOIN positions ON positions.drive_id = drives.id
			WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
		)
		SELECT t, SUM(pkw * ds / 3600.0)::float8 FROM regen WHERE ds > 0 AND ds <= 300 AND pkw > 0 GROUP BY t ORDER BY t DESC`, positionBucketExpr)
	case "duration":
		query = fmt.Sprintf(`WITH bounded AS (
			-- 状态窗口先裁剪到请求范围内，再按本地 bucket 汇总分钟数
			SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t,
				EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamp), $3::timestamp) - GREATEST(start_date, $2::timestamp))) / 60.0 AS minutes
			FROM states
			WHERE car_id=$1 AND start_date < $3::timestamp AND COALESCE(end_date, $3::timestamp) >= $2::timestamp
		)
		SELECT t, SUM(minutes)::float8 FROM bounded WHERE minutes > 0 GROUP BY t ORDER BY t DESC`, bucketExpression(bucket, "GREATEST(start_date, $2::timestamp)"))
	case "vampire_drain":
		by, _, err := fetchParkingEnergyByBucketWithTimeout(carID, startUTC, endUTC, "day", 1200*time.Millisecond)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(by))
		for k, v := range by {
			val := any(nil)
			if v != nil {
				val = *v
			}
			out = append(out, map[string]any{"time": localBucketTime(k + "T00:00:00"), "value": val})
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i]["time"].(string) > out[j]["time"].(string) })
		return convertMetricSeriesUnits(out, metric, unitsLength), nil
	default:
		return []map[string]any{}, nil
	}
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	for rows.Next() {
		var t string
		var v sql.NullFloat64
		if err := rows.Scan(&t, &v); err != nil {
			return nil, err
		}
		val := any(nil)
		if v.Valid {
			val = v.Float64
		}
		result = append(result, map[string]any{"time": localBucketTime(t), "value": val})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return convertMetricSeriesUnits(result, metric, unitsLength), nil
}
