package v2

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeV2BatteryRepository struct {
	exists     bool
	summary    V2BatterySummary
	stats      V2BatteryStats
	timeseries []V2BatteryTimeseriesItem
}

func (r *fakeV2BatteryRepository) CarExists(context.Context, int64) (bool, error) {
	return r.exists, nil
}

func (r *fakeV2BatteryRepository) Summary(context.Context, int64, V2TimeRange) (V2BatterySummary, V2BatteryStats, error) {
	return r.summary, r.stats, nil
}

func (r *fakeV2BatteryRepository) Timeseries(context.Context, int64, V2TimeRange, string) ([]V2BatteryTimeseriesItem, V2BatteryStats, error) {
	return r.timeseries, r.stats, nil
}

func TestV2BatteryServiceBuildBatteryReturnsSummary(t *testing.T) {
	currentRated := 420.0
	currentDegradation := 3.0
	repository := &fakeV2BatteryRepository{
		exists: true,
		summary: V2BatterySummary{
			RangeAtFullCharge:         &V2BatteryRange{Rated: &currentRated},
			EstimatedRangeDegradation: &currentDegradation,
		},
		stats: V2BatteryStats{SampleRows: 8},
	}
	service := NewV2BatteryService(repository)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)

	response, carID, err := service.BuildBattery(context.Background(), "1", V2TimeRange{
		Period:   "custom",
		Timezone: "UTC",
		Compare:  "none",
		Start:    start,
		End:      end,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carID != 1 || response.Summary.RangeAtFullCharge == nil || response.Summary.RangeAtFullCharge.Rated == nil {
		t.Fatalf("unexpected response: carID=%d response=%#v", carID, response)
	}
}

func TestV2BatteryServiceInsufficientSamplesOmitDegradation(t *testing.T) {
	repository := &fakeV2BatteryRepository{
		exists:  true,
		summary: V2BatterySummary{},
		stats:   V2BatteryStats{SampleRows: 2},
	}
	service := NewV2BatteryService(repository)

	response, _, err := service.BuildBattery(context.Background(), "1", V2TimeRange{Compare: "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Summary.EstimatedRangeDegradation != nil {
		t.Fatalf("expected nil degradation: %#v", response.Summary)
	}
}

func TestV2BatteryServiceInvalidBatteryLevelWarns(t *testing.T) {
	service := NewV2BatteryService(&fakeV2BatteryRepository{
		exists:  true,
		summary: V2BatterySummary{},
		stats:   V2BatteryStats{SampleRows: 1, InvalidBatteryRows: 1},
	})

	_, _, err := service.BuildBattery(context.Background(), "1", V2TimeRange{Compare: "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestV2BatteryServiceRejectsInvalidGroupBy(t *testing.T) {
	service := NewV2BatteryService(&fakeV2BatteryRepository{exists: true})
	_, _, err := service.BuildBatteryTimeseries(context.Background(), "1", V2TimeRange{}, "hour")
	if !errors.Is(err, errV2InvalidDrivingGroupBy) {
		t.Fatalf("expected invalid group_by, got %v", err)
	}
}
