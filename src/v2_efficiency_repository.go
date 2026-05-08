package main

import (
	"context"
	"database/sql"
	"fmt"
)

type PostgresV2EfficiencyRepository struct {
	db *sql.DB
}

func NewPostgresV2EfficiencyRepository(db *sql.DB) PostgresV2EfficiencyRepository {
	return PostgresV2EfficiencyRepository{db: db}
}

func (r PostgresV2EfficiencyRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2EfficiencyRepository) Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2EfficiencySummary, V2EfficiencyStats, error) {
	var summary V2EfficiencySummary
	var stats V2EfficiencyStats
	var estimatedEnergy sql.NullFloat64
	var avgConsumption sql.NullFloat64
	var bestConsumption sql.NullFloat64
	var worstConsumption sql.NullFloat64
	var avgTemperature sql.NullFloat64
	var avgSpeed sql.NullFloat64
	var estimatedRegen sql.NullFloat64

	err := r.db.QueryRowContext(ctx, `
		WITH drive_metrics AS (
			SELECT
				drives.id,
				drives.distance,
				drives.duration_min,
				drives.outside_temp_avg,
				CASE WHEN drives.duration_min > 0 THEN drives.distance / drives.duration_min * 60 END AS avg_speed_kmh,
				CASE
					WHEN drives.distance > 0
						AND drives.start_rated_range_km IS NOT NULL
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
			WHERE drives.car_id = $1
				AND drives.end_date IS NOT NULL
				AND drives.start_date >= $2
				AND drives.start_date < $3
		)
		SELECT
			COUNT(*) AS drive_count,
			COALESCE(SUM(distance), 0) AS distance_km,
			SUM(estimated_energy_consumed_kwh) AS estimated_energy_consumed_kwh,
			AVG(consumption_wh_per_km) AS avg_consumption_wh_per_km,
			MIN(consumption_wh_per_km) AS best_consumption_wh_per_km,
			MAX(consumption_wh_per_km) AS worst_consumption_wh_per_km,
			AVG(outside_temp_avg) AS avg_temperature_c,
			AVG(avg_speed_kmh) AS avg_speed_kmh,
			SUM(estimated_regenerated_energy_kwh) AS estimated_regenerated_energy_kwh,
			COUNT(estimated_energy_consumed_kwh) AS energy_estimate_rows,
			COUNT(outside_temp_avg) AS temperature_rows
		FROM drive_metrics`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	).Scan(
		&summary.DriveCount,
		&summary.DistanceKM,
		&estimatedEnergy,
		&avgConsumption,
		&bestConsumption,
		&worstConsumption,
		&avgTemperature,
		&avgSpeed,
		&estimatedRegen,
		&stats.EnergyEstimateRows,
		&stats.TemperatureRows,
	)
	if err != nil {
		return summary, stats, err
	}
	stats.DriveRows = summary.DriveCount
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
	if avgTemperature.Valid {
		summary.AvgTemperatureC = &avgTemperature.Float64
	}
	if avgSpeed.Valid {
		summary.AvgSpeedKMH = &avgSpeed.Float64
	}
	if estimatedRegen.Valid {
		summary.EstimatedRegeneratedEnergyKWh = &estimatedRegen.Float64
	}
	return summary, stats, nil
}

func (r PostgresV2EfficiencyRepository) Factors(ctx context.Context, carID int64, timeRange V2TimeRange, dimension string) ([]V2EfficiencyFactorItem, V2EfficiencyStats, error) {
	bucketSQL, joins, err := efficiencyFactorBucketSQL(dimension)
	if err != nil {
		return nil, V2EfficiencyStats{}, err
	}
	needsTZ := dimension == "hour_of_day" || dimension == "day_of_week"
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		WITH drive_metrics AS (
			SELECT
				drives.id,
				drives.distance,
				drives.duration_min,
				drives.outside_temp_avg,
				CASE WHEN drives.duration_min > 0 THEN drives.distance / drives.duration_min * 60 END AS avg_speed_kmh,
				CASE
					WHEN drives.distance > 0
						AND drives.start_rated_range_km IS NOT NULL
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
				start_position.elevation AS start_elevation,
				end_position.elevation AS end_elevation,
				%s AS bucket
			FROM drives
			LEFT JOIN cars ON cars.id = drives.car_id
			LEFT JOIN positions start_position ON start_position.id = drives.start_position_id
			LEFT JOIN positions end_position ON end_position.id = drives.end_position_id
			%s
			WHERE drives.car_id = $1
				AND drives.end_date IS NOT NULL
				AND drives.start_date >= $2
				AND drives.start_date < $3
		)
		SELECT
			bucket,
			COUNT(*) AS drive_count,
			COALESCE(SUM(distance), 0) AS distance_km,
			SUM(estimated_energy_consumed_kwh) AS estimated_energy_consumed_kwh,
			AVG(consumption_wh_per_km) AS avg_consumption_wh_per_km,
			AVG(outside_temp_avg) AS avg_temperature_c,
			AVG(avg_speed_kmh) AS avg_speed_kmh,
			COUNT(estimated_energy_consumed_kwh) AS energy_estimate_rows,
			COUNT(outside_temp_avg) AS temperature_rows,
			COUNT(start_elevation) + COUNT(end_elevation) AS elevation_rows
		FROM drive_metrics
		GROUP BY bucket
		ORDER BY bucket`, bucketSQL, joins),
		func() []any {
			args := []any{carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time}
			if needsTZ {
				args = append(args, timeRange.Timezone)
			}
			return args
		}()...,
	)
	if err != nil {
		return nil, V2EfficiencyStats{}, err
	}
	defer rows.Close()

	var items []V2EfficiencyFactorItem
	var stats V2EfficiencyStats
	for rows.Next() {
		var item V2EfficiencyFactorItem
		var estimatedEnergy sql.NullFloat64
		var avgConsumption sql.NullFloat64
		var avgTemperature sql.NullFloat64
		var avgSpeed sql.NullFloat64
		var energyRows, temperatureRows, elevationRows int64
		if err := rows.Scan(&item.Bucket, &item.DriveCount, &item.DistanceKM, &estimatedEnergy, &avgConsumption, &avgTemperature, &avgSpeed, &energyRows, &temperatureRows, &elevationRows); err != nil {
			return nil, stats, err
		}
		if estimatedEnergy.Valid {
			item.EstimatedEnergyConsumedKWh = &estimatedEnergy.Float64
		}
		if avgConsumption.Valid {
			item.AvgConsumptionWhPerKM = &avgConsumption.Float64
		}
		if avgTemperature.Valid {
			item.AvgTemperatureC = &avgTemperature.Float64
		}
		if avgSpeed.Valid {
			item.AvgSpeedKMH = &avgSpeed.Float64
		}
		stats.DriveRows += item.DriveCount
		stats.EnergyEstimateRows += energyRows
		stats.TemperatureRows += temperatureRows
		stats.ElevationRows += elevationRows
		items = append(items, item)
	}
	return items, stats, rows.Err()
}

func efficiencyFactorBucketSQL(dimension string) (string, string, error) {
	switch dimension {
	case "temperature":
		return `CASE
			WHEN drives.outside_temp_avg IS NULL THEN 'unknown'
			WHEN drives.outside_temp_avg < -10 THEN '<-10'
			WHEN drives.outside_temp_avg < 0 THEN '-10-000'
			WHEN drives.outside_temp_avg < 10 THEN '000-010'
			WHEN drives.outside_temp_avg < 20 THEN '010-020'
			WHEN drives.outside_temp_avg < 30 THEN '020-030'
			WHEN drives.outside_temp_avg < 40 THEN '030-040'
			ELSE '040+'
		END`, "", nil
	case "speed":
		return `CASE
			WHEN drives.duration_min <= 0 OR drives.distance IS NULL THEN 'unknown'
			WHEN drives.distance / drives.duration_min * 60 < 30 THEN '000-030'
			WHEN drives.distance / drives.duration_min * 60 < 60 THEN '030-060'
			WHEN drives.distance / drives.duration_min * 60 < 90 THEN '060-090'
			WHEN drives.distance / drives.duration_min * 60 < 120 THEN '090-120'
			ELSE '120+'
		END`, "", nil
	case "distance":
		return `CASE
			WHEN drives.distance < 5 THEN '000-005'
			WHEN drives.distance < 10 THEN '005-010'
			WHEN drives.distance < 25 THEN '010-025'
			WHEN drives.distance < 50 THEN '025-050'
			WHEN drives.distance < 100 THEN '050-100'
			ELSE '100+'
		END`, "", nil
	case "elevation":
		return `CASE
			WHEN start_position.elevation IS NULL OR end_position.elevation IS NULL THEN 'unknown'
			WHEN end_position.elevation - start_position.elevation < -200 THEN '<-200'
			WHEN end_position.elevation - start_position.elevation < -100 THEN '-200--100'
			WHEN end_position.elevation - start_position.elevation < -25 THEN '-100--025'
			WHEN end_position.elevation - start_position.elevation <= 25 THEN '-025-025'
			WHEN end_position.elevation - start_position.elevation <= 100 THEN '025-100'
			WHEN end_position.elevation - start_position.elevation <= 200 THEN '100-200'
			ELSE '200+'
		END`, "", nil
	case "location":
		return `COALESCE(start_geofence.name, CONCAT_WS(', ', COALESCE(start_address.name, nullif(CONCAT_WS(' ', start_address.road, start_address.house_number), '')), start_address.city), 'unknown')`,
			`LEFT JOIN addresses start_address ON drives.start_address_id = start_address.id
			LEFT JOIN geofences start_geofence ON drives.start_geofence_id = start_geofence.id`, nil
	case "hour_of_day":
		return `LPAD(EXTRACT(HOUR FROM drives.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4)::int::text, 2, '0')`, "", nil
	case "day_of_week":
		return `TRIM(TO_CHAR(drives.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4, 'Day'))`, "", nil
	default:
		return "", "", errV2InvalidEfficiencyDimension
	}
}
