package main

type V2BatteryAPIResponse struct {
	Data V2BatteryResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}

type V2BatteryResponse struct {
	Summary    V2BatteryAnalyticsSummary    `json:"summary"`
	Comparison map[string]V2ComparisonValue `json:"comparison,omitempty"`
}

type V2BatteryTimeseriesAPIResponse struct {
	Data V2BatteryTimeseriesResponse `json:"data"`
	Meta V2Meta                      `json:"meta"`
}

type V2BatteryTimeseriesResponse struct {
	GroupBy string                    `json:"group_by"`
	Items   []V2BatteryTimeseriesItem `json:"items"`
}

type V2BatteryDistributionAPIResponse struct {
	Data V2BatteryDistributionResponse `json:"data"`
	Meta V2Meta                        `json:"meta"`
}

type V2BatteryDistributionResponse struct {
	Items []V2BatteryDistributionItem `json:"items"`
}

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

type V2BatteryTimeseriesItem struct {
	PeriodStart                       string   `json:"period_start"`
	EstimatedRatedRangeAt100PercentKM *float64 `json:"estimated_rated_range_at_100_percent_km,omitempty"`
	EstimatedIdealRangeAt100PercentKM *float64 `json:"estimated_ideal_range_at_100_percent_km,omitempty"`
	AvgBatteryLevelPercent            *float64 `json:"avg_battery_level_percent,omitempty"`
	SampleCount                       int64    `json:"sample_count"`
}

type V2BatteryDistributionItem struct {
	Bucket                 string  `json:"bucket"`
	MinBatteryLevelPercent int64   `json:"min_battery_level_percent"`
	MaxBatteryLevelPercent int64   `json:"max_battery_level_percent"`
	SampleCount            int64   `json:"sample_count"`
	Percent                float64 `json:"percent"`
}
