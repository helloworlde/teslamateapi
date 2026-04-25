package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
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

var metricRegistry = map[string]metricDef{
	"distance":      {Key: "distance", Name: "distance", Unit: "km", Scope: "drives", ChartType: "bar"},
	"efficiency":    {Key: "efficiency", Name: "efficiency", Unit: "Wh/km", Scope: "drives", ChartType: "line"},
	"speed":         {Key: "speed", Name: "speed", Unit: "km/h", Scope: "drives", ChartType: "line"},
	"energy":        {Key: "energy", Name: "energy", Unit: "kWh", Scope: "drives", ChartType: "area"},
	"cost":          {Key: "cost", Name: "cost", Unit: "currency", Scope: "charges", ChartType: "bar"},
	"power":         {Key: "power", Name: "power", Unit: "kW", Scope: "charges", ChartType: "line"},
	"soc":           {Key: "soc", Name: "soc", Unit: "%", Scope: "battery", ChartType: "line"},
	"range":         {Key: "range", Name: "range", Unit: "km", Scope: "battery", ChartType: "line"},
	"regeneration":  {Key: "regeneration", Name: "regeneration", Unit: "kWh", Scope: "drives", ChartType: "area"},
	"vampire_drain": {Key: "vampire_drain", Name: "vampire_drain", Unit: "kWh", Scope: "states", ChartType: "bar"},
}

func TeslaMateAPICarsDashboardV2(c *gin.Context) {
	dr, warnings := parseDateRangeWithMonthFallback(c, "month")
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
	data := map[string]any{
		"car_id":     ctx.CarID,
		"range":      buildRangeDTO(dr),
		"statistics": statistics,
		"calendar": map[string]any{
			"bucket":  "day",
			"summary": calendarSummary,
			"items":   calendarItems,
		},
		"series":         []any{},
		"distributions":  []any{},
		"insights":       []any{},
		"recent_drives":  []any{},
		"recent_charges": []any{},
	}
	writeV1Object(c, data, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func TeslaMateAPICarsCalendarV2(c *gin.Context) {
	dr, warnings := parseDateRangeWithMonthFallback(c, "custom")
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsCalendarV2")
	if !ok {
		return
	}
	bucket := strings.ToLower(strings.TrimSpace(c.DefaultQuery("bucket", "day")))
	switch bucket {
	case "day", "week", "month":
	default:
		writeV1Error(c, http.StatusBadRequest, "invalid_bucket", "bucket must be day|week|month", nil)
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	metricSet := parseMetricSet(c.Query("metrics"))
	includeRegen := metricSet["regeneration"]
	includePark := metricSet["park_energy"] || metricSet["vampire_drain"]
	items, summary, calendarWarnings, err := fetchUnifiedCalendar(ctx.CarID, startUTC, endUTC, bucket, includeRegen, includePark)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load calendar", map[string]any{"reason": err.Error()})
		return
	}
	warnings = append(warnings, calendarWarnings...)
	resp := map[string]any{
		"car_id":  ctx.CarID,
		"range":   buildRangeDTO(dr),
		"bucket":  bucket,
		"summary": summary,
		"items":   items,
	}
	writeV1Object(c, resp, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func TeslaMateAPICarsUnifiedStatisticsV2(c *gin.Context) {
	dr, warnings := parseDateRangeWithMonthFallback(c, "month")
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsUnifiedStatisticsV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	driveSummary, err := fetchDriveHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load drive summary", map[string]any{"reason": err.Error()})
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load charge summary", map[string]any{"reason": err.Error()})
		return
	}
	statistics, err := fetchStatisticsSummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, ctx.UnitsTemperature, driveSummary, chargeSummary)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load statistics", map[string]any{"reason": err.Error()})
		return
	}
	regeneration, regenErr := fetchRegenerationSummary(ctx.CarID, startUTC, endUTC, driveSummary, ctx.UnitsLength)
	if regenErr != nil {
		warnings = append(warnings, map[string]any{
			"code":    "regeneration_unavailable",
			"message": "failed to load regeneration metrics, returned as null",
			"reason":  regenErr.Error(),
		})
	}
	batterySnapshot, batteryErr := fetchBatterySnapshot(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if batteryErr != nil {
		warnings = append(warnings, map[string]any{
			"code":    "battery_snapshot_unavailable",
			"message": "failed to load battery snapshot, returned as null",
			"reason":  batteryErr.Error(),
		})
		batterySnapshot = map[string]any{
			"soc_start_percent": nil,
			"soc_end_percent":   nil,
			"range_start_km":    nil,
			"range_end_km":      nil,
		}
	}
	currency := "UNKNOWN"
	if statistics.TotalCost == nil || statistics.AverageCostPerKwh == nil {
		warnings = append(warnings, map[string]any{
			"code":    "charge_cost_or_currency_missing",
			"message": "cost metrics require charging_processes.cost and currency setting",
		})
	}
	chargeEfficiencyPercent := toPercent(statistics.ChargingEfficiency)
	regeneratedEnergy := any(nil)
	regenerationRatio := any(nil)
	if regeneration != nil {
		regeneratedEnergy = regeneration.EstimatedRecoveredEnergyKwh
		regenerationRatio = regeneration.RecoveryShare
		if regeneration.MetricsEstimated {
			warnings = append(warnings, map[string]any{
				"code":    "regeneration_estimated",
				"message": "regeneration metrics are estimated from position power samples",
			})
		}
	}
	parkEnergyKwh, parkEnergyErr := fetchParkingEnergyTotal(ctx.CarID, startUTC, endUTC)
	if parkEnergyErr != nil {
		warnings = append(warnings, map[string]any{
			"code":    "park_energy_unavailable",
			"message": "failed to load parking energy, returned as null",
			"reason":  parkEnergyErr.Error(),
		})
	}
	parkingSummary, parkingErr := fetchParkingHistorySummary(ctx.CarID, startUTC, endUTC, nil)
	if parkingErr != nil {
		warnings = append(warnings, map[string]any{
			"code":    "parking_summary_unavailable",
			"message": "failed to load parking summary",
			"reason":  parkingErr.Error(),
		})
	}
	avgDriveDurationSec := any(nil)
	if driveSummary.DriveCount > 0 {
		avgDriveDurationSec = float64(driveSummary.TotalDurationMin*60) / float64(driveSummary.DriveCount)
	}
	avgChargeDurationSec := any(nil)
	if chargeSummary.ChargeCount > 0 {
		avgChargeDurationSec = float64(chargeSummary.TotalDurationMin*60) / float64(chargeSummary.ChargeCount)
	}
	writeV1Object(c, map[string]any{
		"car_id": ctx.CarID,
		"period": dr.Period,
		"range":  buildRangeDTO(dr),
		"overview": map[string]any{
			"trip_count":               driveSummary.DriveCount,
			"charge_count":             chargeSummary.ChargeCount,
			"distance_km":              driveSummary.TotalDistance,
			"drive_duration_s":         driveSummary.TotalDurationMin * 60,
			"charge_duration_s":        chargeSummary.TotalDurationMin * 60,
			"energy_used_kwh":          driveSummary.TotalEnergyConsumed,
			"energy_added_kwh":         chargeSummary.TotalEnergyAdded,
			"park_energy_kwh":          parkEnergyKwh,
			"avg_efficiency_wh_per_km": driveSummary.AverageConsumption,
		},
		"drive": map[string]any{
			"count":                    driveSummary.DriveCount,
			"distance_km":              driveSummary.TotalDistance,
			"duration_s":               driveSummary.TotalDurationMin * 60,
			"avg_duration_s":           avgDriveDurationSec,
			"avg_speed_kmh":            driveSummary.AverageSpeed,
			"max_speed_kmh":            driveSummary.MaxSpeed,
			"avg_efficiency_wh_per_km": driveSummary.AverageConsumption,
			"used_energy_kwh":          driveSummary.TotalEnergyConsumed,
			"regenerated_energy_kwh":   regeneratedEnergy,
			"regeneration_ratio":       regenerationRatio,
		},
		"charge": map[string]any{
			"count":                  chargeSummary.ChargeCount,
			"duration_s":             chargeSummary.TotalDurationMin * 60,
			"avg_duration_s":         avgChargeDurationSec,
			"energy_added_kwh":       chargeSummary.TotalEnergyAdded,
			"charger_energy_kwh":     chargeSummary.TotalEnergyUsed,
			"cost":                   statistics.TotalCost,
			"cost_per_kwh":           statistics.AverageCostPerKwh,
			"cost_per_100_km":        statistics.AverageCostPer100Distance,
			"currency":               currency,
			"avg_power_kw":           chargeSummary.AveragePower,
			"max_power_kw":           chargeSummary.MaxPower,
			"avg_efficiency_percent": chargeEfficiencyPercent,
		},
		"battery": map[string]any{
			"soc_start_percent": batterySnapshot["soc_start_percent"],
			"soc_end_percent":   batterySnapshot["soc_end_percent"],
			"range_start_km":    batterySnapshot["range_start_km"],
			"range_end_km":      batterySnapshot["range_end_km"],
			"vampire_drain_kwh": parkEnergyKwh,
			"park_energy_kwh":   parkEnergyKwh,
			"parking_duration_s": func() any {
				if parkingSummary == nil {
					return nil
				}
				return parkingSummary.TotalDurationMin * 60
			}(),
		},
	}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func TeslaMateAPICarsSeriesV2(c *gin.Context) {
	scope := strings.ToLower(strings.TrimSpace(c.DefaultQuery("scope", "drives")))
	bucket := strings.ToLower(strings.TrimSpace(c.DefaultQuery("bucket", "day")))
	metrics := parseCSV(c.Query("metrics"))
	if len(metrics) == 0 {
		metrics = defaultSeriesMetrics(scope)
	}
	switch bucket {
	case "raw", "hour", "day", "week", "month", "year":
	default:
		writeV1Error(c, http.StatusBadRequest, "invalid_bucket", "bucket must be raw|hour|day|week|month|year", nil)
		return
	}
	dr, warnings := parseDateRangeWithMonthFallback(c, "custom")
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsSeriesV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	series := make([]any, 0, len(metrics))
	for _, metric := range metrics {
		def, ok := metricRegistry[metric]
		if !ok {
			warnings = append(warnings, map[string]any{"code": "unsupported_metric", "message": "unsupported metric", "metric": metric})
			continue
		}
		if def.Scope != scope && scope != "overview" {
			warnings = append(warnings, map[string]any{"code": "scope_metric_mismatch", "message": "metric does not belong to current scope", "scope": scope, "metric": metric})
			continue
		}
		points, err := fetchMetricSeries(ctx.CarID, scope, metric, bucket, startUTC, endUTC)
		if err != nil {
			warnings = append(warnings, map[string]any{"code": "metric_query_failed", "message": err.Error(), "metric": metric})
			continue
		}
		series = append(series, map[string]any{
			"metric":     def.Key,
			"name":       def.Name,
			"unit":       def.Unit,
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

func TeslaMateAPICarsDistributionsV2(c *gin.Context) {
	dr, warnings := parseDateRangeWithMonthFallback(c, "custom")
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsDistributionsV2")
	if !ok {
		return
	}
	metrics := parseCSV(c.Query("metrics"))
	if len(metrics) == 0 {
		metrics = []string{"drive_start_hour", "drive_distance", "drive_duration", "charge_start_hour", "charge_energy"}
	}
	startUTC, endUTC := dbTimeRange(dr)
	distributions := make([]any, 0, len(metrics))
	for _, metric := range metrics {
		item, err := fetchDistribution(ctx.CarID, metric, startUTC, endUTC)
		if err != nil {
			warnings = append(warnings, map[string]any{"code": "distribution_query_failed", "metric": metric, "message": err.Error()})
			continue
		}
		distributions = append(distributions, item)
	}
	writeV1Object(c, map[string]any{
		"car_id":        ctx.CarID,
		"range":         buildRangeDTO(dr),
		"distributions": distributions,
	}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func TeslaMateAPICarsUnifiedInsightsV2(c *gin.Context) {
	dr, warnings := parseDateRangeWithMonthFallback(c, "custom")
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsUnifiedInsightsV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	types := parseCSV(c.Query("types"))
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	insights, insightWarnings := buildSimpleInsights(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, types, limit)
	warnings = append(warnings, insightWarnings...)
	levels := map[string]int{"positive": 0, "warning": 0, "info": 0}
	for _, item := range insights {
		level, _ := item["level"].(string)
		levels[level]++
	}
	writeV1Object(c, map[string]any{
		"car_id": ctx.CarID,
		"range":  buildRangeDTO(dr),
		"summary": map[string]any{
			"positive_count": levels["positive"],
			"warning_count":  levels["warning"],
			"info_count":     levels["info"],
			"total_count":    len(insights),
		},
		"insights": insights,
	}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func TeslaMateAPICarsUnifiedTimelineV2(c *gin.Context) {
	offset, limit, err := parseOffsetLimit(c, 50, 200)
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_pagination", err.Error(), nil)
		return
	}
	dr, warnings := parseDateRangeWithMonthFallback(c, "custom")
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsUnifiedTimelineV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	page := offset/limit + 1
	items, total, err := fetchTimelineEvents(ctx.CarID, startUTC, endUTC, page, limit, "desc")
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load timeline", map[string]any{"reason": err.Error()})
		return
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		entityID := 0
		if id, err := strconv.Atoi(item.SourceID); err == nil {
			entityID = id
		}
		out = append(out, map[string]any{
			"id":          item.ID,
			"type":        item.Type,
			"start_date":  item.StartDate,
			"end_date":    item.EndDate,
			"title":       item.Title,
			"summary":     item.Metrics,
			"entity_type": item.Type,
			"entity_id":   entityID,
		})
	}
	writeV1List(c, out, v1Pagination{Limit: limit, Offset: offset, Total: total}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func TeslaMateAPICarsMapVisitedUnifiedV2(c *gin.Context) {
	dr, warnings := parseDateRangeWithMonthFallback(c, "custom")
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsMapVisitedUnifiedV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	points, bounds, truncated, err := fetchVisitedMap(ctx.CarID, startUTC, endUTC, 10000)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load visited map", map[string]any{"reason": err.Error()})
		return
	}
	visited := map[string]int{}
	for _, p := range points {
		key := fmt.Sprintf("%.4f,%.4f", p.Latitude, p.Longitude)
		visited[key]++
	}
	visitedPoints := make([]any, 0, len(visited))
	for key, count := range visited {
		parts := strings.Split(key, ",")
		lat, _ := strconv.ParseFloat(parts[0], 64)
		lng, _ := strconv.ParseFloat(parts[1], 64)
		visitedPoints = append(visitedPoints, map[string]any{"latitude": lat, "longitude": lng, "count": count})
	}
	sort.SliceStable(visitedPoints, func(i, j int) bool {
		return visitedPoints[i].(map[string]any)["count"].(int) > visitedPoints[j].(map[string]any)["count"].(int)
	})
	data := map[string]any{
		"car_id":         ctx.CarID,
		"range":          buildRangeDTO(dr),
		"distance_km":    nil,
		"drive_count":    nil,
		"visited_points": visitedPoints,
		"heatmap":        []any{},
	}
	if bounds != nil {
		data["bounds"] = map[string]any{
			"north": bounds.MaxLatitude,
			"south": bounds.MinLatitude,
			"east":  bounds.MaxLongitude,
			"west":  bounds.MinLongitude,
		}
	}
	if truncated {
		warnings = append(warnings, map[string]any{"code": "data_truncated", "message": "visited points were truncated to limit 10000"})
	}
	writeV1Object(c, data, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func toPercent(v *float64) *float64 {
	if v == nil {
		return nil
	}
	p := *v * 100
	return &p
}

func fetchBatterySnapshot(carID int, startUTC, endUTC, unitsLength string) (map[string]any, error) {
	query := `
		WITH start_pos AS (
			SELECT battery_level, rated_battery_range_km
			FROM positions
			WHERE car_id = $1 AND date >= $2 AND date <= $3
			ORDER BY date ASC
			LIMIT 1
		),
		end_pos AS (
			SELECT battery_level, rated_battery_range_km
			FROM positions
			WHERE car_id = $1 AND date >= $2 AND date <= $3
			ORDER BY date DESC
			LIMIT 1
		)
		SELECT
			(SELECT battery_level FROM start_pos),
			(SELECT battery_level FROM end_pos),
			(SELECT rated_battery_range_km FROM start_pos),
			(SELECT rated_battery_range_km FROM end_pos)`
	var (
		socStart   sql.NullInt64
		socEnd     sql.NullInt64
		rangeStart sql.NullFloat64
		rangeEnd   sql.NullFloat64
	)
	if err := db.QueryRow(query, carID, startUTC, endUTC).Scan(&socStart, &socEnd, &rangeStart, &rangeEnd); err != nil {
		return nil, err
	}
	startRange := any(nil)
	endRange := any(nil)
	if rangeStart.Valid {
		v := rangeStart.Float64
		if strings.EqualFold(unitsLength, "mi") {
			v = kilometersToMiles(v)
		}
		startRange = v
	}
	if rangeEnd.Valid {
		v := rangeEnd.Float64
		if strings.EqualFold(unitsLength, "mi") {
			v = kilometersToMiles(v)
		}
		endRange = v
	}
	return map[string]any{
		"soc_start_percent": intOrNil(socStart),
		"soc_end_percent":   intOrNil(socEnd),
		"range_start_km":    startRange,
		"range_end_km":      endRange,
	}, nil
}

func intOrNil(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return int(v.Int64)
}

func fetchParkingEnergyTotal(carID int, startUTC, endUTC string) (*float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	query := `
		WITH state_windows AS (
			SELECT
				s.start_date,
				COALESCE(s.end_date, NOW() AT TIME ZONE 'UTC') AS end_date
			FROM states s
			WHERE s.car_id = $1
				AND s.state::text IN ('online', 'offline', 'asleep')
				AND s.start_date >= $2
				AND COALESCE(s.end_date, NOW() AT TIME ZONE 'UTC') <= $3
		),
		energy_rows AS (
			SELECT
				CASE
					WHEN start_pos.rated_battery_range_km IS NOT NULL
						AND end_pos.rated_battery_range_km IS NOT NULL
						AND (start_pos.rated_battery_range_km - end_pos.rated_battery_range_km) > 0
					THEN (start_pos.rated_battery_range_km - end_pos.rated_battery_range_km) * cars.efficiency
					ELSE NULL
				END AS park_energy_kwh
			FROM state_windows w
			LEFT JOIN cars ON cars.id = $1
			LEFT JOIN LATERAL (
				SELECT p.rated_battery_range_km
				FROM positions p
				WHERE p.car_id = $1 AND p.date >= w.start_date AND p.date <= w.end_date
				ORDER BY p.date ASC
				LIMIT 1
			) start_pos ON TRUE
			LEFT JOIN LATERAL (
				SELECT p.rated_battery_range_km
				FROM positions p
				WHERE p.car_id = $1 AND p.date >= w.start_date AND p.date <= w.end_date
				ORDER BY p.date DESC
				LIMIT 1
			) end_pos ON TRUE
		)
		SELECT SUM(park_energy_kwh)::float8 FROM energy_rows`

	var v sql.NullFloat64
	if err := db.QueryRowContext(ctx, query, carID, startUTC, endUTC).Scan(&v); err != nil {
		return nil, err
	}
	if !v.Valid {
		return nil, nil
	}
	return &v.Float64, nil
}

func fetchParkingEnergyByBucketWithTimeout(carID int, startUTC, endUTC, trunc string, timeout time.Duration) (map[string]*float64, *float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	query := fmt.Sprintf(`
		WITH state_windows AS (
			SELECT
				s.start_date,
				COALESCE(s.end_date, NOW() AT TIME ZONE 'UTC') AS end_date
			FROM states s
			WHERE s.car_id = $1
				AND s.state::text IN ('online', 'offline', 'asleep')
				AND s.start_date >= $2
				AND COALESCE(s.end_date, NOW() AT TIME ZONE 'UTC') <= $3
		),
		energy_rows AS (
			SELECT
				TO_CHAR(date_trunc('%s', timezone($4, w.start_date)), 'YYYY-MM-DD') AS bucket_date,
				CASE
					WHEN start_pos.rated_battery_range_km IS NOT NULL
						AND end_pos.rated_battery_range_km IS NOT NULL
						AND (start_pos.rated_battery_range_km - end_pos.rated_battery_range_km) > 0
					THEN (start_pos.rated_battery_range_km - end_pos.rated_battery_range_km) * cars.efficiency
					ELSE NULL
				END AS park_energy_kwh
			FROM state_windows w
			LEFT JOIN cars ON cars.id = $1
			LEFT JOIN LATERAL (
				SELECT p.rated_battery_range_km
				FROM positions p
				WHERE p.car_id = $1 AND p.date >= w.start_date AND p.date <= w.end_date
				ORDER BY p.date ASC
				LIMIT 1
			) start_pos ON TRUE
			LEFT JOIN LATERAL (
				SELECT p.rated_battery_range_km
				FROM positions p
				WHERE p.car_id = $1 AND p.date >= w.start_date AND p.date <= w.end_date
				ORDER BY p.date DESC
				LIMIT 1
			) end_pos ON TRUE
		)
		SELECT bucket_date, SUM(park_energy_kwh)::float8
		FROM energy_rows
		GROUP BY bucket_date
		ORDER BY bucket_date ASC`, trunc)

	rows, err := db.QueryContext(ctx, query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	result := map[string]*float64{}
	var total float64
	var hasTotal bool
	for rows.Next() {
		var bucket string
		var v sql.NullFloat64
		if err := rows.Scan(&bucket, &v); err != nil {
			return nil, nil, err
		}
		if v.Valid {
			value := v.Float64
			result[bucket] = &value
			total += value
			hasTotal = true
		} else {
			result[bucket] = nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if !hasTotal {
		return result, nil, nil
	}
	return result, &total, nil
}

func fetchRegeneratedEnergyByBucketWithTimeout(carID int, startUTC, endUTC, trunc string, timeout time.Duration) (map[string]*float64, *float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	query := fmt.Sprintf(`
		WITH regen_samples AS (
			SELECT
				TO_CHAR(date_trunc('%s', timezone($4, positions.date)), 'YYYY-MM-DD') AS bucket_date,
				ABS(LEAST(COALESCE(positions.power, 0)::float8, 0)) AS regen_power_kw,
				EXTRACT(EPOCH FROM (positions.date - LAG(positions.date) OVER (PARTITION BY drives.id ORDER BY positions.id))) AS delta_sec
			FROM drives
			INNER JOIN positions ON positions.drive_id = drives.id
			WHERE drives.car_id = $1
				AND drives.end_date IS NOT NULL
				AND drives.start_date >= $2
				AND drives.end_date <= $3
		)
		SELECT
			bucket_date,
			SUM(regen_power_kw * delta_sec / 3600.0)::float8 AS regenerated_energy_kwh
		FROM regen_samples
		WHERE delta_sec > 0 AND delta_sec <= 300 AND regen_power_kw > 0
		GROUP BY bucket_date
		ORDER BY bucket_date ASC`, trunc)
	rows, err := db.QueryContext(ctx, query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	result := map[string]*float64{}
	var total float64
	var hasTotal bool
	for rows.Next() {
		var bucket string
		var v sql.NullFloat64
		if err := rows.Scan(&bucket, &v); err != nil {
			return nil, nil, err
		}
		if v.Valid {
			value := v.Float64
			result[bucket] = &value
			total += value
			hasTotal = true
		} else {
			result[bucket] = nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if !hasTotal {
		return result, nil, nil
	}
	return result, &total, nil
}

func parseCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		v := strings.ToLower(strings.TrimSpace(p))
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func parseMetricSet(raw string) map[string]bool {
	set := map[string]bool{}
	for _, item := range parseCSV(raw) {
		set[item] = true
	}
	return set
}

func defaultSeriesMetrics(scope string) []string {
	switch scope {
	case "drives":
		return []string{"distance", "speed", "efficiency", "energy"}
	case "charges":
		return []string{"energy", "power", "cost", "soc"}
	case "battery":
		return []string{"soc", "range"}
	case "states":
		return []string{"vampire_drain"}
	case "overview":
		return []string{"distance", "energy", "cost", "efficiency"}
	default:
		return []string{"distance"}
	}
}

func fetchUnifiedCalendar(carID int, startUTC, endUTC, bucket string, includeRegen bool, includePark bool) ([]any, map[string]any, []any, error) {
	trunc := "day"
	switch bucket {
	case "week":
		trunc = "week"
	case "month":
		trunc = "month"
	}
	query := fmt.Sprintf(`
		WITH drives_agg AS (
			SELECT date_trunc('%s', timezone($4, start_date)) AS bucket,
				COUNT(*)::int AS drive_count,
				COALESCE(SUM(distance), 0)::float8 AS distance_km,
				COALESCE(SUM(duration_min), 0)::float8 * 60 AS duration_s,
				COALESCE(SUM(
					CASE
						WHEN (start_rated_range_km - end_rated_range_km) > 0 THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency
						ELSE 0
					END
				), 0)::float8 AS drive_energy_kwh
			FROM drives
			LEFT JOIN cars ON cars.id = drives.car_id
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3
			GROUP BY bucket
		),
		charges_agg AS (
			SELECT date_trunc('%s', timezone($4, start_date)) AS bucket,
				COUNT(*)::int AS charge_count,
				COALESCE(SUM(charge_energy_added), 0)::float8 AS charge_energy_kwh,
				NULLIF(SUM(CASE WHEN cost > 0 THEN cost ELSE 0 END), 0)::float8 AS charge_cost
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3
			GROUP BY bucket
		)
		SELECT TO_CHAR(COALESCE(d.bucket, c.bucket), 'YYYY-MM-DD') AS bucket_date,
			COALESCE(d.drive_count, 0),
			COALESCE(c.charge_count, 0),
			COALESCE(d.distance_km, 0),
			COALESCE(d.duration_s, 0),
			COALESCE(d.drive_energy_kwh, 0),
			c.charge_energy_kwh,
			c.charge_cost
		FROM drives_agg d
		FULL JOIN charges_agg c ON d.bucket = c.bucket
		ORDER BY bucket_date ASC`, trunc, trunc)
	rows, err := db.Query(query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	items := make([]any, 0)
	warnings := make([]any, 0)
	summary := map[string]any{
		"drive_days":               0,
		"charge_days":              0,
		"drive_count":              0,
		"charge_count":             0,
		"distance_km":              0.0,
		"drive_duration_s":         0.0,
		"avg_efficiency_wh_per_km": nil,
		"avg_speed_kmh":            nil,
		"charge_energy_kwh":        0.0,
		"charge_cost":              nil,
		"used_energy_kwh":          0.0,
		"drive_energy_kwh":         0.0,
		"park_energy_kwh":          nil,
		"regenerated_energy_kwh":   nil,
	}
	totalCost := 0.0
	hasCost := false
	for rows.Next() {
		var date string
		var driveCount int
		var chargeCount int
		var distanceKm float64
		var durationSec float64
		var driveEnergyKwh float64
		var chargeEnergy sql.NullFloat64
		var chargeCost sql.NullFloat64
		if err := rows.Scan(&date, &driveCount, &chargeCount, &distanceKm, &durationSec, &driveEnergyKwh, &chargeEnergy, &chargeCost); err != nil {
			return nil, nil, nil, err
		}
		if driveCount > 0 {
			summary["drive_days"] = summary["drive_days"].(int) + 1
		}
		if chargeCount > 0 {
			summary["charge_days"] = summary["charge_days"].(int) + 1
		}
		summary["drive_count"] = summary["drive_count"].(int) + driveCount
		summary["charge_count"] = summary["charge_count"].(int) + chargeCount
		summary["distance_km"] = summary["distance_km"].(float64) + distanceKm
		summary["drive_duration_s"] = summary["drive_duration_s"].(float64) + durationSec
		summary["drive_energy_kwh"] = summary["drive_energy_kwh"].(float64) + driveEnergyKwh
		summary["used_energy_kwh"] = summary["used_energy_kwh"].(float64) + driveEnergyKwh
		chargeEnergyVal := any(nil)
		chargeCostVal := any(nil)
		if chargeEnergy.Valid {
			chargeEnergyVal = chargeEnergy.Float64
			summary["charge_energy_kwh"] = summary["charge_energy_kwh"].(float64) + chargeEnergy.Float64
		}
		if chargeCost.Valid {
			chargeCostVal = chargeCost.Float64
			totalCost += chargeCost.Float64
			hasCost = true
		}
		var avgSpeed any
		if durationSec > 0 {
			avgSpeed = distanceKm / (durationSec / 3600.0)
		}
		var avgEfficiency any
		if distanceKm > 0 && driveEnergyKwh > 0 {
			avgEfficiency = driveEnergyKwh * 1000.0 / distanceKm
		}
		badges := []any{}
		if driveCount > 0 {
			badges = append(badges, map[string]any{
				"type":  "drive",
				"label": fmt.Sprintf("%.1fkm", distanceKm),
				"value": distanceKm,
				"unit":  "km",
			})
		}
		if chargeCount > 0 && chargeEnergy.Valid {
			badges = append(badges, map[string]any{
				"type":  "charge",
				"label": fmt.Sprintf("%.1fkWh", chargeEnergy.Float64),
				"value": chargeEnergy.Float64,
				"unit":  "kWh",
			})
		}
		items = append(items, map[string]any{
			"date":                     date,
			"has_drive":                driveCount > 0,
			"has_charge":               chargeCount > 0,
			"drive_count":              driveCount,
			"charge_count":             chargeCount,
			"distance_km":              distanceKm,
			"drive_duration_s":         durationSec,
			"avg_speed_kmh":            avgSpeed,
			"avg_efficiency_wh_per_km": avgEfficiency,
			"charge_energy_kwh":        chargeEnergyVal,
			"charge_cost":              chargeCostVal,
			"used_energy_kwh":          driveEnergyKwh,
			"drive_energy_kwh":         driveEnergyKwh,
			"park_energy_kwh":          nil,
			"regenerated_energy_kwh":   nil,
			"badges":                   badges,
		})
	}
	if hasCost {
		summary["charge_cost"] = totalCost
	}
	if summary["distance_km"].(float64) > 0 && summary["drive_energy_kwh"].(float64) > 0 {
		v := summary["drive_energy_kwh"].(float64) * 1000.0 / summary["distance_km"].(float64)
		summary["avg_efficiency_wh_per_km"] = v
	}
	if summary["drive_duration_s"].(float64) > 0 {
		v := summary["distance_km"].(float64) / (summary["drive_duration_s"].(float64) / 3600.0)
		summary["avg_speed_kmh"] = v
	}
	if includePark {
		parkByDate, parkTotal, parkErr := fetchParkingEnergyByBucketWithTimeout(carID, startUTC, endUTC, trunc, 1200*time.Millisecond)
		if parkErr != nil {
			warnings = append(warnings, map[string]any{
				"code":    "park_energy_timeout",
				"message": "park energy query timed out, returned as null to keep endpoint responsive",
				"reason":  parkErr.Error(),
			})
		} else {
			if parkTotal != nil {
				summary["park_energy_kwh"] = *parkTotal
			}
			for i := range items {
				m, ok := items[i].(map[string]any)
				if !ok {
					continue
				}
				date, _ := m["date"].(string)
				if v, ok := parkByDate[date]; ok && v != nil {
					m["park_energy_kwh"] = *v
				}
			}
		}
	}
	if includeRegen {
		regenByDate, regenTotal, regenErr := fetchRegeneratedEnergyByBucketWithTimeout(carID, startUTC, endUTC, trunc, 1200*time.Millisecond)
		if regenErr != nil {
			warnings = append(warnings, map[string]any{
				"code":    "regeneration_timeout",
				"message": "regeneration query timed out, returned as null to keep endpoint responsive",
				"reason":  regenErr.Error(),
			})
		} else {
			if regenTotal != nil {
				summary["regenerated_energy_kwh"] = *regenTotal
			}
			for i := range items {
				m, ok := items[i].(map[string]any)
				if !ok {
					continue
				}
				date, _ := m["date"].(string)
				if v, ok := regenByDate[date]; ok && v != nil {
					m["regenerated_energy_kwh"] = *v
				}
			}
		}
	}
	return items, summary, warnings, rows.Err()
}

func fetchMetricSeries(carID int, scope, metric, bucket, startUTC, endUTC string) ([]map[string]any, error) {
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
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, SUM(distance)::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "efficiency":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, CASE WHEN SUM(distance) > 0 THEN SUM(CASE WHEN (start_rated_range_km-end_rated_range_km) > 0 THEN (start_rated_range_km-end_rated_range_km) * cars.efficiency ELSE 0 END) / SUM(distance) * 1000.0 ELSE NULL END::float8 FROM drives LEFT JOIN cars ON cars.id = drives.car_id WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date <= $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "energy":
		if scope == "charges" {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, SUM(charge_energy_added)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3 GROUP BY t ORDER BY t`, bucketExpr)
		} else {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, SUM(CASE WHEN (start_rated_range_km-end_rated_range_km) > 0 THEN (start_rated_range_km-end_rated_range_km) * cars.efficiency ELSE 0 END)::float8 FROM drives LEFT JOIN cars ON cars.id = drives.car_id WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date <= $3 GROUP BY t ORDER BY t`, bucketExpr)
		}
	case "cost":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, NULLIF(SUM(CASE WHEN cost > 0 THEN cost ELSE 0 END),0)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "speed":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, CASE WHEN SUM(duration_min) > 0 THEN SUM(distance)/(SUM(duration_min)/60.0) ELSE NULL END::float8 FROM drives WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "power":
		query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(NULLIF(charges.charger_power,0))::float8 FROM charging_processes LEFT JOIN charges ON charges.charging_process_id = charging_processes.id WHERE charging_processes.car_id=$1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date <= $3 GROUP BY t ORDER BY t`, bucketExpr)
	case "soc":
		if scope == "charges" {
			query = fmt.Sprintf(`SELECT TO_CHAR(%s, 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(end_battery_level)::float8 FROM charging_processes WHERE car_id=$1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3 GROUP BY t ORDER BY t`, bucketExpr)
		} else {
			query = `SELECT TO_CHAR(date_trunc('day', timezone($4, date)), 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(battery_level)::float8 FROM positions WHERE car_id=$1 AND date >= $2 AND date <= $3 GROUP BY t ORDER BY t`
		}
	case "range":
		query = `SELECT TO_CHAR(date_trunc('day', timezone($4, date)), 'YYYY-MM-DD"T"HH24:MI:SS') AS t, AVG(rated_battery_range_km)::float8 FROM positions WHERE car_id=$1 AND date >= $2 AND date <= $3 AND rated_battery_range_km IS NOT NULL GROUP BY t ORDER BY t`
	case "regeneration":
		query = `WITH regen AS (
			SELECT TO_CHAR(date_trunc('day', timezone($4, positions.date)), 'YYYY-MM-DD"T"HH24:MI:SS') AS t,
				ABS(LEAST(COALESCE(positions.power, 0)::float8, 0)) AS pkw,
				EXTRACT(EPOCH FROM (positions.date - LAG(positions.date) OVER (PARTITION BY drives.id ORDER BY positions.id))) AS ds
			FROM drives INNER JOIN positions ON positions.drive_id = drives.id
			WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date <= $3
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
			out = append(out, map[string]any{"time": getTimeInTimeZone(k + "T00:00:00Z"), "value": val})
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i]["time"].(string) < out[j]["time"].(string) })
		return out, nil
	default:
		return []map[string]any{}, nil
	}
	rows, err := db.Query(query, carID, startUTC, endUTC, appUsersTimezone.String())
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
		result = append(result, map[string]any{"time": getTimeInTimeZone(t + "Z"), "value": val})
	}
	return result, rows.Err()
}

func fetchDistribution(carID int, metric, startUTC, endUTC string) (map[string]any, error) {
	switch metric {
	case "drive_start_hour":
		rows, err := db.Query(`
			SELECT EXTRACT(HOUR FROM timezone($4, start_date))::int AS hour, COUNT(*)::int
			FROM drives
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3
			GROUP BY hour ORDER BY hour ASC`, carID, startUTC, endUTC, appUsersTimezone.String())
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		buckets := make([]any, 0)
		for rows.Next() {
			var hour, count int
			if err := rows.Scan(&hour, &count); err != nil {
				return nil, err
			}
			buckets = append(buckets, map[string]any{
				"label": fmt.Sprintf("%02d:00-%02d:00", hour, (hour+1)%24),
				"from":  hour,
				"to":    hour + 1,
				"count": count,
				"value": count,
			})
		}
		return map[string]any{
			"metric":     metric,
			"name":       "drive_start_hour",
			"unit":       "count",
			"chart_type": "bar",
			"buckets":    buckets,
		}, rows.Err()
	case "drive_duration":
		return fetchNumericDistribution(`
			SELECT CASE
				WHEN duration_min < 10 THEN '0-10'
				WHEN duration_min < 20 THEN '10-20'
				WHEN duration_min < 40 THEN '20-40'
				WHEN duration_min < 60 THEN '40-60'
				ELSE '60+'
			END AS label,
			COUNT(*)::int
			FROM drives
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3
			GROUP BY label
			ORDER BY label`, carID, startUTC, endUTC, metric, "drive_duration", "count")
	case "drive_distance":
		return fetchNumericDistribution(`
			SELECT CASE
				WHEN distance < 5 THEN '0-5'
				WHEN distance < 10 THEN '5-10'
				WHEN distance < 20 THEN '10-20'
				WHEN distance < 50 THEN '20-50'
				ELSE '50+'
			END AS label,
			COUNT(*)::int
			FROM drives
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3
			GROUP BY label
			ORDER BY label`, carID, startUTC, endUTC, metric, "drive_distance", "count")
	case "drive_speed":
		return fetchNumericDistribution(`
			SELECT CASE
				WHEN speed_max < 20 THEN '0-20'
				WHEN speed_max < 40 THEN '20-40'
				WHEN speed_max < 60 THEN '40-60'
				WHEN speed_max < 100 THEN '60-100'
				ELSE '100+'
			END AS label,
			COUNT(*)::int
			FROM drives
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3
			GROUP BY label
			ORDER BY label`, carID, startUTC, endUTC, metric, "drive_speed", "count")
	case "charge_start_hour":
		rows, err := db.Query(`
			SELECT EXTRACT(HOUR FROM timezone($4, start_date))::int AS hour, COUNT(*)::int
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3
			GROUP BY hour ORDER BY hour ASC`, carID, startUTC, endUTC, appUsersTimezone.String())
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		buckets := make([]any, 0)
		for rows.Next() {
			var hour, count int
			if err := rows.Scan(&hour, &count); err != nil {
				return nil, err
			}
			buckets = append(buckets, map[string]any{
				"label": fmt.Sprintf("%02d:00-%02d:00", hour, (hour+1)%24),
				"from":  hour,
				"to":    hour + 1,
				"count": count,
				"value": count,
			})
		}
		return map[string]any{
			"metric":     metric,
			"name":       "charge_start_hour",
			"unit":       "count",
			"chart_type": "bar",
			"buckets":    buckets,
		}, rows.Err()
	case "charge_duration":
		return fetchNumericDistribution(`
			SELECT CASE
				WHEN duration_min < 30 THEN '0-30'
				WHEN duration_min < 60 THEN '30-60'
				WHEN duration_min < 120 THEN '60-120'
				WHEN duration_min < 240 THEN '120-240'
				ELSE '240+'
			END AS label,
			COUNT(*)::int
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3
			GROUP BY label
			ORDER BY label`, carID, startUTC, endUTC, metric, "charge_duration", "count")
	case "charge_energy":
		return fetchNumericDistribution(`
			SELECT CASE
				WHEN charge_energy_added < 10 THEN '0-10'
				WHEN charge_energy_added < 20 THEN '10-20'
				WHEN charge_energy_added < 40 THEN '20-40'
				WHEN charge_energy_added < 70 THEN '40-70'
				ELSE '70+'
			END AS label,
			COUNT(*)::int
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date <= $3
			GROUP BY label
			ORDER BY label`, carID, startUTC, endUTC, metric, "charge_energy", "count")
	case "charge_power":
		return fetchNumericDistribution(`
			SELECT CASE
				WHEN COALESCE(charger_power,0) < 4 THEN '0-4'
				WHEN charger_power < 8 THEN '4-8'
				WHEN charger_power < 12 THEN '8-12'
				WHEN charger_power < 50 THEN '12-50'
				ELSE '50+'
			END AS label,
			COUNT(*)::int
			FROM charges
			INNER JOIN charging_processes ON charging_processes.id = charges.charging_process_id
			WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date <= $3
			GROUP BY label
			ORDER BY label`, carID, startUTC, endUTC, metric, "charge_power", "count")
	default:
		return map[string]any{
			"metric":     metric,
			"name":       metric,
			"unit":       "count",
			"chart_type": "bar",
			"buckets":    []any{},
		}, nil
	}
}

func fetchNumericDistribution(query string, carID int, startUTC, endUTC, metric, name, unit string) (map[string]any, error) {
	rows, err := db.Query(query, carID, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	buckets := make([]any, 0)
	for rows.Next() {
		var label string
		var count int
		if err := rows.Scan(&label, &count); err != nil {
			return nil, err
		}
		buckets = append(buckets, map[string]any{
			"label": label,
			"count": count,
			"value": count,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{
		"metric":     metric,
		"name":       name,
		"unit":       unit,
		"chart_type": "bar",
		"buckets":    buckets,
	}, nil
}

func buildSimpleInsights(carID int, startUTC, endUTC, unitsLength string, types []string, limit int) ([]map[string]any, []any) {
	items := make([]map[string]any, 0)
	warnings := make([]any, 0)
	typeSet := map[string]bool{}
	for _, t := range types {
		typeSet[t] = true
	}
	accept := func(tp string) bool {
		if len(typeSet) == 0 {
			return true
		}
		return typeSet[tp]
	}
	appendInsight := func(id, tp, level, title, desc string, current any, baseline any, related map[string]any) {
		if len(items) >= limit || !accept(tp) {
			return
		}
		items = append(items, map[string]any{
			"id":          id,
			"type":        tp,
			"level":       level,
			"title":       title,
			"description": desc,
			"metrics": map[string]any{
				"current":       current,
				"baseline":      baseline,
				"delta_percent": calcDeltaPercent(current, baseline),
			},
			"related": related,
		})
	}

	currentDrive, err := fetchDriveHistorySummary(carID, startUTC, endUTC, unitsLength)
	if err != nil {
		warnings = append(warnings, map[string]any{"code": "insight_drive_unavailable", "message": err.Error()})
		return items, warnings
	}
	currentCharge, err := fetchChargeHistorySummary(carID, startUTC, endUTC, unitsLength)
	if err != nil {
		warnings = append(warnings, map[string]any{"code": "insight_charge_unavailable", "message": err.Error()})
		return items, warnings
	}
	currentRegen, regenErr := fetchRegenerationSummary(carID, startUTC, endUTC, currentDrive, unitsLength)
	if regenErr != nil {
		warnings = append(warnings, map[string]any{"code": "insight_regen_unavailable", "message": regenErr.Error()})
	}
	currentPark, parkErr := fetchParkingEnergyTotal(carID, startUTC, endUTC)
	if parkErr != nil {
		warnings = append(warnings, map[string]any{"code": "insight_park_unavailable", "message": parkErr.Error()})
	}

	startT, startErr := time.ParseInLocation(dbTimestampFormat, startUTC, time.UTC)
	endT, endErr := time.ParseInLocation(dbTimestampFormat, endUTC, time.UTC)
	if startErr != nil || endErr != nil || !endT.After(startT) {
		warnings = append(warnings, map[string]any{"code": "insight_baseline_unavailable", "message": "invalid range for baseline comparison"})
		return items, warnings
	}
	duration := endT.Sub(startT)
	baseStart := startT.Add(-duration)
	baseEnd := startT
	baseStartUTC := baseStart.UTC().Format(dbTimestampFormat)
	baseEndUTC := baseEnd.UTC().Format(dbTimestampFormat)

	baseDrive, driveBaseErr := fetchDriveHistorySummary(carID, baseStartUTC, baseEndUTC, unitsLength)
	if driveBaseErr != nil {
		warnings = append(warnings, map[string]any{"code": "insight_drive_baseline_unavailable", "message": driveBaseErr.Error()})
	}
	baseCharge, chargeBaseErr := fetchChargeHistorySummary(carID, baseStartUTC, baseEndUTC, unitsLength)
	if chargeBaseErr != nil {
		warnings = append(warnings, map[string]any{"code": "insight_charge_baseline_unavailable", "message": chargeBaseErr.Error()})
	}
	var baseRegen *RegenerationSummary
	if driveBaseErr == nil {
		baseRegen, _ = fetchRegenerationSummary(carID, baseStartUTC, baseEndUTC, baseDrive, unitsLength)
	}
	basePark, _ := fetchParkingEnergyTotal(carID, baseStartUTC, baseEndUTC)

	if currentDrive.AverageConsumption != nil && baseDrive != nil && baseDrive.AverageConsumption != nil {
		cur := *currentDrive.AverageConsumption
		base := *baseDrive.AverageConsumption
		if base > 0 && cur >= base*1.1 {
			appendInsight("drive_efficiency_worse", "efficiency", "warning", "drive_efficiency_worse", "average efficiency worsened versus baseline period", cur, base, map[string]any{"entity_type": "drive"})
		}
		if base > 0 && cur <= base*0.9 {
			appendInsight("drive_efficiency_better", "efficiency", "positive", "drive_efficiency_better", "average efficiency improved versus baseline period", cur, base, map[string]any{"entity_type": "drive"})
		}
	}
	if currentCharge.AverageCostPerKwh != nil && baseCharge != nil && baseCharge.AverageCostPerKwh != nil {
		cur := *currentCharge.AverageCostPerKwh
		base := *baseCharge.AverageCostPerKwh
		if base > 0 && cur >= base*1.15 {
			appendInsight("charge_cost_higher", "cost", "warning", "charge_cost_higher", "charging unit cost significantly higher than baseline", cur, base, map[string]any{"entity_type": "charge"})
		}
		if base > 0 && cur <= base*0.85 {
			appendInsight("charge_cost_lower", "cost", "positive", "charge_cost_lower", "charging unit cost significantly lower than baseline", cur, base, map[string]any{"entity_type": "charge"})
		}
	}
	if currentCharge.ChargingEfficiency != nil && baseCharge != nil && baseCharge.ChargingEfficiency != nil {
		cur := *currentCharge.ChargingEfficiency * 100.0
		base := *baseCharge.ChargingEfficiency * 100.0
		if cur < base-5 {
			appendInsight("charge_efficiency_drop", "charging", "warning", "charge_efficiency_drop", "charging efficiency dropped over 5 points", cur, base, map[string]any{"entity_type": "charge"})
		}
	}
	if currentRegen != nil && currentRegen.RecoveryShare != nil && baseRegen != nil && baseRegen.RecoveryShare != nil {
		cur := *currentRegen.RecoveryShare
		base := *baseRegen.RecoveryShare
		if base > 0 && cur >= base*1.1 {
			appendInsight("regen_share_higher", "driving", "positive", "regen_share_higher", "regeneration share increased versus baseline", cur, base, map[string]any{"entity_type": "drive"})
		}
	}
	if currentPark != nil && basePark != nil {
		cur := *currentPark
		base := *basePark
		if base > 0 && cur >= base*1.2 {
			appendInsight("vampire_drain_higher", "battery", "warning", "vampire_drain_higher", "parking energy loss increased versus baseline", cur, base, map[string]any{"entity_type": "state"})
		}
	}
	if currentDrive.DriveCount > 0 {
		ratio := float64(currentDrive.LowSpeedTripCount) / float64(currentDrive.DriveCount)
		if ratio >= 0.45 {
			appendInsight("traffic_ratio_high", "anomaly", "info", "traffic_ratio_high", "high share of low-speed trips indicates congestion", ratio, nil, map[string]any{"entity_type": "drive"})
		}
	}
	if currentCharge.ChargeCount > 0 {
		days := duration.Hours() / 24.0
		if days > 0 {
			freq := float64(currentCharge.ChargeCount) / days
			if freq >= 1.2 {
				appendInsight("charge_frequency_high", "charging", "info", "charge_frequency_high", "charging frequency is high for this range", freq, nil, map[string]any{"entity_type": "charge"})
			}
		}
	}
	return items, warnings
}

func calcDeltaPercent(current any, baseline any) any {
	cur, ok1 := asFloat64(current)
	base, ok2 := asFloat64(baseline)
	if !ok1 || !ok2 || base == 0 {
		return nil
	}
	return (cur - base) / base * 100.0
}

func asFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case *float64:
		if t == nil {
			return 0, false
		}
		return *t, true
	case int:
		return float64(t), true
	case *int:
		if t == nil {
			return 0, false
		}
		return float64(*t), true
	default:
		return 0, false
	}
}
