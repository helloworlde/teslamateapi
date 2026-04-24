package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type ParkingStateBreakdown struct {
	State        string  `json:"state"`
	SessionCount int     `json:"session_count"`
	DurationMin  int     `json:"duration_min"`
	Share        float64 `json:"share"`
}

type ParkingHistorySummary struct {
	Coverage           HistorySummaryCoverage  `json:"coverage"`
	SessionCount       int                     `json:"session_count"`
	TotalDurationMin   int                     `json:"total_duration_min"`
	AverageDurationMin *float64                `json:"average_duration_min"`
	LongestDurationMin *int                    `json:"longest_duration_min"`
	DominantState      *string                 `json:"dominant_state"`
	StateBreakdown     []ParkingStateBreakdown `json:"state_breakdown"`
}

type ParkingPeriod struct {
	ParkingID   int     `json:"parking_id"`
	State       string  `json:"state"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date"`
	DurationMin int     `json:"duration_min"`
	DurationStr string  `json:"duration_str"`
	IsOpen      bool    `json:"is_open"`
}

type TeslaMateParkingFilters struct {
	StartDate *string  `json:"start_date"`
	EndDate   *string  `json:"end_date"`
	States    []string `json:"states,omitempty"`
	Page      int      `json:"page"`
	Show      int      `json:"show"`
}

type TeslaMateParkingData struct {
	Car            TeslaMateSummaryCar     `json:"car"`
	Filters        TeslaMateParkingFilters `json:"filters"`
	Parking        []ParkingPeriod         `json:"parking"`
	TeslaMateUnits TeslaMateSummaryUnits   `json:"units"`
}

type TeslaMateParkingJSONData struct {
	Data TeslaMateParkingData `json:"data"`
}

func TeslaMateAPICarsParkingSummaryV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsParkingSummaryV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load parking summary.", err.Error())
		return
	}

	summary, err := fetchParkingHistorySummary(CarID, parsedStartDate, parsedEndDate, nil)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load parking summary.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.ParkingSummary = summary

	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"parking_summary": data.ParkingSummary,
	}))
}

func TeslaMateAPICarsParkingV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsParkingV1"

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

	parkingStates, err := parseParkingStates(c.Query("states"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid parking parameter.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load parking history.", err.Error())
		return
	}

	parking, err := fetchParkingPeriods(CarID, parsedStartDate, parsedEndDate, parkingStates, page, show)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load parking history.", err.Error())
		return
	}

	jsonData := TeslaMateParkingJSONData{
		Data: TeslaMateParkingData{
			Car: TeslaMateSummaryCar{
				CarID:   CarID,
				CarName: carName,
			},
			Filters: TeslaMateParkingFilters{
				StartDate: summaryFilterDate(parsedStartDate),
				EndDate:   summaryFilterDate(parsedEndDate),
				States:    parkingStates,
				Page:      page,
				Show:      show,
			},
			Parking: parking,
			TeslaMateUnits: TeslaMateSummaryUnits{
				UnitsLength:      unitsLength,
				UnitsTemperature: unitsTemperature,
			},
		},
	}

	TeslaMateAPIHandleSuccessResponse(c, actionName, jsonData)
}

func parseParkingStates(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	allowed := map[string]bool{
		"online":  true,
		"offline": true,
		"asleep":  true,
	}
	result := make([]string, 0, 3)
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		state := strings.ToLower(strings.TrimSpace(part))
		if state == "" {
			continue
		}
		if !allowed[state] {
			return nil, fmt.Errorf("unsupported state %q", state)
		}
		if !seen[state] {
			result = append(result, state)
			seen[state] = true
		}
	}
	return result, nil
}

func fetchParkingPeriods(CarID int, parsedStartDate string, parsedEndDate string, parkingStates []string, page int, show int) ([]ParkingPeriod, error) {
	offset := (page - 1) * show
	query := `
		SELECT
			states.id,
			states.state::text,
			states.start_date,
			states.end_date,
			GREATEST(
				COALESCE(EXTRACT(EPOCH FROM (COALESCE(states.end_date, NOW() AT TIME ZONE 'UTC') - states.start_date)) / 60, 0),
				0
			)::int AS duration_min
		FROM states
		WHERE states.car_id = $1`

	queryParams := []any{CarID}
	paramIndex := 2

	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND states.start_date >= $%d", paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND COALESCE(states.end_date, NOW() AT TIME ZONE 'UTC') <= $%d", paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}
	if len(parkingStates) > 0 {
		query += fmt.Sprintf(" AND states.state::text = ANY($%d)", paramIndex)
		queryParams = append(queryParams, pq.Array(parkingStates))
		paramIndex++
	}

	query += fmt.Sprintf(`
		ORDER BY states.start_date DESC
		LIMIT $%d OFFSET $%d;`, paramIndex, paramIndex+1)
	queryParams = append(queryParams, show, offset)

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ParkingPeriod, 0)
	for rows.Next() {
		var (
			item      ParkingPeriod
			startDate string
			endDate   sql.NullString
		)
		if err := rows.Scan(&item.ParkingID, &item.State, &startDate, &endDate, &item.DurationMin); err != nil {
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

func fetchParkingHistorySummary(CarID int, parsedStartDate string, parsedEndDate string, parkingStates []string) (*ParkingHistorySummary, error) {
	baseQuery := `
		WITH filtered_states AS (
			SELECT
				states.state::text AS state,
				states.start_date,
				COALESCE(states.end_date, NOW() AT TIME ZONE 'UTC') AS effective_end_date,
				GREATEST(
					COALESCE(EXTRACT(EPOCH FROM (COALESCE(states.end_date, NOW() AT TIME ZONE 'UTC') - states.start_date)) / 60, 0),
					0
				)::int AS duration_min
			FROM states
			WHERE states.car_id = $1`

	queryParams := []any{CarID}
	paramIndex := 2

	if parsedStartDate != "" {
		baseQuery += fmt.Sprintf(" AND states.start_date >= $%d", paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		baseQuery += fmt.Sprintf(" AND COALESCE(states.end_date, NOW() AT TIME ZONE 'UTC') <= $%d", paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}
	if len(parkingStates) > 0 {
		baseQuery += fmt.Sprintf(" AND states.state::text = ANY($%d)", paramIndex)
		queryParams = append(queryParams, pq.Array(parkingStates))
		paramIndex++
	}
	baseQuery += `
		)`

	var (
		sessionCount       int
		totalDurationMin   int
		averageDurationMin sql.NullFloat64
		longestDurationMin sql.NullInt64
		coverageStart      sql.NullString
		coverageEnd        sql.NullString
	)

	overallQuery := baseQuery + `
		SELECT
			COUNT(*) AS session_count,
			COALESCE(SUM(duration_min), 0) AS total_duration_min,
			CASE WHEN COUNT(*) > 0 THEN AVG(duration_min)::float8 ELSE NULL END AS average_duration_min,
			MAX(NULLIF(duration_min, 0)) AS longest_duration_min,
			MIN(start_date) AS coverage_start,
			MAX(effective_end_date) AS coverage_end
		FROM filtered_states;`

	if err := db.QueryRow(overallQuery, queryParams...).Scan(
		&sessionCount,
		&totalDurationMin,
		&averageDurationMin,
		&longestDurationMin,
		&coverageStart,
		&coverageEnd,
	); err != nil {
		return nil, err
	}
	if sessionCount == 0 {
		return nil, nil
	}

	breakdownQuery := baseQuery + `
		SELECT
			state,
			COUNT(*) AS session_count,
			COALESCE(SUM(duration_min), 0) AS total_duration_min
		FROM filtered_states
		GROUP BY state
		ORDER BY total_duration_min DESC, state ASC;`

	rows, err := db.Query(breakdownQuery, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stateBreakdown := make([]ParkingStateBreakdown, 0)
	var dominantState *string
	for rows.Next() {
		var (
			state       string
			itemCount   int
			durationMin int
		)
		if err := rows.Scan(&state, &itemCount, &durationMin); err != nil {
			return nil, err
		}
		if dominantState == nil {
			stateCopy := state
			dominantState = &stateCopy
		}
		share := 0.0
		if totalDurationMin > 0 {
			share = float64(durationMin) / float64(totalDurationMin)
		}
		stateBreakdown = append(stateBreakdown, ParkingStateBreakdown{
			State:        state,
			SessionCount: itemCount,
			DurationMin:  durationMin,
			Share:        share,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ParkingHistorySummary{
		Coverage: HistorySummaryCoverage{
			StartDate: timeZoneStringPointer(coverageStart),
			EndDate:   timeZoneStringPointer(coverageEnd),
		},
		SessionCount:       sessionCount,
		TotalDurationMin:   totalDurationMin,
		AverageDurationMin: floatPointer(averageDurationMin),
		LongestDurationMin: intPointer(longestDurationMin),
		DominantState:      dominantState,
		StateBreakdown:     stateBreakdown,
	}, nil
}
