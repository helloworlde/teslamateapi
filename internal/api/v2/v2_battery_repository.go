package v2

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
func (r PostgresV2BatteryRepository) Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2BatterySummary, V2BatteryStats, error) {
	var summary V2BatterySummary
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
		summary.LatestLevel = &latestBattery.Int64
	}
	if latestRatedRange.Valid {
		summary.LatestRatedRange = &latestRatedRange.Float64
	}
	if latestIdealRange.Valid {
		summary.LatestIdealRange = &latestIdealRange.Float64
	}

	// Estimated full range from latest observation
	var estimatedRated, estimatedIdeal *float64
	if latestBattery.Valid && latestBattery.Int64 > 0 && latestRatedRange.Valid {
		v := latestRatedRange.Float64 / float64(latestBattery.Int64) * 100
		estimatedRated = &v
	}
	if latestBattery.Valid && latestBattery.Int64 > 0 && latestIdealRange.Valid {
		v := latestIdealRange.Float64 / float64(latestBattery.Int64) * 100
		estimatedIdeal = &v
	}
	if estimatedRated != nil || estimatedIdeal != nil {
		summary.RangeAtFullCharge = &V2BatteryRange{Rated: estimatedRated, Ideal: estimatedIdeal}
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
	stats.SampleRows = sampleCount

	if baselineRated.Valid || baselineIdeal.Valid {
		base := &V2BatteryRange{}
		if baselineRated.Valid {
			base.Rated = &baselineRated.Float64
		}
		if baselineIdeal.Valid {
			base.Ideal = &baselineIdeal.Float64
		}
		summary.BaselineRangeAtFullCharge = base
	}

	if sampleCount >= v2MinimumBatteryRangeSamples &&
		estimatedRated != nil &&
		baselineRated.Valid && baselineRated.Float64 > 0 {
		degradation := (baselineRated.Float64 - *estimatedRated) / baselineRated.Float64 * 100
		summary.EstimatedRangeDegradation = &degradation
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
		var sampleCount int64
		var invalidRows int64
		if err := rows.Scan(&periodStart, &rated, &ideal, &avgBattery, &sampleCount, &invalidRows); err != nil {
			return nil, stats, err
		}
		item.PeriodStart = periodStart.In(location).Format(time.RFC3339)
		if rated.Valid || ideal.Valid {
			rng := &V2BatteryRange{}
			if rated.Valid {
				rng.Rated = &rated.Float64
			}
			if ideal.Valid {
				rng.Ideal = &ideal.Float64
			}
			item.RangeAtFullCharge = rng
		}
		if avgBattery.Valid {
			item.AvgLevel = &avgBattery.Float64
		}
		stats.SampleRows += sampleCount
		stats.InvalidBatteryRows += invalidRows
		items = append(items, item)
	}
	return items, stats, rows.Err()
}
