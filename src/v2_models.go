package main

import "time"

const (
	v2DefaultPeriod  = "month"
	v2DefaultCompare = "none"
)

var (
	v2AllowedPeriods = map[string]bool{
		"day":     true,
		"week":    true,
		"month":   true,
		"quarter": true,
		"year":    true,
		"custom":  true,
	}
	v2AllowedCompares = map[string]bool{
		"none":            true,
		"previous_period": true,
	}
)

// @name V2AnalyticsQuery
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

// @name V2TimeRange
type V2TimeRange struct {
	Period        string
	Timezone      string
	Compare       string
	Start         time.Time
	End           time.Time
	PreviousStart *time.Time
	PreviousEnd   *time.Time
}

// @name V2Unit
type V2Unit struct {
	Distance    string `json:"distance" example:"km"`
	Energy      string `json:"energy" example:"kWh"`
	Power       string `json:"power" example:"kW"`
	Temperature string `json:"temperature" example:"C"`
	Currency    string `json:"currency" example:"CNY"`
}

// @name V2Meta
type V2Meta struct {
	CarID       int64  `json:"car_id,omitempty" example:"1"`
	Period      string `json:"period,omitempty" enums:"day,week,month,quarter,year,custom,lifetime"`
	Timezone    string `json:"timezone,omitempty" example:"Asia/Shanghai"`
	Start       string `json:"start,omitempty" format:"date-time"`
	End         string `json:"end,omitempty" format:"date-time"`
	Compare     string `json:"compare,omitempty" enums:"none,previous_period"`
	Unit        V2Unit `json:"unit"`
	GeneratedAt string `json:"generated_at" format:"date-time"`
}

// @name V2APIResponse
type V2APIResponse struct {
	Data interface{} `json:"data"`
	Meta V2Meta      `json:"meta"`
}

// @name APIErrorResponse
type APIErrorResponse struct {
	Error APIErrorBody `json:"error"`
}

// @name APIErrorBody
type APIErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty" swaggertype:"object"`
}

// @name V2ComparisonValue
type V2ComparisonValue struct {
	Current      *float64 `json:"current,omitempty"`
	Previous     *float64 `json:"previous,omitempty"`
	Delta        *float64 `json:"delta,omitempty"`
	DeltaPercent *float64 `json:"delta_percent,omitempty"`
}

// @name V2CapabilitiesResponse
type V2CapabilitiesResponse struct {
	Version          string                 `json:"version" example:"v2"`
	Domains          []V2CapabilitiesDomain `json:"domains"`
	BreakdownOptions map[string][]string    `json:"breakdown_options"`
}

// @name V2CapabilitiesDomain
type V2CapabilitiesDomain struct {
	Name              string `json:"name" example:"driving"`
	Path              string `json:"path" example:"/v2/cars/{car_id}/analytics/driving"`
	SupportsCompare   bool   `json:"supports_compare"`
	SupportsTimeseries bool  `json:"supports_timeseries"`
	SupportsBreakdown bool   `json:"supports_breakdown"`
}

// @name V2CapabilitiesAPIResponse
type V2CapabilitiesAPIResponse struct {
	Data V2CapabilitiesResponse `json:"data"`
	Meta V2Meta                 `json:"meta"`
}

// @name V2SummaryAPIResponse
type V2SummaryAPIResponse struct {
	Data V2SummaryResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}

// @name V2SummaryResponse
type V2SummaryResponse struct {
	Summary V2Summary `json:"summary"`
	// Comparison keys: driving/charging/parking/battery/updates/vehicle/cost field paths; values are current vs previous period.
	Comparison map[string]V2ComparisonValue `json:"comparison,omitempty"`
}

// @name V2Summary
type V2Summary struct {
	Driving  V2DrivingSummary  `json:"driving"`
	Charging V2ChargingSummary `json:"charging"`
	Parking  V2ParkingSummary  `json:"parking"`
	Battery  V2BatterySummary  `json:"battery"`
	Updates  V2UpdateSummary   `json:"updates"`
	Vehicle  V2VehicleSummary  `json:"vehicle"`
	Cost     V2CostSummary     `json:"cost"`
}

// @name V2DrivingSummary
type V2DrivingSummary struct {
	DriveCount              int64    `json:"drive_count"`
	DistanceKM              float64  `json:"distance_km"`
	DurationMin             float64  `json:"duration_min"`
	AvgTripDistanceKM       *float64 `json:"avg_trip_distance_km,omitempty"`
	AvgDurationMin          *float64 `json:"avg_duration_min,omitempty"`
	AvgSpeedKMH             *float64 `json:"avg_speed_kmh,omitempty"`
	LongestDriveDurationMin *float64 `json:"longest_drive_duration_min,omitempty"`
	MaxSpeedKMH             float64  `json:"max_speed_kmh"`
	PeakDrivePowerKW        *float64 `json:"peak_drive_power_kw,omitempty"`
	PeakRegenPowerKW        *float64 `json:"peak_regen_power_kw,omitempty"`
	NetEnergyKWh            *float64 `json:"net_energy_kwh,omitempty"`
	AvgConsumptionWhPerKM   float64  `json:"avg_consumption_wh_per_km"`
	BestEfficiencyWhPerKM   *float64 `json:"best_efficiency_wh_per_km,omitempty"`
	WorstEfficiencyWhPerKM  *float64 `json:"worst_efficiency_wh_per_km,omitempty"`
}

// @name V2ChargingSummary
type V2ChargingSummary struct {
	SessionCount              int64    `json:"session_count"`
	EnergyAddedKWh            float64  `json:"energy_added_kwh"`
	EnergyUsedKWh             float64  `json:"energy_used_kwh"`
	DurationMin               float64  `json:"duration_min"`
	AvgDurationMin            *float64 `json:"avg_duration_min,omitempty"`
	LongestSessionDurationMin *float64 `json:"longest_session_duration_min,omitempty"`
	AvgEnergyAddedKWh         *float64 `json:"avg_energy_added_kwh,omitempty"`
	LargestSessionKWh         *float64 `json:"largest_session_kwh,omitempty"`
	AvgPowerKW                *float64 `json:"avg_power_kw,omitempty"`
	MaxPowerKW                *float64 `json:"max_power_kw,omitempty"`
	ChargingEfficiencyPercent *float64 `json:"charging_efficiency_percent,omitempty"`
	Cost                      float64  `json:"cost"`
	AvgCost                   *float64 `json:"avg_cost,omitempty"`
	MaxCost                   *float64 `json:"max_cost,omitempty"`
}

// @name V2VehicleSummary
type V2VehicleSummary struct {
	OdometerKM                    *float64 `json:"odometer_km,omitempty"`
	RatedEfficiencyKWhPer100KM    *float64 `json:"rated_efficiency_kwh_per_100km,omitempty"`
	TrackedConsumptionKWhPer100KM *float64 `json:"tracked_consumption_kwh_per_100km,omitempty"`
	TrackedWallKWhPer100KM        *float64 `json:"tracked_wall_kwh_per_100km,omitempty"`
	ChargingEfficiencyPercent     *float64 `json:"charging_efficiency_percent,omitempty"`
	OdometerCoveragePercent       *float64 `json:"odometer_coverage_percent,omitempty"`
	TrackedDistanceKM             float64  `json:"tracked_distance_km"`
	TrackedDrives                 int64    `json:"tracked_drives"`
	TrackedCharges                int64    `json:"tracked_charges"`
}

// @name V2ParkingSummary
type V2ParkingSummary struct {
	ParkedDurationMin   float64 `json:"parked_duration_min"`
	AsleepDurationMin   float64 `json:"asleep_duration_min"`
	OnlineDurationMin   float64 `json:"online_duration_min"`
	OfflineDurationMin  float64 `json:"offline_duration_min"`
	VampireDrainPercent float64 `json:"vampire_drain_percent"`
}

// @name V2BatterySummary
type V2BatterySummary struct {
	LatestBatteryLevelPercent *int64   `json:"latest_battery_level_percent,omitempty"`
	LatestRatedRangeKM        *float64 `json:"latest_rated_range_km,omitempty"`
	LatestIdealRangeKM        *float64 `json:"latest_ideal_range_km,omitempty"`
}

// @name V2UpdateSummary
type V2UpdateSummary struct {
	UpdateCount   int64   `json:"update_count"`
	LatestVersion *string `json:"latest_version,omitempty"`
}

// @name V2CostSummary
type V2CostSummary struct {
	ChargingCost float64  `json:"charging_cost"`
	CostPerKM    *float64 `json:"cost_per_km,omitempty"`
	CostPer100KM *float64 `json:"cost_per_100km,omitempty"`
}
