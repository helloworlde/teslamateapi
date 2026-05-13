package v2

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeV2SummaryRepository struct {
	exists     bool
	summaries  []V2Summary
	stats      V2SummaryStats
	existsErr  error
	summaryErr error
	calls      int
}

func (r *fakeV2SummaryRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, r.existsErr
}

func (r *fakeV2SummaryRepository) Summary(context.Context, int64, timeBound, timeBound) (V2Summary, V2SummaryStats, error) {
	if r.summaryErr != nil {
		return V2Summary{}, V2SummaryStats{}, r.summaryErr
	}
	index := r.calls
	r.calls++
	if index >= len(r.summaries) {
		return V2Summary{}, r.stats, nil
	}
	return r.summaries[index], r.stats, nil
}

func TestV2SummaryServiceBuildSummaryWithComparison(t *testing.T) {
	repository := &fakeV2SummaryRepository{
		exists: true,
		summaries: []V2Summary{
			{
				Driving:  V2DrivingSummary{DriveCount: 2, Distance: 120, Duration: 5400},
				Charging: V2ChargingSummary{SessionCount: 1, EnergyAdded: 30, EnergyUsed: 32, Cost: 60},
				Cost:     V2CostSummary{ChargingCost: 60},
			},
			{
				Driving:  V2DrivingSummary{DriveCount: 1, Distance: 80, Duration: 3000},
				Charging: V2ChargingSummary{SessionCount: 1, EnergyAdded: 20, EnergyUsed: 22, Cost: 40},
				Cost:     V2CostSummary{ChargingCost: 40},
			},
		},
		stats: V2SummaryStats{DriveRows: 2, ChargeRows: 1, CostRows: 1, StateRows: 1},
	}
	service := NewV2SummaryService(repository)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	previousStart := start.Add(-end.Sub(start))
	timeRange := V2TimeRange{
		Period:        "custom",
		Timezone:      "UTC",
		Compare:       "previous_period",
		Start:         start,
		End:           end,
		PreviousStart: &previousStart,
		PreviousEnd:   &start,
	}

	response, err := service.BuildSummary(context.Background(), "1", timeRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.Driving.Distance != 120 {
		t.Fatalf("unexpected distance: %v", response.Summary.Driving.Distance)
	}
	comparison, ok := response.Comparison["driving.distance"]
	if !ok || comparison.Delta == nil || *comparison.Delta != 40 {
		t.Fatalf("unexpected comparison: %#v", response.Comparison)
	}
}

func TestV2SummaryServiceNoDataReturnsEmptySummary(t *testing.T) {
	repository := &fakeV2SummaryRepository{exists: true}
	service := NewV2SummaryService(repository)
	timeRange := V2TimeRange{
		Period:   "month",
		Timezone: "UTC",
		Compare:  "none",
		Start:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}

	response, err := service.BuildSummary(context.Background(), "1", timeRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.Driving.DriveCount != 0 || response.Summary.Charging.SessionCount != 0 {
		t.Fatalf("expected empty summary, got %#v", response.Summary)
	}
}

func TestV2SummaryServiceRejectsInvalidCarID(t *testing.T) {
	service := NewV2SummaryService(&fakeV2SummaryRepository{exists: true})
	_, err := service.BuildSummary(context.Background(), "not-an-int", V2TimeRange{})
	if err == nil {
		t.Fatal("expected invalid car id error")
	}
}

func TestV2SummaryServiceReturnsCarNotFound(t *testing.T) {
	service := NewV2SummaryService(&fakeV2SummaryRepository{exists: false})
	_, err := service.BuildSummary(context.Background(), "42", V2TimeRange{})
	if !errors.Is(err, errV2CarNotFound) {
		t.Fatalf("expected car not found, got %v", err)
	}
}
