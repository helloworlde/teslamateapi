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

	DaysSincePrior *int64   `json:"days_since_prior,omitempty"`
	WindowStart    *string  `json:"window_start,omitempty"`
	WindowEnd      *string  `json:"window_end,omitempty"`
	IntervalMin    *float64 `json:"interval_min,omitempty"`

	DrivingTripCount     *int64   `json:"driving_trip_count,omitempty"`
	DrivingDurationMin   *float64 `json:"driving_duration_min,omitempty"`
	DrivingDistanceKM    *float64 `json:"driving_distance_km,omitempty"`
	NetDriveEnergyKWh    *float64 `json:"net_drive_energy_kwh,omitempty"`
	DriveEfficiencyWhPerKM  *float64 `json:"drive_efficiency_wh_per_km,omitempty"`
	AvgConsumptionKWhPer100KM *float64 `json:"avg_consumption_kwh_per_100km,omitempty"`

	ChargingSessionCount    *int64   `json:"charging_session_count,omitempty"`
	ChargingDurationMin     *float64 `json:"charging_duration_min,omitempty"`
	BatteryEnergyKWh        *float64 `json:"battery_energy_kwh,omitempty"`
	WallEnergyKWh           *float64 `json:"wall_energy_kwh,omitempty"`
	ChargeEfficiencyPercent *float64 `json:"charge_efficiency_percent,omitempty"`
	ChargeCost              *float64 `json:"charge_cost,omitempty"`

	InactiveDurationMin *float64 `json:"inactive_duration_min,omitempty"`
}

// V2UpdateAnalyticsAPIResponse is the swagger wrapper for update analytics
type V2UpdateAnalyticsAPIResponse struct {
	Data V2UpdateAnalyticsResponse `json:"data"`
	Meta V2Meta                    `json:"meta"`
}
