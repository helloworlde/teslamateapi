package v2

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeV2LifecycleRepository struct {
	exists    bool
	lifecycle V2LifecycleResponse
	events    []V2TimelineEvent
	hasMore   bool
	err       error
}

func (r *fakeV2LifecycleRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2LifecycleRepository) Lifecycle(_ context.Context, _ int64, _ time.Time) (V2LifecycleResponse, error) {
	return r.lifecycle, r.err
}

func (r *fakeV2LifecycleRepository) Timeline(_ context.Context, _ int64, _ []string, _ int, _, _ *time.Time) ([]V2TimelineEvent, bool, *time.Time, error) {
	return r.events, r.hasMore, nil, r.err
}

func TestV2LifecycleServiceBuildLifecycle(t *testing.T) {
	firstAt := "2023-01-01T00:00:00Z"
	lastAt := "2024-01-01T00:00:00Z"
	daily := 50.0
	monthly := daily * 30.44
	service := NewV2LifecycleService(&fakeV2LifecycleRepository{
		exists: true,
		lifecycle: V2LifecycleResponse{
			FirstRecordedAt:      &firstAt,
			LastRecordedAt:       &lastAt,
			RecordedDays:         365,
			DriveCount:           100,
			Distance:             5000,
			ChargingSessionCount: 50,
			EnergyAdded:          1000,
			UpdateCount:          5,
			AvgDailyDistance:     &daily,
			AvgMonthlyDistance:   &monthly,
		},
	})
	response, err := service.BuildLifecycle(context.Background(), "1", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.DriveCount != 100 {
		t.Fatalf("expected drive_count 100, got %d", response.DriveCount)
	}
	if response.Distance != 5000 {
		t.Fatalf("expected distance 5000, got %f", response.Distance)
	}
}

func TestV2LifecycleServiceCarNotFound(t *testing.T) {
	service := NewV2LifecycleService(&fakeV2LifecycleRepository{exists: false})
	_, err := service.BuildLifecycle(context.Background(), "1", time.Time{})
	if !errors.Is(err, errV2CarNotFound) {
		t.Fatalf("expected car not found, got %v", err)
	}
}

func TestV2LifecycleServiceInvalidCarID(t *testing.T) {
	service := NewV2LifecycleService(&fakeV2LifecycleRepository{exists: true})
	_, err := service.BuildLifecycle(context.Background(), "bad", time.Time{})
	if err == nil || err.Error() != "invalid car id" {
		t.Fatalf("expected invalid car id, got %v", err)
	}
}

func TestV2LifecycleServiceBuildTimeline(t *testing.T) {
	service := NewV2LifecycleService(&fakeV2LifecycleRepository{
		exists: true,
		events: []V2TimelineEvent{
			{Type: "drive", ID: 1, StartTime: "2024-01-01T10:00:00Z", Title: "Drive 50.0 km"},
			{Type: "charging", ID: 2, StartTime: "2024-01-01T12:00:00Z", Title: "Charging 20.00 kWh"},
		},
		hasMore: false,
	})
	response, err := service.BuildTimeline(context.Background(), "1", nil, 50, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(response.Events))
	}
	if response.Total != 2 {
		t.Fatalf("expected total 2, got %d", response.Total)
	}
}

func TestV2LifecycleServiceTimelineEmpty(t *testing.T) {
	service := NewV2LifecycleService(&fakeV2LifecycleRepository{
		exists: true,
		events: nil,
	})
	response, err := service.BuildTimeline(context.Background(), "1", nil, 50, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Events) != 0 {
		t.Fatalf("expected empty events")
	}
}

func TestV2LifecycleServiceTimelineDefaultLimit(t *testing.T) {
	service := NewV2LifecycleService(&fakeV2LifecycleRepository{exists: true})
	response, err := service.BuildTimeline(context.Background(), "1", nil, 0, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = response
}
