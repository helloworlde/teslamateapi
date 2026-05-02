package main

import (
	"database/sql"
	"fmt"

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
		baseQuery += fmt.Sprintf(" AND COALESCE(states.end_date, NOW() AT TIME ZONE 'UTC') < $%d", paramIndex)
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

	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(queryCtx, overallQuery, queryParams...).Scan(
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
		return &ParkingHistorySummary{
			Coverage:       HistorySummaryCoverage{},
			StateBreakdown: []ParkingStateBreakdown{},
		}, nil
	}

	breakdownQuery := baseQuery + `
		SELECT
			state,
			COUNT(*) AS session_count,
			COALESCE(SUM(duration_min), 0) AS total_duration_min
		FROM filtered_states
		GROUP BY state
		ORDER BY total_duration_min DESC, state ASC;`

	rows, err := db.QueryContext(queryCtx, breakdownQuery, queryParams...)
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
