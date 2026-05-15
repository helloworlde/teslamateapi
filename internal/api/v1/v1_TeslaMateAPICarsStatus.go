package v1

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
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
	"github.com/tobiasehlert/teslamateapi/internal/config"
	"github.com/tobiasehlert/teslamateapi/internal/conv"
	"github.com/tobiasehlert/teslamateapi/internal/nullx"
)

// statusInfo holds the status info for a car
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

type StatusCache struct {
	mqttDisabled  bool
	mqttConnected bool

	topicScan string // scan parameter (expect it to generate car ID then relevant parameter)

	cache map[int]*statusInfo
	mu    sync.Mutex
}

func getMQTTNameSpace() (MQTTNameSpace string) {
	// adding MQTTNameSpace info
	MQTTNameSpace = config.Env("MQTT_NAMESPACE", "")
	if len(MQTTNameSpace) > 0 {
		MQTTNameSpace = ("/" + MQTTNameSpace)
	}
	return MQTTNameSpace
}

func StartMQTT() (*StatusCache, error) {
	s := StatusCache{
		cache: make(map[int]*statusInfo),
	}
	// getting mqtt flag
	s.mqttDisabled = config.EnvAsBool("DISABLE_MQTT", false)
	if s.mqttDisabled {
		return nil, errors.New("[notice] TeslaMateAPICarsStatusV1 DISABLE_MQTT is set to true.. can not return status for car without mqtt")
	}

	// default values that get might get overwritten..
	MQTTPort := 0
	MQTTProtocol := "tcp"

	// creating connection string towards mqtt
	MQTTTLS := config.EnvAsBool("MQTT_TLS", false)
	if MQTTTLS {
		MQTTPort = config.EnvAsInt("MQTT_PORT", 8883)
		MQTTProtocol = "tls"
	} else {
		MQTTPort = config.EnvAsInt("MQTT_PORT", 1883)
	}
	MQTTHost := config.Env("MQTT_HOST", "mosquitto")
	MQTTUser := config.Env("MQTT_USERNAME", "")
	MQTTPass := config.Env("MQTT_PASSWORD", "")
	MQTTClientId := config.Env("MQTT_CLIENTID", randstr.String(4))
	// MQTTInvCert := config.EnvAsBool("MQTT_TLS_ACCEPT_INVALID_CERTS", false)

	// creating mqttURL to connect with
	// mqtt[s]://@host.domain[:port]
	mqttURL := fmt.Sprintf("%s://%s:%d", MQTTProtocol, MQTTHost, MQTTPort)

	// create options for the MQTT client connection
	opts := mqtt.NewClientOptions().AddBroker(mqttURL)
	// setting generic MQTT settings in opts
	opts.SetKeepAlive(2 * time.Second)               // setting keepalive for client
	opts.SetDefaultPublishHandler(s.newMessage)      // using f mqtt.MessageHandler function
	opts.SetConnectionLostHandler(s.connectionLost)  // Logs ConnectionLost events
	opts.SetReconnectingHandler(reconnectingHandler) // Logs reconnect events
	opts.SetConnectionAttemptHandler(connectingHandler)
	opts.SetOnConnectHandler(s.connectedHandler)
	opts.SetPingTimeout(1 * time.Second)             // setting pingtimeout for client
	opts.SetClientID("teslamateapi-" + MQTTClientId) // setting mqtt client id for TeslaMateApi
	opts.SetCleanSession(true)                       // removal of all subscriptions on disconnect
	opts.SetOrderMatters(false)                      // don't care about order (removes need for callbacks to return immediately)
	opts.SetAutoReconnect(true)                      // if connection drops automatically re-establish it
	opts.AutoReconnect = true
	// setting authentication if provided
	if len(MQTTUser) > 0 {
		opts.SetUsername(MQTTUser)
	}
	if len(MQTTPass) > 0 {
		opts.SetPassword(MQTTPass)
	}

	// creating MQTT connection with options
	m := mqtt.NewClient(opts)
	if token := m.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("[error] TeslaMateAPICarsStatusV1 failed to connect to MQTT: %w", token.Error())
		// Note : May want to use opts.ConnectRetry which will keep trying the connection
	}

	// showing mqtt successfully connected
	if gin.IsDebugging() {
		log.Println("[debug] TeslaMateAPICarsStatusV1 successfully connected to mqtt.")
	}

	s.topicScan = fmt.Sprintf("teslamate%s/cars/%%d/%%s", getMQTTNameSpace())

	// setting readyz endpoint to true (when using MQTT)
	apicommon.IsReady.Store(true)

	// Thats all - newMessage will be called when something new arrives
	return &s, nil
}

func reconnectingHandler(c mqtt.Client, options *mqtt.ClientOptions) {
	log.Println("[info] mqtt reconnecting...")

}

func connectingHandler(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
	log.Println("[info] mqtt connecting...")
	return tlsCfg
}

func (s *StatusCache) connectedHandler(c mqtt.Client) {
	log.Println("[info] mqtt connected.")
	s.mqttConnected = true

	// Subscribe - we will accept info on any car...
	topic := fmt.Sprintf("teslamate%s/cars/#", getMQTTNameSpace())
	if token := c.Subscribe(topic, 0, s.newMessage); token.Wait() && token.Error() != nil {
		log.Panic(token.Error()) // Note : May want to use opts.ConnectRetry which will keep trying the connection
	}
	log.Println("[info] subscribed to: " + topic)

	// setting readyz endpoint to true (when using MQTT)
	apicommon.IsReady.Store(true)
}

// connectionLost - called by mqtt package when the connection get lost
func (s *StatusCache) connectionLost(c mqtt.Client, err error) {
	log.Println("[error] MQTT connection lost: " + err.Error())
	s.mqttConnected = false

	// setting readyz endpoint to false (when using MQTT)
	apicommon.IsReady.Store(false)
}

// newMessage - called by mqtt package when new message received
func (s *StatusCache) newMessage(c mqtt.Client, msg mqtt.Message) {
	//log.Println("[info] mqtt - received: " + string(msg.Topic()) + " with value: " + string(msg.Payload()))
	// topic is in the format teslamateMQTT_NAMESPACE/cars/carID/display_name
	var (
		carID     int
		MqttTopic string
	)
	_, err := fmt.Sscanf(msg.Topic(), s.topicScan, &carID, &MqttTopic)
	if err != nil {
		log.Printf("[warning] TeslaMateAPICarsStatusV1 unexpected topic format (%s) - ignoring message: %v", msg.Topic(), err)
		return
	}

	// extracting the last part of topic
	s.mu.Lock()
	defer s.mu.Unlock()
	stat := s.cache[carID]
	if stat == nil {
		stat = &statusInfo{}
		s.cache[carID] = stat
	}

	//log.Printf(MqttTopic + " set to: " + string(msg.Payload()))
	// running if-else statements to collect data and put into overall vars..
	switch MqttTopic {
	case "display_name":
		stat.MQTTDataDisplayName = string(msg.Payload())
	case "state":
		stat.MQTTDataState = string(msg.Payload())
	case "since":
		stat.MQTTDataStateSince = string(msg.Payload())
	case "healthy":
		stat.MQTTDataHealthy = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "version":
		stat.MQTTDataVersion = string(msg.Payload())
	case "update_available":
		stat.MQTTDataUpdateAvailable = apicommon.ConvertStringToBool(string(msg.Payload()))
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
		stat.MQTTDataPower = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "speed":
		stat.MQTTDataSpeed = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "heading":
		stat.MQTTDataHeading = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "elevation":
		stat.MQTTDataElevation = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "locked":
		stat.MQTTDataLocked = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "sentry_mode":
		stat.MQTTDataSentryMode = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "windows_open":
		stat.MQTTDataWindowsOpen = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "doors_open":
		stat.MQTTDataDoorsOpen = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "driver_front_door_open":
		stat.MQTTDataDriverFrontDoorOpen = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "driver_rear_door_open":
		stat.MQTTDataDriverRearDoorOpen = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "passenger_front_door_open":
		stat.MQTTDataPassengerFrontDoorOpen = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "passenger_rear_door_open":
		stat.MQTTDataPassengerRearDoorOpen = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "trunk_open":
		stat.MQTTDataTrunkOpen = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "frunk_open":
		stat.MQTTDataFrunkOpen = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "is_user_present":
		stat.MQTTDataIsUserPresent = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "center_display_state":
		stat.MQTTDataCenterDisplayState = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "is_climate_on":
		stat.MQTTDataIsClimateOn = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "inside_temp":
		stat.MQTTDataInsideTemp = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "outside_temp":
		stat.MQTTDataOutsideTemp = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "is_preconditioning":
		stat.MQTTDataIsPreconditioning = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "climate_keeper_mode":
		stat.MQTTDataClimateKeeperMode = string(msg.Payload())
	case "odometer":
		stat.MQTTDataOdometer = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "est_battery_range_km":
		stat.MQTTDataEstBatteryRange = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "rated_battery_range_km":
		stat.MQTTDataRatedBatteryRange = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "ideal_battery_range_km":
		stat.MQTTDataIdealBatteryRange = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "battery_level":
		stat.MQTTDataBatteryLevel = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "usable_battery_level":
		stat.MQTTDataUsableBatteryLevel = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "plugged_in":
		stat.MQTTDataPluggedIn = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "charging_state":
		stat.MQTTDataChargingState = strings.ToLower(string(msg.Payload()))
	case "charge_energy_added":
		stat.MQTTDataChargeEnergyAdded = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "charge_limit_soc":
		stat.MQTTDataChargeLimitSoc = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "charge_port_door_open":
		stat.MQTTDataChargePortDoorOpen = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "charger_actual_current":
		stat.MQTTDataChargerActualCurrent = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "charger_phases":
		stat.MQTTDataChargerPhases = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "charger_power":
		stat.MQTTDataChargerPower = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "charger_voltage":
		stat.MQTTDataChargerVoltage = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "charge_current_request":
		stat.MQTTDataChargeCurrentRequest = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "charge_current_request_max":
		stat.MQTTDataChargeCurrentRequestMax = apicommon.ConvertStringToInteger(string(msg.Payload()))
	case "scheduled_charging_start_time":
		stat.MQTTDataScheduledChargingStartTime = string(msg.Payload())
	case "time_to_full_charge":
		stat.MQTTDataTimeToFullCharge = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "tpms_pressure_fl":
		stat.MQTTDataTpmsPressureFL = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "tpms_pressure_fr":
		stat.MQTTDataTpmsPressureFR = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "tpms_pressure_rl":
		stat.MQTTDataTpmsPressureRL = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "tpms_pressure_rr":
		stat.MQTTDataTpmsPressureRR = apicommon.ConvertStringToFloat(string(msg.Payload()))
	case "tpms_soft_warning_fl":
		stat.MQTTDataTpmsSoftWarningFL = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "tpms_soft_warning_fr":
		stat.MQTTDataTpmsSoftWarningFR = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "tpms_soft_warning_rl":
		stat.MQTTDataTpmsSoftWarningRL = apicommon.ConvertStringToBool(string(msg.Payload()))
	case "tpms_soft_warning_rr":
		stat.MQTTDataTpmsSoftWarningRR = apicommon.ConvertStringToBool(string(msg.Payload()))

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
		stat.MQTTDataActiveRoute.DistanceToArrival = conv.MiToKm(tmp.DistanceToArrival)
		stat.MQTTDataActiveRoute.MinutesToArrival = tmp.MinutesToArrival
		stat.MQTTDataActiveRoute.TrafficMinutesDelay = tmp.TrafficMinutesDelay
		stat.MQTTDataActiveRoute.Location = statusInfoLocation(tmp.Location)

	// deprecated
	case "latitude", "longitude", "active_route_destination", "active_route_latitude", "active_route_longitude":
		// doing nothing

	// default
	default:
		log.Printf("[warning] TeslaMateAPICarsStatusV1 mqtt.MessageHandler issue.. extraction of data for %s not implemented!", MqttTopic)
	}
}

// TeslaMateAPICarsStatusV1 godoc
//
// @Summary 车辆实时状态（MQTT）
// @Description 基于 MQTT 推送的实时车辆遥测；需启用 MQTT，未启用时返回 501。
// @Tags v1
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Success 200 {object} V1JSONEnvelope
// @Failure 200 {object} V1ErrorEnvelope
// @Failure 501 {object} V1ErrorEnvelope
// @Router /v1/cars/{CarID}/status [get]
func (s *StatusCache) TeslaMateAPICarsStatusV1(c *gin.Context) {
	if s.mqttDisabled {
		log.Println("[notice] TeslaMateAPICarsStatusV1 DISABLE_MQTT is set to true.. can not return status for car without mqtt!")
		apicommon.HandleOtherResponse(c, http.StatusNotImplemented, "TeslaMateAPICarsStatusV1", gin.H{"error": "mqtt disabled.. status not accessible!"})
		return
	}

	if !s.mqttConnected {
		log.Println("[notice] TeslaMateAPICarsStatusV1 mqtt is disconnected.. can not return status for car without mqtt!")
		apicommon.HandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsStatusV1", gin.H{"error": "mqtt disconnected.. status not accessible!"})
		return
	}

	// getting CarID param from URL
	carID := apicommon.ConvertStringToInteger(c.Param("CarID"))

	// Now see what data we have on the car
	s.mu.Lock()
	stat := s.cache[carID]
	s.mu.Unlock()

	if stat == nil {
		// or should it be http.StatusNoContent instead?
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsStatusV1", "no info on this car ID", "-")
		return
	}

	// creating structs for /cars
	// BatteryDetails 电池信息（MQTTInformation 子结构）
	type BatteryDetails struct {
		EstBatteryRange    float64 `json:"est_battery_range"`    // 估算续航里程（km）
		RatedBatteryRange  float64 `json:"rated_battery_range"`  // 额定续航里程（km）
		IdealBatteryRange  float64 `json:"ideal_battery_range"`  // 理想续航里程（km）
		BatteryLevel       int     `json:"battery_level"`        // 电量百分比
		UsableBatteryLevel int     `json:"usable_battery_level"` // 可用电量百分比
	}
	// CarDetails 车辆基本信息（MQTTInformation 子结构）
	type CarDetails struct {
		Model       string `json:"model"`        // 车型
		TrimBadging string `json:"trim_badging"` // 配置标识（如 P100D）
	}
	// CarExterior 车辆外观（MQTTInformation 子结构）
	type CarExterior struct {
		ExteriorColor string `json:"exterior_color"` // 外观颜色
		SpoilerType   string `json:"spoiler_type"`   // 尾翼类型
		WheelType     string `json:"wheel_type"`     // 轮毂类型
	}
	// CarLocation 经纬度坐标
	type CarLocation struct {
		Latitude  float64 `json:"latitude"`  // 最近一次上报的纬度
		Longitude float64 `json:"longitude"` // 最近一次上报的经度
	}
	// CarGeodata 地理位置（MQTTInformation 子结构）
	type CarGeodata struct {
		Geofence  string      `json:"geofence"`  // 当前位置所属地理围栏名称（若有）
		Location  CarLocation `json:"location"`  // 位置坐标
		Latitude  float64     `json:"latitude"`  // 已弃用：最近一次上报的纬度
		Longitude float64     `json:"longitude"` // 已弃用：最近一次上报的经度
	}
	// CarStatus 车辆状态（MQTTInformation 子结构）
	type CarStatus struct {
		Healthy                bool `json:"healthy"`                   // 该车辆 Logger 健康状态
		Locked                 bool `json:"locked"`                    // 是否上锁
		SentryMode             bool `json:"sentry_mode"`               // 哨兵模式是否开启
		WindowsOpen            bool `json:"windows_open"`              // 是否有车窗打开
		DoorsOpen              bool `json:"doors_open"`                // 是否有车门打开
		DriverFrontDoorOpen    bool `json:"driver_front_door_open"`    // 主驾前门是否打开
		DriverRearDoorOpen     bool `json:"driver_rear_door_open"`     // 主驾后门是否打开
		PassengerFrontDoorOpen bool `json:"passenger_front_door_open"` // 副驾前门是否打开
		PassengerRearDoorOpen  bool `json:"passenger_rear_door_open"`  // 副驾后门是否打开
		TrunkOpen              bool `json:"trunk_open"`                // 后备箱是否打开
		FrunkOpen              bool `json:"frunk_open"`                // 前备箱是否打开
		IsUserPresent          bool `json:"is_user_present"`           // 车内是否有人
		CenterDisplayState     int  `json:"center_display_state"`      // 中控屏状态
	}
	// CarVersions 软件版本（MQTTInformation 子结构）
	type CarVersions struct {
		Version         string `json:"version"`          // 当前软件版本
		UpdateAvailable bool   `json:"update_available"` // 是否有可用更新
		UpdateVersion   string `json:"update_version"`   // 待升级版本
	}
	// ChargingDetails 充电信息（MQTTInformation 子结构）
	type ChargingDetails struct {
		PluggedIn                  bool    `json:"plugged_in"`                    // 是否已插入充电器
		ChargingState              string  `json:"charging_state"`                // 充电状态
		ChargeEnergyAdded          float64 `json:"charge_energy_added"`           // 本次已补充电量（kWh）
		ChargeLimitSoc             int     `json:"charge_limit_soc"`              // 充电上限百分比
		ChargePortDoorOpen         bool    `json:"charge_port_door_open"`         // 充电口盖是否打开
		ChargerActualCurrent       float64 `json:"charger_actual_current"`        // 充电桩实际输出电流（A）
		ChargerPhases              int     `json:"charger_phases"`                // 充电相数（1-3）
		ChargerPower               float64 `json:"charger_power"`                 // 充电功率（kW）
		ChargerVoltage             int     `json:"charger_voltage"`               // 充电电压（V）
		ChargeCurrentRequest       int     `json:"charge_current_request"`        // 车辆请求电流（A）
		ChargeCurrentRequestMax    int     `json:"charge_current_request_max"`    // 车辆可接受最大电流（A）
		ScheduledChargingStartTime string  `json:"scheduled_charging_start_time"` // 计划充电开始时间
		TimeToFullCharge           float64 `json:"time_to_full_charge"`           // 预计充满剩余时长（小时）
	}
	// ClimateDetails 空调与温度（MQTTInformation 子结构）
	type ClimateDetails struct {
		IsClimateOn       bool    `json:"is_climate_on"`       // 空调是否开启
		InsideTemp        float64 `json:"inside_temp"`         // 车内温度（°C）
		OutsideTemp       float64 `json:"outside_temp"`        // 车外温度（°C）
		IsPreconditioning bool    `json:"is_preconditioning"`  // 是否正在预空调
		ClimateKeeperMode string  `json:"climate_keeper_mode"` // 留守模式（dog/camp 等）
	}
	// ActiveRouteDetails 当前导航路线（DrivingDetails 子结构）
	type ActiveRouteDetails struct {
		Destination         string      `json:"destination"`           // 导航目的地名称
		EnergyAtArrival     int         `json:"energy_at_arrival"`     // 到达时预计剩余电量（kWh）
		DistanceToArrival   float64     `json:"distance_to_arrival"`   // 到达剩余距离（km）
		MinutesToArrival    float64     `json:"minutes_to_arrival"`    // 到达剩余时长（分钟）
		TrafficMinutesDelay float64     `json:"traffic_minutes_delay"` // 交通延误（分钟）
		Location            CarLocation `json:"location"`              // 目的地坐标
	}
	// DrivingDetails 行驶信息（MQTTInformation 子结构）
	type DrivingDetails struct {
		ActiveRoute            ActiveRouteDetails `json:"active_route"`             // 当前导航路线
		ActiveRouteDestination string             `json:"active_route_destination"` // 已弃用：导航目的地
		ActiveRouteLatitude    float64            `json:"active_route_latitude"`    // 已弃用：目的地纬度
		ActiveRouteLongitude   float64            `json:"active_route_longitude"`   // 已弃用：目的地经度
		ShiftState             string             `json:"shift_state"`              // 档位状态（D/N/R/P）
		Power                  int                `json:"power"`                    // 当前电池功率（kW，正值放电、负值充电）
		Speed                  int                `json:"speed"`                    // 当前车速（km/h）
		Heading                int                `json:"heading"`                  // 行进方向角度
		Elevation              int                `json:"elevation"`                // 海拔高度（m）
	}
	// TpmsDetails 胎压详情（MQTTInformation 子结构）
	type TpmsDetails struct {
		TpmsPressureFL    float64 `json:"tpms_pressure_fl"`     // 左前轮胎压（BAR）
		TpmsPressureFR    float64 `json:"tpms_pressure_fr"`     // 右前轮胎压（BAR）
		TpmsPressureRL    float64 `json:"tpms_pressure_rl"`     // 左后轮胎压（BAR）
		TpmsPressureRR    float64 `json:"tpms_pressure_rr"`     // 右后轮胎压（BAR）
		TpmsSoftWarningFL bool    `json:"tpms_soft_warning_fl"` // 左前轮胎压软警告
		TpmsSoftWarningFR bool    `json:"tpms_soft_warning_fr"` // 右前轮胎压软警告
		TpmsSoftWarningRL bool    `json:"tpms_soft_warning_rl"` // 左后轮胎压软警告
		TpmsSoftWarningRR bool    `json:"tpms_soft_warning_rr"` // 右后轮胎压软警告
	}
	// MQTTInformation 实时遥测信息（Cars 子结构）
	type MQTTInformation struct {
		DisplayName     string          `json:"display_name"`     // 车辆名称
		State           string          `json:"state"`            // 车辆状态（online/asleep/charging 等）
		StateSince      string          `json:"state_since"`      // 上次状态变化时间
		Odometer        float64         `json:"odometer"`         // 里程表读数（km）
		CarStatus       CarStatus       `json:"car_status"`       // 车辆状态
		CarDetails      CarDetails      `json:"car_details"`      // 车辆基本信息
		CarExterior     CarExterior     `json:"car_exterior"`     // 车辆外观
		CarGeodata      CarGeodata      `json:"car_geodata"`      // 地理位置
		CarVersions     CarVersions     `json:"car_versions"`     // 软件版本
		DrivingDetails  DrivingDetails  `json:"driving_details"`  // 行驶信息
		ClimateDetails  ClimateDetails  `json:"climate_details"`  // 空调与温度
		BatteryDetails  BatteryDetails  `json:"battery_details"`  // 电池信息
		ChargingDetails ChargingDetails `json:"charging_details"` // 充电信息
		TpmsDetails     TpmsDetails     `json:"tpms_details"`     // 胎压信息
	}
	// Car 车辆主信息（Data 子结构）
	type Car struct {
		CarID   int          `json:"car_id"`   // 车辆 ID
		CarName nullx.String `json:"car_name"` // 车辆名称（可空）
	}
	// TeslaMateUnits 单位偏好（Data 子结构）
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`      // 长度单位
		UnitsPressure    string `json:"unit_of_pressure"`    // 压力单位
		UnitsTemperature string `json:"unit_of_temperature"` // 温度单位
	}
	// Data 响应数据主体（JSONData 子结构）
	type Data struct {
		Car             Car             `json:"car"`
		MQTTInformation MQTTInformation `json:"status"`
		TeslaMateUnits  TeslaMateUnits  `json:"units"`
	}
	// JSONData 响应外层包装
	type JSONData struct {
		Data Data `json:"data"`
	}

	// creating required vars
	var (
		CarData                                      Car
		MQTTInformationData                          MQTTInformation
		UnitsLength, UnitsPressure, UnitsTemperature string
	)

	// getting data from database (assume that carID is unique!)
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
	err := apicommon.DB.QueryRow(query, carID).Scan(&CarData.CarID,
		&CarData.CarName,
		&UnitsLength,
		&UnitsPressure,
		&UnitsTemperature)

	// checking for errors in query (this will include no rows found)
	if err != nil {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsStatusV1", "Unable to load cars.", err.Error())
		return
	}

	// setting data from MQTT into data fields to return
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
	MQTTInformationData.CarGeodata.Location = CarLocation(stat.MQTTDataLocation)
	MQTTInformationData.DrivingDetails.ActiveRoute.Destination = stat.MQTTDataActiveRoute.Destination
	MQTTInformationData.DrivingDetails.ActiveRoute.EnergyAtArrival = stat.MQTTDataActiveRoute.EnergyAtArrival
	MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival = stat.MQTTDataActiveRoute.DistanceToArrival
	MQTTInformationData.DrivingDetails.ActiveRoute.MinutesToArrival = stat.MQTTDataActiveRoute.MinutesToArrival
	MQTTInformationData.DrivingDetails.ActiveRoute.TrafficMinutesDelay = stat.MQTTDataActiveRoute.TrafficMinutesDelay
	MQTTInformationData.DrivingDetails.ActiveRoute.Location = CarLocation(stat.MQTTDataActiveRoute.Location)
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

	// DEPRECATAD - setting values for deprecated fields
	MQTTInformationData.CarGeodata.Latitude = stat.MQTTDataLocation.Latitude
	MQTTInformationData.CarGeodata.Longitude = stat.MQTTDataLocation.Longitude
	MQTTInformationData.DrivingDetails.ActiveRouteDestination = stat.MQTTDataActiveRoute.Destination
	MQTTInformationData.DrivingDetails.ActiveRouteLatitude = stat.MQTTDataActiveRoute.Location.Latitude
	MQTTInformationData.DrivingDetails.ActiveRouteLongitude = stat.MQTTDataActiveRoute.Location.Longitude

	// converting values based of settings UnitsLength
	if UnitsLength == "mi" {
		// drive.OdometerDetails.OdometerStart = conv.KmToMi(drive.OdometerDetails.OdometerStart)
		MQTTInformationData.Odometer = conv.KmToMi(MQTTInformationData.Odometer)
		MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival = conv.KmToMi(MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival)
		MQTTInformationData.DrivingDetails.Speed = conv.KmToMiInt(MQTTInformationData.DrivingDetails.Speed)
		MQTTInformationData.BatteryDetails.EstBatteryRange = conv.KmToMi(MQTTInformationData.BatteryDetails.EstBatteryRange)
		MQTTInformationData.BatteryDetails.RatedBatteryRange = conv.KmToMi(MQTTInformationData.BatteryDetails.RatedBatteryRange)
		MQTTInformationData.BatteryDetails.IdealBatteryRange = conv.KmToMi(MQTTInformationData.BatteryDetails.IdealBatteryRange)
	}
	// converting values based of settings UnitsPressure
	if UnitsPressure == "psi" {
		MQTTInformationData.TpmsDetails.TpmsPressureFL = conv.BarToPsi(MQTTInformationData.TpmsDetails.TpmsPressureFL)
		MQTTInformationData.TpmsDetails.TpmsPressureFR = conv.BarToPsi(MQTTInformationData.TpmsDetails.TpmsPressureFR)
		MQTTInformationData.TpmsDetails.TpmsPressureRL = conv.BarToPsi(MQTTInformationData.TpmsDetails.TpmsPressureRL)
		MQTTInformationData.TpmsDetails.TpmsPressureRR = conv.BarToPsi(MQTTInformationData.TpmsDetails.TpmsPressureRR)
	}
	// converting values based of settings UnitsTemperature
	if UnitsTemperature == "F" {
		MQTTInformationData.ClimateDetails.InsideTemp = conv.CelsiusToFahrenheit(MQTTInformationData.ClimateDetails.InsideTemp)
		MQTTInformationData.ClimateDetails.OutsideTemp = conv.CelsiusToFahrenheit(MQTTInformationData.ClimateDetails.OutsideTemp)
	}

	// adjusting to timezone differences from UTC to be userspecific
	MQTTInformationData.StateSince = apicommon.GetTimeInTimeZone(MQTTInformationData.StateSince)
	MQTTInformationData.ChargingDetails.ScheduledChargingStartTime = apicommon.GetTimeInTimeZone(MQTTInformationData.ChargingDetails.ScheduledChargingStartTime)

	//
	// build the data-blob
	jsonData := JSONData{
		Data{
			Car:             CarData,
			MQTTInformation: MQTTInformationData,
			TeslaMateUnits: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsPressure:    UnitsPressure,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// return jsonData
	apicommon.HandleSuccessResponse(c, "TeslaMateAPICarsStatusV1", jsonData)
}
