package v1

import (
	"math"
	"strings"
	"testing"
)

func TestV1EstimatedUsageCostUsesRatedRangeEnergyNotSOCDrop(t *testing.T) {
	t.Parallel()

	pricePerKWh := 0.647662
	drives := []struct {
		name       string
		startSOC   int
		endSOC     int
		startRange float64
		endRange   float64
		wantEnergy float64
	}{
		{name: "drive 1189", startSOC: 65, endSOC: 64, startRange: 156.23, endRange: 152, wantEnergy: 0.6345},
		{name: "drive 1190", startSOC: 63, endSOC: 62, startRange: 180.61, endRange: 173, wantEnergy: 1.1415},
	}

	var costs []float64
	for _, drive := range drives {
		if got := drive.startSOC - drive.endSOC; got != 1 {
			t.Fatalf("%s SOC drop = %d; want 1", drive.name, got)
		}
		cost := testV1EstimatedUsageCostFromRange(
			ptrFloat64(drive.startRange),
			ptrFloat64(drive.endRange),
			ptrFloat64(0.15),
			ptrFloat64(pricePerKWh),
		)
		if cost == nil {
			t.Fatalf("%s cost is nil; want calculable", drive.name)
		}
		want := drive.wantEnergy * pricePerKWh
		if !almostEqual(*cost, want) {
			t.Fatalf("%s cost = %f; want %f", drive.name, *cost, want)
		}
		costs = append(costs, *cost)
	}

	if almostEqual(costs[0], costs[1]) {
		t.Fatalf("same 1%% SOC drop must not force equal costs: got %f and %f", costs[0], costs[1])
	}
}

func TestV1EstimatedUsageCostNullSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		startRange  *float64
		endRange    *float64
		efficiency  *float64
		pricePerKWh *float64
		wantNil     bool
		want        float64
	}{
		{
			name:        "energy consumed net unavailable",
			startRange:  ptrFloat64(100),
			endRange:    ptrFloat64(100),
			efficiency:  ptrFloat64(0.15),
			pricePerKWh: ptrFloat64(0.65),
			wantNil:     true,
		},
		{
			name:        "charge energy unavailable",
			startRange:  ptrFloat64(100),
			endRange:    ptrFloat64(95),
			efficiency:  ptrFloat64(0.15),
			pricePerKWh: nil,
			wantNil:     true,
		},
		{
			name:        "charge energy with zero cost",
			startRange:  ptrFloat64(100),
			endRange:    ptrFloat64(95),
			efficiency:  ptrFloat64(0.15),
			pricePerKWh: ptrFloat64(0),
			want:        0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := testV1EstimatedUsageCostFromRange(tt.startRange, tt.endRange, tt.efficiency, tt.pricePerKWh)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("cost = %f; want nil", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("cost = nil; want %f", tt.want)
			}
			if !almostEqual(*got, tt.want) {
				t.Fatalf("cost = %f; want %f", *got, tt.want)
			}
		})
	}
}

func TestV1EstimatedUsageCostSQLDoesNotUseSOCDrop(t *testing.T) {
	t.Parallel()

	for _, forbidden := range []string{"battery_level", "kwh_per_pct", "cap."} {
		if strings.Contains(v1EstimatedUsageCostSQL, forbidden) {
			t.Fatalf("estimated usage cost SQL must not contain %q: %s", forbidden, v1EstimatedUsageCostSQL)
		}
	}
	for _, required := range []string{
		"(start_rated_range_km - end_rated_range_km) * cars.efficiency * charge_price.cost_per_kwh",
		"charge_price.cost_per_kwh IS NOT NULL",
	} {
		if !strings.Contains(v1EstimatedUsageCostSQL, required) {
			t.Fatalf("estimated usage cost SQL missing %q: %s", required, v1EstimatedUsageCostSQL)
		}
	}
}

func testV1EstimatedUsageCostFromRange(startRange, endRange, efficiency, pricePerKWh *float64) *float64 {
	energy := testV1EnergyConsumedNet(startRange, endRange, efficiency)
	if energy == nil || pricePerKWh == nil {
		return nil
	}
	cost := *energy * *pricePerKWh
	return &cost
}

func testV1EnergyConsumedNet(startRange, endRange, efficiency *float64) *float64 {
	if startRange == nil || endRange == nil || efficiency == nil || *startRange <= *endRange {
		return nil
	}
	energy := (*startRange - *endRange) * *efficiency
	return &energy
}

func ptrFloat64(v float64) *float64 {
	return &v
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
