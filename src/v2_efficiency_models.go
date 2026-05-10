package main

// @name V2EfficiencyAPIResponse
type V2EfficiencyAPIResponse struct {
	Data V2EfficiencyResponse `json:"data"`
	Meta V2Meta               `json:"meta"`
}

// @name V2EfficiencyResponse
type V2EfficiencyResponse struct {
	Summary V2EfficiencySummary `json:"summary"`
}

// @name V2EfficiencyFactorsAPIResponse
type V2EfficiencyFactorsAPIResponse struct {
	Data V2EfficiencyFactorsResponse `json:"data"`
	Meta V2Meta                      `json:"meta"`
}

// @name V2EfficiencyFactorsResponse
type V2EfficiencyFactorsResponse struct {
	Dimension string                   `json:"dimension"`
	Items     []V2EfficiencyFactorItem `json:"items"`
}

// @name V2EfficiencySummary
type V2EfficiencySummary struct {
	DriveCount                    int64    `json:"drive_count"`
	DistanceKM                    float64  `json:"distance_km"`
	EstimatedEnergyConsumedKWh    *float64 `json:"estimated_energy_consumed_kwh,omitempty"`
	AvgConsumptionWhPerKM         *float64 `json:"avg_consumption_wh_per_km,omitempty"`
	BestConsumptionWhPerKM        *float64 `json:"best_consumption_wh_per_km,omitempty"`
	WorstConsumptionWhPerKM       *float64 `json:"worst_consumption_wh_per_km,omitempty"`
	AvgTemperatureC               *float64 `json:"avg_temperature_c,omitempty"`
	AvgSpeedKMH                   *float64 `json:"avg_speed_kmh,omitempty"`
	EstimatedRegeneratedEnergyKWh *float64 `json:"estimated_regenerated_energy_kwh,omitempty"`
}

// @name V2EfficiencyFactorItem
type V2EfficiencyFactorItem struct {
	Bucket                     string   `json:"bucket"`
	DriveCount                 int64    `json:"drive_count"`
	DistanceKM                 float64  `json:"distance_km"`
	EstimatedEnergyConsumedKWh *float64 `json:"estimated_energy_consumed_kwh,omitempty"`
	AvgConsumptionWhPerKM      *float64 `json:"avg_consumption_wh_per_km,omitempty"`
	AvgTemperatureC            *float64 `json:"avg_temperature_c,omitempty"`
	AvgSpeedKMH                *float64 `json:"avg_speed_kmh,omitempty"`
}
