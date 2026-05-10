package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// @name PostgresV2BatteryRepository
type PostgresV2BatteryRepository struct {
	db *sql.DB
}

func NewPostgresV2BatteryRepository(db *sql.DB) PostgresV2BatteryRepository {
	return PostgresV2BatteryRepository{db: db}
}

func (r PostgresV2BatteryRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

// Summary returns battery stats for the period.
// Latest battery values come from a single point query on positions (LIMIT 1, uses index).
// Sample count and baseline use drives + start_position PK join (avoids full positions scan).
func (r PostgresV2BatteryRepository) Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2BatteryAnalyticsSummary, V2BatteryStats, error) {
	var summary V2BatteryAnalyticsSummary
	var stats V2BatteryStats

	// Latest values: single-row point query, relies on (car_id, date DESC) index
	var latestBattery sql.NullInt64
	var latestRatedRange, latestIdealRange sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT battery_level, rated_battery_range_km, ideal_battery_range_km
		FROM positions
		WHERE car_id = $1 AND date >= $2 AND date < $3
		ORDER BY date DESC LIMIT 1`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	).Scan(&latestBattery, &latestRatedRange, &latestIdealRange)
	if err != nil && err != sql.ErrNoRows {
		return summary, stats, err
	}
	if latestBattery.Valid {
		summary.LatestBatteryLevelPercent = &latestBattery.Int64
	}
	if latestRatedRange.Valid {
		summary.LatestRatedRangeKM = &latestRatedRange.Float64
	}
	if latestIdealRange.Valid {
		summary.LatestIdealRangeKM = &latestIdealRange.Float64
	}

	// Estimated full range from latest observation
	if latestBattery.Valid && latestBattery.Int64 > 0 && latestRatedRange.Valid {
		v := latestRatedRange.Float64 / float64(latestBattery.Int64) * 100
		summary.EstimatedRatedRangeAt100PercentKM = &v
	}
	if latestBattery.Valid && latestBattery.Int64 > 0 && latestIdealRange.Valid {
		v := latestIdealRange.Float64 / float64(latestBattery.Int64) * 100
		summary.EstimatedIdealRangeAt100PercentKM = &v
	}

	// Sample count + baseline from drives + start_position PK join (one position per drive, fast)
	var sampleCount int64
	var baselineRated, baselineIdeal sql.NullFloat64
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE sp.battery_level > 0 AND drives.start_rated_range_km IS NOT NULL) AS sample_count,
			MAX(CASE WHEN sp.battery_level > 0 AND drives.start_rated_range_km IS NOT NULL
				THEN drives.start_rated_range_km / sp.battery_level * 100 END) AS baseline_rated,
			MAX(CASE WHEN sp.battery_level > 0 AND drives.start_rated_range_km IS NOT NULL
				THEN drives.start_rated_range_km / sp.battery_level * 100 END) AS baseline_ideal
		FROM drives
		LEFT JOIN positions sp ON sp.id = drives.start_position_id
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL`,
		carID,
	).Scan(&sampleCount, &baselineRated, &baselineIdeal)
	if err != nil {
		return summary, stats, err
	}
	summary.SampleCount = sampleCount
	stats.SampleRows = sampleCount

	if baselineRated.Valid {
		summary.BaselineRatedRangeAt100PercentKM = &baselineRated.Float64
	}
	if baselineIdeal.Valid {
		summary.BaselineIdealRangeAt100PercentKM = &baselineIdeal.Float64
	}

	if sampleCount >= v2MinimumBatteryRangeSamples &&
		summary.EstimatedRatedRangeAt100PercentKM != nil &&
		baselineRated.Valid && baselineRated.Float64 > 0 {
		degradation := (baselineRated.Float64 - *summary.EstimatedRatedRangeAt100PercentKM) / baselineRated.Float64 * 100
		summary.EstimatedRangeDegradationPercent = &degradation
	}

	return summary, stats, nil
}

// Timeseries uses drives + start_position PK join to avoid full positions table scans.
// Groups by start_date of each drive, computing estimated full-range from start battery.
func (r PostgresV2BatteryRepository) Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2BatteryTimeseriesItem, V2BatteryStats, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			GREATEST(date_trunc('%s', drives.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4), $2 AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
			AVG(drives.start_rated_range_km / sp.battery_level * 100)
				FILTER (WHERE sp.battery_level > 0 AND drives.start_rated_range_km IS NOT NULL) AS estimated_rated_range_at_100,
			AVG(drives.start_rated_range_km / sp.battery_level * 100)
				FILTER (WHERE sp.battery_level > 0 AND drives.start_rated_range_km IS NOT NULL) AS estimated_ideal_range_at_100,
			AVG(sp.battery_level) FILTER (WHERE sp.battery_level IS NOT NULL AND sp.battery_level > 0) AS avg_battery_level,
			COUNT(*) FILTER (WHERE sp.battery_level > 0 AND drives.start_rated_range_km IS NOT NULL) AS sample_count,
			COUNT(*) FILTER (WHERE sp.battery_level IS NULL OR sp.battery_level <= 0) AS invalid_rows
		FROM drives
		LEFT JOIN positions sp ON sp.id = drives.start_position_id
		WHERE drives.car_id = $1
			AND drives.start_date >= $2
			AND drives.start_date < $3
			AND drives.end_date IS NOT NULL
		GROUP BY 1
		ORDER BY 1`, postgresDateTruncUnit(groupBy)),
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, timeRange.Timezone,
	)
	if err != nil {
		return nil, V2BatteryStats{}, err
	}
	defer rows.Close()

	var items []V2BatteryTimeseriesItem
	var stats V2BatteryStats
	location := timeRangeLocation(timeRange)
	for rows.Next() {
		var periodStart time.Time
		var item V2BatteryTimeseriesItem
		var rated sql.NullFloat64
		var ideal sql.NullFloat64
		var avgBattery sql.NullFloat64
		var invalidRows int64
		if err := rows.Scan(&periodStart, &rated, &ideal, &avgBattery, &item.SampleCount, &invalidRows); err != nil {
			return nil, stats, err
		}
		item.PeriodStart = periodStart.In(location).Format(time.RFC3339)
		if rated.Valid {
			item.EstimatedRatedRangeAt100PercentKM = &rated.Float64
		}
		if ideal.Valid {
			item.EstimatedIdealRangeAt100PercentKM = &ideal.Float64
		}
		if avgBattery.Valid {
			item.AvgBatteryLevelPercent = &avgBattery.Float64
		}
		stats.SampleRows += item.SampleCount
		stats.InvalidBatteryRows += invalidRows
		items = append(items, item)
	}
	return items, stats, rows.Err()
}

// Distribution uses drives + position PK joins to get start/end battery levels.
// Avoids large positions table scans by using FK-based point lookups.
func (r PostgresV2BatteryRepository) Distribution(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2BatteryDistributionItem, V2BatteryStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH battery_samples AS (
			SELECT sp.battery_level AS battery_level
			FROM drives
			LEFT JOIN positions sp ON sp.id = drives.start_position_id
			WHERE drives.car_id = $1 AND drives.start_date >= $2 AND drives.start_date < $3
				AND drives.end_date IS NOT NULL AND sp.battery_level IS NOT NULL
				AND sp.battery_level >= 0 AND sp.battery_level <= 100
			UNION ALL
			SELECT ep.battery_level AS battery_level
			FROM drives
			LEFT JOIN positions ep ON ep.id = drives.end_position_id
			WHERE drives.car_id = $1 AND drives.start_date >= $2 AND drives.start_date < $3
				AND drives.end_date IS NOT NULL AND ep.battery_level IS NOT NULL
				AND ep.battery_level >= 0 AND ep.battery_level <= 100
		),
		buckets AS (
			SELECT generate_series(0, 90, 10)::int AS bucket_min
		),
		bucketed AS (
			SELECT LEAST((battery_level / 10) * 10, 90)::int AS bucket_min, COUNT(*) AS sample_count
			FROM battery_samples GROUP BY 1
		),
		total AS (
			SELECT COUNT(*) AS sample_count FROM battery_samples
		)
		SELECT
			buckets.bucket_min,
			buckets.bucket_min + 10 AS bucket_max,
			COALESCE(bucketed.sample_count, 0) AS sample_count,
			CASE WHEN total.sample_count > 0
				THEN COALESCE(bucketed.sample_count, 0)::float / total.sample_count * 100
				ELSE 0 END AS percent
		FROM buckets
		LEFT JOIN bucketed ON bucketed.bucket_min = buckets.bucket_min
		CROSS JOIN total
		ORDER BY buckets.bucket_min`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	)
	if err != nil {
		return nil, V2BatteryStats{}, err
	}
	defer rows.Close()

	var items []V2BatteryDistributionItem
	var stats V2BatteryStats
	for rows.Next() {
		var item V2BatteryDistributionItem
		if err := rows.Scan(&item.MinBatteryLevelPercent, &item.MaxBatteryLevelPercent, &item.SampleCount, &item.Percent); err != nil {
			return nil, stats, err
		}
		item.Bucket = batteryLevelBucketLabel(item.MinBatteryLevelPercent, item.MaxBatteryLevelPercent)
		stats.SampleRows += item.SampleCount
		items = append(items, item)
	}
	return items, stats, rows.Err()
}

func batteryLevelBucketLabel(min int64, max int64) string {
	if max >= 100 {
		return fmt.Sprintf("%d-100", min)
	}
	return fmt.Sprintf("%d-%d", min, max)
}
