package v1

import (
	"strings"

	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

var modelYVINDescriptions = struct {
	wmi             map[string]string
	bodyType        map[byte]string
	restraintSystem map[byte]string
	motorDriveUnit  map[byte]string
	productionYear  map[byte]int
	plant           map[byte]string
}{
	wmi: map[string]string{
		"5YJ": "Tesla Fremont, CA (USA) through Model Year M",
		"7SA": "Tesla Fremont, CA (USA) / Tesla Austin, TX (USA) from Model Year N onwards",
		"LRW": "Tesla China (Giga Shanghai)",
		"XP7": "Tesla Berlin (Germany)",
	},
	bodyType: map[byte]string{
		'G': "MPV 5 DR / LHD",
		'H': "MPV 5 DR / RHD",
	},
	restraintSystem: map[byte]string{
		'A': "Type 2 Manual seatbelts (FR, SR*3, TR*2) with front Airbags, PODS, side Inflatable restraints, Knee Airbags",
		'D': "Type 2 Manual seatbelts (FR, SR*3) with front Airbags, PODS, side Inflatable restraints, Knee Airbags",
		'C': "Type 2 Manual seatbelts with front Airbags, PODS, side Inflatable restraints",
	},
	motorDriveUnit: map[byte]string{
		'D': "Single Motor Standard",
		'E': "Dual Motor Standard",
		'F': "Dual Motor Performance (3DU 800A)",
		'J': "Single Motor Standard",
		'K': "Dual Motor Standard",
		'L': "Dual Motor Performance",
		'R': "Single Motor Standard (3DU 600A)",
		'S': "Single Motor Standard (DUB 600A)",
	},
	productionYear: map[byte]int{
		'H': 2017,
		'J': 2018,
		'K': 2019,
		'L': 2020,
		'M': 2021,
		'N': 2022,
		'P': 2023,
		'R': 2024,
		'S': 2025,
		'T': 2026,
	},
	plant: map[byte]string{
		'A': "Tesla Austin, TX (USA)",
		'B': "Tesla Berlin (Germany)",
		'C': "Tesla China (Giga Shanghai)",
		'F': "Tesla Fremont, CA (USA)",
	},
}

func decodeTeslaModelYVIN(raw string) *dto.V1CarVINDetails {
	vin := strings.ToUpper(strings.TrimSpace(raw))
	if len(vin) != 17 || vin[3] != 'Y' {
		return nil
	}

	wmi := vin[:3]
	manufacturer, ok := modelYVINDescriptions.wmi[wmi]
	if !ok {
		return nil
	}

	year, yearOK := modelYVINDescriptions.productionYear[vin[9]]
	bodyType, bodyTypeOK := modelYVINDescriptions.bodyType[vin[4]]
	restraintSystem, restraintOK := modelYVINDescriptions.restraintSystem[vin[5]]
	fuelType, fuelTypeOK := modelYVINFuelType(wmi, vin[6])
	motorDriveUnit, motorOK := modelYVINDescriptions.motorDriveUnit[vin[7]]
	plant, plantOK := modelYVINDescriptions.plant[vin[10]]
	if !yearOK || !bodyTypeOK || !restraintOK || !fuelTypeOK || !motorOK || !plantOK {
		return nil
	}

	return &dto.V1CarVINDetails{
		WMI:             wmi,
		Manufacturer:    manufacturer,
		LineSeries:      "Model Y",
		BodyType:        bodyType,
		RestraintSystem: restraintSystem,
		FuelType:        fuelType,
		MotorDriveUnit:  motorDriveUnit,
		CheckDigit:      vin[8:9],
		ProductionYear:  year,
		YearType:        modelYVINYearType(wmi),
		Plant:           plant,
		SequenceNumber:  vin[11:17],
	}
}

func modelYVINYearType(wmi string) string {
	switch wmi {
	case "LRW", "XP7":
		return "calendar_year"
	default:
		return "model_year"
	}
}

func modelYVINFuelType(wmi string, code byte) (string, bool) {
	switch code {
	case 'E':
		if wmi == "LRW" {
			return "Ternary System Li-ion battery", true
		}
		return "Electric", true
	case 'F':
		return "Lithium Iron Phosphate Battery", wmi == "LRW"
	default:
		return "", false
	}
}
