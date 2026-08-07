package dto

// V1DrivesOdometerDetails carries start/end/distance odometer values.
type V1DrivesOdometerDetails struct {
	OdometerStart    float64 `json:"odometer_start" example:"42000.0"`
	OdometerEnd      float64 `json:"odometer_end" example:"42100.0"`
	OdometerDistance float64 `json:"odometer_distance" example:"100.0"`
}

// V1DrivesBatteryDetails carries SOC and reduced-range flags.
type V1DrivesBatteryDetails struct {
	StartUsableBatteryLevel int  `json:"start_usable_battery_level" example:"82"`
	StartBatteryLevel       int  `json:"start_battery_level" example:"82"`
	EndUsableBatteryLevel   int  `json:"end_usable_battery_level" example:"50"`
	EndBatteryLevel         int  `json:"end_battery_level" example:"50"`
	ReducedRange            bool `json:"reduced_range" example:"false"`
	IsSufficientlyPrecise   bool `json:"is_sufficiently_precise" example:"true"`
}

// V1DrivesPreferredRange carries start/end/diff range for one drive.
type V1DrivesPreferredRange struct {
	StartRange float64 `json:"start_range" example:"320.5"`
	EndRange   float64 `json:"end_range" example:"180.5"`
	RangeDiff  float64 `json:"range_diff" example:"140.0"`
}

// V1DriveListItem represents a single drive row in
// GET /api/v1/cars/{CarID}/drives.
type V1DriveListItem struct {
	DriveID           int                     `json:"drive_id" example:"1234"`
	StartDate         string                  `json:"start_date" example:"2024-01-01T08:00:00+01:00"`
	EndDate           string                  `json:"end_date" example:"2024-01-01T09:30:00+01:00"`
	StartAddress      string                  `json:"start_address" example:"Home"`
	EndAddress        string                  `json:"end_address" example:"Office"`
	OdometerDetails   V1DrivesOdometerDetails `json:"odometer_details"`
	DurationMin       int                     `json:"duration_min" example:"90"`
	DurationStr       string                  `json:"duration_str" example:"01:30"`
	SpeedMax          int                     `json:"speed_max" example:"130"`
	SpeedAvg          float64                 `json:"speed_avg" example:"66.7"`
	PowerMax          int                     `json:"power_max" example:"250"`
	PowerMin          int                     `json:"power_min" example:"-50"`
	BatteryDetails    V1DrivesBatteryDetails  `json:"battery_details"`
	RangeIdeal        V1DrivesPreferredRange  `json:"range_ideal"`
	RangeRated        V1DrivesPreferredRange  `json:"range_rated"`
	OutsideTempAvg    float64                 `json:"outside_temp_avg" example:"15.3"`
	InsideTempAvg     float64                 `json:"inside_temp_avg" example:"21.0"`
	EnergyConsumedNet *float64                `json:"energy_consumed_net" example:"22.5"`
	ConsumptionNet    *float64                `json:"consumption_net" example:"225.0"`
	// RangeAchievementPct = distance / rated-range drop × 100 (objective).
	// Null when the rated-range drop is non-positive (charging mid-drive,
	// missing range readings). Unit-independent: a ratio of two distances.
	RangeAchievementPct *float64 `json:"range_achievement_pct" example:"92.5"`
	// EstimatedUsageCost = energy_consumed_net × average charging price
	// (SUM(cost) / SUM(charge_energy_added)). 0 when charge energy exists but
	// no charging cost is configured; null when energy_consumed_net or average
	// charging price is unavailable. Currency matches charging_processes.cost.
	EstimatedUsageCost *float64 `json:"estimated_usage_cost" example:"3.42"`
	// Route is the drive's downsampled path as [latitude, longitude] pairs,
	// requested with include_route=true. Absent otherwise — and also absent
	// for a drive whose positions carry no coordinates, so a client must
	// handle a missing route even on a request that asked for one. Pairs
	// rather than objects because a map view fetches hundreds of routes at
	// once and repeated key names would dominate the payload.
	//
	// Sampled, not raw: it keeps the first and last point plus the
	// latitude/longitude extrema so a client can frame a map camera from it,
	// but it must not be used for distance or duration maths — the top-level
	// summary fields remain authoritative for those.
	//
	// max_points_per_drive is a target, not a guarantee: a page holding many
	// drives lowers it so the whole response stays within a fixed point
	// budget.
	Route [][2]float64 `json:"route,omitempty"`
}

// V1DrivesData is the `data` field of V1DrivesResponse.
type V1DrivesData struct {
	Car            Car               `json:"car"`
	Drives         []V1DriveListItem `json:"drives"`
	TeslaMateUnits TeslaMateUnits    `json:"units"`
}

// V1DrivesResponse is the envelope for /api/v1/cars/{CarID}/drives.
type V1DrivesResponse struct {
	Data V1DrivesData `json:"data"`
}
