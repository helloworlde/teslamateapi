package dto

// V1StatusBatteryDetails captures the battery range / SOC fields of /status.
type V1StatusBatteryDetails struct {
	EstBatteryRange    float64 `json:"est_battery_range" example:"372.5"`
	RatedBatteryRange  float64 `json:"rated_battery_range" example:"401.63"`
	IdealBatteryRange  float64 `json:"ideal_battery_range" example:"335.79"`
	BatteryLevel       int     `json:"battery_level" example:"88"`
	UsableBatteryLevel int     `json:"usable_battery_level" example:"85"`
}

// V1StatusCarDetails captures model/trim metadata for /status.
type V1StatusCarDetails struct {
	Model       string `json:"model" example:"S"`
	TrimBadging string `json:"trim_badging" example:"P100D"`
}

// V1StatusCarExterior captures exterior-config fields for /status.
type V1StatusCarExterior struct {
	ExteriorColor string `json:"exterior_color" example:"DeepBlue"`
	SpoilerType   string `json:"spoiler_type" example:"None"`
	WheelType     string `json:"wheel_type" example:"Pinwheel18"`
}

// V1StatusCarLocation is a lat/long pair re-used by multiple status structs.
type V1StatusCarLocation struct {
	Latitude  float64 `json:"latitude" example:"35.278131"`
	Longitude float64 `json:"longitude" example:"29.744801"`
}

// V1StatusCarGeodata captures the current geofence/location of /status.
// Latitude/Longitude are deprecated mirrors of Location.
type V1StatusCarGeodata struct {
	Geofence  string              `json:"geofence" example:"Home"`
	Location  V1StatusCarLocation `json:"location"`
	Latitude  float64             `json:"latitude" example:"35.278131"`
	Longitude float64             `json:"longitude" example:"29.744801"`
}

// V1StatusCarStatus captures door/lock/sentry-mode booleans for /status.
type V1StatusCarStatus struct {
	Healthy                bool  `json:"healthy" example:"true"`
	Locked                 *bool `json:"locked" example:"true"`
	SentryMode             bool  `json:"sentry_mode" example:"false"`
	WindowsOpen            bool  `json:"windows_open" example:"false"`
	DoorsOpen              bool  `json:"doors_open" example:"false"`
	DriverFrontDoorOpen    bool  `json:"driver_front_door_open" example:"false"`
	DriverRearDoorOpen     bool  `json:"driver_rear_door_open" example:"false"`
	PassengerFrontDoorOpen bool  `json:"passenger_front_door_open" example:"false"`
	PassengerRearDoorOpen  bool  `json:"passenger_rear_door_open" example:"false"`
	TrunkOpen              bool  `json:"trunk_open" example:"false"`
	FrunkOpen              bool  `json:"frunk_open" example:"false"`
	IsUserPresent          bool  `json:"is_user_present" example:"false"`
	CenterDisplayState     int   `json:"center_display_state" example:"0"`
}

// V1StatusCarVersions captures software version info for /status.
type V1StatusCarVersions struct {
	Version         string `json:"version" example:"2024.32.12.2"`
	UpdateAvailable bool   `json:"update_available" example:"false"`
	UpdateVersion   string `json:"update_version" example:"2024.32.12.3"`
}

// V1StatusChargingDetails captures charging telemetry for /status.
type V1StatusChargingDetails struct {
	PluggedIn                  bool    `json:"plugged_in" example:"true"`
	ChargingState              string  `json:"charging_state" example:"charging"`
	ChargeEnergyAdded          float64 `json:"charge_energy_added" example:"5.06"`
	ChargeLimitSoc             int     `json:"charge_limit_soc" example:"90"`
	ChargePortDoorOpen         bool    `json:"charge_port_door_open" example:"true"`
	ChargerActualCurrent       float64 `json:"charger_actual_current" example:"2.05"`
	ChargerPhases              int     `json:"charger_phases" example:"3"`
	ChargerPower               float64 `json:"charger_power" example:"48.9"`
	ChargerVoltage             int     `json:"charger_voltage" example:"240"`
	ChargeCurrentRequest       int     `json:"charge_current_request" example:"40"`
	ChargeCurrentRequestMax    int     `json:"charge_current_request_max" example:"40"`
	ScheduledChargingStartTime string  `json:"scheduled_charging_start_time" example:"2024-02-29T23:00:07+01:00"`
	TimeToFullCharge           float64 `json:"time_to_full_charge" example:"1.83"`
}

// V1StatusClimateDetails captures inside/outside temp + climate state.
type V1StatusClimateDetails struct {
	IsClimateOn       bool    `json:"is_climate_on" example:"true"`
	InsideTemp        float64 `json:"inside_temp" example:"20.8"`
	OutsideTemp       float64 `json:"outside_temp" example:"18.4"`
	IsPreconditioning bool    `json:"is_preconditioning" example:"false"`
	ClimateKeeperMode string  `json:"climate_keeper_mode" example:"dog"`
}

// V1StatusActiveRouteDetails captures the current navigation route info.
type V1StatusActiveRouteDetails struct {
	Destination         string              `json:"destination" example:"Home"`
	EnergyAtArrival     int                 `json:"energy_at_arrival" example:"73"`
	DistanceToArrival   float64             `json:"distance_to_arrival" example:"10.43"`
	MinutesToArrival    float64             `json:"minutes_to_arrival" example:"23.46"`
	TrafficMinutesDelay float64             `json:"traffic_minutes_delay" example:"0.0"`
	Location            V1StatusCarLocation `json:"location"`
}

// V1StatusDrivingDetails captures shift/speed/heading and the deprecated
// active_route_* mirrors for /status.
type V1StatusDrivingDetails struct {
	ActiveRoute            V1StatusActiveRouteDetails `json:"active_route"`
	ActiveRouteDestination string                     `json:"active_route_destination" example:"Home"`
	ActiveRouteLatitude    float64                    `json:"active_route_latitude" example:"35.278131"`
	ActiveRouteLongitude   float64                    `json:"active_route_longitude" example:"29.744801"`
	ShiftState             string                     `json:"shift_state" example:"D"`
	Power                  int                        `json:"power" example:"-9"`
	Speed                  int                        `json:"speed" example:"12"`
	Heading                int                        `json:"heading" example:"340"`
	Elevation              int                        `json:"elevation" example:"70"`
}

// V1StatusTpmsDetails captures tire-pressure values for /status.
type V1StatusTpmsDetails struct {
	TpmsPressureFL    float64 `json:"tpms_pressure_fl" example:"2.9"`
	TpmsPressureFR    float64 `json:"tpms_pressure_fr" example:"2.8"`
	TpmsPressureRL    float64 `json:"tpms_pressure_rl" example:"2.9"`
	TpmsPressureRR    float64 `json:"tpms_pressure_rr" example:"2.8"`
	TpmsSoftWarningFL bool    `json:"tpms_soft_warning_fl" example:"true"`
	TpmsSoftWarningFR bool    `json:"tpms_soft_warning_fr" example:"false"`
	TpmsSoftWarningRL bool    `json:"tpms_soft_warning_rl" example:"false"`
	TpmsSoftWarningRR bool    `json:"tpms_soft_warning_rr" example:"false"`
}

// V1StatusInformation is the full status struct exposed by /status.
type V1StatusInformation struct {
	DisplayName     string                  `json:"display_name" example:"Blue Thunder"`
	State           string                  `json:"state" example:"asleep"`
	StateSince      string                  `json:"state_since" example:"2024-02-29T23:00:07+01:00"`
	Odometer        float64                 `json:"odometer" example:"42100.5"`
	CarStatus       V1StatusCarStatus       `json:"car_status"`
	CarDetails      V1StatusCarDetails      `json:"car_details"`
	CarExterior     V1StatusCarExterior     `json:"car_exterior"`
	CarGeodata      V1StatusCarGeodata      `json:"car_geodata"`
	CarVersions     V1StatusCarVersions     `json:"car_versions"`
	DrivingDetails  V1StatusDrivingDetails  `json:"driving_details"`
	ClimateDetails  V1StatusClimateDetails  `json:"climate_details"`
	BatteryDetails  V1StatusBatteryDetails  `json:"battery_details"`
	ChargingDetails V1StatusChargingDetails `json:"charging_details"`
	TpmsDetails     V1StatusTpmsDetails     `json:"tpms_details"`
}

// V1StatusData is the `data` field of V1StatusResponse.
// The status field is keyed `status`, not `mqtt_information`.
type V1StatusData struct {
	Car            Car                        `json:"car"`
	Status         V1StatusInformation        `json:"status"`
	TeslaMateUnits TeslaMateUnitsWithPressure `json:"units"`
}

// V1StatusResponse is the envelope for /api/v1/cars/{CarID}/status.
type V1StatusResponse struct {
	Data V1StatusData `json:"data"`
}
