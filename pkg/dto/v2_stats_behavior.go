package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2BehaviorHeatmapCell is one weekday×hour cell of the usage heatmap. Both
// figures are raw counts over session start times — objective, no thresholds.
// Weekday follows Postgres DOW (0=Sunday … 6=Saturday); hour is 0–23. The
// timestamps are bucketed in the user's timezone so "8am" is local wall-clock.
type V2BehaviorHeatmapCell struct {
	Weekday               int     `json:"weekday" example:"1"`
	Hour                  int     `json:"hour" example:"8"`
	DrivesCount           int     `json:"drives_count" example:"42"`
	DrivesDistance        float64 `json:"drives_distance" example:"512.4"`
	DrivesDurationMin     int     `json:"drives_duration_min" example:"780"`
	ChargesCount          int     `json:"charges_count" example:"3"`
	ChargesEnergyAddedKWh float64 `json:"charges_energy_added_kwh" example:"52.1"`
	ChargesEnergyUsedKWh  float64 `json:"charges_energy_used_kwh" example:"54.0"`
	ChargesDurationMin    int     `json:"charges_duration_min" example:"180"`
}

// V2BehaviorChargeLevelBucket is one 10-point state-of-charge band. StartCount
// / EndCount are how many charging sessions began / ended with their battery
// level in [BucketLow, BucketHigh]. Objective: a direct histogram of
// charging_processes.start_battery_level / end_battery_level.
type V2BehaviorChargeLevelBucket struct {
	BucketLow  int `json:"bucket_low" example:"20"`
	BucketHigh int `json:"bucket_high" example:"30"`
	StartCount int `json:"start_count" example:"58"`
	EndCount   int `json:"end_count" example:"4"`
}

// V2BehaviorTripTypeBucket is one distance band of the trip-length histogram.
// All figures are objective: trip counts, summed distance, and energy derived
// from rated-range drop × efficiency (the shared energy model). DistHigh is
// null for the open-ended top band.
type V2BehaviorTripTypeBucket struct {
	Key        string           `json:"key" example:"5"`
	DistLow    float64          `json:"dist_low" example:"5"`
	DistHigh   nullable.Float64 `json:"dist_high" swaggertype:"number" example:"20"`
	TripsCount int              `json:"trips_count" example:"312"`
	Distance   float64          `json:"distance" example:"3840.0"`
	EnergyKWh  float64          `json:"energy_kwh" example:"690.0"`
}

// V2BehaviorData is the `data` field of V2BehaviorResponse. It bundles three
// independent objective distributions so a client can render the whole
// behaviour-profile screen from one request. Every section scans only the
// per-session drives / charging_processes tables — never positions.
type V2BehaviorData struct {
	Car          Car                           `json:"car"`
	Heatmap      []V2BehaviorHeatmapCell       `json:"heatmap"`
	ChargeLevels []V2BehaviorChargeLevelBucket `json:"charge_levels"`
	TripTypes    []V2BehaviorTripTypeBucket    `json:"trip_types"`
	Units        TeslaMateUnits                `json:"units"`
}

// V2BehaviorResponse is the envelope for
// /api/v2/cars/{CarID}/stats/behavior.
type V2BehaviorResponse struct {
	Data V2BehaviorData `json:"data"`
}
