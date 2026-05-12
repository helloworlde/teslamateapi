package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeV2DrivingRepository struct {
	exists     bool
	summary    V2DrivingSummary
	stats      V2DrivingStats
	timeseries []V2DrivingTimeseriesItem
}

func (r *fakeV2DrivingRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2DrivingRepository) Summary(context.Context, int64, timeBound, timeBound) (V2DrivingSummary, V2DrivingStats, error) {
	return r.summary, r.stats, nil
}

func (r *fakeV2DrivingRepository) Timeseries(context.Context, int64, V2TimeRange, string) ([]V2DrivingTimeseriesItem, V2DrivingStats, error) {
	return r.timeseries, r.stats, nil
}

func TestV2DrivingServiceBuildDrivingReturnsSummary(t *testing.T) {
	repository := &fakeV2DrivingRepository{
		exists:  true,
		summary: V2DrivingSummary{DriveCount: 2, Distance: 100, Duration: 3600},
		stats:   V2DrivingStats{DriveRows: 2, EnergyEstimateRows: 2, TemperatureRows: 1},
	}
	service := NewV2DrivingService(repository)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)

	response, carID, err := service.BuildDriving(context.Background(), "1", V2TimeRange{
		Period:   "custom",
		Timezone: "UTC",
		Compare:  "none",
		Start:    start,
		End:      end,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carID != 1 || response.Summary.Distance != 100 {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
}

func TestV2DrivingServiceNoDataReturnsEmpty(t *testing.T) {
	service := NewV2DrivingService(&fakeV2DrivingRepository{exists: true})
	response, _, err := service.BuildDriving(context.Background(), "1", V2TimeRange{Compare: "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.DriveCount != 0 {
		t.Fatalf("expected no drives, got %#v", response.Summary)
	}
}

func TestV2DrivingServiceRejectsInvalidGroupBy(t *testing.T) {
	service := NewV2DrivingService(&fakeV2DrivingRepository{exists: true})
	_, _, err := service.BuildTimeseries(context.Background(), "1", V2TimeRange{}, "hour")
	if !errors.Is(err, errV2InvalidDrivingGroupBy) {
		t.Fatalf("expected invalid group_by, got %v", err)
	}
}

func TestV2DrivingServiceRejectsInvalidCarID(t *testing.T) {
	service := NewV2DrivingService(&fakeV2DrivingRepository{exists: true})
	_, _, err := service.BuildDriving(context.Background(), "bad", V2TimeRange{})
	if err == nil {
		t.Fatal("expected invalid car id error")
	}
}
