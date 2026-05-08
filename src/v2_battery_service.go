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

func (s V2BatteryService) BuildBattery(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2BatteryResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2BatteryResponse{}, V2DataQuality{}, 0, err
	}
	if timeRange.Compare == "previous_year" || timeRange.Compare == "lifetime_average" {
		return V2BatteryResponse{}, V2DataQuality{}, 0, errV2CompareUnsupported
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2BatteryResponse{}, V2DataQuality{}, 0, err
	}
	summary, stats, err := s.repository.Summary(ctx, carID, timeRange)
	if err != nil {
		return V2BatteryResponse{}, V2DataQuality{}, 0, err
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
			return V2BatteryResponse{}, V2DataQuality{}, 0, err
		}
		response.Comparison = buildV2BatteryComparison(summary, previous)
	}
	return response, buildV2BatteryDataQuality(stats, summary), carID, nil
}

func (s V2BatteryService) BuildBatteryTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2BatteryTimeseriesResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2BatteryTimeseriesResponse{}, V2DataQuality{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2BatteryTimeseriesResponse{}, V2DataQuality{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2BatteryTimeseriesResponse{}, V2DataQuality{}, 0, err
	}
	items, stats, err := s.repository.Timeseries(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2BatteryTimeseriesResponse{}, V2DataQuality{}, 0, err
	}
	return V2BatteryTimeseriesResponse{GroupBy: groupBy, Items: items}, buildV2BatterySampleDataQuality(stats), carID, nil
}

func (s V2BatteryService) BuildBatteryDistribution(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2BatteryDistributionResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2BatteryDistributionResponse{}, V2DataQuality{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2BatteryDistributionResponse{}, V2DataQuality{}, 0, err
	}
	items, stats, err := s.repository.Distribution(ctx, carID, timeRange)
	if err != nil {
		return V2BatteryDistributionResponse{}, V2DataQuality{}, 0, err
	}
	quality := V2DataQuality{Complete: true, SampleCount: stats.SampleRows}
	if stats.SampleRows == 0 {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "positions.battery_level")
		quality.Warnings = append(quality.Warnings, "No battery_level samples are available for the selected period.")
	}
	return V2BatteryDistributionResponse{Items: items}, quality, carID, nil
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

func buildV2BatteryDataQuality(stats V2BatteryStats, summary V2BatteryAnalyticsSummary) V2DataQuality {
	quality := V2DataQuality{
		Complete:    true,
		SampleCount: stats.SampleRows,
		Warnings: []string{
			"Battery range analytics are estimates from TeslaMate samples and are not official state of health.",
			"estimated_range_degradation_percent can be affected by temperature, BMS calibration, tire and wheel configuration, and sample distribution.",
		},
	}
	if stats.SampleRows == 0 {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "positions.battery_level", "positions.rated_battery_range_km", "positions.ideal_battery_range_km")
		quality.Warnings = append(quality.Warnings, "No usable battery range samples are available for the selected period.")
	}
	if stats.InvalidBatteryRows > 0 {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "positions.battery_level")
		quality.Warnings = append(quality.Warnings, "Some position samples have null or zero battery_level and were ignored for full-range estimates.")
	}
	if summary.EstimatedRangeDegradationPercent == nil && stats.SampleRows > 0 {
		quality.Complete = false
		quality.Warnings = append(quality.Warnings, "estimated_range_degradation_percent is null because there are too few usable samples or no baseline rated range.")
	}
	return quality
}

func buildV2BatterySampleDataQuality(stats V2BatteryStats) V2DataQuality {
	quality := V2DataQuality{
		Complete:    true,
		SampleCount: stats.SampleRows,
		Warnings: []string{
			"Battery range analytics are estimates from TeslaMate samples and are not official state of health.",
			"Estimated full-range trends can be affected by temperature, BMS calibration, tire and wheel configuration, and sample distribution.",
		},
	}
	if stats.SampleRows == 0 {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "positions.battery_level", "positions.rated_battery_range_km", "positions.ideal_battery_range_km")
		quality.Warnings = append(quality.Warnings, "No usable battery range samples are available for the selected period.")
	}
	if stats.InvalidBatteryRows > 0 {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "positions.battery_level")
		quality.Warnings = append(quality.Warnings, "Some position samples have null or zero battery_level and were ignored for full-range estimates.")
	}
	return quality
}
