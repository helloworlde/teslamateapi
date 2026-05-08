package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeV2CalendarRepository struct {
	exists   bool
	driving  map[string]V2CalendarDay
	charging map[string]V2CalendarDay
	updates  map[string]int64
}

func (r *fakeV2CalendarRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2CalendarRepository) DailyDriving(context.Context, int64, timeBound, timeBound, *time.Location) (map[string]V2CalendarDay, error) {
	if r.driving == nil {
		return map[string]V2CalendarDay{}, nil
	}
	return r.driving, nil
}

func (r *fakeV2CalendarRepository) DailyCharging(context.Context, int64, timeBound, timeBound, *time.Location) (map[string]V2CalendarDay, error) {
	if r.charging == nil {
		return map[string]V2CalendarDay{}, nil
	}
	return r.charging, nil
}

func (r *fakeV2CalendarRepository) DailyUpdates(context.Context, int64, timeBound, timeBound, *time.Location) (map[string]int64, error) {
	if r.updates == nil {
		return map[string]int64{}, nil
	}
	return r.updates, nil
}

func TestV2CalendarServiceBuildCalendar(t *testing.T) {
	service := NewV2CalendarService(&fakeV2CalendarRepository{
		exists: true,
		driving: map[string]V2CalendarDay{
			"2024-01-01": {DriveCount: 2, DistanceKM: 50, DriveDurationMin: 60},
			"2024-01-02": {DriveCount: 1, DistanceKM: 150, DriveDurationMin: 90},
		},
		charging: map[string]V2CalendarDay{
			"2024-01-01": {ChargingSessionCount: 1, EnergyAddedKWh: 20},
		},
		updates: map[string]int64{
			"2024-01-02": 1,
		},
	})

	loc, _ := time.LoadLocation("UTC")
	response, quality, carID, err := service.BuildCalendar(context.Background(), "1", V2TimeRange{
		Timezone: "UTC",
		Start:    time.Date(2024, 1, 1, 0, 0, 0, 0, loc),
		End:      time.Date(2024, 1, 3, 0, 0, 0, 0, loc),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carID != 1 {
		t.Fatalf("expected carID 1")
	}
	if len(response.Days) != 2 {
		t.Fatalf("expected 2 days (Jan 1 and Jan 2), got %d", len(response.Days))
	}
	day1 := response.Days[0]
	if day1.Date != "2024-01-01" {
		t.Fatalf("expected 2024-01-01, got %s", day1.Date)
	}
	if day1.DriveCount != 2 || day1.DistanceKM != 50 {
		t.Fatalf("unexpected day1 driving data: %+v", day1)
	}
	if day1.ChargingSessionCount != 1 {
		t.Fatalf("expected 1 charging session on day1")
	}
	if day1.ActivityLevel.Driving != 2 { // 30-100km bucket
		t.Fatalf("expected driving activity 2, got %d", day1.ActivityLevel.Driving)
	}
	day2 := response.Days[1]
	if day2.UpdateCount != 1 {
		t.Fatalf("expected 1 update on day2")
	}
	if day2.ActivityLevel.Driving != 3 { // 101-200km
		t.Fatalf("expected driving activity 3, got %d", day2.ActivityLevel.Driving)
	}
	if quality.SampleCount != 5 { // 2+1+1+1 = 5
		t.Fatalf("expected sample_count 5, got %d", quality.SampleCount)
	}
}

func TestV2CalendarServiceCarNotFound(t *testing.T) {
	service := NewV2CalendarService(&fakeV2CalendarRepository{exists: false})
	_, _, _, err := service.BuildCalendar(context.Background(), "1", V2TimeRange{
		Start: time.Now(),
		End:   time.Now().Add(24 * time.Hour),
	})
	if !errors.Is(err, errV2CarNotFound) {
		t.Fatalf("expected car not found, got %v", err)
	}
}

func TestV2CalendarServiceInvalidCarID(t *testing.T) {
	service := NewV2CalendarService(&fakeV2CalendarRepository{exists: true})
	_, _, _, err := service.BuildCalendar(context.Background(), "bad", V2TimeRange{})
	if err == nil || err.Error() != "invalid car id" {
		t.Fatalf("expected invalid car id error, got %v", err)
	}
}

func TestV2CalendarServiceActivityLevels(t *testing.T) {
	tests := []struct {
		distanceKM float64
		sessions   int64
		drain      float64
		wantDrive  int
		wantCharge int
		wantDrain  int
	}{
		{0, 0, 0, 0, 0, 0},
		{15, 1, 0.5, 1, 1, 1},
		{50, 2, 1.5, 2, 2, 2},
		{150, 3, 3, 3, 3, 3},
		{250, 4, 6, 4, 4, 4},
	}

	for _, tc := range tests {
		day := V2CalendarDay{
			DistanceKM:           tc.distanceKM,
			ChargingSessionCount: tc.sessions,
			VampireDrainPercent:  tc.drain,
		}
		al := computeActivityLevel(day)
		if al.Driving != tc.wantDrive || al.Charging != tc.wantCharge || al.ParkingDrain != tc.wantDrain {
			t.Errorf("distance=%.0f sessions=%d drain=%.1f: got driving=%d charging=%d drain=%d, want %d %d %d",
				tc.distanceKM, tc.sessions, tc.drain, al.Driving, al.Charging, al.ParkingDrain,
				tc.wantDrive, tc.wantCharge, tc.wantDrain)
		}
	}
}

func TestV2CalendarServiceEmptyRange(t *testing.T) {
	service := NewV2CalendarService(&fakeV2CalendarRepository{exists: true})
	loc, _ := time.LoadLocation("UTC")
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	// Same start and end means 0 days
	response, _, _, err := service.BuildCalendar(context.Background(), "1", V2TimeRange{
		Timezone: "UTC",
		Start:    t1,
		End:      t1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Days) != 0 {
		t.Fatalf("expected 0 days for same start/end")
	}
}
