package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2DrivePowerDrive identifies the single drive being analysed.
type V2DrivePowerDrive struct {
	DriveID         int     `json:"drive_id" example:"1234"`
	StartDate       string  `json:"start_date" example:"2024-01-01T08:00:00+01:00"`
	EndDate         string  `json:"end_date" example:"2024-01-01T08:30:00+01:00"`
	Distance        float64 `json:"distance" example:"18.4"`
	DurationMin     int     `json:"duration_min" example:"30"`
	DurationSeconds float64 `json:"duration_seconds" example:"1800"`
}

// V2DrivePowerMetrics contains observed power-integral statistics for one
// drive. Energy values are estimates from positions.power samples, not BMS
// metered battery accounting.
type V2DrivePowerMetrics struct {
	ObservedBatteryOutputEnergyKWh float64          `json:"observed_battery_output_energy_kwh" example:"3.42"`
	ObservedRegenRecoveredKWh      float64          `json:"observed_regen_recovered_kwh" example:"0.58"`
	ObservedNetBatteryEnergyKWh    float64          `json:"observed_net_battery_energy_kwh" example:"2.84"`
	RegenShareOfOutputPct          nullable.Float64 `json:"regen_share_of_output_pct" swaggertype:"number" example:"16.96"`
	RegenShareOfPowerActivityPct   nullable.Float64 `json:"regen_share_of_power_activity_pct" swaggertype:"number" example:"14.50"`
	PeakOutputPowerKW              int              `json:"peak_output_power_kw" example:"220"`
	PeakRegenPowerKW               int              `json:"peak_regen_power_kw" example:"64"`
	AvgOutputPowerKW               nullable.Float64 `json:"avg_output_power_kw" swaggertype:"number" example:"38.2"`
	AvgRegenPowerKW                nullable.Float64 `json:"avg_regen_power_kw" swaggertype:"number" example:"18.4"`
	OutputDurationSeconds          float64          `json:"output_duration_seconds" example:"322.4"`
	RegenDurationSeconds           float64          `json:"regen_duration_seconds" example:"113.5"`
}

// V2DrivePowerDataQuality explains how much of the drive had usable high-rate
// power samples. Confidence is based on valid_sample_seconds / drive duration.
type V2DrivePowerDataQuality struct {
	Confidence              string  `json:"confidence" example:"high" enums:"high,medium,low,unavailable"`
	SampleCoveragePct       float64 `json:"sample_coverage_pct" example:"96.4"`
	PowerSampleCount        int     `json:"power_sample_count" example:"1790"`
	ValidIntervalCount      int     `json:"valid_interval_count" example:"1725"`
	IgnoredGapCount         int     `json:"ignored_gap_count" example:"8"`
	ValidSampleSeconds      float64 `json:"valid_sample_seconds" example:"1735.2"`
	DriveDurationSeconds    float64 `json:"drive_duration_seconds" example:"1800"`
	MaxValidIntervalSeconds float64 `json:"max_valid_interval_seconds" example:"1.5"`
	HasPowerSamples         bool    `json:"has_power_samples" example:"true"`
}

// V2DrivePowerData is the `data` field for the drive power endpoint.
type V2DrivePowerData struct {
	Car         Car                     `json:"car"`
	Drive       V2DrivePowerDrive       `json:"drive"`
	Metrics     V2DrivePowerMetrics     `json:"metrics"`
	DataQuality V2DrivePowerDataQuality `json:"data_quality"`
	Units       TeslaMateUnits          `json:"units"`
}

// V2DrivePowerResponse is the envelope for
// /api/v2/cars/{CarID}/drives/{DriveID}/power.
type V2DrivePowerResponse struct {
	Data V2DrivePowerData `json:"data"`
}
