package main

import "context"

const v2MinimumBatteryRangeSamples int64 = 5

type V2BatteryRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2BatteryAnalyticsSummary, V2BatteryStats, error)
	Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2BatteryTimeseriesItem, V2BatteryStats, error)
	Distribution(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2BatteryDistributionItem, V2BatteryStats, error)
}

type V2BatteryStats struct {
	SampleRows         int64
	InvalidBatteryRows int64
}

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
	if timeRange.Compare == "previous_year" || timeRange.Compare == "lifetime_average" {
		return V2BatteryResponse{}, 0, errV2CompareUnsupported
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2BatteryResponse{}, 0, err
	}
	summary, _, err := s.repository.Summary(ctx, carID, timeRange)
	if err != nil {
		return V2BatteryResponse{}, 0, err
	}
	response := V2BatteryResponse{Summary: summary}
	if timeRange.Compare == "previous_period" && timeRange.PreviousStart != nil && timeRange.PreviousEnd != nil {
		previousRange := timeRange
		previousRange.Start = *timeRange.PreviousStart
		previousRange.End = *timeRange.PreviousEnd
		previousRange.PreviousStart = nil
		previousRange.PreviousEnd = nil
		previous, _, err := s.repository.Summary(ctx, carID, previousRange)
		if err != nil {
			return V2BatteryResponse{}, 0, err
		}
		response.Comparison = buildV2BatteryComparison(summary, previous)
	}
	return response, carID, nil
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

func (s V2BatteryService) BuildBatteryDistribution(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2BatteryDistributionResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2BatteryDistributionResponse{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2BatteryDistributionResponse{}, 0, err
	}
	items, _, err := s.repository.Distribution(ctx, carID, timeRange)
	if err != nil {
		return V2BatteryDistributionResponse{}, 0, err
	}
	return V2BatteryDistributionResponse{Items: items}, carID, nil
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

func buildV2BatteryComparison(current V2BatteryAnalyticsSummary, previous V2BatteryAnalyticsSummary) map[string]V2ComparisonValue {
	comparison := map[string]V2ComparisonValue{
		"sample_count": compareFloat(float64(current.SampleCount), float64(previous.SampleCount)),
	}
	if current.EstimatedRatedRangeAt100PercentKM != nil && previous.EstimatedRatedRangeAt100PercentKM != nil {
		comparison["estimated_rated_range_at_100_percent_km"] = compareFloat(*current.EstimatedRatedRangeAt100PercentKM, *previous.EstimatedRatedRangeAt100PercentKM)
	}
	if current.EstimatedIdealRangeAt100PercentKM != nil && previous.EstimatedIdealRangeAt100PercentKM != nil {
		comparison["estimated_ideal_range_at_100_percent_km"] = compareFloat(*current.EstimatedIdealRangeAt100PercentKM, *previous.EstimatedIdealRangeAt100PercentKM)
	}
	return comparison
}
