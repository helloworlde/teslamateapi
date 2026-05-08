package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeV2BatteryRepository struct {
	exists       bool
	summary      V2BatteryAnalyticsSummary
	previous     V2BatteryAnalyticsSummary
	stats        V2BatteryStats
	timeseries   []V2BatteryTimeseriesItem
	distribution []V2BatteryDistributionItem
	calls        int
}

func (r *fakeV2BatteryRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2BatteryRepository) Summary(context.Context, int64, V2TimeRange) (V2BatteryAnalyticsSummary, V2BatteryStats, error) {
	r.calls++
	if r.calls == 2 {
		return r.previous, r.stats, nil
	}
	return r.summary, r.stats, nil
}

func (r *fakeV2BatteryRepository) Timeseries(context.Context, int64, V2TimeRange, string) ([]V2BatteryTimeseriesItem, V2BatteryStats, error) {
	return r.timeseries, r.stats, nil
}

func (r *fakeV2BatteryRepository) Distribution(context.Context, int64, V2TimeRange) ([]V2BatteryDistributionItem, V2BatteryStats, error) {
	return r.distribution, r.stats, nil
}

func TestV2BatteryServiceBuildBatteryWithComparison(t *testing.T) {
	currentRated := 420.0
	previousRated := 430.0
	currentDegradation := 3.0
	repository := &fakeV2BatteryRepository{
		exists: true,
		summary: V2BatteryAnalyticsSummary{
			EstimatedRatedRangeAt100PercentKM: &currentRated,
			EstimatedRangeDegradationPercent:  &currentDegradation,
			SampleCount:                       8,
		},
		previous: V2BatteryAnalyticsSummary{
			EstimatedRatedRangeAt100PercentKM: &previousRated,
			SampleCount:                       6,
		},
		stats: V2BatteryStats{SampleRows: 8},
	}
	service := NewV2BatteryService(repository)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	previousStart := start.Add(-end.Sub(start))

	response, quality, carID, err := service.BuildBattery(context.Background(), "1", V2TimeRange{
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
	if carID != 1 || response.Summary.EstimatedRatedRangeAt100PercentKM == nil {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
	comparison := response.Comparison["estimated_rated_range_at_100_percent_km"]
	if comparison.Delta == nil || *comparison.Delta != -10 {
		t.Fatalf("unexpected comparison: %#v", response.Comparison)
	}
	if !quality.Complete || len(quality.Warnings) == 0 {
		t.Fatalf("expected estimate warnings: %#v", quality)
	}
}

func TestV2BatteryServiceInsufficientSamplesOmitDegradation(t *testing.T) {
	repository := &fakeV2BatteryRepository{
		exists:  true,
		summary: V2BatteryAnalyticsSummary{SampleCount: 2},
		stats:   V2BatteryStats{SampleRows: 2},
	}
	service := NewV2BatteryService(repository)

	response, quality, _, err := service.BuildBattery(context.Background(), "1", V2TimeRange{Compare: "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.EstimatedRangeDegradationPercent != nil {
		t.Fatalf("expected nil degradation: %#v", response.Summary)
	}
	if quality.Complete {
		t.Fatalf("expected incomplete quality: %#v", quality)
	}
}

func TestV2BatteryServiceInvalidBatteryLevelWarns(t *testing.T) {
	service := NewV2BatteryService(&fakeV2BatteryRepository{
		exists:  true,
		summary: V2BatteryAnalyticsSummary{SampleCount: 1},
		stats:   V2BatteryStats{SampleRows: 1, InvalidBatteryRows: 1},
	})

	_, quality, _, err := service.BuildBattery(context.Background(), "1", V2TimeRange{Compare: "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quality.Complete || len(quality.MissingFields) == 0 {
		t.Fatalf("expected invalid battery warning: %#v", quality)
	}
}

func TestV2BatteryServiceDistributionBuckets(t *testing.T) {
	service := NewV2BatteryService(&fakeV2BatteryRepository{
		exists: true,
		distribution: []V2BatteryDistributionItem{
			{Bucket: "0-10", MinBatteryLevelPercent: 0, MaxBatteryLevelPercent: 10, SampleCount: 1, Percent: 25},
			{Bucket: "90-100", MinBatteryLevelPercent: 90, MaxBatteryLevelPercent: 100, SampleCount: 3, Percent: 75},
		},
		stats: V2BatteryStats{SampleRows: 4},
	})

	response, quality, _, err := service.BuildBatteryDistribution(context.Background(), "1", V2TimeRange{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Items) != 2 || response.Items[1].Bucket != "90-100" || quality.SampleCount != 4 {
		t.Fatalf("unexpected distribution: response=%#v quality=%#v", response, quality)
	}
}

func TestV2BatteryServiceRejectsInvalidGroupBy(t *testing.T) {
	service := NewV2BatteryService(&fakeV2BatteryRepository{exists: true})
	_, _, _, err := service.BuildBatteryTimeseries(context.Background(), "1", V2TimeRange{}, "hour")
	if !errors.Is(err, errV2InvalidDrivingGroupBy) {
		t.Fatalf("expected invalid group_by, got %v", err)
	}
}
