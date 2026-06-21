package v2

import "testing"

func TestTemperatureConsumptionBucketExprUsesFiveDegreeBuckets(t *testing.T) {
	if temperatureConsumptionBucketCelsius != 5 {
		t.Fatalf("temperatureConsumptionBucketCelsius = %d; want 5", temperatureConsumptionBucketCelsius)
	}

	got := temperatureConsumptionBucketExpr("d.outside_temp_avg")
	want := "(floor(d.outside_temp_avg / 5) * 5)"
	if got != want {
		t.Fatalf("temperatureConsumptionBucketExpr() = %q; want %q", got, want)
	}
}
