package v2

// @name V2UpdateAnalyticsResponse
type V2UpdateAnalyticsResponse struct {
	UpdateCount              int64             `json:"update_count"`                          // OTA 更新次数
	LatestVersion            *string           `json:"latest_version,omitempty"`              // 最近版本
	LatestUpdatedAt          *string           `json:"latest_updated_at,omitempty"`           // 最近更新时间
	AvgUpdateDuration        *float64          `json:"avg_update_duration,omitempty"`         // 平均更新时长 (秒)
	MedianDaysBetweenUpdates *float64          `json:"median_days_between_updates,omitempty"` // (added)
	Versions                 []V2UpdateVersion `json:"versions"`                              // 版本列表
}

// @name V2UpdateVersion
type V2UpdateVersion struct {
	Event   V2UpdateEvent          `json:"event"`             // 事件
	Window  *V2UpdateWindow        `json:"window,omitempty"`  // 观察窗口
	Metrics *V2UpdateWindowMetrics `json:"metrics,omitempty"` // 指标
}

// @name V2UpdateEvent
type V2UpdateEvent struct {
	Version        string   `json:"version"`                    // 版本
	StartedAt      string   `json:"started_at"`                 // 开始时间
	CompletedAt    *string  `json:"completed_at,omitempty"`     // 完成时间
	Duration       *float64 `json:"duration,omitempty"`         // 时长 (秒)
	DaysSincePrior *int64   `json:"days_since_prior,omitempty"` // 距上次更新天数
}

// @name V2UpdateWindow
type V2UpdateWindow struct {
	Start    string   `json:"start"`              // 起始时间
	End      string   `json:"end"`                // 结束时间
	Interval *float64 `json:"interval,omitempty"` // 区间秒数
}

// @name V2UpdateWindowMetrics
type V2UpdateWindowMetrics struct {
	Driving          V2UpdateDrivingMetrics  `json:"driving"`                     // 行驶指标
	Charging         V2UpdateChargingMetrics `json:"charging"`                    // 充电指标
	InactiveDuration *float64                `json:"inactive_duration,omitempty"` // 不活跃时长 (秒)
}

// @name V2UpdateDrivingMetrics
type V2UpdateDrivingMetrics struct {
	TripCount      int64    `json:"trip_count"`                // 行程数
	Duration       float64  `json:"duration"`                  // 时长 (秒)
	Distance       float64  `json:"distance"`                  // 距离 (km)
	NetEnergy      *float64 `json:"net_energy,omitempty"`      // 净能量 (kWh)
	AvgConsumption *float64 `json:"avg_consumption,omitempty"` // 平均能耗 (Wh/km)
}

// @name V2UpdateChargingMetrics
type V2UpdateChargingMetrics struct {
	SessionCount  int64    `json:"session_count"`            // 充电会话数
	Duration      float64  `json:"duration"`                 // 时长 (秒)
	BatteryEnergy *float64 `json:"battery_energy,omitempty"` // 电池端能量 (kWh)
	WallEnergy    *float64 `json:"wall_energy,omitempty"`    // 墙端用电 (kWh)
	Efficiency    *float64 `json:"efficiency,omitempty"`     // 效率
	Cost          *float64 `json:"cost,omitempty"`           // 费用
}

// V2UpdateAnalyticsAPIResponse is the swagger wrapper for update analytics
// @name V2UpdateAnalyticsAPIResponse
type V2UpdateAnalyticsAPIResponse struct {
	Data V2UpdateAnalyticsResponse `json:"data"` // 响应数据
	Meta V2Meta                    `json:"meta"` // 响应元信息
}
