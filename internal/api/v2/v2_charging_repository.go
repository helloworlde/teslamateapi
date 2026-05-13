package v2

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// @name PostgresV2ChargingRepository
type PostgresV2ChargingRepository struct {
	db *sql.DB
}

func NewPostgresV2ChargingRepository(db *sql.DB) PostgresV2ChargingRepository {
	return PostgresV2ChargingRepository{db: db}
}

func (r PostgresV2ChargingRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2ChargingRepository) Summary(ctx context.Context, carID int64, start timeBound, end timeBound) (V2ChargingSummary, V2ChargingStats, error) {
	var summary V2ChargingSummary
	var stats V2ChargingStats
	var energyUsed sql.NullFloat64
	var cost sql.NullFloat64
	var startBattery sql.NullFloat64
	var endBattery sql.NullFloat64
	var avgPower sql.NullFloat64
	var maxPower sql.NullFloat64
	var longestSession sql.NullFloat64
	var avgEnergyAdded sql.NullFloat64
	var largestSession sql.NullFloat64
	var avgCost sql.NullFloat64
	var maxCost sql.NullFloat64

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS session_count,
			COALESCE(SUM(charge_energy_added), 0) AS energy_added,
			COALESCE(SUM(GREATEST(COALESCE(charge_energy_used, 0), COALESCE(charge_energy_added, 0))), 0) AS energy_used,
			COALESCE(SUM(duration_min) * 60, 0) AS duration_seconds,
			MAX(duration_min) * 60 AS longest_session_duration,
			AVG(charge_energy_added) AS avg_energy_added,
			MAX(charge_energy_added) AS largest_session,
			SUM(charge_energy_used) AS energy_used_raw,
			SUM(cost) AS cost,
			AVG(cost) AS avg_cost,
			MAX(cost) AS max_cost,
			AVG(start_battery_level) AS start_battery_avg,
			AVG(end_battery_level) AS end_battery_avg,
			COUNT(charge_energy_used) AS energy_used_rows,
			COUNT(cost) AS cost_rows,
			COUNT(*) FILTER (WHERE COALESCE(max_charge.charger_power, 0) < 20) AS ac_session_count,
			COUNT(*) FILTER (WHERE COALESCE(max_charge.charger_power, 0) >= 20) AS dc_session_count,
			COALESCE(SUM(charge_energy_added) FILTER (WHERE COALESCE(max_charge.charger_power, 0) < 20), 0) AS ac_energy,
			COALESCE(SUM(charge_energy_added) FILTER (WHERE COALESCE(max_charge.charger_power, 0) >= 20), 0) AS dc_energy,
			AVG(NULLIF(max_charge.charger_power, 0)) AS avg_power,
			MAX(NULLIF(max_charge.charger_power, 0)) AS max_power,
			COUNT(NULLIF(max_charge.charger_power, 0)) AS power_rows
		FROM charging_processes
		LEFT JOIN LATERAL (
			SELECT MAX(charger_power) AS charger_power
			FROM charges
			WHERE charges.charging_process_id = charging_processes.id
		) max_charge ON true
		WHERE charging_processes.car_id = $1
			AND charging_processes.end_date IS NOT NULL
			AND charging_processes.start_date >= $2
			AND charging_processes.start_date < $3`,
		carID, start.Time, end.Time,
	).Scan(
		&summary.SessionCount,
		&summary.EnergyAdded,
		&summary.EnergyUsed,
		&summary.Duration,
		&longestSession,
		&avgEnergyAdded,
		&largestSession,
		&energyUsed,
		&cost,
		&avgCost,
		&maxCost,
		&startBattery,
		&endBattery,
		&stats.EnergyUsedRows,
		&stats.CostRows,
		&summary.ACSessionCount,
		&summary.DCSessionCount,
		&summary.ACEnergy,
		&summary.DCEnergy,
		&avgPower,
		&maxPower,
		&stats.PowerRows,
	)
	if err != nil {
		return summary, stats, err
	}
	stats.SessionRows = summary.SessionCount
	if summary.SessionCount > 0 {
		summary.AvgDuration = float64Ptr(summary.Duration / float64(summary.SessionCount))
	}
	if longestSession.Valid {
		summary.LongestSessionDuration = &longestSession.Float64
	}
	if avgEnergyAdded.Valid {
		summary.AvgEnergyAdded = &avgEnergyAdded.Float64
	}
	if largestSession.Valid {
		summary.LargestSession = &largestSession.Float64
	}
	if energyUsed.Valid && energyUsed.Float64 > 0 {
		summary.ChargingEfficiency = float64Ptr(summary.EnergyAdded / energyUsed.Float64 * 100)
	}
	if cost.Valid {
		summary.Cost = cost.Float64
		if summary.EnergyAdded > 0 {
			summary.AvgCostPerEnergy = float64Ptr(cost.Float64 / summary.EnergyAdded)
		}
	}
	if avgCost.Valid {
		summary.AvgCost = &avgCost.Float64
	}
	if maxCost.Valid {
		summary.MaxCost = &maxCost.Float64
	}
	if startBattery.Valid {
		summary.StartBatteryAvg = &startBattery.Float64
	}
	if endBattery.Valid {
		summary.EndBatteryAvg = &endBattery.Float64
	}
	if avgPower.Valid {
		summary.AvgPower = &avgPower.Float64
	}
	if maxPower.Valid {
		summary.MaxPower = &maxPower.Float64
	}
	return summary, stats, nil
}

func (r PostgresV2ChargingRepository) Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2ChargingTimeseriesItem, V2ChargingStats, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			GREATEST(date_trunc('%s', charging_processes.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4), $2 AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
			COUNT(*) AS session_count,
			COALESCE(SUM(charge_energy_added), 0) AS energy_added,
			SUM(charge_energy_used) AS energy_used,
			COALESCE(SUM(duration_min) * 60, 0) AS duration_seconds,
			SUM(cost) AS cost,
			AVG(NULLIF(max_charge.charger_power, 0)) AS avg_power,
			COUNT(charge_energy_used) AS energy_used_rows,
			COUNT(cost) AS cost_rows,
			COUNT(NULLIF(max_charge.charger_power, 0)) AS power_rows
		FROM charging_processes
		LEFT JOIN LATERAL (
			SELECT MAX(charger_power) AS charger_power
			FROM charges
			WHERE charges.charging_process_id = charging_processes.id
		) max_charge ON true
		WHERE charging_processes.car_id = $1
			AND charging_processes.end_date IS NOT NULL
			AND charging_processes.start_date >= $2
			AND charging_processes.start_date < $3
		GROUP BY 1
		ORDER BY 1`, postgresDateTruncUnit(groupBy)),
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, timeRange.Timezone,
	)
	if err != nil {
		return nil, V2ChargingStats{}, err
	}
	defer rows.Close()

	var items []V2ChargingTimeseriesItem
	var stats V2ChargingStats
	location := timeRangeLocation(timeRange)
	for rows.Next() {
		var periodStart time.Time
		var item V2ChargingTimeseriesItem
		var energyUsed sql.NullFloat64
		var cost sql.NullFloat64
		var avgPower sql.NullFloat64
		var energyRows, costRows, powerRows int64
		if err := rows.Scan(&periodStart, &item.SessionCount, &item.EnergyAdded, &energyUsed, &item.Duration, &cost, &avgPower, &energyRows, &costRows, &powerRows); err != nil {
			return nil, stats, err
		}
		item.PeriodStart = periodStart.In(location).Format(time.RFC3339)
		if energyUsed.Valid {
			item.EnergyUsed = &energyUsed.Float64
		}
		if cost.Valid {
			item.Cost = &cost.Float64
		}
		if avgPower.Valid {
			item.AvgPower = &avgPower.Float64
		}
		stats.SessionRows += item.SessionCount
		stats.EnergyUsedRows += energyRows
		stats.CostRows += costRows
		stats.PowerRows += powerRows
		items = append(items, item)
	}
	return items, stats, rows.Err()
}

func (r PostgresV2ChargingRepository) Locations(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2ChargingLocationItem, V2ChargingStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			COALESCE(geofences.name, CONCAT_WS(', ', COALESCE(addresses.name, nullif(CONCAT_WS(' ', addresses.road, addresses.house_number), '')), addresses.city), 'unknown') AS location_name,
			charging_processes.geofence_id,
			charging_processes.address_id,
			COUNT(*) AS session_count,
			COALESCE(SUM(charge_energy_added), 0) AS energy_added,
			SUM(charge_energy_used) AS energy_used,
			SUM(cost) AS cost,
			COUNT(charge_energy_used) AS energy_used_rows,
			COUNT(cost) AS cost_rows
		FROM charging_processes
		LEFT JOIN geofences ON geofences.id = charging_processes.geofence_id
		LEFT JOIN addresses ON addresses.id = charging_processes.address_id
		WHERE charging_processes.car_id = $1
			AND charging_processes.end_date IS NOT NULL
			AND charging_processes.start_date >= $2
			AND charging_processes.start_date < $3
		GROUP BY 1, 2, 3
		ORDER BY session_count DESC, energy_added DESC`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	)
	if err != nil {
		return nil, V2ChargingStats{}, err
	}
	defer rows.Close()

	var items []V2ChargingLocationItem
	var stats V2ChargingStats
	for rows.Next() {
		var item V2ChargingLocationItem
		var geofenceID sql.NullInt64
		var addressID sql.NullInt64
		var energyUsed sql.NullFloat64
		var cost sql.NullFloat64
		var energyRows, costRows int64
		if err := rows.Scan(&item.LocationName, &geofenceID, &addressID, &item.SessionCount, &item.EnergyAdded, &energyUsed, &cost, &energyRows, &costRows); err != nil {
			return nil, stats, err
		}
		if geofenceID.Valid {
			item.GeofenceID = &geofenceID.Int64
		}
		if addressID.Valid {
			item.AddressID = &addressID.Int64
		}
		if energyUsed.Valid {
			item.EnergyUsed = &energyUsed.Float64
			if energyUsed.Float64 > 0 {
				item.ChargingEfficiency = float64Ptr(item.EnergyAdded / energyUsed.Float64 * 100)
			}
		}
		if cost.Valid {
			item.Cost = &cost.Float64
		}
		stats.SessionRows += item.SessionCount
		stats.EnergyUsedRows += energyRows
		stats.CostRows += costRows
		stats.PowerRows += item.SessionCount
		items = append(items, item)
	}
	return items, stats, rows.Err()
}

func (r PostgresV2ChargingRepository) Types(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2ChargingTypeItem, V2ChargingStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			CASE
				WHEN LOWER(COALESCE(max_charge.fast_charger_brand, '')) LIKE '%tesla%'
					OR LOWER(COALESCE(max_charge.fast_charger_type, '')) LIKE '%supercharger%'
				THEN 'supercharger'
				WHEN COALESCE(max_charge.fast_charger_present, false)
					OR COALESCE(max_charge.charger_power, 0) >= 20
				THEN 'dc'
				WHEN COALESCE(max_charge.charger_power, 0) > 0 THEN 'ac'
				ELSE 'unknown'
			END AS charging_type,
			COUNT(*) AS session_count,
			COALESCE(SUM(charge_energy_added), 0) AS energy_added,
			SUM(charge_energy_used) AS energy_used,
			SUM(cost) AS cost,
			COUNT(charge_energy_used) AS energy_used_rows,
			COUNT(cost) AS cost_rows,
			COUNT(NULLIF(max_charge.charger_power, 0)) AS power_rows
		FROM charging_processes
		LEFT JOIN LATERAL (
			SELECT
				MAX(charger_power) AS charger_power,
				BOOL_OR(COALESCE(fast_charger_present, false)) AS fast_charger_present,
				MAX(NULLIF(fast_charger_brand, '')) AS fast_charger_brand,
				MAX(NULLIF(fast_charger_type, '')) AS fast_charger_type
			FROM charges
			WHERE charges.charging_process_id = charging_processes.id
		) max_charge ON true
		WHERE charging_processes.car_id = $1
			AND charging_processes.end_date IS NOT NULL
			AND charging_processes.start_date >= $2
			AND charging_processes.start_date < $3
		GROUP BY 1
		ORDER BY session_count DESC`,
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time,
	)
	if err != nil {
		return nil, V2ChargingStats{}, err
	}
	defer rows.Close()

	var items []V2ChargingTypeItem
	var stats V2ChargingStats
	for rows.Next() {
		var item V2ChargingTypeItem
		var energyUsed sql.NullFloat64
		var cost sql.NullFloat64
		var energyRows, costRows, powerRows int64
		if err := rows.Scan(&item.ChargingType, &item.SessionCount, &item.EnergyAdded, &energyUsed, &cost, &energyRows, &costRows, &powerRows); err != nil {
			return nil, stats, err
		}
		if energyUsed.Valid {
			item.EnergyUsed = &energyUsed.Float64
		}
		if cost.Valid {
			item.Cost = &cost.Float64
		}
		stats.SessionRows += item.SessionCount
		stats.EnergyUsedRows += energyRows
		stats.CostRows += costRows
		stats.PowerRows += powerRows
		items = append(items, item)
	}
	return items, stats, rows.Err()
}
