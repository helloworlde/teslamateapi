package v2

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// @name PostgresV2DrivingRepository
type PostgresV2DrivingRepository struct {
	db *sql.DB
}

func NewPostgresV2DrivingRepository(db *sql.DB) PostgresV2DrivingRepository {
	return PostgresV2DrivingRepository{db: db}
}

func (r PostgresV2DrivingRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2DrivingRepository) Summary(ctx context.Context, carID int64, start timeBound, end timeBound) (V2DrivingSummary, V2DrivingStats, error) {
	var summary V2DrivingSummary
	var stats V2DrivingStats
	var maxSpeed sql.NullFloat64
	var avgSpeed sql.NullFloat64
	var estimatedEnergy sql.NullFloat64
	var avgConsumption sql.NullFloat64
	var bestConsumption sql.NullFloat64
	var worstConsumption sql.NullFloat64
	var estimatedRegen sql.NullFloat64
	var batteryLevelUsed sql.NullFloat64

	err := r.db.QueryRowContext(ctx, `
		WITH drive_metrics AS (
			SELECT
				drives.distance,
				drives.duration_min * 60 AS duration_seconds,
				drives.speed_max,
				drives.start_rated_range_km,
				drives.end_rated_range_km,
				start_position.battery_level AS start_battery_level,
				end_position.battery_level AS end_battery_level,
				cars.efficiency AS car_efficiency,
				CASE WHEN drives.duration_min > 0 THEN drives.distance / drives.duration_min * 60 END AS avg_speed,
				CASE
					WHEN drives.start_rated_range_km IS NOT NULL
						AND drives.end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency
				END AS estimated_energy_consumed,
				CASE
					WHEN drives.distance > 0
						AND drives.start_rated_range_km IS NOT NULL
						AND drives.end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / drives.distance * 1000
				END AS consumption,
				CASE
					WHEN drives.start_rated_range_km IS NOT NULL
						AND drives.end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (drives.start_rated_range_km - drives.end_rated_range_km) < 0
					THEN (drives.end_rated_range_km - drives.start_rated_range_km) * cars.efficiency
				END AS estimated_energy_regen
			FROM drives
			LEFT JOIN cars ON cars.id = drives.car_id
			LEFT JOIN positions start_position ON start_position.id = drives.start_position_id
			LEFT JOIN positions end_position ON end_position.id = drives.end_position_id
			WHERE drives.car_id = $1
				AND drives.end_date IS NOT NULL
				AND drives.start_date >= $2
				AND drives.start_date < $3
		)
		SELECT
			COUNT(*) AS drive_count,
			COALESCE(SUM(distance), 0) AS distance,
			COALESCE(SUM(duration_seconds), 0) AS duration,
			MAX(speed_max) AS max_speed,
			-- Distance-weighted avg: a 1 km errand and a 200 km highway leg
			-- shouldn't carry equal weight (audit §2.1, spec §1.1).
			CASE WHEN COALESCE(SUM(duration_seconds), 0) > 0
			     THEN SUM(distance) / SUM(duration_seconds) * 3600
			END AS avg_speed,
			SUM(estimated_energy_consumed) AS estimated_energy_consumed,
			AVG(consumption) AS avg_consumption,
			MIN(consumption) AS best_consumption,
			MAX(consumption) AS worst_consumption,
			SUM(estimated_energy_regen) AS estimated_energy_regen,
			COALESCE(SUM(GREATEST(COALESCE(start_rated_range_km, 0) - COALESCE(end_rated_range_km, 0), 0)), 0) AS range_loss,
			NULLIF(SUM(GREATEST(COALESCE(start_battery_level, 0) - COALESCE(end_battery_level, 0), 0)), 0) AS battery_level_used,
			COUNT(estimated_energy_consumed) AS energy_estimate_rows
		FROM drive_metrics`,
		carID, start.Time, end.Time,
	).Scan(
		&summary.DriveCount,
		&summary.Distance,
		&summary.Duration,
		&maxSpeed,
		&avgSpeed,
		&estimatedEnergy,
		&avgConsumption,
		&bestConsumption,
		&worstConsumption,
		&estimatedRegen,
		&summary.RangeLoss,
		&batteryLevelUsed,
		&stats.EnergyEstimateRows,
	)
	if err != nil {
		return summary, stats, err
	}

	stats.DriveRows = summary.DriveCount
	if summary.DriveCount > 0 {
		summary.AvgDistance = float64Ptr(summary.Distance / float64(summary.DriveCount))
		summary.AvgDuration = float64Ptr(summary.Duration / float64(summary.DriveCount))
	}
	if maxSpeed.Valid {
		summary.MaxSpeed = &maxSpeed.Float64
	}
	if avgSpeed.Valid {
		summary.AvgSpeed = &avgSpeed.Float64
	}
	if estimatedEnergy.Valid {
		summary.EstimatedEnergyConsumed = &estimatedEnergy.Float64
	}
	if avgConsumption.Valid {
		summary.AvgConsumption = &avgConsumption.Float64
	}
	if bestConsumption.Valid {
		summary.BestConsumption = &bestConsumption.Float64
	}
	if worstConsumption.Valid {
		summary.WorstConsumption = &worstConsumption.Float64
	}
	if estimatedRegen.Valid {
		summary.EstimatedEnergyRegen = &estimatedRegen.Float64
	}
	if batteryLevelUsed.Valid {
		summary.BatteryLevelUsed = &batteryLevelUsed.Float64
	}
	return summary, stats, nil
}

func (r PostgresV2DrivingRepository) Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2DrivingTimeseriesItem, V2DrivingStats, error) {
	truncUnit := postgresDateTruncUnit(groupBy)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			GREATEST(date_trunc('%s', drives.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4), $2 AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
			COUNT(*) AS drive_count,
			COALESCE(SUM(drives.distance), 0) AS distance,
			COALESCE(SUM(drives.duration_min) * 60, 0) AS duration,
			-- Distance-weighted avg, same reasoning as Summary.
			CASE WHEN COALESCE(SUM(drives.duration_min), 0) > 0
			     THEN SUM(drives.distance) / SUM(drives.duration_min) * 60
			END AS avg_speed,
			SUM(
				CASE
					WHEN drives.start_rated_range_km IS NOT NULL
						AND drives.end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency
					ELSE NULL
				END
			) AS estimated_energy_consumed,
			AVG(
				CASE
					WHEN drives.distance > 0
						AND drives.start_rated_range_km IS NOT NULL
						AND drives.end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / drives.distance * 1000
					ELSE NULL
				END
			) AS avg_consumption,
			COUNT(
				CASE
					WHEN start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
					THEN 1
					ELSE NULL
				END
			) AS energy_estimate_rows
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND drives.start_date >= $2
			AND drives.start_date < $3
		GROUP BY 1
		ORDER BY 1`, truncUnit),
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, timeRange.Timezone,
	)
	if err != nil {
		return nil, V2DrivingStats{}, err
	}
	defer rows.Close()

	var items []V2DrivingTimeseriesItem
	var stats V2DrivingStats
	location := timeRangeLocation(timeRange)
	for rows.Next() {
		var periodStart time.Time
		var item V2DrivingTimeseriesItem
		var avgSpeed sql.NullFloat64
		var estimatedEnergy sql.NullFloat64
		var avgConsumption sql.NullFloat64
		var energyRows int64
		if err := rows.Scan(&periodStart, &item.DriveCount, &item.Distance, &item.Duration, &avgSpeed, &estimatedEnergy, &avgConsumption, &energyRows); err != nil {
			return nil, stats, err
		}
		item.PeriodStart = periodStart.In(location).Format(time.RFC3339)
		if avgSpeed.Valid {
			item.AvgSpeed = &avgSpeed.Float64
		}
		if estimatedEnergy.Valid {
			item.EstimatedEnergyConsumed = &estimatedEnergy.Float64
		}
		if avgConsumption.Valid {
			item.AvgConsumption = &avgConsumption.Float64
		}
		stats.DriveRows += item.DriveCount
		stats.EnergyEstimateRows += energyRows
		items = append(items, item)
	}
	return items, stats, rows.Err()
}

func postgresDateTruncUnit(groupBy string) string {
	switch groupBy {
	case "week":
		return "week"
	case "month":
		return "month"
	case "year":
		return "year"
	default:
		return "day"
	}
}
