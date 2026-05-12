package main

// @name V2ChargingAPIResponse
type V2ChargingAPIResponse struct {
	Data V2ChargingResponse `json:"data"`
	Meta V2Meta             `json:"meta"`
}

// @name V2ChargingResponse
type V2ChargingResponse struct {
	Summary    V2ChargingAnalyticsSummary    `json:"summary"`
	Timeseries *V2ChargingTimeseriesResponse `json:"timeseries,omitempty"`
	Breakdown  *V2ChargingBreakdownResponse  `json:"breakdown,omitempty"`
}

// @name V2ChargingBreakdownResponse
type V2ChargingBreakdownResponse struct {
	By        string                   `json:"by"`
	Locations []V2ChargingLocationItem `json:"locations,omitempty"`
	Types     []V2ChargingTypeItem     `json:"types,omitempty"`
}

// @name V2ChargingTimeseriesResponse
type V2ChargingTimeseriesResponse struct {
	GroupBy string                     `json:"group_by"`
	Items   []V2ChargingTimeseriesItem `json:"items"`
}

// @name V2ChargingAnalyticsSummary
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

// @name V2ChargingTimeseriesItem
type V2ChargingTimeseriesItem struct {
	PeriodStart    string   `json:"period_start"`
	SessionCount   int64    `json:"session_count"`
	EnergyAddedKWh float64  `json:"energy_added_kwh"`
	EnergyUsedKWh  *float64 `json:"energy_used_kwh,omitempty"`
	DurationMin    float64  `json:"duration_min"`
	Cost           *float64 `json:"cost,omitempty"`
	AvgPowerKW     *float64 `json:"avg_power_kw,omitempty"`
}

// @name V2ChargingLocationItem
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

// @name V2ChargingTypeItem
type V2ChargingTypeItem struct {
	ChargingType   string   `json:"charging_type"`
	SessionCount   int64    `json:"session_count"`
	EnergyAddedKWh float64  `json:"energy_added_kwh"`
	EnergyUsedKWh  *float64 `json:"energy_used_kwh,omitempty"`
	Cost           *float64 `json:"cost,omitempty"`
}

