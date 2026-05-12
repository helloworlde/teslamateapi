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
	Metrics  string `form:"metrics" json:"metrics,omitempty" example:"distance,duration"`
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
	Speed       string `json:"speed" example:"km/h"`
	Duration    string `json:"duration" example:"seconds"`
	Elevation   string `json:"elevation" example:"m"`
	Consumption string `json:"consumption" example:"Wh/km"`
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

// V2DrivingSummary is the canonical driving aggregate; consumed by both
// /analytics/summary and /analytics/driving.
//
// @name V2DrivingSummary
type V2DrivingSummary struct {
	DriveCount              int64    `json:"drive_count"`
	Distance                float64  `json:"distance"`
	Duration                float64  `json:"duration"`
	AvgDistance             *float64 `json:"avg_distance,omitempty"`
	AvgDuration             *float64 `json:"avg_duration,omitempty"`
	LongestDriveDuration    *float64 `json:"longest_drive_duration,omitempty"`
	MaxSpeed                *float64 `json:"max_speed,omitempty"`
	AvgSpeed                *float64 `json:"avg_speed,omitempty"`
	PeakDrivePower          *float64 `json:"peak_drive_power,omitempty"`
	PeakRegenPower          *float64 `json:"peak_regen_power,omitempty"`
	EstimatedEnergyConsumed *float64 `json:"estimated_energy_consumed,omitempty"`
	EstimatedEnergyRegen    *float64 `json:"estimated_energy_regen,omitempty"`
	NetEnergy               *float64 `json:"net_energy,omitempty"`
	AvgConsumption          *float64 `json:"avg_consumption,omitempty"`
	BestConsumption         *float64 `json:"best_consumption,omitempty"`
	WorstConsumption        *float64 `json:"worst_consumption,omitempty"`
	RangeLoss               float64  `json:"range_loss"`
	BatteryLevelUsed        *float64 `json:"battery_level_used,omitempty"`
	AvgOutsideTemp          *float64 `json:"avg_outside_temp,omitempty"`
}

// V2ChargingSummary is the canonical charging aggregate; consumed by both
// /analytics/summary and /analytics/charging.
//
// @name V2ChargingSummary
type V2ChargingSummary struct {
	SessionCount           int64    `json:"session_count"`
	EnergyAdded            float64  `json:"energy_added"`
	EnergyUsed             float64  `json:"energy_used"`
	Duration               float64  `json:"duration"`
	AvgDuration            *float64 `json:"avg_duration,omitempty"`
	LongestSessionDuration *float64 `json:"longest_session_duration,omitempty"`
	AvgEnergyAdded         *float64 `json:"avg_energy_added,omitempty"`
	LargestSession         *float64 `json:"largest_session,omitempty"`
	AvgPower               *float64 `json:"avg_power,omitempty"`
	MaxPower               *float64 `json:"max_power,omitempty"`
	ChargingEfficiency     *float64 `json:"charging_efficiency,omitempty"`
	Cost                   float64  `json:"cost"`
	AvgCost                *float64 `json:"avg_cost,omitempty"`
	MaxCost                *float64 `json:"max_cost,omitempty"`
	AvgCostPerEnergy       *float64 `json:"avg_cost_per_energy,omitempty"`
	StartBatteryAvg        *float64 `json:"start_battery_avg,omitempty"`
	EndBatteryAvg          *float64 `json:"end_battery_avg,omitempty"`
	ACSessionCount         int64    `json:"ac_session_count"`
	DCSessionCount         int64    `json:"dc_session_count"`
	ACEnergy               float64  `json:"ac_energy"`
	DCEnergy               float64  `json:"dc_energy"`
}

// @name V2VehicleSummary
type V2VehicleSummary struct {
	Odometer        *float64 `json:"odometer,omitempty"`
	TrackedDistance float64  `json:"tracked_distance"`
	TrackedDrives   int64    `json:"tracked_drives"`
	TrackedCharges  int64    `json:"tracked_charges"`
}

// V2ParkingSummary is the canonical parking aggregate; consumed by both
// /analytics/summary and /analytics/parking.
//
// @name V2ParkingSummary
type V2ParkingSummary struct {
	ParkingSessionCount    int64    `json:"parking_session_count"`
	ParkedDuration         float64  `json:"parked_duration"`
	AvgParkedDuration      *float64 `json:"avg_parked_duration,omitempty"`
	AsleepDuration         float64  `json:"asleep_duration"`
	OnlineDuration         float64  `json:"online_duration"`
	OfflineDuration        float64  `json:"offline_duration"`
	VampireDrainPercent    *float64 `json:"vampire_drain_percent,omitempty"`
	EstimatedVampireDrain  *float64 `json:"estimated_vampire_drain,omitempty"`
	AvgDrainPercentPerDay  *float64 `json:"avg_drain_percent_per_day,omitempty"`
	StateTransitionCount   int64    `json:"state_transition_count"`
}

// @name V2BatterySummary
type V2BatterySummary struct {
	LatestLevel               *int64          `json:"latest_level,omitempty"`
	LatestRatedRange          *float64        `json:"latest_rated_range,omitempty"`
	LatestIdealRange          *float64        `json:"latest_ideal_range,omitempty"`
	RangeAtFullCharge         *V2BatteryRange `json:"range_at_full_charge,omitempty"`
	BaselineRangeAtFullCharge *V2BatteryRange `json:"baseline_range_at_full_charge,omitempty"`
	EstimatedRangeDegradation *float64        `json:"estimated_range_degradation,omitempty"`
}

// @name V2BatteryRange
type V2BatteryRange struct {
	Rated *float64 `json:"rated,omitempty"`
	Ideal *float64 `json:"ideal,omitempty"`
}

// @name V2UpdateSummary
type V2UpdateSummary struct {
	UpdateCount   int64   `json:"update_count"`
	LatestVersion *string `json:"latest_version,omitempty"`
}

// @name V2CostSummary
type V2CostSummary struct {
	ChargingCost    float64  `json:"charging_cost"`
	CostPerDistance *float64 `json:"cost_per_distance,omitempty"`
}
