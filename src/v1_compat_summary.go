package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type HistorySummaryCoverage struct {
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

type DriveHistorySummary struct {
	Coverage                 HistorySummaryCoverage `json:"coverage"`
	DriveCount               int                    `json:"drive_count"`
	TotalDurationMin         int                    `json:"total_duration_min"`
	TotalDistance            float64                `json:"total_distance"`
	LongestDistance          *float64               `json:"longest_distance"`
	LongestDurationMin       *int                   `json:"longest_duration_min"`
	AverageDistance          *float64               `json:"average_distance"`
	AverageDurationMin       *float64               `json:"average_duration_min"`
	AverageSpeed             *float64               `json:"average_speed"`
	TotalEnergyConsumed      *float64               `json:"total_energy_consumed"`
	AverageConsumption       *float64               `json:"average_consumption"`
	BestConsumption          *float64               `json:"best_consumption"`
	WorstConsumption         *float64               `json:"worst_consumption"`
	MaxSpeed                 *int                   `json:"max_speed"`
	PeakDrivePower           *int                   `json:"peak_drive_power"`
	PeakRegenPower           *int                   `json:"peak_regen_power"`
	LowSpeedTripCount        int                    `json:"low_speed_trip_count"`
	CongestionLikeTripCount  int                    `json:"congestion_like_trip_count"`
	HighConsumptionTripCount int                    `json:"high_consumption_trip_count"`
}

type ChargeHistorySummary struct {
	Coverage                 HistorySummaryCoverage `json:"coverage"`
	ChargeCount              int                    `json:"charge_count"`
	TotalDurationMin         int                    `json:"total_duration_min"`
	TotalEnergyAdded         float64                `json:"total_energy_added"`
	TotalEnergyUsed          *float64               `json:"total_energy_used"`
	LongestDurationMin       *int                   `json:"longest_duration_min"`
	LargestEnergyAdded       *float64               `json:"largest_energy_added"`
	AverageEnergyAdded       *float64               `json:"average_energy_added"`
	AverageDurationMin       *float64               `json:"average_duration_min"`
	AveragePower             *float64               `json:"average_power"`
	MaxPower                 *int                   `json:"max_power"`
	ChargingEfficiency       *float64               `json:"charging_efficiency"`
	TotalCost                *float64               `json:"total_cost"`
	AverageCost              *float64               `json:"average_cost"`
	HighestCost              *float64               `json:"highest_cost"`
	AverageCostPerKwh        *float64               `json:"average_cost_per_kwh"`
	CostPer100Distance       *float64               `json:"cost_per_100_distance"`
	LowEfficiencyChargeCount int                    `json:"low_efficiency_charge_count"`
	AbnormalChargeCount      int                    `json:"abnormal_charge_count"`
}
type SummaryTimeSeriesPoint struct {
	ID    string  `json:"id"`
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type SummaryCategoryValue struct {
	ID     string         `json:"id"`
	Label  string         `json:"label"`
	Period string         `json:"period,omitempty"`
	Value  float64        `json:"value"`
	Unit   string         `json:"unit,omitempty"`
	Extra  map[string]any `json:"extra,omitempty"`
}
type TeslaMateSummaryUnits struct {
	UnitsLength           string `json:"unit_of_length"`
	UnitsTemperature      string `json:"unit_of_temperature"`
	UnitOfSpeed           string `json:"unit_of_speed"`
	UnitOfConsumption     string `json:"unit_of_consumption"`
	UnitOfCostPerDistance string `json:"unit_of_cost_per_distance"`
}

func fetchSummaryMetadata(CarID int) (string, string, NullString, error) {
	var unitsLength, unitsTemperature string
	var carName NullString

	err := db.QueryRow(`
		SELECT
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name
		FROM cars
		WHERE cars.id = $1;`, CarID).Scan(&unitsLength, &unitsTemperature, &carName)
	if err != nil {
		return "", "", "", err
	}

	return unitsLength, unitsTemperature, carName, nil
}

func fetchDriveHistorySummary(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) (*DriveHistorySummary, error) {
	lowSpeedKmh := 15.0
	congestionKmh := 10.0
	minCongestionDistKm := 1.0
	if unitsLength == "mi" {
		lowSpeedKmh = 10.0 * 1.609344
		congestionKmh = 6.0 * 1.609344
		minCongestionDistKm = 0.6 * 1.609344
	}
	query := `
		WITH filtered_drives AS (
			SELECT
				drives.start_date,
				drives.end_date,
				GREATEST(COALESCE(drives.duration_min, 0), 0) AS duration_min,
				GREATEST(COALESCE(drives.distance, 0), 0) AS distance,
				drives.speed_max,
				drives.power_max,
				drives.power_min,
				CASE
					WHEN (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency
					ELSE NULL
				END AS energy_consumed_net,
				CASE
					WHEN (
						drives.duration_min > 1
						AND drives.distance > 1
						AND (
							start_position.usable_battery_level IS NULL
							OR end_position.usable_battery_level IS NULL
							OR (end_position.battery_level - end_position.usable_battery_level) = 0
						)
					)
					AND NULLIF(drives.distance, 0) IS NOT NULL
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / NULLIF(drives.distance, 0) * 1000
					ELSE NULL
				END AS consumption_net
			FROM drives
			LEFT JOIN cars ON drives.car_id = cars.id
			LEFT JOIN positions start_position ON drives.start_position_id = start_position.id
			LEFT JOIN positions end_position ON drives.end_position_id = end_position.id
			WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, paramIndex = appendSummaryDateFilters(query, queryParams, paramIndex, "drives", parsedStartDate, parsedEndDate)
	query += fmt.Sprintf(`
		),
		aggregated_drives AS (
			SELECT
				COUNT(*) AS drive_count,
				COALESCE(SUM(duration_min), 0) AS total_duration_min,
				COALESCE(SUM(distance), 0) AS total_distance,
				MAX(NULLIF(distance, 0)) AS longest_distance,
				MAX(NULLIF(duration_min, 0)) AS longest_duration_min,
				CASE WHEN COUNT(*) > 0 AND SUM(distance) > 0 THEN SUM(distance) / COUNT(*) ELSE NULL END AS average_distance,
				CASE WHEN COUNT(*) > 0 AND SUM(duration_min) > 0 THEN SUM(duration_min)::float8 / COUNT(*) ELSE NULL END AS average_duration_min,
				CASE WHEN SUM(duration_min) > 0 AND SUM(distance) > 0 THEN SUM(distance) / (SUM(duration_min)::float8 / 60.0) ELSE NULL END AS average_speed,
				NULLIF(SUM(GREATEST(COALESCE(energy_consumed_net, 0), 0)), 0) AS total_energy_consumed,
				CASE
					WHEN SUM(distance) > 0 AND SUM(GREATEST(COALESCE(energy_consumed_net, 0), 0)) > 0
					THEN SUM(GREATEST(COALESCE(energy_consumed_net, 0), 0)) / SUM(distance) * 1000.0
					ELSE NULL
				END AS average_consumption,
				MIN(CASE WHEN consumption_net > 0 THEN consumption_net ELSE NULL END) AS best_consumption,
				MAX(CASE WHEN consumption_net > 0 THEN consumption_net ELSE NULL END) AS worst_consumption,
				MAX(speed_max) AS max_speed,
				MAX(power_max) AS peak_drive_power,
				MAX(CASE WHEN power_min < 0 THEN ABS(power_min) ELSE NULL END) AS peak_regen_power,
				MIN(start_date) AS coverage_start,
				MAX(end_date) AS coverage_end
			FROM filtered_drives
		)
		SELECT
			agg.drive_count,
			agg.total_duration_min,
			agg.total_distance,
			agg.longest_distance,
			agg.longest_duration_min,
			agg.average_distance,
			agg.average_duration_min,
			agg.average_speed,
			agg.total_energy_consumed,
			agg.average_consumption,
			agg.best_consumption,
			agg.worst_consumption,
			agg.max_speed,
			agg.peak_drive_power,
			agg.peak_regen_power,
			agg.coverage_start,
			agg.coverage_end,
			(SELECT COUNT(*)::int FROM filtered_drives fd
			 WHERE fd.duration_min >= 10
			   AND fd.duration_min > 0
			   AND COALESCE(fd.distance, 0) > 0
			   AND (fd.distance / (fd.duration_min::float8 / 60.0)) < %g
			) AS low_speed_trip_count,
			(SELECT COUNT(*)::int FROM filtered_drives fd
			 WHERE fd.duration_min > 0
			   AND COALESCE(fd.distance, 0) >= %g
			   AND (fd.distance / (fd.duration_min::float8 / 60.0)) < %g
			) AS congestion_like_trip_count,
			(SELECT COUNT(*)::int FROM filtered_drives fd
			 WHERE fd.consumption_net IS NOT NULL
			   AND agg.average_consumption IS NOT NULL
			   AND fd.consumption_net > agg.average_consumption * 1.5
			) AS high_consumption_trip_count
		FROM aggregated_drives agg;`, lowSpeedKmh, minCongestionDistKm, congestionKmh)

	var (
		driveCount               int
		totalDurationMin         int
		totalDistance            float64
		longestDistance          sql.NullFloat64
		longestDurationMin       sql.NullInt64
		averageDistance          sql.NullFloat64
		averageDurationMin       sql.NullFloat64
		averageSpeed             sql.NullFloat64
		totalEnergyConsumed      sql.NullFloat64
		averageConsumption       sql.NullFloat64
		bestConsumption          sql.NullFloat64
		worstConsumption         sql.NullFloat64
		maxSpeed                 sql.NullInt64
		peakDrivePower           sql.NullInt64
		peakRegenPower           sql.NullInt64
		coverageStart            sql.NullString
		coverageEnd              sql.NullString
		lowSpeedTripCount        int
		congestionLikeTripCount  int
		highConsumptionTripCount int
	)

	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	err := db.QueryRowContext(queryCtx, query, queryParams...).Scan(
		&driveCount,
		&totalDurationMin,
		&totalDistance,
		&longestDistance,
		&longestDurationMin,
		&averageDistance,
		&averageDurationMin,
		&averageSpeed,
		&totalEnergyConsumed,
		&averageConsumption,
		&bestConsumption,
		&worstConsumption,
		&maxSpeed,
		&peakDrivePower,
		&peakRegenPower,
		&coverageStart,
		&coverageEnd,
		&lowSpeedTripCount,
		&congestionLikeTripCount,
		&highConsumptionTripCount,
	)
	if err != nil {
		return nil, err
	}
	if driveCount == 0 {
		return &DriveHistorySummary{
			Coverage: HistorySummaryCoverage{},
		}, nil
	}

	if unitsLength == "mi" {
		totalDistance = kilometersToMiles(totalDistance)
		longestDistance = kilometersToMilesSqlNullFloat64(longestDistance)
		averageDistance = kilometersToMilesSqlNullFloat64(averageDistance)
		averageSpeed = kmhToMphNull(averageSpeed)
		averageConsumption = whPerKmToWhPerMiNull(averageConsumption)
		bestConsumption = whPerKmToWhPerMiNull(bestConsumption)
		worstConsumption = whPerKmToWhPerMiNull(worstConsumption)
		maxSpeed = kilometersToMilesSqlNullInt64(maxSpeed)
	}

	return &DriveHistorySummary{
		Coverage: HistorySummaryCoverage{
			StartDate: timeZoneStringPointer(coverageStart),
			EndDate:   timeZoneStringPointer(coverageEnd),
		},
		DriveCount:               driveCount,
		TotalDurationMin:         totalDurationMin,
		TotalDistance:            totalDistance,
		LongestDistance:          floatPointer(longestDistance),
		LongestDurationMin:       intPointer(longestDurationMin),
		AverageDistance:          floatPointer(averageDistance),
		AverageDurationMin:       floatPointer(averageDurationMin),
		AverageSpeed:             floatPointer(averageSpeed),
		TotalEnergyConsumed:      floatPointer(totalEnergyConsumed),
		AverageConsumption:       floatPointer(averageConsumption),
		BestConsumption:          floatPointer(bestConsumption),
		WorstConsumption:         floatPointer(worstConsumption),
		MaxSpeed:                 intPointer(maxSpeed),
		PeakDrivePower:           intPointer(peakDrivePower),
		PeakRegenPower:           intPointer(peakRegenPower),
		LowSpeedTripCount:        lowSpeedTripCount,
		CongestionLikeTripCount:  congestionLikeTripCount,
		HighConsumptionTripCount: highConsumptionTripCount,
	}, nil
}

func fetchChargeHistorySummary(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) (*ChargeHistorySummary, error) {
	query := `
		WITH filtered_charges AS (
			SELECT
				charging_processes.id,
				charging_processes.start_date,
				charging_processes.end_date,
				GREATEST(COALESCE(charging_processes.duration_min, 0), 0) AS duration_min,
				GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0) AS charge_energy_added,
				GREATEST(
					COALESCE(charging_processes.charge_energy_used, 0),
					COALESCE(charging_processes.charge_energy_added, 0),
					0
				) AS charge_energy_used,
				charging_processes.cost AS cost
			FROM charging_processes
			WHERE charging_processes.car_id=$1 AND charging_processes.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, paramIndex = appendSummaryDateFilters(query, queryParams, paramIndex, "charging_processes", parsedStartDate, parsedEndDate)
	query += `
		),
		charge_quality AS (
			SELECT
				COUNT(*) FILTER (WHERE
					COALESCE(charge_energy_added, 0) <= 0
					OR (duration_min >= 120 AND COALESCE(charge_energy_added, 0) < 1)
					OR (charge_energy_used > 0 AND charge_energy_added > 0 AND (charge_energy_added / charge_energy_used) < 0.6)
					OR (charge_energy_used > 0 AND charge_energy_added > 0 AND (charge_energy_added / charge_energy_used) > 1.1)
					OR (cost IS NOT NULL AND cost < 0)
				)::int AS abnormal_charge_count,
				COUNT(*) FILTER (WHERE
					charge_energy_used > 0 AND charge_energy_added > 0
					AND ((charge_energy_added / charge_energy_used) < 0.6 OR (charge_energy_added / charge_energy_used) > 1.1)
				)::int AS low_efficiency_charge_count
			FROM filtered_charges
		),
		aggregated_charges AS (
			SELECT
				COUNT(*) AS charge_count,
				COALESCE(SUM(duration_min), 0) AS total_duration_min,
				COALESCE(SUM(charge_energy_added), 0) AS total_energy_added,
				NULLIF(SUM(charge_energy_used), 0) AS total_energy_used,
				MAX(NULLIF(duration_min, 0)) AS longest_duration_min,
				MAX(NULLIF(charge_energy_added, 0)) AS largest_energy_added,
				CASE WHEN COUNT(*) > 0 AND SUM(charge_energy_added) > 0 THEN SUM(charge_energy_added) / COUNT(*) ELSE NULL END AS average_energy_added,
				CASE WHEN COUNT(*) > 0 AND SUM(duration_min) > 0 THEN SUM(duration_min)::float8 / COUNT(*) ELSE NULL END AS average_duration_min,
				CASE
					WHEN SUM(duration_min) > 0 AND GREATEST(SUM(charge_energy_used), SUM(charge_energy_added)) > 0
					THEN GREATEST(SUM(charge_energy_used), SUM(charge_energy_added)) / (SUM(duration_min)::float8 / 60.0)
					ELSE NULL
				END AS average_power,
				CASE
					WHEN SUM(charge_energy_used) > 0 AND SUM(charge_energy_added) > 0
					THEN SUM(charge_energy_added) / SUM(charge_energy_used)
					ELSE NULL
				END AS charging_efficiency,
				NULLIF(SUM(CASE WHEN cost IS NOT NULL AND cost > 0 THEN cost ELSE 0 END), 0) AS total_cost,
				CASE
					WHEN COUNT(CASE WHEN cost IS NOT NULL AND cost > 0 THEN 1 ELSE NULL END) > 0
					THEN SUM(CASE WHEN cost IS NOT NULL AND cost > 0 THEN cost ELSE 0 END) / COUNT(CASE WHEN cost IS NOT NULL AND cost > 0 THEN 1 ELSE NULL END)
					ELSE NULL
				END AS average_cost,
				MAX(CASE WHEN cost IS NOT NULL AND cost > 0 THEN cost ELSE NULL END) AS highest_cost,
				MIN(start_date) AS coverage_start,
				MAX(end_date) AS coverage_end
			FROM filtered_charges
		),
		peak_power AS (
			SELECT MAX(NULLIF(charges.charger_power, 0)) AS max_power
			FROM filtered_charges
			LEFT JOIN charges ON charges.charging_process_id = filtered_charges.id
		)
		SELECT
			charge_count,
			total_duration_min,
			total_energy_added,
			total_energy_used,
			longest_duration_min,
			largest_energy_added,
			average_energy_added,
			average_duration_min,
			average_power,
			max_power,
			charging_efficiency,
			total_cost,
			average_cost,
			highest_cost,
			coverage_start,
			coverage_end,
			cq.abnormal_charge_count,
			cq.low_efficiency_charge_count
		FROM aggregated_charges, peak_power, charge_quality cq;`

	var (
		chargeCount              int
		totalDurationMin         int
		totalEnergyAdded         float64
		totalEnergyUsed          sql.NullFloat64
		longestDurationMin       sql.NullInt64
		largestEnergyAdded       sql.NullFloat64
		averageEnergyAdded       sql.NullFloat64
		averageDurationMin       sql.NullFloat64
		averagePower             sql.NullFloat64
		maxPower                 sql.NullInt64
		chargingEfficiency       sql.NullFloat64
		totalCost                sql.NullFloat64
		averageCost              sql.NullFloat64
		highestCost              sql.NullFloat64
		coverageStart            sql.NullString
		coverageEnd              sql.NullString
		abnormalChargeCount      int
		lowEfficiencyChargeCount int
	)

	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	err := db.QueryRowContext(queryCtx, query, queryParams...).Scan(
		&chargeCount,
		&totalDurationMin,
		&totalEnergyAdded,
		&totalEnergyUsed,
		&longestDurationMin,
		&largestEnergyAdded,
		&averageEnergyAdded,
		&averageDurationMin,
		&averagePower,
		&maxPower,
		&chargingEfficiency,
		&totalCost,
		&averageCost,
		&highestCost,
		&coverageStart,
		&coverageEnd,
		&abnormalChargeCount,
		&lowEfficiencyChargeCount,
	)
	if err != nil {
		return nil, err
	}
	if chargeCount == 0 {
		return &ChargeHistorySummary{
			Coverage: HistorySummaryCoverage{},
		}, nil
	}

	periodDriveKm, err := fetchPeriodDriveDistanceKm(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		return nil, err
	}
	var averageCostPerKwh *float64
	if totalEnergyAdded > 0 && totalCost.Valid && totalCost.Float64 > 0 {
		v := totalCost.Float64 / totalEnergyAdded
		averageCostPerKwh = &v
	}
	var costPer100 *float64
	denom := periodDriveKm
	if unitsLength == "mi" && denom > 0 {
		denom = kilometersToMiles(denom)
	}
	if denom > 0 && totalCost.Valid && totalCost.Float64 > 0 {
		v := totalCost.Float64 / denom * 100.0
		costPer100 = &v
	}

	return &ChargeHistorySummary{
		Coverage: HistorySummaryCoverage{
			StartDate: timeZoneStringPointer(coverageStart),
			EndDate:   timeZoneStringPointer(coverageEnd),
		},
		ChargeCount:              chargeCount,
		TotalDurationMin:         totalDurationMin,
		TotalEnergyAdded:         totalEnergyAdded,
		TotalEnergyUsed:          floatPointer(totalEnergyUsed),
		LongestDurationMin:       intPointer(longestDurationMin),
		LargestEnergyAdded:       floatPointer(largestEnergyAdded),
		AverageEnergyAdded:       floatPointer(averageEnergyAdded),
		AverageDurationMin:       floatPointer(averageDurationMin),
		AveragePower:             floatPointer(averagePower),
		MaxPower:                 intPointer(maxPower),
		ChargingEfficiency:       floatPointer(chargingEfficiency),
		TotalCost:                floatPointer(totalCost),
		AverageCost:              floatPointer(averageCost),
		HighestCost:              floatPointer(highestCost),
		AverageCostPerKwh:        averageCostPerKwh,
		CostPer100Distance:       costPer100,
		LowEfficiencyChargeCount: lowEfficiencyChargeCount,
		AbnormalChargeCount:      abnormalChargeCount,
	}, nil
}

func fetchPeriodDriveDistanceKm(CarID int, parsedStartDate string, parsedEndDate string) (float64, error) {
	q := `
		SELECT COALESCE(SUM(GREATEST(COALESCE(drives.distance, 0), 0)), 0)::float8
		FROM drives
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL`
	params := []any{CarID}
	idx := 2
	q, params, idx = appendSummaryDateFilters(q, params, idx, "drives", parsedStartDate, parsedEndDate)
	var dist sql.NullFloat64
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(queryCtx, q, params...).Scan(&dist); err != nil {
		return 0, err
	}
	if !dist.Valid {
		return 0, nil
	}
	return dist.Float64, nil
}

func appendSummaryDateFilters(query string, queryParams []any, paramIndex int, table string, parsedStartDate string, parsedEndDate string) (string, []any, int) {
	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND %s.start_date >= $%d", table, paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND %s.end_date < $%d", table, paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}
	return query, queryParams, paramIndex
}

func floatPointer(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func intPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func timeZoneStringPointer(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	result := getTimeInTimeZone(value.String)
	return &result
}

func kilometersToMilesSqlNullFloat64(value sql.NullFloat64) sql.NullFloat64 {
	if value.Valid {
		value.Float64 = kilometersToMiles(value.Float64)
	}
	return value
}

func kilometersToMilesSqlNullInt64(value sql.NullInt64) sql.NullInt64 {
	if value.Valid {
		value.Int64 = int64(kilometersToMilesInteger(int(value.Int64)))
	}
	return value
}
