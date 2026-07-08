package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2ConsumptionGroup is one grouping bucket of the consumption analysis —
// a temperature band, a firmware version, or a season. All figures are
// objective: derived from rated-range drop * efficiency over distance, the
// same energy model used across the codebase. No estimates or thresholds.
type V2ConsumptionGroup struct {
	Key         string  `json:"key" example:"20"`
	Consumption float64 `json:"consumption" example:"135.0"`
	TripsCount  int     `json:"trips_count" example:"166"`
	Distance    float64 `json:"distance" example:"3200.5"`
	EnergyKWh   float64 `json:"energy_kwh" example:"432.0"`
	// EstimatedUsageCost = SOC-derived drive energy × average charging price
	// (SUM(cost) / SUM(charge_energy_added)). Currency matches
	// charging_processes.cost. Null when any positive-distance drive in the
	// group lacks a calibratable SOC basis.
	EstimatedUsageCost nullable.Float64 `json:"estimated_usage_cost" swaggertype:"number" example:"112.32"`
	// CostPerDistance = estimated_usage_cost / distance. Null when drive cost
	// or distance is unavailable, including partial drive SOC data.
	CostPerDistance nullable.Float64 `json:"cost_per_distance" swaggertype:"number" example:"0.035"`
	DeltaVsAvgPct   float64          `json:"delta_vs_avg_pct" example:"-22.4"`
	TempLow         nullable.Float64 `json:"temp_low" swaggertype:"number" example:"20"`
	TempHigh        nullable.Float64 `json:"temp_high" swaggertype:"number" example:"25"`
}

// V2ConsumptionData is the `data` field of V2ConsumptionResponse.
type V2ConsumptionData struct {
	Car                Car     `json:"car"`
	GroupBy            string  `json:"group_by" example:"temperature" enums:"temperature,version,season,month"`
	OverallConsumption float64 `json:"overall_consumption" example:"174.0"`
	// OverallEstimatedUsageCost is the sum of group estimated_usage_cost values.
	// Null when no group has a calculable drive cost, or when any group with
	// positive-distance drives lacks a complete SOC basis.
	OverallEstimatedUsageCost nullable.Float64 `json:"overall_estimated_usage_cost" swaggertype:"number" example:"2034.5"`
	// OverallCostPerDistance = overall_estimated_usage_cost / grouped distance.
	// Null when overall drive cost or distance is unavailable, including partial
	// drive SOC data in any positive-distance group.
	OverallCostPerDistance nullable.Float64     `json:"overall_cost_per_distance" swaggertype:"number" example:"0.048"`
	Groups                 []V2ConsumptionGroup `json:"groups"`
	Units                  TeslaMateUnits       `json:"units"`
}

// V2ConsumptionResponse is the envelope for
// /api/v2/cars/{CarID}/stats/consumption.
type V2ConsumptionResponse struct {
	Data V2ConsumptionData `json:"data"`
}
