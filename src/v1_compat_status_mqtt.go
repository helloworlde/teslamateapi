package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/thanhpk/randstr"
)

// statusInfo 保存单辆车的 MQTT 状态信息。
type statusInfo struct {
	MQTTDataDisplayName                string
	MQTTDataState                      string
	MQTTDataStateSince                 string
	MQTTDataHealthy                    bool
	MQTTDataVersion                    string
	MQTTDataUpdateAvailable            bool
	MQTTDataUpdateVersion              string
	MQTTDataModel                      string
	MQTTDataTrimBadging                string
	MQTTDataExteriorColor              string
	MQTTDataWheelType                  string
	MQTTDataSpoilerType                string
	MQTTDataGeofence                   string
	MQTTDataShiftState                 string
	MQTTDataPower                      int
	MQTTDataSpeed                      int
	MQTTDataHeading                    int
	MQTTDataElevation                  int
	MQTTDataLocked                     bool
	MQTTDataSentryMode                 bool
	MQTTDataWindowsOpen                bool
	MQTTDataDoorsOpen                  bool
	MQTTDataDriverFrontDoorOpen        bool
	MQTTDataDriverRearDoorOpen         bool
	MQTTDataPassengerFrontDoorOpen     bool
	MQTTDataPassengerRearDoorOpen      bool
	MQTTDataTrunkOpen                  bool
	MQTTDataFrunkOpen                  bool
	MQTTDataIsUserPresent              bool
	MQTTDataCenterDisplayState         int
	MQTTDataIsClimateOn                bool
	MQTTDataInsideTemp                 float64
	MQTTDataOutsideTemp                float64
	MQTTDataIsPreconditioning          bool
	MQTTDataClimateKeeperMode          string
	MQTTDataOdometer                   float64
	MQTTDataEstBatteryRange            float64
	MQTTDataRatedBatteryRange          float64
	MQTTDataIdealBatteryRange          float64
	MQTTDataBatteryLevel               int
	MQTTDataUsableBatteryLevel         int
	MQTTDataPluggedIn                  bool
	MQTTDataChargingState              string
	MQTTDataChargeEnergyAdded          float64
	MQTTDataChargeLimitSoc             int
	MQTTDataChargePortDoorOpen         bool
	MQTTDataChargerActualCurrent       float64
	MQTTDataChargerPhases              int
	MQTTDataChargerPower               float64
	MQTTDataChargerVoltage             int
	MQTTDataChargeCurrentRequest       int
	MQTTDataChargeCurrentRequestMax    int
	MQTTDataScheduledChargingStartTime string
	MQTTDataTimeToFullCharge           float64
	MQTTDataTpmsPressureFL             float64
	MQTTDataTpmsPressureFR             float64
	MQTTDataTpmsPressureRL             float64
	MQTTDataTpmsPressureRR             float64
	MQTTDataTpmsSoftWarningFL          bool
	MQTTDataTpmsSoftWarningFR          bool
	MQTTDataTpmsSoftWarningRL          bool
	MQTTDataTpmsSoftWarningRR          bool
	MQTTDataLocation                   statusInfoLocation
	MQTTDataActiveRoute                statusInfoActiveRoute
}

type statusInfoActiveRoute struct {
	Destination         string
	EnergyAtArrival     int
	DistanceToArrival   float64
	MinutesToArrival    float64
	TrafficMinutesDelay float64
	Location            statusInfoLocation
}
type statusInfoLocation struct {
	Latitude  float64
	Longitude float64
}

type statusCache struct {
	mqttDisabled  bool
	mqttConnected bool

	topicScan string // MQTT topic 扫描格式，用于解析车辆 ID 和状态字段。

	cache map[int]*statusInfo
	mu    sync.Mutex
}

func getMQTTNameSpace() (MQTTNameSpace string) {
	// 读取 MQTT 命名空间。
	MQTTNameSpace = getEnv("MQTT_NAMESPACE", "")
	if len(MQTTNameSpace) > 0 {
		MQTTNameSpace = ("/" + MQTTNameSpace)
	}
	return MQTTNameSpace
}

func startMQTT() (*statusCache, error) {
	s := statusCache{
		cache: make(map[int]*statusInfo),
	}
	// 读取 MQTT 禁用开关。
	s.mqttDisabled = getEnvAsBool("DISABLE_MQTT", false)
	if s.mqttDisabled {
		return nil, errors.New("[notice] TeslaMateAPICarsStatusV1 DISABLE_MQTT is set to true.. can not return status for car without mqtt")
	}

	// 设置可被环境变量覆盖的默认值。
	MQTTPort := 0
	MQTTProtocol := "tcp"

	// 构建 MQTT 连接参数。
	MQTTTLS := getEnvAsBool("MQTT_TLS", false)
	if MQTTTLS {
		MQTTPort = getEnvAsInt("MQTT_PORT", 8883)
		MQTTProtocol = "tls"
	} else {
		MQTTPort = getEnvAsInt("MQTT_PORT", 1883)
	}
	MQTTHost := getEnv("MQTT_HOST", "mosquitto")
	MQTTUser := getEnv("MQTT_USERNAME", "")
	MQTTPass := getEnv("MQTT_PASSWORD", "")
	MQTTClientId := getEnv("MQTT_CLIENTID", randstr.String(4))
	// MQTTInvCert := getEnvAsBool("MQTT_TLS_ACCEPT_INVALID_CERTS", false)

	// 构建 MQTT broker URL。
	// 格式：mqtt[s]://@host.domain[:port]
	mqttURL := fmt.Sprintf("%s://%s:%d", MQTTProtocol, MQTTHost, MQTTPort)

	// 创建 MQTT 客户端连接选项。
	opts := mqtt.NewClientOptions().AddBroker(mqttURL)
	// 设置通用 MQTT 客户端参数。
	opts.SetKeepAlive(2 * time.Second)               // 设置客户端 keepalive。
	opts.SetDefaultPublishHandler(s.newMessage)      // 设置默认消息处理器。
	opts.SetConnectionLostHandler(s.connectionLost)  // 记录连接丢失事件。
	opts.SetReconnectingHandler(reconnectingHandler) // 记录重连事件。
	opts.SetConnectionAttemptHandler(connectingHandler)
	opts.SetOnConnectHandler(s.connectedHandler)
	opts.SetPingTimeout(1 * time.Second)             // 设置客户端 ping 超时。
	opts.SetClientID("teslamateapi-" + MQTTClientId) // 设置 TeslaMateApi MQTT 客户端 ID。
	opts.SetCleanSession(true)                       // 断开连接时移除所有订阅。
	opts.SetOrderMatters(false)                      // 不要求消息有序，避免回调阻塞。
	opts.SetAutoReconnect(true)                      // 连接断开后自动重连。
	opts.AutoReconnect = true
	// 按需设置认证信息。
	if len(MQTTUser) > 0 {
		opts.SetUsername(MQTTUser)
	}
	if len(MQTTPass) > 0 {
		opts.SetPassword(MQTTPass)
	}

	// 使用连接选项创建 MQTT 连接。
	m := mqtt.NewClient(opts)
	if token := m.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("[error] TeslaMateAPICarsStatusV1 failed to connect to MQTT: %w", token.Error())
		// 可按需改用 opts.ConnectRetry 持续重试连接。
	}

	// 调试模式下记录 MQTT 连接成功信息。
	if gin.IsDebugging() {
		log.Println("[debug] TeslaMateAPICarsStatusV1 successfully connected to mqtt.")
	}

	s.topicScan = fmt.Sprintf("teslamate%s/cars/%%d/%%s", getMQTTNameSpace())

	// 使用 MQTT 时，连接成功后将 readyz 置为 true。
	isReady.Store(true)

	// 后续 MQTT 消息由 newMessage 处理。
	return &s, nil
}

func reconnectingHandler(c mqtt.Client, options *mqtt.ClientOptions) {
	log.Println("[info] mqtt reconnecting...")

}

func connectingHandler(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
	log.Println("[info] mqtt connecting...")
	return tlsCfg
}

func (s *statusCache) connectedHandler(c mqtt.Client) {
	log.Println("[info] mqtt connected.")
	s.mqttConnected = true

	// 订阅所有车辆状态 topic。
	topic := fmt.Sprintf("teslamate%s/cars/#", getMQTTNameSpace())
	if token := c.Subscribe(topic, 0, s.newMessage); token.Wait() && token.Error() != nil {
		log.Panic(token.Error()) // 可按需改用 opts.ConnectRetry 持续重试连接。
	}
	log.Println("[info] subscribed to: " + topic)

	// 使用 MQTT 时，订阅成功后将 readyz 置为 true。
	isReady.Store(true)
}

// connectionLost 在 MQTT 连接丢失时被客户端回调。
func (s *statusCache) connectionLost(c mqtt.Client, err error) {
	log.Println("[error] MQTT connection lost: " + err.Error())
	s.mqttConnected = false

	// 使用 MQTT 时，连接丢失后将 readyz 置为 false。
	isReady.Store(false)
}

// newMessage 在收到 MQTT 消息时被客户端回调。
func (s *statusCache) newMessage(c mqtt.Client, msg mqtt.Message) {
	//log.Println("[info] mqtt - 收到: " + string(msg.Topic()) + " 值: " + string(msg.Payload()))
	// topic 格式为 teslamateMQTT_NAMESPACE/cars/carID/display_name。
	var (
		carID     int
		MqttTopic string
	)
	_, err := fmt.Sscanf(msg.Topic(), s.topicScan, &carID, &MqttTopic)
	if err != nil {
		log.Printf("[warning] TeslaMateAPICarsStatusV1 unexpected topic format (%s) - ignoring message: %v", msg.Topic(), err)
		return
	}

	// 解析 topic 末尾的状态字段。
	s.mu.Lock()
	defer s.mu.Unlock()
	stat := s.cache[carID]
	if stat == nil {
		stat = &statusInfo{}
		s.cache[carID] = stat
	}

	//log.Printf(MqttTopic + " 设置为: " + string(msg.Payload()))
	// 根据 topic 字段写入状态缓存。
	switch MqttTopic {
	case "display_name":
		stat.MQTTDataDisplayName = string(msg.Payload())
	case "state":
		stat.MQTTDataState = string(msg.Payload())
	case "since":
		stat.MQTTDataStateSince = string(msg.Payload())
	case "healthy":
		stat.MQTTDataHealthy = convertStringToBool(string(msg.Payload()))
	case "version":
		stat.MQTTDataVersion = string(msg.Payload())
	case "update_available":
		stat.MQTTDataUpdateAvailable = convertStringToBool(string(msg.Payload()))
	case "update_version":
		stat.MQTTDataUpdateVersion = string(msg.Payload())
	case "model":
		stat.MQTTDataModel = string(msg.Payload())
	case "trim_badging":
		stat.MQTTDataTrimBadging = string(msg.Payload())
	case "exterior_color":
		stat.MQTTDataExteriorColor = string(msg.Payload())
	case "wheel_type":
		stat.MQTTDataWheelType = string(msg.Payload())
	case "spoiler_type":
		stat.MQTTDataSpoilerType = string(msg.Payload())
	case "geofence":
		stat.MQTTDataGeofence = string(msg.Payload())
	case "shift_state":
		stat.MQTTDataShiftState = string(msg.Payload())
	case "power":
		stat.MQTTDataPower = convertStringToInteger(string(msg.Payload()))
	case "speed":
		stat.MQTTDataSpeed = convertStringToInteger(string(msg.Payload()))
	case "heading":
		stat.MQTTDataHeading = convertStringToInteger(string(msg.Payload()))
	case "elevation":
		stat.MQTTDataElevation = convertStringToInteger(string(msg.Payload()))
	case "locked":
		stat.MQTTDataLocked = convertStringToBool(string(msg.Payload()))
	case "sentry_mode":
		stat.MQTTDataSentryMode = convertStringToBool(string(msg.Payload()))
	case "windows_open":
		stat.MQTTDataWindowsOpen = convertStringToBool(string(msg.Payload()))
	case "doors_open":
		stat.MQTTDataDoorsOpen = convertStringToBool(string(msg.Payload()))
	case "driver_front_door_open":
		stat.MQTTDataDriverFrontDoorOpen = convertStringToBool(string(msg.Payload()))
	case "driver_rear_door_open":
		stat.MQTTDataDriverRearDoorOpen = convertStringToBool(string(msg.Payload()))
	case "passenger_front_door_open":
		stat.MQTTDataPassengerFrontDoorOpen = convertStringToBool(string(msg.Payload()))
	case "passenger_rear_door_open":
		stat.MQTTDataPassengerRearDoorOpen = convertStringToBool(string(msg.Payload()))
	case "trunk_open":
		stat.MQTTDataTrunkOpen = convertStringToBool(string(msg.Payload()))
	case "frunk_open":
		stat.MQTTDataFrunkOpen = convertStringToBool(string(msg.Payload()))
	case "is_user_present":
		stat.MQTTDataIsUserPresent = convertStringToBool(string(msg.Payload()))
	case "center_display_state":
		stat.MQTTDataCenterDisplayState = convertStringToInteger(string(msg.Payload()))
	case "is_climate_on":
		stat.MQTTDataIsClimateOn = convertStringToBool(string(msg.Payload()))
	case "inside_temp":
		stat.MQTTDataInsideTemp = convertStringToFloat(string(msg.Payload()))
	case "outside_temp":
		stat.MQTTDataOutsideTemp = convertStringToFloat(string(msg.Payload()))
	case "is_preconditioning":
		stat.MQTTDataIsPreconditioning = convertStringToBool(string(msg.Payload()))
	case "climate_keeper_mode":
		stat.MQTTDataClimateKeeperMode = string(msg.Payload())
	case "odometer":
		stat.MQTTDataOdometer = convertStringToFloat(string(msg.Payload()))
	case "est_battery_range_km":
		stat.MQTTDataEstBatteryRange = convertStringToFloat(string(msg.Payload()))
	case "rated_battery_range_km":
		stat.MQTTDataRatedBatteryRange = convertStringToFloat(string(msg.Payload()))
	case "ideal_battery_range_km":
		stat.MQTTDataIdealBatteryRange = convertStringToFloat(string(msg.Payload()))
	case "battery_level":
		stat.MQTTDataBatteryLevel = convertStringToInteger(string(msg.Payload()))
	case "usable_battery_level":
		stat.MQTTDataUsableBatteryLevel = convertStringToInteger(string(msg.Payload()))
	case "plugged_in":
		stat.MQTTDataPluggedIn = convertStringToBool(string(msg.Payload()))
	case "charging_state":
		stat.MQTTDataChargingState = strings.ToLower(string(msg.Payload()))
	case "charge_energy_added":
		stat.MQTTDataChargeEnergyAdded = convertStringToFloat(string(msg.Payload()))
	case "charge_limit_soc":
		stat.MQTTDataChargeLimitSoc = convertStringToInteger(string(msg.Payload()))
	case "charge_port_door_open":
		stat.MQTTDataChargePortDoorOpen = convertStringToBool(string(msg.Payload()))
	case "charger_actual_current":
		stat.MQTTDataChargerActualCurrent = convertStringToFloat(string(msg.Payload()))
	case "charger_phases":
		stat.MQTTDataChargerPhases = convertStringToInteger(string(msg.Payload()))
	case "charger_power":
		stat.MQTTDataChargerPower = convertStringToFloat(string(msg.Payload()))
	case "charger_voltage":
		stat.MQTTDataChargerVoltage = convertStringToInteger(string(msg.Payload()))
	case "charge_current_request":
		stat.MQTTDataChargeCurrentRequest = convertStringToInteger(string(msg.Payload()))
	case "charge_current_request_max":
		stat.MQTTDataChargeCurrentRequestMax = convertStringToInteger(string(msg.Payload()))
	case "scheduled_charging_start_time":
		stat.MQTTDataScheduledChargingStartTime = string(msg.Payload())
	case "time_to_full_charge":
		stat.MQTTDataTimeToFullCharge = convertStringToFloat(string(msg.Payload()))
	case "tpms_pressure_fl":
		stat.MQTTDataTpmsPressureFL = convertStringToFloat(string(msg.Payload()))
	case "tpms_pressure_fr":
		stat.MQTTDataTpmsPressureFR = convertStringToFloat(string(msg.Payload()))
	case "tpms_pressure_rl":
		stat.MQTTDataTpmsPressureRL = convertStringToFloat(string(msg.Payload()))
	case "tpms_pressure_rr":
		stat.MQTTDataTpmsPressureRR = convertStringToFloat(string(msg.Payload()))
	case "tpms_soft_warning_fl":
		stat.MQTTDataTpmsSoftWarningFL = convertStringToBool(string(msg.Payload()))
	case "tpms_soft_warning_fr":
		stat.MQTTDataTpmsSoftWarningFR = convertStringToBool(string(msg.Payload()))
	case "tpms_soft_warning_rl":
		stat.MQTTDataTpmsSoftWarningRL = convertStringToBool(string(msg.Payload()))
	case "tpms_soft_warning_rr":
		stat.MQTTDataTpmsSoftWarningRR = convertStringToBool(string(msg.Payload()))

	case "location":
		var tmp struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		_ = json.Unmarshal(msg.Payload(), &tmp)
		stat.MQTTDataLocation = statusInfoLocation(tmp)

	case "active_route":
		var tmp struct {
			Destination         string  `json:"destination"`
			EnergyAtArrival     int     `json:"energy_at_arrival"`
			DistanceToArrival   float64 `json:"miles_to_arrival"`
			MinutesToArrival    float64 `json:"minutes_to_arrival"`
			TrafficMinutesDelay float64 `json:"traffic_minutes_delay"`
			Location            struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"location"`
		}
		_ = json.Unmarshal(msg.Payload(), &tmp)
		stat.MQTTDataActiveRoute.Destination = tmp.Destination
		stat.MQTTDataActiveRoute.EnergyAtArrival = tmp.EnergyAtArrival
		stat.MQTTDataActiveRoute.DistanceToArrival = milesToKilometers(tmp.DistanceToArrival)
		stat.MQTTDataActiveRoute.MinutesToArrival = tmp.MinutesToArrival
		stat.MQTTDataActiveRoute.TrafficMinutesDelay = tmp.TrafficMinutesDelay
		stat.MQTTDataActiveRoute.Location = statusInfoLocation(tmp.Location)

	// 已废弃字段。
	case "latitude", "longitude", "active_route_destination", "active_route_latitude", "active_route_longitude":
		// 保持兼容，不写入新数据。

	// 默认分支。
	default:
		log.Printf("[warning] TeslaMateAPICarsStatusV1 mqtt.MessageHandler issue.. extraction of data for %s not implemented!", MqttTopic)
	}
}

// TeslaMateAPICarsStatusV1 返回基于 MQTT 缓存的兼容车辆状态。
func (s *statusCache) TeslaMateAPICarsStatusV1(c *gin.Context) {
	if s.mqttDisabled {
		log.Println("[notice] TeslaMateAPICarsStatusV1 DISABLE_MQTT is set to true.. can not return status for car without mqtt!")
		TeslaMateAPIHandleOtherResponse(c, http.StatusNotImplemented, "TeslaMateAPICarsStatusV1", gin.H{"error": "mqtt disabled.. status not accessible!"})
		return
	}

	if !s.mqttConnected {
		log.Println("[notice] TeslaMateAPICarsStatusV1 mqtt is disconnected.. can not return status for car without mqtt!")
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsStatusV1", gin.H{"error": "mqtt disconnected.. status not accessible!"})
		return
	}

	// 从 URL 读取车辆 ID。
	carID := convertStringToInteger(c.Param("CarID"))

	// 查找车辆在 MQTT 缓存中的状态。
	s.mu.Lock()
	stat := s.cache[carID]
	s.mu.Unlock()

	if stat == nil {
		// 兼容旧接口，保持错误响应封装。
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatusV1", "no info on this car ID", "-")
		return
	}

	// 创建响应所需变量。
	var (
		CarData                                      CarRefV1
		MQTTInformationData                          CarMQTTStatusPayloadV1
		UnitsLength, UnitsPressure, UnitsTemperature string
	)

	// 从数据库读取车辆和单位信息。
	query := `
		SELECT
			id,
			name,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_pressure FROM settings LIMIT 1) as unit_of_pressure,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature
		FROM cars
		WHERE id=$1
		LIMIT 1;`
	err := db.QueryRow(query, carID).Scan(&CarData.CarID,
		&CarData.CarName,
		&UnitsLength,
		&UnitsPressure,
		&UnitsTemperature)

	// 检查查询错误，包括未找到车辆。
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatusV1", "Unable to load cars.", err.Error())
		return
	}

	// 将 MQTT 缓存数据写入响应字段。
	MQTTInformationData.DisplayName = stat.MQTTDataDisplayName
	MQTTInformationData.State = stat.MQTTDataState
	MQTTInformationData.StateSince = stat.MQTTDataStateSince
	MQTTInformationData.CarStatus.Healthy = stat.MQTTDataHealthy
	MQTTInformationData.CarVersions.Version = stat.MQTTDataVersion
	MQTTInformationData.CarVersions.UpdateAvailable = stat.MQTTDataUpdateAvailable
	MQTTInformationData.CarVersions.UpdateVersion = stat.MQTTDataUpdateVersion
	MQTTInformationData.CarDetails.Model = stat.MQTTDataModel
	MQTTInformationData.CarDetails.TrimBadging = stat.MQTTDataTrimBadging
	MQTTInformationData.CarExterior.ExteriorColor = stat.MQTTDataExteriorColor
	MQTTInformationData.CarExterior.WheelType = stat.MQTTDataWheelType
	MQTTInformationData.CarExterior.SpoilerType = stat.MQTTDataSpoilerType
	MQTTInformationData.CarGeodata.Geofence = stat.MQTTDataGeofence
	MQTTInformationData.CarGeodata.Location = CarLocationV1{Latitude: stat.MQTTDataLocation.Latitude, Longitude: stat.MQTTDataLocation.Longitude}
	MQTTInformationData.DrivingDetails.ActiveRoute.Destination = stat.MQTTDataActiveRoute.Destination
	MQTTInformationData.DrivingDetails.ActiveRoute.EnergyAtArrival = stat.MQTTDataActiveRoute.EnergyAtArrival
	MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival = stat.MQTTDataActiveRoute.DistanceToArrival
	MQTTInformationData.DrivingDetails.ActiveRoute.MinutesToArrival = stat.MQTTDataActiveRoute.MinutesToArrival
	MQTTInformationData.DrivingDetails.ActiveRoute.TrafficMinutesDelay = stat.MQTTDataActiveRoute.TrafficMinutesDelay
	MQTTInformationData.DrivingDetails.ActiveRoute.Location = CarLocationV1{Latitude: stat.MQTTDataActiveRoute.Location.Latitude, Longitude: stat.MQTTDataActiveRoute.Location.Longitude}
	MQTTInformationData.DrivingDetails.ShiftState = stat.MQTTDataShiftState
	MQTTInformationData.DrivingDetails.Power = stat.MQTTDataPower
	MQTTInformationData.DrivingDetails.Speed = stat.MQTTDataSpeed
	MQTTInformationData.DrivingDetails.Heading = stat.MQTTDataHeading
	MQTTInformationData.DrivingDetails.Elevation = stat.MQTTDataElevation
	MQTTInformationData.CarStatus.Locked = stat.MQTTDataLocked
	MQTTInformationData.CarStatus.SentryMode = stat.MQTTDataSentryMode
	MQTTInformationData.CarStatus.WindowsOpen = stat.MQTTDataWindowsOpen
	MQTTInformationData.CarStatus.DoorsOpen = stat.MQTTDataDoorsOpen
	MQTTInformationData.CarStatus.DriverFrontDoorOpen = stat.MQTTDataDriverFrontDoorOpen
	MQTTInformationData.CarStatus.DriverRearDoorOpen = stat.MQTTDataDriverRearDoorOpen
	MQTTInformationData.CarStatus.PassengerFrontDoorOpen = stat.MQTTDataPassengerFrontDoorOpen
	MQTTInformationData.CarStatus.PassengerRearDoorOpen = stat.MQTTDataPassengerRearDoorOpen
	MQTTInformationData.CarStatus.TrunkOpen = stat.MQTTDataTrunkOpen
	MQTTInformationData.CarStatus.FrunkOpen = stat.MQTTDataFrunkOpen
	MQTTInformationData.CarStatus.IsUserPresent = stat.MQTTDataIsUserPresent
	MQTTInformationData.CarStatus.CenterDisplayState = stat.MQTTDataCenterDisplayState
	MQTTInformationData.ClimateDetails.IsClimateOn = stat.MQTTDataIsClimateOn
	MQTTInformationData.ClimateDetails.InsideTemp = stat.MQTTDataInsideTemp
	MQTTInformationData.ClimateDetails.OutsideTemp = stat.MQTTDataOutsideTemp
	MQTTInformationData.ClimateDetails.IsPreconditioning = stat.MQTTDataIsPreconditioning
	MQTTInformationData.ClimateDetails.ClimateKeeperMode = stat.MQTTDataClimateKeeperMode
	MQTTInformationData.Odometer = stat.MQTTDataOdometer
	MQTTInformationData.BatteryDetails.EstBatteryRange = stat.MQTTDataEstBatteryRange
	MQTTInformationData.BatteryDetails.RatedBatteryRange = stat.MQTTDataRatedBatteryRange
	MQTTInformationData.BatteryDetails.IdealBatteryRange = stat.MQTTDataIdealBatteryRange
	MQTTInformationData.BatteryDetails.BatteryLevel = stat.MQTTDataBatteryLevel
	MQTTInformationData.BatteryDetails.UsableBatteryLevel = stat.MQTTDataUsableBatteryLevel
	MQTTInformationData.ChargingDetails.PluggedIn = stat.MQTTDataPluggedIn
	MQTTInformationData.ChargingDetails.ChargingState = stat.MQTTDataChargingState
	MQTTInformationData.ChargingDetails.ChargeEnergyAdded = stat.MQTTDataChargeEnergyAdded
	MQTTInformationData.ChargingDetails.ChargeLimitSoc = stat.MQTTDataChargeLimitSoc
	MQTTInformationData.ChargingDetails.ChargePortDoorOpen = stat.MQTTDataChargePortDoorOpen
	MQTTInformationData.ChargingDetails.ChargerActualCurrent = stat.MQTTDataChargerActualCurrent
	MQTTInformationData.ChargingDetails.ChargerPhases = stat.MQTTDataChargerPhases
	MQTTInformationData.ChargingDetails.ChargerPower = stat.MQTTDataChargerPower
	MQTTInformationData.ChargingDetails.ChargerVoltage = stat.MQTTDataChargerVoltage
	MQTTInformationData.ChargingDetails.ChargeCurrentRequest = stat.MQTTDataChargeCurrentRequest
	MQTTInformationData.ChargingDetails.ChargeCurrentRequestMax = stat.MQTTDataChargeCurrentRequestMax
	MQTTInformationData.ChargingDetails.ScheduledChargingStartTime = stat.MQTTDataScheduledChargingStartTime
	MQTTInformationData.ChargingDetails.TimeToFullCharge = stat.MQTTDataTimeToFullCharge
	MQTTInformationData.TpmsDetails.TpmsPressureFL = stat.MQTTDataTpmsPressureFL
	MQTTInformationData.TpmsDetails.TpmsPressureFR = stat.MQTTDataTpmsPressureFR
	MQTTInformationData.TpmsDetails.TpmsPressureRL = stat.MQTTDataTpmsPressureRL
	MQTTInformationData.TpmsDetails.TpmsPressureRR = stat.MQTTDataTpmsPressureRR
	MQTTInformationData.TpmsDetails.TpmsSoftWarningFL = stat.MQTTDataTpmsSoftWarningFL
	MQTTInformationData.TpmsDetails.TpmsSoftWarningFR = stat.MQTTDataTpmsSoftWarningFR
	MQTTInformationData.TpmsDetails.TpmsSoftWarningRL = stat.MQTTDataTpmsSoftWarningRL
	MQTTInformationData.TpmsDetails.TpmsSoftWarningRR = stat.MQTTDataTpmsSoftWarningRR

	// 为兼容旧客户端保留已废弃字段。
	MQTTInformationData.CarGeodata.Latitude = stat.MQTTDataLocation.Latitude
	MQTTInformationData.CarGeodata.Longitude = stat.MQTTDataLocation.Longitude
	MQTTInformationData.DrivingDetails.ActiveRouteDestination = stat.MQTTDataActiveRoute.Destination
	MQTTInformationData.DrivingDetails.ActiveRouteLatitude = stat.MQTTDataActiveRoute.Location.Latitude
	MQTTInformationData.DrivingDetails.ActiveRouteLongitude = stat.MQTTDataActiveRoute.Location.Longitude

	// 根据长度单位设置转换数值。
	if UnitsLength == "mi" {
		// 历史兼容示例：行程里程字段转换为英里。
		MQTTInformationData.Odometer = kilometersToMiles(MQTTInformationData.Odometer)
		MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival = kilometersToMiles(MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival)
		MQTTInformationData.DrivingDetails.Speed = kilometersToMilesInteger(MQTTInformationData.DrivingDetails.Speed)
		MQTTInformationData.BatteryDetails.EstBatteryRange = kilometersToMiles(MQTTInformationData.BatteryDetails.EstBatteryRange)
		MQTTInformationData.BatteryDetails.RatedBatteryRange = kilometersToMiles(MQTTInformationData.BatteryDetails.RatedBatteryRange)
		MQTTInformationData.BatteryDetails.IdealBatteryRange = kilometersToMiles(MQTTInformationData.BatteryDetails.IdealBatteryRange)
	}
	// 根据压力单位设置转换数值。
	if UnitsPressure == "psi" {
		MQTTInformationData.TpmsDetails.TpmsPressureFL = barToPsi(MQTTInformationData.TpmsDetails.TpmsPressureFL)
		MQTTInformationData.TpmsDetails.TpmsPressureFR = barToPsi(MQTTInformationData.TpmsDetails.TpmsPressureFR)
		MQTTInformationData.TpmsDetails.TpmsPressureRL = barToPsi(MQTTInformationData.TpmsDetails.TpmsPressureRL)
		MQTTInformationData.TpmsDetails.TpmsPressureRR = barToPsi(MQTTInformationData.TpmsDetails.TpmsPressureRR)
	}
	// 根据温度单位设置转换数值。
	if UnitsTemperature == "F" {
		MQTTInformationData.ClimateDetails.InsideTemp = celsiusToFahrenheit(MQTTInformationData.ClimateDetails.InsideTemp)
		MQTTInformationData.ClimateDetails.OutsideTemp = celsiusToFahrenheit(MQTTInformationData.ClimateDetails.OutsideTemp)
	}

	// 按用户配置时区转换时间字段。
	MQTTInformationData.StateSince = getTimeInTimeZone(MQTTInformationData.StateSince)
	MQTTInformationData.ChargingDetails.ScheduledChargingStartTime = getTimeInTimeZone(MQTTInformationData.ChargingDetails.ScheduledChargingStartTime)

	jsonData := CarStatusV1Envelope{
		Data: CarStatusV1Data{
			Car:             CarData,
			MQTTInformation: MQTTInformationData,
			TeslaMateUnits: UnitsLengthTempPressureV1{
				UnitsLength:      UnitsLength,
				UnitsPressure:    UnitsPressure,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// 返回响应数据。
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsStatusV1", jsonData)
}
