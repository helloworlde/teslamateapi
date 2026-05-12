package main

import (
	"context"
	"errors"
	"testing"
)

type fakeV2CostRepository struct {
	exists   bool
	response V2CostResponse
	stats    V2CostStats
}

func (r *fakeV2CostRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2CostRepository) Cost(context.Context, int64, V2TimeRange, string) (V2CostResponse, V2CostStats, error) {
	return r.response, r.stats, nil
}

func TestV2CostServiceBuildCost(t *testing.T) {
	cost := 45.0
	energy := 30.0
	costPerEnergy := 1.5
	costPerDistance := 30.0
	service := NewV2CostService(&fakeV2CostRepository{
		exists: true,
		response: V2CostResponse{
			Summary: V2CostSummaryDetails{
				ChargingCost:    &cost,
				EnergyUsed:      &energy,
				Distance:        150,
				CostPerEnergy:   &costPerEnergy,
				CostPerDistance: &costPerDistance,
			},
		},
		stats: V2CostStats{SessionRows: 2, CostRows: 2, EnergyUsedRows: 2},
	})

	response, carID, err := service.BuildCost(context.Background(), "1", V2TimeRange{Period: "month"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carID != 1 || response.Summary.ChargingCost == nil {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
}

func TestV2CostServiceMissingCostWarns(t *testing.T) {
	service := NewV2CostService(&fakeV2CostRepository{
		exists: true,
		stats:  V2CostStats{SessionRows: 2, CostRows: 1, EnergyUsedRows: 2},
	})

	_, _, err := service.BuildCost(context.Background(), "1", V2TimeRange{Period: "month"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestV2CostServiceDistanceZeroOmitsDistanceCosts(t *testing.T) {
	cost := 10.0
	service := NewV2CostService(&fakeV2CostRepository{
		exists: true,
		response: V2CostResponse{
			Summary: V2CostSummaryDetails{ChargingCost: &cost, Distance: 0},
		},
		stats: V2CostStats{SessionRows: 1, CostRows: 1, EnergyUsedRows: 1},
	})

	response, _, err := service.BuildCost(context.Background(), "1", V2TimeRange{Period: "month"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.CostPerDistance != nil {
		t.Fatalf("expected distance costs omitted: %#v", response.Summary)
	}
}

func TestV2CostServiceEnergyUsedZeroOmitsCostPerEnergy(t *testing.T) {
	cost := 10.0
	energy := 0.0
	service := NewV2CostService(&fakeV2CostRepository{
		exists: true,
		response: V2CostResponse{
			Summary: V2CostSummaryDetails{ChargingCost: &cost, EnergyUsed: &energy, Distance: 10},
		},
		stats: V2CostStats{SessionRows: 1, CostRows: 1, EnergyUsedRows: 1},
	})

	response, _, err := service.BuildCost(context.Background(), "1", V2TimeRange{Period: "month"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.CostPerEnergy != nil {
		t.Fatalf("expected cost_per_energy omitted: %#v", response.Summary)
	}
}

func TestV2CostServiceRejectsInvalidGroupBy(t *testing.T) {
	service := NewV2CostService(&fakeV2CostRepository{exists: true})
	_, _, err := service.BuildCost(context.Background(), "1", V2TimeRange{}, "hour")
	if !errors.Is(err, errV2InvalidDrivingGroupBy) {
		t.Fatalf("expected invalid group_by, got %v", err)
	}
}
