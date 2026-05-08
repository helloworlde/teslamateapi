package main

type V2CostAPIResponse struct {
	Data V2CostResponse `json:"data"`
	Meta V2Meta         `json:"meta"`
}

type V2CostResponse struct {
	DataScope      V2CostDataScope      `json:"data_scope"`
	Summary        V2CostSummaryDetails `json:"summary"`
	CostByPeriod   []V2CostPeriodItem   `json:"cost_by_period,omitempty"`
	CostByLocation []V2CostLocationItem `json:"cost_by_location,omitempty"`
}

type V2CostDataScope struct {
	Included []string `json:"included"`
	Excluded []string `json:"excluded"`
}

type V2CostSummaryDetails struct {
	ChargingCost  *float64 `json:"charging_cost,omitempty"`
	EnergyUsedKWh *float64 `json:"energy_used_kwh,omitempty"`
	DistanceKM    float64  `json:"distance_km"`
	CostPerKWh    *float64 `json:"cost_per_kwh,omitempty"`
	CostPerKM     *float64 `json:"cost_per_km,omitempty"`
	CostPer100KM  *float64 `json:"cost_per_100km,omitempty"`
}

type V2CostPeriodItem struct {
	PeriodStart   string   `json:"period_start"`
	ChargingCost  *float64 `json:"charging_cost,omitempty"`
	EnergyUsedKWh *float64 `json:"energy_used_kwh,omitempty"`
}

type V2CostLocationItem struct {
	LocationName  string   `json:"location_name"`
	ChargingCost  *float64 `json:"charging_cost,omitempty"`
	EnergyUsedKWh *float64 `json:"energy_used_kwh,omitempty"`
}
