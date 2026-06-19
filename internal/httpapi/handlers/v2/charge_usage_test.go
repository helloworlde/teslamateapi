package v2

import (
	"database/sql"
	"testing"

	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

func TestBuildChargeUsageData(t *testing.T) {
	tests := []struct {
		name                    string
		input                   chargeUsageInputs
		wantStatus              string
		wantAvailableKWh        float64
		wantBatteryUsedKWh      float64
		wantBatteryUsedRatePct  float64
		wantUntrackedKWh        float64
		wantUnmatchedKWh        float64
		wantUntrackedSharePct   float64
		wantExpectedKWh         float64
		wantReconciliationKWh   float64
		wantTrackedRatePct      float64
		wantDrivingWhPerKm      float64
		wantAddedBatteryLevel   int64
		wantUsedBatteryLevel    int64
		wantUsedBatteryLevelOK  bool
		wantAddedBatteryLevelOK bool
		wantRangeDataComplete   bool
		wantAccountingStatus    string
		wantReconciliationOK    bool
		wantCycleMetricsOK      bool
	}{
		{
			name: "cycle uses post-charge inventory as available energy",
			input: chargeUsageInputs{
				carID:                   1,
				carName:                 NullString("Blue"),
				chargeID:                10,
				chargeStartDate:         "2026-05-26T18:25:00Z",
				chargeEndDate:           "2026-05-26T18:55:00Z",
				analysisEndDate:         "2026-05-27T06:00:00Z",
				isComplete:              true,
				startBatteryLevel:       nullInt64(55),
				postChargeBatteryLevel:  nullInt64(95),
				endBatteryLevel:         nullInt64(40),
				startBatteryKWh:         nullFloat64(20),
				postChargeBatteryKWh:    nullFloat64(48),
				endBatteryKWh:           nullFloat64(15),
				wallEnergyKWh:           32,
				chargeEnergyAddedKWh:    30,
				chargeCost:              16,
				drivingEnergyKWh:        20,
				parkingEnergyKWh:        3,
				totalDistance:           100,
				driveCount:              2,
				driveDurationMin:        75,
				hasPreChargeRangeData:   true,
				hasPostChargeRangeData:  true,
				hasEndRangeData:         true,
				drivesRangeDataComplete: true,
			},
			wantStatus:              "inventory_reconciliation_mismatch",
			wantAvailableKWh:        48,
			wantBatteryUsedKWh:      33,
			wantBatteryUsedRatePct:  68.75,
			wantUntrackedKWh:        10,
			wantUntrackedSharePct:   20.8333333333,
			wantExpectedKWh:         50,
			wantReconciliationKWh:   -2,
			wantTrackedRatePct:      47.9166666667,
			wantDrivingWhPerKm:      200,
			wantAddedBatteryLevel:   40,
			wantUsedBatteryLevel:    55,
			wantAddedBatteryLevelOK: true,
			wantUsedBatteryLevelOK:  true,
			wantRangeDataComplete:   true,
			wantAccountingStatus:    "complete",
			wantReconciliationOK:    true,
			wantCycleMetricsOK:      true,
		},
		{
			name: "tracked usage can exceed available energy",
			input: chargeUsageInputs{
				startBatteryKWh:         nullFloat64(5),
				postChargeBatteryKWh:    nullFloat64(15),
				endBatteryKWh:           nullFloat64(4),
				wallEnergyKWh:           11,
				chargeEnergyAddedKWh:    10,
				drivingEnergyKWh:        9,
				parkingEnergyKWh:        7,
				hasPreChargeRangeData:   true,
				hasPostChargeRangeData:  true,
				hasEndRangeData:         true,
				drivesRangeDataComplete: true,
			},
			wantStatus:             "usage_exceeds_available_energy",
			wantAvailableKWh:       15,
			wantBatteryUsedKWh:     11,
			wantBatteryUsedRatePct: 73.3333333333,
			wantUntrackedKWh:       5,
			wantUnmatchedKWh:       5,
			wantUntrackedSharePct:  33.3333333333,
			wantExpectedKWh:        15,
			wantTrackedRatePct:     106.6666666667,
			wantAccountingStatus:   "complete",
			wantReconciliationOK:   true,
			wantCycleMetricsOK:     true,
			wantRangeDataComplete:  true,
		},
		{
			name: "pre-charge range is context only for cycle completeness",
			input: chargeUsageInputs{
				postChargeBatteryKWh:    nullFloat64(40),
				endBatteryKWh:           nullFloat64(25),
				drivingEnergyKWh:        10,
				parkingEnergyKWh:        2,
				hasPostChargeRangeData:  true,
				hasEndRangeData:         true,
				drivesRangeDataComplete: true,
			},
			wantStatus:             "has_untracked_energy",
			wantAvailableKWh:       40,
			wantBatteryUsedKWh:     15,
			wantBatteryUsedRatePct: 37.5,
			wantUntrackedKWh:       3,
			wantUntrackedSharePct:  7.5,
			wantTrackedRatePct:     30,
			wantRangeDataComplete:  true,
			wantAccountingStatus:   "complete",
			wantCycleMetricsOK:     true,
		},
		{
			name: "nullable battery levels stay null",
			input: chargeUsageInputs{
				chargeEnergyAddedKWh: 0,
			},
			wantStatus:           "partial",
			wantAccountingStatus: "partial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildChargeUsageData(tt.input)
			assertNullableClose(t, got.Metrics.VehicleAvailableEnergyKWh, tt.wantAvailableKWh, tt.input.hasPostChargeRangeData)
			assertNullableClose(t, got.Metrics.BatteryUsedEnergyKWh, tt.wantBatteryUsedKWh, tt.input.hasPostChargeRangeData && tt.input.hasEndRangeData)
			assertNullableClose(t, got.Metrics.BatteryUsedRatePct, tt.wantBatteryUsedRatePct, tt.input.hasPostChargeRangeData && tt.input.hasEndRangeData)
			assertNullableClose(t, got.Metrics.UntrackedEnergyKWh, tt.wantUntrackedKWh, tt.wantCycleMetricsOK)
			assertNullableClose(t, got.Metrics.UnmatchedUsageKWh, tt.wantUnmatchedKWh, tt.wantCycleMetricsOK)
			assertNullableClose(t, got.Metrics.UntrackedShareOfAvailablePct, tt.wantUntrackedSharePct, tt.wantCycleMetricsOK)
			if got.Metrics.InventoryExpectedEnergyKWh.Valid != tt.wantReconciliationOK {
				t.Fatalf("InventoryExpectedEnergyKWh.Valid = %v; want %v", got.Metrics.InventoryExpectedEnergyKWh.Valid, tt.wantReconciliationOK)
			}
			if got.Metrics.InventoryReconciliationKWh.Valid != tt.wantReconciliationOK {
				t.Fatalf("InventoryReconciliationKWh.Valid = %v; want %v", got.Metrics.InventoryReconciliationKWh.Valid, tt.wantReconciliationOK)
			}
			if got.Metrics.InventoryExpectedEnergyKWh.Valid {
				assertClose(t, got.Metrics.InventoryExpectedEnergyKWh.Float64, tt.wantExpectedKWh)
				assertClose(t, got.Metrics.InventoryReconciliationKWh.Float64, tt.wantReconciliationKWh)
			}
			assertNullableClose(t, got.Metrics.TrackedUsageRatePct, tt.wantTrackedRatePct, tt.wantCycleMetricsOK)
			assertNullableClose(t, got.Metrics.DrivingEnergyPerDistanceWh, tt.wantDrivingWhPerKm, tt.input.drivesRangeDataComplete)
			if got.BalanceStatus != tt.wantStatus {
				t.Fatalf("BalanceStatus = %q; want %q", got.BalanceStatus, tt.wantStatus)
			}
			if got.Battery.AddedBatteryLevel.Valid != tt.wantAddedBatteryLevelOK {
				t.Fatalf("AddedBatteryLevel.Valid = %v; want %v", got.Battery.AddedBatteryLevel.Valid, tt.wantAddedBatteryLevelOK)
			}
			if got.Battery.AddedBatteryLevel.Valid && got.Battery.AddedBatteryLevel.Int64 != tt.wantAddedBatteryLevel {
				t.Fatalf("AddedBatteryLevel = %d; want %d", got.Battery.AddedBatteryLevel.Int64, tt.wantAddedBatteryLevel)
			}
			if got.Battery.UsedBatteryLevel.Valid != tt.wantUsedBatteryLevelOK {
				t.Fatalf("UsedBatteryLevel.Valid = %v; want %v", got.Battery.UsedBatteryLevel.Valid, tt.wantUsedBatteryLevelOK)
			}
			if got.Battery.UsedBatteryLevel.Valid && got.Battery.UsedBatteryLevel.Int64 != tt.wantUsedBatteryLevel {
				t.Fatalf("UsedBatteryLevel = %d; want %d", got.Battery.UsedBatteryLevel.Int64, tt.wantUsedBatteryLevel)
			}
			if got.Metrics.VehicleAvailableEnergyKWh.Valid {
				assertClose(t, inboundChargeUsageEnergy(got.Links, "cycle_available"), got.Metrics.VehicleAvailableEnergyKWh.Float64)
				assertClose(t, outboundChargeUsageEnergy(got.Links, "cycle_available"), got.Metrics.VehicleAvailableEnergyKWh.Float64)
			}
			if got.Metrics.UnmatchedUsageKWh.Valid {
				assertClose(t, outboundChargeUsageEnergy(got.Links, "unmatched_usage"), got.Metrics.UnmatchedUsageKWh.Float64)
			}
			if got.DataQuality.RangeDataComplete != tt.wantRangeDataComplete {
				t.Fatalf("RangeDataComplete = %v; want %v", got.DataQuality.RangeDataComplete, tt.wantRangeDataComplete)
			}
			if got.DataQuality.AccountingStatus != tt.wantAccountingStatus {
				t.Fatalf("AccountingStatus = %q; want %q", got.DataQuality.AccountingStatus, tt.wantAccountingStatus)
			}
		})
	}
}

func TestBuildChargeUsageDataOmitsUnknownCycleFlow(t *testing.T) {
	got := buildChargeUsageData(chargeUsageInputs{
		chargeEnergyAddedKWh: 10,
		drivingEnergyKWh:     5,
		parkingEnergyKWh:     2,
	})

	if got.BalanceStatus != "partial" {
		t.Fatalf("BalanceStatus = %q; want partial", got.BalanceStatus)
	}
	if got.Metrics.VehicleAvailableEnergyKWh.Valid {
		t.Fatal("VehicleAvailableEnergyKWh should be null without post-charge range data")
	}
	if got.Metrics.UntrackedEnergyKWh.Valid {
		t.Fatal("UntrackedEnergyKWh should be null without complete cycle range data")
	}
	assertClose(t, inboundChargeUsageEnergy(got.Links, "cycle_available"), 0)
	assertClose(t, outboundChargeUsageEnergy(got.Links, "cycle_available"), 0)
}

func TestBuildChargeUsageDataFlagsInventoryReconciliationMismatch(t *testing.T) {
	got := buildChargeUsageData(chargeUsageInputs{
		startBatteryLevel:       nullInt64(9),
		postChargeBatteryLevel:  nullInt64(100),
		endBatteryLevel:         nullInt64(28),
		startBatteryKWh:         nullFloat64(6.192),
		postChargeBatteryKWh:    nullFloat64(63.916),
		endBatteryKWh:           nullFloat64(17.791),
		wallEnergyKWh:           58.05,
		chargeEnergyAddedKWh:    54.64,
		drivingEnergyKWh:        35.881,
		parkingEnergyKWh:        10.244,
		hasPreChargeRangeData:   true,
		hasPostChargeRangeData:  true,
		hasEndRangeData:         true,
		drivesRangeDataComplete: true,
	})

	assertNullableClose(t, got.Battery.StartBatteryEnergyKWh, 6.192, true)
	assertNullableClose(t, got.Battery.PostChargeEnergyKWh, 63.916, true)
	assertNullableClose(t, got.Battery.EndBatteryEnergyKWh, 17.791, true)
	assertNullableClose(t, got.Metrics.VehicleAvailableEnergyKWh, 63.916, true)
	assertNullableClose(t, got.Metrics.InventoryExpectedEnergyKWh, 60.832, true)
	assertNullableClose(t, got.Metrics.InventoryReconciliationKWh, 3.084, true)
	assertNullableClose(t, got.Metrics.BatteryUsedEnergyKWh, 46.125, true)
	assertNullableClose(t, got.Metrics.UnmatchedUsageKWh, 0, true)
	if got.BalanceStatus != "inventory_reconciliation_mismatch" {
		t.Fatalf("BalanceStatus = %q; want inventory_reconciliation_mismatch", got.BalanceStatus)
	}
}

func TestChargeUsagePartialFieldsAreUnique(t *testing.T) {
	fields := chargeUsagePartialFields(chargeUsageInputs{})
	seen := map[string]bool{}
	for _, field := range fields {
		if seen[field] {
			t.Fatalf("duplicate partial field %q in %#v", field, fields)
		}
		seen[field] = true
	}
}

func nullInt64(v int64) NullInt64 {
	return NullInt64{NullInt64: sql.NullInt64{Int64: v, Valid: true}}
}

func nullFloat64(v float64) NullFloat64 {
	return NullFloat64{NullFloat64: sql.NullFloat64{Float64: v, Valid: true}}
}

func assertNullableClose(t *testing.T, got NullFloat64, want float64, wantValid bool) {
	t.Helper()
	if got.Valid != wantValid {
		t.Fatalf("nullable valid = %v; want %v", got.Valid, wantValid)
	}
	if got.Valid {
		assertClose(t, got.Float64, want)
	}
}

func inboundChargeUsageEnergy(links []dto.V2ChargeUsageLink, target string) float64 {
	var total float64
	for _, link := range links {
		if link.Target == target {
			total += link.EnergyKWh
		}
	}
	return total
}

func outboundChargeUsageEnergy(links []dto.V2ChargeUsageLink, source string) float64 {
	var total float64
	for _, link := range links {
		if link.Source == source {
			total += link.EnergyKWh
		}
	}
	return total
}
