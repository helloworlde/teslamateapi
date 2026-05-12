package main

// @name V2BatteryAPIResponse
type V2BatteryAPIResponse struct {
	Data V2BatteryResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}

// @name V2BatteryResponse
type V2BatteryResponse struct {
	Summary V2BatterySummary `json:"summary"`
}

// @name V2BatteryTimeseriesAPIResponse
type V2BatteryTimeseriesAPIResponse struct {
	Data V2BatteryTimeseriesResponse `json:"data"`
	Meta V2Meta                      `json:"meta"`
}

// @name V2BatteryTimeseriesResponse
type V2BatteryTimeseriesResponse struct {
	GroupBy string                    `json:"group_by"`
	Items   []V2BatteryTimeseriesItem `json:"items"`
}

// @name V2BatteryTimeseriesItem
type V2BatteryTimeseriesItem struct {
	PeriodStart       string          `json:"period_start"`
	RangeAtFullCharge *V2BatteryRange `json:"range_at_full_charge,omitempty"`
	AvgLevel          *float64        `json:"avg_level,omitempty"`
}
