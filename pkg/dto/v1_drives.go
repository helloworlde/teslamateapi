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
	// EstimatedUsageCost = (lifetime charging cost / lifetime distance) × this
	// drive's distance. Semi-objective: amortises a global per-distance rate
	// onto one trip. 0 when no charging cost is configured; null when there is
	// no lifetime distance yet. In the same currency as charging_processes.cost.
	EstimatedUsageCost *float64 `json:"estimated_usage_cost" example:"3.42"`
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
