package main

import (
	"context"
)

type V2ChargingRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, start timeBound, end timeBound) (V2ChargingAnalyticsSummary, V2ChargingStats, error)
	Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2ChargingTimeseriesItem, V2ChargingStats, error)
	Locations(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2ChargingLocationItem, V2ChargingStats, error)
	Types(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2ChargingTypeItem, V2ChargingStats, error)
	Cost(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) (V2ChargingCostResponse, V2ChargingStats, error)
}

type V2ChargingStats struct {
	SessionRows    int64
	EnergyUsedRows int64
	CostRows       int64
	PowerRows      int64
}

type V2ChargingService struct {
	repository V2ChargingRepository
}

func NewV2ChargingService(repository V2ChargingRepository) V2ChargingService {
	return V2ChargingService{repository: repository}
}

func (s V2ChargingService) BuildCharging(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingResponse{}, V2DataQuality{}, 0, err
	}
	if timeRange.Compare == "previous_year" || timeRange.Compare == "lifetime_average" {
		return V2ChargingResponse{}, V2DataQuality{}, 0, errV2CompareUnsupported
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingResponse{}, V2DataQuality{}, 0, err
	}
	summary, stats, err := s.repository.Summary(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End))
	if err != nil {
		return V2ChargingResponse{}, V2DataQuality{}, 0, err
	}
	response := V2ChargingResponse{Summary: summary}
	if timeRange.Compare == "previous_period" && timeRange.PreviousStart != nil && timeRange.PreviousEnd != nil {
		previous, _, err := s.repository.Summary(ctx, carID, asTimeBound(*timeRange.PreviousStart), asTimeBound(*timeRange.PreviousEnd))
		if err != nil {
			return V2ChargingResponse{}, V2DataQuality{}, 0, err
		}
		response.Comparison = buildV2ChargingComparison(summary, previous)
	}
	return response, buildV2ChargingDataQuality(stats), carID, nil
}

func (s V2ChargingService) BuildChargingTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingTimeseriesResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingTimeseriesResponse{}, V2DataQuality{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2ChargingTimeseriesResponse{}, V2DataQuality{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingTimeseriesResponse{}, V2DataQuality{}, 0, err
	}
	items, stats, err := s.repository.Timeseries(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2ChargingTimeseriesResponse{}, V2DataQuality{}, 0, err
	}
	return V2ChargingTimeseriesResponse{GroupBy: groupBy, Items: items}, buildV2ChargingDataQuality(stats), carID, nil
}

func (s V2ChargingService) BuildChargingLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingLocationsResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingLocationsResponse{}, V2DataQuality{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingLocationsResponse{}, V2DataQuality{}, 0, err
	}
	items, stats, err := s.repository.Locations(ctx, carID, timeRange)
	if err != nil {
		return V2ChargingLocationsResponse{}, V2DataQuality{}, 0, err
	}
	return V2ChargingLocationsResponse{Items: items}, buildV2ChargingDataQuality(stats), carID, nil
}

func (s V2ChargingService) BuildChargingTypes(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingTypesResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingTypesResponse{}, V2DataQuality{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingTypesResponse{}, V2DataQuality{}, 0, err
	}
	items, stats, err := s.repository.Types(ctx, carID, timeRange)
	if err != nil {
		return V2ChargingTypesResponse{}, V2DataQuality{}, 0, err
	}
	return V2ChargingTypesResponse{Items: items}, buildV2ChargingDataQuality(stats), carID, nil
}

func (s V2ChargingService) BuildChargingCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingCostResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingCostResponse{}, V2DataQuality{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2ChargingCostResponse{}, V2DataQuality{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingCostResponse{}, V2DataQuality{}, 0, err
	}
	response, stats, err := s.repository.Cost(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2ChargingCostResponse{}, V2DataQuality{}, 0, err
	}
	return response, buildV2ChargingDataQuality(stats), carID, nil
}

func (s V2ChargingService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}

func buildV2ChargingComparison(current V2ChargingAnalyticsSummary, previous V2ChargingAnalyticsSummary) map[string]V2ComparisonValue {
	return map[string]V2ComparisonValue{
		"session_count":    compareFloat(float64(current.SessionCount), float64(previous.SessionCount)),
		"energy_added_kwh": compareFloat(current.EnergyAddedKWh, previous.EnergyAddedKWh),
		"duration_min":     compareFloat(current.DurationMin, previous.DurationMin),
	}
}

func buildV2ChargingDataQuality(stats V2ChargingStats) V2DataQuality {
	quality := V2DataQuality{Complete: true, SampleCount: stats.SessionRows}
	if stats.SessionRows > 0 && stats.EnergyUsedRows < stats.SessionRows {
		quality.MissingFields = append(quality.MissingFields, "charging_processes.charge_energy_used")
		quality.Warnings = append(quality.Warnings, "Some charging sessions do not have energy_used_kwh; charging_efficiency_percent is omitted when energy_used_kwh is unavailable.")
	}
	if stats.SessionRows > 0 && stats.CostRows < stats.SessionRows {
		quality.MissingFields = append(quality.MissingFields, "charging_processes.cost")
		quality.Warnings = append(quality.Warnings, "Some charging sessions do not have cost data.")
	}
	if stats.SessionRows > 0 && stats.PowerRows == 0 {
		quality.MissingFields = append(quality.MissingFields, "charges.charger_power")
	}
	if len(quality.MissingFields) > 0 || len(quality.Warnings) > 0 {
		quality.Complete = false
	}
	return quality
}
