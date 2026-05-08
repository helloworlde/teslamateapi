package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

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

func (r PostgresV2BatteryRepository) Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2BatteryAnalyticsSummary, V2BatteryStats, error) {
	var summary V2BatteryAnalyticsSummary
	var stats V2BatteryStats
	var latestBattery sql.NullInt64
	var latestRatedRange sql.NullFloat64
	var latestIdealRange sql.NullFloat64
	var latestEstimatedRated sql.NullFloat64
	var latestEstimatedIdeal sql.NullFloat64
	var baselineRated sql.NullFloat64
	var baselineIdeal sql.NullFloat64
	var invalidBatteryRows int64

	err := r.db.QueryRowContext(ctx, `
		WITH period_positions AS (
			SELECT
				battery_level,
				rated_battery_range_km,
				ideal_battery_range_km,
				CASE
					WHEN battery_level > 0 AND rated_battery_range_km IS NOT NULL
					THEN rated_battery_range_km / battery_level * 100
				END AS estimated_rated_range_at_100,
				CASE
					WHEN battery_level > 0 AND ideal_battery_range_km IS NOT NULL
					THEN ideal_battery_range_km / battery_level * 100
				END AS estimated_ideal_range_at_100,
				date
			FROM positions
			WHERE car_id = $1
				AND date >= $2
				AND date < $3
		),
		latest AS (
			SELECT *
			FROM period_positions
			ORDER BY date DESC
			LIMIT 1
		),
		baseline AS (
			SELECT
				MAX(rated_battery_range_km / battery_level * 100) FILTER (WHERE battery_level > 0 AND rated_battery_range_km IS NOT NULL) AS baseline_rated,
				MAX(ideal_battery_range_km / battery_level * 100) FILTER (WHERE battery_level > 0 AND ideal_battery_range_km IS NOT NULL) AS baseline_ideal
			FROM positions
			WHERE car_id = $1
		)
		SELECT
			(SELECT battery_level FROM latest) AS latest_battery_level,
			(SELECT rated_battery_range_km FROM latest) AS latest_rated_range,
			(SELECT ideal_battery_range_km FROM latest) AS latest_ideal_range,
			(SELECT estimated_rated_range_at_100 FROM latest) AS latest_estimated_rated,
			(SELECT estimated_ideal_range_at_100 FROM latest) AS latest_estimated_ideal,
			baseline.baseline_rated,
			baseline.baseline_ideal,
			COUNT(*) FILTER (WHERE period_positions.battery_level > 0 AND (period_positions.rated_battery_range_km IS NOT NULL OR period_positions.ideal_battery_range_km IS NOT NULL)) AS sample_count,
			COUNT(*) FILTER (WHERE period_positions.battery_level IS NULL OR period_positions.battery_level <= 0) AS invalid_battery_rows
		FROM baseline
		LEFT JOIN period_positions ON true
		GROUP BY baseline.baseline_rated, baseline.baseline_ideal`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	).Scan(
		&latestBattery,
		&latestRatedRange,
		&latestIdealRange,
		&latestEstimatedRated,
		&latestEstimatedIdeal,
		&baselineRated,
		&baselineIdeal,
		&summary.SampleCount,
		&invalidBatteryRows,
	)
	if err != nil {
		return summary, stats, err
	}

	stats.SampleRows = summary.SampleCount
	stats.InvalidBatteryRows = invalidBatteryRows
	if latestBattery.Valid {
		summary.LatestBatteryLevelPercent = &latestBattery.Int64
	}
	if latestRatedRange.Valid {
		summary.LatestRatedRangeKM = &latestRatedRange.Float64
	}
	if latestIdealRange.Valid {
		summary.LatestIdealRangeKM = &latestIdealRange.Float64
	}
	if latestEstimatedRated.Valid {
		summary.EstimatedRatedRangeAt100PercentKM = &latestEstimatedRated.Float64
	}
	if latestEstimatedIdeal.Valid {
		summary.EstimatedIdealRangeAt100PercentKM = &latestEstimatedIdeal.Float64
	}
	if baselineRated.Valid {
		summary.BaselineRatedRangeAt100PercentKM = &baselineRated.Float64
	}
	if baselineIdeal.Valid {
		summary.BaselineIdealRangeAt100PercentKM = &baselineIdeal.Float64
	}
	if summary.SampleCount >= v2MinimumBatteryRangeSamples && latestEstimatedRated.Valid && baselineRated.Valid && baselineRated.Float64 > 0 {
		summary.EstimatedRangeDegradationPercent = float64Ptr((baselineRated.Float64 - latestEstimatedRated.Float64) / baselineRated.Float64 * 100)
	}
	return summary, stats, nil
}

func (r PostgresV2BatteryRepository) Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2BatteryTimeseriesItem, V2BatteryStats, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			date_trunc('%s', positions.date AT TIME ZONE 'UTC' AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
			AVG(rated_battery_range_km / battery_level * 100) FILTER (WHERE battery_level > 0 AND rated_battery_range_km IS NOT NULL) AS estimated_rated_range_at_100,
			AVG(ideal_battery_range_km / battery_level * 100) FILTER (WHERE battery_level > 0 AND ideal_battery_range_km IS NOT NULL) AS estimated_ideal_range_at_100,
			AVG(battery_level) FILTER (WHERE battery_level IS NOT NULL) AS avg_battery_level,
			COUNT(*) FILTER (WHERE battery_level > 0 AND (rated_battery_range_km IS NOT NULL OR ideal_battery_range_km IS NOT NULL)) AS sample_count,
			COUNT(*) FILTER (WHERE battery_level IS NULL OR battery_level <= 0) AS invalid_battery_rows
		FROM positions
		WHERE positions.car_id = $1
			AND positions.date >= $2
			AND positions.date < $3
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

func (r PostgresV2BatteryRepository) Distribution(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2BatteryDistributionItem, V2BatteryStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH valid_positions AS (
			SELECT battery_level
			FROM positions
			WHERE car_id = $1
				AND date >= $2
				AND date < $3
				AND battery_level IS NOT NULL
				AND battery_level >= 0
				AND battery_level <= 100
		),
		buckets AS (
			SELECT generate_series(0, 90, 10)::int AS bucket_min
		),
		bucketed AS (
			SELECT
				LEAST((battery_level / 10) * 10, 90)::int AS bucket_min,
				COUNT(*) AS sample_count
			FROM valid_positions
			GROUP BY 1
		),
		total AS (
			SELECT COUNT(*) AS sample_count FROM valid_positions
		)
		SELECT
			buckets.bucket_min,
			buckets.bucket_min + 10 AS bucket_max,
			COALESCE(bucketed.sample_count, 0) AS sample_count,
			CASE WHEN total.sample_count > 0 THEN COALESCE(bucketed.sample_count, 0)::float / total.sample_count * 100 ELSE 0 END AS percent
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
