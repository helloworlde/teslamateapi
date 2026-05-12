package main

// @name V2ParkingAPIResponse
type V2ParkingAPIResponse struct {
	Data V2ParkingResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}

// @name V2ParkingResponse
type V2ParkingResponse struct {
	Summary   V2ParkingSummary            `json:"summary"`
	Breakdown *V2ParkingBreakdownResponse `json:"breakdown,omitempty"`
}

// @name V2ParkingBreakdownResponse
type V2ParkingBreakdownResponse struct {
	By        string                  `json:"by"`
	Locations []V2ParkingLocationItem `json:"locations,omitempty"`
	States    []V2ParkingStateItem    `json:"states,omitempty"`
}

// V2ParkingStatesResponse is the internal shape returned by the parking
// repository's state breakdown query. It is not directly exposed via swagger
// after consolidation; values are folded into V2ParkingBreakdownResponse.
type V2ParkingStatesResponse struct {
	TotalDuration        float64              `json:"total_duration"`
	StateTransitionCount int64                `json:"state_transition_count"`
	Items                []V2ParkingStateItem `json:"items"`
}

// @name V2ParkingLocationItem
type V2ParkingLocationItem struct {
	LocationName        string   `json:"location_name"`
	GeofenceID          *int64   `json:"geofence_id,omitempty"`
	AddressID           *int64   `json:"address_id,omitempty"`
	ParkingSessionCount int64    `json:"parking_session_count"`
	ParkedDuration      float64  `json:"parked_duration"`
	AvgParkedDuration   *float64 `json:"avg_parked_duration,omitempty"`
}

// @name V2ParkingStateItem
type V2ParkingStateItem struct {
	State           string  `json:"state"`
	Duration        float64 `json:"duration"`
	Percent         float64 `json:"percent"`
	TransitionCount int64   `json:"transition_count"`
}
