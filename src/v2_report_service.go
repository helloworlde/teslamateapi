package main

import (
	"context"
	"fmt"
	"strings"
)

type V2ReportService struct {
	summaryBuilder  V2SummaryBuilder
	drivingBuilder  V2DrivingBuilder
	chargingBuilder V2ChargingBuilder
	updateBuilder   V2UpdateBuilder
}

func NewV2ReportService(summaryBuilder V2SummaryBuilder, drivingBuilder V2DrivingBuilder, chargingBuilder V2ChargingBuilder, updateBuilder V2UpdateBuilder) V2ReportService {
	return V2ReportService{
		summaryBuilder:  summaryBuilder,
		drivingBuilder:  drivingBuilder,
		chargingBuilder: chargingBuilder,
		updateBuilder:   updateBuilder,
	}
}

var v2AllowedReportModules = map[string]bool{
	"summary":  true,
	"driving":  true,
	"charging": true,
	"updates":  true,
}

var v2DefaultReportModules = []string{"summary", "driving", "charging", "updates"}

func (s V2ReportService) BuildReport(ctx context.Context, carIDParam string, timeRange V2TimeRange, include []string) (V2ReportResponse, V2DataQuality, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ReportResponse{}, V2DataQuality{}, err
	}

	modules := include
	if len(modules) == 0 {
		modules = v2DefaultReportModules
	}

	title := fmt.Sprintf("Report for car %d", carID)
	period := timeRange.Period
	if period == "" {
		period = "custom"
	}

	var sections []V2ReportSection
	var totalSamples int64

	for _, mod := range modules {
		mod = strings.TrimSpace(strings.ToLower(mod))
		switch mod {
		case "summary":
			if s.summaryBuilder == nil {
				continue
			}
			response, quality, err := s.summaryBuilder.BuildSummary(ctx, carIDParam, timeRange)
			if err != nil {
				return V2ReportResponse{}, V2DataQuality{}, err
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
			totalSamples += quality.SampleCount
			sections = append(sections, V2ReportSection{
				Type:        "summary",
				Title:       "Period Summary",
				Metrics:     metrics,
				DataQuality: quality,
			})
		case "driving":
			if s.drivingBuilder == nil {
				continue
			}
			response, quality, _, err := s.drivingBuilder.BuildDriving(ctx, carIDParam, timeRange)
			if err != nil {
				return V2ReportResponse{}, V2DataQuality{}, err
			}
			if response.Summary.DriveCount == 0 {
				continue
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
			totalSamples += quality.SampleCount
			sections = append(sections, V2ReportSection{
				Type:        "driving",
				Title:       "Driving Analytics",
				Metrics:     metrics,
				DataQuality: quality,
			})
		case "charging":
			if s.chargingBuilder == nil {
				continue
			}
			response, quality, _, err := s.chargingBuilder.BuildCharging(ctx, carIDParam, timeRange)
			if err != nil {
				return V2ReportResponse{}, V2DataQuality{}, err
			}
			if response.Summary.SessionCount == 0 {
				continue
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
			totalSamples += quality.SampleCount
			sections = append(sections, V2ReportSection{
				Type:        "charging",
				Title:       "Charging Analytics",
				Metrics:     metrics,
				DataQuality: quality,
			})
		case "updates":
			if s.updateBuilder == nil {
				continue
			}
			response, quality, _, err := s.updateBuilder.BuildUpdates(ctx, carIDParam, timeRange)
			if err != nil {
				return V2ReportResponse{}, V2DataQuality{}, err
			}
			if response.UpdateCount == 0 {
				continue
			}
			metrics := map[string]interface{}{
				"update_count":   response.UpdateCount,
				"latest_version": response.LatestVersion,
			}
			if response.AvgUpdateDurationMin != nil {
				metrics["avg_update_duration_min"] = *response.AvgUpdateDurationMin
			}
			totalSamples += quality.SampleCount
			sections = append(sections, V2ReportSection{
				Type:        "updates",
				Title:       "Update History",
				Metrics:     metrics,
				DataQuality: quality,
			})
		}
	}

	if sections == nil {
		sections = []V2ReportSection{}
	}

	quality := V2DataQuality{
		Complete:    true,
		SampleCount: totalSamples,
	}

	return V2ReportResponse{
		Title:    title,
		Period:   period,
		Sections: sections,
	}, quality, nil
}
