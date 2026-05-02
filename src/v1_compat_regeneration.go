package main

import (
	"database/sql"
	"fmt"
)

type RegenerationSummary struct {
	Coverage                     HistorySummaryCoverage `json:"coverage"`
	MetricsEstimated             bool                   `json:"metrics_estimated"`
	DriveCountWithRegeneration   int                    `json:"drive_count_with_regeneration"`
	RegenerationEventCount       int                    `json:"regeneration_event_count"`
	TotalRecoveredEnergy         *float64               `json:"total_recovered_energy,omitempty"`
	EstimatedRecoveredEnergyKwh  *float64               `json:"estimated_recovered_energy,omitempty"`
	AverageRecoveredEnergy       *float64               `json:"average_recovered_energy,omitempty"`
	TotalRegenerationDurationMin int                    `json:"total_regeneration_duration_min"`
	MaxPeakRegenerationPower     *float64               `json:"max_peak_regeneration_power,omitempty"`
	AveragePeakRegenerationPower *float64               `json:"average_peak_regeneration_power,omitempty"`
	RecoveryShare                *float64               `json:"recovery_share,omitempty"`
	RegenEnergyPer100Distance    *float64               `json:"estimated_regen_energy_per_100_distance,omitempty"`
	MonthlyRecoveredEnergy       []SummaryCategoryValue `json:"monthly_recovered_energy"`
}

func fetchRegenerationSummary(CarID int, parsedStartDate string, parsedEndDate string, driveSummary *DriveHistorySummary, unitsLength string) (*RegenerationSummary, error) {
	query := `
		WITH position_samples AS (
			SELECT
				drives.id AS drive_id,
				drives.start_date,
				positions.id AS position_id,
				positions.date,
				COALESCE(positions.power, 0)::float8 AS power,
				LAG(positions.date) OVER (PARTITION BY drives.id ORDER BY positions.id) AS previous_date
			FROM drives
			INNER JOIN positions ON positions.drive_id = drives.id
			WHERE drives.car_id = $1
				AND drives.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, paramIndex = appendSummaryDateFilters(query, queryParams, paramIndex, "drives", parsedStartDate, parsedEndDate)
	query += `
		),
		regen_samples AS (
			SELECT
				drive_id,
				start_date,
				ABS(LEAST(power, 0)) AS recovered_power_kw,
				EXTRACT(EPOCH FROM (date - previous_date)) AS delta_sec
			FROM position_samples
			WHERE previous_date IS NOT NULL
				AND power < 0
		),
		normalized_regen AS (
			SELECT
				drive_id,
				start_date,
				recovered_power_kw,
				delta_sec
			FROM regen_samples
			WHERE delta_sec > 0
				AND delta_sec <= 300
		),
		per_drive AS (
			SELECT
				drive_id,
				COUNT(*)::int AS regen_event_count,
				SUM(recovered_power_kw * delta_sec / 3600.0) AS recovered_energy_kwh,
				SUM(delta_sec) / 60.0 AS regen_duration_min,
				MAX(recovered_power_kw) AS peak_regen_power,
				MIN(start_date) AS first_start_date,
				MAX(start_date) AS last_start_date
			FROM normalized_regen
			GROUP BY drive_id
		)
		SELECT
			COUNT(*)::int AS drive_count_with_regeneration,
			COALESCE(SUM(regen_event_count), 0)::int AS regeneration_event_count,
			SUM(recovered_energy_kwh) AS total_recovered_energy,
			AVG(recovered_energy_kwh) AS average_recovered_energy,
			COALESCE(SUM(regen_duration_min), 0)::int AS total_regeneration_duration_min,
			MAX(peak_regen_power) AS max_peak_regeneration_power,
			AVG(peak_regen_power) AS average_peak_regeneration_power,
			MIN(first_start_date) AS coverage_start,
			MAX(last_start_date) AS coverage_end
		FROM per_drive;`

	var (
		driveCountWithRegeneration   int
		regenerationEventCount       int
		totalRecoveredEnergy         sql.NullFloat64
		averageRecoveredEnergy       sql.NullFloat64
		totalRegenerationDurationMin int
		maxPeakRegenerationPower     sql.NullFloat64
		averagePeakRegenerationPower sql.NullFloat64
		coverageStart                sql.NullString
		coverageEnd                  sql.NullString
	)

	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(queryCtx, query, queryParams...).Scan(
		&driveCountWithRegeneration,
		&regenerationEventCount,
		&totalRecoveredEnergy,
		&averageRecoveredEnergy,
		&totalRegenerationDurationMin,
		&maxPeakRegenerationPower,
		&averagePeakRegenerationPower,
		&coverageStart,
		&coverageEnd,
	); err != nil {
		return nil, err
	}

	monthlyRecoveredEnergy, err := fetchMonthlyRecoveredEnergy(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		return nil, err
	}

	if driveCountWithRegeneration == 0 && len(monthlyRecoveredEnergy) == 0 {
		return &RegenerationSummary{MetricsEstimated: true, MonthlyRecoveredEnergy: monthlyRecoveredEnergy}, nil
	}

	estEnergy := floatPointer(totalRecoveredEnergy)
	summary := &RegenerationSummary{
		MetricsEstimated: true,
		Coverage: HistorySummaryCoverage{
			StartDate: timeZoneStringPointer(coverageStart),
			EndDate:   timeZoneStringPointer(coverageEnd),
		},
		DriveCountWithRegeneration:   driveCountWithRegeneration,
		RegenerationEventCount:       regenerationEventCount,
		TotalRecoveredEnergy:         estEnergy,
		EstimatedRecoveredEnergyKwh:  estEnergy,
		AverageRecoveredEnergy:       floatPointer(averageRecoveredEnergy),
		TotalRegenerationDurationMin: totalRegenerationDurationMin,
		MaxPeakRegenerationPower:     floatPointer(maxPeakRegenerationPower),
		AveragePeakRegenerationPower: floatPointer(averagePeakRegenerationPower),
		MonthlyRecoveredEnergy:       monthlyRecoveredEnergy,
	}

	if driveSummary != nil && driveSummary.TotalDistance > 0 && totalRecoveredEnergy.Valid && totalRecoveredEnergy.Float64 > 0 {
		v := totalRecoveredEnergy.Float64 / driveSummary.TotalDistance * 100.0
		summary.RegenEnergyPer100Distance = &v
	}

	if driveSummary != nil && driveSummary.TotalEnergyConsumed != nil && totalRecoveredEnergy.Valid {
		denominator := *driveSummary.TotalEnergyConsumed + totalRecoveredEnergy.Float64
		if denominator > 0 {
			value := totalRecoveredEnergy.Float64 / denominator
			summary.RecoveryShare = &value
		}
	}

	return summary, nil
}

func fetchMonthlyRecoveredEnergy(CarID int, parsedStartDate string, parsedEndDate string) ([]SummaryCategoryValue, error) {
	query := `
		WITH position_samples AS (
			SELECT
				drives.start_date,
				positions.id AS position_id,
				positions.date,
				COALESCE(positions.power, 0)::float8 AS power,
				LAG(positions.date) OVER (PARTITION BY drives.id ORDER BY positions.id) AS previous_date
			FROM drives
			INNER JOIN positions ON positions.drive_id = drives.id
			WHERE drives.car_id = $1
				AND drives.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, _ = appendSummaryDateFilters(query, queryParams, paramIndex, "drives", parsedStartDate, parsedEndDate)
	query += fmt.Sprintf(`
		),
		normalized_regen AS (
			SELECT
				TO_CHAR(date_trunc('month', timezone($%d, start_date)), 'YYYY-MM') AS period,
				ABS(LEAST(power, 0)) * EXTRACT(EPOCH FROM (date - previous_date)) / 3600.0 AS recovered_energy_kwh
			FROM position_samples
			WHERE previous_date IS NOT NULL
				AND power < 0
				AND EXTRACT(EPOCH FROM (date - previous_date)) > 0
				AND EXTRACT(EPOCH FROM (date - previous_date)) <= 300
		)
		SELECT
			period,
			COALESCE(SUM(recovered_energy_kwh), 0)
		FROM normalized_regen
		GROUP BY period
		ORDER BY period ASC;`, len(queryParams)+1)
	queryParams = append(queryParams, appUsersTimezone.String())

	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SummaryCategoryValue, 0)
	index := 0
	for rows.Next() {
		var (
			period string
			value  float64
		)
		if err := rows.Scan(&period, &value); err != nil {
			return nil, err
		}
		index++
		result = append(result, SummaryCategoryValue{
			ID:     fmt.Sprintf("regen-month-%d", index),
			Label:  period,
			Period: period,
			Value:  value,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
