package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2GeofenceRow is one row in the by-geofence stats response.
type V2GeofenceRow struct {
	GeofenceID               nullable.Int64 `json:"geofence_id" swaggertype:"integer" extensions:"x-nullable" example:"3"`
	GeofenceName             string         `json:"geofence_name" example:"Home"`
	DrivesArrived            int            `json:"drives_arrived" example:"125"`
	DrivesDeparted           int            `json:"drives_departed" example:"126"`
	ChargesCount             int            `json:"charges_count" example:"42"`
	ChargesEnergyAddedKWh    float64        `json:"charges_energy_added_kwh" example:"1820.5"`
	ChargesCost              float64        `json:"charges_cost" example:"425.30"`
	ParkingsCount            int            `json:"parkings_count" example:"125"`
	ParkingsTotalDurationMin int            `json:"parkings_total_duration_min" example:"50000"`
}

// V2ByGeofenceData is the `data` field of V2ByGeofenceResponse.
type V2ByGeofenceData struct {
	Car       Car             `json:"car"`
	Geofences []V2GeofenceRow `json:"geofences"`
	Units     TeslaMateUnits  `json:"units"`
}

// V2ByGeofenceResponse is the envelope for /api/v2/cars/{CarID}/stats/by-geofence.
type V2ByGeofenceResponse struct {
	Data V2ByGeofenceData `json:"data"`
}
