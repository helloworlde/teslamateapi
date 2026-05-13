package main

// @name V2DrivingAPIResponse
type V2DrivingAPIResponse struct {
	Data V2DrivingResponse `json:"data"` // 响应数据
	Meta V2Meta `json:"meta"` // 响应元信息
}

// @name V2DrivingResponse
type V2DrivingResponse struct {
	Summary V2DrivingSummary `json:"summary"`
}

// @name V2DrivingTimeseriesAPIResponse
type V2DrivingTimeseriesAPIResponse struct {
	Data V2DrivingTimeseriesResponse `json:"data"` // 响应数据
	Meta V2Meta `json:"meta"` // 响应元信息
}

// @name V2DrivingTimeseriesResponse
type V2DrivingTimeseriesResponse struct {
	GroupBy string `json:"group_by"` // 聚合粒度
	Items   []V2DrivingTimeseriesItem `json:"items"` // 条目列表
}

// @name V2DrivingTimeseriesItem
type V2DrivingTimeseriesItem struct {
	PeriodStart             string `json:"period_start"` // 周期起始
	DriveCount              int64 `json:"drive_count"` // 行程数
	Distance                float64 `json:"distance"` // 距离 (km)
	Duration                float64 `json:"duration"` // 时长 (秒)
	AvgSpeed                *float64 `json:"avg_speed,omitempty"` // 平均速度 (km/h)
	EstimatedEnergyConsumed *float64 `json:"estimated_energy_consumed,omitempty"`
	AvgConsumption          *float64 `json:"avg_consumption,omitempty"` // 平均能耗 (Wh/km)
}
