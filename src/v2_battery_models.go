package main

// @name V2BatteryAPIResponse
type V2BatteryAPIResponse struct {
	Data V2BatteryResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}

// @name V2BatteryResponse
type V2BatteryResponse struct {
	Summary V2BatteryAnalyticsSummary `json:"summary"`
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

// @name V2BatteryAnalyticsSummary
type V2BatteryAnalyticsSummary struct {
	LatestBatteryLevelPercent         *int64   `json:"latest_battery_level_percent,omitempty"`
	LatestRatedRangeKM                *float64 `json:"latest_rated_range_km,omitempty"`
	LatestIdealRangeKM                *float64 `json:"latest_ideal_range_km,omitempty"`
	EstimatedRatedRangeAt100PercentKM *float64 `json:"estimated_rated_range_at_100_percent_km,omitempty"`
	EstimatedIdealRangeAt100PercentKM *float64 `json:"estimated_ideal_range_at_100_percent_km,omitempty"`
	BaselineRatedRangeAt100PercentKM  *float64 `json:"baseline_rated_range_at_100_percent_km,omitempty"`
	BaselineIdealRangeAt100PercentKM  *float64 `json:"baseline_ideal_range_at_100_percent_km,omitempty"`
	EstimatedRangeDegradationPercent  *float64 `json:"estimated_range_degradation_percent,omitempty"`
	SampleCount                       int64    `json:"sample_count"`
}

// @name V2BatteryTimeseriesItem
type V2BatteryTimeseriesItem struct {
	PeriodStart                       string   `json:"period_start"`
	EstimatedRatedRangeAt100PercentKM *float64 `json:"estimated_rated_range_at_100_percent_km,omitempty"`
	EstimatedIdealRangeAt100PercentKM *float64 `json:"estimated_ideal_range_at_100_percent_km,omitempty"`
	AvgBatteryLevelPercent            *float64 `json:"avg_battery_level_percent,omitempty"`
	SampleCount                       int64    `json:"sample_count"`
}

