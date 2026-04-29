package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func TeslaMateAPICarsCalendarV2(c *gin.Context) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid calendar range", map[string]any{"reason": err.Error()})
		return
	}
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
	items, summary, err := fetchUnifiedCalendar(ctx.CarID, startUTC, endUTC, bucket, includeRegen, includePark)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load calendar", map[string]any{"reason": err.Error()})
		return
	}
	resp := map[string]any{
		"car_id":  ctx.CarID,
		"range":   buildRangeDTO(dr),
		"bucket":  bucket,
		"summary": summary,
		"items":   items,
	}
	writeV1Object(c, resp, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"))
}

func fetchUnifiedCalendar(carID int, startUTC, endUTC, bucket string, includeRegen bool, includePark bool) ([]any, map[string]any, error) {
	// 日历聚合属于历史数据读多写少场景，按车辆、时间范围、bucket 和可选指标缓存。
	type cachedCalendar struct {
		items   []any
		summary map[string]any
	}
	key := aggregateCacheKey("calendar", carID, startUTC, endUTC, bucket, includeRegen, includePark, appUsersTimezone.String())
	cached, err := cachedValue(key, aggregateCacheTTL(endUTC), func() (cachedCalendar, error) {
		items, summary, err := fetchUnifiedCalendarUncached(carID, startUTC, endUTC, bucket, includeRegen, includePark)
		return cachedCalendar{items: items, summary: summary}, err
	})
	if err != nil {
		return nil, nil, err
	}
	return cached.items, cached.summary, nil
}

func fetchUnifiedCalendarUncached(carID int, startUTC, endUTC, bucket string, includeRegen bool, includePark bool) ([]any, map[string]any, error) {
	trunc := "day"
	switch bucket {
	case "week":
		trunc = "week"
	case "month":
		trunc = "month"
	}
	// SQL 说明：
	// 1. drives_agg 与 charges_agg 先按环境时区截断到 day/week/month；
	// 2. FULL JOIN 保留只有行程或只有充电的日期；
	// 3. 最终按 bucket_date DESC 返回，保证时间类响应默认倒序。
	query := fmt.Sprintf(`
		WITH drives_agg AS (
			-- 行程侧聚合：次数、距离、时长、估算耗电量
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
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
			GROUP BY bucket
		),
		charges_agg AS (
			-- 充电侧聚合：次数、补能、费用
			SELECT date_trunc('%s', timezone($4, start_date)) AS bucket,
				COUNT(*)::int AS charge_count,
				COALESCE(SUM(charge_energy_added), 0)::float8 AS charge_energy_kwh,
				NULLIF(SUM(CASE WHEN cost > 0 THEN cost ELSE 0 END), 0)::float8 AS charge_cost
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
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
		ORDER BY bucket_date DESC`, trunc, trunc)
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := make([]any, 0)
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
			return nil, nil, err
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
		if parkErr == nil {
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
		if regenErr == nil {
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
	return items, summary, rows.Err()
}
