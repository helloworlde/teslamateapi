package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// @name PostgresV2LifecycleRepository
type PostgresV2LifecycleRepository struct {
	db *sql.DB
}

func NewPostgresV2LifecycleRepository(db *sql.DB) PostgresV2LifecycleRepository {
	return PostgresV2LifecycleRepository{db: db}
}

func (r PostgresV2LifecycleRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

// Lifecycle returns cumulative stats from the first recorded event up to asOf (inclusive).
func (r PostgresV2LifecycleRepository) Lifecycle(ctx context.Context, carID int64, asOf time.Time) (V2LifecycleResponse, error) {
	var response V2LifecycleResponse

	var firstDate, lastDate sql.NullTime
	var distanceKM sql.NullFloat64
	var driveCount sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			MIN(d.start_date) as first_date,
			MAX(d.end_date) as last_date,
			COUNT(*) as drive_count,
			COALESCE(SUM(d.distance), 0) as distance_km
		FROM drives d WHERE d.car_id = $1 AND d.end_date IS NOT NULL AND d.end_date <= $2`,
		carID, asOf,
	).Scan(&firstDate, &lastDate, &driveCount, &distanceKM)
	if err != nil {
		return response, err
	}

	if driveCount.Valid {
		response.DriveCount = driveCount.Int64
	}
	if distanceKM.Valid {
		response.DistanceKM = distanceKM.Float64
	}
	if firstDate.Valid {
		s := firstDate.Time.Format(time.RFC3339)
		response.FirstRecordedAt = &s
	}
	if lastDate.Valid {
		s := lastDate.Time.Format(time.RFC3339)
		response.LastRecordedAt = &s
	}

	if firstDate.Valid && lastDate.Valid {
		days := int64(lastDate.Time.Sub(firstDate.Time).Hours()/24) + 1
		response.RecordedDays = days
		if days > 0 && response.DistanceKM > 0 {
			daily := response.DistanceKM / float64(days)
			monthly := daily * 30.44
			response.AvgDailyDistanceKM = &daily
			response.AvgMonthlyDistanceKM = &monthly
		}
	}

	// Charging stats
	var sessionCount sql.NullInt64
	var energyAdded, energyUsed, cost sql.NullFloat64
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as session_count,
			COALESCE(SUM(charge_energy_added), 0) as energy_added_kwh,
			COALESCE(SUM(GREATEST(COALESCE(charge_energy_used, 0), COALESCE(charge_energy_added, 0))), 0) as energy_used_kwh,
			COALESCE(SUM(cost), 0) as cost
		FROM charging_processes WHERE car_id = $1 AND end_date IS NOT NULL AND end_date <= $2`,
		carID, asOf,
	).Scan(&sessionCount, &energyAdded, &energyUsed, &cost)
	if err != nil {
		return response, err
	}
	if sessionCount.Valid {
		response.ChargingSessionCount = sessionCount.Int64
	}
	if energyAdded.Valid {
		response.EnergyAddedKWh = energyAdded.Float64
	}
	if energyUsed.Valid {
		response.EnergyUsedKWh = energyUsed.Float64
	}
	if cost.Valid {
		response.ChargingCost = cost.Float64
	}

	if response.DistanceKM > 0 && response.ChargingCost > 0 {
		costPer100 := response.ChargingCost / response.DistanceKM * 100
		response.CostPer100KM = &costPer100
	}

	// Avg consumption from drives
	var avgConsumption sql.NullFloat64
	err = r.db.QueryRowContext(ctx, `
		SELECT AVG(
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
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.end_date <= $2`,
		carID, asOf,
	).Scan(&avgConsumption)
	if err != nil {
		return response, err
	}
	if avgConsumption.Valid {
		response.AvgConsumptionWhPerKM = &avgConsumption.Float64
	}

	// Updates
	var updateCount sql.NullInt64
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM updates WHERE car_id = $1 AND start_date <= $2`, carID, asOf).Scan(&updateCount)
	if err != nil {
		return response, err
	}
	if updateCount.Valid {
		response.UpdateCount = updateCount.Int64
	}

	return response, nil
}

// Timeline returns cursor-paginated unified event list.
// If before is set: return events with start_date < before, order DESC.
// If after is set: return events with start_date > after, order ASC.
// Otherwise: return most recent events DESC.
// hasMore=true means there are additional events beyond the returned page.
// nextCursor is the start_time of the boundary event for the next page call.
func (r PostgresV2LifecycleRepository) Timeline(ctx context.Context, carID int64, eventTypes []string, limit int, before, after *time.Time) ([]V2TimelineEvent, bool, *time.Time, error) {
	orderDir := "DESC"
	cursorCond := ""
	var cursorArg interface{} = time.Time{}
	hasCursor := false

	if before != nil {
		cursorCond = "AND start_date < $5"
		cursorArg = *before
		orderDir = "DESC"
		hasCursor = true
	} else if after != nil {
		cursorCond = "AND start_date > $5"
		cursorArg = *after
		orderDir = "ASC"
		hasCursor = true
	}

	typeFilter := len(eventTypes) == 0
	includeType := func(t string) bool {
		if typeFilter {
			return true
		}
		for _, et := range eventTypes {
			if et == t {
				return true
			}
		}
		return false
	}

	var unions []string
	if includeType("drive") {
		unions = append(unions, fmt.Sprintf(`
			(SELECT 'drive' as type, id, start_date, end_date,
				COALESCE(distance::text, '') as title_metric
			FROM drives WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND start_date < $3 %s
			ORDER BY start_date %s LIMIT $4)`, cursorCond, orderDir))
	}
	if includeType("charging") {
		unions = append(unions, fmt.Sprintf(`
			(SELECT 'charging' as type, id, start_date, end_date,
				COALESCE(charge_energy_added::text, '') as title_metric
			FROM charging_processes WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND start_date < $3 %s
			ORDER BY start_date %s LIMIT $4)`, cursorCond, orderDir))
	}
	if includeType("update") {
		unions = append(unions, fmt.Sprintf(`
			(SELECT 'update' as type, id, start_date, end_date,
				COALESCE(version, '') as title_metric
			FROM updates WHERE car_id = $1 AND start_date >= $2 AND start_date < $3 %s
			ORDER BY start_date %s LIMIT $4)`, cursorCond, orderDir))
	}

	if len(unions) == 0 {
		return []V2TimelineEvent{}, false, nil, nil
	}

	// Fetch limit+1 to detect has_more
	fetchLimit := limit + 1
	query := fmt.Sprintf(`
		SELECT type, id, start_date, end_date, title_metric
		FROM (
			%s
		) t
		ORDER BY start_date %s
		LIMIT $4`,
		joinStrings(unions, " UNION ALL "),
		orderDir,
	)

	// Epoch sentinel for "no lower bound" (start_date bound)
	epochStart := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	epochEnd := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)

	var rows *sql.Rows
	var err error
	if hasCursor {
		rows, err = r.db.QueryContext(ctx, query, carID, epochStart, epochEnd, fetchLimit, cursorArg)
	} else {
		rows, err = r.db.QueryContext(ctx, query, carID, epochStart, epochEnd, fetchLimit)
	}
	if err != nil {
		return nil, false, nil, err
	}
	defer rows.Close()

	var events []V2TimelineEvent
	for rows.Next() {
		var evType, titleMetric string
		var id int64
		var startDate time.Time
		var endDate sql.NullTime
		if err := rows.Scan(&evType, &id, &startDate, &endDate, &titleMetric); err != nil {
			return nil, false, nil, err
		}
		ev := V2TimelineEvent{
			Type:      evType,
			ID:        id,
			StartTime: startDate.Format(time.RFC3339),
		}
		if endDate.Valid {
			s := endDate.Time.Format(time.RFC3339)
			ev.EndTime = &s
		}
		switch evType {
		case "drive":
			ev.Title = fmt.Sprintf("Drive %s km", titleMetric)
		case "charging":
			ev.Title = fmt.Sprintf("Charging %s kWh", titleMetric)
		case "update":
			ev.Title = fmt.Sprintf("Update to %s", titleMetric)
		default:
			ev.Title = evType
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, false, nil, err
	}

	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}

	var nextCursor *time.Time
	if hasMore && len(events) > 0 {
		last := events[len(events)-1]
		t, _ := time.Parse(time.RFC3339, last.StartTime)
		nextCursor = &t
	}

	return events, hasMore, nextCursor, nil
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
