package main

type V2LocationAnalyticsAPIResponse struct {
	Data V2LocationAnalyticsResponse `json:"data"`
	Meta V2Meta                      `json:"meta"`
}

type V2LocationAnalyticsResponse struct {
	Sort  string                    `json:"sort"`
	Items []V2LocationAnalyticsItem `json:"items"`
}

type V2LocationAnalyticsItem struct {
	LocationName         string   `json:"location_name"`
	GeofenceID           *int64   `json:"geofence_id,omitempty"`
	AddressID            *int64   `json:"address_id,omitempty"`
	DriveStartCount      int64    `json:"drive_start_count"`
	DriveEndCount        int64    `json:"drive_end_count"`
	ChargingSessionCount int64    `json:"charging_session_count"`
	ParkingSessionCount  int64    `json:"parking_session_count"`
	ParkingDurationMin   float64  `json:"parking_duration_min"`
	EnergyAddedKWh       float64  `json:"energy_added_kwh"`
	ChargingCost         *float64 `json:"charging_cost,omitempty"`
	VampireDrainPercent  float64  `json:"vampire_drain_percent"`
}
