package main

// @name V2CostAPIResponse
type V2CostAPIResponse struct {
	Data V2CostResponse `json:"data"`
	Meta V2Meta         `json:"meta"`
}

// @name V2CostResponse
type V2CostResponse struct {
	Summary        V2CostSummaryDetails `json:"summary"`
	CostByPeriod   []V2CostPeriodItem   `json:"cost_by_period,omitempty"`
	CostByLocation []V2CostLocationItem `json:"cost_by_location,omitempty"`
}

// @name V2CostSummaryDetails
type V2CostSummaryDetails struct {
	ChargingCost    *float64 `json:"charging_cost,omitempty"`
	EnergyUsed      *float64 `json:"energy_used,omitempty"`
	Distance        float64  `json:"distance"`
	CostPerEnergy   *float64 `json:"cost_per_energy,omitempty"`
	CostPerDistance *float64 `json:"cost_per_distance,omitempty"`
}

// @name V2CostPeriodItem
type V2CostPeriodItem struct {
	PeriodStart  string   `json:"period_start"`
	ChargingCost *float64 `json:"charging_cost,omitempty"`
	EnergyUsed   *float64 `json:"energy_used,omitempty"`
}

// @name V2CostLocationItem
type V2CostLocationItem struct {
	LocationName string   `json:"location_name"`
	ChargingCost *float64 `json:"charging_cost,omitempty"`
	EnergyUsed   *float64 `json:"energy_used,omitempty"`
}
