package v2

import (
	"context"
	"database/sql"
)

// @name PostgresV2ParkingRepository
type PostgresV2ParkingRepository struct {
	db *sql.DB
}

func NewPostgresV2ParkingRepository(db *sql.DB) PostgresV2ParkingRepository {
	return PostgresV2ParkingRepository{db: db}
}

func (r PostgresV2ParkingRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2ParkingRepository) Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2ParkingSummary, V2ParkingStats, error) {
	var summary V2ParkingSummary
	var stats V2ParkingStats
	if err := r.loadStateSummary(ctx, carID, timeRange, &summary, &stats); err != nil {
		return summary, stats, err
	}
	if err := r.loadSessionSummary(ctx, carID, timeRange, &summary, &stats); err != nil {
		return summary, stats, err
	}
	if err := r.loadDrainSummary(ctx, carID, timeRange, &summary, &stats); err != nil {
		return summary, stats, err
	}
	return summary, stats, nil
}

func (r PostgresV2ParkingRepository) StateBreakdown(ctx context.Context, carID int64, timeRange V2TimeRange) (V2ParkingStatesResponse, V2ParkingStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			COALESCE(state::text, 'unknown') AS state,
			COUNT(*) AS transition_count,
			COALESCE(SUM(EXTRACT(EPOCH FROM (LEAST(COALESCE(end_date, $3::timestamptz), $3::timestamptz) - GREATEST(start_date, $2::timestamptz)))), 0) AS duration_seconds
		FROM states
		WHERE car_id = $1
			AND start_date < $3::timestamptz
			AND COALESCE(end_date, $3::timestamptz) > $2::timestamptz
		GROUP BY 1
		ORDER BY duration_seconds DESC`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	)
	if err != nil {
		return V2ParkingStatesResponse{}, V2ParkingStats{}, err
	}
	defer rows.Close()

	var response V2ParkingStatesResponse
	var stats V2ParkingStats
	for rows.Next() {
		var item V2ParkingStateItem
		if err := rows.Scan(&item.State, &item.TransitionCount, &item.Duration); err != nil {
			return response, stats, err
		}
		response.TotalDuration += item.Duration
		response.StateTransitionCount += item.TransitionCount
		stats.StateRows += item.TransitionCount
		response.Items = append(response.Items, item)
	}
	if err := rows.Err(); err != nil {
		return response, stats, err
	}
	for i := range response.Items {
		if response.TotalDuration > 0 {
			response.Items[i].Percent = response.Items[i].Duration / response.TotalDuration * 100
		}
	}
	return response, stats, nil
}

func (r PostgresV2ParkingRepository) Locations(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2ParkingLocationItem, V2ParkingStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH events AS (
			SELECT start_date, end_date, end_geofence_id AS geofence_id, end_address_id AS address_id
			FROM drives
			WHERE car_id = $1
				AND end_date IS NOT NULL
				AND start_date < $3::timestamptz
				AND end_date > ($2::timestamptz - INTERVAL '90 days')
			UNION ALL
			SELECT start_date, end_date, geofence_id, address_id
			FROM charging_processes
			WHERE car_id = $1
				AND end_date IS NOT NULL
				AND start_date < $3::timestamptz
				AND end_date > ($2::timestamptz - INTERVAL '90 days')
		),
		sessions AS (
			SELECT
				end_date AS parking_start,
				LEAD(start_date) OVER (ORDER BY start_date, end_date) AS parking_end,
				geofence_id,
				address_id
			FROM events
		)
		SELECT
			COALESCE(geofences.name, CONCAT_WS(', ', COALESCE(addresses.name, nullif(CONCAT_WS(' ', addresses.road, addresses.house_number), '')), addresses.city), 'unknown') AS location_name,
			sessions.geofence_id,
			sessions.address_id,
			COUNT(*) AS parking_session_count,
			COALESCE(SUM(EXTRACT(EPOCH FROM (LEAST(parking_end, $3::timestamptz) - GREATEST(parking_start, $2::timestamptz)))), 0) AS parked_duration_seconds
		FROM sessions
		LEFT JOIN geofences ON geofences.id = sessions.geofence_id
		LEFT JOIN addresses ON addresses.id = sessions.address_id
		WHERE parking_end IS NOT NULL
			AND parking_start < $3::timestamptz
			AND parking_end > $2::timestamptz
			AND parking_end > parking_start
		GROUP BY 1, 2, 3
		ORDER BY parked_duration_seconds DESC`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	)
	if err != nil {
		return nil, V2ParkingStats{}, err
	}
	defer rows.Close()

	var items []V2ParkingLocationItem
	var stats V2ParkingStats
	for rows.Next() {
		var item V2ParkingLocationItem
		var geofenceID sql.NullInt64
		var addressID sql.NullInt64
		if err := rows.Scan(&item.LocationName, &geofenceID, &addressID, &item.ParkingSessionCount, &item.ParkedDuration); err != nil {
			return nil, stats, err
		}
		if geofenceID.Valid {
			item.GeofenceID = &geofenceID.Int64
		}
		if addressID.Valid {
			item.AddressID = &addressID.Int64
		}
		if item.ParkingSessionCount > 0 {
			item.AvgParkedDuration = float64Ptr(item.ParkedDuration / float64(item.ParkingSessionCount))
		}
		stats.SessionRows += item.ParkingSessionCount
		items = append(items, item)
	}
	return items, stats, rows.Err()
}

func (r PostgresV2ParkingRepository) loadStateSummary(ctx context.Context, carID int64, timeRange V2TimeRange, summary *V2ParkingSummary, stats *V2ParkingStats) error {
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
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	).Scan(&stats.StateRows, &summary.ParkedDuration, &summary.AsleepDuration, &summary.OnlineDuration, &summary.OfflineDuration)
	if err != nil {
		return err
	}
	summary.StateTransitionCount = stats.StateRows
	return nil
}

func (r PostgresV2ParkingRepository) loadSessionSummary(ctx context.Context, carID int64, timeRange V2TimeRange, summary *V2ParkingSummary, stats *V2ParkingStats) error {
	var avg sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		WITH events AS (
			SELECT start_date, end_date
			FROM drives
			WHERE car_id = $1
				AND end_date IS NOT NULL
				AND start_date < $3::timestamptz
				AND end_date > ($2::timestamptz - INTERVAL '90 days')
			UNION ALL
			SELECT start_date, end_date
			FROM charging_processes
			WHERE car_id = $1
				AND end_date IS NOT NULL
				AND start_date < $3::timestamptz
				AND end_date > ($2::timestamptz - INTERVAL '90 days')
		),
		sessions AS (
			SELECT
				end_date AS parking_start,
				LEAD(start_date) OVER (ORDER BY start_date, end_date) AS parking_end
			FROM events
		)
		SELECT
			COUNT(*) AS parking_session_count,
			AVG(EXTRACT(EPOCH FROM (LEAST(parking_end, $3::timestamptz) - GREATEST(parking_start, $2::timestamptz)))) AS avg_parked_duration_seconds
		FROM sessions
		WHERE parking_end IS NOT NULL
			AND parking_start < $3::timestamptz
			AND parking_end > $2::timestamptz
			AND parking_end > parking_start`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	).Scan(&summary.ParkingSessionCount, &avg)
	if err != nil {
		return err
	}
	if avg.Valid {
		summary.AvgParkedDuration = &avg.Float64
	}
	stats.SessionRows = summary.ParkingSessionCount
	return nil
}

func (r PostgresV2ParkingRepository) loadDrainSummary(ctx context.Context, carID int64, timeRange V2TimeRange, summary *V2ParkingSummary, stats *V2ParkingStats) error {
	startBattery, startRange, ok, err := r.parkedPositionSnapshot(ctx, carID, timeRange, true)
	if err != nil || !ok {
		return err
	}
	endBattery, endRange, ok, err := r.parkedPositionSnapshot(ctx, carID, timeRange, false)
	if err != nil || !ok {
		return err
	}
	stats.PositionRows = 2
	if startBattery.Valid && endBattery.Valid && startBattery.Float64 > endBattery.Float64 {
		drain := startBattery.Float64 - endBattery.Float64
		summary.VampireDrainPercent = &drain
		days := timeRange.End.Sub(timeRange.Start).Hours() / 24
		if days > 0 {
			summary.AvgDrainPercentPerDay = float64Ptr(drain / days)
		}
	}
	if startRange.Valid && endRange.Valid && startRange.Float64 > endRange.Float64 {
		rangeLoss := startRange.Float64 - endRange.Float64
		var efficiency sql.NullFloat64
		if err := r.db.QueryRowContext(ctx, `SELECT efficiency FROM cars WHERE id = $1`, carID).Scan(&efficiency); err != nil {
			return err
		}
		if efficiency.Valid {
			summary.EstimatedVampireDrain = float64Ptr(rangeLoss * efficiency.Float64)
		}
	}
	return nil
}

func (r PostgresV2ParkingRepository) parkedPositionSnapshot(ctx context.Context, carID int64, timeRange V2TimeRange, ascending bool) (sql.NullFloat64, sql.NullFloat64, bool, error) {
	orderDirection := "ASC"
	if !ascending {
		orderDirection = "DESC"
	}
	var battery sql.NullFloat64
	var ratedRange sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT battery_level::float, rated_battery_range_km
		FROM positions
		WHERE car_id = $1
			AND date >= $2::timestamptz
			AND date < $3::timestamptz
			AND battery_level IS NOT NULL
			AND NOT EXISTS (
				SELECT 1 FROM drives
				WHERE drives.car_id = positions.car_id
					AND drives.end_date IS NOT NULL
					AND positions.date >= drives.start_date
					AND positions.date <= drives.end_date
			)
			AND NOT EXISTS (
				SELECT 1 FROM charging_processes
				WHERE charging_processes.car_id = positions.car_id
					AND charging_processes.end_date IS NOT NULL
					AND positions.date >= charging_processes.start_date
					AND positions.date <= charging_processes.end_date
			)
		ORDER BY date `+orderDirection+`
		LIMIT 1`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	).Scan(&battery, &ratedRange)
	if err == sql.ErrNoRows {
		return battery, ratedRange, false, nil
	}
	if err != nil {
		return battery, ratedRange, false, err
	}
	return battery, ratedRange, true, nil
}
