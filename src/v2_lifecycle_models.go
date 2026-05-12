package main

// @name V2LifecycleResponse
type V2LifecycleResponse struct {
	AsOf                 *string  `json:"as_of,omitempty"`
	FirstRecordedAt      *string  `json:"first_recorded_at,omitempty"`
	LastRecordedAt       *string  `json:"last_recorded_at,omitempty"`
	RecordedDays         int64    `json:"recorded_days"`
	DriveCount           int64    `json:"drive_count"`
	Distance             float64  `json:"distance"`
	ChargingSessionCount int64    `json:"charging_session_count"`
	EnergyAdded          float64  `json:"energy_added"`
	EnergyUsed           float64  `json:"energy_used"`
	ChargingCost         float64  `json:"charging_cost"`
	UpdateCount          int64    `json:"update_count"`
	AvgDailyDistance     *float64 `json:"avg_daily_distance,omitempty"`
	AvgMonthlyDistance   *float64 `json:"avg_monthly_distance,omitempty"`
	AvgConsumption       *float64 `json:"avg_consumption,omitempty"`
	CostPerDistance      *float64 `json:"cost_per_distance,omitempty"`
}

// @name V2TimelineEvent
type V2TimelineEvent struct {
	Type      string                 `json:"type"`
	ID        int64                  `json:"id"`
	StartTime string                 `json:"start_time"`
	EndTime   *string                `json:"end_time,omitempty"`
	Title     string                 `json:"title"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// @name V2TimelineResponse
type V2TimelineResponse struct {
	Events     []V2TimelineEvent `json:"events"`
	Total      int64             `json:"total"`
	HasMore    bool              `json:"has_more"`
	NextCursor *string           `json:"next_cursor,omitempty"`
}

// V2LifecycleAPIResponse is the swagger wrapper for lifecycle analytics
// @name V2LifecycleAPIResponse
type V2LifecycleAPIResponse struct {
	Data V2LifecycleResponse `json:"data"`
	Meta V2Meta              `json:"meta"`
}

// V2TimelineAPIResponse is the swagger wrapper for timeline analytics
// @name V2TimelineAPIResponse
type V2TimelineAPIResponse struct {
	Data V2TimelineResponse `json:"data"`
	Meta V2Meta             `json:"meta"`
}
