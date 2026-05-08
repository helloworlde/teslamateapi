package main

type V2ChargingAPIResponse struct {
	Data V2ChargingResponse `json:"data"`
	Meta V2Meta             `json:"meta"`
}

type V2ChargingResponse struct {
	Summary    V2ChargingAnalyticsSummary   `json:"summary"`
	Comparison map[string]V2ComparisonValue `json:"comparison,omitempty"`
}

type V2ChargingTimeseriesAPIResponse struct {
	Data V2ChargingTimeseriesResponse `json:"data"`
	Meta V2Meta                       `json:"meta"`
}

type V2ChargingTimeseriesResponse struct {
	GroupBy string                     `json:"group_by"`
	Items   []V2ChargingTimeseriesItem `json:"items"`
}

type V2ChargingLocationsAPIResponse struct {
	Data V2ChargingLocationsResponse `json:"data"`
	Meta V2Meta                      `json:"meta"`
}

type V2ChargingLocationsResponse struct {
	Items []V2ChargingLocationItem `json:"items"`
}

type V2ChargingTypesAPIResponse struct {
	Data V2ChargingTypesResponse `json:"data"`
	Meta V2Meta                  `json:"meta"`
}

type V2ChargingTypesResponse struct {
	Items []V2ChargingTypeItem `json:"items"`
}

type V2ChargingCostAPIResponse struct {
	Data V2ChargingCostResponse `json:"data"`
	Meta V2Meta                 `json:"meta"`
}

type V2ChargingCostResponse struct {
	Summary        V2ChargingCostSummary        `json:"summary"`
	CostByPeriod   []V2ChargingCostPeriodItem   `json:"cost_by_period,omitempty"`
	CostByLocation []V2ChargingCostLocationItem `json:"cost_by_location,omitempty"`
}

type V2ChargingAnalyticsSummary struct {
	SessionCount              int64    `json:"session_count"`
	EnergyAddedKWh            float64  `json:"energy_added_kwh"`
	EnergyUsedKWh             *float64 `json:"energy_used_kwh,omitempty"`
	ChargingEfficiencyPercent *float64 `json:"charging_efficiency_percent,omitempty"`
	DurationMin               float64  `json:"duration_min"`
	AvgDurationMin            *float64 `json:"avg_duration_min,omitempty"`
	AvgPowerKW                *float64 `json:"avg_power_kw,omitempty"`
	MaxPowerKW                *float64 `json:"max_power_kw,omitempty"`
	Cost                      *float64 `json:"cost,omitempty"`
	AvgCostPerKWh             *float64 `json:"avg_cost_per_kwh,omitempty"`
	StartBatteryAvgPercent    *float64 `json:"start_battery_avg_percent,omitempty"`
	EndBatteryAvgPercent      *float64 `json:"end_battery_avg_percent,omitempty"`
	ACSessionCount            int64    `json:"ac_session_count"`
	DCSessionCount            int64    `json:"dc_session_count"`
	ACEnergyKWh               float64  `json:"ac_energy_kwh"`
	DCEnergyKWh               float64  `json:"dc_energy_kwh"`
}

type V2ChargingTimeseriesItem struct {
	PeriodStart    string   `json:"period_start"`
	SessionCount   int64    `json:"session_count"`
	EnergyAddedKWh float64  `json:"energy_added_kwh"`
	EnergyUsedKWh  *float64 `json:"energy_used_kwh,omitempty"`
	DurationMin    float64  `json:"duration_min"`
	Cost           *float64 `json:"cost,omitempty"`
	AvgPowerKW     *float64 `json:"avg_power_kw,omitempty"`
}

type V2ChargingLocationItem struct {
	LocationName              string   `json:"location_name"`
	GeofenceID                *int64   `json:"geofence_id,omitempty"`
	AddressID                 *int64   `json:"address_id,omitempty"`
	SessionCount              int64    `json:"session_count"`
	EnergyAddedKWh            float64  `json:"energy_added_kwh"`
	EnergyUsedKWh             *float64 `json:"energy_used_kwh,omitempty"`
	Cost                      *float64 `json:"cost,omitempty"`
	ChargingEfficiencyPercent *float64 `json:"charging_efficiency_percent,omitempty"`
}

type V2ChargingTypeItem struct {
	ChargingType   string   `json:"charging_type"`
	SessionCount   int64    `json:"session_count"`
	EnergyAddedKWh float64  `json:"energy_added_kwh"`
	EnergyUsedKWh  *float64 `json:"energy_used_kwh,omitempty"`
	Cost           *float64 `json:"cost,omitempty"`
}

type V2ChargingCostSummary struct {
	ChargingCost  *float64 `json:"charging_cost,omitempty"`
	EnergyUsedKWh *float64 `json:"energy_used_kwh,omitempty"`
	DistanceKM    float64  `json:"distance_km"`
	CostPerKWh    *float64 `json:"cost_per_kwh,omitempty"`
	CostPerKM     *float64 `json:"cost_per_km,omitempty"`
	CostPer100KM  *float64 `json:"cost_per_100km,omitempty"`
}

type V2ChargingCostPeriodItem struct {
	PeriodStart   string   `json:"period_start"`
	ChargingCost  *float64 `json:"charging_cost,omitempty"`
	EnergyUsedKWh *float64 `json:"energy_used_kwh,omitempty"`
}

type V2ChargingCostLocationItem struct {
	LocationName  string   `json:"location_name"`
	ChargingCost  *float64 `json:"charging_cost,omitempty"`
	EnergyUsedKWh *float64 `json:"energy_used_kwh,omitempty"`
}
