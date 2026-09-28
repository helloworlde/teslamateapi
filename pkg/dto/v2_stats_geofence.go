package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2GeofenceRow is one row in the by-geofence stats response.
type V2GeofenceRow struct {
	GeofenceID            nullable.Int64   `json:"geofence_id" swaggertype:"integer" extensions:"x-nullable" example:"3"`
	GeofenceName          string           `json:"geofence_name" example:"Home"`
	DrivesArrived         int              `json:"drives_arrived" example:"125"`
	DrivesDeparted        int              `json:"drives_departed" example:"126"`
	ChargesCount          int              `json:"charges_count" example:"42"`
	ChargesEnergyAddedKWh nullable.Float64 `json:"charges_energy_added_kwh" swaggertype:"number" format:"double" extensions:"x-nullable" example:"1820.5"`
	ChargesCost           nullable.Float64 `json:"charges_cost" swaggertype:"number" format:"double" extensions:"x-nullable" example:"425.30"`
	// Every charging metric uses its own valid samples; a missing result is null,
	// while a recorded zero remains zero. Session counts expose partial coverage.
	ChargesEnergyUsedKWh           nullable.Float64 `json:"charges_energy_used_kwh" swaggertype:"number" format:"double" extensions:"x-nullable" example:"2000"`
	ChargesDurationMin             nullable.Int64   `json:"charges_duration_min" swaggertype:"integer" format:"int64" extensions:"x-nullable" example:"12600"`
	ChargesEnergyAddedSessionCount int              `json:"charges_energy_added_session_count" example:"41"`
	ChargesEnergyUsedSessionCount  int              `json:"charges_energy_used_session_count" example:"42"`
	ChargesCostSessionCount        int              `json:"charges_cost_session_count" example:"40"`
	ChargesDurationSessionCount    int              `json:"charges_duration_session_count" example:"42"`
	// ChargesUnitCostPerKWh uses only sessions with valid cost and positive
	// charger energy; explicit zero cost is valid.
	ChargesUnitCostPerKWh        nullable.Float64 `json:"charges_unit_cost_per_kwh" swaggertype:"number" format:"double" extensions:"x-nullable" example:"0.21265"`
	ChargesUnitCostSessionCount  int              `json:"charges_unit_cost_session_count" example:"40"`
	ChargesUnitCostEnergyUsedKWh float64          `json:"charges_unit_cost_energy_used_kwh" example:"1950"`
	// ChargesEfficiencyPct uses only sessions with valid paired vehicle/charger
	// energy. It is a ratio of sums, never an average of per-session ratios.
	ChargesEfficiencyPct           nullable.Float64 `json:"charges_efficiency_pct" swaggertype:"number" format:"double" extensions:"x-nullable" example:"91.025"`
	ChargesEfficiencySessionCount  int              `json:"charges_efficiency_session_count" example:"39"`
	ChargesEfficiencyEnergyUsedKWh float64          `json:"charges_efficiency_energy_used_kwh" example:"1900"`
	ParkingsCount                  int              `json:"parkings_count" example:"125"`
	ParkingsTotalDurationMin       int              `json:"parkings_total_duration_min" example:"50000"`
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
