package dto

import "github.com/tobiasehlert/teslamateapi/src/nullable"

// V1ChargesListBatteryDetails carries start/end SOC for one charge session.
type V1ChargesListBatteryDetails struct {
	StartBatteryLevel int `json:"start_battery_level" example:"32"`
	EndBatteryLevel   int `json:"end_battery_level" example:"82"`
}

// V1ChargesListPreferredRange carries start/end range for one charge session.
type V1ChargesListPreferredRange struct {
	StartRange float64 `json:"start_range" example:"110.5"`
	EndRange   float64 `json:"end_range" example:"320.7"`
}

// V1ChargeListItem represents a single charging session row in
// GET /api/v1/cars/{CarID}/charges.
type V1ChargeListItem struct {
	ChargeID           int                         `json:"charge_id" example:"1234"`
	StartDate          string                      `json:"start_date" example:"2024-01-01T20:00:00+01:00"`
	EndDate            string                      `json:"end_date" example:"2024-01-01T22:30:00+01:00"`
	Address            string                      `json:"address" example:"Home"`
	ChargeEnergyAdded  float64                     `json:"charge_energy_added" example:"38.45"`
	ChargeEnergyUsed   float64                     `json:"charge_energy_used" example:"40.10"`
	Cost               float64                     `json:"cost" example:"5.42"`
	DurationMin        int                         `json:"duration_min" example:"150"`
	DurationStr        string                      `json:"duration_str" example:"02:30"`
	BatteryDetails     V1ChargesListBatteryDetails `json:"battery_details"`
	RangeIdeal         V1ChargesListPreferredRange `json:"range_ideal"`
	RangeRated         V1ChargesListPreferredRange `json:"range_rated"`
	OutsideTempAvg     float64                     `json:"outside_temp_avg" example:"12.5"`
	Odometer           float64                     `json:"odometer" example:"42100.7"`
	Latitude           float64                     `json:"latitude" example:"52.5200"`
	Longitude          float64                     `json:"longitude" example:"13.4050"`
	FastChargerPresent bool                        `json:"fast_charger_present" example:"false"`
	FastChargerBrand   nullable.String             `json:"fast_charger_brand" swaggertype:"string" example:"Tesla"`
	FastChargerType    nullable.String             `json:"fast_charger_type" swaggertype:"string" example:"Combo"`
	PeakChargerPower   int                         `json:"peak_charger_power" example:"48"`
	PeakChargerVoltage int                         `json:"peak_charger_voltage" example:"240"`
	ChargerPhases      nullable.Int64              `json:"charger_phases" swaggertype:"integer" example:"3"`
	ConnChargeCable    nullable.String             `json:"conn_charge_cable" swaggertype:"string" example:"IEC"`
	PilotCurrentMax    nullable.Int64              `json:"pilot_current_max" swaggertype:"integer" example:"32"`
	ChargeType         string                      `json:"charge_type" example:"ac" enums:"ac,dc,unknown"`
	IsTeslaCharger     bool                        `json:"is_tesla_charger" example:"true"`
}

// V1ChargesData is the `data` field of V1ChargesResponse.
type V1ChargesData struct {
	Car            Car                `json:"car"`
	Charges        []V1ChargeListItem `json:"charges"`
	TeslaMateUnits TeslaMateUnits     `json:"units"`
}

// V1ChargesResponse is the envelope for GET /api/v1/cars/{CarID}/charges.
type V1ChargesResponse struct {
	Data V1ChargesData `json:"data"`
}
