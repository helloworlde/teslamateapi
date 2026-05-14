package v2

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// @name PostgresV2CostRepository
type PostgresV2CostRepository struct {
	db *sql.DB
}

func NewPostgresV2CostRepository(db *sql.DB) PostgresV2CostRepository {
	return PostgresV2CostRepository{db: db}
}

func (r PostgresV2CostRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2CostRepository) Cost(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) (V2CostResponse, V2CostStats, error) {
	var response V2CostResponse
	var stats V2CostStats
	var cost sql.NullFloat64
	var energyUsed sql.NullFloat64
	var minSessionCost sql.NullFloat64
	var medianSessionCost sql.NullFloat64
	var maxSessionCost sql.NullFloat64

	err := r.db.QueryRowContext(ctx, `
		SELECT
			SUM(cost) AS charging_cost,
			SUM(charge_energy_used) AS energy_used,
			COUNT(*) AS session_count,
			COUNT(cost) AS cost_rows,
			COUNT(charge_energy_used) AS energy_used_rows,
			MIN(cost) FILTER (WHERE cost IS NOT NULL AND cost > 0) AS min_session_cost,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY cost) FILTER (WHERE cost IS NOT NULL AND cost > 0) AS median_session_cost,
			MAX(cost) AS max_session_cost,
			COALESCE((SELECT SUM(distance) FROM drives WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND start_date < $3), 0) AS distance
		FROM charging_processes
		WHERE car_id = $1
			AND end_date IS NOT NULL
			AND start_date >= $2
			AND start_date < $3`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	).Scan(&cost, &energyUsed, &stats.SessionRows, &stats.CostRows, &stats.EnergyUsedRows, &minSessionCost, &medianSessionCost, &maxSessionCost, &response.Summary.Distance)
	if err != nil {
		return response, stats, err
	}

	if cost.Valid {
		response.Summary.ChargingCost = &cost.Float64
	}
	if energyUsed.Valid {
		response.Summary.EnergyUsed = &energyUsed.Float64
	}
	if cost.Valid && energyUsed.Valid && energyUsed.Float64 > 0 {
		response.Summary.CostPerEnergy = float64Ptr(cost.Float64 / energyUsed.Float64)
	}
	if cost.Valid && response.Summary.Distance > 0 {
		costPer100KM := cost.Float64 / response.Summary.Distance * 100
		response.Summary.CostPerDistance = &costPer100KM
	}
	if minSessionCost.Valid {
		response.Summary.MinSessionCost = &minSessionCost.Float64
	}
	if medianSessionCost.Valid {
		response.Summary.MedianSessionCost = &medianSessionCost.Float64
	}
	if maxSessionCost.Valid {
		response.Summary.MaxSessionCost = &maxSessionCost.Float64
	}

	periods, err := r.costByPeriod(ctx, carID, timeRange, groupBy)
	if err != nil {
		return response, stats, err
	}
	locations, err := r.costByLocation(ctx, carID, timeRange)
	if err != nil {
		return response, stats, err
	}
	response.CostByPeriod = periods
	response.CostByLocation = locations
	return response, stats, nil
}

func (r PostgresV2CostRepository) costByPeriod(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2CostPeriodItem, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			GREATEST(date_trunc('%s', charging_processes.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4), $2 AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
			SUM(charging_processes.cost) AS charging_cost,
			SUM(charging_processes.charge_energy_used) AS energy_used
		FROM charging_processes
		WHERE charging_processes.car_id = $1
			AND charging_processes.end_date IS NOT NULL
			AND charging_processes.start_date >= $2
			AND charging_processes.start_date < $3
		GROUP BY 1
		ORDER BY 1`, postgresDateTruncUnit(groupBy)),
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, timeRange.Timezone,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []V2CostPeriodItem
	location := timeRangeLocation(timeRange)
	for rows.Next() {
		var item V2CostPeriodItem
		var periodStart time.Time
		var cost sql.NullFloat64
		var energyUsed sql.NullFloat64
		if err := rows.Scan(&periodStart, &cost, &energyUsed); err != nil {
			return nil, err
		}
		item.PeriodStart = periodStart.In(location).Format(time.RFC3339)
		if cost.Valid {
			item.ChargingCost = &cost.Float64
		}
		if energyUsed.Valid {
			item.EnergyUsed = &energyUsed.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r PostgresV2CostRepository) costByLocation(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2CostLocationItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			COALESCE(geofences.name, CONCAT_WS(', ', COALESCE(addresses.name, nullif(CONCAT_WS(' ', addresses.road, addresses.house_number), '')), addresses.city), 'unknown') AS location_name,
			SUM(charging_processes.cost) AS charging_cost,
			SUM(charging_processes.charge_energy_used) AS energy_used
		FROM charging_processes
		LEFT JOIN geofences ON geofences.id = charging_processes.geofence_id
		LEFT JOIN addresses ON addresses.id = charging_processes.address_id
		WHERE charging_processes.car_id = $1
			AND charging_processes.end_date IS NOT NULL
			AND charging_processes.start_date >= $2
			AND charging_processes.start_date < $3
		GROUP BY 1
		ORDER BY charging_cost DESC NULLS LAST`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []V2CostLocationItem
	for rows.Next() {
		var item V2CostLocationItem
		var cost sql.NullFloat64
		var energyUsed sql.NullFloat64
		if err := rows.Scan(&item.LocationName, &cost, &energyUsed); err != nil {
			return nil, err
		}
		if cost.Valid {
			item.ChargingCost = &cost.Float64
		}
		if energyUsed.Valid {
			item.EnergyUsed = &energyUsed.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
