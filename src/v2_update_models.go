package main

// @name V2UpdateAnalyticsResponse
type V2UpdateAnalyticsResponse struct {
	UpdateCount             int64             `json:"update_count"`
	LatestVersion           *string           `json:"latest_version,omitempty"`
	LatestUpdatedAt         *string           `json:"latest_updated_at,omitempty"`
	AvgUpdateDuration       *float64          `json:"avg_update_duration,omitempty"`
	MedianDaysBetweenUpdates *float64         `json:"median_days_between_updates,omitempty"` // (added)
	Versions                []V2UpdateVersion `json:"versions"`
}

// @name V2UpdateVersion
type V2UpdateVersion struct {
	Event   V2UpdateEvent          `json:"event"`
	Window  *V2UpdateWindow        `json:"window,omitempty"`
	Metrics *V2UpdateWindowMetrics `json:"metrics,omitempty"`
}

// @name V2UpdateEvent
type V2UpdateEvent struct {
	Version        string   `json:"version"`
	StartedAt      string   `json:"started_at"`
	CompletedAt    *string  `json:"completed_at,omitempty"`
	Duration       *float64 `json:"duration,omitempty"`
	DaysSincePrior *int64   `json:"days_since_prior,omitempty"`
}

// @name V2UpdateWindow
type V2UpdateWindow struct {
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Interval *float64 `json:"interval,omitempty"`
}

// @name V2UpdateWindowMetrics
type V2UpdateWindowMetrics struct {
	Driving          V2UpdateDrivingMetrics  `json:"driving"`
	Charging         V2UpdateChargingMetrics `json:"charging"`
	InactiveDuration *float64                `json:"inactive_duration,omitempty"`
}

// @name V2UpdateDrivingMetrics
type V2UpdateDrivingMetrics struct {
	TripCount      int64    `json:"trip_count"`
	Duration       float64  `json:"duration"`
	Distance       float64  `json:"distance"`
	NetEnergy      *float64 `json:"net_energy,omitempty"`
	AvgConsumption *float64 `json:"avg_consumption,omitempty"`
}

// @name V2UpdateChargingMetrics
type V2UpdateChargingMetrics struct {
	SessionCount  int64    `json:"session_count"`
	Duration      float64  `json:"duration"`
	BatteryEnergy *float64 `json:"battery_energy,omitempty"`
	WallEnergy    *float64 `json:"wall_energy,omitempty"`
	Efficiency    *float64 `json:"efficiency,omitempty"`
	Cost          *float64 `json:"cost,omitempty"`
}

// V2UpdateAnalyticsAPIResponse is the swagger wrapper for update analytics
// @name V2UpdateAnalyticsAPIResponse
type V2UpdateAnalyticsAPIResponse struct {
	Data V2UpdateAnalyticsResponse `json:"data"`
	Meta V2Meta                    `json:"meta"`
}
