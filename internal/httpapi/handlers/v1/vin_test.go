package v1

import "testing"

func TestDecodeTeslaVINModelYEnglish(t *testing.T) {
	got := decodeTeslaVIN("LRWYGCEKXMC000001", "en")
	if got == nil {
		t.Fatal("decodeTeslaVIN returned nil")
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

func TestDecodeTeslaVINModelYChinese(t *testing.T) {
	got := decodeTeslaVIN("LRWYGCFJ0SC000001", "zh_CN.UTF-8")
	if got == nil {
		t.Fatal("decodeTeslaVIN returned nil")
	}

	if got.WMI != "LRW" {
		t.Fatalf("WMI = %q, want LRW", got.WMI)
	}
	if got.Manufacturer != "特斯拉中国（上海超级工厂）" {
		t.Fatalf("Manufacturer = %q", got.Manufacturer)
	}
	if got.LineSeries != "Model Y" {
		t.Fatalf("LineSeries = %q", got.LineSeries)
	}
	if got.BodyType != "MPV 5 门 / 左舵" {
		t.Fatalf("BodyType = %q", got.BodyType)
	}
	if got.RestraintSystem != "2 型手动安全带，带前排安全气囊、乘员检测系统和侧面充气约束系统" {
		t.Fatalf("RestraintSystem = %q", got.RestraintSystem)
	}
	if got.FuelType != "磷酸铁锂电池" {
		t.Fatalf("FuelType = %q", got.FuelType)
	}
	if got.MotorDriveUnit != "单电机标准版" {
		t.Fatalf("MotorDriveUnit = %q", got.MotorDriveUnit)
	}
	if got.CheckDigit != "0" {
		t.Fatalf("CheckDigit = %q", got.CheckDigit)
	}
	if got.ProductionYear != 2025 {
		t.Fatalf("ProductionYear = %d", got.ProductionYear)
	}
	if got.YearType != "日历年" {
		t.Fatalf("YearType = %q", got.YearType)
	}
	if got.Plant != "特斯拉中国（上海超级工厂）" {
		t.Fatalf("Plant = %q", got.Plant)
	}
	if got.SequenceNumber != "000001" {
		t.Fatalf("SequenceNumber = %q", got.SequenceNumber)
	}
}

func TestDecodeTeslaVINModelSUsesActualVINPositions(t *testing.T) {
	got := decodeTeslaVIN("5YJSA1E26KF000000", "en")
	if got == nil {
		t.Fatal("decodeTeslaVIN returned nil")
	}

	if got.WMI != "5YJ" {
		t.Fatalf("WMI = %q, want 5YJ", got.WMI)
	}
	if got.Manufacturer != "Tesla, Inc. (United States)" {
		t.Fatalf("Manufacturer = %q", got.Manufacturer)
	}
	if got.LineSeries != "Model S" {
		t.Fatalf("LineSeries = %q, want Model S", got.LineSeries)
	}
	if got.BodyType != "" {
		t.Fatalf("BodyType = %q, want empty for non-Model Y VIN", got.BodyType)
	}
	if got.FuelType != "" {
		t.Fatalf("FuelType = %q, want empty for non-Model Y VIN", got.FuelType)
	}
	if got.MotorDriveUnit != "" {
		t.Fatalf("MotorDriveUnit = %q, want empty for non-Model Y VIN", got.MotorDriveUnit)
	}
	if got.CheckDigit != "6" {
		t.Fatalf("CheckDigit = %q, want 6", got.CheckDigit)
	}
	if got.ProductionYear != 2019 {
		t.Fatalf("ProductionYear = %d, want 2019", got.ProductionYear)
	}
	if got.YearType != "model_year" {
		t.Fatalf("YearType = %q, want model_year", got.YearType)
	}
	if got.Plant != "Tesla Fremont, CA (USA)" {
		t.Fatalf("Plant = %q, want Tesla Fremont, CA (USA)", got.Plant)
	}
	if got.SequenceNumber != "000000" {
		t.Fatalf("SequenceNumber = %q, want 000000", got.SequenceNumber)
	}
}

func TestDecodeTeslaVINSupports2026YearCode(t *testing.T) {
	got := decodeTeslaVIN("LRWYGCEKXTC000001", "en")
	if got == nil {
		t.Fatal("decodeTeslaVIN returned nil")
	}

	if got.ProductionYear != 2026 {
		t.Fatalf("ProductionYear = %d, want 2026", got.ProductionYear)
	}
	if got.YearType != "calendar_year" {
		t.Fatalf("YearType = %q, want calendar_year", got.YearType)
	}
}

func TestDecodeTeslaVINRejectsNonTeslaOrInvalidVIN(t *testing.T) {
	tests := []string{
		"",
		"LRWYGCEKXMC00001",
		"ZZZYGCEKXMC000001",
	}

	for _, tt := range tests {
		if got := decodeTeslaVIN(tt, "en"); got != nil {
			t.Fatalf("decodeTeslaVIN(%q) = %#v, want nil", tt, got)
		}
	}
}

func TestNormalizeVINLanguage(t *testing.T) {
	tests := map[string]string{
		"":            "en",
		"C.UTF-8":     "en",
		"en":          "en",
		"zh":          "zh",
		"zh-CN":       "zh",
		"zh_CN.UTF-8": "zh",
		"zh-Hans":     "zh",
	}

	for input, want := range tests {
		if got := normalizeVINLanguage(input); got != want {
			t.Fatalf("normalizeVINLanguage(%q) = %q, want %q", input, got, want)
		}
	}
}
