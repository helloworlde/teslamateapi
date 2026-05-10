package main

import "context"

type V2CostRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Cost(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) (V2CostResponse, V2CostStats, error)
}

type V2CostStats struct {
	SessionRows    int64
	CostRows       int64
	EnergyUsedRows int64
}

type V2CostService struct {
	repository V2CostRepository
}

func NewV2CostService(repository V2CostRepository) V2CostService {
	return V2CostService{repository: repository}
}

func (s V2CostService) BuildCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2CostResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2CostResponse{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2CostResponse{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2CostResponse{}, 0, err
	}
	response, _, err := s.repository.Cost(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2CostResponse{}, 0, err
	}
	if len(response.DataScope.Included) == 0 {
		response.DataScope = defaultV2CostDataScope()
	}
	return response, carID, nil
}

func (s V2CostService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}
