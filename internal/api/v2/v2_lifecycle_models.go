package v2

// @name V2LifecycleResponse
type V2LifecycleResponse struct {
	AsOf                 *string  `json:"as_of,omitempty"` // 快照时间
	FirstRecordedAt      *string  `json:"first_recorded_at,omitempty"`
	LastRecordedAt       *string  `json:"last_recorded_at,omitempty"`
	RecordedDays         int64    `json:"recorded_days"`
	DriveCount           int64    `json:"drive_count"`            // 行程数
	Distance             float64  `json:"distance"`               // 距离 (km)
	ChargingSessionCount int64    `json:"charging_session_count"` // 充电会话数
	EnergyAdded          float64  `json:"energy_added"`           // 充入电池能量 (kWh)
	EnergyUsed           float64  `json:"energy_used"`            // 墙端用电 (kWh)
	ChargingCost         float64  `json:"charging_cost"`          // 充电费用
	UpdateCount          int64    `json:"update_count"`           // OTA 更新次数
	AvgDailyDistance     *float64 `json:"avg_daily_distance,omitempty"`
	AvgMonthlyDistance   *float64 `json:"avg_monthly_distance,omitempty"`
	AvgConsumption       *float64 `json:"avg_consumption,omitempty"`   // 平均能耗 (Wh/km)
	CostPerDistance      *float64 `json:"cost_per_distance,omitempty"` // 单位里程费用
}

// @name V2TimelineEvent
type V2TimelineEvent struct {
	Type      string                 `json:"type"` // 类型
	ID        int64                  `json:"id"`   // ID
	StartTime string                 `json:"start_time"`
	EndTime   *string                `json:"end_time,omitempty"`
	Title     string                 `json:"title"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// @name V2TimelineResponse
type V2TimelineResponse struct {
	Events     []V2TimelineEvent `json:"events"` // 事件列表
	Total      int64             `json:"total"`
	HasMore    bool              `json:"has_more"`              // 是否还有更多
	NextCursor *string           `json:"next_cursor,omitempty"` // 下一页游标
}

// V2LifecycleAPIResponse is the swagger wrapper for lifecycle analytics
// @name V2LifecycleAPIResponse
type V2LifecycleAPIResponse struct {
	Data V2LifecycleResponse `json:"data"` // 响应数据
	Meta V2Meta              `json:"meta"` // 响应元信息
}

// V2TimelineAPIResponse is the swagger wrapper for timeline analytics
// @name V2TimelineAPIResponse
type V2TimelineAPIResponse struct {
	Data V2TimelineResponse `json:"data"` // 响应数据
	Meta V2Meta             `json:"meta"` // 响应元信息
}
