package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2SummaryBucket is one bucket of period-aggregated stats.
type V2SummaryBucket struct {
	BucketStart              nullable.String `json:"bucket_start" swaggertype:"string" example:"2024-01-01T00:00:00+01:00"`
	BucketEnd                nullable.String `json:"bucket_end" swaggertype:"string" example:"2024-02-01T00:00:00+01:00"`
	DrivesCount              int             `json:"drives_count" example:"42"`
	DrivesDistance           float64         `json:"drives_distance" example:"1850.5"`
	DrivesDurationMin        int             `json:"drives_duration_min" example:"2200"`
	DrivesEnergyConsumedKWh  float64         `json:"drives_energy_consumed_kwh" example:"380.5"`
	DrivesAvgConsumption     float64         `json:"drives_avg_consumption" example:"205.6"`
	ChargesCount             int             `json:"charges_count" example:"15"`
	ChargesEnergyAddedKWh    float64         `json:"charges_energy_added_kwh" example:"425.7"`
	ChargesCost              float64         `json:"charges_cost" example:"125.40"`
	FastChargeRatio          float64         `json:"fast_charge_ratio" example:"0.32"`
	ParkingsTotalDurationMin int             `json:"parkings_total_duration_min" example:"42000"`
	VampireDrainKWh          float64         `json:"vampire_drain_kwh" example:"5.2"`
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
