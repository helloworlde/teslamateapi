package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type GrossConsumptionSummary struct {
	AverageConsumptionGross *float64 `json:"average_consumption_gross"`
	DataComplete            *bool    `json:"data_complete"`
}

type StatisticsSummary struct {
	Coverage                   HistorySummaryCoverage `json:"coverage"`
	DriveCount                 int                    `json:"drive_count"`
	ChargeCount                int                    `json:"charge_count"`
	TimeDrivenMin              int                    `json:"time_driven_min"`
	Distance                   float64                `json:"distance"`
	MaxSpeed                   *int                   `json:"max_speed"`
	AverageOutsideTemp         *float64               `json:"average_outside_temp"`
	AverageConsumptionNet      *float64               `json:"average_consumption_net"`
	AverageConsumptionGross    *float64               `json:"average_consumption_gross"`
	DrivingEfficiency          *float64               `json:"driving_efficiency"`
	EnergyUsed                 *float64               `json:"energy_used"`
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

type TeslaMateStateTimelineFilters struct {
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Page      int     `json:"page"`
	Show      int     `json:"show"`
}

type TeslaMateStateTimelineData struct {
	Car            TeslaMateSummaryCar           `json:"car"`
	Filters        TeslaMateStateTimelineFilters `json:"filters"`
	Timeline       []StateTimelineItem           `json:"timeline"`
	TeslaMateUnits TeslaMateSummaryUnits         `json:"units"`
}

type TeslaMateStateTimelineJSONData struct {
	Data TeslaMateStateTimelineData `json:"data"`
}

func TeslaMateAPICarsStatisticsSummaryV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsStatisticsSummaryV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load statistics summary.", err.Error())
		return
	}

	driveSummary, err := fetchDriveHistorySummary(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load statistics summary.", err.Error())
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load statistics summary.", err.Error())
		return
	}
	statisticsSummary, err := fetchStatisticsSummary(CarID, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature, driveSummary, chargeSummary)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load statistics summary.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.StatisticsSummary = statisticsSummary

	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"statistics_summary": data.StatisticsSummary,
	}))
}

func TeslaMateAPICarsStateSummaryV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsStateSummaryV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load state summary.", err.Error())
		return
	}
	stateSummary, err := fetchStateSummary(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load state summary.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.StateSummary = stateSummary

	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"state_summary": data.StateSummary,
	}))
}

func TeslaMateAPICarsStateTimelineV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsStateTimelineV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	page := convertStringToInteger(c.DefaultQuery("page", "1"))
	show := convertStringToInteger(c.DefaultQuery("show", "100"))
	if page <= 0 {
		page = 1
	}
	if show <= 0 {
		show = 100
	}
	if show > 500 {
		show = 500
	}

	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load state timeline.", err.Error())
		return
	}

	timeline, err := fetchStateTimeline(CarID, parsedStartDate, parsedEndDate, page, show)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load state timeline.", err.Error())
		return
	}

	jsonData := TeslaMateStateTimelineJSONData{
		Data: TeslaMateStateTimelineData{
			Car: TeslaMateSummaryCar{
				CarID:   CarID,
				CarName: carName,
			},
			Filters: TeslaMateStateTimelineFilters{
				StartDate: summaryFilterDate(parsedStartDate),
				EndDate:   summaryFilterDate(parsedEndDate),
				Page:      page,
				Show:      show,
			},
			Timeline: timeline,
			TeslaMateUnits: TeslaMateSummaryUnits{
				UnitsLength:      unitsLength,
				UnitsTemperature: unitsTemperature,
			},
		},
	}

	TeslaMateAPIHandleSuccessResponse(c, actionName, jsonData)
}

func TeslaMateAPICarsChartStateDurationV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsChartStateDurationV1"
	TeslaMateAPIHandleChartCategoryResponse(c, actionName, "Unable to load state duration chart.", func(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) ([]SummaryCategoryValue, error) {
		return fetchStateDurationChart(CarID, parsedStartDate, parsedEndDate)
	}, "state_duration")
}

func TeslaMateAPIHandleChartCategoryResponse(
	c *gin.Context,
	actionName string,
	errorMessage string,
	fetch func(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) ([]SummaryCategoryValue, error),
	fieldName string,
) {
	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, errorMessage, err.Error())
		return
	}

	items, err := fetch(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, errorMessage, err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		fieldName: items,
	}))
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
		query += fmt.Sprintf(" AND drives.end_date <= $%d", paramIndex)
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
		query += fmt.Sprintf(" AND charging_processes.end_date <= $%d", chargeParamIndex)
	}

	query += `
		)
		SELECT AVG(temp)::float8
		FROM samples;`

	var value sql.NullFloat64
	if err := db.QueryRow(query, queryParams...).Scan(&value); err != nil {
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
		return nil, nil
	}

	averageOutsideTemp, err := fetchAverageOutsideTemp(CarID, parsedStartDate, parsedEndDate, unitsTemperature)
	if err != nil {
		return nil, err
	}

	grossSummary := fetchGrossConsumptionSummary(driveSummary, chargeSummary)
	statistics := &StatisticsSummary{
		Coverage: HistorySummaryCoverage{
			StartDate: minSummaryDate(
				coverageStartDate(driveSummary),
				coverageStartDate(chargeSummary),
			),
			EndDate: maxSummaryDate(
				coverageEndDate(driveSummary),
				coverageEndDate(chargeSummary),
			),
		},
		AverageOutsideTemp: averageOutsideTemp,
	}

	if driveSummary != nil {
		statistics.DriveCount = driveSummary.DriveCount
		statistics.TimeDrivenMin = driveSummary.TotalDurationMin
		statistics.Distance = driveSummary.TotalDistance
		statistics.MaxSpeed = driveSummary.MaxSpeed
		statistics.AverageConsumptionNet = driveSummary.AverageConsumption
	}
	if chargeSummary != nil {
		statistics.ChargeCount = chargeSummary.ChargeCount
		statistics.TotalCost = chargeSummary.TotalCost

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

	rows, err := db.Query(query, queryParams...)
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

func fetchStateDurationChart(CarID int, parsedStartDate string, parsedEndDate string) ([]SummaryCategoryValue, error) {
	breakdown, _, _, err := fetchStateBreakdown(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		return nil, err
	}

	result := make([]SummaryCategoryValue, 0, len(breakdown))
	for index, item := range breakdown {
		result = append(result, SummaryCategoryValue{
			ID:    fmt.Sprintf("state-duration-%d", index+1),
			Label: item.State,
			Value: float64(item.DurationMin),
		})
	}
	return result, nil
}

func fetchStateBreakdown(CarID int, parsedStartDate string, parsedEndDate string) ([]StateBreakdown, HistorySummaryCoverage, int, error) {
	baseQuery, queryParams := buildStateTimelineBaseQuery(CarID, parsedStartDate, parsedEndDate)

	var coverageStart sql.NullString
	var coverageEnd sql.NullString
	if err := db.QueryRow(baseQuery+`
		SELECT
			MIN(start_date),
			MAX(COALESCE(end_date, NOW() AT TIME ZONE 'UTC'))
		FROM timeline;`, queryParams...).Scan(&coverageStart, &coverageEnd); err != nil {
		return nil, HistorySummaryCoverage{}, 0, err
	}

	rows, err := db.Query(baseQuery+`
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
	if err := db.QueryRow(query, queryParams...).Scan(&value); err != nil {
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
		query += fmt.Sprintf(" AND COALESCE(%s, NOW() AT TIME ZONE 'UTC') <= $%d", endExpr, paramIndex)
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
