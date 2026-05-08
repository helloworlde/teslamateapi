package main

import (
	"context"
	"database/sql"
	"time"
)

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
		SELECT version, start_date, end_date,
			EXTRACT(EPOCH FROM (end_date - start_date)) / 60 AS duration_min
		FROM updates
		WHERE car_id = $1 AND start_date >= $2 AND start_date < $3
		ORDER BY start_date DESC`,
		carID, start.Time, end.Time,
	)
	if err != nil {
		return response, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var version V2UpdateVersion
		var versionStr sql.NullString
		var endDate sql.NullTime
		var durationMin sql.NullFloat64
		var startDate time.Time
		if err := rows.Scan(&versionStr, &startDate, &endDate, &durationMin); err != nil {
			return response, 0, err
		}
		if versionStr.Valid {
			version.Version = versionStr.String
		}
		version.StartedAt = startDate.Format(time.RFC3339)
		if endDate.Valid {
			s := endDate.Time.Format(time.RFC3339)
			version.CompletedAt = &s
		}
		if durationMin.Valid {
			version.DurationMin = &durationMin.Float64
		}
		response.Versions = append(response.Versions, version)
	}
	if err := rows.Err(); err != nil {
		return response, 0, err
	}

	return response, response.UpdateCount, nil
}
