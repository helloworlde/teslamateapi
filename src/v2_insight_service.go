package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// @name V2InsightService
type V2InsightService struct {
	drivingBuilder  V2DrivingBuilder
	chargingBuilder V2ChargingBuilder
}

func NewV2InsightService(drivingBuilder V2DrivingBuilder, chargingBuilder V2ChargingBuilder) V2InsightService {
	return V2InsightService{
		drivingBuilder:  drivingBuilder,
		chargingBuilder: chargingBuilder,
	}
}

func (s V2InsightService) BuildInsights(ctx context.Context, carIDParam string, timeRange V2TimeRange, category string, minSeverity string) (V2InsightResponse, error) {
	_, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2InsightResponse{}, err
	}

	// Compute baseline period internally — never modify the caller's timeRange.
	var prevStart, prevEnd time.Time
	if timeRange.PreviousStart != nil && timeRange.PreviousEnd != nil {
		prevStart = *timeRange.PreviousStart
		prevEnd = *timeRange.PreviousEnd
	} else {
		duration := timeRange.End.Sub(timeRange.Start)
		prevEnd = timeRange.Start
		prevStart = prevEnd.Add(-duration)
	}
	// currentTR strips comparison so downstream builders don't do redundant compare fetches.
	currentTR := timeRange
	currentTR.Compare = "none"
	currentTR.PreviousStart = nil
	currentTR.PreviousEnd = nil

	prevTR := V2TimeRange{
		Period:   timeRange.Period,
		Timezone: timeRange.Timezone,
		Compare:  "none",
		Start:    prevStart,
		End:      prevEnd,
	}

	currentPeriodStr := fmt.Sprintf("%s to %s",
		timeRange.Start.Format("2006-01-02"),
		timeRange.End.Format("2006-01-02"),
	)
	previousPeriodStr := fmt.Sprintf("%s to %s",
		prevStart.Format("2006-01-02"),
		prevEnd.Format("2006-01-02"),
	)

	var insights []V2Insight

	// Driving insights
	if s.drivingBuilder != nil && (category == "" || category == "driving" || category == "cost") {
		currentDriving, _, err := s.drivingBuilder.BuildDriving(ctx, carIDParam, currentTR)
		if err != nil {
			return V2InsightResponse{}, err
		}

		previousDriving, _, err := s.drivingBuilder.BuildDriving(ctx, carIDParam, prevTR)
		if err != nil {
			return V2InsightResponse{}, err
		}

		// Driving distance insight
		if (category == "" || category == "driving") &&
			currentDriving.Summary.DriveCount >= 5 && previousDriving.Summary.DriveCount >= 5 &&
			currentDriving.Summary.DistanceKM > 50 {

			insight := buildDistanceInsight(
				currentDriving.Summary.DistanceKM,
				previousDriving.Summary.DistanceKM,
				currentPeriodStr,
				previousPeriodStr,
			)
			if insight != nil {
				insights = append(insights, *insight)
			}
		}

		// Consumption insight
		if (category == "" || category == "driving") &&
			currentDriving.Summary.DriveCount >= 5 && previousDriving.Summary.DriveCount >= 5 &&
			currentDriving.Summary.DistanceKM > 50 &&
			currentDriving.Summary.AvgConsumptionWhPerKM != nil && previousDriving.Summary.AvgConsumptionWhPerKM != nil {

			insight := buildConsumptionInsight(
				*currentDriving.Summary.AvgConsumptionWhPerKM,
				*previousDriving.Summary.AvgConsumptionWhPerKM,
				currentPeriodStr,
				previousPeriodStr,
			)
			if insight != nil {
				insights = append(insights, *insight)
			}
		}
	}

	// Charging insights
	if s.chargingBuilder != nil && (category == "" || category == "charging" || category == "cost") {
		currentCharging, _, err := s.chargingBuilder.BuildCharging(ctx, carIDParam, currentTR)
		if err != nil {
			return V2InsightResponse{}, err
		}

		previousCharging, _, err := s.chargingBuilder.BuildCharging(ctx, carIDParam, prevTR)
		if err != nil {
			return V2InsightResponse{}, err
		}

		// Charging cost insight
		if (category == "" || category == "cost") &&
			currentCharging.Summary.SessionCount >= 3 && previousCharging.Summary.SessionCount >= 3 &&
			currentCharging.Summary.Cost != nil && previousCharging.Summary.Cost != nil {

			insight := buildChargingCostInsight(
				*currentCharging.Summary.Cost,
				*previousCharging.Summary.Cost,
				currentPeriodStr,
				previousPeriodStr,
			)
			if insight != nil {
				insights = append(insights, *insight)
			}
		}

		// DC charging ratio insight
		if (category == "" || category == "charging") &&
			currentCharging.Summary.SessionCount >= 3 && previousCharging.Summary.SessionCount >= 3 {

			insight := buildDCRatioInsight(
				currentCharging.Summary,
				previousCharging.Summary,
				currentPeriodStr,
				previousPeriodStr,
			)
			if insight != nil {
				insights = append(insights, *insight)
			}
		}
	}

	// Filter by min_severity
	if minSeverity == "warning" {
		var filtered []V2Insight
		for _, ins := range insights {
			if ins.Severity == "warning" {
				filtered = append(filtered, ins)
			}
		}
		insights = filtered
	}

	if insights == nil {
		insights = []V2Insight{}
	}

	return V2InsightResponse{Insights: insights, Total: len(insights)}, nil
}

func buildDistanceInsight(current, previous float64, currentPeriod, baselinePeriod string) *V2Insight {
	if previous == 0 {
		return nil
	}
	delta := current - previous
	deltaPercent := delta / previous * 100
	absPct := math.Abs(deltaPercent)

	severity := "info"
	if absPct >= 10 {
		severity = "warning"
	}

	id := "driving_distance_increased"
	if delta < 0 {
		id = "driving_distance_decreased"
	}
	title := fmt.Sprintf("Driving distance %s %.1f%% compared to previous period",
		changeWord(delta), math.Abs(deltaPercent))

	return &V2Insight{
		ID:             id,
		Category:       "driving",
		Severity:       severity,
		Title:          title,
		Description:    fmt.Sprintf("Current period: %.1f km. Previous period: %.1f km.", current, previous),
		CurrentPeriod:  currentPeriod,
		BaselinePeriod: baselinePeriod,
		Metrics: map[string]interface{}{
			"distance_km": current,
		},
		Evidence: map[string]interface{}{
			"current":       current,
			"previous":      previous,
			"delta":         delta,
			"delta_percent": deltaPercent,
		},
	}
}

func buildConsumptionInsight(current, previous float64, currentPeriod, baselinePeriod string) *V2Insight {
	if previous == 0 {
		return nil
	}
	delta := current - previous
	deltaPercent := delta / previous * 100
	absPct := math.Abs(deltaPercent)

	severity := "info"
	if absPct >= 10 {
		severity = "warning"
	}

	id := "consumption_increased"
	if delta < 0 {
		id = "consumption_decreased"
	}
	title := fmt.Sprintf("Average energy consumption %s %.1f%% compared to previous period",
		changeWord(delta), absPct)

	return &V2Insight{
		ID:             id,
		Category:       "driving",
		Severity:       severity,
		Title:          title,
		Description:    fmt.Sprintf("Current: %.1f Wh/km. Previous: %.1f Wh/km.", current, previous),
		CurrentPeriod:  currentPeriod,
		BaselinePeriod: baselinePeriod,
		Metrics: map[string]interface{}{
			"avg_consumption_wh_per_km": current,
		},
		Evidence: map[string]interface{}{
			"current":       current,
			"previous":      previous,
			"delta":         delta,
			"delta_percent": deltaPercent,
		},
	}
}

func buildChargingCostInsight(current, previous float64, currentPeriod, baselinePeriod string) *V2Insight {
	if previous == 0 {
		return nil
	}
	delta := current - previous
	deltaPercent := delta / previous * 100
	absPct := math.Abs(deltaPercent)

	if delta <= 0 {
		return nil // Only report increases
	}

	severity := "info"
	if absPct >= 10 {
		severity = "warning"
	}

	return &V2Insight{
		ID:             "charging_cost_increased",
		Category:       "cost",
		Severity:       severity,
		Title:          fmt.Sprintf("Charging cost increased %.1f%% compared to previous period", absPct),
		Description:    fmt.Sprintf("Current: %.2f. Previous: %.2f.", current, previous),
		CurrentPeriod:  currentPeriod,
		BaselinePeriod: baselinePeriod,
		Metrics: map[string]interface{}{
			"charging_cost": current,
		},
		Evidence: map[string]interface{}{
			"current":       current,
			"previous":      previous,
			"delta":         delta,
			"delta_percent": deltaPercent,
		},
	}
}

func buildDCRatioInsight(current, previous V2ChargingAnalyticsSummary, currentPeriod, baselinePeriod string) *V2Insight {
	if current.SessionCount == 0 || previous.SessionCount == 0 {
		return nil
	}
	currentRatio := float64(current.DCSessionCount) / float64(current.SessionCount)
	previousRatio := float64(previous.DCSessionCount) / float64(previous.SessionCount)

	if previousRatio == 0 {
		return nil
	}
	delta := currentRatio - previousRatio
	if delta <= 0 {
		return nil // Only report increases
	}
	deltaPercent := delta / previousRatio * 100
	absPct := math.Abs(deltaPercent)

	severity := "info"
	if absPct >= 10 {
		severity = "warning"
	}

	return &V2Insight{
		ID:             "dc_charging_ratio_increased",
		Category:       "charging",
		Severity:       severity,
		Title:          fmt.Sprintf("DC charging session ratio increased %.1f%% compared to previous period", absPct),
		Description:    fmt.Sprintf("Current DC ratio: %.1f%%. Previous: %.1f%%.", currentRatio*100, previousRatio*100),
		CurrentPeriod:  currentPeriod,
		BaselinePeriod: baselinePeriod,
		Metrics: map[string]interface{}{
			"dc_session_ratio": currentRatio,
		},
		Evidence: map[string]interface{}{
			"current":       currentRatio,
			"previous":      previousRatio,
			"delta":         delta,
			"delta_percent": deltaPercent,
		},
	}
}

func changeWord(delta float64) string {
	if delta >= 0 {
		return "increased"
	}
	return "decreased"
}

func periodLabel(t time.Time) string {
	return t.Format("2006-01-02")
}

func parseIncludeList(param string) []string {
	if param == "" {
		return nil
	}
	parts := strings.Split(param, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
