package main

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type InsightEventMetrics struct {
	SpeedBeforeKph  *float64 `json:"speed_before,omitempty"`
	SpeedAfterKph   *float64 `json:"speed_after,omitempty"`
	SpeedDropKph    *float64 `json:"speed_drop,omitempty"`
	DecelerationMS2 *float64 `json:"deceleration_ms2,omitempty"`
	PowerBeforeKw   *float64 `json:"power_before_kw,omitempty"`
	PowerAfterKw    *float64 `json:"power_after_kw,omitempty"`
	PowerDropKw     *float64 `json:"power_drop_kw,omitempty"`
	WakeDurationMin *float64 `json:"wake_duration_min,omitempty"`
	BatteryLevel    *int     `json:"battery_level,omitempty"`
}

type InsightEvent struct {
	EventID     string              `json:"event_id"`
	Type        string              `json:"type"`
	Severity    string              `json:"severity"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	StartDate   string              `json:"start_date"`
	EndDate     *string             `json:"end_date,omitempty"`
	DriveID     *int                `json:"drive_id,omitempty"`
	ChargeID    *int                `json:"charge_id,omitempty"`
	StateID     *int                `json:"state_id,omitempty"`
	Metrics     InsightEventMetrics `json:"metrics"`
}

type InsightSummary struct {
	Coverage               HistorySummaryCoverage `json:"coverage"`
	TotalEvents            int                    `json:"total_events"`
	HarshBrakeCount        int                    `json:"harsh_brake_count"`
	ChargePowerDropCount   int                    `json:"charge_power_drop_count"`
	SleepInterruptionCount int                    `json:"sleep_interruption_count"`
}

type InsightEventFilters struct {
	StartDate *string  `json:"start_date"`
	EndDate   *string  `json:"end_date"`
	Types     []string `json:"types,omitempty"`
	Page      int      `json:"page"`
	Show      int      `json:"show"`
}

type TeslaMateInsightEventsData struct {
	Car            TeslaMateSummaryCar   `json:"car"`
	Filters        InsightEventFilters   `json:"filters"`
	Events         []InsightEvent        `json:"events"`
	TeslaMateUnits TeslaMateSummaryUnits `json:"units"`
}

type TeslaMateInsightEventsJSONData struct {
	Data TeslaMateInsightEventsData `json:"data"`
}

type insightEventInternal struct {
	SortDate time.Time
	Event    InsightEvent
}

func TeslaMateAPICarsInsightSummaryV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsInsightSummaryV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load insight summary.", err.Error())
		return
	}

	summary, err := fetchInsightSummary(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load insight summary.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"insight_summary": summary,
	}))
}

func TeslaMateAPICarsInsightEventsV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsInsightEventsV1"

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

	types, err := parseInsightTypes(c.Query("types"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid insight parameter.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load insight events.", err.Error())
		return
	}

	events, err := fetchInsightEvents(CarID, parsedStartDate, parsedEndDate, unitsLength, types, page, show)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load insight events.", err.Error())
		return
	}

	jsonData := TeslaMateInsightEventsJSONData{
		Data: TeslaMateInsightEventsData{
			Car: TeslaMateSummaryCar{
				CarID:   CarID,
				CarName: carName,
			},
			Filters: InsightEventFilters{
				StartDate: summaryFilterDate(parsedStartDate),
				EndDate:   summaryFilterDate(parsedEndDate),
				Types:     types,
				Page:      page,
				Show:      show,
			},
			Events: events,
			TeslaMateUnits: TeslaMateSummaryUnits{
				UnitsLength:      unitsLength,
				UnitsTemperature: unitsTemperature,
			},
		},
	}

	TeslaMateAPIHandleSuccessResponse(c, actionName, jsonData)
}

func fetchInsightSummary(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) (*InsightSummary, error) {
	events, err := fetchInsightEvents(CarID, parsedStartDate, parsedEndDate, unitsLength, nil, 1, 10000)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, nil
	}

	var (
		coverageStart *string
		coverageEnd   *string
		summary       InsightSummary
	)

	for _, event := range events {
		summary.TotalEvents++
		switch event.Type {
		case "harsh_brake":
			summary.HarshBrakeCount++
		case "charge_power_drop":
			summary.ChargePowerDropCount++
		case "sleep_interruption":
			summary.SleepInterruptionCount++
		}

		startCopy := event.StartDate
		coverageStart = minSummaryDate(coverageStart, &startCopy)
		if event.EndDate != nil {
			coverageEnd = maxSummaryDate(coverageEnd, event.EndDate)
		} else {
			coverageEnd = maxSummaryDate(coverageEnd, &startCopy)
		}
	}

	summary.Coverage = HistorySummaryCoverage{
		StartDate: coverageStart,
		EndDate:   coverageEnd,
	}
	return &summary, nil
}

func fetchInsightEvents(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string, insightTypes []string, page int, show int) ([]InsightEvent, error) {
	filter := map[string]bool{
		"harsh_brake":        len(insightTypes) == 0,
		"charge_power_drop":  len(insightTypes) == 0,
		"sleep_interruption": len(insightTypes) == 0,
	}
	for _, item := range insightTypes {
		filter[item] = true
	}

	allEvents := make([]insightEventInternal, 0)
	if filter["harsh_brake"] {
		items, err := fetchHarshBrakeEvents(CarID, parsedStartDate, parsedEndDate, unitsLength)
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, items...)
	}
	if filter["charge_power_drop"] {
		items, err := fetchChargePowerDropEvents(CarID, parsedStartDate, parsedEndDate)
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, items...)
	}
	if filter["sleep_interruption"] {
		items, err := fetchSleepInterruptionEvents(CarID, parsedStartDate, parsedEndDate)
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, items...)
	}

	sort.Slice(allEvents, func(i int, j int) bool {
		return allEvents[i].SortDate.After(allEvents[j].SortDate)
	})

	offset := (page - 1) * show
	if offset >= len(allEvents) {
		return []InsightEvent{}, nil
	}
	end := offset + show
	if end > len(allEvents) {
		end = len(allEvents)
	}

	result := make([]InsightEvent, 0, end-offset)
	for _, item := range allEvents[offset:end] {
		result = append(result, item.Event)
	}
	return result, nil
}

func fetchHarshBrakeEvents(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) ([]insightEventInternal, error) {
	query := `
		WITH samples AS (
			SELECT
				drives.id AS drive_id,
				positions.id AS position_id,
				positions.date,
				COALESCE(positions.speed, 0)::float8 AS speed,
				LAG(positions.date) OVER (PARTITION BY drives.id ORDER BY positions.id) AS previous_date,
				LAG(COALESCE(positions.speed, 0)::float8) OVER (PARTITION BY drives.id ORDER BY positions.id) AS previous_speed
			FROM drives
			INNER JOIN positions ON positions.drive_id = drives.id
			WHERE drives.car_id = $1
				AND drives.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, _ = appendSummaryDateFilters(query, queryParams, paramIndex, "drives", parsedStartDate, parsedEndDate)
	query += `
		),
		candidates AS (
			SELECT
				drive_id,
				position_id,
				date AS event_date,
				previous_speed,
				speed,
				EXTRACT(EPOCH FROM (date - previous_date)) AS delta_sec,
				GREATEST(previous_speed - speed, 0) AS speed_drop
			FROM samples
			WHERE previous_date IS NOT NULL
				AND previous_speed IS NOT NULL
		)
		SELECT
			drive_id,
			position_id,
			event_date,
			previous_speed,
			speed,
			speed_drop,
			(speed_drop * 1000.0 / 3600.0) / NULLIF(delta_sec, 0) AS deceleration_ms2
		FROM candidates
		WHERE delta_sec BETWEEN 1 AND 10
			AND speed_drop >= 15
			AND ((speed_drop * 1000.0 / 3600.0) / NULLIF(delta_sec, 0)) >= 4.0
		ORDER BY event_date DESC;`

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]insightEventInternal, 0)
	for rows.Next() {
		var (
			driveID         int
			positionID      int
			eventDateString string
			speedBefore     float64
			speedAfter      float64
			speedDrop       float64
			decelerationMS2 float64
		)
		if err := rows.Scan(&driveID, &positionID, &eventDateString, &speedBefore, &speedAfter, &speedDrop, &decelerationMS2); err != nil {
			return nil, err
		}

		if unitsLength == "mi" {
			speedBefore = kilometersToMiles(speedBefore)
			speedAfter = kilometersToMiles(speedAfter)
			speedDrop = kilometersToMiles(speedDrop)
		}

		eventTime, err := time.Parse(dbTimestampFormat, eventDateString)
		if err != nil {
			return nil, err
		}
		severity := "medium"
		if decelerationMS2 >= 6.0 {
			severity = "high"
		}

		driveIDCopy := driveID
		speedBeforeCopy := speedBefore
		speedAfterCopy := speedAfter
		speedDropCopy := speedDrop
		decelerationCopy := decelerationMS2
		result = append(result, insightEventInternal{
			SortDate: eventTime,
			Event: InsightEvent{
				EventID:     fmt.Sprintf("harsh-brake-%d", positionID),
				Type:        "harsh_brake",
				Severity:    severity,
				Title:       "Harsh braking detected",
				Description: "Detected a short-interval speed drop consistent with hard braking during a drive.",
				StartDate:   getTimeInTimeZone(eventDateString),
				DriveID:     &driveIDCopy,
				Metrics: InsightEventMetrics{
					SpeedBeforeKph:  &speedBeforeCopy,
					SpeedAfterKph:   &speedAfterCopy,
					SpeedDropKph:    &speedDropCopy,
					DecelerationMS2: &decelerationCopy,
				},
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func fetchChargePowerDropEvents(CarID int, parsedStartDate string, parsedEndDate string) ([]insightEventInternal, error) {
	query := `
		WITH samples AS (
			SELECT
				charging_processes.id AS charge_id,
				charges.id AS sample_id,
				charges.date,
				COALESCE(charges.charger_power, 0)::float8 AS charger_power,
				COALESCE(charges.battery_level, 0) AS battery_level,
				LAG(COALESCE(charges.charger_power, 0)::float8) OVER (PARTITION BY charging_processes.id ORDER BY charges.id) AS previous_power,
				LAG(charges.date) OVER (PARTITION BY charging_processes.id ORDER BY charges.id) AS previous_date
			FROM charging_processes
			INNER JOIN charges ON charges.charging_process_id = charging_processes.id
			WHERE charging_processes.car_id = $1
				AND charging_processes.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, _ = appendSummaryDateFilters(query, queryParams, paramIndex, "charging_processes", parsedStartDate, parsedEndDate)
	query += `
		)
		SELECT
			charge_id,
			sample_id,
			date,
			previous_power,
			charger_power,
			previous_power - charger_power AS power_drop_kw,
			battery_level
		FROM samples
		WHERE previous_date IS NOT NULL
			AND EXTRACT(EPOCH FROM (date - previous_date)) BETWEEN 1 AND 1800
			AND previous_power >= 7
			AND (previous_power - charger_power) >= 5
			AND battery_level < 95
		ORDER BY date DESC;`

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]insightEventInternal, 0)
	for rows.Next() {
		var (
			chargeID        int
			sampleID        int
			eventDateString string
			powerBefore     float64
			powerAfter      float64
			powerDrop       float64
			batteryLevel    int
		)
		if err := rows.Scan(&chargeID, &sampleID, &eventDateString, &powerBefore, &powerAfter, &powerDrop, &batteryLevel); err != nil {
			return nil, err
		}

		eventTime, err := time.Parse(dbTimestampFormat, eventDateString)
		if err != nil {
			return nil, err
		}
		severity := "medium"
		if powerDrop >= 15 || powerAfter <= powerBefore*0.4 {
			severity = "high"
		}

		chargeIDCopy := chargeID
		powerBeforeCopy := powerBefore
		powerAfterCopy := powerAfter
		powerDropCopy := powerDrop
		batteryLevelCopy := batteryLevel
		result = append(result, insightEventInternal{
			SortDate: eventTime,
			Event: InsightEvent{
				EventID:     fmt.Sprintf("charge-power-drop-%d", sampleID),
				Type:        "charge_power_drop",
				Severity:    severity,
				Title:       "Charging power dropped unexpectedly",
				Description: "Detected a sharp charging-power drop before the session reached the normal end-of-charge taper range.",
				StartDate:   getTimeInTimeZone(eventDateString),
				ChargeID:    &chargeIDCopy,
				Metrics: InsightEventMetrics{
					PowerBeforeKw: &powerBeforeCopy,
					PowerAfterKw:  &powerAfterCopy,
					PowerDropKw:   &powerDropCopy,
					BatteryLevel:  &batteryLevelCopy,
				},
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func fetchSleepInterruptionEvents(CarID int, parsedStartDate string, parsedEndDate string) ([]insightEventInternal, error) {
	query := `
		WITH ordered_states AS (
			SELECT
				states.id,
				states.state::text AS state,
				states.start_date,
				states.end_date,
				LEAD(states.state::text) OVER (ORDER BY states.start_date) AS next_state,
				LEAD(states.start_date) OVER (ORDER BY states.start_date) AS next_start_date,
				LEAD(states.end_date) OVER (ORDER BY states.start_date) AS next_end_date,
				LEAD(states.state::text, 2) OVER (ORDER BY states.start_date) AS next_next_state,
				LEAD(states.start_date, 2) OVER (ORDER BY states.start_date) AS next_next_start_date
			FROM states
			WHERE states.car_id = $1`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, _ = appendStateTimelineDateFilters(query, queryParams, paramIndex, "states.start_date", "states.end_date", parsedStartDate, parsedEndDate)
	query += `
		)
		SELECT
			id,
			COALESCE(end_date, next_start_date) AS wake_start,
			COALESCE(next_end_date, next_next_start_date) AS wake_end,
			EXTRACT(EPOCH FROM (COALESCE(next_end_date, next_next_start_date) - COALESCE(end_date, next_start_date))) / 60.0 AS wake_duration_min
		FROM ordered_states
		WHERE state = 'asleep'
			AND end_date IS NOT NULL
			AND next_state IN ('online', 'offline')
			AND next_next_state = 'asleep'
			AND COALESCE(next_end_date, next_next_start_date) IS NOT NULL
			AND EXTRACT(EPOCH FROM (COALESCE(next_end_date, next_next_start_date) - COALESCE(end_date, next_start_date))) / 60.0 BETWEEN 1 AND 60
			AND NOT EXISTS (
				SELECT 1
				FROM drives
				WHERE drives.car_id = $1
					AND drives.start_date < COALESCE(next_end_date, next_next_start_date)
					AND COALESCE(drives.end_date, drives.start_date) > COALESCE(end_date, next_start_date)
			)
			AND NOT EXISTS (
				SELECT 1
				FROM charging_processes
				WHERE charging_processes.car_id = $1
					AND charging_processes.start_date < COALESCE(next_end_date, next_next_start_date)
					AND COALESCE(charging_processes.end_date, charging_processes.start_date) > COALESCE(end_date, next_start_date)
			)
			AND NOT EXISTS (
				SELECT 1
				FROM updates
				WHERE updates.car_id = $1
					AND updates.start_date < COALESCE(next_end_date, next_next_start_date)
					AND COALESCE(updates.end_date, updates.start_date) > COALESCE(end_date, next_start_date)
			)
		ORDER BY wake_start DESC;`

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]insightEventInternal, 0)
	for rows.Next() {
		var (
			stateID         int
			wakeStartString string
			wakeEnd         sql.NullString
			wakeDurationMin float64
		)
		if err := rows.Scan(&stateID, &wakeStartString, &wakeEnd, &wakeDurationMin); err != nil {
			return nil, err
		}

		eventTime, err := time.Parse(dbTimestampFormat, wakeStartString)
		if err != nil {
			return nil, err
		}
		severity := "medium"
		if wakeDurationMin >= 20 {
			severity = "high"
		}

		stateIDCopy := stateID
		wakeDurationCopy := wakeDurationMin
		result = append(result, insightEventInternal{
			SortDate: eventTime,
			Event: InsightEvent{
				EventID:     fmt.Sprintf("sleep-interruption-%d", stateID),
				Type:        "sleep_interruption",
				Severity:    severity,
				Title:       "Vehicle woke during sleep",
				Description: "Detected a short online/offline interruption between asleep states without a matching drive, charge, or update session.",
				StartDate:   getTimeInTimeZone(wakeStartString),
				EndDate:     timeZoneStringPointer(wakeEnd),
				StateID:     &stateIDCopy,
				Metrics: InsightEventMetrics{
					WakeDurationMin: &wakeDurationCopy,
				},
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func parseInsightTypes(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	allowed := map[string]bool{
		"harsh_brake":        true,
		"charge_power_drop":  true,
		"sleep_interruption": true,
	}

	result := make([]string, 0, 3)
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		value := strings.TrimSpace(strings.ToLower(part))
		if value == "" {
			continue
		}
		if !allowed[value] {
			return nil, fmt.Errorf("unsupported insight type %q", value)
		}
		if !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}
	return result, nil
}
