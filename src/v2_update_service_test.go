package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

type fakeV2UpdateRepository struct {
	exists   bool
	response V2UpdateAnalyticsResponse
	count    int64
	err      error
}

func (r *fakeV2UpdateRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2UpdateRepository) Updates(context.Context, int64, timeBound, timeBound) (V2UpdateAnalyticsResponse, int64, error) {
	return r.response, r.count, r.err
}

func TestV2UpdateServiceNormalRecords(t *testing.T) {
	version := "2024.12.1"
	completedAt := "2024-12-01T10:30:00Z"
	duration := 45.5
	service := NewV2UpdateService(&fakeV2UpdateRepository{
		exists: true,
		response: V2UpdateAnalyticsResponse{
			UpdateCount:          2,
			LatestVersion:        &version,
			AvgUpdateDurationMin: &duration,
			Versions: []V2UpdateVersion{
				{
					Version:     "2024.12.1",
					StartedAt:   "2024-12-01T10:00:00Z",
					CompletedAt: &completedAt,
					DurationMin: &duration,
				},
			},
		},
		count: 2,
	})
	response, carID, err := service.BuildUpdates(context.Background(), "1", V2TimeRange{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carID != 1 {
		t.Fatalf("expected carID 1, got %d", carID)
	}
	if response.UpdateCount != 2 {
		t.Fatalf("expected update_count 2, got %d", response.UpdateCount)
	}
	if response.LatestVersion == nil || *response.LatestVersion != version {
		t.Fatalf("expected latest_version %q", version)
	}
}

func TestV2UpdateServiceNoRecords(t *testing.T) {
	service := NewV2UpdateService(&fakeV2UpdateRepository{
		exists:   true,
		response: V2UpdateAnalyticsResponse{UpdateCount: 0, Versions: []V2UpdateVersion{}},
		count:    0,
	})
	response, _, err := service.BuildUpdates(context.Background(), "1", V2TimeRange{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.UpdateCount != 0 {
		t.Fatalf("expected update_count 0")
	}
	if response.LatestVersion != nil {
		t.Fatalf("expected no latest_version")
	}
	if len(response.Versions) != 0 {
		t.Fatalf("expected empty versions")
	}
}

func TestV2UpdateServiceInvalidCarID(t *testing.T) {
	service := NewV2UpdateService(&fakeV2UpdateRepository{exists: true})
	_, _, err := service.BuildUpdates(context.Background(), "abc", V2TimeRange{})
	if err == nil || err.Error() != "invalid car id" {
		t.Fatalf("expected invalid car id error, got %v", err)
	}
}

func TestV2UpdateServiceCarNotFound(t *testing.T) {
	service := NewV2UpdateService(&fakeV2UpdateRepository{exists: false})
	_, _, err := service.BuildUpdates(context.Background(), "1", V2TimeRange{})
	if !errors.Is(err, errV2CarNotFound) {
		t.Fatalf("expected car not found error, got %v", err)
	}
}

func TestV2UpdateServiceDefaultTimezone(t *testing.T) {
	_ = os.Setenv("TZ", "America/New_York")
	defer os.Unsetenv("TZ")

	service := NewV2UpdateService(&fakeV2UpdateRepository{
		exists:   true,
		response: V2UpdateAnalyticsResponse{UpdateCount: 0, Versions: []V2UpdateVersion{}},
		count:    0,
	})
	timeRange := V2TimeRange{
		Timezone: "America/New_York",
		Start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	_, _, err := service.BuildUpdates(context.Background(), "1", timeRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestV2UpdateServiceTimezoneOverride(t *testing.T) {
	service := NewV2UpdateService(&fakeV2UpdateRepository{
		exists:   true,
		response: V2UpdateAnalyticsResponse{UpdateCount: 0, Versions: []V2UpdateVersion{}},
		count:    0,
	})
	timeRange := V2TimeRange{
		Timezone: "Asia/Tokyo",
		Start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	_, _, err := service.BuildUpdates(context.Background(), "1", timeRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
