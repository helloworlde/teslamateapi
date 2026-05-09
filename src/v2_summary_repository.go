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
	if err := r.loadVehicleSummary(ctx, carID, &summary); err != nil {
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
	var avgConsumption, bestEfficiency, worstEfficiency, netEnergy sql.NullFloat64
	var longestDrive, avgSpeed, peakDrive, peakRegen sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS drive_count,
			COALESCE(SUM(distance), 0) AS distance_km,
			COALESCE(SUM(duration_min), 0) AS duration_min,
			COALESCE(MAX(speed_max), 0) AS max_speed_kmh,
			MAX(duration_min) AS longest_drive_duration_min,
			AVG(CASE WHEN duration_min > 0 THEN distance / duration_min * 60 END) AS avg_speed_kmh,
			MAX(power_max) AS peak_drive_power_kw,
			-MIN(NULLIF(power_min, 0)) AS peak_regen_power_kw,
			SUM(
				CASE
					WHEN distance > 0
						AND start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency
				END
			) AS net_energy_kwh,
			AVG(
				CASE
					WHEN distance > 0
						AND start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / distance * 1000
				END
			) AS avg_consumption_wh_per_km,
			MIN(
				CASE
					WHEN distance > 0
						AND start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / distance * 1000
				END
			) AS best_efficiency_wh_per_km,
			MAX(
				CASE
					WHEN distance > 0
						AND start_rated_range_km IS NOT NULL
						AND end_rated_range_km IS NOT NULL
						AND cars.efficiency IS NOT NULL
						AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / distance * 1000
				END
			) AS worst_efficiency_wh_per_km
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
		&longestDrive,
		&avgSpeed,
		&peakDrive,
		&peakRegen,
		&netEnergy,
		&avgConsumption,
		&bestEfficiency,
		&worstEfficiency,
	)
	if err != nil {
		return err
	}
	if summary.Driving.DriveCount > 0 {
		avg := summary.Driving.DistanceKM / float64(summary.Driving.DriveCount)
		summary.Driving.AvgTripDistanceKM = &avg
		avgDur := summary.Driving.DurationMin / float64(summary.Driving.DriveCount)
		summary.Driving.AvgDurationMin = &avgDur
	}
	if longestDrive.Valid {
		summary.Driving.LongestDriveDurationMin = &longestDrive.Float64
	}
	if avgSpeed.Valid {
		summary.Driving.AvgSpeedKMH = &avgSpeed.Float64
	}
	if peakDrive.Valid {
		summary.Driving.PeakDrivePowerKW = &peakDrive.Float64
	}
	if peakRegen.Valid {
		summary.Driving.PeakRegenPowerKW = &peakRegen.Float64
	}
	if netEnergy.Valid {
		summary.Driving.NetEnergyKWh = &netEnergy.Float64
	}
	if avgConsumption.Valid {
		summary.Driving.AvgConsumptionWhPerKM = avgConsumption.Float64
	}
	if bestEfficiency.Valid {
		summary.Driving.BestEfficiencyWhPerKM = &bestEfficiency.Float64
	}
	if worstEfficiency.Valid {
		summary.Driving.WorstEfficiencyWhPerKM = &worstEfficiency.Float64
	}
	stats.DriveRows = summary.Driving.DriveCount
	return nil
}

func (r PostgresV2SummaryRepository) loadChargingSummary(ctx context.Context, carID int64, start timeBound, end timeBound, summary *V2Summary, stats *V2SummaryStats) error {
	var longestSession, avgEnergy, largestSession, avgPower, maxPower, avgCost, maxCost sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS session_count,
			COALESCE(SUM(cp.charge_energy_added), 0) AS energy_added_kwh,
			COALESCE(SUM(GREATEST(COALESCE(cp.charge_energy_used, 0), COALESCE(cp.charge_energy_added, 0))), 0) AS energy_used_kwh,
			COALESCE(SUM(cp.duration_min), 0) AS duration_min,
			MAX(cp.duration_min) AS longest_session_duration_min,
			AVG(cp.charge_energy_added) AS avg_energy_added_kwh,
			MAX(cp.charge_energy_added) AS largest_session_kwh,
			AVG(NULLIF(max_charge.charger_power, 0)) AS avg_power_kw,
			MAX(NULLIF(max_charge.charger_power, 0)) AS max_power_kw,
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
		&summary.Charging.EnergyAddedKWh,
		&summary.Charging.EnergyUsedKWh,
		&summary.Charging.DurationMin,
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
		avgDur := summary.Charging.DurationMin / float64(summary.Charging.SessionCount)
		summary.Charging.AvgDurationMin = &avgDur
	}
	if longestSession.Valid {
		summary.Charging.LongestSessionDurationMin = &longestSession.Float64
	}
	if avgEnergy.Valid {
		summary.Charging.AvgEnergyAddedKWh = &avgEnergy.Float64
	}
	if largestSession.Valid {
		summary.Charging.LargestSessionKWh = &largestSession.Float64
	}
	if avgPower.Valid {
		summary.Charging.AvgPowerKW = &avgPower.Float64
	}
	if maxPower.Valid {
		summary.Charging.MaxPowerKW = &maxPower.Float64
	}
	if summary.Charging.EnergyUsedKWh > 0 {
		eff := summary.Charging.EnergyAddedKWh / summary.Charging.EnergyUsedKWh * 100
		summary.Charging.ChargeEfficiencyPercent = &eff
	}
	if avgCost.Valid {
		summary.Charging.AvgCost = &avgCost.Float64
	}
	if maxCost.Valid {
		summary.Charging.MaxCost = &maxCost.Float64
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

func (r PostgresV2SummaryRepository) loadVehicleSummary(ctx context.Context, carID int64, summary *V2Summary) error {
	var odometer sql.NullFloat64
	var ratedEfficiency sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT MAX(d.end_km) AS odometer_km, c.efficiency AS rated_efficiency
		FROM drives d
		LEFT JOIN cars c ON c.id = d.car_id
		WHERE d.car_id = $1 AND d.end_date IS NOT NULL`,
		carID,
	).Scan(&odometer, &ratedEfficiency)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	summary.Vehicle.TrackedDistanceKM = summary.Driving.DistanceKM
	summary.Vehicle.TrackedDrives = summary.Driving.DriveCount
	summary.Vehicle.TrackedCharges = summary.Charging.SessionCount

	if odometer.Valid {
		summary.Vehicle.OdometerKM = &odometer.Float64
		if summary.Driving.DistanceKM > 0 && odometer.Float64 > 0 {
			pct := summary.Driving.DistanceKM / odometer.Float64 * 100
			summary.Vehicle.OdometerCoveragePercent = &pct
		}
	}
	if ratedEfficiency.Valid {
		v := ratedEfficiency.Float64 * 100
		summary.Vehicle.RatedEfficiencyKWhPer100KM = &v
	}
	if summary.Driving.DistanceKM > 0 {
		cons := summary.Charging.EnergyAddedKWh / summary.Driving.DistanceKM * 100
		summary.Vehicle.TrackedConsumptionKWhPer100KM = &cons
		wall := summary.Charging.EnergyUsedKWh / summary.Driving.DistanceKM * 100
		summary.Vehicle.TrackedWallKWhPer100KM = &wall
	}
	if summary.Charging.ChargeEfficiencyPercent != nil {
		summary.Vehicle.ChargeEfficiencyPercent = summary.Charging.ChargeEfficiencyPercent
	}
	return nil
}
