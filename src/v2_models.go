package main

import "time"

const (
	v2DefaultPeriod  = "month"
	v2DefaultCompare = "none"
)

var (
	v2AllowedPeriods = map[string]bool{
		"day":      true,
		"week":     true,
		"month":    true,
		"quarter":  true,
		"year":     true,
		"custom":   true,
		"lifetime": true,
	}
	v2AllowedCompares = map[string]bool{
		"none":             true,
		"previous_period":  true,
		"previous_year":    true,
		"lifetime_average": true,
	}
)

type V2AnalyticsQuery struct {
	Period   string `form:"period" json:"period" example:"month"`
	Start    string `form:"start" json:"start" example:"2026-05-01T00:00:00+08:00"`
	End      string `form:"end" json:"end" example:"2026-05-31T23:59:59+08:00"`
	Timezone string `form:"timezone" json:"timezone" example:"Asia/Shanghai"`
	Compare  string `form:"compare" json:"compare" example:"previous_period"`
	GroupBy  string `form:"group_by" json:"group_by,omitempty" example:"day"`
	Metrics  string `form:"metrics" json:"metrics,omitempty" example:"distance_km,duration_min"`
	Include  string `form:"include" json:"include,omitempty" example:"summary,comparison,timeseries"`
}

type V2TimeRange struct {
	Period        string
	Timezone      string
	Compare       string
	Start         time.Time
	End           time.Time
	PreviousStart *time.Time
	PreviousEnd   *time.Time
}

type V2Unit struct {
	Distance    string `json:"distance" example:"km"`
	Energy      string `json:"energy" example:"kWh"`
	Power       string `json:"power" example:"kW"`
	Temperature string `json:"temperature" example:"C"`
	Currency    string `json:"currency" example:"CNY"`
}

type V2DataQuality struct {
	Complete      bool     `json:"complete"`
	SampleCount   int64    `json:"sample_count,omitempty"`
	MissingFields []string `json:"missing_fields,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

type V2Meta struct {
	CarID       int64          `json:"car_id,omitempty"`
	Period      string         `json:"period,omitempty"`
	Timezone    string         `json:"timezone,omitempty"`
	Start       string         `json:"start,omitempty"`
	End         string         `json:"end,omitempty"`
	Compare     string         `json:"compare,omitempty"`
	Unit        V2Unit         `json:"unit"`
	GeneratedAt string         `json:"generated_at"`
	DataQuality *V2DataQuality `json:"data_quality,omitempty"`
}

type V2APIResponse struct {
	Data interface{} `json:"data"`
	Meta V2Meta      `json:"meta"`
}

type APIErrorResponse struct {
	Error APIErrorBody `json:"error"`
}

type APIErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type V2ComparisonValue struct {
	Current      *float64 `json:"current,omitempty"`
	Previous     *float64 `json:"previous,omitempty"`
	Delta        *float64 `json:"delta,omitempty"`
	DeltaPercent *float64 `json:"delta_percent,omitempty"`
}

type V2InfoResponse struct {
	Version  string   `json:"version" example:"v2"`
	Scope    string   `json:"scope" example:"analytics"`
	Features []string `json:"features"`
}

type V2InfoAPIResponse struct {
	Data V2InfoResponse `json:"data"`
	Meta V2Meta         `json:"meta"`
}

type V2SummaryAPIResponse struct {
	Data V2SummaryResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}

type V2SummaryResponse struct {
	Summary    V2Summary                    `json:"summary"`
	Comparison map[string]V2ComparisonValue `json:"comparison,omitempty"`
}

type V2Summary struct {
	Driving  V2DrivingSummary  `json:"driving"`
	Charging V2ChargingSummary `json:"charging"`
	Parking  V2ParkingSummary  `json:"parking"`
	Battery  V2BatterySummary  `json:"battery"`
	Updates  V2UpdateSummary   `json:"updates"`
	Cost     V2CostSummary     `json:"cost"`
}

type V2DrivingSummary struct {
	DriveCount            int64   `json:"drive_count"`
	DistanceKM            float64 `json:"distance_km"`
	DurationMin           float64 `json:"duration_min"`
	MaxSpeedKMH           float64 `json:"max_speed_kmh"`
	AvgConsumptionWhPerKM float64 `json:"avg_consumption_wh_per_km"`
}

type V2ChargingSummary struct {
	SessionCount   int64   `json:"session_count"`
	EnergyAddedKWh float64 `json:"energy_added_kwh"`
	EnergyUsedKWh  float64 `json:"energy_used_kwh"`
	DurationMin    float64 `json:"duration_min"`
	Cost           float64 `json:"cost"`
}

type V2ParkingSummary struct {
	ParkedDurationMin   float64 `json:"parked_duration_min"`
	AsleepDurationMin   float64 `json:"asleep_duration_min"`
	OnlineDurationMin   float64 `json:"online_duration_min"`
	OfflineDurationMin  float64 `json:"offline_duration_min"`
	VampireDrainPercent float64 `json:"vampire_drain_percent"`
}

type V2BatterySummary struct {
	LatestBatteryLevelPercent *int64   `json:"latest_battery_level_percent,omitempty"`
	LatestRatedRangeKM        *float64 `json:"latest_rated_range_km,omitempty"`
	LatestIdealRangeKM        *float64 `json:"latest_ideal_range_km,omitempty"`
}

type V2UpdateSummary struct {
	UpdateCount   int64   `json:"update_count"`
	LatestVersion *string `json:"latest_version,omitempty"`
}

type V2CostSummary struct {
	ChargingCost float64  `json:"charging_cost"`
	CostPerKM    *float64 `json:"cost_per_km,omitempty"`
	CostPer100KM *float64 `json:"cost_per_100km,omitempty"`
}
