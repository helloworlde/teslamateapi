package dto

import "github.com/tobiasehlert/teslamateapi/src/nullable"

// V2DrivesAgg is the drives section of the lifetime stats response.
type V2DrivesAgg struct {
	Count                  int     `json:"count" example:"1893"`
	TotalDistance          float64 `json:"total_distance" example:"42500.5"`
	TotalDurationMin       int     `json:"total_duration_min" example:"38000"`
	TotalEnergyConsumedKWh float64 `json:"total_energy_consumed_kwh" example:"7825.0"`
	AvgConsumption         float64 `json:"avg_consumption" example:"184.0"`
	BestConsumption        float64 `json:"best_consumption" example:"120.0"`
	LongestDistance        float64 `json:"longest_distance" example:"800.0"`
	MaxSpeed               int     `json:"max_speed" example:"180"`
}

// V2ChargesAgg is the charges section of the lifetime stats response.
type V2ChargesAgg struct {
	Count               int     `json:"count" example:"427"`
	TotalEnergyAddedKWh float64 `json:"total_energy_added_kwh" example:"16500.0"`
	TotalEnergyUsedKWh  float64 `json:"total_energy_used_kwh" example:"17200.0"`
	TotalCost           float64 `json:"total_cost" example:"4321.50"`
	FastChargeCount     int     `json:"fast_charge_count" example:"73"`
	FastChargeEnergyKWh float64 `json:"fast_charge_energy_kwh" example:"3200.0"`
	PeakPowerMaxKW      int     `json:"peak_power_max_kw" example:"250"`
}

// V2ParkingsAgg is the parkings section of the lifetime stats response.
type V2ParkingsAgg struct {
	Count              int     `json:"count" example:"1893"`
	TotalDurationMin   int     `json:"total_duration_min" example:"500000"`
	TotalEnergyDropKWh float64 `json:"total_vampire_drain_kwh" example:"125.0"`
}

// V2Lifetime is the `data` field of V2LifetimeResponse.
type V2Lifetime struct {
	Car      Car             `json:"car"`
	Since    nullable.String `json:"since" swaggertype:"string" example:"2020-01-01T00:00:00+01:00"`
	Drives   V2DrivesAgg     `json:"drives"`
	Charges  V2ChargesAgg    `json:"charges"`
	Parkings V2ParkingsAgg   `json:"parkings"`
	Units    TeslaMateUnits  `json:"units"`
}

// V2LifetimeResponse is the envelope for /api/v2/cars/{CarID}/stats/lifetime.
type V2LifetimeResponse struct {
	Data V2Lifetime `json:"data"`
}
