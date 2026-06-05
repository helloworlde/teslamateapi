package convert

import (
	"math"
	"testing"
)

func TestWhPerKmToWhPerMile(t *testing.T) {
	got := WhPerKmToWhPerMile(150)
	want := 241.4016
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("WhPerKmToWhPerMile() = %f, want %f", got, want)
	}
}
