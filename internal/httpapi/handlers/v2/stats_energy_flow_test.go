package v2

import (
	"math"
	"testing"

	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

func TestBuildEnergyFlowData(t *testing.T) {
	tests := []struct {
		name                    string
		input                   energyFlowInputs
		wantStatus              string
		wantLossKWh             float64
		wantUnattributedKWh     float64
		wantUnmatchedUsageKWh   float64
		wantVehicleAvailableKWh float64
		wantEndBatteryKWh       float64
		wantEndBatteryCost      float64
		wantDrivingCost         float64
		wantDrivingCostPerKm    float64
		wantDrivingUsageRatePct float64
		wantLossRatePct         float64
	}{
		{
			name: "balanced wall input with vehicle residual",
			input: energyFlowInputs{
				carID:             1,
				carName:           NullString("Blue"),
				totalDistance:     100,
				wallEnergyKWh:     110,
				vehicleAddedKWh:   100,
				totalChargingCost: 55,
				drivingEnergyKWh:  70,
				parkingEnergyKWh:  10,
				unitsLength:       "km",
				unitsTemperature:  "C",
			},
			wantStatus:              "has_unattributed_vehicle_energy",
			wantLossKWh:             10,
			wantUnattributedKWh:     20,
			wantVehicleAvailableKWh: 100,
			wantDrivingCost:         35,
			wantDrivingCostPerKm:    0.35,
			wantDrivingUsageRatePct: 63.6363636364,
			wantLossRatePct:         9.0909090909,
		},
		{
			name: "ending battery inventory is not treated as unattributed loss",
			input: energyFlowInputs{
				totalDistance:     100,
				wallEnergyKWh:     120,
				vehicleAddedKWh:   100,
				totalChargingCost: 60,
				drivingEnergyKWh:  60,
				parkingEnergyKWh:  10,
				startBatteryKWh:   10,
				endBatteryKWh:     25,
			},
			wantStatus:              "has_unattributed_vehicle_energy",
			wantLossKWh:             20,
			wantUnattributedKWh:     15,
			wantVehicleAvailableKWh: 110,
			wantEndBatteryKWh:       25,
			wantEndBatteryCost:      11.3636363636,
			wantDrivingCost:         27.2727272727,
			wantDrivingCostPerKm:    0.2727272727,
			wantDrivingUsageRatePct: 50,
			wantLossRatePct:         16.6666666667,
		},
		{
			name: "usage can exceed vehicle-added energy when historical data is incomplete",
			input: energyFlowInputs{
				totalDistance:     50,
				wallEnergyKWh:     90,
				vehicleAddedKWh:   80,
				totalChargingCost: 45,
				drivingEnergyKWh:  70,
				parkingEnergyKWh:  20,
			},
			wantStatus:              "usage_exceeds_supply",
			wantLossKWh:             10,
			wantUnmatchedUsageKWh:   10,
			wantVehicleAvailableKWh: 90,
			wantDrivingCost:         31.1111111111,
			wantDrivingCostPerKm:    0.6222222222,
			wantDrivingUsageRatePct: 77.7777777778,
			wantLossRatePct:         11.1111111111,
		},
		{
			name: "zero wall energy avoids division by zero",
			input: energyFlowInputs{
				vehicleAddedKWh:   0,
				totalChargingCost: 20,
				drivingEnergyKWh:  5,
				parkingEnergyKWh:  2,
			},
			wantStatus:              "usage_exceeds_supply",
			wantUnmatchedUsageKWh:   7,
			wantVehicleAvailableKWh: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildEnergyFlowData(tt.input)
			assertClose(t, got.Metrics.ChargingLossEnergyKWh, tt.wantLossKWh)
			assertClose(t, got.Metrics.UnattributedVehicleEnergyKWh, tt.wantUnattributedKWh)
			assertClose(t, got.Metrics.UnmatchedVehicleUsageKWh, tt.wantUnmatchedUsageKWh)
			assertClose(t, got.Metrics.VehicleAvailableEnergyKWh, tt.wantVehicleAvailableKWh)
			assertClose(t, got.Metrics.EndBatteryEnergyKWh, tt.wantEndBatteryKWh)
			assertClose(t, got.Metrics.EndBatteryCost, tt.wantEndBatteryCost)
			assertClose(t, got.Metrics.DrivingCost, tt.wantDrivingCost)
			assertClose(t, got.Metrics.DrivingCostPerDistance, tt.wantDrivingCostPerKm)
			assertClose(t, got.Metrics.ActualDrivingUsageRatePct, tt.wantDrivingUsageRatePct)
			assertClose(t, got.Metrics.ActualLossRatePct, tt.wantLossRatePct)
			if got.BalanceStatus != tt.wantStatus {
				t.Fatalf("BalanceStatus = %q; want %q", got.BalanceStatus, tt.wantStatus)
			}
			assertClose(t, outboundEnergy(got.Links, "vehicle_added"), got.Metrics.VehicleEnergyAddedKWh)
			assertClose(t, outboundEnergy(got.Links, "vehicle_available"), got.Metrics.VehicleAvailableEnergyKWh)
			assertClose(t, inboundEnergy(got.Links, "driving_usage"), got.Metrics.DrivingEnergyKWh)
			assertClose(t, inboundEnergy(got.Links, "parking_usage"), got.Metrics.ParkingEnergyKWh)
			assertClose(t, inboundEnergy(got.Links, "end_battery_inventory"), got.Metrics.EndBatteryEnergyKWh)
			assertClose(t, outboundEnergy(got.Links, "unmatched_vehicle_usage"), got.Metrics.UnmatchedVehicleUsageKWh)
			// The whole model must reconcile to the charging bill: every terminal
			// sink cost (loss + the vehicle-side buckets) re-sums to total cost.
			// Skipped when wall energy is 0, where cost cannot be priced at all.
			if tt.input.wallEnergyKWh > 0 {
				assertClose(t, terminalCost(got.Nodes), tt.input.totalChargingCost)
			}
		})
	}
}

// terminalCost sums the cost of every leaf sink in the flow. These must add up
// to the total charging cost — no destination may be double-counted and no
// phantom (free) energy may carry cost.
func terminalCost(nodes []dto.V2EnergyFlowNode) float64 {
	var total float64
	for _, n := range nodes {
		switch n.ID {
		case "charging_loss", "driving_usage", "parking_usage",
			"end_battery_inventory", "unattributed_vehicle_energy":
			total += n.Cost
		}
	}
	return total
}

func TestMakeEnergyFlowLinkUsesSourcePercent(t *testing.T) {
	got := makeEnergyFlowLink("wall_input", "vehicle_added", 80, 40, 100)
	assertClose(t, got.PercentOfSource, 80)
	assertClose(t, got.EnergyKWh, 80)
	assertClose(t, got.Cost, 40)
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.0000001 {
		t.Fatalf("got %.10f; want %.10f", got, want)
	}
}

func outboundEnergy(links []dto.V2EnergyFlowLink, source string) float64 {
	var total float64
	for _, link := range links {
		if link.Source == source {
			total += link.EnergyKWh
		}
	}
	return total
}

func inboundEnergy(links []dto.V2EnergyFlowLink, target string) float64 {
	var total float64
	for _, link := range links {
		if link.Target == target {
			total += link.EnergyKWh
		}
	}
	return total
}
