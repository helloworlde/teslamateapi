package main

import (
	"context"
	"errors"
	"testing"
)

type fakeV2EfficiencyRepository struct {
	exists  bool
	summary V2EfficiencySummary
	stats   V2EfficiencyStats
	factors []V2EfficiencyFactorItem
}

func (r *fakeV2EfficiencyRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2EfficiencyRepository) Summary(context.Context, int64, V2TimeRange) (V2EfficiencySummary, V2EfficiencyStats, error) {
	return r.summary, r.stats, nil
}

func (r *fakeV2EfficiencyRepository) Factors(context.Context, int64, V2TimeRange, string) ([]V2EfficiencyFactorItem, V2EfficiencyStats, error) {
	return r.factors, r.stats, nil
}

func TestV2EfficiencyServiceBuildSummary(t *testing.T) {
	energy := 12.5
	avgConsumption := 155.0
	service := NewV2EfficiencyService(&fakeV2EfficiencyRepository{
		exists: true,
		summary: V2EfficiencySummary{
			DriveCount:                 2,
			DistanceKM:                 80,
			EstimatedEnergyConsumedKWh: &energy,
			AvgConsumptionWhPerKM:      &avgConsumption,
		},
		stats: V2EfficiencyStats{DriveRows: 2, EnergyEstimateRows: 2, TemperatureRows: 2},
	})

	response, quality, carID, err := service.BuildEfficiency(context.Background(), "1", V2TimeRange{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carID != 1 || response.Summary.DriveCount != 2 || response.Summary.EstimatedEnergyConsumedKWh == nil {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
	if !quality.Complete || quality.SampleCount != 2 {
		t.Fatalf("unexpected quality: %#v", quality)
	}
}

func TestV2EfficiencyServiceNoDrivesWarns(t *testing.T) {
	service := NewV2EfficiencyService(&fakeV2EfficiencyRepository{exists: true})

	response, quality, _, err := service.BuildEfficiency(context.Background(), "1", V2TimeRange{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.DriveCount != 0 || quality.Complete {
		t.Fatalf("expected no-drive warning: response=%#v quality=%#v", response, quality)
	}
}

func TestV2EfficiencyServiceRejectsInvalidDimension(t *testing.T) {
	service := NewV2EfficiencyService(&fakeV2EfficiencyRepository{exists: true})
	_, _, _, err := service.BuildEfficiencyFactors(context.Background(), "1", V2TimeRange{}, "bad")
	if !errors.Is(err, errV2InvalidEfficiencyDimension) {
		t.Fatalf("expected invalid dimension, got %v", err)
	}
}

func TestEfficiencyFactorBucketSQL(t *testing.T) {
	for _, dimension := range []string{"temperature", "speed", "distance", "elevation", "location", "hour_of_day", "day_of_week"} {
		bucketSQL, _, err := efficiencyFactorBucketSQL(dimension)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", dimension, err)
		}
		if bucketSQL == "" {
			t.Fatalf("empty bucket SQL for %s", dimension)
		}
	}
	if _, _, err := efficiencyFactorBucketSQL("bad"); !errors.Is(err, errV2InvalidEfficiencyDimension) {
		t.Fatalf("expected invalid dimension, got %v", err)
	}
}
