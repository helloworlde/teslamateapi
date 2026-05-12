package main

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

func (r PostgresV2DrivingRepository) Summary(ctx context.Context, carID int64, start timeBound, end timeBound) (V2DrivingAnalyticsSummary, V2DrivingStats, error) {
	var summary V2DrivingAnalyticsSummary
	var stats V2DrivingStats
	var avgSpeed sql.NullFloat64
	var estimatedEnergy sql.NullFloat64
	var avgConsumption sql.NullFloat64
	var bestConsumption sql.NullFloat64
	var worstConsumption sql.NullFloat64
	var estimatedRegen sql.NullFloat64
	var avgOutsideTemp sql.NullFloat64

	err := r.db.QueryRowContext(ctx, `
		WITH drive_metrics AS (
			SELECT
				drives.distance,
				drives.duration_min,
				drives.speed_max,
				drives.outside_temp_avg,
				drives.start_rated_range_km,
				drives.end_rated_range_km,
				start_position.battery_level AS start_battery_level,
				end_position.battery_level AS end_battery_level,
				cars.efficiency AS car_efficiency,
				CASE WHEN drives.duration_min > 0 THEN drives.distance / drives.duration_min * 60 END AS avg_speed_kmh,
				CASE
					WHEN drives.start_rated_range_km IS NOT NULL
						AND drives.end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency
				END AS estimated_energy_consumed_kwh,
				CASE
					WHEN drives.distance > 0
						AND drives.start_rated_range_km IS NOT NULL
						AND drives.end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / drives.distance * 1000
				END AS consumption_wh_per_km,
				CASE
					WHEN drives.start_rated_range_km IS NOT NULL
						AND drives.end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (drives.start_rated_range_km - drives.end_rated_range_km) < 0
					THEN (drives.end_rated_range_km - drives.start_rated_range_km) * cars.efficiency
				END AS estimated_regenerated_energy_kwh
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
			COALESCE(SUM(distance), 0) AS distance_km,
			COALESCE(SUM(duration_min), 0) AS duration_min,
			COALESCE(MAX(speed_max), 0) AS max_speed_kmh,
			AVG(avg_speed_kmh) AS avg_speed_kmh,
			SUM(estimated_energy_consumed_kwh) AS estimated_energy_consumed_kwh,
			AVG(consumption_wh_per_km) AS avg_consumption_wh_per_km,
			MIN(consumption_wh_per_km) AS best_consumption_wh_per_km,
			MAX(consumption_wh_per_km) AS worst_consumption_wh_per_km,
			SUM(estimated_regenerated_energy_kwh) AS estimated_regenerated_energy_kwh,
			COALESCE(SUM(GREATEST(COALESCE(start_rated_range_km, 0) - COALESCE(end_rated_range_km, 0), 0)), 0) AS range_loss_km,
			COALESCE(SUM(GREATEST(COALESCE(start_battery_level, 0) - COALESCE(end_battery_level, 0), 0)), 0) AS battery_level_used_percent,
			AVG(outside_temp_avg) AS avg_outside_temp_c,
			COUNT(estimated_energy_consumed_kwh) AS energy_estimate_rows,
			COUNT(outside_temp_avg) AS temperature_rows
		FROM drive_metrics`,
		carID, start.Time, end.Time,
	).Scan(
		&summary.DriveCount,
		&summary.DistanceKM,
		&summary.DurationMin,
		&summary.MaxSpeedKMH,
		&avgSpeed,
		&estimatedEnergy,
		&avgConsumption,
		&bestConsumption,
		&worstConsumption,
		&estimatedRegen,
		&summary.RangeLossKM,
		&summary.BatteryLevelUsedPercent,
		&avgOutsideTemp,
		&stats.EnergyEstimateRows,
		&stats.TemperatureRows,
	)
	if err != nil {
		return summary, stats, err
	}

	stats.DriveRows = summary.DriveCount
	if summary.DriveCount > 0 {
		summary.AvgDistanceKM = float64Ptr(summary.DistanceKM / float64(summary.DriveCount))
		summary.AvgDurationMin = float64Ptr(summary.DurationMin / float64(summary.DriveCount))
	}
	if avgSpeed.Valid {
		summary.AvgSpeedKMH = &avgSpeed.Float64
	}
	if estimatedEnergy.Valid {
		summary.EstimatedEnergyConsumedKWh = &estimatedEnergy.Float64
	}
	if avgConsumption.Valid {
		summary.AvgConsumptionWhPerKM = &avgConsumption.Float64
	}
	if bestConsumption.Valid {
		summary.BestConsumptionWhPerKM = &bestConsumption.Float64
	}
	if worstConsumption.Valid {
		summary.WorstConsumptionWhPerKM = &worstConsumption.Float64
	}
	if estimatedRegen.Valid {
		summary.EstimatedRegeneratedEnergyKWh = &estimatedRegen.Float64
	}
	if avgOutsideTemp.Valid {
		summary.AvgOutsideTempC = &avgOutsideTemp.Float64
	}
	return summary, stats, nil
}

func (r PostgresV2DrivingRepository) Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2DrivingTimeseriesItem, V2DrivingStats, error) {
	truncUnit := postgresDateTruncUnit(groupBy)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			GREATEST(date_trunc('%s', drives.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4), $2 AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
			COUNT(*) AS drive_count,
			COALESCE(SUM(drives.distance), 0) AS distance_km,
			COALESCE(SUM(drives.duration_min), 0) AS duration_min,
			AVG(CASE WHEN drives.duration_min > 0 THEN drives.distance / drives.duration_min * 60 ELSE NULL END) AS avg_speed_kmh,
			SUM(
				CASE
					WHEN drives.start_rated_range_km IS NOT NULL
						AND drives.end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency
					ELSE NULL
				END
			) AS estimated_energy_consumed_kwh,
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
			) AS avg_consumption_wh_per_km,
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
		if err := rows.Scan(&periodStart, &item.DriveCount, &item.DistanceKM, &item.DurationMin, &avgSpeed, &estimatedEnergy, &avgConsumption, &energyRows); err != nil {
			return nil, stats, err
		}
		item.PeriodStart = periodStart.In(location).Format(time.RFC3339)
		if avgSpeed.Valid {
			item.AvgSpeedKMH = &avgSpeed.Float64
		}
		if estimatedEnergy.Valid {
			item.EstimatedEnergyConsumedKWh = &estimatedEnergy.Float64
		}
		if avgConsumption.Valid {
			item.AvgConsumptionWhPerKM = &avgConsumption.Float64
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

