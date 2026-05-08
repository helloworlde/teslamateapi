package main

type V2CalendarDay struct {
	Date                 string          `json:"date"`
	DriveCount           int64           `json:"drive_count"`
	DistanceKM           float64         `json:"distance_km"`
	DriveDurationMin     float64         `json:"drive_duration_min"`
	ChargingSessionCount int64           `json:"charging_session_count"`
	EnergyAddedKWh       float64         `json:"energy_added_kwh"`
	ChargingCost         float64         `json:"charging_cost"`
	ParkingDurationMin   float64         `json:"parking_duration_min"`
	VampireDrainPercent  float64         `json:"vampire_drain_percent"`
	UpdateCount          int64           `json:"update_count"`
	ActivityLevel        V2ActivityLevel `json:"activity_level"`
}

type V2ActivityLevel struct {
	Driving      int `json:"driving"`
	Charging     int `json:"charging"`
	ParkingDrain int `json:"parking_drain"`
}

type V2CalendarResponse struct {
	Days []V2CalendarDay `json:"days"`
}

// V2CalendarAPIResponse is the swagger wrapper for calendar analytics
type V2CalendarAPIResponse struct {
	Data V2CalendarResponse `json:"data"`
	Meta V2Meta             `json:"meta"`
}
