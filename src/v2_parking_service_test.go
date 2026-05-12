package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeV2ParkingRepository struct {
	exists    bool
	summary   V2ParkingSummary
	stats     V2ParkingStats
	locations []V2ParkingLocationItem
	states    V2ParkingStatesResponse
}

func (r *fakeV2ParkingRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2ParkingRepository) Summary(context.Context, int64, V2TimeRange) (V2ParkingSummary, V2ParkingStats, error) {
	return r.summary, r.stats, nil
}

func (r *fakeV2ParkingRepository) Locations(context.Context, int64, V2TimeRange) ([]V2ParkingLocationItem, V2ParkingStats, error) {
	return r.locations, r.stats, nil
}

func (r *fakeV2ParkingRepository) StateBreakdown(context.Context, int64, V2TimeRange) (V2ParkingStatesResponse, V2ParkingStats, error) {
	return r.states, r.stats, nil
}

func TestV2ParkingServiceBuildParkingReturnsSummary(t *testing.T) {
	repository := &fakeV2ParkingRepository{
		exists:  true,
		summary: V2ParkingSummary{ParkingSessionCount: 3, ParkedDuration: 18000, AsleepDuration: 7200},
		stats:   V2ParkingStats{StateRows: 3, SessionRows: 3, PositionRows: 2},
	}
	service := NewV2ParkingService(repository)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)

	response, carID, err := service.BuildParking(context.Background(), "1", V2TimeRange{
		Period:   "custom",
		Timezone: "UTC",
		Compare:  "none",
		Start:    start,
		End:      end,
	}, V2ParkingBuildOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carID != 1 || response.Summary.ParkedDuration != 18000 {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
}

func TestV2ParkingServiceNoStatesWarns(t *testing.T) {
	service := NewV2ParkingService(&fakeV2ParkingRepository{exists: true})
	_, _, err := service.BuildParking(context.Background(), "1", V2TimeRange{Compare: "none"}, V2ParkingBuildOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestV2ParkingServiceIncludeLocationBreakdown(t *testing.T) {
	service := NewV2ParkingService(&fakeV2ParkingRepository{
		exists:    true,
		locations: []V2ParkingLocationItem{{LocationName: "Home", ParkingSessionCount: 1, ParkedDuration: 3600}},
		stats:     V2ParkingStats{SessionRows: 1},
	})
	resp, _, err := service.BuildParking(context.Background(), "1", V2TimeRange{}, V2ParkingBuildOptions{
		IncludeBreakdown: true,
		BreakdownBy:      "location",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Breakdown == nil || resp.Breakdown.By != "location" || len(resp.Breakdown.Locations) != 1 {
		t.Fatalf("expected location breakdown, got %#v", resp.Breakdown)
	}
}

func TestV2ParkingServiceIncludeStateBreakdown(t *testing.T) {
	service := NewV2ParkingService(&fakeV2ParkingRepository{
		exists: true,
		states: V2ParkingStatesResponse{
			TotalDuration:        6000,
			StateTransitionCount: 2,
			Items: []V2ParkingStateItem{
				{State: "asleep", Duration: 4800, Percent: 80, TransitionCount: 1},
				{State: "online", Duration: 1200, Percent: 20, TransitionCount: 1},
			},
		},
		stats: V2ParkingStats{StateRows: 2},
	})
	resp, _, err := service.BuildParking(context.Background(), "1", V2TimeRange{}, V2ParkingBuildOptions{
		IncludeBreakdown: true,
		BreakdownBy:      "state",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Breakdown == nil || resp.Breakdown.By != "state" || len(resp.Breakdown.States) != 2 {
		t.Fatalf("expected state breakdown, got %#v", resp.Breakdown)
	}
}

func TestV2ParkingServiceRejectsInvalidBreakdown(t *testing.T) {
	service := NewV2ParkingService(&fakeV2ParkingRepository{exists: true})
	_, _, err := service.BuildParking(context.Background(), "1", V2TimeRange{}, V2ParkingBuildOptions{
		IncludeBreakdown: true,
		BreakdownBy:      "rainbow",
	})
	if !errors.Is(err, errV2InvalidParkingBreakdown) {
		t.Fatalf("expected invalid breakdown, got %v", err)
	}
}

func TestV2ParkingServiceRejectsInvalidCarID(t *testing.T) {
	service := NewV2ParkingService(&fakeV2ParkingRepository{exists: true})
	_, _, err := service.BuildParking(context.Background(), "bad", V2TimeRange{}, V2ParkingBuildOptions{})
	if err == nil {
		t.Fatal("expected invalid car id error")
	}
}
