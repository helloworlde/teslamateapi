package v2

import "testing"

func TestBuildDrivePowerData(t *testing.T) {
	got := buildDrivePowerData(drivePowerInputs{
		carID:                  1,
		carName:                NullString("Blue"),
		driveID:                42,
		startDate:              "2026-05-26T08:00:00+08:00",
		endDate:                "2026-05-26T08:30:00+08:00",
		distance:               18,
		durationMin:            30,
		durationSeconds:        1800,
		batteryOutputEnergyKWh: 3.6,
		regenRecoveredKWh:      0.9,
		outputDurationSeconds:  360,
		regenDurationSeconds:   180,
		peakOutputPowerKW:      180,
		peakRegenPowerKW:       60,
		powerSampleCount:       1780,
		validIntervalCount:     1740,
		ignoredGapCount:        3,
		validSampleSeconds:     1728,
		unitsLength:            "km",
		unitsTemperature:       "C",
	})

	assertClose(t, got.Metrics.ObservedBatteryOutputEnergyKWh, 3.6)
	assertClose(t, got.Metrics.ObservedRegenRecoveredKWh, 0.9)
	assertClose(t, got.Metrics.ObservedNetBatteryEnergyKWh, 2.7)
	assertNullableClose(t, got.Metrics.RegenShareOfOutputPct, 25, true)
	assertNullableClose(t, got.Metrics.RegenShareOfPowerActivityPct, 20, true)
	assertNullableClose(t, got.Metrics.AvgOutputPowerKW, 36, true)
	assertNullableClose(t, got.Metrics.AvgRegenPowerKW, 18, true)
	assertClose(t, got.DataQuality.SampleCoveragePct, 96)
	if got.DataQuality.Confidence != "high" {
		t.Fatalf("Confidence = %q; want high", got.DataQuality.Confidence)
	}
	if !got.DataQuality.HasPowerSamples {
		t.Fatal("HasPowerSamples = false; want true")
	}
}

func TestBuildDrivePowerDataUnavailable(t *testing.T) {
	got := buildDrivePowerData(drivePowerInputs{
		durationSeconds:    1800,
		powerSampleCount:   1,
		validIntervalCount: 0,
	})

	if got.DataQuality.Confidence != "unavailable" {
		t.Fatalf("Confidence = %q; want unavailable", got.DataQuality.Confidence)
	}
	if got.Metrics.RegenShareOfOutputPct.Valid {
		t.Fatal("RegenShareOfOutputPct should be null without output energy")
	}
	if got.Metrics.AvgOutputPowerKW.Valid {
		t.Fatal("AvgOutputPowerKW should be null without output seconds")
	}
}

func TestDrivePowerConfidence(t *testing.T) {
	tests := []struct {
		coverage float64
		has      bool
		want     string
	}{
		{coverage: 99, has: true, want: "high"},
		{coverage: 95, has: true, want: "high"},
		{coverage: 90, has: true, want: "medium"},
		{coverage: 80, has: true, want: "medium"},
		{coverage: 79.9, has: true, want: "low"},
		{coverage: 100, has: false, want: "unavailable"},
	}
	for _, tt := range tests {
		if got := drivePowerConfidence(tt.coverage, tt.has); got != tt.want {
			t.Fatalf("drivePowerConfidence(%v, %v) = %q; want %q", tt.coverage, tt.has, got, tt.want)
		}
	}
}
