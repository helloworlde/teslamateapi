package dto

import "github.com/tobiasehlert/teslamateapi/src/nullable"

// V1ChargeCurrentBatteryDetails captures start/current SOC for the active session.
type V1ChargeCurrentBatteryDetails struct {
	StartBatteryLevel   int `json:"start_battery_level" example:"32"`
	CurrentBatteryLevel int `json:"current_battery_level" example:"58"`
}

// V1ChargeCurrentPreferredRange captures start/current/added range.
type V1ChargeCurrentPreferredRange struct {
	StartRange   float64 `json:"start_range" example:"110.5"`
	CurrentRange float64 `json:"current_range" example:"205.4"`
	AddedRange   float64 `json:"added_range" example:"94.9"`
}

// V1ChargeCurrentChargerDetails carries the per-sample charger telemetry for
// the latest charge sample (already COALESCEd to 0 when null).
type V1ChargeCurrentChargerDetails struct {
	ChargerActualCurrent int `json:"charger_actual_current" example:"32"`
	ChargerPhases        int `json:"charger_phases" example:"3"`
	ChargerPilotCurrent  int `json:"charger_pilot_current" example:"32"`
	ChargerPower         int `json:"charger_power" example:"22"`
	ChargerVoltage       int `json:"charger_voltage" example:"240"`
}

// V1ChargeCurrentFastChargerInfo carries fast-charger fields for the active
// session. brand/type are pointers so they stay omitted when unknown.
type V1ChargeCurrentFastChargerInfo struct {
	FastChargerPresent bool    `json:"fast_charger_present" example:"false"`
	FastChargerBrand   *string `json:"fast_charger_brand,omitempty" example:"Tesla"`
	FastChargerType    *string `json:"fast_charger_type,omitempty" example:"Combo"`
}

// V1ChargeCurrentBatteryInfo carries battery range / heater state for the
// latest charge sample.
type V1ChargeCurrentBatteryInfo struct {
	RatedBatteryRange    float64       `json:"rated_battery_range" example:"205.4"`
	BatteryHeater        bool          `json:"battery_heater" example:"false"`
	BatteryHeaterOn      bool          `json:"battery_heater_on" example:"false"`
	BatteryHeaterNoPower nullable.Bool `json:"battery_heater_no_power" swaggertype:"boolean"`
}

// V1ChargeCurrentDetail is one sample inside V1ChargeCurrent.ChargeDetails.
type V1ChargeCurrentDetail struct {
	DetailID             int                            `json:"detail_id" example:"123456"`
	Date                 string                         `json:"date" example:"2024-01-01T21:30:00+01:00"`
	BatteryLevel         int                            `json:"battery_level" example:"58"`
	UsableBatteryLevel   int                            `json:"usable_battery_level" example:"57"`
	ChargeEnergyAdded    float64                        `json:"charge_energy_added" example:"15.2"`
	NotEnoughPowerToHeat nullable.Bool                  `json:"not_enough_power_to_heat" swaggertype:"boolean"`
	ChargerDetails       V1ChargeCurrentChargerDetails  `json:"charger_details"`
	BatteryInfo          V1ChargeCurrentBatteryInfo     `json:"battery_info"`
	ConnChargeCable      interface{}                    `json:"conn_charge_cable,omitempty" swaggertype:"string" example:"IEC"`
	FastChargerInfo      V1ChargeCurrentFastChargerInfo `json:"fast_charger_info"`
	OutsideTemp          float64                        `json:"outside_temp" example:"11.5"`
}

// V1ChargeCurrent represents the active charging session.
type V1ChargeCurrent struct {
	ChargeID          int                           `json:"charge_id" example:"1234"`
	StartDate         string                        `json:"start_date" example:"2024-01-01T20:00:00+01:00"`
	IsCharging        bool                          `json:"is_charging" example:"true"`
	Address           string                        `json:"address" example:"Home"`
	ChargeEnergyAdded float64                       `json:"charge_energy_added" example:"15.2"`
	Cost              float64                       `json:"cost" example:"3.20"`
	DurationMin       int                           `json:"duration_min" example:"95"`
	DurationStr       string                        `json:"duration_str" example:"01:35"`
	BatteryDetails    V1ChargeCurrentBatteryDetails `json:"battery_details"`
	RatedRange        V1ChargeCurrentPreferredRange `json:"rated_range"`
	OutsideTempAvg    float64                       `json:"outside_temp_avg" example:"11.5"`
	Odometer          float64                       `json:"odometer" example:"42500.0"`
	ChargeDetails     []V1ChargeCurrentDetail       `json:"charge_details"`
}

// V1ChargesCurrentData is the `data` field of V1ChargesCurrentResponse.
type V1ChargesCurrentData struct {
	Car            Car             `json:"car"`
	Charge         V1ChargeCurrent `json:"charge"`
	TeslaMateUnits TeslaMateUnits  `json:"units"`
}

// V1ChargesCurrentResponse is the envelope for /api/v1/cars/{CarID}/charges/current.
type V1ChargesCurrentResponse struct {
	Data V1ChargesCurrentData `json:"data"`
}
