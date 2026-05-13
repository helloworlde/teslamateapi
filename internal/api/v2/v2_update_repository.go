package v2

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
	var medianDays sql.NullFloat64

	err := r.db.QueryRowContext(ctx, `
		WITH gaps AS (
			SELECT EXTRACT(EPOCH FROM (
				start_date - LAG(start_date) OVER (ORDER BY start_date)
			)) / 86400.0 AS days_between
			FROM updates
			WHERE car_id = $1 AND start_date >= $2 AND start_date < $3
		)
		SELECT
			(SELECT COUNT(*) FROM updates WHERE car_id = $1 AND start_date >= $2 AND start_date < $3) AS update_count,
			(SELECT MAX(version) FILTER (WHERE version IS NOT NULL) FROM updates WHERE car_id = $1 AND start_date >= $2 AND start_date < $3) AS latest_version,
			(SELECT MAX(start_date) FROM updates WHERE car_id = $1 AND start_date >= $2 AND start_date < $3) AS latest_updated_at,
			(SELECT AVG(EXTRACT(EPOCH FROM (end_date - start_date))) FILTER (WHERE end_date IS NOT NULL)
			   FROM updates WHERE car_id = $1 AND start_date >= $2 AND start_date < $3) AS avg_update_duration,
			(SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY days_between) FROM gaps WHERE days_between IS NOT NULL) AS median_days_between_updates
		`,
		carID, start.Time, end.Time,
	).Scan(&response.UpdateCount, &latestVersion, &latestUpdatedAt, &avgDuration, &medianDays)
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
		response.AvgUpdateDuration = &avgDuration.Float64
	}
	if medianDays.Valid {
		response.MedianDaysBetweenUpdates = &medianDays.Float64
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
			EXTRACT(EPOCH FROM (vw.end_date - vw.start_date)) AS update_duration,
			ROUND(vw.days_since_prior)::bigint AS days_since_prior,
			vw.window_start, vw.window_end,
			EXTRACT(EPOCH FROM (vw.window_end - vw.window_start)) AS interval_seconds,
			ds.trip_count, ds.driving_duration, ds.distance, ds.net_drive_energy,
			cs.session_count, cs.charging_duration, cs.battery_energy, cs.wall_energy, cs.charge_cost,
			ist.inactive_duration
		FROM version_windows vw
		LEFT JOIN LATERAL (
			SELECT
				COUNT(*) AS trip_count,
				COALESCE(SUM(d.duration_min) * 60, 0) AS driving_duration,
				COALESCE(SUM(d.distance), 0) AS distance,
				SUM(
					CASE
						WHEN d.distance > 0
							AND d.start_rated_range_km IS NOT NULL
							AND d.end_rated_range_km IS NOT NULL
							AND cars.efficiency IS NOT NULL
							AND (d.start_rated_range_km - d.end_rated_range_km) > 0
						THEN (d.start_rated_range_km - d.end_rated_range_km) * cars.efficiency
					END
				) AS net_drive_energy
			FROM drives d
			LEFT JOIN cars ON cars.id = d.car_id
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
				AND d.start_date >= vw.window_start AND d.start_date < vw.window_end
		) ds ON true
		LEFT JOIN LATERAL (
			SELECT
				COUNT(*) AS session_count,
				COALESCE(SUM(cp.duration_min) * 60, 0) AS charging_duration,
				COALESCE(SUM(cp.charge_energy_added), 0) AS battery_energy,
				COALESCE(SUM(GREATEST(COALESCE(cp.charge_energy_used, 0), COALESCE(cp.charge_energy_added, 0))), 0) AS wall_energy,
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
					))
				) FILTER (WHERE s.state IN ('asleep', 'offline')), 0) AS inactive_duration
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
		var versionStr sql.NullString
		var endDate, windowStart, windowEnd sql.NullTime
		var updateDuration, intervalSeconds sql.NullFloat64
		var daysSinceInt sql.NullInt64
		var tripCount sql.NullInt64
		var drivingDuration, distance, netEnergy sql.NullFloat64
		var sessionCount sql.NullInt64
		var chargingDuration, batteryEnergy, wallEnergy, chargeCost, inactiveDuration sql.NullFloat64
		var startDate time.Time
		if err := rows.Scan(
			&versionStr, &startDate, &endDate,
			&updateDuration, &daysSinceInt,
			&windowStart, &windowEnd, &intervalSeconds,
			&tripCount, &drivingDuration, &distance, &netEnergy,
			&sessionCount, &chargingDuration, &batteryEnergy, &wallEnergy, &chargeCost,
			&inactiveDuration,
		); err != nil {
			return response, 0, err
		}
		var v V2UpdateVersion
		if versionStr.Valid {
			v.Event.Version = versionStr.String
		}
		v.Event.StartedAt = startDate.Format(time.RFC3339)
		if endDate.Valid {
			s := endDate.Time.Format(time.RFC3339)
			v.Event.CompletedAt = &s
		}
		if updateDuration.Valid {
			v.Event.Duration = &updateDuration.Float64
		}
		if daysSinceInt.Valid {
			v.Event.DaysSincePrior = &daysSinceInt.Int64
		}
		if windowStart.Valid && windowEnd.Valid {
			window := &V2UpdateWindow{
				Start: windowStart.Time.Format(time.RFC3339),
				End:   windowEnd.Time.Format(time.RFC3339),
			}
			if intervalSeconds.Valid {
				window.Interval = &intervalSeconds.Float64
			}
			v.Window = window
		}

		metrics := &V2UpdateWindowMetrics{}
		hasMetrics := false
		if tripCount.Valid {
			metrics.Driving.TripCount = tripCount.Int64
			hasMetrics = true
		}
		if drivingDuration.Valid {
			metrics.Driving.Duration = drivingDuration.Float64
		}
		if distance.Valid {
			metrics.Driving.Distance = distance.Float64
		}
		if netEnergy.Valid {
			metrics.Driving.NetEnergy = &netEnergy.Float64
			if distance.Valid && distance.Float64 > 0 {
				cons := netEnergy.Float64 / distance.Float64 * 1000
				metrics.Driving.AvgConsumption = &cons
			}
		}
		if sessionCount.Valid {
			metrics.Charging.SessionCount = sessionCount.Int64
			hasMetrics = true
		}
		if chargingDuration.Valid {
			metrics.Charging.Duration = chargingDuration.Float64
		}
		if batteryEnergy.Valid {
			metrics.Charging.BatteryEnergy = &batteryEnergy.Float64
		}
		if wallEnergy.Valid {
			metrics.Charging.WallEnergy = &wallEnergy.Float64
		}
		if wallEnergy.Valid && wallEnergy.Float64 > 0 && batteryEnergy.Valid {
			eff := batteryEnergy.Float64 / wallEnergy.Float64 * 100
			metrics.Charging.Efficiency = &eff
		}
		if chargeCost.Valid {
			metrics.Charging.Cost = &chargeCost.Float64
		}
		if inactiveDuration.Valid {
			metrics.InactiveDuration = &inactiveDuration.Float64
			hasMetrics = true
		}
		if hasMetrics {
			v.Metrics = metrics
		}
		response.Versions = append(response.Versions, v)
	}
	if err := rows.Err(); err != nil {
		return response, 0, err
	}

	return response, response.UpdateCount, nil
}
