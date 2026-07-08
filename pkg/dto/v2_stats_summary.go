package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2SummaryBucket is one bucket of period-aggregated stats.
type V2SummaryBucket struct {
	BucketStart    nullable.String `json:"bucket_start" swaggertype:"string" example:"2024-01-01T00:00:00+01:00"`
	BucketEnd      nullable.String `json:"bucket_end" swaggertype:"string" example:"2024-02-01T00:00:00+01:00"`
	DrivesCount    int             `json:"drives_count" example:"42"`
	DrivesDistance float64         `json:"drives_distance" example:"1850.5"`
	// DrivesDistanceLifetimeCumulative is lifetime drive distance through this
	// bucket, including any pre-window baseline.
	DrivesDistanceLifetimeCumulative float64 `json:"drives_distance_lifetime_cumulative" example:"24500.7"`
	DrivesDurationMin                int     `json:"drives_duration_min" example:"2200"`
	DrivesEnergyConsumedKWh          float64 `json:"drives_energy_consumed_kwh" example:"380.5"`
	// DrivesEstimatedUsageCost = SOC-derived drive energy × this bucket's
	// average charging price (charges_cost / charges_energy_added_kwh). Null
	// when the bucket has no battery-side charge energy or when any
	// positive-distance drive lacks a calibratable SOC basis.
	DrivesEstimatedUsageCost nullable.Float64 `json:"drives_estimated_usage_cost" swaggertype:"number" example:"98.93"`
	// DrivesCostPerDistance = drives_estimated_usage_cost / drives_distance.
	// Null when drive cost or distance is unavailable, including partial drive
	// SOC data.
	DrivesCostPerDistance           nullable.Float64 `json:"drives_cost_per_distance" swaggertype:"number" example:"0.053"`
	DrivesAvgConsumption            float64          `json:"drives_avg_consumption" example:"205.6"`
	DrivesLongestDistance           float64          `json:"drives_longest_distance" example:"320.4"`
	DrivesLongestDurationMin        int              `json:"drives_longest_duration_min" example:"210"`
	DrivesMaxSpeed                  int              `json:"drives_max_speed" example:"180"`
	DrivesBestConsumption           float64          `json:"drives_best_consumption" example:"120.0"`
	DrivesWorstConsumption          float64          `json:"drives_worst_consumption" example:"285.4"`
	DrivesPeakDrivePowerKW          int              `json:"drives_peak_drive_power_kw" example:"380"`
	DrivesPeakRegenPowerKW          int              `json:"drives_peak_regen_power_kw" example:"60"`
	DrivesLongestDistanceStartDate  nullable.String  `json:"drives_longest_distance_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesLongestDistanceEndDate    nullable.String  `json:"drives_longest_distance_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesLongestDurationStartDate  nullable.String  `json:"drives_longest_duration_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesLongestDurationEndDate    nullable.String  `json:"drives_longest_duration_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesMaxSpeedStartDate         nullable.String  `json:"drives_max_speed_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesMaxSpeedEndDate           nullable.String  `json:"drives_max_speed_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesBestConsumptionStartDate  nullable.String  `json:"drives_best_consumption_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesBestConsumptionEndDate    nullable.String  `json:"drives_best_consumption_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesWorstConsumptionStartDate nullable.String  `json:"drives_worst_consumption_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesWorstConsumptionEndDate   nullable.String  `json:"drives_worst_consumption_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesPeakDrivePowerStartDate   nullable.String  `json:"drives_peak_drive_power_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesPeakDrivePowerEndDate     nullable.String  `json:"drives_peak_drive_power_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesPeakRegenPowerStartDate   nullable.String  `json:"drives_peak_regen_power_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesPeakRegenPowerEndDate     nullable.String  `json:"drives_peak_regen_power_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	ChargesCount                    int              `json:"charges_count" example:"15"`
	ChargesEnergyAddedKWh           float64          `json:"charges_energy_added_kwh" example:"425.7"`
	// ChargesEnergyAddedKWhLifetimeCumulative is lifetime battery-side charge
	// energy added through this bucket, including any pre-window baseline.
	ChargesEnergyAddedKWhLifetimeCumulative float64 `json:"charges_energy_added_kwh_lifetime_cumulative" example:"5820.4"`
	ChargesEnergyUsedKWh                    float64 `json:"charges_energy_used_kwh" example:"445.0"`
	// ChargesEnergyUsedKWhLifetimeCumulative is lifetime wall-side charge
	// energy used through this bucket, including any pre-window baseline.
	ChargesEnergyUsedKWhLifetimeCumulative float64 `json:"charges_energy_used_kwh_lifetime_cumulative" example:"6104.8"`
	ChargesDurationMin                     int     `json:"charges_duration_min" example:"640"`
	ChargesCost                            float64 `json:"charges_cost" example:"125.40"`
	// ChargesCostLifetimeCumulative is lifetime charging cost through this
	// bucket, including any pre-window baseline.
	ChargesCostLifetimeCumulative   float64         `json:"charges_cost_lifetime_cumulative" example:"1678.74"`
	ChargesACCount                  int             `json:"charges_ac_count" example:"10"`
	ChargesDCCount                  int             `json:"charges_dc_count" example:"5"`
	ChargesACEnergyAddedKWh         float64         `json:"charges_ac_energy_added_kwh" example:"221.4"`
	ChargesDCEnergyAddedKWh         float64         `json:"charges_dc_energy_added_kwh" example:"204.3"`
	ChargesACEnergyUsedKWh          float64         `json:"charges_ac_energy_used_kwh" example:"232.0"`
	ChargesDCEnergyUsedKWh          float64         `json:"charges_dc_energy_used_kwh" example:"213.0"`
	ChargesACDurationMin            int             `json:"charges_ac_duration_min" example:"500"`
	ChargesDCDurationMin            int             `json:"charges_dc_duration_min" example:"140"`
	ChargesACCost                   float64         `json:"charges_ac_cost" example:"40.20"`
	ChargesDCCost                   float64         `json:"charges_dc_cost" example:"85.20"`
	ChargesACAvgDurationMin         float64         `json:"charges_ac_avg_duration_min" example:"50.0"`
	ChargesDCAvgDurationMin         float64         `json:"charges_dc_avg_duration_min" example:"28.0"`
	ChargesACAvgEnergyPerSessionKWh float64         `json:"charges_ac_avg_energy_per_session_kwh" example:"22.1"`
	ChargesDCAvgEnergyPerSessionKWh float64         `json:"charges_dc_avg_energy_per_session_kwh" example:"40.9"`
	ChargesACAvgSessionCost         float64         `json:"charges_ac_avg_session_cost" example:"4.02"`
	ChargesDCAvgSessionCost         float64         `json:"charges_dc_avg_session_cost" example:"17.04"`
	ChargesACAvgCostPerKWh          float64         `json:"charges_ac_avg_cost_per_kwh" example:"0.18"`
	ChargesDCAvgCostPerKWh          float64         `json:"charges_dc_avg_cost_per_kwh" example:"0.42"`
	ChargesDCRatio                  float64         `json:"charges_dc_ratio" example:"0.32"`
	ChargesLongestSessionMin        int             `json:"charges_longest_session_duration_min" example:"180"`
	ChargesLargestSessionKWh        float64         `json:"charges_largest_session_kwh" example:"78.4"`
	ChargesMaxSessionCost           float64         `json:"charges_max_session_cost" example:"42.5"`
	ChargesMaxPowerKW               int             `json:"charges_max_power_kw" example:"250"`
	ChargesLongestSessionStartDate  nullable.String `json:"charges_longest_session_start_date" swaggertype:"string" example:"2024-01-10T19:30:00+01:00"`
	ChargesLongestSessionEndDate    nullable.String `json:"charges_longest_session_end_date" swaggertype:"string" example:"2024-01-10T22:00:00+01:00"`
	ChargesLargestSessionStartDate  nullable.String `json:"charges_largest_session_start_date" swaggertype:"string" example:"2024-01-10T19:30:00+01:00"`
	ChargesLargestSessionEndDate    nullable.String `json:"charges_largest_session_end_date" swaggertype:"string" example:"2024-01-10T22:00:00+01:00"`
	ChargesMaxSessionCostStartDate  nullable.String `json:"charges_max_session_cost_start_date" swaggertype:"string" example:"2024-01-10T19:30:00+01:00"`
	ChargesMaxSessionCostEndDate    nullable.String `json:"charges_max_session_cost_end_date" swaggertype:"string" example:"2024-01-10T22:00:00+01:00"`
	ChargesMaxPowerDate             nullable.String `json:"charges_max_power_date" swaggertype:"string" example:"2024-01-10T20:15:00+01:00"`
	ChargesACAvgPowerKW             float64         `json:"charges_ac_avg_power_kw" example:"7.2"`
	ChargesDCAvgPowerKW             float64         `json:"charges_dc_avg_power_kw" example:"120.5"`
	ParkingsTotalDurationMin        int             `json:"parkings_total_duration_min" example:"42000"`
	VampireDrainKWh                 float64         `json:"vampire_drain_kwh" example:"5.2"`
	// VampireDrainKWhLifetimeCumulative is lifetime vampire-drain energy through
	// this bucket, including any pre-window parking baseline.
	VampireDrainKWhLifetimeCumulative float64 `json:"vampire_drain_kwh_lifetime_cumulative" example:"125.0"`
}

// V2SummaryData is the `data` field of V2SummaryResponse.
type V2SummaryData struct {
	Car     Car               `json:"car"`
	Period  string            `json:"period" example:"month" enums:"day,week,month,year"`
	Buckets []V2SummaryBucket `json:"buckets"`
	Units   TeslaMateUnits    `json:"units"`
}

// V2SummaryResponse is the envelope for /api/v2/cars/{CarID}/stats/summary.
type V2SummaryResponse struct {
	Data V2SummaryData `json:"data"`
}
