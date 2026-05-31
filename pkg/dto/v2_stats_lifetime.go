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
	Count                  int             `json:"count" example:"1893"`
	TotalDistance          float64         `json:"total_distance" example:"42500.5"`
	TotalDurationMin       int             `json:"total_duration_min" example:"38000"`
	TotalEnergyConsumedKWh float64         `json:"total_energy_consumed_kwh" example:"7825.0"`
	AvgConsumption         float64         `json:"avg_consumption" example:"184.0"`
	BestConsumption        float64         `json:"best_consumption" example:"120.0"`
	LongestDistance        float64         `json:"longest_distance" example:"800.0"`
	ShortestDistance       float64         `json:"shortest_distance" example:"0.5"`
	MaxSpeed               int             `json:"max_speed" example:"180"`
	AvgSpeed               float64         `json:"avg_speed" example:"58.5"`
	AvgDistancePerDrive    float64         `json:"avg_distance_per_drive" example:"22.4"`
	AvgDurationPerDrive    float64         `json:"avg_duration_per_drive_min" example:"20.1"`
	MaxRegenPower          int             `json:"max_regen_power_kw" example:"60"`
	AvgOutsideTemp         nullable.Float64 `json:"avg_outside_temp" swaggertype:"number" example:"14.5"`
	AvgInsideTemp          nullable.Float64 `json:"avg_inside_temp" swaggertype:"number" example:"21.0"`
	ActiveDays             int             `json:"active_days" example:"612"`
	LastDriveDate          nullable.String `json:"last_drive_date" swaggertype:"string" example:"2026-05-30T18:42:00+01:00"`
	CurrentOdometer        float64         `json:"current_odometer" example:"42500.0"`
}

// V2ChargesAgg is the charges section of the lifetime stats response.
type V2ChargesAgg struct {
	Count                   int             `json:"count" example:"427"`
	TotalEnergyAddedKWh     float64         `json:"total_energy_added_kwh" example:"16500.0"`
	TotalEnergyUsedKWh      float64         `json:"total_energy_used_kwh" example:"17200.0"`
	TotalCost               float64         `json:"total_cost" example:"4321.50"`
	AvgCostPerKWh           float64         `json:"avg_cost_per_kwh" example:"0.26"`
	AvgEnergyPerSession     float64         `json:"avg_energy_per_session_kwh" example:"38.6"`
	AvgDurationMin          float64         `json:"avg_duration_min" example:"42.3"`
	FastChargeCount         int             `json:"fast_charge_count" example:"73"`
	FastChargeEnergyKWh     float64         `json:"fast_charge_energy_kwh" example:"3200.0"`
	ACChargeCount           int             `json:"ac_charge_count" example:"354"`
	ACChargeEnergyKWh       float64         `json:"ac_charge_energy_kwh" example:"13300.0"`
	GeofencedChargeEnergyKWh    float64     `json:"geofenced_charge_energy_kwh" example:"9800.0"`
	NonGeofencedChargeEnergyKWh float64     `json:"non_geofenced_charge_energy_kwh" example:"6700.0"`
	FreeSuperchargingKWh    float64         `json:"free_supercharging_kwh" example:"125.0"`
	PeakPowerMaxKW          int             `json:"peak_power_max_kw" example:"250"`
	PeakVoltageMax          int             `json:"peak_voltage_max" example:"480"`
	MinStartBatteryLevel    nullable.Int64  `json:"min_start_battery_level" swaggertype:"integer" example:"3"`
	MaxEndBatteryLevel      nullable.Int64  `json:"max_end_battery_level" swaggertype:"integer" example:"100"`
	DistinctChargeLocations int             `json:"distinct_charge_locations" example:"42"`
	FirstChargeDate         nullable.String `json:"first_charge_date" swaggertype:"string" example:"2020-01-05T19:30:00+01:00"`
	LastChargeDate          nullable.String `json:"last_charge_date" swaggertype:"string" example:"2026-05-30T22:00:00+01:00"`
}

// V2ParkingsAgg is the parkings section of the lifetime stats response.
type V2ParkingsAgg struct {
	Count              int     `json:"count" example:"1893"`
	TotalDurationMin   int     `json:"total_duration_min" example:"500000"`
	AvgDurationMin     float64 `json:"avg_duration_min" example:"264.2"`
	LongestParkingMin  int     `json:"longest_parking_min" example:"43200"`
	TotalEnergyDropKWh float64 `json:"total_vampire_drain_kwh" example:"125.0"`
}

// V2UpdatesAgg is the firmware-updates section of the lifetime stats response.
type V2UpdatesAgg struct {
	Count            int             `json:"count" example:"58"`
	FirstVersion     nullable.String `json:"first_version" swaggertype:"string" example:"2020.4.10"`
	LatestVersion    nullable.String `json:"latest_version" swaggertype:"string" example:"2026.20.1"`
	LatestUpdateDate nullable.String `json:"latest_update_date" swaggertype:"string" example:"2026-05-15T03:00:00+01:00"`
}

// V2Lifetime is the `data` field of V2LifetimeResponse.
type V2Lifetime struct {
	Car      Car             `json:"car"`
	CarMeta  V2CarMeta       `json:"car_meta"`
	Since    nullable.String `json:"since" swaggertype:"string" example:"2020-01-01T00:00:00+01:00"`
	Drives   V2DrivesAgg     `json:"drives"`
	Charges  V2ChargesAgg    `json:"charges"`
	Parkings V2ParkingsAgg   `json:"parkings"`
	Updates  V2UpdatesAgg    `json:"updates"`
	Units    TeslaMateUnits  `json:"units"`
}

// V2LifetimeResponse is the envelope for /api/v2/cars/{CarID}/stats/lifetime.
type V2LifetimeResponse struct {
	Data V2Lifetime `json:"data"`
}
