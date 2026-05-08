package main

import (
	"context"
	"errors"
)

var errV2InvalidEfficiencyDimension = errors.New("invalid efficiency factor dimension")

var v2AllowedEfficiencyDimensions = map[string]bool{
	"temperature": true,
	"speed":       true,
	"distance":    true,
	"elevation":   true,
	"location":    true,
	"hour_of_day": true,
	"day_of_week": true,
}

type V2EfficiencyRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2EfficiencySummary, V2EfficiencyStats, error)
	Factors(ctx context.Context, carID int64, timeRange V2TimeRange, dimension string) ([]V2EfficiencyFactorItem, V2EfficiencyStats, error)
}

type V2EfficiencyStats struct {
	DriveRows          int64
	EnergyEstimateRows int64
	TemperatureRows    int64
	ElevationRows      int64
}

type V2EfficiencyService struct {
	repository V2EfficiencyRepository
}

func NewV2EfficiencyService(repository V2EfficiencyRepository) V2EfficiencyService {
	return V2EfficiencyService{repository: repository}
}

func (s V2EfficiencyService) BuildEfficiency(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2EfficiencyResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2EfficiencyResponse{}, V2DataQuality{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2EfficiencyResponse{}, V2DataQuality{}, 0, err
	}
	summary, stats, err := s.repository.Summary(ctx, carID, timeRange)
	if err != nil {
		return V2EfficiencyResponse{}, V2DataQuality{}, 0, err
	}
	return V2EfficiencyResponse{Summary: summary}, buildV2EfficiencyDataQuality(stats), carID, nil
}

func (s V2EfficiencyService) BuildEfficiencyFactors(ctx context.Context, carIDParam string, timeRange V2TimeRange, dimension string) (V2EfficiencyFactorsResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2EfficiencyFactorsResponse{}, V2DataQuality{}, 0, err
	}
	if dimension == "" {
		dimension = "temperature"
	}
	if !v2AllowedEfficiencyDimensions[dimension] {
		return V2EfficiencyFactorsResponse{}, V2DataQuality{}, 0, errV2InvalidEfficiencyDimension
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2EfficiencyFactorsResponse{}, V2DataQuality{}, 0, err
	}
	items, stats, err := s.repository.Factors(ctx, carID, timeRange, dimension)
	if err != nil {
		return V2EfficiencyFactorsResponse{}, V2DataQuality{}, 0, err
	}
	return V2EfficiencyFactorsResponse{Dimension: dimension, Items: items}, buildV2EfficiencyDataQuality(stats), carID, nil
}

func (s V2EfficiencyService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}

func buildV2EfficiencyDataQuality(stats V2EfficiencyStats) V2DataQuality {
	quality := V2DataQuality{
		Complete:    true,
		SampleCount: stats.DriveRows,
		Warnings: []string{
			"estimated_energy_consumed_kwh and estimated_regenerated_energy_kwh are derived from rated range deltas and vehicle efficiency when available.",
			"Efficiency factor buckets are factual groupings only and do not imply causation.",
		},
	}
	if stats.DriveRows == 0 {
		quality.Complete = false
		quality.Warnings = append(quality.Warnings, "No completed drives are available for the selected period.")
		return quality
	}
	if stats.EnergyEstimateRows < stats.DriveRows {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "drives.start_rated_range_km", "drives.end_rated_range_km", "cars.efficiency")
	}
	if stats.TemperatureRows == 0 {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "drives.outside_temp_avg")
	}
	return quality
}
