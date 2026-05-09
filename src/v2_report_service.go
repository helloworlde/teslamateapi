package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type V2ReportService struct {
	summaryBuilder    V2SummaryBuilder
	drivingBuilder    V2DrivingBuilder
	chargingBuilder   V2ChargingBuilder
	updateBuilder     V2UpdateBuilder
	parkingBuilder    V2ParkingBuilder
	batteryBuilder    V2BatteryBuilder
	efficiencyBuilder V2EfficiencyBuilder
	costBuilder       V2CostBuilder
	locationBuilder   V2LocationBuilder
	insightBuilder    V2InsightBuilder
}

func NewV2ReportService(
	summaryBuilder V2SummaryBuilder,
	drivingBuilder V2DrivingBuilder,
	chargingBuilder V2ChargingBuilder,
	updateBuilder V2UpdateBuilder,
	parkingBuilder V2ParkingBuilder,
	batteryBuilder V2BatteryBuilder,
	efficiencyBuilder V2EfficiencyBuilder,
	costBuilder V2CostBuilder,
	locationBuilder V2LocationBuilder,
	insightBuilder V2InsightBuilder,
) V2ReportService {
	return V2ReportService{
		summaryBuilder:    summaryBuilder,
		drivingBuilder:    drivingBuilder,
		chargingBuilder:   chargingBuilder,
		updateBuilder:     updateBuilder,
		parkingBuilder:    parkingBuilder,
		batteryBuilder:    batteryBuilder,
		efficiencyBuilder: efficiencyBuilder,
		costBuilder:       costBuilder,
		locationBuilder:   locationBuilder,
		insightBuilder:    insightBuilder,
	}
}

var v2DefaultReportModules = []string{"summary", "driving", "charging", "parking", "battery", "efficiency", "cost", "locations", "updates"}

type reportSectionResult struct {
	section *V2ReportSection
	err     error
}

func (s V2ReportService) BuildReport(ctx context.Context, carIDParam string, timeRange V2TimeRange, include []string) (V2ReportResponse, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ReportResponse{}, err
	}

	modules := include
	if len(modules) == 0 {
		modules = v2DefaultReportModules
	}

	period := timeRange.Period
	if period == "" {
		period = "custom"
	}

	// Concurrent section builds
	type indexedResult struct {
		idx int
		res reportSectionResult
	}
	results := make(chan indexedResult, len(modules))
	var wg sync.WaitGroup

	for i, mod := range modules {
		mod = strings.TrimSpace(strings.ToLower(mod))
		wg.Add(1)
		go func(idx int, mod string) {
			defer wg.Done()
			section, err := s.buildSection(ctx, carIDParam, timeRange, mod)
			results <- indexedResult{idx, reportSectionResult{section, err}}
		}(i, mod)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	sectionsByIdx := make(map[int]*V2ReportSection)
	for r := range results {
		if r.res.err != nil {
			return V2ReportResponse{}, r.res.err
		}
		if r.res.section != nil {
			sectionsByIdx[r.idx] = r.res.section
		}
	}

	// Preserve requested module order
	var sections []V2ReportSection
	for i := range modules {
		if sec, ok := sectionsByIdx[i]; ok {
			sections = append(sections, *sec)
		}
	}
	if sections == nil {
		sections = []V2ReportSection{}
	}

	return V2ReportResponse{
		Title:    fmt.Sprintf("Report for car %d", carID),
		Period:   period,
		Sections: sections,
	}, nil
}

func (s V2ReportService) buildSection(ctx context.Context, carIDParam string, timeRange V2TimeRange, mod string) (*V2ReportSection, error) {
	switch mod {
	case "summary":
		if s.summaryBuilder == nil {
			return nil, nil
		}
		response, err := s.summaryBuilder.BuildSummary(ctx, carIDParam, timeRange)
		if err != nil {
			return nil, err
		}
		metrics := map[string]interface{}{
			"drive_count":            response.Summary.Driving.DriveCount,
			"distance_km":            response.Summary.Driving.DistanceKM,
			"duration_min":           response.Summary.Driving.DurationMin,
			"charging_session_count": response.Summary.Charging.SessionCount,
			"energy_added_kwh":       response.Summary.Charging.EnergyAddedKWh,
			"charging_cost":          response.Summary.Cost.ChargingCost,
			"update_count":           response.Summary.Updates.UpdateCount,
		}
		return &V2ReportSection{Type: "summary", Title: "Period Summary", Metrics: metrics}, nil

	case "driving":
		if s.drivingBuilder == nil {
			return nil, nil
		}
		response, _, err := s.drivingBuilder.BuildDriving(ctx, carIDParam, timeRange)
		if err != nil {
			return nil, err
		}
		if response.Summary.DriveCount == 0 {
			return nil, nil
		}
		metrics := map[string]interface{}{
			"drive_count":   response.Summary.DriveCount,
			"distance_km":   response.Summary.DistanceKM,
			"duration_min":  response.Summary.DurationMin,
			"max_speed_kmh": response.Summary.MaxSpeedKMH,
		}
		if response.Summary.AvgConsumptionWhPerKM != nil {
			metrics["avg_consumption_wh_per_km"] = *response.Summary.AvgConsumptionWhPerKM
		}
		return &V2ReportSection{Type: "driving", Title: "Driving Analytics", Metrics: metrics}, nil

	case "charging":
		if s.chargingBuilder == nil {
			return nil, nil
		}
		response, _, err := s.chargingBuilder.BuildCharging(ctx, carIDParam, timeRange)
		if err != nil {
			return nil, err
		}
		if response.Summary.SessionCount == 0 {
			return nil, nil
		}
		metrics := map[string]interface{}{
			"session_count":    response.Summary.SessionCount,
			"energy_added_kwh": response.Summary.EnergyAddedKWh,
			"duration_min":     response.Summary.DurationMin,
		}
		if response.Summary.EnergyUsedKWh != nil {
			metrics["energy_used_kwh"] = *response.Summary.EnergyUsedKWh
		}
		if response.Summary.Cost != nil {
			metrics["cost"] = *response.Summary.Cost
		}
		return &V2ReportSection{Type: "charging", Title: "Charging Analytics", Metrics: metrics}, nil

	case "parking":
		if s.parkingBuilder == nil {
			return nil, nil
		}
		response, _, err := s.parkingBuilder.BuildParking(ctx, carIDParam, timeRange)
		if err != nil {
			return nil, err
		}
		if response.Summary.ParkingSessionCount == 0 {
			return nil, nil
		}
		metrics := map[string]interface{}{
			"parking_session_count": response.Summary.ParkingSessionCount,
			"parked_duration_min":   response.Summary.ParkedDurationMin,
			"online_duration_min":   response.Summary.OnlineDurationMin,
			"offline_duration_min":  response.Summary.OfflineDurationMin,
		}
		return &V2ReportSection{Type: "parking", Title: "Parking Analytics", Metrics: metrics}, nil

	case "battery":
		if s.batteryBuilder == nil {
			return nil, nil
		}
		response, _, err := s.batteryBuilder.BuildBattery(ctx, carIDParam, timeRange)
		if err != nil {
			return nil, err
		}
		if response.Summary.SampleCount == 0 {
			return nil, nil
		}
		metrics := map[string]interface{}{
			"sample_count": response.Summary.SampleCount,
		}
		if response.Summary.LatestBatteryLevelPercent != nil {
			metrics["latest_battery_level_percent"] = *response.Summary.LatestBatteryLevelPercent
		}
		if response.Summary.EstimatedRatedRangeAt100PercentKM != nil {
			metrics["estimated_rated_range_at_100_percent_km"] = *response.Summary.EstimatedRatedRangeAt100PercentKM
		}
		if response.Summary.EstimatedRangeDegradationPercent != nil {
			metrics["estimated_range_degradation_percent"] = *response.Summary.EstimatedRangeDegradationPercent
		}
		return &V2ReportSection{Type: "battery", Title: "Battery Analytics", Metrics: metrics}, nil

	case "efficiency":
		if s.efficiencyBuilder == nil {
			return nil, nil
		}
		response, _, err := s.efficiencyBuilder.BuildEfficiency(ctx, carIDParam, timeRange)
		if err != nil {
			return nil, err
		}
		if response.Summary.DriveCount == 0 {
			return nil, nil
		}
		metrics := map[string]interface{}{
			"drive_count": response.Summary.DriveCount,
			"distance_km": response.Summary.DistanceKM,
		}
		if response.Summary.AvgConsumptionWhPerKM != nil {
			metrics["avg_consumption_wh_per_km"] = *response.Summary.AvgConsumptionWhPerKM
		}
		if response.Summary.AvgTemperatureC != nil {
			metrics["avg_temperature_c"] = *response.Summary.AvgTemperatureC
		}
		return &V2ReportSection{Type: "efficiency", Title: "Efficiency Analytics", Metrics: metrics}, nil

	case "cost":
		if s.costBuilder == nil {
			return nil, nil
		}
		response, _, err := s.costBuilder.BuildCost(ctx, carIDParam, timeRange, "month")
		if err != nil {
			return nil, err
		}
		if response.Summary.ChargingCost == nil {
			return nil, nil
		}
		metrics := map[string]interface{}{
			"charging_cost": *response.Summary.ChargingCost,
			"distance_km":   response.Summary.DistanceKM,
		}
		if response.Summary.EnergyUsedKWh != nil {
			metrics["energy_used_kwh"] = *response.Summary.EnergyUsedKWh
		}
		if response.Summary.CostPer100KM != nil {
			metrics["cost_per_100km"] = *response.Summary.CostPer100KM
		}
		return &V2ReportSection{Type: "cost", Title: "Cost Analytics", Metrics: metrics}, nil

	case "locations":
		if s.locationBuilder == nil {
			return nil, nil
		}
		response, _, err := s.locationBuilder.BuildLocations(ctx, carIDParam, timeRange, "parking_duration_desc")
		if err != nil {
			return nil, err
		}
		if len(response.Items) == 0 {
			return nil, nil
		}
		metrics := map[string]interface{}{
			"location_count": len(response.Items),
		}
		return &V2ReportSection{Type: "locations", Title: "Location Analytics", Metrics: metrics}, nil

	case "updates":
		if s.updateBuilder == nil {
			return nil, nil
		}
		response, _, err := s.updateBuilder.BuildUpdates(ctx, carIDParam, timeRange)
		if err != nil {
			return nil, err
		}
		if response.UpdateCount == 0 {
			return nil, nil
		}
		metrics := map[string]interface{}{
			"update_count":   response.UpdateCount,
			"latest_version": response.LatestVersion,
		}
		if response.AvgUpdateDurationMin != nil {
			metrics["avg_update_duration_min"] = *response.AvgUpdateDurationMin
		}
		return &V2ReportSection{Type: "updates", Title: "Update History", Metrics: metrics}, nil

	case "insights":
		if s.insightBuilder == nil {
			return nil, nil
		}
		response, err := s.insightBuilder.BuildInsights(ctx, carIDParam, timeRange, "", "info")
		if err != nil {
			return nil, err
		}
		if len(response.Insights) == 0 {
			return nil, nil
		}
		metrics := map[string]interface{}{
			"insight_count": len(response.Insights),
		}
		return &V2ReportSection{Type: "insights", Title: "Objective Insights", Metrics: metrics}, nil

	default:
		return nil, nil
	}
}
