package main

import (
	"context"
	"testing"
	"time"
)

type fakeV2ParkingRepository struct {
	exists    bool
	summary   V2ParkingAnalyticsSummary
	previous  V2ParkingAnalyticsSummary
	stats     V2ParkingStats
	locations []V2ParkingLocationItem
	states    V2ParkingStatesResponse
	calls     int
}

func (r *fakeV2ParkingRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2ParkingRepository) Summary(context.Context, int64, V2TimeRange) (V2ParkingAnalyticsSummary, V2ParkingStats, error) {
	r.calls++
	if r.calls == 2 {
		return r.previous, r.stats, nil
	}
	return r.summary, r.stats, nil
}

func (r *fakeV2ParkingRepository) Locations(context.Context, int64, V2TimeRange) ([]V2ParkingLocationItem, V2ParkingStats, error) {
	return r.locations, r.stats, nil
}

func (r *fakeV2ParkingRepository) StateBreakdown(context.Context, int64, V2TimeRange) (V2ParkingStatesResponse, V2ParkingStats, error) {
	return r.states, r.stats, nil
}

func TestV2ParkingServiceBuildParkingWithComparison(t *testing.T) {
	repository := &fakeV2ParkingRepository{
		exists:   true,
		summary:  V2ParkingAnalyticsSummary{ParkingSessionCount: 3, ParkedDurationMin: 300, AsleepDurationMin: 120},
		previous: V2ParkingAnalyticsSummary{ParkingSessionCount: 2, ParkedDurationMin: 100, AsleepDurationMin: 80},
		stats:    V2ParkingStats{StateRows: 3, SessionRows: 3, PositionRows: 2},
	}
	service := NewV2ParkingService(repository)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	previousStart := start.Add(-end.Sub(start))

	response, quality, carID, err := service.BuildParking(context.Background(), "1", V2TimeRange{
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
	if carID != 1 || response.Summary.ParkedDurationMin != 300 {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
	comparison := response.Comparison["parked_duration_min"]
	if comparison.Delta == nil || *comparison.Delta != 200 {
		t.Fatalf("unexpected comparison: %#v", response.Comparison)
	}
	if quality.SampleCount != 6 {
		t.Fatalf("unexpected sample count: %#v", quality)
	}
}

func TestV2ParkingServiceNoStatesWarns(t *testing.T) {
	service := NewV2ParkingService(&fakeV2ParkingRepository{exists: true})
	_, quality, _, err := service.BuildParking(context.Background(), "1", V2TimeRange{Compare: "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quality.Complete {
		t.Fatalf("expected incomplete quality: %#v", quality)
	}
	if len(quality.MissingFields) == 0 {
		t.Fatalf("expected missing fields: %#v", quality)
	}
}

func TestV2ParkingServiceLocationsAddsInferenceWarning(t *testing.T) {
	service := NewV2ParkingService(&fakeV2ParkingRepository{
		exists:    true,
		locations: []V2ParkingLocationItem{{LocationName: "Home", ParkingSessionCount: 1, ParkedDurationMin: 60}},
		stats:     V2ParkingStats{SessionRows: 1},
	})
	response, quality, _, err := service.BuildParkingLocations(context.Background(), "1", V2TimeRange{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Items) != 1 || quality.Complete {
		t.Fatalf("unexpected response quality: response=%#v quality=%#v", response, quality)
	}
}

func TestV2ParkingServiceStatesPercent(t *testing.T) {
	service := NewV2ParkingService(&fakeV2ParkingRepository{
		exists: true,
		states: V2ParkingStatesResponse{
			TotalDurationMin:     100,
			StateTransitionCount: 2,
			Items: []V2ParkingStateItem{
				{State: "asleep", DurationMin: 80, Percent: 80, TransitionCount: 1},
				{State: "online", DurationMin: 20, Percent: 20, TransitionCount: 1},
			},
		},
		stats: V2ParkingStats{StateRows: 2},
	})
	response, _, _, err := service.BuildParkingStates(context.Background(), "1", V2TimeRange{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StateTransitionCount != 2 || response.Items[0].Percent != 80 {
		t.Fatalf("unexpected state response: %#v", response)
	}
}

func TestV2ParkingServiceRejectsInvalidCarID(t *testing.T) {
	service := NewV2ParkingService(&fakeV2ParkingRepository{exists: true})
	_, _, _, err := service.BuildParking(context.Background(), "bad", V2TimeRange{})
	if err == nil {
		t.Fatal("expected invalid car id error")
	}
}
