package main

import (
	"context"
	"errors"
)

var errV2InvalidParkingBreakdown = errors.New("invalid parking breakdown")

type V2ParkingRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2ParkingSummary, V2ParkingStats, error)
	Locations(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2ParkingLocationItem, V2ParkingStats, error)
	StateBreakdown(ctx context.Context, carID int64, timeRange V2TimeRange) (V2ParkingStatesResponse, V2ParkingStats, error)
}

// @name V2ParkingStats
type V2ParkingStats struct {
	StateRows    int64
	SessionRows  int64
	PositionRows int64
}

// @name V2ParkingService
type V2ParkingService struct {
	repository V2ParkingRepository
}

func NewV2ParkingService(repository V2ParkingRepository) V2ParkingService {
	return V2ParkingService{repository: repository}
}

// V2ParkingBuildOptions toggles the optional breakdown block on /analytics/parking.
type V2ParkingBuildOptions struct {
	IncludeBreakdown bool
	BreakdownBy      string
}

func (s V2ParkingService) BuildParking(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2ParkingBuildOptions) (V2ParkingResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ParkingResponse{}, 0, err
	}
	if opts.IncludeBreakdown {
		switch opts.BreakdownBy {
		case "location", "state":
		default:
			return V2ParkingResponse{}, 0, errV2InvalidParkingBreakdown
		}
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ParkingResponse{}, 0, err
	}
	summary, _, err := s.repository.Summary(ctx, carID, timeRange)
	if err != nil {
		return V2ParkingResponse{}, 0, err
	}
	response := V2ParkingResponse{Summary: summary}
	if opts.IncludeBreakdown {
		breakdown := &V2ParkingBreakdownResponse{By: opts.BreakdownBy}
		switch opts.BreakdownBy {
		case "location":
			items, _, err := s.repository.Locations(ctx, carID, timeRange)
			if err != nil {
				return V2ParkingResponse{}, 0, err
			}
			breakdown.Locations = items
		case "state":
			states, _, err := s.repository.StateBreakdown(ctx, carID, timeRange)
			if err != nil {
				return V2ParkingResponse{}, 0, err
			}
			breakdown.States = states.Items
		}
		response.Breakdown = breakdown
	}
	return response, carID, nil
}

func (s V2ParkingService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}

