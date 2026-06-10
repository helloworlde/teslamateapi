package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V1CarDetails represents make/model metadata for a car.
type V1CarDetails struct {
	EID         int64            `json:"eid" example:"123456789012345"`
	VID         int64            `json:"vid" example:"987654321098765"`
	Vin         string           `json:"vin" example:"LRWYGCEKXMC000001"`
	VINDetails  *V1CarVINDetails `json:"vin_details,omitempty"`
	Model       nullable.String  `json:"model" swaggertype:"string" example:"Y"`
	TrimBadging nullable.String  `json:"trim_badging" swaggertype:"string" example:"LR"`
	Efficiency  nullable.Float64 `json:"efficiency" swaggertype:"number" example:"0.184"`
}

// V1CarVINDetails carries Tesla VIN-decoded static metadata.
type V1CarVINDetails struct {
	WMI             string `json:"wmi" example:"LRW"`
	Manufacturer    string `json:"manufacturer,omitempty" example:"Tesla China (Giga Shanghai)"`
	LineSeries      string `json:"line_series,omitempty" example:"Model Y"`
	BodyType        string `json:"body_type,omitempty" example:"MPV 5 DR / LHD"`
	RestraintSystem string `json:"restraint_system,omitempty" example:"Type 2 Manual seatbelts with front Airbags, PODS, side Inflatable restraints"`
	FuelType        string `json:"fuel_type,omitempty" example:"Ternary System Li-ion battery"`
	MotorDriveUnit  string `json:"motor_drive_unit,omitempty" example:"Dual Motor Standard"`
	CheckDigit      string `json:"check_digit" example:"X"`
	ProductionYear  int    `json:"production_year,omitempty" example:"2021"`
	YearType        string `json:"year_type,omitempty" example:"calendar_year"`
	Plant           string `json:"plant,omitempty" example:"Tesla China (Giga Shanghai)"`
	SequenceNumber  string `json:"sequence_number" example:"000001"`
}

// V1CarExterior captures exterior config fields surfaced by the /cars list.
type V1CarExterior struct {
	ExteriorColor string `json:"exterior_color" example:"DeepBlue"`
	SpoilerType   string `json:"spoiler_type" example:"None"`
	WheelType     string `json:"wheel_type" example:"Pinwheel18"`
}

// V1CarSettings captures TeslaMate-side settings for the car.
type V1CarSettings struct {
	SuspendMin          int  `json:"suspend_min" example:"21"`
	SuspendAfterIdleMin int  `json:"suspend_after_idle_min" example:"15"`
	ReqNotUnlocked      bool `json:"req_not_unlocked" example:"false"`
	FreeSupercharging   bool `json:"free_supercharging" example:"false"`
	UseStreamingAPI     bool `json:"use_streaming_api" example:"true"`
}

// V1CarTeslaMateDetails captures inserted/updated audit timestamps.
type V1CarTeslaMateDetails struct {
	InsertedAt string `json:"inserted_at" example:"2020-01-01T00:00:00+01:00"`
	UpdatedAt  string `json:"updated_at" example:"2024-01-01T00:00:00+01:00"`
}

// V1CarTeslaMateStats captures denormalized count totals for the car.
type V1CarTeslaMateStats struct {
	TotalCharges int `json:"total_charges" example:"427"`
	TotalDrives  int `json:"total_drives" example:"1893"`
	TotalUpdates int `json:"total_updates" example:"38"`
}

// V1Car is one row in GET /api/v1/cars.
type V1Car struct {
	CarID            int                   `json:"car_id" example:"1"`
	Name             nullable.String       `json:"name" swaggertype:"string" example:"Blue Thunder"`
	CarDetails       V1CarDetails          `json:"car_details"`
	CarExterior      V1CarExterior         `json:"car_exterior"`
	CarSettings      V1CarSettings         `json:"car_settings"`
	TeslaMateDetails V1CarTeslaMateDetails `json:"teslamate_details"`
	TeslaMateStats   V1CarTeslaMateStats   `json:"teslamate_stats"`
}

// V1CarsData is the `data` field of V1CarsResponse.
type V1CarsData struct {
	Cars []V1Car `json:"cars"`
}

// V1CarsResponse is the envelope for GET /api/v1/cars (and /api/v1/cars/{CarID}).
type V1CarsResponse struct {
	Data V1CarsData `json:"data"`
}
