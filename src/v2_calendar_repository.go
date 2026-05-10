package main

import (
	"context"
	"database/sql"
	"time"
)

// @name PostgresV2CalendarRepository
type PostgresV2CalendarRepository struct {
	db *sql.DB
}

func NewPostgresV2CalendarRepository(db *sql.DB) PostgresV2CalendarRepository {
	return PostgresV2CalendarRepository{db: db}
}

func (r PostgresV2CalendarRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2CalendarRepository) DailyDriving(ctx context.Context, carID int64, start, end timeBound, location *time.Location) (map[string]V2CalendarDay, error) {
	tzName := location.String()
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			(start_date AT TIME ZONE 'UTC' AT TIME ZONE $4)::date AS local_date,
			COUNT(*) AS drive_count,
			COALESCE(SUM(distance), 0) AS distance_km,
			COALESCE(SUM(duration_min), 0) AS duration_min
		FROM drives
		WHERE car_id = $1 AND end_date IS NOT NULL
			AND start_date >= $2 AND start_date < $3
		GROUP BY local_date`,
		carID, start.Time, end.Time, tzName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]V2CalendarDay)
	for rows.Next() {
		var dateStr string
		var day V2CalendarDay
		if err := rows.Scan(&dateStr, &day.DriveCount, &day.DistanceKM, &day.DriveDurationMin); err != nil {
			return nil, err
		}
		day.Date = dateStr
		result[dateStr] = day
	}
	return result, rows.Err()
}

func (r PostgresV2CalendarRepository) DailyCharging(ctx context.Context, carID int64, start, end timeBound, location *time.Location) (map[string]V2CalendarDay, error) {
	tzName := location.String()
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			(start_date AT TIME ZONE 'UTC' AT TIME ZONE $4)::date AS local_date,
			COUNT(*) AS session_count,
			COALESCE(SUM(charge_energy_added), 0) AS energy_added_kwh,
			COALESCE(SUM(cost), 0) AS charging_cost
		FROM charging_processes
		WHERE car_id = $1 AND end_date IS NOT NULL
			AND start_date >= $2 AND start_date < $3
		GROUP BY local_date`,
		carID, start.Time, end.Time, tzName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]V2CalendarDay)
	for rows.Next() {
		var dateStr string
		var day V2CalendarDay
		if err := rows.Scan(&dateStr, &day.ChargingSessionCount, &day.EnergyAddedKWh, &day.ChargingCost); err != nil {
			return nil, err
		}
		day.Date = dateStr
		result[dateStr] = day
	}
	return result, rows.Err()
}

func (r PostgresV2CalendarRepository) DailyUpdates(ctx context.Context, carID int64, start, end timeBound, location *time.Location) (map[string]int64, error) {
	tzName := location.String()
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			(start_date AT TIME ZONE 'UTC' AT TIME ZONE $4)::date AS local_date,
			COUNT(*) AS update_count
		FROM updates
		WHERE car_id = $1
			AND start_date >= $2 AND start_date < $3
		GROUP BY local_date`,
		carID, start.Time, end.Time, tzName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var dateStr string
		var count int64
		if err := rows.Scan(&dateStr, &count); err != nil {
			return nil, err
		}
		result[dateStr] = count
	}
	return result, rows.Err()
}
