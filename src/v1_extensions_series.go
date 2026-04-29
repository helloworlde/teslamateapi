package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type metricDef struct {
	Key       string
	Name      string
	Unit      string
	Scope     string
	ChartType string
}

func TeslaMateAPICarsDriveSeriesV2(c *gin.Context) {
	writeScopedSeries(c, "drives", []string{"distance", "speed", "efficiency", "energy", "regeneration"})
}

func TeslaMateAPICarsChargeSeriesV2(c *gin.Context) {
	writeScopedSeries(c, "charges", []string{"energy", "power", "cost", "soc"})
}

func TeslaMateAPICarsBatterySeriesV2(c *gin.Context) {
	writeScopedSeries(c, "battery", []string{"soc", "range"})
}

func TeslaMateAPICarsStateSeriesV2(c *gin.Context) {
	writeScopedSeries(c, "states", []string{"vampire_drain"})
}

func writeScopedSeries(c *gin.Context, scope string, defaultMetrics []string) {
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
	warnings := []any{}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsSeriesV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	series := make([]any, 0, len(metrics))
	for _, metric := range metrics {
		def, ok := metricDefinition(scope, metric)
		if !ok {
			warnings = append(warnings, map[string]any{"code": "unsupported_metric", "message": "unsupported metric", "metric": metric})
			continue
		}
		points, err := fetchMetricSeries(ctx.CarID, scope, metric, bucket, startUTC, endUTC, ctx.UnitsLength)
		if err != nil {
			warnings = append(warnings, nonFatalWarning("metric_query_failed", "failed to load metric series", map[string]any{"metric": metric}, err))
			continue
		}
		series = append(series, map[string]any{
			"metric":     def.Key,
			"name":       def.Name,
			"unit":       metricUnit(def.Key, scope, ctx.UnitsLength),
			"chart_type": def.ChartType,
			"points":     points,
		})
	}
	writeV1Object(c, map[string]any{
		"car_id": ctx.CarID,
		"scope":  scope,
		"bucket": bucket,
		"range":  buildRangeDTO(dr),
		"series": series,
	}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func metricDefinition(scope, metric string) (metricDef, bool) {
	defs := map[string]map[string]metricDef{
		"drives": {
			"distance":     {Key: "distance", Name: "drive_distance", Unit: "km", Scope: "drives", ChartType: "bar"},
			"efficiency":   {Key: "efficiency", Name: "drive_efficiency", Unit: "Wh/km", Scope: "drives", ChartType: "line"},
			"speed":        {Key: "speed", Name: "drive_speed", Unit: "km/h", Scope: "drives", ChartType: "line"},
			"energy":       {Key: "energy", Name: "drive_energy", Unit: "kWh", Scope: "drives", ChartType: "area"},
			"regeneration": {Key: "regeneration", Name: "drive_regeneration", Unit: "kWh", Scope: "drives", ChartType: "area"},
		},
		"charges": {
			"energy": {Key: "energy", Name: "charge_energy", Unit: "kWh", Scope: "charges", ChartType: "bar"},
			"power":  {Key: "power", Name: "charge_power", Unit: "kW", Scope: "charges", ChartType: "line"},
			"cost":   {Key: "cost", Name: "charge_cost", Unit: "currency", Scope: "charges", ChartType: "bar"},
			"soc":    {Key: "soc", Name: "charge_end_soc", Unit: "%", Scope: "charges", ChartType: "line"},
		},
		"battery": {
			"soc":   {Key: "soc", Name: "battery_soc", Unit: "%", Scope: "battery", ChartType: "line"},
			"range": {Key: "range", Name: "battery_rated_range", Unit: "km", Scope: "battery", ChartType: "line"},
		},
		"states": {
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

func fetchDashboardMetricSeries(carID int, startUTC, endUTC, unitsLength string) ([]any, []any) {
	specs := []struct {
		Scope     string
		Metric    string
		Name      string
		ChartType string
	}{
		{Scope: "drives", Metric: "distance", Name: "drive_distance", ChartType: "bar"},
		{Scope: "drives", Metric: "efficiency", Name: "drive_efficiency", ChartType: "line"},
		{Scope: "drives", Metric: "energy", Name: "drive_energy", ChartType: "area"},
		{Scope: "charges", Metric: "energy", Name: "charge_energy", ChartType: "bar"},
		{Scope: "charges", Metric: "cost", Name: "charge_cost", ChartType: "bar"},
		{Scope: "battery", Metric: "soc", Name: "battery_level", ChartType: "line"},
		{Scope: "battery", Metric: "range", Name: "rated_range", ChartType: "line"},
	}
	series := make([]any, 0, len(specs))
	warnings := make([]any, 0)
	for _, spec := range specs {
		points, err := fetchMetricSeries(carID, spec.Scope, spec.Metric, "day", startUTC, endUTC, unitsLength)
		if err != nil {
			warnings = append(warnings, nonFatalWarning("dashboard_series_unavailable", "failed to load dashboard series", map[string]any{"metric": spec.Name}, err))
			continue
		}
		series = append(series, map[string]any{
			"metric":     spec.Name,
			"name":       spec.Name,
			"unit":       metricUnit(spec.Metric, spec.Scope, unitsLength),
			"chart_type": spec.ChartType,
			"points":     points,
		})
	}
	return series, warnings
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
	case "speed":
		return speed
	case "efficiency":
		return consumption
	case "energy", "regeneration", "vampire_drain":
		return "kWh"
	case "cost":
		return "currency"
	case "power":
		return "kW"
	case "soc":
		return "%"
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
		case "distance", "range", "speed":
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
	key := aggregateCacheKey("series", carID, scope, metric, bucket, startUTC, endUTC, unitsLength, appUsersTimezone.String())
	return cachedValue(key, aggregateCacheTTL(endUTC), func() ([]map[string]any, error) {
		return fetchMetricSeriesUncached(carID, scope, metric, bucket, startUTC, endUTC, unitsLength)
	})
}

func fetchMetricSeriesUncached(carID int, scope, metric, bucket, startUTC, endUTC, unitsLength string) ([]map[string]any, error) {
	if bucket == "raw" {
		bucket = "hour"
	}
	bucketExpr := "date_trunc('day', timezone($4, start_date))"
	switch bucket {
	case "hour":
		bucketExpr = "date_trunc('hour', timezone($4, start_date))"
	case "week":
		bucketExpr = "date_trunc('week', timezone($4, start_date))"
	case "month":
		bucketExpr = "date_trunc('month', timezone($4, start_date))"
	case "year":
		bucketExpr = "date_trunc('year', timezone($4, start_date))"
	}
	var query string
	switch metric {
	case "distance":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, SUM(distance)::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "efficiency":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, CASE WHEN SUM(distance) > 0 THEN SUM(CASE WHEN (start_rated_range_km-end_rated_range_km) > 0 THEN (start_rated_range_km-end_rated_range_km) * cars.efficiency ELSE 0 END) / SUM(distance) * 1000.0 ELSE NULL END::float8 FROM drives LEFT JOIN cars ON cars.id = drives.car_id WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "energy":
		if scope == "charges" {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, SUM(charge_energy_added)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t`, bucketExpr)
		} else {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, SUM(CASE WHEN (start_rated_range_km-end_rated_range_km) > 0 THEN (start_rated_range_km-end_rated_range_km) * cars.efficiency ELSE 0 END)::float8 FROM drives LEFT JOIN cars ON cars.id = drives.car_id WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3 GROUP BY t ORDER BY t`, bucketExpr)
		}
	case "cost":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, NULLIF(SUM(CASE WHEN cost > 0 THEN cost ELSE 0 END),0)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "speed":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, CASE WHEN SUM(duration_min) > 0 THEN SUM(distance)/(SUM(duration_min)/60.0) ELSE NULL END::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "power":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(NULLIF(charges.charger_power,0))::float8 FROM charging_processes LEFT JOIN charges ON charges.charging_process_id = charging_processes.id WHERE charging_processes.car_id=$1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date < $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "soc":
		if scope == "charges" {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(end_battery_level)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 GROUP BY t ORDER BY t`, bucketExpr)
		} else {
			query = `SELECT TO_CHAR(date_trunc('day', timezone($4, date)), 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(battery_level)::float8 FROM positions WHERE car_id=$1 AND date >= $2 AND date < $3 GROUP BY t ORDER BY t`
		}
	case "range":
		query = `SELECT TO_CHAR(date_trunc('day', timezone($4, date)), 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(rated_battery_range_km)::float8 FROM positions WHERE car_id=$1 AND date >= $2 AND date < $3 AND rated_battery_range_km IS NOT NULL GROUP BY t ORDER BY t`
	case "regeneration":
		query = `WITH regen AS (
			SELECT TO_CHAR(date_trunc('day', timezone($4, positions.date)), 'YYYY-MM-DD"T"HH24:MI:SS') AS t,
				ABS(LEAST(COALESCE(positions.power, 0)::float8, 0)) AS pkw,
				EXTRACT(EPOCH FROM (positions.date - LAG(positions.date) OVER (PARTITION BY drives.id ORDER BY positions.id))) AS ds
			FROM drives INNER JOIN positions ON positions.drive_id = drives.id
			WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
		)
		SELECT t, SUM(pkw * ds / 3600.0)::float8 FROM regen WHERE ds > 0 AND ds <= 300 AND pkw > 0 GROUP BY t ORDER BY t`
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
		sort.SliceStable(out, func(i, j int) bool { return out[i]["time"].(string) < out[j]["time"].(string) })
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
