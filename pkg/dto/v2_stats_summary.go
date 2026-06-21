package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2SummaryBucket is one bucket of period-aggregated stats.
type V2SummaryBucket struct {
	BucketStart                     nullable.String `json:"bucket_start" swaggertype:"string" example:"2024-01-01T00:00:00+01:00"`
	BucketEnd                       nullable.String `json:"bucket_end" swaggertype:"string" example:"2024-02-01T00:00:00+01:00"`
	DrivesCount                     int             `json:"drives_count" example:"42"`
	DrivesDistance                  float64         `json:"drives_distance" example:"1850.5"`
	DrivesDurationMin               int             `json:"drives_duration_min" example:"2200"`
	DrivesEnergyConsumedKWh         float64         `json:"drives_energy_consumed_kwh" example:"380.5"`
	DrivesAvgConsumption            float64         `json:"drives_avg_consumption" example:"205.6"`
	DrivesLongestDistance           float64         `json:"drives_longest_distance" example:"320.4"`
	DrivesLongestDurationMin        int             `json:"drives_longest_duration_min" example:"210"`
	DrivesMaxSpeed                  int             `json:"drives_max_speed" example:"180"`
	DrivesBestConsumption           float64         `json:"drives_best_consumption" example:"120.0"`
	DrivesWorstConsumption          float64         `json:"drives_worst_consumption" example:"285.4"`
	DrivesPeakDrivePowerKW          int             `json:"drives_peak_drive_power_kw" example:"380"`
	DrivesPeakRegenPowerKW          int             `json:"drives_peak_regen_power_kw" example:"60"`
	DrivesLongestDistanceStartDate  nullable.String `json:"drives_longest_distance_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesLongestDistanceEndDate    nullable.String `json:"drives_longest_distance_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesLongestDurationStartDate  nullable.String `json:"drives_longest_duration_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesLongestDurationEndDate    nullable.String `json:"drives_longest_duration_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesMaxSpeedStartDate         nullable.String `json:"drives_max_speed_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesMaxSpeedEndDate           nullable.String `json:"drives_max_speed_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesBestConsumptionStartDate  nullable.String `json:"drives_best_consumption_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesBestConsumptionEndDate    nullable.String `json:"drives_best_consumption_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesWorstConsumptionStartDate nullable.String `json:"drives_worst_consumption_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesWorstConsumptionEndDate   nullable.String `json:"drives_worst_consumption_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesPeakDrivePowerStartDate   nullable.String `json:"drives_peak_drive_power_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesPeakDrivePowerEndDate     nullable.String `json:"drives_peak_drive_power_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	DrivesPeakRegenPowerStartDate   nullable.String `json:"drives_peak_regen_power_start_date" swaggertype:"string" example:"2024-01-10T08:00:00+01:00"`
	DrivesPeakRegenPowerEndDate     nullable.String `json:"drives_peak_regen_power_end_date" swaggertype:"string" example:"2024-01-10T12:00:00+01:00"`
	ChargesCount                    int             `json:"charges_count" example:"15"`
	ChargesEnergyAddedKWh           float64         `json:"charges_energy_added_kwh" example:"425.7"`
	ChargesEnergyUsedKWh            float64         `json:"charges_energy_used_kwh" example:"445.0"`
	ChargesDurationMin              int             `json:"charges_duration_min" example:"640"`
	ChargesCost                     float64         `json:"charges_cost" example:"125.40"`
	FastChargeRatio                 float64         `json:"fast_charge_ratio" example:"0.32"`
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
	ChargesAvgPowerACKW             float64         `json:"charges_avg_power_ac_kw" example:"7.2"`
	ChargesAvgPowerDCKW             float64         `json:"charges_avg_power_dc_kw" example:"120.5"`
	ParkingsTotalDurationMin        int             `json:"parkings_total_duration_min" example:"42000"`
	VampireDrainKWh                 float64         `json:"vampire_drain_kwh" example:"5.2"`
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
