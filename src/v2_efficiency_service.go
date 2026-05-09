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

func (s V2EfficiencyService) BuildEfficiency(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2EfficiencyResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2EfficiencyResponse{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2EfficiencyResponse{}, 0, err
	}
	summary, _, err := s.repository.Summary(ctx, carID, timeRange)
	if err != nil {
		return V2EfficiencyResponse{}, 0, err
	}
	return V2EfficiencyResponse{Summary: summary}, carID, nil
}

func (s V2EfficiencyService) BuildEfficiencyFactors(ctx context.Context, carIDParam string, timeRange V2TimeRange, dimension string) (V2EfficiencyFactorsResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2EfficiencyFactorsResponse{}, 0, err
	}
	if dimension == "" {
		dimension = "temperature"
	}
	if !v2AllowedEfficiencyDimensions[dimension] {
		return V2EfficiencyFactorsResponse{}, 0, errV2InvalidEfficiencyDimension
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2EfficiencyFactorsResponse{}, 0, err
	}
	items, _, err := s.repository.Factors(ctx, carID, timeRange, dimension)
	if err != nil {
		return V2EfficiencyFactorsResponse{}, 0, err
	}
	return V2EfficiencyFactorsResponse{Dimension: dimension, Items: items}, carID, nil
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

