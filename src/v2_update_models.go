package main

type V2UpdateAnalyticsResponse struct {
	UpdateCount          int64             `json:"update_count"`
	LatestVersion        *string           `json:"latest_version,omitempty"`
	LatestUpdatedAt      *string           `json:"latest_updated_at,omitempty"`
	AvgUpdateDurationMin *float64          `json:"avg_update_duration_min,omitempty"`
	Versions             []V2UpdateVersion `json:"versions"`
}

type V2UpdateVersion struct {
	Version     string   `json:"version"`
	StartedAt   string   `json:"started_at"`
	CompletedAt *string  `json:"completed_at,omitempty"`
	DurationMin *float64 `json:"duration_min,omitempty"`
}

// V2UpdateAnalyticsAPIResponse is the swagger wrapper for update analytics
type V2UpdateAnalyticsAPIResponse struct {
	Data V2UpdateAnalyticsResponse `json:"data"`
	Meta V2Meta                    `json:"meta"`
}
