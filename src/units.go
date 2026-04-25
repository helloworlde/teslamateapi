package main

import "database/sql"

func buildSummaryUnits(unitLength, unitTemperature string) TeslaMateSummaryUnits {
	speed := "km/h"
	cons := "Wh/km"
	costPer := "per_100_km"
	if unitLength == "mi" {
		speed = "mph"
		cons = "Wh/mi"
		costPer = "per_100_mi"
	}
	return TeslaMateSummaryUnits{
		UnitsLength:           unitLength,
		UnitsTemperature:      unitTemperature,
		UnitOfSpeed:           speed,
		UnitOfConsumption:     cons,
		UnitOfCostPerDistance: costPer,
	}
}

func whPerKmToWhPerMi(whPerKm float64) float64 {
	return whPerKm * 1.609344
}

func whPerKmToWhPerMiNull(v sql.NullFloat64) sql.NullFloat64 {
	if v.Valid {
		v.Float64 = whPerKmToWhPerMi(v.Float64)
	}
	return v
}

func kmhToMphNull(v sql.NullFloat64) sql.NullFloat64 {
	return kilometersToMilesSqlNullFloat64(v)
}
