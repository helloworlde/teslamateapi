package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeV2ChargingRepository struct {
	exists     bool
	summary    V2ChargingAnalyticsSummary
	previous   V2ChargingAnalyticsSummary
	stats      V2ChargingStats
	timeseries []V2ChargingTimeseriesItem
	locations  []V2ChargingLocationItem
	types      []V2ChargingTypeItem
	cost       V2ChargingCostResponse
	calls      int
}

func (r *fakeV2ChargingRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2ChargingRepository) Summary(context.Context, int64, timeBound, timeBound) (V2ChargingAnalyticsSummary, V2ChargingStats, error) {
	r.calls++
	if r.calls == 2 {
		return r.previous, r.stats, nil
	}
	return r.summary, r.stats, nil
}

func (r *fakeV2ChargingRepository) Timeseries(context.Context, int64, V2TimeRange, string) ([]V2ChargingTimeseriesItem, V2ChargingStats, error) {
	return r.timeseries, r.stats, nil
}

func (r *fakeV2ChargingRepository) Locations(context.Context, int64, V2TimeRange) ([]V2ChargingLocationItem, V2ChargingStats, error) {
	return r.locations, r.stats, nil
}

func (r *fakeV2ChargingRepository) Types(context.Context, int64, V2TimeRange) ([]V2ChargingTypeItem, V2ChargingStats, error) {
	return r.types, r.stats, nil
}

func (r *fakeV2ChargingRepository) Cost(context.Context, int64, V2TimeRange, string) (V2ChargingCostResponse, V2ChargingStats, error) {
	return r.cost, r.stats, nil
}

func TestV2ChargingServiceBuildChargingWithComparison(t *testing.T) {
	repository := &fakeV2ChargingRepository{
		exists:   true,
		summary:  V2ChargingAnalyticsSummary{SessionCount: 2, EnergyAddedKWh: 80, DurationMin: 120},
		previous: V2ChargingAnalyticsSummary{SessionCount: 1, EnergyAddedKWh: 30, DurationMin: 60},
		stats:    V2ChargingStats{SessionRows: 2, EnergyUsedRows: 2, CostRows: 2, PowerRows: 2},
	}
	service := NewV2ChargingService(repository)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	previousStart := start.Add(-end.Sub(start))

	response, carID, err := service.BuildCharging(context.Background(), "1", V2TimeRange{
		Period:        "custom",
		Timezone:      "UTC",
		Compare:       "previous_period",
		Start:         start,
		End:           end,
		PreviousStart: &previousStart,
		PreviousEnd:   &start,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carID != 1 || response.Summary.EnergyAddedKWh != 80 {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
	comparison := response.Comparison["energy_added_kwh"]
	if comparison.Delta == nil || *comparison.Delta != 50 {
		t.Fatalf("unexpected comparison: %#v", response.Comparison)
	}
}

func TestV2ChargingServiceNoDataReturnsEmpty(t *testing.T) {
	service := NewV2ChargingService(&fakeV2ChargingRepository{exists: true})
	response, _, err := service.BuildCharging(context.Background(), "1", V2TimeRange{Compare: "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.SessionCount != 0 {
		t.Fatalf("expected no sessions, got %#v", response.Summary)
	}
}

func TestV2ChargingServiceMissingCostAndEnergyUsedWarns(t *testing.T) {
	service := NewV2ChargingService(&fakeV2ChargingRepository{
		exists:  true,
		summary: V2ChargingAnalyticsSummary{SessionCount: 2, EnergyAddedKWh: 40},
		stats:   V2ChargingStats{SessionRows: 2, EnergyUsedRows: 1, CostRows: 0},
	})
	_, _, err := service.BuildCharging(context.Background(), "1", V2TimeRange{Compare: "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestV2ChargingServiceRejectsInvalidGroupBy(t *testing.T) {
	service := NewV2ChargingService(&fakeV2ChargingRepository{exists: true})
	_, _, err := service.BuildChargingTimeseries(context.Background(), "1", V2TimeRange{}, "hour")
	if !errors.Is(err, errV2InvalidDrivingGroupBy) {
		t.Fatalf("expected invalid group_by, got %v", err)
	}
}

func TestV2ChargingServiceRejectsInvalidCarID(t *testing.T) {
	service := NewV2ChargingService(&fakeV2ChargingRepository{exists: true})
	_, _, err := service.BuildCharging(context.Background(), "bad", V2TimeRange{})
	if err == nil {
		t.Fatal("expected invalid car id error")
	}
}
