package main

import (
	"database/sql"
	"math"
	"testing"
)

func TestWhPerKmToWhPerMi(t *testing.T) {
	got := whPerKmToWhPerMi(150)
	want := 150 * 1.609344
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("whPerKmToWhPerMi: %v want %v", got, want)
	}
}

func TestKmhToMphNull(t *testing.T) {
	v := sql.NullFloat64{Valid: true, Float64: 100}
	out := kmhToMphNull(v)
	if !out.Valid || math.Abs(out.Float64-kilometersToMiles(100)) > 1e-6 {
		t.Fatalf("got %+v", out)
	}
}
