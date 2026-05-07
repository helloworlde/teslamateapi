package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

var (
	errV2InvalidDrivingDimension = errors.New("invalid driving distribution dimension")
	errV2InvalidDrivingRanking   = errors.New("invalid driving ranking type")
	errV2InvalidDrivingGroupBy   = errors.New("invalid driving group_by")
)

var v2AllowedDrivingDimensions = map[string]bool{
	"hour_of_day":        true,
	"day_of_week":        true,
	"distance_bucket":    true,
	"duration_bucket":    true,
	"speed_bucket":       true,
	"consumption_bucket": true,
	"temperature_bucket": true,
}

var v2AllowedDrivingRankingTypes = map[string]bool{
	"longest_distance":     true,
	"longest_duration":     true,
	"highest_speed":        true,
	"lowest_consumption":   true,
	"highest_consumption":  true,
	"highest_distance_day": true,
}

var v2AllowedDrivingGroupBy = map[string]bool{
	"day":   true,
	"week":  true,
	"month": true,
	"year":  true,
}

type V2DrivingRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, start timeBound, end timeBound) (V2DrivingAnalyticsSummary, V2DrivingStats, error)
	Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2DrivingTimeseriesItem, V2DrivingStats, error)
	Distribution(ctx context.Context, carID int64, timeRange V2TimeRange, dimension string) ([]V2DrivingDistributionItem, V2DrivingStats, error)
	Ranking(ctx context.Context, carID int64, timeRange V2TimeRange, rankingType string, limit int) ([]V2DrivingRankingItem, V2DrivingStats, error)
}

type V2DrivingStats struct {
	DriveRows          int64
	EnergyEstimateRows int64
	TemperatureRows    int64
}

type V2DrivingService struct {
	repository V2DrivingRepository
}

func NewV2DrivingService(repository V2DrivingRepository) V2DrivingService {
	return V2DrivingService{repository: repository}
}

func (s V2DrivingService) BuildDriving(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2DrivingResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2DrivingResponse{}, V2DataQuality{}, 0, err
	}
	if timeRange.Compare == "previous_year" || timeRange.Compare == "lifetime_average" {
		return V2DrivingResponse{}, V2DataQuality{}, 0, errV2CompareUnsupported
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2DrivingResponse{}, V2DataQuality{}, 0, err
	}

	summary, stats, err := s.repository.Summary(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End))
	if err != nil {
		return V2DrivingResponse{}, V2DataQuality{}, 0, err
	}
	response := V2DrivingResponse{Summary: summary}
	if timeRange.Compare == "previous_period" && timeRange.PreviousStart != nil && timeRange.PreviousEnd != nil {
		previous, _, err := s.repository.Summary(ctx, carID, asTimeBound(*timeRange.PreviousStart), asTimeBound(*timeRange.PreviousEnd))
		if err != nil {
			return V2DrivingResponse{}, V2DataQuality{}, 0, err
		}
		response.Comparison = buildV2DrivingComparison(summary, previous)
	}

	return response, buildV2DrivingDataQuality(stats), carID, nil
}

func (s V2DrivingService) BuildTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2DrivingTimeseriesResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2DrivingTimeseriesResponse{}, V2DataQuality{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2DrivingTimeseriesResponse{}, V2DataQuality{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2DrivingTimeseriesResponse{}, V2DataQuality{}, 0, err
	}

	items, stats, err := s.repository.Timeseries(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2DrivingTimeseriesResponse{}, V2DataQuality{}, 0, err
	}
	return V2DrivingTimeseriesResponse{GroupBy: groupBy, Items: items}, buildV2DrivingDataQuality(stats), carID, nil
}

func (s V2DrivingService) BuildDistribution(ctx context.Context, carIDParam string, timeRange V2TimeRange, dimension string) (V2DrivingDistributionResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2DrivingDistributionResponse{}, V2DataQuality{}, 0, err
	}
	if dimension == "" {
		dimension = "hour_of_day"
	}
	if !v2AllowedDrivingDimensions[dimension] {
		return V2DrivingDistributionResponse{}, V2DataQuality{}, 0, errV2InvalidDrivingDimension
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2DrivingDistributionResponse{}, V2DataQuality{}, 0, err
	}

	items, stats, err := s.repository.Distribution(ctx, carID, timeRange, dimension)
	if err != nil {
		return V2DrivingDistributionResponse{}, V2DataQuality{}, 0, err
	}
	return V2DrivingDistributionResponse{Dimension: dimension, Items: items}, buildV2DrivingDataQuality(stats), carID, nil
}

func (s V2DrivingService) BuildRanking(ctx context.Context, carIDParam string, timeRange V2TimeRange, rankingType string, limit int) (V2DrivingRankingResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2DrivingRankingResponse{}, V2DataQuality{}, 0, err
	}
	if rankingType == "" {
		rankingType = "longest_distance"
	}
	if !v2AllowedDrivingRankingTypes[rankingType] {
		return V2DrivingRankingResponse{}, V2DataQuality{}, 0, errV2InvalidDrivingRanking
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2DrivingRankingResponse{}, V2DataQuality{}, 0, err
	}

	items, stats, err := s.repository.Ranking(ctx, carID, timeRange, rankingType, limit)
	if err != nil {
		return V2DrivingRankingResponse{}, V2DataQuality{}, 0, err
	}
	return V2DrivingRankingResponse{Type: rankingType, Items: items}, buildV2DrivingDataQuality(stats), carID, nil
}

func parseV2CarID(carIDParam string) (int64, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return 0, fmt.Errorf("invalid car id")
	}
	return carID, nil
}

func (s V2DrivingService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}

func defaultV2DrivingGroupBy(period string) string {
	switch period {
	case "year", "lifetime":
		return "month"
	case "quarter":
		return "week"
	default:
		return "day"
	}
}

func buildV2DrivingComparison(current V2DrivingAnalyticsSummary, previous V2DrivingAnalyticsSummary) map[string]V2ComparisonValue {
	return map[string]V2ComparisonValue{
		"drive_count":   compareFloat(float64(current.DriveCount), float64(previous.DriveCount)),
		"distance_km":   compareFloat(current.DistanceKM, previous.DistanceKM),
		"duration_min":  compareFloat(current.DurationMin, previous.DurationMin),
		"max_speed_kmh": compareFloat(current.MaxSpeedKMH, previous.MaxSpeedKMH),
		"range_loss_km": compareFloat(current.RangeLossKM, previous.RangeLossKM),
	}
}

func buildV2DrivingDataQuality(stats V2DrivingStats) V2DataQuality {
	quality := V2DataQuality{
		Complete:    true,
		SampleCount: stats.DriveRows,
	}
	if stats.DriveRows > 0 {
		quality.Warnings = append(quality.Warnings, "estimated_energy_consumed_kwh is derived from rated range loss and vehicle efficiency when available.")
		quality.Warnings = append(quality.Warnings, "estimated_regenerated_energy_kwh is not returned unless sufficient direct data exists.")
		quality.Warnings = append(quality.Warnings, "total_ascent_m and total_descent_m are zero until elevation aggregation is implemented from position samples.")
	}
	if stats.DriveRows > 0 && stats.EnergyEstimateRows == 0 {
		quality.MissingFields = append(quality.MissingFields, "drives.start_rated_range_km", "drives.end_rated_range_km", "cars.efficiency")
	}
	if stats.DriveRows > 0 && stats.TemperatureRows == 0 {
		quality.MissingFields = append(quality.MissingFields, "drives.outside_temp_avg")
	}
	if len(quality.MissingFields) > 0 || len(quality.Warnings) > 0 {
		quality.Complete = false
	}
	return quality
}

func v2LimitFromQuery(value string) int {
	if value == "" {
		return 10
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 10
	}
	return limit
}
