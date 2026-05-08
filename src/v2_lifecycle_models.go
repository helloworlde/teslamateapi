package main

type V2LifecycleResponse struct {
	FirstRecordedAt       *string  `json:"first_recorded_at,omitempty"`
	LastRecordedAt        *string  `json:"last_recorded_at,omitempty"`
	RecordedDays          int64    `json:"recorded_days"`
	DriveCount            int64    `json:"drive_count"`
	DistanceKM            float64  `json:"distance_km"`
	ChargingSessionCount  int64    `json:"charging_session_count"`
	EnergyAddedKWh        float64  `json:"energy_added_kwh"`
	EnergyUsedKWh         float64  `json:"energy_used_kwh"`
	ChargingCost          float64  `json:"charging_cost"`
	UpdateCount           int64    `json:"update_count"`
	AvgDailyDistanceKM    *float64 `json:"avg_daily_distance_km,omitempty"`
	AvgMonthlyDistanceKM  *float64 `json:"avg_monthly_distance_km,omitempty"`
	AvgConsumptionWhPerKM *float64 `json:"avg_consumption_wh_per_km,omitempty"`
	CostPer100KM          *float64 `json:"cost_per_100km,omitempty"`
}

type V2TimelineEvent struct {
	Type      string                 `json:"type"`
	ID        int64                  `json:"id"`
	StartTime string                 `json:"start_time"`
	EndTime   *string                `json:"end_time,omitempty"`
	Title     string                 `json:"title"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

type V2TimelineResponse struct {
	Events []V2TimelineEvent `json:"events"`
	Total  int64             `json:"total"`
}

// V2LifecycleAPIResponse is the swagger wrapper for lifecycle analytics
type V2LifecycleAPIResponse struct {
	Data V2LifecycleResponse `json:"data"`
	Meta V2Meta              `json:"meta"`
}

// V2TimelineAPIResponse is the swagger wrapper for timeline analytics
type V2TimelineAPIResponse struct {
	Data V2TimelineResponse `json:"data"`
	Meta V2Meta             `json:"meta"`
}
