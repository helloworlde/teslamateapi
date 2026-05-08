package main

type V2ParkingAPIResponse struct {
	Data V2ParkingResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}

type V2ParkingResponse struct {
	Summary    V2ParkingAnalyticsSummary    `json:"summary"`
	Comparison map[string]V2ComparisonValue `json:"comparison,omitempty"`
}

type V2ParkingLocationsAPIResponse struct {
	Data V2ParkingLocationsResponse `json:"data"`
	Meta V2Meta                     `json:"meta"`
}

type V2ParkingLocationsResponse struct {
	Items []V2ParkingLocationItem `json:"items"`
}

type V2ParkingStatesAPIResponse struct {
	Data V2ParkingStatesResponse `json:"data"`
	Meta V2Meta                  `json:"meta"`
}

type V2ParkingStatesResponse struct {
	TotalDurationMin     float64              `json:"total_duration_min"`
	StateTransitionCount int64                `json:"state_transition_count"`
	Items                []V2ParkingStateItem `json:"items"`
}

type V2ParkingAnalyticsSummary struct {
	ParkingSessionCount      int64    `json:"parking_session_count"`
	ParkedDurationMin        float64  `json:"parked_duration_min"`
	AvgParkedDurationMin     *float64 `json:"avg_parked_duration_min,omitempty"`
	AsleepDurationMin        float64  `json:"asleep_duration_min"`
	OnlineDurationMin        float64  `json:"online_duration_min"`
	OfflineDurationMin       float64  `json:"offline_duration_min"`
	VampireDrainPercent      *float64 `json:"vampire_drain_percent,omitempty"`
	VampireDrainRangeKM      *float64 `json:"vampire_drain_range_km,omitempty"`
	EstimatedVampireDrainKWh *float64 `json:"estimated_vampire_drain_kwh,omitempty"`
	AvgDrainPercentPerDay    *float64 `json:"avg_drain_percent_per_day,omitempty"`
	StateTransitionCount     int64    `json:"state_transition_count"`
}

type V2ParkingLocationItem struct {
	LocationName             string   `json:"location_name"`
	GeofenceID               *int64   `json:"geofence_id,omitempty"`
	AddressID                *int64   `json:"address_id,omitempty"`
	ParkingSessionCount      int64    `json:"parking_session_count"`
	ParkedDurationMin        float64  `json:"parked_duration_min"`
	AvgParkedDurationMin     *float64 `json:"avg_parked_duration_min,omitempty"`
	AsleepDurationMin        float64  `json:"asleep_duration_min"`
	OnlineDurationMin        float64  `json:"online_duration_min"`
	OfflineDurationMin       float64  `json:"offline_duration_min"`
	VampireDrainPercent      *float64 `json:"vampire_drain_percent,omitempty"`
	VampireDrainRangeKM      *float64 `json:"vampire_drain_range_km,omitempty"`
	EstimatedVampireDrainKWh *float64 `json:"estimated_vampire_drain_kwh,omitempty"`
}

type V2ParkingStateItem struct {
	State           string  `json:"state"`
	DurationMin     float64 `json:"duration_min"`
	Percent         float64 `json:"percent"`
	TransitionCount int64   `json:"transition_count"`
}
