package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeV2DrivingRepository struct {
	exists       bool
	summary      V2DrivingAnalyticsSummary
	previous     V2DrivingAnalyticsSummary
	stats        V2DrivingStats
	timeseries   []V2DrivingTimeseriesItem
	distribution []V2DrivingDistributionItem
	ranking      []V2DrivingRankingItem
	calls        int
}

func (r *fakeV2DrivingRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2DrivingRepository) Summary(context.Context, int64, timeBound, timeBound) (V2DrivingAnalyticsSummary, V2DrivingStats, error) {
	r.calls++
	if r.calls == 2 {
		return r.previous, r.stats, nil
	}
	return r.summary, r.stats, nil
}

func (r *fakeV2DrivingRepository) Timeseries(context.Context, int64, V2TimeRange, string) ([]V2DrivingTimeseriesItem, V2DrivingStats, error) {
	return r.timeseries, r.stats, nil
}

func (r *fakeV2DrivingRepository) Distribution(context.Context, int64, V2TimeRange, string) ([]V2DrivingDistributionItem, V2DrivingStats, error) {
	return r.distribution, r.stats, nil
}

func (r *fakeV2DrivingRepository) Ranking(context.Context, int64, V2TimeRange, string, int) ([]V2DrivingRankingItem, V2DrivingStats, error) {
	return r.ranking, r.stats, nil
}

func TestV2DrivingServiceBuildDrivingWithComparison(t *testing.T) {
	repository := &fakeV2DrivingRepository{
		exists:   true,
		summary:  V2DrivingAnalyticsSummary{DriveCount: 2, DistanceKM: 100, DurationMin: 60},
		previous: V2DrivingAnalyticsSummary{DriveCount: 1, DistanceKM: 40, DurationMin: 30},
		stats:    V2DrivingStats{DriveRows: 2, EnergyEstimateRows: 2, TemperatureRows: 1},
	}
	service := NewV2DrivingService(repository)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	previousStart := start.Add(-end.Sub(start))

	response, quality, carID, err := service.BuildDriving(context.Background(), "1", V2TimeRange{
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
	if carID != 1 || response.Summary.DistanceKM != 100 {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
	comparison := response.Comparison["distance_km"]
	if comparison.Delta == nil || *comparison.Delta != 60 {
		t.Fatalf("unexpected comparison: %#v", response.Comparison)
	}
	if quality.SampleCount != 2 {
		t.Fatalf("unexpected sample count: %d", quality.SampleCount)
	}
}

func TestV2DrivingServiceNoDataReturnsEmpty(t *testing.T) {
	service := NewV2DrivingService(&fakeV2DrivingRepository{exists: true})
	response, _, _, err := service.BuildDriving(context.Background(), "1", V2TimeRange{Compare: "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.DriveCount != 0 {
		t.Fatalf("expected no drives, got %#v", response.Summary)
	}
}

func TestV2DrivingServiceRejectsInvalidDimension(t *testing.T) {
	service := NewV2DrivingService(&fakeV2DrivingRepository{exists: true})
	_, _, _, err := service.BuildDistribution(context.Background(), "1", V2TimeRange{}, "bad")
	if !errors.Is(err, errV2InvalidDrivingDimension) {
		t.Fatalf("expected invalid dimension, got %v", err)
	}
}

func TestV2DrivingServiceRejectsInvalidRankingType(t *testing.T) {
	service := NewV2DrivingService(&fakeV2DrivingRepository{exists: true})
	_, _, _, err := service.BuildRanking(context.Background(), "1", V2TimeRange{}, "bad", 10)
	if !errors.Is(err, errV2InvalidDrivingRanking) {
		t.Fatalf("expected invalid ranking type, got %v", err)
	}
}

func TestV2DrivingServiceRejectsInvalidGroupBy(t *testing.T) {
	service := NewV2DrivingService(&fakeV2DrivingRepository{exists: true})
	_, _, _, err := service.BuildTimeseries(context.Background(), "1", V2TimeRange{}, "hour")
	if !errors.Is(err, errV2InvalidDrivingGroupBy) {
		t.Fatalf("expected invalid group_by, got %v", err)
	}
}

func TestV2DrivingServiceRejectsInvalidCarID(t *testing.T) {
	service := NewV2DrivingService(&fakeV2DrivingRepository{exists: true})
	_, _, _, err := service.BuildDriving(context.Background(), "bad", V2TimeRange{})
	if err == nil {
		t.Fatal("expected invalid car id error")
	}
}
