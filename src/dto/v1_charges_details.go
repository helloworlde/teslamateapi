package dto

import "github.com/tobiasehlert/teslamateapi/src/nullable"

// V1ChargeDetailBatteryDetails carries start/end SOC for the charge session.
type V1ChargeDetailBatteryDetails struct {
	StartBatteryLevel int `json:"start_battery_level" example:"32"`
	EndBatteryLevel   int `json:"end_battery_level" example:"82"`
}

// V1ChargeDetailPreferredRange carries start/end range for the charge session.
type V1ChargeDetailPreferredRange struct {
	StartRange float64 `json:"start_range" example:"110.5"`
	EndRange   float64 `json:"end_range" example:"320.7"`
}

// V1ChargeDetailChargerDetails carries the per-sample charger telemetry.
type V1ChargeDetailChargerDetails struct {
	ChargerActualCurrent int `json:"charger_actual_current" example:"32"`
	ChargerPhases        int `json:"charger_phases" example:"3"`
	ChargerPilotCurrent  int `json:"charger_pilot_current" example:"32"`
	ChargerPower         int `json:"charger_power" example:"22"`
	ChargerVoltage       int `json:"charger_voltage" example:"240"`
}

// V1ChargeDetailFastChargerInfo carries fast-charger fields for a sample.
type V1ChargeDetailFastChargerInfo struct {
	FastChargerPresent bool            `json:"fast_charger_present" example:"false"`
	FastChargerBrand   nullable.String `json:"fast_charger_brand" swaggertype:"string" example:"Tesla"`
	FastChargerType    string          `json:"fast_charger_type" example:"Combo"`
}

// V1ChargeDetailBatteryInfo carries battery range / heater state for a sample.
type V1ChargeDetailBatteryInfo struct {
	IdealBatteryRange    float64       `json:"ideal_battery_range" example:"320.5"`
	RatedBatteryRange    float64       `json:"rated_battery_range" example:"205.4"`
	BatteryHeater        bool          `json:"battery_heater" example:"false"`
	BatteryHeaterOn      bool          `json:"battery_heater_on" example:"false"`
	BatteryHeaterNoPower nullable.Bool `json:"battery_heater_no_power" swaggertype:"boolean"`
}

// V1ChargeDetailItem is one sample inside V1ChargeDetail.ChargeDetails.
type V1ChargeDetailItem struct {
	DetailID             int                           `json:"detail_id" example:"123456"`
	Date                 string                        `json:"date" example:"2024-01-01T21:30:00+01:00"`
	BatteryLevel         int                           `json:"battery_level" example:"58"`
	UsableBatteryLevel   int                           `json:"usable_battery_level" example:"57"`
	ChargeEnergyAdded    float64                       `json:"charge_energy_added" example:"15.2"`
	NotEnoughPowerToHeat nullable.Bool                 `json:"not_enough_power_to_heat" swaggertype:"boolean"`
	ChargerDetails       V1ChargeDetailChargerDetails  `json:"charger_details"`
	BatteryInfo          V1ChargeDetailBatteryInfo     `json:"battery_info"`
	ConnChargeCable      string                        `json:"conn_charge_cable" example:"IEC"`
	FastChargerInfo      V1ChargeDetailFastChargerInfo `json:"fast_charger_info"`
	OutsideTemp          float64                       `json:"outside_temp" example:"11.5"`
}

// V1ChargeDetail is the full per-session blob for the charge-detail endpoint.
type V1ChargeDetail struct {
	ChargeID          int                          `json:"charge_id" example:"1234"`
	StartDate         string                       `json:"start_date" example:"2024-01-01T20:00:00+01:00"`
	EndDate           string                       `json:"end_date" example:"2024-01-01T22:30:00+01:00"`
	Address           string                       `json:"address" example:"Home"`
	ChargeEnergyAdded float64                      `json:"charge_energy_added" example:"38.45"`
	ChargeEnergyUsed  float64                      `json:"charge_energy_used" example:"40.10"`
	Cost              float64                      `json:"cost" example:"5.42"`
	DurationMin       int                          `json:"duration_min" example:"150"`
	DurationStr       string                       `json:"duration_str" example:"02:30"`
	BatteryDetails    V1ChargeDetailBatteryDetails `json:"battery_details"`
	RangeIdeal        V1ChargeDetailPreferredRange `json:"range_ideal"`
	RangeRated        V1ChargeDetailPreferredRange `json:"range_rated"`
	OutsideTempAvg    float64                      `json:"outside_temp_avg" example:"12.5"`
	Odometer          float64                      `json:"odometer" example:"42100.7"`
	Latitude          float64                      `json:"latitude" example:"52.5200"`
	Longitude         float64                      `json:"longitude" example:"13.4050"`
	ChargeDetails     []V1ChargeDetailItem         `json:"charge_details"`
}

// V1ChargeDetailData is the `data` field of V1ChargeDetailResponse.
type V1ChargeDetailData struct {
	Car            Car            `json:"car"`
	Charge         V1ChargeDetail `json:"charge"`
	TeslaMateUnits TeslaMateUnits `json:"units"`
}

// V1ChargeDetailResponse is the envelope for /api/v1/cars/{CarID}/charges/{ChargeID}.
type V1ChargeDetailResponse struct {
	Data V1ChargeDetailData `json:"data"`
}
