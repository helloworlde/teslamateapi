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

// @name V2ChargingStats
type V2ChargingStats struct {
	SessionRows    int64
	EnergyUsedRows int64
	CostRows       int64
	PowerRows      int64
}

// @name V2ChargingService
type V2ChargingService struct {
	repository V2ChargingRepository
}

func NewV2ChargingService(repository V2ChargingRepository) V2ChargingService {
	return V2ChargingService{repository: repository}
}

func (s V2ChargingService) BuildCharging(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingResponse{}, 0, err
	}
	if timeRange.Compare == "previous_year" || timeRange.Compare == "lifetime_average" {
		return V2ChargingResponse{}, 0, errV2CompareUnsupported
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingResponse{}, 0, err
	}
	summary, _, err := s.repository.Summary(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End))
	if err != nil {
		return V2ChargingResponse{}, 0, err
	}
	response := V2ChargingResponse{Summary: summary}
	if timeRange.Compare == "previous_period" && timeRange.PreviousStart != nil && timeRange.PreviousEnd != nil {
		previous, _, err := s.repository.Summary(ctx, carID, asTimeBound(*timeRange.PreviousStart), asTimeBound(*timeRange.PreviousEnd))
		if err != nil {
			return V2ChargingResponse{}, 0, err
		}
		response.Comparison = buildV2ChargingComparison(summary, previous)
	}
	return response, carID, nil
}

func (s V2ChargingService) BuildChargingTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingTimeseriesResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingTimeseriesResponse{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2ChargingTimeseriesResponse{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingTimeseriesResponse{}, 0, err
	}
	items, _, err := s.repository.Timeseries(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2ChargingTimeseriesResponse{}, 0, err
	}
	return V2ChargingTimeseriesResponse{GroupBy: groupBy, Items: items}, carID, nil
}

func (s V2ChargingService) BuildChargingLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingLocationsResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingLocationsResponse{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingLocationsResponse{}, 0, err
	}
	items, _, err := s.repository.Locations(ctx, carID, timeRange)
	if err != nil {
		return V2ChargingLocationsResponse{}, 0, err
	}
	return V2ChargingLocationsResponse{Items: items}, carID, nil
}

func (s V2ChargingService) BuildChargingTypes(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingTypesResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingTypesResponse{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingTypesResponse{}, 0, err
	}
	items, _, err := s.repository.Types(ctx, carID, timeRange)
	if err != nil {
		return V2ChargingTypesResponse{}, 0, err
	}
	return V2ChargingTypesResponse{Items: items}, carID, nil
}

func (s V2ChargingService) BuildChargingCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingCostResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingCostResponse{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2ChargingCostResponse{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingCostResponse{}, 0, err
	}
	response, _, err := s.repository.Cost(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2ChargingCostResponse{}, 0, err
	}
	return response, carID, nil
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
