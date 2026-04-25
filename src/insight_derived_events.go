package main

import (
	"database/sql"
	"fmt"
	"time"
)

func driveTripThresholds(unitsLength string) (lowSpeedKmh, congestionKmh, minCongestionDistKm float64) {
	lowSpeedKmh = insightLowSpeedKmhThreshold
	congestionKmh = insightCongestionKmhThreshold
	minCongestionDistKm = insightCongestionMinDistanceKm
	if unitsLength == "mi" {
		lowSpeedKmh = insightLowSpeedMphThreshold * 1.609344
		congestionKmh = insightCongestionMphThreshold * 1.609344
		minCongestionDistKm = insightCongestionMinDistanceMi * 1.609344
	}
	return lowSpeedKmh, congestionKmh, minCongestionDistKm
}

func insightSortTime(s string) time.Time {
	t, err := time.Parse(dbTimestampFormat, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func appendInsightSort(ev insightEventInternal, sortID int) insightEventInternal {
	ev.SortType = ev.Event.Type
	ev.SortID = sortID
	return ev
}

func fetchLowSpeedTripInsightEvents(CarID int, parsedStartDate, parsedEndDate, unitsLength string) ([]insightEventInternal, error) {
	lowKmh, _, _ := driveTripThresholds(unitsLength)
	q := `
		SELECT
			drives.id,
			drives.start_date,
			drives.end_date,
			drives.duration_min,
			drives.distance,
			drives.distance / NULLIF(drives.duration_min::float8 / 60.0, 0) AS avg_speed_kmh
		FROM drives
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND drives.duration_min >= $2
			AND drives.duration_min > 0
			AND COALESCE(drives.distance, 0) > 0
			AND (drives.distance / (drives.duration_min::float8 / 60.0)) < $3`
	params := []any{CarID, insightLowSpeedMinDurationMin, lowKmh}
	idx := 4
	q, params, idx = appendSummaryDateFilters(q, params, idx, "drives", parsedStartDate, parsedEndDate)
	q += ` ORDER BY drives.start_date DESC;`

	rows, err := db.Query(q, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]insightEventInternal, 0)
	for rows.Next() {
		var (
			id           int
			startS, endS string
			dur          int
			dist         float64
			avgKmh       float64
		)
		if err := rows.Scan(&id, &startS, &endS, &dur, &dist, &avgKmh); err != nil {
			return nil, err
		}
		dispDist := dist
		dispAvg := avgKmh
		if unitsLength == "mi" {
			dispDist = kilometersToMiles(dist)
			dispAvg = kilometersToMiles(avgKmh)
		}
		driveID := id
		startRFC := getTimeInTimeZone(startS)
		endRFC := getTimeInTimeZone(endS)
		ev := insightEventInternal{
			SortDate: insightSortTime(startS),
			Event: InsightEvent{
				EventID:     fmt.Sprintf("low-speed-trip-%d", id),
				Type:        "low_speed_trip",
				Severity:    "low",
				Title:       "Low-speed trip",
				Description: "Trip average speed stayed below the low-speed threshold while lasting at least 10 minutes.",
				StartDate:   startRFC,
				EndDate:     &endRFC,
				DriveID:     &driveID,
				Metrics: InsightEventMetrics{
					DurationMin: &dur,
					Distance:    &dispDist,
					AvgSpeed:    &dispAvg,
				},
			},
		}
		out = append(out, appendInsightSort(ev, id))
	}
	return out, rows.Err()
}

func fetchCongestionLikeTripInsightEvents(CarID int, parsedStartDate, parsedEndDate, unitsLength string) ([]insightEventInternal, error) {
	_, congKmh, minDist := driveTripThresholds(unitsLength)
	q := `
		SELECT
			drives.id,
			drives.start_date,
			drives.end_date,
			drives.duration_min,
			drives.distance,
			drives.distance / NULLIF(drives.duration_min::float8 / 60.0, 0) AS avg_speed_kmh
		FROM drives
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND drives.duration_min > 0
			AND COALESCE(drives.distance, 0) >= $2
			AND (drives.distance / (drives.duration_min::float8 / 60.0)) < $3`
	params := []any{CarID, minDist, congKmh}
	idx := 4
	q, params, idx = appendSummaryDateFilters(q, params, idx, "drives", parsedStartDate, parsedEndDate)
	q += ` ORDER BY drives.start_date DESC;`

	rows, err := db.Query(q, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]insightEventInternal, 0)
	for rows.Next() {
		var (
			id           int
			startS, endS string
			dur          int
			dist         float64
			avgKmh       float64
		)
		if err := rows.Scan(&id, &startS, &endS, &dur, &dist, &avgKmh); err != nil {
			return nil, err
		}
		dispDist := dist
		dispAvg := avgKmh
		if unitsLength == "mi" {
			dispDist = kilometersToMiles(dist)
			dispAvg = kilometersToMiles(avgKmh)
		}
		driveID := id
		startRFC := getTimeInTimeZone(startS)
		endRFC := getTimeInTimeZone(endS)
		sev := "medium"
		if avgKmh < congKmh*0.5 {
			sev = "high"
		}
		ev := insightEventInternal{
			SortDate: insightSortTime(startS),
			Event: InsightEvent{
				EventID:     fmt.Sprintf("congestion-like-trip-%d", id),
				Type:        "congestion_like_trip",
				Severity:    sev,
				Title:       "Congestion-like trip",
				Description: "Low average speed over meaningful distance; may indicate heavy traffic or similar conditions.",
				StartDate:   startRFC,
				EndDate:     &endRFC,
				DriveID:     &driveID,
				Metrics: InsightEventMetrics{
					DurationMin: &dur,
					Distance:    &dispDist,
					AvgSpeed:    &dispAvg,
				},
			},
		}
		out = append(out, appendInsightSort(ev, id))
	}
	return out, rows.Err()
}

func fetchHighConsumptionDriveInsightEvents(CarID int, parsedStartDate, parsedEndDate, unitsLength string) ([]insightEventInternal, error) {
	q := `
		WITH filtered AS (
			SELECT
				drives.id,
				drives.start_date,
				drives.end_date,
				drives.duration_min,
				drives.distance,
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
			WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL`
	params := []any{CarID}
	idx := 2
	q, params, idx = appendSummaryDateFilters(q, params, idx, "drives", parsedStartDate, parsedEndDate)
	q += fmt.Sprintf(`
		),
		agg AS (
			SELECT AVG(consumption_net) AS period_avg
			FROM filtered
			WHERE consumption_net IS NOT NULL AND consumption_net > 0
		)
		SELECT f.id, f.start_date, f.end_date, f.duration_min, f.distance, f.consumption_net, a.period_avg
		FROM filtered f
		CROSS JOIN agg a
		WHERE f.consumption_net IS NOT NULL AND a.period_avg IS NOT NULL
			AND f.consumption_net > a.period_avg * %g
		ORDER BY f.start_date DESC;`, insightHighConsumptionMultiplier)

	rows, err := db.Query(q, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]insightEventInternal, 0)
	for rows.Next() {
		var (
			id              int
			startS, endS    string
			dur             int
			dist            float64
			consNet, period sql.NullFloat64
		)
		if err := rows.Scan(&id, &startS, &endS, &dur, &dist, &consNet, &period); err != nil {
			return nil, err
		}
		if !consNet.Valid || !period.Valid || period.Float64 <= 0 {
			continue
		}
		dispCons := consNet.Float64
		dispAvg := period.Float64
		dispDist := dist
		if unitsLength == "mi" {
			dispCons = whPerKmToWhPerMi(consNet.Float64)
			dispAvg = whPerKmToWhPerMi(period.Float64)
			dispDist = kilometersToMiles(dist)
		}
		delta := (dispCons/dispAvg - 1.0) * 100.0
		driveID := id
		startRFC := getTimeInTimeZone(startS)
		endRFC := getTimeInTimeZone(endS)
		ev := insightEventInternal{
			SortDate: insightSortTime(startS),
			Event: InsightEvent{
				EventID:     fmt.Sprintf("high-consumption-drive-%d", id),
				Type:        "high_consumption_drive",
				Severity:    "medium",
				Title:       "High consumption drive",
				Description: "This drive consumed significantly more energy than the selected-period average.",
				StartDate:   startRFC,
				EndDate:     &endRFC,
				DriveID:     &driveID,
				Metrics: InsightEventMetrics{
					Consumption:        &dispCons,
					AverageConsumption: &dispAvg,
					DeltaPercent:       &delta,
					DurationMin:        &dur,
					Distance:           &dispDist,
				},
			},
		}
		if delta >= 80 {
			ev.Event.Severity = "high"
		}
		out = append(out, appendInsightSort(ev, id))
	}
	return out, rows.Err()
}

func fetchLowEfficiencyChargeInsightEvents(CarID int, parsedStartDate, parsedEndDate string) ([]insightEventInternal, error) {
	q := `
		SELECT
			charging_processes.id,
			charging_processes.start_date,
			charging_processes.end_date,
			charging_processes.duration_min,
			charging_processes.charge_energy_added,
			charging_processes.charge_energy_used,
			CASE
				WHEN charging_processes.charge_energy_used > 0 AND charging_processes.charge_energy_added > 0
				THEN charging_processes.charge_energy_added / charging_processes.charge_energy_used
				ELSE NULL
			END AS efficiency
		FROM charging_processes
		WHERE charging_processes.car_id = $1
			AND charging_processes.end_date IS NOT NULL
			AND charging_processes.charge_energy_used > 0
			AND charging_processes.charge_energy_added > 0
			AND (
				(charging_processes.charge_energy_added / charging_processes.charge_energy_used) < $2
				OR (charging_processes.charge_energy_added / charging_processes.charge_energy_used) > $3
			)`
	params := []any{CarID, insightEfficiencyLow, insightEfficiencyHigh}
	idx := 4
	q, params, idx = appendSummaryDateFilters(q, params, idx, "charging_processes", parsedStartDate, parsedEndDate)
	q += ` ORDER BY charging_processes.start_date DESC;`

	rows, err := db.Query(q, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]insightEventInternal, 0)
	for rows.Next() {
		var (
			id           int
			startS, endS string
			dur          int
			added, used  float64
			eff          sql.NullFloat64
		)
		if err := rows.Scan(&id, &startS, &endS, &dur, &added, &used, &eff); err != nil {
			return nil, err
		}
		if !eff.Valid {
			continue
		}
		cid := id
		startRFC := getTimeInTimeZone(startS)
		endRFC := getTimeInTimeZone(endS)
		e := eff.Float64
		sev := "medium"
		if e < 0.45 || e > 1.15 {
			sev = "high"
		}
		ev := insightEventInternal{
			SortDate: insightSortTime(startS),
			Event: InsightEvent{
				EventID:     fmt.Sprintf("low-efficiency-charge-%d", id),
				Type:        "low_efficiency_charge",
				Severity:    sev,
				Title:       "Low charging efficiency",
				Description: "Charging energy added vs grid energy used is outside the expected efficiency band.",
				StartDate:   startRFC,
				EndDate:     &endRFC,
				ChargeID:    &cid,
				Metrics: InsightEventMetrics{
					Efficiency:  &e,
					EnergyAdded: &added,
					EnergyUsed:  &used,
					DurationMin: &dur,
				},
			},
		}
		out = append(out, appendInsightSort(ev, id))
	}
	return out, rows.Err()
}

func fetchAbnormalChargeInsightEvents(CarID int, parsedStartDate, parsedEndDate string) ([]insightEventInternal, error) {
	q := `
		SELECT
			charging_processes.id,
			charging_processes.start_date,
			charging_processes.end_date,
			charging_processes.duration_min,
			charging_processes.charge_energy_added,
			charging_processes.charge_energy_used,
			charging_processes.cost,
			CASE
				WHEN charging_processes.charge_energy_used > 0 AND charging_processes.charge_energy_added > 0
				THEN charging_processes.charge_energy_added / charging_processes.charge_energy_used
				ELSE NULL
			END AS efficiency
		FROM charging_processes
		WHERE charging_processes.car_id = $1
			AND charging_processes.end_date IS NOT NULL
			AND (
				COALESCE(charging_processes.charge_energy_added, 0) <= 0
				OR (charging_processes.duration_min >= $2 AND COALESCE(charging_processes.charge_energy_added, 0) < $3)
				OR (
					charging_processes.charge_energy_used > 0 AND charging_processes.charge_energy_added > 0
					AND (
						(charging_processes.charge_energy_added / charging_processes.charge_energy_used) < $4
						OR (charging_processes.charge_energy_added / charging_processes.charge_energy_used) > $5
					)
				)
				OR (charging_processes.cost IS NOT NULL AND charging_processes.cost < 0)
			)`
	params := []any{CarID, insightChargeLongSessionMin, insightChargeLongSessionKwh, insightEfficiencyLow, insightEfficiencyHigh}
	idx := 6
	q, params, idx = appendSummaryDateFilters(q, params, idx, "charging_processes", parsedStartDate, parsedEndDate)
	q += ` ORDER BY charging_processes.start_date DESC;`

	rows, err := db.Query(q, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]insightEventInternal, 0)
	for rows.Next() {
		var (
			id           int
			startS, endS string
			dur          int
			added, used  sql.NullFloat64
			cost         sql.NullFloat64
			eff          sql.NullFloat64
		)
		if err := rows.Scan(&id, &startS, &endS, &dur, &added, &used, &cost, &eff); err != nil {
			return nil, err
		}
		cid := id
		startRFC := getTimeInTimeZone(startS)
		endRFC := getTimeInTimeZone(endS)
		metrics := InsightEventMetrics{DurationMin: &dur}
		if added.Valid {
			v := added.Float64
			metrics.EnergyAdded = &v
		}
		if used.Valid {
			v := used.Float64
			metrics.EnergyUsed = &v
		}
		if eff.Valid {
			v := eff.Float64
			metrics.Efficiency = &v
		}
		ev := insightEventInternal{
			SortDate: insightSortTime(startS),
			Event: InsightEvent{
				EventID:     fmt.Sprintf("abnormal-charge-%d", id),
				Type:        "abnormal_charge",
				Severity:    "medium",
				Title:       "Abnormal charging session",
				Description: "Session matches one or more abnormal-charging heuristics (energy, duration, efficiency, or cost).",
				StartDate:   startRFC,
				EndDate:     &endRFC,
				ChargeID:    &cid,
				Metrics:     metrics,
			},
		}
		out = append(out, appendInsightSort(ev, id))
	}
	return out, rows.Err()
}

func fetchDeepDischargeDriveInsightEvents(CarID int, parsedStartDate, parsedEndDate string) ([]insightEventInternal, error) {
	q := `
		SELECT
			drives.id,
			drives.start_date,
			drives.end_date,
			COALESCE(end_position.usable_battery_level, end_position.battery_level) AS end_level
		FROM drives
		LEFT JOIN positions end_position ON drives.end_position_id = end_position.id
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND COALESCE(end_position.usable_battery_level, end_position.battery_level) IS NOT NULL
			AND COALESCE(end_position.usable_battery_level, end_position.battery_level) <= 12`
	params := []any{CarID}
	idx := 2
	q, params, idx = appendSummaryDateFilters(q, params, idx, "drives", parsedStartDate, parsedEndDate)
	q += ` ORDER BY drives.start_date DESC;`

	rows, err := db.Query(q, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]insightEventInternal, 0)
	for rows.Next() {
		var (
			id           int
			startS, endS string
			endLevel     int
		)
		if err := rows.Scan(&id, &startS, &endS, &endLevel); err != nil {
			return nil, err
		}
		driveID := id
		startRFC := getTimeInTimeZone(startS)
		endRFC := getTimeInTimeZone(endS)
		bl := endLevel
		sev := "medium"
		if endLevel <= 5 {
			sev = "high"
		}
		ev := insightEventInternal{
			SortDate: insightSortTime(startS),
			Event: InsightEvent{
				EventID:     fmt.Sprintf("deep-discharge-%d", id),
				Type:        "deep_discharge",
				Severity:    sev,
				Title:       "Deep discharge",
				Description: "Drive ended with a very low state of charge; repeated deep cycles can accelerate battery wear.",
				StartDate:   startRFC,
				EndDate:     &endRFC,
				DriveID:     &driveID,
				Metrics: InsightEventMetrics{
					BatteryLevel: &bl,
				},
			},
		}
		out = append(out, appendInsightSort(ev, id))
	}
	return out, rows.Err()
}
