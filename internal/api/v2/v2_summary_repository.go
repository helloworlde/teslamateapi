package v2

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

// @name V2SummaryStats
type V2SummaryStats struct {
	DriveRows    int64
	ChargeRows   int64
	StateRows    int64
	PositionRows int64
	UpdateRows   int64
	CostRows     int64
}

// @name PostgresV2SummaryRepository
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
	if err := r.loadVehicleSummary(ctx, carID, &summary); err != nil {
		return summary, stats, err
	}

	if summary.Driving.Distance > 0 {
		costPer100KM := summary.Charging.Cost / summary.Driving.Distance * 100
		summary.Cost.CostPerDistance = &costPer100KM
	}
	summary.Cost.ChargingCost = summary.Charging.Cost

	return summary, stats, nil
}

func (r PostgresV2SummaryRepository) loadDrivingSummary(ctx context.Context, carID int64, start timeBound, end timeBound, summary *V2Summary, stats *V2SummaryStats) error {
	var avgConsumption, bestConsumption, worstConsumption, netEnergy sql.NullFloat64
	var longestDrive, avgSpeed, maxSpeed, peakDrive, peakRegen sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS drive_count,
			COALESCE(SUM(distance), 0) AS distance,
			COALESCE(SUM(duration_min) * 60, 0) AS duration,
			MAX(speed_max) AS max_speed,
			MAX(duration_min * 60) AS longest_drive_duration,
			AVG(CASE WHEN duration_min > 0 THEN distance / duration_min * 60 END) AS avg_speed,
			MAX(power_max) AS peak_drive_power,
			-MIN(NULLIF(power_min, 0)) AS peak_regen_power,
			SUM(
				CASE
					WHEN distance > 0
						AND start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency
				END
			) AS net_energy,
			AVG(
				CASE
					WHEN distance > 0
						AND start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / distance * 1000
				END
			) AS avg_consumption,
			MIN(
				CASE
					WHEN distance > 0
						AND start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / distance * 1000
				END
			) AS best_consumption,
			MAX(
				CASE
					WHEN distance > 0
						AND start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / distance * 1000
				END
			) AS worst_consumption
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND drives.start_date >= $2
			AND drives.start_date < $3`,
		carID, start.Time, end.Time,
	).Scan(
		&summary.Driving.DriveCount,
		&summary.Driving.Distance,
		&summary.Driving.Duration,
		&maxSpeed,
		&longestDrive,
		&avgSpeed,
		&peakDrive,
		&peakRegen,
		&netEnergy,
		&avgConsumption,
		&bestConsumption,
		&worstConsumption,
	)
	if err != nil {
		return err
	}
	if summary.Driving.DriveCount > 0 {
		avg := summary.Driving.Distance / float64(summary.Driving.DriveCount)
		summary.Driving.AvgDistance = &avg
		avgDur := summary.Driving.Duration / float64(summary.Driving.DriveCount)
		summary.Driving.AvgDuration = &avgDur
	}
	if maxSpeed.Valid {
		summary.Driving.MaxSpeed = &maxSpeed.Float64
	}
	if longestDrive.Valid {
		summary.Driving.LongestDriveDuration = &longestDrive.Float64
	}
	if avgSpeed.Valid {
		summary.Driving.AvgSpeed = &avgSpeed.Float64
	}
	if peakDrive.Valid {
		summary.Driving.PeakDrivePower = &peakDrive.Float64
	}
	if peakRegen.Valid {
		summary.Driving.PeakRegenPower = &peakRegen.Float64
	}
	if netEnergy.Valid {
		summary.Driving.NetEnergy = &netEnergy.Float64
	}
	if avgConsumption.Valid {
		summary.Driving.AvgConsumption = &avgConsumption.Float64
	}
	if bestConsumption.Valid {
		summary.Driving.BestConsumption = &bestConsumption.Float64
	}
	if worstConsumption.Valid {
		summary.Driving.WorstConsumption = &worstConsumption.Float64
	}
	stats.DriveRows = summary.Driving.DriveCount
	return nil
}

func (r PostgresV2SummaryRepository) loadChargingSummary(ctx context.Context, carID int64, start timeBound, end timeBound, summary *V2Summary, stats *V2SummaryStats) error {
	var longestSession, avgEnergy, largestSession, avgPower, maxPower, avgCost, maxCost sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS session_count,
			COALESCE(SUM(cp.charge_energy_added), 0) AS energy_added,
			COALESCE(SUM(GREATEST(COALESCE(cp.charge_energy_used, 0), COALESCE(cp.charge_energy_added, 0))), 0) AS energy_used,
			COALESCE(SUM(cp.duration_min) * 60, 0) AS duration_seconds,
			MAX(cp.duration_min) * 60 AS longest_session_duration,
			AVG(cp.charge_energy_added) AS avg_energy_added,
			MAX(cp.charge_energy_added) AS largest_session,
			AVG(NULLIF(max_charge.charger_power, 0)) AS avg_power,
			MAX(NULLIF(max_charge.charger_power, 0)) AS max_power,
			COALESCE(SUM(cp.cost), 0) AS cost,
			COUNT(cp.cost) AS cost_rows,
			AVG(cp.cost) AS avg_cost,
			MAX(cp.cost) AS max_cost
		FROM charging_processes cp
		LEFT JOIN LATERAL (
			SELECT MAX(charger_power) AS charger_power
			FROM charges
			WHERE charges.charging_process_id = cp.id
		) max_charge ON true
		WHERE cp.car_id = $1
			AND cp.end_date IS NOT NULL
			AND cp.start_date >= $2
			AND cp.start_date < $3`,
		carID, start.Time, end.Time,
	).Scan(
		&summary.Charging.SessionCount,
		&summary.Charging.EnergyAdded,
		&summary.Charging.EnergyUsed,
		&summary.Charging.Duration,
		&longestSession,
		&avgEnergy,
		&largestSession,
		&avgPower,
		&maxPower,
		&summary.Charging.Cost,
		&stats.CostRows,
		&avgCost,
		&maxCost,
	)
	if err != nil {
		return err
	}
	if summary.Charging.SessionCount > 0 {
		avgDur := summary.Charging.Duration / float64(summary.Charging.SessionCount)
		summary.Charging.AvgDuration = &avgDur
	}
	if longestSession.Valid {
		summary.Charging.LongestSessionDuration = &longestSession.Float64
	}
	if avgEnergy.Valid {
		summary.Charging.AvgEnergyAdded = &avgEnergy.Float64
	}
	if largestSession.Valid {
		summary.Charging.LargestSession = &largestSession.Float64
	}
	if avgPower.Valid {
		summary.Charging.AvgPower = &avgPower.Float64
	}
	if maxPower.Valid {
		summary.Charging.MaxPower = &maxPower.Float64
	}
	if summary.Charging.EnergyUsed > 0 {
		eff := summary.Charging.EnergyAdded / summary.Charging.EnergyUsed * 100
		summary.Charging.ChargingEfficiency = &eff
	}
	if avgCost.Valid {
		summary.Charging.AvgCost = &avgCost.Float64
	}
	if maxCost.Valid {
		summary.Charging.MaxCost = &maxCost.Float64
	}
	if summary.Charging.EnergyAdded > 0 && summary.Charging.Cost > 0 {
		v := summary.Charging.Cost / summary.Charging.EnergyAdded
		summary.Charging.AvgCostPerEnergy = &v
	}
	stats.ChargeRows = summary.Charging.SessionCount
	return nil
}

func (r PostgresV2SummaryRepository) loadParkingSummary(ctx context.Context, carID int64, start timeBound, end timeBound, summary *V2Summary, stats *V2SummaryStats) error {
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS state_rows,
			COALESCE(SUM(EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamptz), $3::timestamptz) - GREATEST(start_date, $2::timestamptz)))), 0) AS parked_duration_seconds,
			COALESCE(SUM(CASE WHEN state = 'asleep' THEN EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamptz), $3::timestamptz) - GREATEST(start_date, $2::timestamptz))) ELSE 0 END), 0) AS asleep_duration_seconds,
			COALESCE(SUM(CASE WHEN state = 'online' THEN EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamptz), $3::timestamptz) - GREATEST(start_date, $2::timestamptz))) ELSE 0 END), 0) AS online_duration_seconds,
			COALESCE(SUM(CASE WHEN state = 'offline' THEN EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamptz), $3::timestamptz) - GREATEST(start_date, $2::timestamptz))) ELSE 0 END), 0) AS offline_duration_seconds
		FROM states
		WHERE car_id = $1
			AND start_date < $3::timestamptz
			AND COALESCE(end_date, $3::timestamptz) > $2::timestamptz`,
		carID, start.Time, end.Time,
	).Scan(
		&stats.StateRows,
		&summary.Parking.ParkedDuration,
		&summary.Parking.AsleepDuration,
		&summary.Parking.OnlineDuration,
		&summary.Parking.OfflineDuration,
	)
	if err != nil {
		return err
	}
	summary.Parking.StateTransitionCount = stats.StateRows
	return nil
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
		summary.Battery.LatestLevel = &value
	}
	if ratedRange.Valid {
		value := ratedRange.Float64
		summary.Battery.LatestRatedRange = &value
	}
	if idealRange.Valid {
		value := idealRange.Float64
		summary.Battery.LatestIdealRange = &value
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

func (r PostgresV2SummaryRepository) loadVehicleSummary(ctx context.Context, carID int64, summary *V2Summary) error {
	var odometer sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT MAX(d.end_km) AS odometer
		FROM drives d
		WHERE d.car_id = $1 AND d.end_date IS NOT NULL`,
		carID,
	).Scan(&odometer)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	summary.Vehicle.TrackedDistance = summary.Driving.Distance
	summary.Vehicle.TrackedDrives = summary.Driving.DriveCount
	summary.Vehicle.TrackedCharges = summary.Charging.SessionCount

	if odometer.Valid {
		summary.Vehicle.Odometer = &odometer.Float64
	}
	return nil
}
