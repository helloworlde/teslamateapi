package main

type V2DrivingAPIResponse struct {
	Data V2DrivingResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}

type V2DrivingResponse struct {
	Summary    V2DrivingAnalyticsSummary    `json:"summary"`
	Comparison map[string]V2ComparisonValue `json:"comparison,omitempty"`
}

type V2DrivingTimeseriesAPIResponse struct {
	Data V2DrivingTimeseriesResponse `json:"data"`
	Meta V2Meta                      `json:"meta"`
}

type V2DrivingTimeseriesResponse struct {
	GroupBy string                    `json:"group_by"`
	Items   []V2DrivingTimeseriesItem `json:"items"`
}

type V2DrivingDistributionAPIResponse struct {
	Data V2DrivingDistributionResponse `json:"data"`
	Meta V2Meta                        `json:"meta"`
}

type V2DrivingDistributionResponse struct {
	Dimension string                      `json:"dimension"`
	Items     []V2DrivingDistributionItem `json:"items"`
}

type V2DrivingRankingAPIResponse struct {
	Data V2DrivingRankingResponse `json:"data"`
	Meta V2Meta                   `json:"meta"`
}

type V2DrivingRankingResponse struct {
	Type  string                 `json:"type"`
	Items []V2DrivingRankingItem `json:"items"`
}

type V2DrivingAnalyticsSummary struct {
	DriveCount                    int64    `json:"drive_count"`
	DistanceKM                    float64  `json:"distance_km"`
	DurationMin                   float64  `json:"duration_min"`
	AvgDistanceKM                 *float64 `json:"avg_distance_km,omitempty"`
	AvgDurationMin                *float64 `json:"avg_duration_min,omitempty"`
	MaxSpeedKMH                   float64  `json:"max_speed_kmh"`
	AvgSpeedKMH                   *float64 `json:"avg_speed_kmh,omitempty"`
	EstimatedEnergyConsumedKWh    *float64 `json:"estimated_energy_consumed_kwh,omitempty"`
	AvgConsumptionWhPerKM         *float64 `json:"avg_consumption_wh_per_km,omitempty"`
	RangeLossKM                   float64  `json:"range_loss_km"`
	BatteryLevelUsedPercent       float64  `json:"battery_level_used_percent"`
	EstimatedRegeneratedEnergyKWh *float64 `json:"estimated_regenerated_energy_kwh,omitempty"`
	AvgOutsideTempC               *float64 `json:"avg_outside_temp_c,omitempty"`
	TotalAscentM                  float64  `json:"total_ascent_m"`
	TotalDescentM                 float64  `json:"total_descent_m"`
}

type V2DrivingTimeseriesItem struct {
	PeriodStart                string   `json:"period_start"`
	DriveCount                 int64    `json:"drive_count"`
	DistanceKM                 float64  `json:"distance_km"`
	DurationMin                float64  `json:"duration_min"`
	AvgSpeedKMH                *float64 `json:"avg_speed_kmh,omitempty"`
	EstimatedEnergyConsumedKWh *float64 `json:"estimated_energy_consumed_kwh,omitempty"`
	AvgConsumptionWhPerKM      *float64 `json:"avg_consumption_wh_per_km,omitempty"`
}

type V2DrivingDistributionItem struct {
	Bucket      string  `json:"bucket"`
	DriveCount  int64   `json:"drive_count"`
	DistanceKM  float64 `json:"distance_km"`
	DurationMin float64 `json:"duration_min"`
}

type V2DrivingRankingItem struct {
	Rank        int      `json:"rank"`
	DriveID     *int64   `json:"drive_id,omitempty"`
	PeriodStart *string  `json:"period_start,omitempty"`
	StartTime   *string  `json:"start_time,omitempty"`
	EndTime     *string  `json:"end_time,omitempty"`
	MetricValue float64  `json:"metric_value"`
	MetricUnit  string   `json:"metric_unit"`
	DistanceKM  *float64 `json:"distance_km,omitempty"`
	DurationMin *float64 `json:"duration_min,omitempty"`
	MaxSpeedKMH *float64 `json:"max_speed_kmh,omitempty"`
}
