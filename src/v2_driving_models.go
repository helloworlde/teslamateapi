package main

// @name V2DrivingAPIResponse
type V2DrivingAPIResponse struct {
	Data V2DrivingResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}

// @name V2DrivingResponse
type V2DrivingResponse struct {
	Summary V2DrivingSummary `json:"summary"`
}

// @name V2DrivingTimeseriesAPIResponse
type V2DrivingTimeseriesAPIResponse struct {
	Data V2DrivingTimeseriesResponse `json:"data"`
	Meta V2Meta                      `json:"meta"`
}

// @name V2DrivingTimeseriesResponse
type V2DrivingTimeseriesResponse struct {
	GroupBy string                    `json:"group_by"`
	Items   []V2DrivingTimeseriesItem `json:"items"`
}

// @name V2DrivingTimeseriesItem
type V2DrivingTimeseriesItem struct {
	PeriodStart             string   `json:"period_start"`
	DriveCount              int64    `json:"drive_count"`
	Distance                float64  `json:"distance"`
	Duration                float64  `json:"duration"`
	AvgSpeed                *float64 `json:"avg_speed,omitempty"`
	EstimatedEnergyConsumed *float64 `json:"estimated_energy_consumed,omitempty"`
	AvgConsumption          *float64 `json:"avg_consumption,omitempty"`
}
