package main

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type GrossConsumptionSummary struct {
	AverageConsumptionGross *float64 `json:"average_consumption_gross"`
	DataComplete            *bool    `json:"data_complete"`
}

type StatisticsSummary struct {
	Coverage                   HistorySummaryCoverage `json:"coverage"`
	Trips                      int                    `json:"trips"`
	DriveCount                 int                    `json:"drive_count"`
	ChargeCount                int                    `json:"charge_count"`
	TimeDrivenMin              int                    `json:"time_driven_min"`
	Distance                   float64                `json:"distance"`
	MaxSpeed                   *int                   `json:"max_speed"`
	AverageSpeed               *float64               `json:"average_speed,omitempty"`
	AverageOutsideTemp         *float64               `json:"average_outside_temp"`
	AverageConsumptionNet      *float64               `json:"average_consumption_net"`
	AverageConsumptionGross    *float64               `json:"average_consumption_gross"`
	DrivingEfficiency          *float64               `json:"driving_efficiency"`
	EnergyAdded                *float64               `json:"energy_added,omitempty"`
	EnergyUsed                 *float64               `json:"energy_used"`
	ChargingEfficiency         *float64               `json:"charging_efficiency,omitempty"`
	AverageEnergyUsedPerCharge *float64               `json:"average_energy_used_per_charge"`
	TotalCost                  *float64               `json:"total_cost"`
	AverageCostPerKwh          *float64               `json:"average_cost_per_kwh"`
	AverageCostPer100Distance  *float64               `json:"average_cost_per_100_distance"`
	ConsumptionOverhead        *float64               `json:"consumption_overhead"`
	DataComplete               *bool                  `json:"data_complete"`
}

type StateBreakdown struct {
	State        string  `json:"state"`
	SessionCount int     `json:"session_count"`
	DurationMin  int     `json:"duration_min"`
	Share        float64 `json:"share"`
}

type StateSummary struct {
	Coverage        HistorySummaryCoverage `json:"coverage"`
	CurrentState    *string                `json:"current_state"`
	LastStateChange *string                `json:"last_state_change"`
	ParkedShare     *float64               `json:"parked_share"`
	StateBreakdown  []StateBreakdown       `json:"state_breakdown"`
}

type StateTimelineItem struct {
	TimelineID  string  `json:"timeline_id"`
	State       string  `json:"state"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date"`
	DurationMin int     `json:"duration_min"`
	DurationStr string  `json:"duration_str"`
	IsOpen      bool    `json:"is_open"`
}

type ActivityTimelineEvent struct {
	ID          string         `json:"id"`
	SourceID    string         `json:"source_id"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	StartDate   string         `json:"start_date"`
	EndDate     *string        `json:"end_date,omitempty"`
	DurationMin int            `json:"duration_min"`
	Metrics     map[string]any `json:"metrics"`
}

func activityTimelineTypeAndSource(timelineID string, state string) (string, string) {
	switch {
	case strings.HasPrefix(timelineID, "drive-"):
		return "drive", strings.TrimPrefix(timelineID, "drive-")
	case strings.HasPrefix(timelineID, "charge-"):
		return "charge", strings.TrimPrefix(timelineID, "charge-")
	case strings.HasPrefix(timelineID, "update-"):
		return "update", strings.TrimPrefix(timelineID, "update-")
	case strings.HasPrefix(timelineID, "state-"):
		switch strings.ToLower(state) {
		case "online", "offline", "asleep":
			return "parking", strings.TrimPrefix(timelineID, "state-")
		default:
			return "state", strings.TrimPrefix(timelineID, "state-")
		}
	default:
		return "state", timelineID
	}
}

func mapStateTimelineToActivityEvents(items []StateTimelineItem) []ActivityTimelineEvent {
	out := make([]ActivityTimelineEvent, 0, len(items))
	for _, it := range items {
		typ, src := activityTimelineTypeAndSource(it.TimelineID, it.State)
		title := it.State
		switch typ {
		case "drive":
			title = "Drive"
		case "charge":
			title = "Charge"
		case "update":
			title = "Software update"
		case "parking":
			title = "Parking (" + it.State + ")"
		case "state":
			title = "State: " + it.State
		}
		out = append(out, ActivityTimelineEvent{
			ID:          typ + "-" + src,
			SourceID:    src,
			Type:        typ,
			Title:       title,
			StartDate:   it.StartDate,
			EndDate:     it.EndDate,
			DurationMin: it.DurationMin,
			Metrics: map[string]any{
				"duration_str": it.DurationStr,
				"is_open":      it.IsOpen,
				"state":        it.State,
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].StartDate != out[j].StartDate {
			return out[i].StartDate > out[j].StartDate
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].SourceID < out[j].SourceID
	})
	return out
}

func fetchAverageOutsideTemp(CarID int, parsedStartDate string, parsedEndDate string, unitsTemperature string) (*float64, error) {
	query := `
		WITH samples AS (
			SELECT drives.outside_temp_avg AS temp
			FROM drives
			WHERE drives.car_id = $1
				AND drives.end_date IS NOT NULL
				AND drives.outside_temp_avg IS NOT NULL`
	queryParams := []any{CarID}
	paramIndex := 2
	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND drives.start_date >= $%d", paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND drives.end_date < $%d", paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}

	query += `
			UNION ALL
			SELECT charging_processes.outside_temp_avg AS temp
			FROM charging_processes
			WHERE charging_processes.car_id = $1
				AND charging_processes.end_date IS NOT NULL
				AND charging_processes.outside_temp_avg IS NOT NULL`
	chargeParamIndex := 2
	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND charging_processes.start_date >= $%d", chargeParamIndex)
		chargeParamIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND charging_processes.end_date < $%d", chargeParamIndex)
	}

	query += `
		)
		SELECT AVG(temp)::float8
		FROM samples;`

	var value sql.NullFloat64
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(queryCtx, query, queryParams...).Scan(&value); err != nil {
		return nil, err
	}
	if !value.Valid {
		return nil, nil
	}
	if strings.EqualFold(unitsTemperature, "F") {
		value.Float64 = celsiusToFahrenheit(value.Float64)
	}
	return &value.Float64, nil
}

func fetchGrossConsumptionSummary(driveSummary *DriveHistorySummary, chargeSummary *ChargeHistorySummary) *GrossConsumptionSummary {
	if driveSummary == nil || chargeSummary == nil || driveSummary.TotalDistance <= 0 {
		return nil
	}

	energyUsed := 0.0
	dataComplete := false
	switch {
	case chargeSummary.TotalEnergyUsed != nil && *chargeSummary.TotalEnergyUsed > 0:
		energyUsed = *chargeSummary.TotalEnergyUsed
		dataComplete = true
	case chargeSummary.TotalEnergyAdded > 0:
		energyUsed = chargeSummary.TotalEnergyAdded
	default:
		return nil
	}

	value := energyUsed / driveSummary.TotalDistance * 1000.0
	return &GrossConsumptionSummary{
		AverageConsumptionGross: &value,
		DataComplete:            boolPointer(dataComplete),
	}
}

// fetchStatisticsSummary builds TeslaMate-style statistics for the selected period.
// Net consumption (Wh/km or Wh/mi): energy removed from the traction battery per distance, inferred from rated range loss where data is complete.
// Gross consumption: wall-side or grid-side energy per distance (charge_energy_used preferred, else charge_energy_added) over the same distance window.
func fetchStatisticsSummary(
	CarID int,
	parsedStartDate string,
	parsedEndDate string,
	unitsLength string,
	unitsTemperature string,
	driveSummary *DriveHistorySummary,
	chargeSummary *ChargeHistorySummary,
) (*StatisticsSummary, error) {
	if driveSummary == nil && chargeSummary == nil {
		return &StatisticsSummary{}, nil
	}

	averageOutsideTemp, err := fetchAverageOutsideTemp(CarID, parsedStartDate, parsedEndDate, unitsTemperature)
	if err != nil {
		return nil, err
	}

	grossSummary := fetchGrossConsumptionSummary(driveSummary, chargeSummary)
	statistics := &StatisticsSummary{
		Coverage: HistorySummaryCoverage{
			StartDate: minSummaryDate(
				coverageStartDateDrive(driveSummary),
				coverageStartDateCharge(chargeSummary),
			),
			EndDate: maxSummaryDate(
				coverageEndDateDrive(driveSummary),
				coverageEndDateCharge(chargeSummary),
			),
		},
		AverageOutsideTemp: averageOutsideTemp,
	}

	if driveSummary != nil {
		statistics.DriveCount = driveSummary.DriveCount
		statistics.Trips = driveSummary.DriveCount
		statistics.TimeDrivenMin = driveSummary.TotalDurationMin
		statistics.Distance = driveSummary.TotalDistance
		statistics.MaxSpeed = driveSummary.MaxSpeed
		statistics.AverageSpeed = driveSummary.AverageSpeed
		statistics.AverageConsumptionNet = driveSummary.AverageConsumption
	}
	if chargeSummary != nil {
		statistics.ChargeCount = chargeSummary.ChargeCount
		statistics.TotalCost = chargeSummary.TotalCost
		if chargeSummary.TotalEnergyAdded > 0 {
			v := chargeSummary.TotalEnergyAdded
			statistics.EnergyAdded = &v
		}
		statistics.ChargingEfficiency = chargeSummary.ChargingEfficiency

		if chargeSummary.TotalEnergyUsed != nil {
			statistics.EnergyUsed = chargeSummary.TotalEnergyUsed
			if chargeSummary.ChargeCount > 0 {
				value := *chargeSummary.TotalEnergyUsed / float64(chargeSummary.ChargeCount)
				statistics.AverageEnergyUsedPerCharge = &value
			}
		} else if chargeSummary.TotalEnergyAdded > 0 {
			value := chargeSummary.TotalEnergyAdded
			statistics.EnergyUsed = &value
			if chargeSummary.ChargeCount > 0 {
				perCharge := chargeSummary.TotalEnergyAdded / float64(chargeSummary.ChargeCount)
				statistics.AverageEnergyUsedPerCharge = &perCharge
			}
		}

		if chargeSummary.TotalCost != nil && chargeSummary.TotalEnergyAdded > 0 {
			value := *chargeSummary.TotalCost / chargeSummary.TotalEnergyAdded
			statistics.AverageCostPerKwh = &value
		}
	}

	if statistics.TotalCost != nil && statistics.Distance > 0 {
		value := *statistics.TotalCost / statistics.Distance * 100.0
		statistics.AverageCostPer100Distance = &value
	}

	if grossSummary != nil {
		statistics.AverageConsumptionGross = grossSummary.AverageConsumptionGross
		statistics.DataComplete = grossSummary.DataComplete
	}

	if statistics.AverageConsumptionNet != nil && statistics.AverageConsumptionGross != nil && *statistics.AverageConsumptionGross > 0 {
		drivingEfficiency := *statistics.AverageConsumptionNet / *statistics.AverageConsumptionGross
		consumptionOverhead := 1.0 - drivingEfficiency
		statistics.DrivingEfficiency = &drivingEfficiency
		statistics.ConsumptionOverhead = &consumptionOverhead
	}

	return statistics, nil
}

func fetchStateSummary(CarID int, parsedStartDate string, parsedEndDate string) (*StateSummary, error) {
	breakdown, coverage, totalDuration, err := fetchStateBreakdown(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		return nil, err
	}

	var (
		currentState    sql.NullString
		lastStateChange sql.NullString
	)
	if err := db.QueryRow(`
		SELECT
			states.state::text,
			states.start_date
		FROM states
		WHERE states.car_id = $1
		ORDER BY states.start_date DESC
		LIMIT 1;`, CarID).Scan(&currentState, &lastStateChange); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	parkedShare, err := fetchParkedShare(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		return nil, err
	}

	if len(breakdown) == 0 && !currentState.Valid {
		return nil, nil
	}

	summary := &StateSummary{
		Coverage:        coverage,
		CurrentState:    stringPointer(currentState),
		LastStateChange: timeZoneStringPointer(lastStateChange),
		ParkedShare:     parkedShare,
		StateBreakdown:  breakdown,
	}
	if totalDuration == 0 && len(summary.StateBreakdown) == 0 {
		summary.StateBreakdown = []StateBreakdown{}
	}
	return summary, nil
}

func fetchStateTimeline(CarID int, parsedStartDate string, parsedEndDate string, page int, show int) ([]StateTimelineItem, error) {
	baseQuery, queryParams := buildStateTimelineBaseQuery(CarID, parsedStartDate, parsedEndDate)
	offset := (page - 1) * show
	query := baseQuery + fmt.Sprintf(`
		SELECT
			timeline_id,
			state,
			start_date,
			end_date,
			duration_min
		FROM timeline
		ORDER BY start_date DESC
		LIMIT $%d OFFSET $%d;`, len(queryParams)+1, len(queryParams)+2)
	queryParams = append(queryParams, show, offset)

	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]StateTimelineItem, 0)
	for rows.Next() {
		var (
			item      StateTimelineItem
			startDate string
			endDate   sql.NullString
		)
		if err := rows.Scan(&item.TimelineID, &item.State, &startDate, &endDate, &item.DurationMin); err != nil {
			return nil, err
		}
		item.StartDate = getTimeInTimeZone(startDate)
		item.EndDate = timeZoneStringPointer(endDate)
		item.DurationStr = formatDurationMinutes(item.DurationMin)
		item.IsOpen = !endDate.Valid || endDate.String == ""
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func fetchStateBreakdown(CarID int, parsedStartDate string, parsedEndDate string) ([]StateBreakdown, HistorySummaryCoverage, int, error) {
	baseQuery, queryParams := buildStateTimelineBaseQuery(CarID, parsedStartDate, parsedEndDate)

	var coverageStart sql.NullString
	var coverageEnd sql.NullString
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(queryCtx, baseQuery+`
		SELECT
			MIN(start_date),
			MAX(COALESCE(end_date, NOW() AT TIME ZONE 'UTC'))
		FROM timeline;`, queryParams...).Scan(&coverageStart, &coverageEnd); err != nil {
		return nil, HistorySummaryCoverage{}, 0, err
	}

	rows, err := db.QueryContext(queryCtx, baseQuery+`
		SELECT
			state,
			COUNT(*)::int AS session_count,
			COALESCE(SUM(duration_min), 0)::int AS total_duration_min
		FROM timeline
		GROUP BY state
		ORDER BY total_duration_min DESC, state ASC;`, queryParams...)
	if err != nil {
		return nil, HistorySummaryCoverage{}, 0, err
	}
	defer rows.Close()

	breakdown := make([]StateBreakdown, 0)
	totalDuration := 0
	for rows.Next() {
		var item StateBreakdown
		if err := rows.Scan(&item.State, &item.SessionCount, &item.DurationMin); err != nil {
			return nil, HistorySummaryCoverage{}, 0, err
		}
		totalDuration += item.DurationMin
		breakdown = append(breakdown, item)
	}
	if err := rows.Err(); err != nil {
		return nil, HistorySummaryCoverage{}, 0, err
	}

	if totalDuration > 0 {
		for index := range breakdown {
			breakdown[index].Share = float64(breakdown[index].DurationMin) / float64(totalDuration)
		}
	}

	return breakdown, HistorySummaryCoverage{
		StartDate: timeZoneStringPointer(coverageStart),
		EndDate:   timeZoneStringPointer(coverageEnd),
	}, totalDuration, nil
}

func fetchParkedShare(CarID int, parsedStartDate string, parsedEndDate string) (*float64, error) {
	query := `
		SELECT
			CASE
				WHEN MIN(drives.start_date) IS NULL OR MAX(COALESCE(drives.end_date, NOW() AT TIME ZONE 'UTC')) IS NULL THEN NULL
				WHEN EXTRACT(EPOCH FROM (MAX(COALESCE(drives.end_date, NOW() AT TIME ZONE 'UTC')) - MIN(drives.start_date))) <= 0 THEN NULL
				ELSE GREATEST(
					1.0 - (
						COALESCE(SUM(GREATEST(COALESCE(drives.duration_min, 0), 0)), 0) /
						(EXTRACT(EPOCH FROM (MAX(COALESCE(drives.end_date, NOW() AT TIME ZONE 'UTC')) - MIN(drives.start_date))) / 60.0)
					),
					0
				)
			END AS parked_share
		FROM drives
		WHERE drives.car_id = $1`

	queryParams := []any{CarID}
	query, queryParams, _ = appendStateTimelineDateFilters(query, queryParams, 2, "drives.start_date", "drives.end_date", parsedStartDate, parsedEndDate)

	var value sql.NullFloat64
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(queryCtx, query, queryParams...).Scan(&value); err != nil {
		return nil, err
	}
	if !value.Valid {
		return nil, nil
	}
	return &value.Float64, nil
}

func buildStateTimelineBaseQuery(CarID int, parsedStartDate string, parsedEndDate string) (string, []any) {
	queryParams := []any{CarID}
	paramIndex := 2

	stateSegment := `
		SELECT
			CONCAT('state-', states.id) AS timeline_id,
			states.state::text AS state,
			states.start_date,
			states.end_date,
			GREATEST(
				COALESCE(EXTRACT(EPOCH FROM (COALESCE(states.end_date, NOW() AT TIME ZONE 'UTC') - states.start_date)) / 60, 0),
				0
			)::int AS duration_min
		FROM states
		WHERE states.car_id = $1`
	stateSegment, queryParams, paramIndex = appendStateTimelineDateFilters(stateSegment, queryParams, paramIndex, "states.start_date", "states.end_date", parsedStartDate, parsedEndDate)

	driveSegment := `
		SELECT
			CONCAT('drive-', drives.id) AS timeline_id,
			'driving' AS state,
			drives.start_date,
			drives.end_date,
			GREATEST(
				COALESCE(drives.duration_min, EXTRACT(EPOCH FROM (COALESCE(drives.end_date, NOW() AT TIME ZONE 'UTC') - drives.start_date)) / 60),
				0
			)::int AS duration_min
		FROM drives
		WHERE drives.car_id = $1`
	driveSegment, queryParams, paramIndex = appendStateTimelineDateFilters(driveSegment, queryParams, paramIndex, "drives.start_date", "drives.end_date", parsedStartDate, parsedEndDate)

	chargeSegment := `
		SELECT
			CONCAT('charge-', charging_processes.id) AS timeline_id,
			'charging' AS state,
			charging_processes.start_date,
			charging_processes.end_date,
			GREATEST(
				COALESCE(charging_processes.duration_min, EXTRACT(EPOCH FROM (COALESCE(charging_processes.end_date, NOW() AT TIME ZONE 'UTC') - charging_processes.start_date)) / 60),
				0
			)::int AS duration_min
		FROM charging_processes
		WHERE charging_processes.car_id = $1`
	chargeSegment, queryParams, paramIndex = appendStateTimelineDateFilters(chargeSegment, queryParams, paramIndex, "charging_processes.start_date", "charging_processes.end_date", parsedStartDate, parsedEndDate)

	updateSegment := `
		SELECT
			CONCAT('update-', updates.id) AS timeline_id,
			'updating' AS state,
			updates.start_date,
			updates.end_date,
			GREATEST(
				COALESCE(EXTRACT(EPOCH FROM (COALESCE(updates.end_date, NOW() AT TIME ZONE 'UTC') - updates.start_date)) / 60, 0),
				0
			)::int AS duration_min
		FROM updates
		WHERE updates.car_id = $1`
	updateSegment, queryParams, _ = appendStateTimelineDateFilters(updateSegment, queryParams, paramIndex, "updates.start_date", "updates.end_date", parsedStartDate, parsedEndDate)

	return fmt.Sprintf(`
		WITH timeline AS (
			%s
			UNION ALL
			%s
			UNION ALL
			%s
			UNION ALL
			%s
		)`, stateSegment, driveSegment, chargeSegment, updateSegment), queryParams
}

func appendStateTimelineDateFilters(
	query string,
	queryParams []any,
	paramIndex int,
	startExpr string,
	endExpr string,
	parsedStartDate string,
	parsedEndDate string,
) (string, []any, int) {
	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND %s >= $%d", startExpr, paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND COALESCE(%s, NOW() AT TIME ZONE 'UTC') < $%d", endExpr, paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}
	return query, queryParams, paramIndex
}

func boolPointer(value bool) *bool {
	return &value
}

func stringPointer(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
}

func minSummaryDate(dates ...*string) *string {
	var best *string
	for _, d := range dates {
		if d == nil || *d == "" {
			continue
		}
		if best == nil || *d < *best {
			best = d
		}
	}
	return best
}

func maxSummaryDate(dates ...*string) *string {
	var best *string
	for _, d := range dates {
		if d == nil || *d == "" {
			continue
		}
		if best == nil || *d > *best {
			best = d
		}
	}
	return best
}

func coverageStartDateDrive(d *DriveHistorySummary) *string {
	if d == nil {
		return nil
	}
	return d.Coverage.StartDate
}

func coverageStartDateCharge(c *ChargeHistorySummary) *string {
	if c == nil {
		return nil
	}
	return c.Coverage.StartDate
}

func coverageEndDateDrive(d *DriveHistorySummary) *string {
	if d == nil {
		return nil
	}
	return d.Coverage.EndDate
}

func coverageEndDateCharge(c *ChargeHistorySummary) *string {
	if c == nil {
		return nil
	}
	return c.Coverage.EndDate
}

func formatDurationMinutes(minutes int) string {
	if minutes < 0 {
		minutes = 0
	}
	h := minutes / 60
	m := minutes % 60
	return fmt.Sprintf("%d:%02d", h, m)
}
