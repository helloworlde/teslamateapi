package main

import (
	"context"
	"errors"
)

var errV2InvalidChargingBreakdown = errors.New("invalid charging breakdown")

type V2ChargingRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, start timeBound, end timeBound) (V2ChargingSummary, V2ChargingStats, error)
	Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2ChargingTimeseriesItem, V2ChargingStats, error)
	Locations(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2ChargingLocationItem, V2ChargingStats, error)
	Types(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2ChargingTypeItem, V2ChargingStats, error)
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

// V2ChargingBuildOptions toggles optional timeseries/breakdown blocks on the
// /analytics/charging endpoint via the include / breakdown query params.
type V2ChargingBuildOptions struct {
	IncludeTimeseries bool
	IncludeBreakdown  bool
	GroupBy           string
	BreakdownBy       string
}

func (s V2ChargingService) BuildCharging(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2ChargingBuildOptions) (V2ChargingResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ChargingResponse{}, 0, err
	}

	groupBy := opts.GroupBy
	if opts.IncludeTimeseries {
		if groupBy == "" {
			groupBy = defaultV2DrivingGroupBy(timeRange.Period)
		}
		if !v2AllowedDrivingGroupBy[groupBy] {
			return V2ChargingResponse{}, 0, errV2InvalidDrivingGroupBy
		}
	}
	if opts.IncludeBreakdown {
		switch opts.BreakdownBy {
		case "location", "charger_type":
		default:
			return V2ChargingResponse{}, 0, errV2InvalidChargingBreakdown
		}
	}

	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ChargingResponse{}, 0, err
	}
	summary, _, err := s.repository.Summary(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End))
	if err != nil {
		return V2ChargingResponse{}, 0, err
	}
	response := V2ChargingResponse{Summary: summary}
	if opts.IncludeTimeseries {
		items, _, err := s.repository.Timeseries(ctx, carID, timeRange, groupBy)
		if err != nil {
			return V2ChargingResponse{}, 0, err
		}
		response.Timeseries = &V2ChargingTimeseriesResponse{GroupBy: groupBy, Items: items}
	}
	if opts.IncludeBreakdown {
		breakdown := &V2ChargingBreakdownResponse{By: opts.BreakdownBy}
		switch opts.BreakdownBy {
		case "location":
			items, _, err := s.repository.Locations(ctx, carID, timeRange)
			if err != nil {
				return V2ChargingResponse{}, 0, err
			}
			breakdown.Locations = items
		case "charger_type":
			items, _, err := s.repository.Types(ctx, carID, timeRange)
			if err != nil {
				return V2ChargingResponse{}, 0, err
			}
			breakdown.Types = items
		}
		response.Breakdown = breakdown
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
