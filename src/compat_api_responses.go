package main

// APISystemMessageResponse 描述系统接口的简单成功响应。
type APISystemMessageResponse struct {
	Message string `json:"message" example:"pong"`
	Path    string `json:"path,omitempty" example:"/api/v1"`
}

// APISystemErrorBody 描述系统接口的简单错误响应。
type APISystemErrorBody struct {
	Error string `json:"error" example:"Unable to load summary."`
}

// CarsV1Envelope 描述 GET /v1/cars 与 GET /v1/cars/{CarID} 的兼容成功响应。
type CarsV1Envelope struct {
	Data CarsV1Data `json:"data"`
}

// CarsV1Data 是车辆列表和详情兼容接口的 data 对象。
type CarsV1Data struct {
	Cars []CarsV1Car `json:"cars"`
}

// CarsV1Car 描述兼容车辆接口返回的车辆对象。
type CarsV1Car struct {
	CarID            int                  `json:"car_id" example:"1"`
	Name             NullString           `json:"name" swaggertype:"string" example:"My Tesla"`
	CarDetails       CarsV1CarDetails     `json:"car_details"`
	CarExterior      CarsV1CarExterior    `json:"car_exterior"`
	CarSettings      CarsV1CarSettings    `json:"car_settings"`
	TeslaMateDetails CarsV1TeslaMateMeta  `json:"teslamate_details"`
	TeslaMateStats   CarsV1TeslaMateStats `json:"teslamate_stats"`
}

// CarsV1CarDetails 描述 Tesla 车辆稳定标识和静态元数据。
type CarsV1CarDetails struct {
	EID         int64       `json:"eid" example:"123456789"`
	VID         int64       `json:"vid" example:"987654321"`
	Vin         string      `json:"vin" example:"5YJ3E1EA1KF000000"`
	Model       NullString  `json:"model" swaggertype:"string" example:"Model 3"`
	TrimBadging NullString  `json:"trim_badging" swaggertype:"string" example:""`
	Efficiency  NullFloat64 `json:"efficiency" swaggertype:"number" example:"145"`
}

// CarsV1CarExterior 描述兼容接口返回的车辆外观信息。
type CarsV1CarExterior struct {
	ExteriorColor string `json:"exterior_color"`
	SpoilerType   string `json:"spoiler_type"`
	WheelType     string `json:"wheel_type"`
}

// CarsV1CarSettings 描述兼容接口返回的 TeslaMate 车辆设置。
type CarsV1CarSettings struct {
	SuspendMin          int  `json:"suspend_min" example:"15"`
	SuspendAfterIdleMin int  `json:"suspend_after_idle_min" example:"10"`
	ReqNotUnlocked      bool `json:"req_not_unlocked"`
	FreeSupercharging   bool `json:"free_supercharging"`
	UseStreamingAPI     bool `json:"use_streaming_api"`
}

// CarsV1TeslaMateMeta 描述兼容接口返回的 TeslaMate 创建和更新时间。
type CarsV1TeslaMateMeta struct {
	InsertedAt string `json:"inserted_at" example:"2026-04-01T12:00:00+08:00"`
	UpdatedAt  string `json:"updated_at" example:"2026-04-01T12:00:00+08:00"`
}

// CarsV1TeslaMateStats 描述兼容接口返回的单车聚合计数。
type CarsV1TeslaMateStats struct {
	TotalCharges int `json:"total_charges" example:"120"`
	TotalDrives  int `json:"total_drives" example:"340"`
	TotalUpdates int `json:"total_updates" example:"5"`
}

// CarRefV1 描述兼容接口通用车辆引用对象。
type CarRefV1 struct {
	CarID   int        `json:"car_id"`
	CarName NullString `json:"car_name" swaggertype:"string"`
}

// UnitsLengthTempV1 描述兼容接口返回的长度和温度单位。
type UnitsLengthTempV1 struct {
	UnitsLength      string `json:"unit_of_length"`
	UnitsTemperature string `json:"unit_of_temperature"`
}

// BatteryHealthV1Envelope 描述兼容电池健康接口响应。
type BatteryHealthV1Envelope struct {
	Data BatteryHealthV1Data `json:"data"`
}

// BatteryHealthV1Data 描述兼容电池健康接口的 data 对象。
type BatteryHealthV1Data struct {
	Car            CarRefV1               `json:"car"`
	BatteryHealth  BatteryHealthV1Metrics `json:"battery_health"`
	TeslaMateUnits UnitsLengthTempV1      `json:"units"`
}

// BatteryHealthV1Metrics 描述兼容电池健康指标。
type BatteryHealthV1Metrics struct {
	MaxRange                float64 `json:"max_range"`
	CurrentRange            float64 `json:"current_range"`
	MaxCapacity             float64 `json:"max_capacity"`
	CurrentCapacity         float64 `json:"current_capacity"`
	RatedEfficiency         float64 `json:"rated_efficiency"`
	BatteryHealthPercentage float64 `json:"battery_health_percentage"`
}

// ChargesListV1Envelope 描述兼容充电列表接口响应。
type ChargesListV1Envelope struct {
	Data ChargesListV1Data `json:"data"`
}

// ChargesListV1Data 描述兼容充电列表接口的 data 对象。
type ChargesListV1Data struct {
	Car            CarRefV1           `json:"car"`
	Charges        []ChargeListItemV1 `json:"charges"`
	TeslaMateUnits UnitsLengthTempV1  `json:"units"`
}

// ChargeBatteryStartEndV1 描述充电开始和结束时的电量。
type ChargeBatteryStartEndV1 struct {
	StartBatteryLevel int `json:"start_battery_level"`
	EndBatteryLevel   int `json:"end_battery_level"`
}

// ChargeRangeStartEndV1 描述充电开始和结束时的续航值。
type ChargeRangeStartEndV1 struct {
	StartRange float64 `json:"start_range"`
	EndRange   float64 `json:"end_range"`
}

// ChargeListItemV1 描述兼容充电列表中的一条记录。
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
	// ConnChargeCable：TeslaMate `charges.conn_charge_cable`（如 IEC、NEMA）；无记录时 JSON 省略该字段。
	ConnChargeCable *string `json:"conn_charge_cable,omitempty" swaggertype:"string"`
	// FastChargerBrand：TeslaMate `charges.fast_charger_brand`（直流快充品牌）；交流或非快充时常为空，JSON 省略；无效占位 `<invalid>` 不输出。
	FastChargerBrand *string `json:"fast_charger_brand,omitempty" swaggertype:"string"`
}

// ChargeDetailsV1Envelope 描述兼容充电详情接口响应。
type ChargeDetailsV1Envelope struct {
	Data ChargeDetailsV1Data `json:"data"`
}

// ChargeDetailsV1Data 描述兼容充电详情接口的 data 对象。
type ChargeDetailsV1Data struct {
	Car            CarRefV1           `json:"car"`
	Charge         ChargeDetailFullV1 `json:"charge"`
	TeslaMateUnits UnitsLengthTempV1  `json:"units"`
}

// ChargeChargerHardwareV1 描述充电采样中的充电设备读数。
type ChargeChargerHardwareV1 struct {
	ChargerActualCurrent int `json:"charger_actual_current"`
	ChargerPhases        int `json:"charger_phases"`
	ChargerPilotCurrent  int `json:"charger_pilot_current"`
	ChargerPower         int `json:"charger_power"`
	ChargerVoltage       int `json:"charger_voltage"`
}

// ChargeFastChargerDetailV1 描述充电详情采样中的快充信息。
type ChargeFastChargerDetailV1 struct {
	FastChargerPresent bool       `json:"fast_charger_present"`
	FastChargerBrand   NullString `json:"fast_charger_brand" swaggertype:"string"`
	FastChargerType    string     `json:"fast_charger_type"`
}

// ChargeDetailBatteryInfoV1 描述充电详情采样中的电池读数。
type ChargeDetailBatteryInfoV1 struct {
	IdealBatteryRange    float64  `json:"ideal_battery_range"`
	RatedBatteryRange    float64  `json:"rated_battery_range"`
	BatteryHeater        bool     `json:"battery_heater"`
	BatteryHeaterOn      bool     `json:"battery_heater_on"`
	BatteryHeaterNoPower NullBool `json:"battery_heater_no_power" swaggertype:"boolean"`
}

// ChargeDetailRowV1 描述兼容充电详情中的一条采样记录。
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

// ChargeDetailFullV1 描述兼容充电详情对象。
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

// CurrentChargeV1Envelope 描述兼容当前充电接口响应。
type CurrentChargeV1Envelope struct {
	Data CurrentChargeV1Data `json:"data"`
}

// CurrentChargeV1Data 描述兼容当前充电接口的 data 对象。
type CurrentChargeV1Data struct {
	Car            CarRefV1          `json:"car"`
	Charge         CurrentChargeV1   `json:"charge"`
	TeslaMateUnits UnitsLengthTempV1 `json:"units"`
}

// ChargeBatteryStartCurrentV1 描述充电开始和当前的电量。
type ChargeBatteryStartCurrentV1 struct {
	StartBatteryLevel   int `json:"start_battery_level"`
	CurrentBatteryLevel int `json:"current_battery_level"`
}

// ChargeRatedRangeProgressV1 描述当前充电过程中的额定续航变化。
type ChargeRatedRangeProgressV1 struct {
	StartRange   float64 `json:"start_range"`
	CurrentRange float64 `json:"current_range"`
	AddedRange   float64 `json:"added_range"`
}

// ChargeFastChargerCurrentV1 描述当前充电中的快充状态。
type ChargeFastChargerCurrentV1 struct {
	FastChargerPresent bool    `json:"fast_charger_present"`
	FastChargerBrand   *string `json:"fast_charger_brand,omitempty"`
	FastChargerType    *string `json:"fast_charger_type,omitempty"`
}

// CurrentChargeBatteryInfoV1 描述当前充电采样中的电池读数。
type CurrentChargeBatteryInfoV1 struct {
	RatedBatteryRange    float64  `json:"rated_battery_range"`
	BatteryHeater        bool     `json:"battery_heater"`
	BatteryHeaterOn      bool     `json:"battery_heater_on"`
	BatteryHeaterNoPower NullBool `json:"battery_heater_no_power" swaggertype:"boolean"`
}

// CurrentChargeDetailRowV1 描述兼容当前充电详情中的一条采样记录。
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

// CurrentChargeV1 描述兼容当前充电对象。
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

// DrivesListV1Envelope 描述兼容行程列表接口响应。
type DrivesListV1Envelope struct {
	Data DrivesListV1Data `json:"data"`
}

// DrivesListV1Data 描述兼容行程列表接口的 data 对象。
type DrivesListV1Data struct {
	Car            CarRefV1          `json:"car"`
	Drives         []DriveListItemV1 `json:"drives"`
	TeslaMateUnits UnitsLengthTempV1 `json:"units"`
}

// DriveOdometerV1 描述兼容行程中的里程表读数。
type DriveOdometerV1 struct {
	OdometerStart    float64 `json:"odometer_start"`
	OdometerEnd      float64 `json:"odometer_end"`
	OdometerDistance float64 `json:"odometer_distance"`
}

// DriveBatteryWindowV1 描述兼容行程中的电池读数。
type DriveBatteryWindowV1 struct {
	StartUsableBatteryLevel int  `json:"start_usable_battery_level"`
	StartBatteryLevel       int  `json:"start_battery_level"`
	EndUsableBatteryLevel   int  `json:"end_usable_battery_level"`
	EndBatteryLevel         int  `json:"end_battery_level"`
	ReducedRange            bool `json:"reduced_range"`
	IsSufficientlyPrecise   bool `json:"is_sufficiently_precise"`
}

// DriveRangeIdealRatedV1 描述兼容行程中的续航变化。
type DriveRangeIdealRatedV1 struct {
	StartRange float64 `json:"start_range"`
	EndRange   float64 `json:"end_range"`
	RangeDiff  float64 `json:"range_diff"`
}

// DriveListItemV1 描述兼容行程列表中的一条记录。
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

// DriveDetailsV1Envelope 描述兼容行程详情接口响应。
type DriveDetailsV1Envelope struct {
	Data DriveDetailsV1Data `json:"data"`
}

// DriveDetailsV1Data 描述兼容行程详情接口的 data 对象。
type DriveDetailsV1Data struct {
	Car            CarRefV1          `json:"car"`
	Drive          DriveDetailFullV1 `json:"drive"`
	TeslaMateUnits UnitsLengthTempV1 `json:"units"`
}

// DrivePositionClimateV1 描述行程位置采样中的空调读数。
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

// DrivePositionBatteryV1 描述行程位置采样中的电池读数。
type DrivePositionBatteryV1 struct {
	EstBatteryRange      NullFloat64 `json:"est_battery_range" swaggertype:"number"`
	IdealBatteryRange    NullFloat64 `json:"ideal_battery_range" swaggertype:"number"`
	RatedBatteryRange    NullFloat64 `json:"rated_battery_range" swaggertype:"number"`
	BatteryHeater        NullBool    `json:"battery_heater" swaggertype:"boolean"`
	BatteryHeaterOn      NullBool    `json:"battery_heater_on" swaggertype:"boolean"`
	BatteryHeaterNoPower NullBool    `json:"battery_heater_no_power" swaggertype:"boolean"`
}

// DrivePositionRowV1 描述兼容行程详情中的一条位置采样记录。
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

// DriveDetailFullV1 描述兼容行程详情对象。
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

// UpdatesListV1Envelope 描述兼容软件更新列表接口响应。
type UpdatesListV1Envelope struct {
	Data UpdatesListV1Data `json:"data"`
}

// UpdatesListV1Data 描述兼容软件更新列表接口的 data 对象。
type UpdatesListV1Data struct {
	Car     CarRefV1            `json:"car"`
	Updates []UpdatesListItemV1 `json:"updates"`
}

// UpdatesListItemV1 描述兼容软件更新历史中的一条记录。
type UpdatesListItemV1 struct {
	UpdateID  int    `json:"update_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Version   string `json:"version"`
}

// UnitsLengthTempPressureV1 描述兼容接口返回的长度、压力和温度单位。
type UnitsLengthTempPressureV1 struct {
	UnitsLength      string `json:"unit_of_length"`
	UnitsPressure    string `json:"unit_of_pressure"`
	UnitsTemperature string `json:"unit_of_temperature"`
}

// CarStatusV1Envelope 描述兼容车辆状态接口响应。
type CarStatusV1Envelope struct {
	Data CarStatusV1Data `json:"data"`
}

// CarStatusV1Data 描述兼容车辆状态接口的 data 对象。
type CarStatusV1Data struct {
	Car             CarRefV1                  `json:"car"`
	MQTTInformation CarMQTTStatusPayloadV1    `json:"status"`
	TeslaMateUnits  UnitsLengthTempPressureV1 `json:"units"`
}

// CarLocationV1 描述经纬度坐标。
type CarLocationV1 struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// CarGeodataV1 描述兼容接口返回的地理围栏和坐标数据。
type CarGeodataV1 struct {
	Geofence  string        `json:"geofence"`
	Location  CarLocationV1 `json:"location"`
	Latitude  float64       `json:"latitude"`
	Longitude float64       `json:"longitude"`
}

// StatusBatteryV1 描述兼容车辆状态中的电池读数。
type StatusBatteryV1 struct {
	EstBatteryRange    float64 `json:"est_battery_range"`
	RatedBatteryRange  float64 `json:"rated_battery_range"`
	IdealBatteryRange  float64 `json:"ideal_battery_range"`
	BatteryLevel       int     `json:"battery_level"`
	UsableBatteryLevel int     `json:"usable_battery_level"`
}

// StatusCarDetailsV1 描述兼容车辆状态中的车辆元数据。
type StatusCarDetailsV1 struct {
	Model       string `json:"model"`
	TrimBadging string `json:"trim_badging"`
}

// StatusCarExteriorV1 描述兼容车辆状态中的外观信息。
type StatusCarExteriorV1 struct {
	ExteriorColor string `json:"exterior_color"`
	SpoilerType   string `json:"spoiler_type"`
	WheelType     string `json:"wheel_type"`
}

// StatusCarFlagsV1 描述兼容车辆状态中的状态标记。
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

// StatusVersionsV1 描述兼容车辆状态中的软件版本信息。
type StatusVersionsV1 struct {
	Version         string `json:"version"`
	UpdateAvailable bool   `json:"update_available"`
	UpdateVersion   string `json:"update_version"`
}

// StatusChargingV1 描述兼容车辆状态中的充电字段。
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

// StatusClimateV1 描述兼容车辆状态中的空调字段。
type StatusClimateV1 struct {
	IsClimateOn       bool    `json:"is_climate_on"`
	InsideTemp        float64 `json:"inside_temp"`
	OutsideTemp       float64 `json:"outside_temp"`
	IsPreconditioning bool    `json:"is_preconditioning"`
	ClimateKeeperMode string  `json:"climate_keeper_mode"`
}

// StatusActiveRouteV1 描述兼容车辆状态中的当前导航路线。
type StatusActiveRouteV1 struct {
	Destination         string        `json:"destination"`
	EnergyAtArrival     int           `json:"energy_at_arrival"`
	DistanceToArrival   float64       `json:"distance_to_arrival"`
	MinutesToArrival    float64       `json:"minutes_to_arrival"`
	TrafficMinutesDelay float64       `json:"traffic_minutes_delay"`
	Location            CarLocationV1 `json:"location"`
}

// StatusDrivingV1 描述兼容车辆状态中的驾驶字段。
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

// StatusTPMSV1 描述兼容车辆状态中的胎压监测字段。
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

// CarMQTTStatusPayloadV1 描述由 MQTT 缓存支撑的兼容车辆状态载荷。
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
