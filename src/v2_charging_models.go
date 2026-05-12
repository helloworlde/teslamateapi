package main

// @name V2ChargingAPIResponse
type V2ChargingAPIResponse struct {
	Data V2ChargingResponse `json:"data"`
	Meta V2Meta             `json:"meta"`
}

// @name V2ChargingResponse
type V2ChargingResponse struct {
	Summary    V2ChargingSummary             `json:"summary"`
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

// @name V2ChargingTimeseriesItem
type V2ChargingTimeseriesItem struct {
	PeriodStart  string   `json:"period_start"`
	SessionCount int64    `json:"session_count"`
	EnergyAdded  float64  `json:"energy_added"`
	EnergyUsed   *float64 `json:"energy_used,omitempty"`
	Duration     float64  `json:"duration"`
	Cost         *float64 `json:"cost,omitempty"`
	AvgPower     *float64 `json:"avg_power,omitempty"`
}

// @name V2ChargingLocationItem
type V2ChargingLocationItem struct {
	LocationName       string   `json:"location_name"`
	GeofenceID         *int64   `json:"geofence_id,omitempty"`
	AddressID          *int64   `json:"address_id,omitempty"`
	SessionCount       int64    `json:"session_count"`
	EnergyAdded        float64  `json:"energy_added"`
	EnergyUsed         *float64 `json:"energy_used,omitempty"`
	Cost               *float64 `json:"cost,omitempty"`
	ChargingEfficiency *float64 `json:"charging_efficiency,omitempty"`
}

// @name V2ChargingTypeItem
type V2ChargingTypeItem struct {
	ChargingType string   `json:"charging_type"`
	SessionCount int64    `json:"session_count"`
	EnergyAdded  float64  `json:"energy_added"`
	EnergyUsed   *float64 `json:"energy_used,omitempty"`
	Cost         *float64 `json:"cost,omitempty"`
}
