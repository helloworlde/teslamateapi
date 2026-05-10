package main

import (
	"context"
	"database/sql"
	"fmt"
)

// @name PostgresV2LocationRepository
type PostgresV2LocationRepository struct {
	db *sql.DB
}

func NewPostgresV2LocationRepository(db *sql.DB) PostgresV2LocationRepository {
	return PostgresV2LocationRepository{db: db}
}

func (r PostgresV2LocationRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2LocationRepository) Locations(ctx context.Context, carID int64, timeRange V2TimeRange, sort string) ([]V2LocationAnalyticsItem, V2LocationStats, error) {
	orderBy, err := v2LocationOrderBy(sort)
	if err != nil {
		return nil, V2LocationStats{}, err
	}

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		WITH drive_starts AS (
			SELECT start_geofence_id AS geofence_id, start_address_id AS address_id, COUNT(*) AS drive_start_count
			FROM drives
			WHERE car_id = $1
				AND end_date IS NOT NULL
				AND start_date >= $2
				AND start_date < $3
			GROUP BY 1, 2
		),
		drive_ends AS (
			SELECT end_geofence_id AS geofence_id, end_address_id AS address_id, COUNT(*) AS drive_end_count
			FROM drives
			WHERE car_id = $1
				AND end_date IS NOT NULL
				AND start_date >= $2
				AND start_date < $3
			GROUP BY 1, 2
		),
		charges AS (
			SELECT
				geofence_id,
				address_id,
				COUNT(*) AS charging_session_count,
				COALESCE(SUM(charge_energy_added), 0) AS energy_added_kwh,
				SUM(cost) AS charging_cost
			FROM charging_processes
			WHERE car_id = $1
				AND end_date IS NOT NULL
				AND start_date >= $2
				AND start_date < $3
			GROUP BY 1, 2
		),
		parking_events AS (
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
		parking_sessions AS (
			SELECT
				end_date AS parking_start,
				LEAD(start_date) OVER (ORDER BY start_date, end_date) AS parking_end,
				geofence_id,
				address_id
			FROM parking_events
		),
		parking AS (
			SELECT
				sessions.geofence_id,
				sessions.address_id,
				COUNT(*) AS parking_session_count,
				COALESCE(SUM(EXTRACT(EPOCH FROM (LEAST(sessions.parking_end, $3::timestamptz) - GREATEST(sessions.parking_start, $2::timestamptz))) / 60), 0) AS parking_duration_min,
				COALESCE(SUM(GREATEST(COALESCE(start_pos.battery_level, 0) - COALESCE(end_pos.battery_level, 0), 0)), 0) AS vampire_drain_percent
			FROM parking_sessions sessions
			LEFT JOIN LATERAL (
				SELECT battery_level::float
				FROM positions
				WHERE positions.car_id = $1
					AND positions.date >= GREATEST(sessions.parking_start, $2::timestamptz)
					AND positions.date <= LEAST(sessions.parking_end, $3::timestamptz)
					AND positions.battery_level IS NOT NULL
				ORDER BY positions.date ASC
				LIMIT 1
			) start_pos ON true
			LEFT JOIN LATERAL (
				SELECT battery_level::float
				FROM positions
				WHERE positions.car_id = $1
					AND positions.date >= GREATEST(sessions.parking_start, $2::timestamptz)
					AND positions.date <= LEAST(sessions.parking_end, $3::timestamptz)
					AND positions.battery_level IS NOT NULL
				ORDER BY positions.date DESC
				LIMIT 1
			) end_pos ON true
			WHERE sessions.parking_end IS NOT NULL
				AND sessions.parking_start < $3::timestamptz
				AND sessions.parking_end > $2::timestamptz
				AND sessions.parking_end > sessions.parking_start
			GROUP BY 1, 2
		),
		location_keys AS (
			SELECT geofence_id, address_id FROM drive_starts
			UNION
			SELECT geofence_id, address_id FROM drive_ends
			UNION
			SELECT geofence_id, address_id FROM charges
			UNION
			SELECT geofence_id, address_id FROM parking
		)
		SELECT
			COALESCE(geofences.name, CONCAT_WS(', ', COALESCE(addresses.name, nullif(CONCAT_WS(' ', addresses.road, addresses.house_number), '')), addresses.city), 'unknown') AS location_name,
			location_keys.geofence_id,
			location_keys.address_id,
			COALESCE(drive_starts.drive_start_count, 0) AS drive_start_count,
			COALESCE(drive_ends.drive_end_count, 0) AS drive_end_count,
			COALESCE(charges.charging_session_count, 0) AS charging_session_count,
			COALESCE(parking.parking_session_count, 0) AS parking_session_count,
			COALESCE(parking.parking_duration_min, 0) AS parking_duration_min,
			COALESCE(charges.energy_added_kwh, 0) AS energy_added_kwh,
			charges.charging_cost,
			COALESCE(parking.vampire_drain_percent, 0) AS vampire_drain_percent
		FROM location_keys
		LEFT JOIN drive_starts ON drive_starts.geofence_id IS NOT DISTINCT FROM location_keys.geofence_id
			AND drive_starts.address_id IS NOT DISTINCT FROM location_keys.address_id
		LEFT JOIN drive_ends ON drive_ends.geofence_id IS NOT DISTINCT FROM location_keys.geofence_id
			AND drive_ends.address_id IS NOT DISTINCT FROM location_keys.address_id
		LEFT JOIN charges ON charges.geofence_id IS NOT DISTINCT FROM location_keys.geofence_id
			AND charges.address_id IS NOT DISTINCT FROM location_keys.address_id
		LEFT JOIN parking ON parking.geofence_id IS NOT DISTINCT FROM location_keys.geofence_id
			AND parking.address_id IS NOT DISTINCT FROM location_keys.address_id
		LEFT JOIN geofences ON geofences.id = location_keys.geofence_id
		LEFT JOIN addresses ON addresses.id = location_keys.address_id
		ORDER BY %s`, orderBy),
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	)
	if err != nil {
		return nil, V2LocationStats{}, err
	}
	defer rows.Close()

	var items []V2LocationAnalyticsItem
	var stats V2LocationStats
	for rows.Next() {
		var item V2LocationAnalyticsItem
		var geofenceID sql.NullInt64
		var addressID sql.NullInt64
		var chargingCost sql.NullFloat64
		if err := rows.Scan(
			&item.LocationName,
			&geofenceID,
			&addressID,
			&item.DriveStartCount,
			&item.DriveEndCount,
			&item.ChargingSessionCount,
			&item.ParkingSessionCount,
			&item.ParkingDurationMin,
			&item.EnergyAddedKWh,
			&chargingCost,
			&item.VampireDrainPercent,
		); err != nil {
			return nil, stats, err
		}
		if geofenceID.Valid {
			item.GeofenceID = &geofenceID.Int64
		}
		if addressID.Valid {
			item.AddressID = &addressID.Int64
		}
		if chargingCost.Valid {
			item.ChargingCost = &chargingCost.Float64
		}
		stats.DriveStartRows += item.DriveStartCount
		stats.DriveEndRows += item.DriveEndCount
		stats.ChargingRows += item.ChargingSessionCount
		stats.ParkingRows += item.ParkingSessionCount
		items = append(items, item)
	}
	return items, stats, rows.Err()
}

func v2LocationOrderBy(sort string) (string, error) {
	switch sort {
	case "", "parking_duration_desc":
		return "parking_duration_min DESC, location_name ASC", nil
	case "drive_start_count_desc":
		return "drive_start_count DESC, location_name ASC", nil
	case "drive_end_count_desc":
		return "drive_end_count DESC, location_name ASC", nil
	case "charging_session_count_desc":
		return "charging_session_count DESC, location_name ASC", nil
	case "charging_cost_desc":
		return "charging_cost DESC NULLS LAST, location_name ASC", nil
	default:
		return "", errV2InvalidLocationSort
	}
}
