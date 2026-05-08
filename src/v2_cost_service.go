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

func (s V2CostService) BuildCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2CostResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2CostResponse{}, V2DataQuality{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2CostResponse{}, V2DataQuality{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2CostResponse{}, V2DataQuality{}, 0, err
	}
	response, stats, err := s.repository.Cost(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2CostResponse{}, V2DataQuality{}, 0, err
	}
	if len(response.DataScope.Included) == 0 {
		response.DataScope = defaultV2CostDataScope()
	}
	return response, buildV2CostDataQuality(stats), carID, nil
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

func buildV2CostDataQuality(stats V2CostStats) V2DataQuality {
	quality := V2DataQuality{Complete: true, SampleCount: stats.SessionRows}
	if stats.SessionRows > 0 && stats.CostRows < stats.SessionRows {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "charging_processes.cost")
		quality.Warnings = append(quality.Warnings, "Some charging sessions do not have cost data; charging_cost and derived cost metrics use only available cost values.")
	}
	if stats.SessionRows > 0 && stats.EnergyUsedRows < stats.SessionRows {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "charging_processes.charge_energy_used")
		quality.Warnings = append(quality.Warnings, "Some charging sessions do not have energy_used_kwh; cost_per_kwh is omitted when energy usage is unavailable.")
	}
	return quality
}
