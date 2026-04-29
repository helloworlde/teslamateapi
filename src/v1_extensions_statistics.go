package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func TeslaMateAPICarsUnifiedStatisticsV2(c *gin.Context) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid statistics range", map[string]any{"reason": err.Error()})
		return
	}
	warnings := []any{}
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
		warnings = append(warnings, nonFatalWarning("regeneration_unavailable", "failed to load regeneration metrics, returned as null", nil, regenErr))
	}
	batterySnapshot, batteryErr := fetchBatterySnapshot(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if batteryErr != nil {
		warnings = append(warnings, nonFatalWarning("battery_snapshot_unavailable", "failed to load battery snapshot, returned as null", nil, batteryErr))
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
		warnings = append(warnings, nonFatalWarning("park_energy_unavailable", "failed to load parking energy, returned as null", nil, parkEnergyErr))
	}
	parkingSummary, parkingErr := fetchParkingHistorySummary(ctx.CarID, startUTC, endUTC, nil)
	if parkingErr != nil {
		warnings = append(warnings, nonFatalWarning("parking_summary_unavailable", "failed to load parking summary", nil, parkingErr))
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

func fetchBatterySnapshot(carID int, startUTC, endUTC, unitsLength string) (map[string]any, error) {
	key := aggregateCacheKey("battery_snapshot", carID, startUTC, endUTC, unitsLength)
	return cachedValue(key, aggregateCacheTTL(endUTC), func() (map[string]any, error) {
		return fetchBatterySnapshotUncached(carID, startUTC, endUTC, unitsLength)
	})
}

func fetchBatterySnapshotUncached(carID int, startUTC, endUTC, unitsLength string) (map[string]any, error) {
	query := `
		WITH start_pos AS (
			SELECT battery_level, rated_battery_range_km
			FROM positions
			WHERE car_id = $1 AND date >= $2 AND date < $3
			ORDER BY date ASC
			LIMIT 1
		),
		end_pos AS (
			SELECT battery_level, rated_battery_range_km
			FROM positions
			WHERE car_id = $1 AND date >= $2 AND date < $3
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
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(queryCtx, query, carID, startUTC, endUTC).Scan(&socStart, &socEnd, &rangeStart, &rangeEnd); err != nil {
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

func fetchParkingEnergyTotal(carID int, startUTC, endUTC string) (*float64, error) {
	key := aggregateCacheKey("parking_energy_total", carID, startUTC, endUTC)
	return cachedValue(key, aggregateCacheTTL(endUTC), func() (*float64, error) {
		return fetchParkingEnergyTotalUncached(carID, startUTC, endUTC)
	})
}

func fetchParkingEnergyTotalUncached(carID int, startUTC, endUTC string) (*float64, error) {
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
				AND COALESCE(s.end_date, NOW() AT TIME ZONE 'UTC') < $3
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
	type cachedBuckets struct {
		Values map[string]*float64
		Total  *float64
	}
	key := aggregateCacheKey("parking_energy_bucket", carID, startUTC, endUTC, trunc, appUsersTimezone.String())
	cached, err := cachedValue(key, aggregateCacheTTL(endUTC), func() (cachedBuckets, error) {
		values, total, err := fetchParkingEnergyByBucketUncached(carID, startUTC, endUTC, trunc, timeout)
		return cachedBuckets{Values: values, Total: total}, err
	})
	if err != nil {
		return nil, nil, err
	}
	return cached.Values, cached.Total, nil
}

func fetchParkingEnergyByBucketUncached(carID int, startUTC, endUTC, trunc string, timeout time.Duration) (map[string]*float64, *float64, error) {
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
				AND COALESCE(s.end_date, NOW() AT TIME ZONE 'UTC') < $3
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
	type cachedBuckets struct {
		Values map[string]*float64
		Total  *float64
	}
	key := aggregateCacheKey("regenerated_energy_bucket", carID, startUTC, endUTC, trunc, appUsersTimezone.String())
	cached, err := cachedValue(key, aggregateCacheTTL(endUTC), func() (cachedBuckets, error) {
		values, total, err := fetchRegeneratedEnergyByBucketUncached(carID, startUTC, endUTC, trunc, timeout)
		return cachedBuckets{Values: values, Total: total}, err
	})
	if err != nil {
		return nil, nil, err
	}
	return cached.Values, cached.Total, nil
}

func fetchRegeneratedEnergyByBucketUncached(carID int, startUTC, endUTC, trunc string, timeout time.Duration) (map[string]*float64, *float64, error) {
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
				AND drives.end_date < $3
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
