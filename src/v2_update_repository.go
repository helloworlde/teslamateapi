package main

import (
	"context"
	"database/sql"
	"time"
)

// @name PostgresV2UpdateRepository
type PostgresV2UpdateRepository struct {
	db *sql.DB
}

func NewPostgresV2UpdateRepository(db *sql.DB) PostgresV2UpdateRepository {
	return PostgresV2UpdateRepository{db: db}
}

func (r PostgresV2UpdateRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2UpdateRepository) Updates(ctx context.Context, carID int64, start, end timeBound) (V2UpdateAnalyticsResponse, int64, error) {
	var response V2UpdateAnalyticsResponse
	var latestVersion sql.NullString
	var latestUpdatedAt sql.NullTime
	var avgDuration sql.NullFloat64

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS update_count,
			MAX(version) FILTER (WHERE version IS NOT NULL) AS latest_version,
			MAX(start_date) AS latest_updated_at,
			AVG(EXTRACT(EPOCH FROM (end_date - start_date)) / 60) FILTER (WHERE end_date IS NOT NULL) AS avg_update_duration_min
		FROM updates
		WHERE car_id = $1 AND start_date >= $2 AND start_date < $3`,
		carID, start.Time, end.Time,
	).Scan(&response.UpdateCount, &latestVersion, &latestUpdatedAt, &avgDuration)
	if err != nil {
		return response, 0, err
	}

	if latestVersion.Valid {
		v := latestVersion.String
		response.LatestVersion = &v
	}
	if latestUpdatedAt.Valid {
		s := latestUpdatedAt.Time.Format(time.RFC3339)
		response.LatestUpdatedAt = &s
	}
	if avgDuration.Valid {
		response.AvgUpdateDurationMin = &avgDuration.Float64
	}

	rows, err := r.db.QueryContext(ctx, `
		WITH version_windows AS (
			SELECT
				version, start_date, end_date,
				COALESCE(end_date, start_date) AS window_start,
				COALESCE(
					LEAD(start_date) OVER (PARTITION BY car_id ORDER BY start_date),
					NOW()
				) AS window_end,
				EXTRACT(EPOCH FROM (
					start_date - LAG(end_date) OVER (PARTITION BY car_id ORDER BY start_date)
				)) / 86400 AS days_since_prior
			FROM updates
			WHERE car_id = $1 AND start_date >= $2 AND start_date < $3
		)
		SELECT
			vw.version, vw.start_date, vw.end_date,
			EXTRACT(EPOCH FROM (vw.end_date - vw.start_date)) / 60 AS update_duration_min,
			ROUND(vw.days_since_prior)::bigint AS days_since_prior,
			vw.window_start, vw.window_end,
			EXTRACT(EPOCH FROM (vw.window_end - vw.window_start)) / 60 AS interval_min,
			ds.trip_count, ds.driving_duration_min, ds.distance_km, ds.net_drive_energy_kwh,
			cs.session_count, cs.charging_duration_min, cs.battery_energy_kwh, cs.wall_energy_kwh, cs.charge_cost,
			ist.inactive_min
		FROM version_windows vw
		LEFT JOIN LATERAL (
			SELECT
				COUNT(*) AS trip_count,
				COALESCE(SUM(d.duration_min), 0) AS driving_duration_min,
				COALESCE(SUM(d.distance), 0) AS distance_km,
				SUM(
					CASE
						WHEN d.distance > 0
							AND d.start_rated_range_km IS NOT NULL
							AND d.end_rated_range_km IS NOT NULL
							AND cars.efficiency IS NOT NULL
							AND (d.start_rated_range_km - d.end_rated_range_km) > 0
						THEN (d.start_rated_range_km - d.end_rated_range_km) * cars.efficiency
					END
				) AS net_drive_energy_kwh
			FROM drives d
			LEFT JOIN cars ON cars.id = d.car_id
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
				AND d.start_date >= vw.window_start AND d.start_date < vw.window_end
		) ds ON true
		LEFT JOIN LATERAL (
			SELECT
				COUNT(*) AS session_count,
				COALESCE(SUM(cp.duration_min), 0) AS charging_duration_min,
				COALESCE(SUM(cp.charge_energy_added), 0) AS battery_energy_kwh,
				COALESCE(SUM(GREATEST(COALESCE(cp.charge_energy_used, 0), COALESCE(cp.charge_energy_added, 0))), 0) AS wall_energy_kwh,
				SUM(cp.cost) AS charge_cost
			FROM charging_processes cp
			WHERE cp.car_id = $1 AND cp.end_date IS NOT NULL
				AND cp.start_date >= vw.window_start AND cp.start_date < vw.window_end
		) cs ON true
		LEFT JOIN LATERAL (
			SELECT
				COALESCE(SUM(
					EXTRACT(EPOCH FROM (
						LEAST(COALESCE(s.end_date, vw.window_end), vw.window_end) -
						GREATEST(s.start_date, vw.window_start)
					)) / 60
				) FILTER (WHERE s.state IN ('asleep', 'offline')), 0) AS inactive_min
			FROM states s
			WHERE s.car_id = $1
				AND s.start_date < vw.window_end
				AND COALESCE(s.end_date, vw.window_end) > vw.window_start
		) ist ON true
		ORDER BY vw.start_date DESC`,
		carID, start.Time, end.Time,
	)
	if err != nil {
		return response, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var v V2UpdateVersion
		var versionStr sql.NullString
		var endDate, windowStart, windowEnd sql.NullTime
		var updateDuration, intervalMin sql.NullFloat64
		var daysSinceInt sql.NullInt64
		var tripCount sql.NullInt64
		var drivingDuration, distanceKM, netEnergy sql.NullFloat64
		var sessionCount sql.NullInt64
		var chargingDuration, batteryEnergy, wallEnergy, chargeCost, inactiveMin sql.NullFloat64
		var startDate time.Time
		if err := rows.Scan(
			&versionStr, &startDate, &endDate,
			&updateDuration, &daysSinceInt,
			&windowStart, &windowEnd, &intervalMin,
			&tripCount, &drivingDuration, &distanceKM, &netEnergy,
			&sessionCount, &chargingDuration, &batteryEnergy, &wallEnergy, &chargeCost,
			&inactiveMin,
		); err != nil {
			return response, 0, err
		}
		if versionStr.Valid {
			v.Version = versionStr.String
		}
		v.StartedAt = startDate.Format(time.RFC3339)
		if endDate.Valid {
			s := endDate.Time.Format(time.RFC3339)
			v.CompletedAt = &s
		}
		if updateDuration.Valid {
			v.DurationMin = &updateDuration.Float64
		}
		if daysSinceInt.Valid {
			v.DaysSincePrior = &daysSinceInt.Int64
		}
		if windowStart.Valid {
			s := windowStart.Time.Format(time.RFC3339)
			v.WindowStart = &s
		}
		if windowEnd.Valid {
			s := windowEnd.Time.Format(time.RFC3339)
			v.WindowEnd = &s
		}
		if intervalMin.Valid {
			v.IntervalMin = &intervalMin.Float64
		}
		if tripCount.Valid {
			v.DrivingTripCount = &tripCount.Int64
		}
		if drivingDuration.Valid {
			v.DrivingDurationMin = &drivingDuration.Float64
		}
		if distanceKM.Valid {
			v.DrivingDistanceKM = &distanceKM.Float64
		}
		if netEnergy.Valid {
			v.NetDriveEnergyKWh = &netEnergy.Float64
			if distanceKM.Valid && distanceKM.Float64 > 0 {
				eff := netEnergy.Float64 / distanceKM.Float64 * 1000
				v.DriveEfficiencyWhPerKM = &eff
				cons := netEnergy.Float64 / distanceKM.Float64 * 100
				v.AvgConsumptionKWhPer100KM = &cons
			}
		}
		if sessionCount.Valid {
			v.ChargingSessionCount = &sessionCount.Int64
		}
		if chargingDuration.Valid {
			v.ChargingDurationMin = &chargingDuration.Float64
		}
		if batteryEnergy.Valid {
			v.BatteryEnergyKWh = &batteryEnergy.Float64
		}
		if wallEnergy.Valid {
			v.WallEnergyKWh = &wallEnergy.Float64
		}
		if wallEnergy.Valid && wallEnergy.Float64 > 0 && batteryEnergy.Valid {
			eff := batteryEnergy.Float64 / wallEnergy.Float64 * 100
			v.ChargingEfficiencyPercent = &eff
		}
		if chargeCost.Valid {
			v.ChargeCost = &chargeCost.Float64
		}
		if inactiveMin.Valid {
			v.InactiveDurationMin = &inactiveMin.Float64
		}
		response.Versions = append(response.Versions, v)
	}
	if err := rows.Err(); err != nil {
		return response, 0, err
	}

	return response, response.UpdateCount, nil
}
