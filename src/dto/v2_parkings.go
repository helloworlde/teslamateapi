package dto

import "github.com/tobiasehlert/teslamateapi/src/nullable"

// V2Parking is one row in /api/v2/cars/{CarID}/parkings.
type V2Parking struct {
	PrecedingDriveID  int              `json:"preceding_drive_id" example:"1234"`
	StartDate         string           `json:"start_date" example:"2024-01-01T09:30:00+01:00"`
	EndDate           nullable.String  `json:"end_date" swaggertype:"string" example:"2024-01-01T17:00:00+01:00"`
	DurationMin       int              `json:"duration_min" example:"450"`
	DurationStr       string           `json:"duration_str" example:"07:30"`
	Address           nullable.String  `json:"address" swaggertype:"string" example:"Office"`
	GeofenceID        nullable.Int64   `json:"geofence_id" swaggertype:"integer" example:"3"`
	Latitude          nullable.Float64 `json:"latitude" swaggertype:"number" example:"52.5200"`
	Longitude         nullable.Float64 `json:"longitude" swaggertype:"number" example:"13.4050"`
	StartBatteryLevel nullable.Int64   `json:"start_battery_level" swaggertype:"integer" example:"50"`
	EndBatteryLevel   nullable.Int64   `json:"end_battery_level" swaggertype:"integer" example:"48"`
	UsableBatteryDrop int              `json:"usable_battery_drop" example:"2"`
	EnergyConsumedKWh nullable.Float64 `json:"energy_consumed_kwh" swaggertype:"number" example:"1.2"`
	HadCharging       bool             `json:"had_charging" example:"false"`
}

// V2ParkingsData is the `data` field of V2ParkingsResponse.
type V2ParkingsData struct {
	Car            Car            `json:"car"`
	Parkings       []V2Parking    `json:"parkings"`
	Pagination     Pagination     `json:"pagination"`
	TeslaMateUnits TeslaMateUnits `json:"units"`
}

// V2ParkingsResponse is the envelope for /api/v2/cars/{CarID}/parkings.
type V2ParkingsResponse struct {
	Data V2ParkingsData `json:"data"`
}

// V2ParkingDetailPoint is a position sample inside a parking session.
type V2ParkingDetailPoint struct {
	Date         string           `json:"date" example:"2024-01-01T10:00:00+01:00"`
	BatteryLevel nullable.Int64   `json:"battery_level" swaggertype:"integer" example:"50"`
	UsableLevel  nullable.Int64   `json:"usable_battery_level" swaggertype:"integer" example:"49"`
	OutsideTemp  nullable.Float64 `json:"outside_temp" swaggertype:"number" example:"15.3"`
}

// V2ParkingDetail extends V2Parking with the avg outside temp + sample series.
type V2ParkingDetail struct {
	PrecedingDriveID  int                    `json:"preceding_drive_id" example:"1234"`
	StartDate         string                 `json:"start_date" example:"2024-01-01T09:30:00+01:00"`
	EndDate           nullable.String        `json:"end_date" swaggertype:"string" example:"2024-01-01T17:00:00+01:00"`
	DurationMin       int                    `json:"duration_min" example:"450"`
	DurationStr       string                 `json:"duration_str" example:"07:30"`
	Address           nullable.String        `json:"address" swaggertype:"string" example:"Office"`
	GeofenceID        nullable.Int64         `json:"geofence_id" swaggertype:"integer" example:"3"`
	Latitude          nullable.Float64       `json:"latitude" swaggertype:"number" example:"52.5200"`
	Longitude         nullable.Float64       `json:"longitude" swaggertype:"number" example:"13.4050"`
	StartBatteryLevel nullable.Int64         `json:"start_battery_level" swaggertype:"integer" example:"50"`
	EndBatteryLevel   nullable.Int64         `json:"end_battery_level" swaggertype:"integer" example:"48"`
	UsableBatteryDrop int                    `json:"usable_battery_drop" example:"2"`
	EnergyConsumedKWh nullable.Float64       `json:"energy_consumed_kwh" swaggertype:"number" example:"1.2"`
	HadCharging       bool                   `json:"had_charging" example:"false"`
	OutsideTempAvg    nullable.Float64       `json:"outside_temp_avg" swaggertype:"number" example:"15.3"`
	Details           []V2ParkingDetailPoint `json:"parking_details"`
}

// V2ParkingDetailData is the `data` field of V2ParkingDetailResponse.
type V2ParkingDetailData struct {
	Car            Car             `json:"car"`
	Parking        V2ParkingDetail `json:"parking"`
	TeslaMateUnits TeslaMateUnits  `json:"units"`
}

// V2ParkingDetailResponse is the envelope for the parking-detail endpoint.
type V2ParkingDetailResponse struct {
	Data V2ParkingDetailData `json:"data"`
}
