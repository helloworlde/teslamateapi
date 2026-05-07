package main

import (
	"context"
	"database/sql"
)

type V2SummaryRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, start timeBound, end timeBound) (V2Summary, V2SummaryStats, error)
}

type timeBound struct {
	Time string
}

type V2SummaryStats struct {
	DriveRows    int64
	ChargeRows   int64
	StateRows    int64
	PositionRows int64
	UpdateRows   int64
	CostRows     int64
}

type PostgresV2SummaryRepository struct {
	db *sql.DB
}

func NewPostgresV2SummaryRepository(db *sql.DB) PostgresV2SummaryRepository {
	return PostgresV2SummaryRepository{db: db}
}

func (r PostgresV2SummaryRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2SummaryRepository) Summary(ctx context.Context, carID int64, start timeBound, end timeBound) (V2Summary, V2SummaryStats, error) {
	var summary V2Summary
	var stats V2SummaryStats

	if err := r.loadDrivingSummary(ctx, carID, start, end, &summary, &stats); err != nil {
		return summary, stats, err
	}
	if err := r.loadChargingSummary(ctx, carID, start, end, &summary, &stats); err != nil {
		return summary, stats, err
	}
	if err := r.loadParkingSummary(ctx, carID, start, end, &summary, &stats); err != nil {
		return summary, stats, err
	}
	if err := r.loadBatterySummary(ctx, carID, start, end, &summary, &stats); err != nil {
		return summary, stats, err
	}
	if err := r.loadUpdateSummary(ctx, carID, start, end, &summary, &stats); err != nil {
		return summary, stats, err
	}

	if summary.Driving.DistanceKM > 0 {
		costPerKM := summary.Charging.Cost / summary.Driving.DistanceKM
		costPer100KM := costPerKM * 100
		summary.Cost.CostPerKM = &costPerKM
		summary.Cost.CostPer100KM = &costPer100KM
	}
	summary.Cost.ChargingCost = summary.Charging.Cost

	return summary, stats, nil
}

func (r PostgresV2SummaryRepository) loadDrivingSummary(ctx context.Context, carID int64, start timeBound, end timeBound, summary *V2Summary, stats *V2SummaryStats) error {
	var avgConsumption sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS drive_count,
			COALESCE(SUM(distance), 0) AS distance_km,
			COALESCE(SUM(duration_min), 0) AS duration_min,
			COALESCE(MAX(speed_max), 0) AS max_speed_kmh,
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
			) AS avg_consumption_wh_per_km
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND drives.start_date >= $2
			AND drives.start_date < $3`,
		carID, start.Time, end.Time,
	).Scan(
		&summary.Driving.DriveCount,
		&summary.Driving.DistanceKM,
		&summary.Driving.DurationMin,
		&summary.Driving.MaxSpeedKMH,
		&avgConsumption,
	)
	if err != nil {
		return err
	}
	if avgConsumption.Valid {
		summary.Driving.AvgConsumptionWhPerKM = avgConsumption.Float64
	}
	stats.DriveRows = summary.Driving.DriveCount
	return nil
}

func (r PostgresV2SummaryRepository) loadChargingSummary(ctx context.Context, carID int64, start timeBound, end timeBound, summary *V2Summary, stats *V2SummaryStats) error {
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS session_count,
			COALESCE(SUM(charge_energy_added), 0) AS energy_added_kwh,
			COALESCE(SUM(GREATEST(COALESCE(charge_energy_used, 0), COALESCE(charge_energy_added, 0))), 0) AS energy_used_kwh,
			COALESCE(SUM(duration_min), 0) AS duration_min,
			COALESCE(SUM(cost), 0) AS cost,
			COUNT(cost) AS cost_rows
		FROM charging_processes
		WHERE car_id = $1
			AND end_date IS NOT NULL
			AND start_date >= $2
			AND start_date < $3`,
		carID, start.Time, end.Time,
	).Scan(
		&summary.Charging.SessionCount,
		&summary.Charging.EnergyAddedKWh,
		&summary.Charging.EnergyUsedKWh,
		&summary.Charging.DurationMin,
		&summary.Charging.Cost,
		&stats.CostRows,
	)
	if err != nil {
		return err
	}
	stats.ChargeRows = summary.Charging.SessionCount
	return nil
}

func (r PostgresV2SummaryRepository) loadParkingSummary(ctx context.Context, carID int64, start timeBound, end timeBound, summary *V2Summary, stats *V2SummaryStats) error {
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS state_rows,
			COALESCE(SUM(EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamptz), $3::timestamptz) - GREATEST(start_date, $2::timestamptz))) / 60), 0) AS parked_duration_min,
			COALESCE(SUM(CASE WHEN state = 'asleep' THEN EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamptz), $3::timestamptz) - GREATEST(start_date, $2::timestamptz))) / 60 ELSE 0 END), 0) AS asleep_duration_min,
			COALESCE(SUM(CASE WHEN state = 'online' THEN EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamptz), $3::timestamptz) - GREATEST(start_date, $2::timestamptz))) / 60 ELSE 0 END), 0) AS online_duration_min,
			COALESCE(SUM(CASE WHEN state = 'offline' THEN EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamptz), $3::timestamptz) - GREATEST(start_date, $2::timestamptz))) / 60 ELSE 0 END), 0) AS offline_duration_min
		FROM states
		WHERE car_id = $1
			AND start_date < $3::timestamptz
			AND COALESCE(end_date, $3::timestamptz) > $2::timestamptz`,
		carID, start.Time, end.Time,
	).Scan(
		&stats.StateRows,
		&summary.Parking.ParkedDurationMin,
		&summary.Parking.AsleepDurationMin,
		&summary.Parking.OnlineDurationMin,
		&summary.Parking.OfflineDurationMin,
	)
	return err
}

func (r PostgresV2SummaryRepository) loadBatterySummary(ctx context.Context, carID int64, start timeBound, end timeBound, summary *V2Summary, stats *V2SummaryStats) error {
	var batteryLevel sql.NullInt64
	var ratedRange sql.NullFloat64
	var idealRange sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT battery_level, rated_battery_range_km, ideal_battery_range_km
		FROM positions
		WHERE car_id = $1
			AND date >= $2
			AND date < $3
		ORDER BY date DESC
		LIMIT 1`,
		carID, start.Time, end.Time,
	).Scan(&batteryLevel, &ratedRange, &idealRange)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	stats.PositionRows = 1
	if batteryLevel.Valid {
		value := batteryLevel.Int64
		summary.Battery.LatestBatteryLevelPercent = &value
	}
	if ratedRange.Valid {
		value := ratedRange.Float64
		summary.Battery.LatestRatedRangeKM = &value
	}
	if idealRange.Valid {
		value := idealRange.Float64
		summary.Battery.LatestIdealRangeKM = &value
	}
	return nil
}

func (r PostgresV2SummaryRepository) loadUpdateSummary(ctx context.Context, carID int64, start timeBound, end timeBound, summary *V2Summary, stats *V2SummaryStats) error {
	var latestVersion sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS update_count,
			(
				SELECT version
				FROM updates latest
				WHERE latest.car_id = $1
					AND latest.start_date >= $2
					AND latest.start_date < $3
					AND latest.version IS NOT NULL
				ORDER BY latest.start_date DESC
				LIMIT 1
			) AS latest_version
		FROM updates
		WHERE car_id = $1
			AND start_date >= $2
			AND start_date < $3`,
		carID, start.Time, end.Time,
	).Scan(&summary.Updates.UpdateCount, &latestVersion)
	if err != nil {
		return err
	}
	stats.UpdateRows = summary.Updates.UpdateCount
	if latestVersion.Valid {
		value := latestVersion.String
		summary.Updates.LatestVersion = &value
	}
	return nil
}
