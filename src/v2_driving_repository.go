package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

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
	var avgOutsideTemp sql.NullFloat64

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS drive_count,
			COALESCE(SUM(distance), 0) AS distance_km,
			COALESCE(SUM(duration_min), 0) AS duration_min,
			COALESCE(MAX(speed_max), 0) AS max_speed_kmh,
			AVG(CASE WHEN duration_min > 0 THEN distance / duration_min * 60 ELSE NULL END) AS avg_speed_kmh,
			SUM(
				CASE
					WHEN start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency
					ELSE NULL
				END
			) AS estimated_energy_consumed_kwh,
			AVG(
				CASE
					WHEN distance > 0
						AND start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / distance * 1000
					ELSE NULL
				END
			) AS avg_consumption_wh_per_km,
			COALESCE(SUM(GREATEST(COALESCE(start_rated_range_km, 0) - COALESCE(end_rated_range_km, 0), 0)), 0) AS range_loss_km,
			COALESCE(SUM(GREATEST(COALESCE(start_position.battery_level, 0) - COALESCE(end_position.battery_level, 0), 0)), 0) AS battery_level_used_percent,
			AVG(outside_temp_avg) AS avg_outside_temp_c,
			COUNT(
				CASE
					WHEN start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
					THEN 1
					ELSE NULL
				END
			) AS energy_estimate_rows,
			COUNT(outside_temp_avg) AS temperature_rows
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		LEFT JOIN positions start_position ON start_position.id = drives.start_position_id
		LEFT JOIN positions end_position ON end_position.id = drives.end_position_id
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND drives.start_date >= $2
			AND drives.start_date < $3`,
		carID, start.Time, end.Time,
	).Scan(
		&summary.DriveCount,
		&summary.DistanceKM,
		&summary.DurationMin,
		&summary.MaxSpeedKMH,
		&avgSpeed,
		&estimatedEnergy,
		&avgConsumption,
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
	if avgOutsideTemp.Valid {
		summary.AvgOutsideTempC = &avgOutsideTemp.Float64
	}
	return summary, stats, nil
}

func (r PostgresV2DrivingRepository) Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2DrivingTimeseriesItem, V2DrivingStats, error) {
	truncUnit := postgresDateTruncUnit(groupBy)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			date_trunc('%s', drives.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
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

func (r PostgresV2DrivingRepository) Distribution(ctx context.Context, carID int64, timeRange V2TimeRange, dimension string) ([]V2DrivingDistributionItem, V2DrivingStats, error) {
	bucketSQL, err := drivingDistributionBucketSQL(dimension)
	if err != nil {
		return nil, V2DrivingStats{}, err
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			%s AS bucket,
			COUNT(*) AS drive_count,
			COALESCE(SUM(drives.distance), 0) AS distance_km,
			COALESCE(SUM(drives.duration_min), 0) AS duration_min,
			COUNT(drives.outside_temp_avg) AS temperature_rows
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND drives.start_date >= $2
			AND drives.start_date < $3
		GROUP BY 1
		ORDER BY 1`, bucketSQL),
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, timeRange.Timezone,
	)
	if err != nil {
		return nil, V2DrivingStats{}, err
	}
	defer rows.Close()

	var items []V2DrivingDistributionItem
	var stats V2DrivingStats
	for rows.Next() {
		var item V2DrivingDistributionItem
		var temperatureRows int64
		if err := rows.Scan(&item.Bucket, &item.DriveCount, &item.DistanceKM, &item.DurationMin, &temperatureRows); err != nil {
			return nil, stats, err
		}
		stats.DriveRows += item.DriveCount
		stats.TemperatureRows += temperatureRows
		items = append(items, item)
	}
	return items, stats, rows.Err()
}

func (r PostgresV2DrivingRepository) Ranking(ctx context.Context, carID int64, timeRange V2TimeRange, rankingType string, limit int) ([]V2DrivingRankingItem, V2DrivingStats, error) {
	if rankingType == "highest_distance_day" {
		return r.rankingHighestDistanceDay(ctx, carID, timeRange, limit)
	}
	orderSQL, metricUnit, whereSQL, err := drivingRankingSQL(rankingType)
	if err != nil {
		return nil, V2DrivingStats{}, err
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			drives.id,
			drives.start_date,
			drives.end_date,
			drives.distance,
			drives.duration_min,
			drives.speed_max,
			%s AS metric_value
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND drives.start_date >= $2
			AND drives.start_date < $3
			%s
		ORDER BY metric_value %s
		LIMIT $4`, drivingRankingMetricSQL(rankingType), whereSQL, orderSQL),
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, limit,
	)
	if err != nil {
		return nil, V2DrivingStats{}, err
	}
	defer rows.Close()

	var items []V2DrivingRankingItem
	var stats V2DrivingStats
	location := timeRangeLocation(timeRange)
	rank := 1
	for rows.Next() {
		var id int64
		var startTime time.Time
		var endTime time.Time
		var distance float64
		var duration float64
		var speed float64
		var metric float64
		if err := rows.Scan(&id, &startTime, &endTime, &distance, &duration, &speed, &metric); err != nil {
			return nil, stats, err
		}
		start := startTime.In(location).Format(time.RFC3339)
		end := endTime.In(location).Format(time.RFC3339)
		item := V2DrivingRankingItem{
			Rank:        rank,
			DriveID:     &id,
			StartTime:   &start,
			EndTime:     &end,
			MetricValue: metric,
			MetricUnit:  metricUnit,
			DistanceKM:  &distance,
			DurationMin: &duration,
			MaxSpeedKMH: &speed,
		}
		items = append(items, item)
		stats.DriveRows++
		rank++
	}
	return items, stats, rows.Err()
}

func (r PostgresV2DrivingRepository) rankingHighestDistanceDay(ctx context.Context, carID int64, timeRange V2TimeRange, limit int) ([]V2DrivingRankingItem, V2DrivingStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			date_trunc('day', start_date AT TIME ZONE 'UTC' AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
			COALESCE(SUM(distance), 0) AS distance_km,
			COALESCE(SUM(duration_min), 0) AS duration_min
		FROM drives
		WHERE car_id = $1
			AND end_date IS NOT NULL
			AND start_date >= $2
			AND start_date < $3
		GROUP BY 1
		ORDER BY distance_km DESC
		LIMIT $5`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, timeRange.Timezone, limit,
	)
	if err != nil {
		return nil, V2DrivingStats{}, err
	}
	defer rows.Close()

	var items []V2DrivingRankingItem
	var stats V2DrivingStats
	location := timeRangeLocation(timeRange)
	rank := 1
	for rows.Next() {
		var periodStart time.Time
		var distance float64
		var duration float64
		if err := rows.Scan(&periodStart, &distance, &duration); err != nil {
			return nil, stats, err
		}
		period := periodStart.In(location).Format(time.RFC3339)
		item := V2DrivingRankingItem{
			Rank:        rank,
			PeriodStart: &period,
			MetricValue: distance,
			MetricUnit:  "km",
			DistanceKM:  &distance,
			DurationMin: &duration,
		}
		items = append(items, item)
		stats.DriveRows++
		rank++
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

func drivingDistributionBucketSQL(dimension string) (string, error) {
	switch dimension {
	case "hour_of_day":
		return `LPAD(EXTRACT(HOUR FROM drives.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4)::int::text, 2, '0')`, nil
	case "day_of_week":
		return `TRIM(TO_CHAR(drives.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4, 'Day'))`, nil
	case "distance_bucket":
		return `CASE
			WHEN drives.distance < 5 THEN '000-005'
			WHEN drives.distance < 10 THEN '005-010'
			WHEN drives.distance < 25 THEN '010-025'
			WHEN drives.distance < 50 THEN '025-050'
			WHEN drives.distance < 100 THEN '050-100'
			ELSE '100+'
		END`, nil
	case "duration_bucket":
		return `CASE
			WHEN drives.duration_min < 10 THEN '000-010'
			WHEN drives.duration_min < 30 THEN '010-030'
			WHEN drives.duration_min < 60 THEN '030-060'
			WHEN drives.duration_min < 120 THEN '060-120'
			ELSE '120+'
		END`, nil
	case "speed_bucket":
		return `CASE
			WHEN drives.speed_max < 30 THEN '000-030'
			WHEN drives.speed_max < 60 THEN '030-060'
			WHEN drives.speed_max < 90 THEN '060-090'
			WHEN drives.speed_max < 120 THEN '090-120'
			ELSE '120+'
		END`, nil
	case "consumption_bucket":
		return `CASE
			WHEN drives.distance <= 0 OR cars.efficiency IS NULL OR drives.start_rated_range_km IS NULL OR drives.end_rated_range_km IS NULL THEN 'unknown'
			WHEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / drives.distance * 1000 < 120 THEN '000-120'
			WHEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / drives.distance * 1000 < 160 THEN '120-160'
			WHEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / drives.distance * 1000 < 200 THEN '160-200'
			WHEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / drives.distance * 1000 < 250 THEN '200-250'
			ELSE '250+'
		END`, nil
	case "temperature_bucket":
		return `CASE
			WHEN drives.outside_temp_avg IS NULL THEN 'unknown'
			WHEN drives.outside_temp_avg < -10 THEN '<-10'
			WHEN drives.outside_temp_avg < 0 THEN '-10-000'
			WHEN drives.outside_temp_avg < 10 THEN '000-010'
			WHEN drives.outside_temp_avg < 20 THEN '010-020'
			WHEN drives.outside_temp_avg < 30 THEN '020-030'
			WHEN drives.outside_temp_avg < 40 THEN '030-040'
			ELSE '040+'
		END`, nil
	default:
		return "", errV2InvalidDrivingDimension
	}
}

func drivingRankingMetricSQL(rankingType string) string {
	switch rankingType {
	case "longest_duration":
		return "drives.duration_min"
	case "highest_speed":
		return "drives.speed_max"
	case "lowest_consumption", "highest_consumption":
		return "(drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / NULLIF(drives.distance, 0) * 1000"
	default:
		return "drives.distance"
	}
}

func drivingRankingSQL(rankingType string) (string, string, string, error) {
	switch rankingType {
	case "longest_distance":
		return "DESC", "km", "", nil
	case "longest_duration":
		return "DESC", "min", "", nil
	case "highest_speed":
		return "DESC", "km/h", "", nil
	case "lowest_consumption":
		return "ASC", "Wh/km", "AND drives.distance > 0 AND drives.start_rated_range_km IS NOT NULL AND drives.end_rated_range_km IS NOT NULL", nil
	case "highest_consumption":
		return "DESC", "Wh/km", "AND drives.distance > 0 AND drives.start_rated_range_km IS NOT NULL AND drives.end_rated_range_km IS NOT NULL", nil
	default:
		return "", "", "", errV2InvalidDrivingRanking
	}
}
