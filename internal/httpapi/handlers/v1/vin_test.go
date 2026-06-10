package v1

import "testing"

func TestDecodeTeslaModelYVIN(t *testing.T) {
	got := decodeTeslaModelYVIN("LRWYGCEKXMC000001")
	if got == nil {
		t.Fatal("decodeTeslaModelYVIN returned nil")
	}

	if got.WMI != "LRW" {
		t.Fatalf("WMI = %q, want LRW", got.WMI)
	}
	if got.Manufacturer != "Tesla China (Giga Shanghai)" {
		t.Fatalf("Manufacturer = %q", got.Manufacturer)
	}
	if got.LineSeries != "Model Y" {
		t.Fatalf("LineSeries = %q", got.LineSeries)
	}
	if got.BodyType != "MPV 5 DR / LHD" {
		t.Fatalf("BodyType = %q", got.BodyType)
	}
	if got.RestraintSystem != "Type 2 Manual seatbelts with front Airbags, PODS, side Inflatable restraints" {
		t.Fatalf("RestraintSystem = %q", got.RestraintSystem)
	}
	if got.FuelType != "Ternary System Li-ion battery" {
		t.Fatalf("FuelType = %q", got.FuelType)
	}
	if got.MotorDriveUnit != "Dual Motor Standard" {
		t.Fatalf("MotorDriveUnit = %q", got.MotorDriveUnit)
	}
	if got.CheckDigit != "X" {
		t.Fatalf("CheckDigit = %q", got.CheckDigit)
	}
	if got.ProductionYear != 2021 {
		t.Fatalf("ProductionYear = %d", got.ProductionYear)
	}
	if got.YearType != "calendar_year" {
		t.Fatalf("YearType = %q", got.YearType)
	}
	if got.Plant != "Tesla China (Giga Shanghai)" {
		t.Fatalf("Plant = %q", got.Plant)
	}
	if got.SequenceNumber != "000001" {
		t.Fatalf("SequenceNumber = %q", got.SequenceNumber)
	}
}

func TestDecodeTeslaModelYVINRequiresModelY(t *testing.T) {
	if got := decodeTeslaModelYVIN("5YJSA1E26KF000001"); got != nil {
		t.Fatalf("decodeTeslaModelYVIN returned %#v for non-Model Y VIN", got)
	}
}

func TestDecodeTeslaModelYVINSupports2026YearCode(t *testing.T) {
	got := decodeTeslaModelYVIN("LRWYGCEKXTC000001")
	if got == nil {
		t.Fatal("decodeTeslaModelYVIN returned nil")
	}

	if got.ProductionYear != 2026 {
		t.Fatalf("ProductionYear = %d, want 2026", got.ProductionYear)
	}
	if got.YearType != "calendar_year" {
		t.Fatalf("YearType = %q, want calendar_year", got.YearType)
	}
}

func TestDecodeTeslaModelYVINRejectsUnknownCodes(t *testing.T) {
	tests := []string{
		"",
		"LRWYGCEKXMC00001",
		"ZZZYGCEKXMC000001",
		"LRWYGCEKXMQ000001",
	}

	for _, tt := range tests {
		if got := decodeTeslaModelYVIN(tt); got != nil {
			t.Fatalf("decodeTeslaModelYVIN(%q) = %#v, want nil", tt, got)
		}
	}
}
