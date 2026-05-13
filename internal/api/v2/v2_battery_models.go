package v2

// @name V2BatteryAPIResponse
type V2BatteryAPIResponse struct {
	Data V2BatteryResponse `json:"data"` // 响应数据
	Meta V2Meta            `json:"meta"` // 响应元信息
}

// @name V2BatteryResponse
type V2BatteryResponse struct {
	Summary V2BatterySummary `json:"summary"`
}

// @name V2BatteryTimeseriesAPIResponse
type V2BatteryTimeseriesAPIResponse struct {
	Data V2BatteryTimeseriesResponse `json:"data"` // 响应数据
	Meta V2Meta                      `json:"meta"` // 响应元信息
}

// @name V2BatteryTimeseriesResponse
type V2BatteryTimeseriesResponse struct {
	GroupBy string                    `json:"group_by"` // 聚合粒度
	Items   []V2BatteryTimeseriesItem `json:"items"`    // 条目列表
}

// @name V2BatteryTimeseriesItem
type V2BatteryTimeseriesItem struct {
	PeriodStart       string          `json:"period_start"`                   // 周期起始
	RangeAtFullCharge *V2BatteryRange `json:"range_at_full_charge,omitempty"` // 满电续航 (km)
	AvgLevel          *float64        `json:"avg_level,omitempty"`
}
