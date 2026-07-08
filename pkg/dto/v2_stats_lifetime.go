package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2CarMeta carries the static car metadata surfaced by the lifetime endpoint
// so callers don't have to combine /api/v1/cars + lifetime stats.
type V2CarMeta struct {
	Vin           string           `json:"vin" example:"5YJSA1E26KF000000"`
	Model         nullable.String  `json:"model" swaggertype:"string" example:"S"`
	TrimBadging   nullable.String  `json:"trim_badging" swaggertype:"string" example:"P100D"`
	ExteriorColor nullable.String  `json:"exterior_color" swaggertype:"string" example:"DeepBlue"`
	WheelType     nullable.String  `json:"wheel_type" swaggertype:"string" example:"Pinwheel18"`
	SpoilerType   nullable.String  `json:"spoiler_type" swaggertype:"string" example:"None"`
	Efficiency    nullable.Float64 `json:"efficiency" swaggertype:"number" example:"0.184"`
	InsertedAt    nullable.String  `json:"inserted_at" swaggertype:"string" example:"2020-01-01T00:00:00+01:00"`
}

// V2DrivesAgg is the drives section of the lifetime stats response.
type V2DrivesAgg struct {
	Count                  int     `json:"count" example:"1893"`
	TotalDistance          float64 `json:"total_distance" example:"42500.5"`
	TotalDurationMin       int     `json:"total_duration_min" example:"38000"`
	TotalEnergyConsumedKWh float64 `json:"total_energy_consumed_kwh" example:"7825.0"`
	// EstimatedUsageCost = SOC-derived drive energy × average charging price
	// (SUM(cost) / SUM(charge_energy_added)). Currency matches
	// charging_processes.cost. Null when drive energy or charge price is not
	// fully calibratable for every positive-distance drive.
	EstimatedUsageCost nullable.Float64 `json:"estimated_usage_cost" swaggertype:"number" example:"2034.50"`
	AvgConsumption     float64          `json:"avg_consumption" example:"184.0"`
	BestConsumption    float64          `json:"best_consumption" example:"120.0"`
	WorstConsumption   float64          `json:"worst_consumption" example:"285.4"`
	// RangeAchievementPct = Σ distance / Σ rated-range drop × 100 over all
	// drives (objective). 100 = rated and real distance match; >100 beats
	// rated, <100 falls short. Unit-independent (a ratio of distances).
	RangeAchievementPct       float64          `json:"range_achievement_pct" example:"94.2"`
	TrackingRatePct           nullable.Float64 `json:"tracking_rate_pct" swaggertype:"number" example:"98.7"`
	LongestDistance           float64          `json:"longest_distance" example:"800.0"`
	ShortestDistance          float64          `json:"shortest_distance" example:"0.5"`
	LongestDurationMin        int              `json:"longest_duration_min" example:"420"`
	MaxSpeed                  int              `json:"max_speed" example:"180"`
	PeakDrivePowerKW          int              `json:"peak_drive_power_kw" example:"380"`
	LongestDistanceStartDate  nullable.String  `json:"longest_distance_start_date" swaggertype:"string" example:"2026-05-30T18:00:00+01:00"`
	LongestDistanceEndDate    nullable.String  `json:"longest_distance_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	LongestDurationStartDate  nullable.String  `json:"longest_duration_start_date" swaggertype:"string" example:"2026-05-30T18:00:00+01:00"`
	LongestDurationEndDate    nullable.String  `json:"longest_duration_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	MaxSpeedStartDate         nullable.String  `json:"max_speed_start_date" swaggertype:"string" example:"2026-05-30T18:00:00+01:00"`
	MaxSpeedEndDate           nullable.String  `json:"max_speed_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	PeakDrivePowerStartDate   nullable.String  `json:"peak_drive_power_start_date" swaggertype:"string" example:"2026-05-30T18:00:00+01:00"`
	PeakDrivePowerEndDate     nullable.String  `json:"peak_drive_power_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	MaxRegenPowerStartDate    nullable.String  `json:"max_regen_power_start_date" swaggertype:"string" example:"2026-05-30T18:00:00+01:00"`
	MaxRegenPowerEndDate      nullable.String  `json:"max_regen_power_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	BestConsumptionStartDate  nullable.String  `json:"best_consumption_start_date" swaggertype:"string" example:"2026-05-30T18:00:00+01:00"`
	BestConsumptionEndDate    nullable.String  `json:"best_consumption_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	WorstConsumptionStartDate nullable.String  `json:"worst_consumption_start_date" swaggertype:"string" example:"2026-05-30T18:00:00+01:00"`
	WorstConsumptionEndDate   nullable.String  `json:"worst_consumption_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	AvgSpeed                  float64          `json:"avg_speed" example:"58.5"`
	AvgDistancePerDrive       float64          `json:"avg_distance_per_drive" example:"22.4"`
	AvgDurationPerDrive       float64          `json:"avg_duration_per_drive_min" example:"20.1"`
	MaxRegenPower             int              `json:"max_regen_power_kw" example:"60"`
	AvgOutsideTemp            nullable.Float64 `json:"avg_outside_temp" swaggertype:"number" example:"14.5"`
	AvgInsideTemp             nullable.Float64 `json:"avg_inside_temp" swaggertype:"number" example:"21.0"`
	ActiveDays                int              `json:"active_days" example:"612"`
	LastDriveDate             nullable.String  `json:"last_drive_date" swaggertype:"string" example:"2026-05-30T18:42:00+01:00"`
	CurrentOdometer           float64          `json:"current_odometer" example:"42500.0"`
}

// V2ChargesAgg is the charges section of the lifetime stats response.
type V2ChargesAgg struct {
	Count               int     `json:"count" example:"427"`
	TotalEnergyAddedKWh float64 `json:"total_energy_added_kwh" example:"16500.0"`
	TotalEnergyUsedKWh  float64 `json:"total_energy_used_kwh" example:"17200.0"`
	TotalCost           float64 `json:"total_cost" example:"4321.50"`
	// AvgCostPerKWh = SUM(cost) / SUM(charge_energy_added) over all charges
	// (battery-side rate). Free/zero-cost charges still contribute energy to the
	// denominator, so this is biased low versus the rate paid for billed energy.
	AvgCostPerKWh   float64 `json:"avg_cost_per_kwh" example:"0.26"`
	ACAvgCostPerKWh float64 `json:"ac_avg_cost_per_kwh" example:"0.18"`
	DCAvgCostPerKWh float64 `json:"dc_avg_cost_per_kwh" example:"0.42"`
	// CostPerKm = estimated drive usage cost / total drive distance. Estimated
	// drive usage cost is SOC-derived drive energy × average charging price
	// (SUM(cost) / SUM(charge_energy_added)). Converted to per-mile when
	// unit_of_length is mi. Currency matches charging_processes.cost. Null when
	// estimated drive usage cost or distance is unavailable, including partial
	// drive SOC data.
	CostPerKm             nullable.Float64 `json:"cost_per_distance" swaggertype:"number" example:"0.05"`
	AvgEnergyPerSession   float64          `json:"avg_energy_per_session_kwh" example:"38.6"`
	ACAvgEnergyPerSession float64          `json:"ac_avg_energy_per_session_kwh" example:"22.1"`
	DCAvgEnergyPerSession float64          `json:"dc_avg_energy_per_session_kwh" example:"48.2"`
	TotalDurationMin      int              `json:"total_duration_min" example:"18000"`
	AvgDurationMin        float64          `json:"avg_duration_min" example:"42.3"`
	ACAvgDurationMin      float64          `json:"ac_avg_duration_min" example:"180.4"`
	DCAvgDurationMin      float64          `json:"dc_avg_duration_min" example:"28.6"`
	// DCChargeCount counts sessions with any DC fast charging
	// (fast_charger_present), regardless of network or brand.
	DCChargeCount int `json:"dc_charge_count" example:"73"`
	// SuperchargerCount counts Tesla Supercharger sessions
	// (fast_charger_present AND fast_charger_brand = 'Tesla'); a subset of
	// dc_charge_count.
	SuperchargerCount int `json:"supercharger_count" example:"70"`
	// FreeSuperchargingCount counts Supercharger sessions covered by free
	// supercharging (car_settings.free_supercharging).
	FreeSuperchargingCount      int             `json:"free_supercharging_count" example:"5"`
	DCChargeEnergyAddedKWh      float64         `json:"dc_charge_energy_added_kwh" example:"3200.0"`
	DCChargeEnergyUsedKWh       float64         `json:"dc_charge_energy_used_kwh" example:"3350.0"`
	ACChargeCount               int             `json:"ac_charge_count" example:"354"`
	ACChargeEnergyAddedKWh      float64         `json:"ac_charge_energy_added_kwh" example:"13300.0"`
	ACChargeEnergyUsedKWh       float64         `json:"ac_charge_energy_used_kwh" example:"13850.0"`
	GeofencedChargeEnergyKWh    float64         `json:"geofenced_charge_energy_kwh" example:"9800.0"`
	NonGeofencedChargeEnergyKWh float64         `json:"non_geofenced_charge_energy_kwh" example:"6700.0"`
	FreeSuperchargingAddedKWh   float64         `json:"free_supercharging_added_kwh" example:"125.0"`
	FreeSuperchargingUsedKWh    float64         `json:"free_supercharging_used_kwh" example:"131.0"`
	PeakPowerMaxKW              int             `json:"peak_power_max_kw" example:"250"`
	PeakVoltageMax              int             `json:"peak_voltage_max_v" example:"480"`
	ShortestSessionDurationMin  int             `json:"shortest_session_duration_min" example:"12"`
	LongestSessionDurationMin   int             `json:"longest_session_duration_min" example:"180"`
	LargestSessionEnergyKWh     float64         `json:"largest_session_energy_kwh" example:"78.4"`
	MaxSessionCost              float64         `json:"max_session_cost" example:"220.5"`
	PeakPowerDate               nullable.String `json:"peak_power_date" swaggertype:"string" example:"2026-05-30T20:15:00+01:00"`
	PeakVoltageDate             nullable.String `json:"peak_voltage_date" swaggertype:"string" example:"2026-05-30T20:15:00+01:00"`
	LongestSessionStartDate     nullable.String `json:"longest_session_start_date" swaggertype:"string" example:"2026-05-30T19:30:00+01:00"`
	LongestSessionEndDate       nullable.String `json:"longest_session_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	LargestSessionStartDate     nullable.String `json:"largest_session_start_date" swaggertype:"string" example:"2026-05-30T19:30:00+01:00"`
	LargestSessionEndDate       nullable.String `json:"largest_session_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	MaxSessionCostStartDate     nullable.String `json:"max_session_cost_start_date" swaggertype:"string" example:"2026-05-30T19:30:00+01:00"`
	MaxSessionCostEndDate       nullable.String `json:"max_session_cost_end_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
	AvgSessionCost              float64         `json:"avg_session_cost" example:"12.3"`
	ACAvgSessionCost            float64         `json:"ac_avg_session_cost" example:"4.5"`
	DCAvgSessionCost            float64         `json:"dc_avg_session_cost" example:"19.8"`
	ACAvgPowerKW                float64         `json:"ac_avg_power_kw" example:"7.2"`
	DCAvgPowerKW                float64         `json:"dc_avg_power_kw" example:"120.5"`
	ACMaxPowerKW                int             `json:"ac_max_power_kw" example:"11"`
	DCMaxPowerKW                int             `json:"dc_max_power_kw" example:"250"`
	ACMaxPowerDate              nullable.String `json:"ac_max_power_date" swaggertype:"string" example:"2026-05-30T20:15:00+01:00"`
	DCMaxPowerDate              nullable.String `json:"dc_max_power_date" swaggertype:"string" example:"2026-05-30T20:15:00+01:00"`
	MinStartBatteryLevel        nullable.Int64  `json:"min_start_battery_level" swaggertype:"integer" example:"3"`
	MaxEndBatteryLevel          nullable.Int64  `json:"max_end_battery_level" swaggertype:"integer" example:"100"`
	DistinctChargeLocations     int             `json:"distinct_charge_locations" example:"42"`
	FirstChargeDate             nullable.String `json:"first_charge_date" swaggertype:"string" example:"2020-01-05T19:30:00+01:00"`
	LastChargeDate              nullable.String `json:"last_charge_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
}

// V2ParkingsAgg is the parkings section of the lifetime stats response.
type V2ParkingsAgg struct {
	Count                   int             `json:"count" example:"1893"`
	TotalDurationMin        int             `json:"total_duration_min" example:"500000"`
	AvgDurationMin          float64         `json:"avg_duration_min" example:"264.2"`
	LongestParkingMin       int             `json:"longest_parking_min" example:"43200"`
	LongestParkingStartDate nullable.String `json:"longest_parking_start_date" swaggertype:"string" example:"2026-05-01T18:00:00+01:00"`
	LongestParkingEndDate   nullable.String `json:"longest_parking_end_date" swaggertype:"string" example:"2026-05-30T18:00:00+01:00"`
	TotalEnergyDropKWh      float64         `json:"total_vampire_drain_kwh" example:"125.0"`
}

// V2UpdatesAgg is the firmware-updates section of the lifetime stats response.
type V2UpdatesAgg struct {
	Count                     int             `json:"count" example:"58"`
	FirstVersion              nullable.String `json:"first_version" swaggertype:"string" example:"2020.4.10"`
	LatestVersion             nullable.String `json:"latest_version" swaggertype:"string" example:"2026.20.1"`
	LatestUpdateDate          nullable.String `json:"latest_update_date" swaggertype:"string" example:"2026-05-15T03:00:00+01:00"`
	LongestIntervalDays       nullable.Int64  `json:"longest_interval_days" swaggertype:"integer" example:"180"`
	ShortestIntervalDays      nullable.Int64  `json:"shortest_interval_days" swaggertype:"integer" example:"14"`
	LongestIntervalStartDate  nullable.String `json:"longest_interval_start_date" swaggertype:"string" example:"2025-12-01T03:00:00+01:00"`
	LongestIntervalEndDate    nullable.String `json:"longest_interval_end_date" swaggertype:"string" example:"2026-05-30T03:00:00+01:00"`
	ShortestIntervalStartDate nullable.String `json:"shortest_interval_start_date" swaggertype:"string" example:"2026-05-01T03:00:00+01:00"`
	ShortestIntervalEndDate   nullable.String `json:"shortest_interval_end_date" swaggertype:"string" example:"2026-05-15T03:00:00+01:00"`
}

// V2Lifetime is the `data` field of V2LifetimeResponse.
//
// RecordedDays is distinct days observed across drives + charging_processes
// — different from (now - since) because data may have gaps. AvgDaily /
// AvgMonthly distance are server-computed over *driving* days/months (days
// and months in which the car actually drove), so charge-only days don't
// dilute them; clients can render canonical figures without re-deriving them.
type V2Lifetime struct {
	Car                Car             `json:"car"`
	CarMeta            V2CarMeta       `json:"car_meta"`
	Since              nullable.String `json:"since" swaggertype:"string" example:"2020-01-01T00:00:00+01:00"`
	RecordedDays       int             `json:"recorded_days" example:"612"`
	AvgDailyDistance   float64         `json:"avg_daily_distance" example:"45.6"`
	AvgMonthlyDistance float64         `json:"avg_monthly_distance" example:"1380.0"`
	Drives             V2DrivesAgg     `json:"drives"`
	Charges            V2ChargesAgg    `json:"charges"`
	Parkings           V2ParkingsAgg   `json:"parkings"`
	Updates            V2UpdatesAgg    `json:"updates"`
	Units              TeslaMateUnits  `json:"units"`
}

// V2LifetimeResponse is the envelope for /api/v2/cars/{CarID}/stats/lifetime.
type V2LifetimeResponse struct {
	Data V2Lifetime `json:"data"`
}
