package main

import (
	"context"
	"errors"
	"testing"
)

type fakeV2LocationRepository struct {
	exists bool
	items  []V2LocationAnalyticsItem
	stats  V2LocationStats
}

func (r *fakeV2LocationRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2LocationRepository) Locations(context.Context, int64, V2TimeRange, string) ([]V2LocationAnalyticsItem, V2LocationStats, error) {
	return r.items, r.stats, nil
}

func TestV2LocationServiceBuildLocations(t *testing.T) {
	service := NewV2LocationService(&fakeV2LocationRepository{
		exists: true,
		items: []V2LocationAnalyticsItem{
			{LocationName: "Home", DriveStartCount: 2, DriveEndCount: 1, ChargingSessionCount: 1, ParkingSessionCount: 3, ParkingDurationMin: 120},
		},
		stats: V2LocationStats{DriveStartRows: 2, DriveEndRows: 1, ChargingRows: 1, ParkingRows: 3},
	})

	response, carID, err := service.BuildLocations(context.Background(), "1", V2TimeRange{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carID != 1 || response.Sort != "parking_duration_desc" || len(response.Items) != 1 {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
}

func TestV2LocationServiceUnknownLocation(t *testing.T) {
	service := NewV2LocationService(&fakeV2LocationRepository{
		exists: true,
		items:  []V2LocationAnalyticsItem{{LocationName: "unknown", DriveEndCount: 1}},
		stats:  V2LocationStats{DriveEndRows: 1},
	})

	response, _, err := service.BuildLocations(context.Background(), "1", V2TimeRange{}, "drive_end_count_desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Items[0].LocationName != "unknown" {
		t.Fatalf("unexpected item: %#v", response.Items[0])
	}
}

func TestV2LocationServiceRejectsInvalidSort(t *testing.T) {
	service := NewV2LocationService(&fakeV2LocationRepository{exists: true})
	_, _, err := service.BuildLocations(context.Background(), "1", V2TimeRange{}, "bad")
	if !errors.Is(err, errV2InvalidLocationSort) {
		t.Fatalf("expected invalid sort, got %v", err)
	}
}

func TestV2LocationServiceNoData(t *testing.T) {
	service := NewV2LocationService(&fakeV2LocationRepository{exists: true})
	response, _, err := service.BuildLocations(context.Background(), "1", V2TimeRange{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Items) != 0 {
		t.Fatalf("expected empty items: %#v", response)
	}
}

func TestV2LocationOrderByAllowedSorts(t *testing.T) {
	for _, sort := range []string{"", "drive_start_count_desc", "drive_end_count_desc", "charging_session_count_desc", "parking_duration_desc", "charging_cost_desc"} {
		orderBy, err := v2LocationOrderBy(sort)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", sort, err)
		}
		if orderBy == "" {
			t.Fatalf("empty order by for %q", sort)
		}
	}
}
