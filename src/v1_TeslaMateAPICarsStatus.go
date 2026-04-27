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

type statusCache struct {
	mqttDisabled  bool
	mqttConnected bool

	topicScan string // scan parameter (expect it to generate car ID then relevant parameter)

	cache map[int]*statusInfo
	mu    sync.Mutex
}

func getMQTTNameSpace() (MQTTNameSpace string) {
	// adding MQTTNameSpace info
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
	// getting mqtt flag
	s.mqttDisabled = getEnvAsBool("DISABLE_MQTT", false)
	if s.mqttDisabled {
		return nil, errors.New("[notice] TeslaMateAPICarsStatusV1 DISABLE_MQTT is set to true.. can not return status for car without mqtt")
	}

	// default values that get might get overwritten..
	MQTTPort := 0
	MQTTProtocol := "tcp"

	// creating connection string towards mqtt
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
	isReady.Store(true)

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

func (s *statusCache) connectedHandler(c mqtt.Client) {
	log.Println("[info] mqtt connected.")
	s.mqttConnected = true

	// Subscribe - we will accept info on any car...
	topic := fmt.Sprintf("teslamate%s/cars/#", getMQTTNameSpace())
	if token := c.Subscribe(topic, 0, s.newMessage); token.Wait() && token.Error() != nil {
		log.Panic(token.Error()) // Note : May want to use opts.ConnectRetry which will keep trying the connection
	}
	log.Println("[info] subscribed to: " + topic)

	// setting readyz endpoint to true (when using MQTT)
	isReady.Store(true)
}

// connectionLost - called by mqtt package when the connection get lost
func (s *statusCache) connectionLost(c mqtt.Client, err error) {
	log.Println("[error] MQTT connection lost: " + err.Error())
	s.mqttConnected = false

	// setting readyz endpoint to false (when using MQTT)
	isReady.Store(false)
}

// newMessage - called by mqtt package when new message received
func (s *statusCache) newMessage(c mqtt.Client, msg mqtt.Message) {
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

	// deprecated
	case "latitude", "longitude", "active_route_destination", "active_route_latitude", "active_route_longitude":
		// doing nothing

	// default
	default:
		log.Printf("[warning] TeslaMateAPICarsStatusV1 mqtt.MessageHandler issue.. extraction of data for %s not implemented!", MqttTopic)
	}
}

// TeslaMateAPICarsStatusV1 func
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

	// getting CarID param from URL
	carID := convertStringToInteger(c.Param("CarID"))

	// Now see what data we have on the car
	s.mu.Lock()
	stat := s.cache[carID]
	s.mu.Unlock()

	if stat == nil {
		// or should it be http.StatusNoContent instead?
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatusV1", "no info on this car ID", "-")
		return
	}

	// creating required vars
	var (
		CarData                                      CarRefV1
		MQTTInformationData                          CarMQTTStatusPayloadV1
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
	err := db.QueryRow(query, carID).Scan(&CarData.CarID,
		&CarData.CarName,
		&UnitsLength,
		&UnitsPressure,
		&UnitsTemperature)

	// checking for errors in query (this will include no rows found)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatusV1", "Unable to load cars.", err.Error())
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

	// DEPRECATAD - setting values for deprecated fields
	MQTTInformationData.CarGeodata.Latitude = stat.MQTTDataLocation.Latitude
	MQTTInformationData.CarGeodata.Longitude = stat.MQTTDataLocation.Longitude
	MQTTInformationData.DrivingDetails.ActiveRouteDestination = stat.MQTTDataActiveRoute.Destination
	MQTTInformationData.DrivingDetails.ActiveRouteLatitude = stat.MQTTDataActiveRoute.Location.Latitude
	MQTTInformationData.DrivingDetails.ActiveRouteLongitude = stat.MQTTDataActiveRoute.Location.Longitude

	// converting values based of settings UnitsLength
	if UnitsLength == "mi" {
		// drive.OdometerDetails.OdometerStart = kilometersToMiles(drive.OdometerDetails.OdometerStart)
		MQTTInformationData.Odometer = kilometersToMiles(MQTTInformationData.Odometer)
		MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival = kilometersToMiles(MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival)
		MQTTInformationData.DrivingDetails.Speed = kilometersToMilesInteger(MQTTInformationData.DrivingDetails.Speed)
		MQTTInformationData.BatteryDetails.EstBatteryRange = kilometersToMiles(MQTTInformationData.BatteryDetails.EstBatteryRange)
		MQTTInformationData.BatteryDetails.RatedBatteryRange = kilometersToMiles(MQTTInformationData.BatteryDetails.RatedBatteryRange)
		MQTTInformationData.BatteryDetails.IdealBatteryRange = kilometersToMiles(MQTTInformationData.BatteryDetails.IdealBatteryRange)
	}
	// converting values based of settings UnitsPressure
	if UnitsPressure == "psi" {
		MQTTInformationData.TpmsDetails.TpmsPressureFL = barToPsi(MQTTInformationData.TpmsDetails.TpmsPressureFL)
		MQTTInformationData.TpmsDetails.TpmsPressureFR = barToPsi(MQTTInformationData.TpmsDetails.TpmsPressureFR)
		MQTTInformationData.TpmsDetails.TpmsPressureRL = barToPsi(MQTTInformationData.TpmsDetails.TpmsPressureRL)
		MQTTInformationData.TpmsDetails.TpmsPressureRR = barToPsi(MQTTInformationData.TpmsDetails.TpmsPressureRR)
	}
	// converting values based of settings UnitsTemperature
	if UnitsTemperature == "F" {
		MQTTInformationData.ClimateDetails.InsideTemp = celsiusToFahrenheit(MQTTInformationData.ClimateDetails.InsideTemp)
		MQTTInformationData.ClimateDetails.OutsideTemp = celsiusToFahrenheit(MQTTInformationData.ClimateDetails.OutsideTemp)
	}

	// adjusting to timezone differences from UTC to be userspecific
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

	// return jsonData
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsStatusV1", jsonData)
}
