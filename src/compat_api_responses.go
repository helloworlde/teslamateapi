package main

type APISystemMessageResponse struct {
	Message string `json:"message" example:"pong"`
	Path    string `json:"path,omitempty" example:"/api/v1"`
}

type APISystemErrorBody struct {
	Error string `json:"error" example:"Unable to load summary."`
}

// CarsV1Envelope documents GET /v1/cars and GET /v1/cars/{CarID} success payloads (same envelope; single-car returns one element in cars).
type CarsV1Envelope struct {
	Data CarsV1Data `json:"data"`
}

// CarsV1Data is the `data` object for car list/detail compatible endpoints.
type CarsV1Data struct {
	Cars []CarsV1Car `json:"cars"`
}

// CarsV1Car mirrors TeslaMateAPICarsV1 JSON shape (see v1_TeslaMateAPICars.go).
type CarsV1Car struct {
	CarID            int                  `json:"car_id" example:"1"`
	Name             NullString           `json:"name" swaggertype:"string" example:"My Tesla"`
	CarDetails       CarsV1CarDetails     `json:"car_details"`
	CarExterior      CarsV1CarExterior    `json:"car_exterior"`
	CarSettings      CarsV1CarSettings    `json:"car_settings"`
	TeslaMateDetails CarsV1TeslaMateMeta  `json:"teslamate_details"`
	TeslaMateStats   CarsV1TeslaMateStats `json:"teslamate_stats"`
}

type CarsV1CarDetails struct {
	EID         int64       `json:"eid" example:"123456789"`
	VID         int64       `json:"vid" example:"987654321"`
	Vin         string      `json:"vin" example:"5YJ3E1EA1KF000000"`
	Model       NullString  `json:"model" swaggertype:"string" example:"Model 3"`
	TrimBadging NullString  `json:"trim_badging" swaggertype:"string" example:""`
	Efficiency  NullFloat64 `json:"efficiency" swaggertype:"number" example:"145"`
}

type CarsV1CarExterior struct {
	ExteriorColor string `json:"exterior_color"`
	SpoilerType   string `json:"spoiler_type"`
	WheelType     string `json:"wheel_type"`
}

type CarsV1CarSettings struct {
	SuspendMin          int  `json:"suspend_min" example:"15"`
	SuspendAfterIdleMin int  `json:"suspend_after_idle_min" example:"10"`
	ReqNotUnlocked      bool `json:"req_not_unlocked"`
	FreeSupercharging   bool `json:"free_supercharging"`
	UseStreamingAPI     bool `json:"use_streaming_api"`
}

type CarsV1TeslaMateMeta struct {
	InsertedAt string `json:"inserted_at" example:"2026-04-01T12:00:00+08:00"`
	UpdatedAt  string `json:"updated_at" example:"2026-04-01T12:00:00+08:00"`
}

type CarsV1TeslaMateStats struct {
	TotalCharges int `json:"total_charges" example:"120"`
	TotalDrives  int `json:"total_drives" example:"340"`
	TotalUpdates int `json:"total_updates" example:"5"`
}

type CarRefV1 struct {
	CarID   int        `json:"car_id"`
	CarName NullString `json:"car_name" swaggertype:"string"`
}

type UnitsLengthTempV1 struct {
	UnitsLength      string `json:"unit_of_length"`
	UnitsTemperature string `json:"unit_of_temperature"`
}

type BatteryHealthV1Envelope struct {
	Data BatteryHealthV1Data `json:"data"`
}

type BatteryHealthV1Data struct {
	Car            CarRefV1               `json:"car"`
	BatteryHealth  BatteryHealthV1Metrics `json:"battery_health"`
	TeslaMateUnits UnitsLengthTempV1      `json:"units"`
}

type BatteryHealthV1Metrics struct {
	MaxRange                float64 `json:"max_range"`
	CurrentRange            float64 `json:"current_range"`
	MaxCapacity             float64 `json:"max_capacity"`
	CurrentCapacity         float64 `json:"current_capacity"`
	RatedEfficiency         float64 `json:"rated_efficiency"`
	BatteryHealthPercentage float64 `json:"battery_health_percentage"`
}

type ChargesListV1Envelope struct {
	Data ChargesListV1Data `json:"data"`
}

type ChargesListV1Data struct {
	Car            CarRefV1           `json:"car"`
	Charges        []ChargeListItemV1 `json:"charges"`
	TeslaMateUnits UnitsLengthTempV1  `json:"units"`
}

type ChargeBatteryStartEndV1 struct {
	StartBatteryLevel int `json:"start_battery_level"`
	EndBatteryLevel   int `json:"end_battery_level"`
}

type ChargeRangeStartEndV1 struct {
	StartRange float64 `json:"start_range"`
	EndRange   float64 `json:"end_range"`
}

type ChargeListItemV1 struct {
	ChargeID          int                     `json:"charge_id"`
	StartDate         string                  `json:"start_date"`
	EndDate           string                  `json:"end_date"`
	Address           string                  `json:"address"`
	ChargeEnergyAdded float64                 `json:"charge_energy_added"`
	ChargeEnergyUsed  float64                 `json:"charge_energy_used"`
	Cost              float64                 `json:"cost"`
	DurationMin       int                     `json:"duration_min"`
	DurationStr       string                  `json:"duration_str"`
	BatteryDetails    ChargeBatteryStartEndV1 `json:"battery_details"`
	RangeIdeal        ChargeRangeStartEndV1   `json:"range_ideal"`
	RangeRated        ChargeRangeStartEndV1   `json:"range_rated"`
	OutsideTempAvg    float64                 `json:"outside_temp_avg"`
	Odometer          float64                 `json:"odometer"`
	Latitude          float64                 `json:"latitude"`
	Longitude         float64                 `json:"longitude"`
}

type ChargeDetailsV1Envelope struct {
	Data ChargeDetailsV1Data `json:"data"`
}

type ChargeDetailsV1Data struct {
	Car            CarRefV1           `json:"car"`
	Charge         ChargeDetailFullV1 `json:"charge"`
	TeslaMateUnits UnitsLengthTempV1  `json:"units"`
}

type ChargeChargerHardwareV1 struct {
	ChargerActualCurrent int `json:"charger_actual_current"`
	ChargerPhases        int `json:"charger_phases"`
	ChargerPilotCurrent  int `json:"charger_pilot_current"`
	ChargerPower         int `json:"charger_power"`
	ChargerVoltage       int `json:"charger_voltage"`
}

type ChargeFastChargerDetailV1 struct {
	FastChargerPresent bool       `json:"fast_charger_present"`
	FastChargerBrand   NullString `json:"fast_charger_brand" swaggertype:"string"`
	FastChargerType    string     `json:"fast_charger_type"`
}

type ChargeDetailBatteryInfoV1 struct {
	IdealBatteryRange    float64  `json:"ideal_battery_range"`
	RatedBatteryRange    float64  `json:"rated_battery_range"`
	BatteryHeater        bool     `json:"battery_heater"`
	BatteryHeaterOn      bool     `json:"battery_heater_on"`
	BatteryHeaterNoPower NullBool `json:"battery_heater_no_power" swaggertype:"boolean"`
}

type ChargeDetailRowV1 struct {
	DetailID             int                       `json:"detail_id"`
	Date                 string                    `json:"date"`
	BatteryLevel         int                       `json:"battery_level"`
	UsableBatteryLevel   int                       `json:"usable_battery_level"`
	ChargeEnergyAdded    float64                   `json:"charge_energy_added"`
	NotEnoughPowerToHeat NullBool                  `json:"not_enough_power_to_heat" swaggertype:"boolean"`
	ChargerDetails       ChargeChargerHardwareV1   `json:"charger_details"`
	BatteryInfo          ChargeDetailBatteryInfoV1 `json:"battery_info"`
	ConnChargeCable      string                    `json:"conn_charge_cable"`
	FastChargerInfo      ChargeFastChargerDetailV1 `json:"fast_charger_info"`
	OutsideTemp          float64                   `json:"outside_temp"`
}

type ChargeDetailFullV1 struct {
	ChargeID          int                     `json:"charge_id"`
	StartDate         string                  `json:"start_date"`
	EndDate           string                  `json:"end_date"`
	Address           string                  `json:"address"`
	ChargeEnergyAdded float64                 `json:"charge_energy_added"`
	ChargeEnergyUsed  float64                 `json:"charge_energy_used"`
	Cost              float64                 `json:"cost"`
	DurationMin       int                     `json:"duration_min"`
	DurationStr       string                  `json:"duration_str"`
	BatteryDetails    ChargeBatteryStartEndV1 `json:"battery_details"`
	RangeIdeal        ChargeRangeStartEndV1   `json:"range_ideal"`
	RangeRated        ChargeRangeStartEndV1   `json:"range_rated"`
	OutsideTempAvg    float64                 `json:"outside_temp_avg"`
	Odometer          float64                 `json:"odometer"`
	Latitude          float64                 `json:"latitude"`
	Longitude         float64                 `json:"longitude"`
	ChargeDetails     []ChargeDetailRowV1     `json:"charge_details"`
}

type CurrentChargeV1Envelope struct {
	Data CurrentChargeV1Data `json:"data"`
}

type CurrentChargeV1Data struct {
	Car            CarRefV1          `json:"car"`
	Charge         CurrentChargeV1   `json:"charge"`
	TeslaMateUnits UnitsLengthTempV1 `json:"units"`
}

type ChargeBatteryStartCurrentV1 struct {
	StartBatteryLevel   int `json:"start_battery_level"`
	CurrentBatteryLevel int `json:"current_battery_level"`
}

type ChargeRatedRangeProgressV1 struct {
	StartRange   float64 `json:"start_range"`
	CurrentRange float64 `json:"current_range"`
	AddedRange   float64 `json:"added_range"`
}

type ChargeFastChargerCurrentV1 struct {
	FastChargerPresent bool    `json:"fast_charger_present"`
	FastChargerBrand   *string `json:"fast_charger_brand,omitempty"`
	FastChargerType    *string `json:"fast_charger_type,omitempty"`
}

type CurrentChargeBatteryInfoV1 struct {
	RatedBatteryRange    float64  `json:"rated_battery_range"`
	BatteryHeater        bool     `json:"battery_heater"`
	BatteryHeaterOn      bool     `json:"battery_heater_on"`
	BatteryHeaterNoPower NullBool `json:"battery_heater_no_power" swaggertype:"boolean"`
}

type CurrentChargeDetailRowV1 struct {
	DetailID             int                        `json:"detail_id"`
	Date                 string                     `json:"date"`
	BatteryLevel         int                        `json:"battery_level"`
	UsableBatteryLevel   int                        `json:"usable_battery_level"`
	ChargeEnergyAdded    float64                    `json:"charge_energy_added"`
	NotEnoughPowerToHeat NullBool                   `json:"not_enough_power_to_heat" swaggertype:"boolean"`
	ChargerDetails       ChargeChargerHardwareV1    `json:"charger_details"`
	BatteryInfo          CurrentChargeBatteryInfoV1 `json:"battery_info"`
	ConnChargeCable      interface{}                `json:"conn_charge_cable,omitempty"`
	FastChargerInfo      ChargeFastChargerCurrentV1 `json:"fast_charger_info"`
	OutsideTemp          float64                    `json:"outside_temp"`
}

type CurrentChargeV1 struct {
	ChargeID          int                         `json:"charge_id"`
	StartDate         string                      `json:"start_date"`
	IsCharging        bool                        `json:"is_charging"`
	Address           string                      `json:"address"`
	ChargeEnergyAdded float64                     `json:"charge_energy_added"`
	Cost              float64                     `json:"cost"`
	DurationMin       int                         `json:"duration_min"`
	DurationStr       string                      `json:"duration_str"`
	BatteryDetails    ChargeBatteryStartCurrentV1 `json:"battery_details"`
	RatedRange        ChargeRatedRangeProgressV1  `json:"rated_range"`
	OutsideTempAvg    float64                     `json:"outside_temp_avg"`
	Odometer          float64                     `json:"odometer"`
	ChargeDetails     []CurrentChargeDetailRowV1  `json:"charge_details"`
}

type DrivesListV1Envelope struct {
	Data DrivesListV1Data `json:"data"`
}

type DrivesListV1Data struct {
	Car            CarRefV1          `json:"car"`
	Drives         []DriveListItemV1 `json:"drives"`
	TeslaMateUnits UnitsLengthTempV1 `json:"units"`
}

type DriveOdometerV1 struct {
	OdometerStart    float64 `json:"odometer_start"`
	OdometerEnd      float64 `json:"odometer_end"`
	OdometerDistance float64 `json:"odometer_distance"`
}

type DriveBatteryWindowV1 struct {
	StartUsableBatteryLevel int  `json:"start_usable_battery_level"`
	StartBatteryLevel       int  `json:"start_battery_level"`
	EndUsableBatteryLevel   int  `json:"end_usable_battery_level"`
	EndBatteryLevel         int  `json:"end_battery_level"`
	ReducedRange            bool `json:"reduced_range"`
	IsSufficientlyPrecise   bool `json:"is_sufficiently_precise"`
}

type DriveRangeIdealRatedV1 struct {
	StartRange float64 `json:"start_range"`
	EndRange   float64 `json:"end_range"`
	RangeDiff  float64 `json:"range_diff"`
}

type DriveListItemV1 struct {
	DriveID           int                    `json:"drive_id"`
	StartDate         string                 `json:"start_date"`
	EndDate           string                 `json:"end_date"`
	StartAddress      string                 `json:"start_address"`
	EndAddress        string                 `json:"end_address"`
	OdometerDetails   DriveOdometerV1        `json:"odometer_details"`
	DurationMin       int                    `json:"duration_min"`
	DurationStr       string                 `json:"duration_str"`
	SpeedMax          int                    `json:"speed_max"`
	SpeedAvg          float64                `json:"speed_avg"`
	PowerMax          int                    `json:"power_max"`
	PowerMin          int                    `json:"power_min"`
	BatteryDetails    DriveBatteryWindowV1   `json:"battery_details"`
	RangeIdeal        DriveRangeIdealRatedV1 `json:"range_ideal"`
	RangeRated        DriveRangeIdealRatedV1 `json:"range_rated"`
	OutsideTempAvg    float64                `json:"outside_temp_avg"`
	InsideTempAvg     float64                `json:"inside_temp_avg"`
	EnergyConsumedNet *float64               `json:"energy_consumed_net"`
	ConsumptionNet    *float64               `json:"consumption_net"`
}

type DriveDetailsV1Envelope struct {
	Data DriveDetailsV1Data `json:"data"`
}

type DriveDetailsV1Data struct {
	Car            CarRefV1          `json:"car"`
	Drive          DriveDetailFullV1 `json:"drive"`
	TeslaMateUnits UnitsLengthTempV1 `json:"units"`
}

type DrivePositionClimateV1 struct {
	InsideTemp           NullFloat64 `json:"inside_temp" swaggertype:"number"`
	OutsideTemp          NullFloat64 `json:"outside_temp" swaggertype:"number"`
	IsClimateOn          NullBool    `json:"is_climate_on" swaggertype:"boolean"`
	FanStatus            NullInt64   `json:"fan_status" swaggertype:"integer"`
	DriverTempSetting    NullFloat64 `json:"driver_temp_setting" swaggertype:"number"`
	PassengerTempSetting NullFloat64 `json:"passenger_temp_setting" swaggertype:"number"`
	IsRearDefrosterOn    NullBool    `json:"is_rear_defroster_on" swaggertype:"boolean"`
	IsFrontDefrosterOn   NullBool    `json:"is_front_defroster_on" swaggertype:"boolean"`
}

type DrivePositionBatteryV1 struct {
	EstBatteryRange      NullFloat64 `json:"est_battery_range" swaggertype:"number"`
	IdealBatteryRange    NullFloat64 `json:"ideal_battery_range" swaggertype:"number"`
	RatedBatteryRange    NullFloat64 `json:"rated_battery_range" swaggertype:"number"`
	BatteryHeater        NullBool    `json:"battery_heater" swaggertype:"boolean"`
	BatteryHeaterOn      NullBool    `json:"battery_heater_on" swaggertype:"boolean"`
	BatteryHeaterNoPower NullBool    `json:"battery_heater_no_power" swaggertype:"boolean"`
}

type DrivePositionRowV1 struct {
	DetailID           int                    `json:"detail_id"`
	Date               string                 `json:"date"`
	Latitude           float64                `json:"latitude"`
	Longitude          float64                `json:"longitude"`
	Speed              int                    `json:"speed"`
	Power              int                    `json:"power"`
	Odometer           float64                `json:"odometer"`
	BatteryLevel       int                    `json:"battery_level"`
	UsableBatteryLevel NullInt64              `json:"usable_battery_level" swaggertype:"integer"`
	Elevation          NullInt64              `json:"elevation" swaggertype:"integer"`
	ClimateInfo        DrivePositionClimateV1 `json:"climate_info"`
	BatteryInfo        DrivePositionBatteryV1 `json:"battery_info"`
}

type DriveDetailFullV1 struct {
	DriveID           int                    `json:"drive_id"`
	StartDate         string                 `json:"start_date"`
	EndDate           string                 `json:"end_date"`
	StartAddress      string                 `json:"start_address"`
	EndAddress        string                 `json:"end_address"`
	OdometerDetails   DriveOdometerV1        `json:"odometer_details"`
	DurationMin       int                    `json:"duration_min"`
	DurationStr       string                 `json:"duration_str"`
	SpeedMax          int                    `json:"speed_max"`
	SpeedAvg          float64                `json:"speed_avg"`
	PowerMax          int                    `json:"power_max"`
	PowerMin          int                    `json:"power_min"`
	BatteryDetails    DriveBatteryWindowV1   `json:"battery_details"`
	RangeIdeal        DriveRangeIdealRatedV1 `json:"range_ideal"`
	RangeRated        DriveRangeIdealRatedV1 `json:"range_rated"`
	OutsideTempAvg    float64                `json:"outside_temp_avg"`
	InsideTempAvg     float64                `json:"inside_temp_avg"`
	EnergyConsumedNet *float64               `json:"energy_consumed_net"`
	ConsumptionNet    *float64               `json:"consumption_net"`
	DriveDetails      []DrivePositionRowV1   `json:"drive_details"`
}

type EnabledCommandsV1Envelope struct {
	EnabledCommands []string `json:"enabled_commands"`
}

type TeslaPassthroughJSONBody map[string]interface{}

type UpdatesListV1Envelope struct {
	Data UpdatesListV1Data `json:"data"`
}

type UpdatesListV1Data struct {
	Car     CarRefV1            `json:"car"`
	Updates []UpdatesListItemV1 `json:"updates"`
}

type UpdatesListItemV1 struct {
	UpdateID  int    `json:"update_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Version   string `json:"version"`
}

type UnitsLengthTempPressureV1 struct {
	UnitsLength      string `json:"unit_of_length"`
	UnitsPressure    string `json:"unit_of_pressure"`
	UnitsTemperature string `json:"unit_of_temperature"`
}

type CarStatusV1Envelope struct {
	Data CarStatusV1Data `json:"data"`
}

type CarStatusV1Data struct {
	Car             CarRefV1                  `json:"car"`
	MQTTInformation CarMQTTStatusPayloadV1    `json:"status"`
	TeslaMateUnits  UnitsLengthTempPressureV1 `json:"units"`
}

type CarLocationV1 struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type CarGeodataV1 struct {
	Geofence  string        `json:"geofence"`
	Location  CarLocationV1 `json:"location"`
	Latitude  float64       `json:"latitude"`
	Longitude float64       `json:"longitude"`
}

type StatusBatteryV1 struct {
	EstBatteryRange    float64 `json:"est_battery_range"`
	RatedBatteryRange  float64 `json:"rated_battery_range"`
	IdealBatteryRange  float64 `json:"ideal_battery_range"`
	BatteryLevel       int     `json:"battery_level"`
	UsableBatteryLevel int     `json:"usable_battery_level"`
}

type StatusCarDetailsV1 struct {
	Model       string `json:"model"`
	TrimBadging string `json:"trim_badging"`
}

type StatusCarExteriorV1 struct {
	ExteriorColor string `json:"exterior_color"`
	SpoilerType   string `json:"spoiler_type"`
	WheelType     string `json:"wheel_type"`
}

type StatusCarFlagsV1 struct {
	Healthy                bool `json:"healthy"`
	Locked                 bool `json:"locked"`
	SentryMode             bool `json:"sentry_mode"`
	WindowsOpen            bool `json:"windows_open"`
	DoorsOpen              bool `json:"doors_open"`
	DriverFrontDoorOpen    bool `json:"driver_front_door_open"`
	DriverRearDoorOpen     bool `json:"driver_rear_door_open"`
	PassengerFrontDoorOpen bool `json:"passenger_front_door_open"`
	PassengerRearDoorOpen  bool `json:"passenger_rear_door_open"`
	TrunkOpen              bool `json:"trunk_open"`
	FrunkOpen              bool `json:"frunk_open"`
	IsUserPresent          bool `json:"is_user_present"`
	CenterDisplayState     int  `json:"center_display_state"`
}

type StatusVersionsV1 struct {
	Version         string `json:"version"`
	UpdateAvailable bool   `json:"update_available"`
	UpdateVersion   string `json:"update_version"`
}

type StatusChargingV1 struct {
	PluggedIn                  bool    `json:"plugged_in"`
	ChargingState              string  `json:"charging_state"`
	ChargeEnergyAdded          float64 `json:"charge_energy_added"`
	ChargeLimitSoc             int     `json:"charge_limit_soc"`
	ChargePortDoorOpen         bool    `json:"charge_port_door_open"`
	ChargerActualCurrent       float64 `json:"charger_actual_current"`
	ChargerPhases              int     `json:"charger_phases"`
	ChargerPower               float64 `json:"charger_power"`
	ChargerVoltage             int     `json:"charger_voltage"`
	ChargeCurrentRequest       int     `json:"charge_current_request"`
	ChargeCurrentRequestMax    int     `json:"charge_current_request_max"`
	ScheduledChargingStartTime string  `json:"scheduled_charging_start_time"`
	TimeToFullCharge           float64 `json:"time_to_full_charge"`
}

type StatusClimateV1 struct {
	IsClimateOn       bool    `json:"is_climate_on"`
	InsideTemp        float64 `json:"inside_temp"`
	OutsideTemp       float64 `json:"outside_temp"`
	IsPreconditioning bool    `json:"is_preconditioning"`
	ClimateKeeperMode string  `json:"climate_keeper_mode"`
}

type StatusActiveRouteV1 struct {
	Destination         string        `json:"destination"`
	EnergyAtArrival     int           `json:"energy_at_arrival"`
	DistanceToArrival   float64       `json:"distance_to_arrival"`
	MinutesToArrival    float64       `json:"minutes_to_arrival"`
	TrafficMinutesDelay float64       `json:"traffic_minutes_delay"`
	Location            CarLocationV1 `json:"location"`
}

type StatusDrivingV1 struct {
	ActiveRoute            StatusActiveRouteV1 `json:"active_route"`
	ActiveRouteDestination string              `json:"active_route_destination"`
	ActiveRouteLatitude    float64             `json:"active_route_latitude"`
	ActiveRouteLongitude   float64             `json:"active_route_longitude"`
	ShiftState             string              `json:"shift_state"`
	Power                  int                 `json:"power"`
	Speed                  int                 `json:"speed"`
	Heading                int                 `json:"heading"`
	Elevation              int                 `json:"elevation"`
}

type StatusTPMSV1 struct {
	TpmsPressureFL    float64 `json:"tpms_pressure_fl"`
	TpmsPressureFR    float64 `json:"tpms_pressure_fr"`
	TpmsPressureRL    float64 `json:"tpms_pressure_rl"`
	TpmsPressureRR    float64 `json:"tpms_pressure_rr"`
	TpmsSoftWarningFL bool    `json:"tpms_soft_warning_fl"`
	TpmsSoftWarningFR bool    `json:"tpms_soft_warning_fr"`
	TpmsSoftWarningRL bool    `json:"tpms_soft_warning_rl"`
	TpmsSoftWarningRR bool    `json:"tpms_soft_warning_rr"`
}

type CarMQTTStatusPayloadV1 struct {
	DisplayName     string              `json:"display_name"`
	State           string              `json:"state"`
	StateSince      string              `json:"state_since"`
	Odometer        float64             `json:"odometer"`
	CarStatus       StatusCarFlagsV1    `json:"car_status"`
	CarDetails      StatusCarDetailsV1  `json:"car_details"`
	CarExterior     StatusCarExteriorV1 `json:"car_exterior"`
	CarGeodata      CarGeodataV1        `json:"car_geodata"`
	CarVersions     StatusVersionsV1    `json:"car_versions"`
	DrivingDetails  StatusDrivingV1     `json:"driving_details"`
	ClimateDetails  StatusClimateV1     `json:"climate_details"`
	BatteryDetails  StatusBatteryV1     `json:"battery_details"`
	ChargingDetails StatusChargingV1    `json:"charging_details"`
	TpmsDetails     StatusTPMSV1        `json:"tpms_details"`
}
