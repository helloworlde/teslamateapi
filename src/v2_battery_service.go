package main

import "context"

const v2MinimumBatteryRangeSamples int64 = 5

type V2BatteryRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2BatteryAnalyticsSummary, V2BatteryStats, error)
	Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2BatteryTimeseriesItem, V2BatteryStats, error)
}

// @name V2BatteryStats
type V2BatteryStats struct {
	SampleRows         int64
	InvalidBatteryRows int64
}

// @name V2BatteryService
type V2BatteryService struct {
	repository V2BatteryRepository
}

func NewV2BatteryService(repository V2BatteryRepository) V2BatteryService {
	return V2BatteryService{repository: repository}
}

func (s V2BatteryService) BuildBattery(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2BatteryResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2BatteryResponse{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2BatteryResponse{}, 0, err
	}
	summary, _, err := s.repository.Summary(ctx, carID, timeRange)
	if err != nil {
		return V2BatteryResponse{}, 0, err
	}
	return V2BatteryResponse{Summary: summary}, carID, nil
}

func (s V2BatteryService) BuildBatteryTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2BatteryTimeseriesResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2BatteryTimeseriesResponse{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2BatteryTimeseriesResponse{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2BatteryTimeseriesResponse{}, 0, err
	}
	items, _, err := s.repository.Timeseries(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2BatteryTimeseriesResponse{}, 0, err
	}
	return V2BatteryTimeseriesResponse{GroupBy: groupBy, Items: items}, carID, nil
}

func (s V2BatteryService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}

