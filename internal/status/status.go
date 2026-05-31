// Package status owns the MQTT subscription that backs the v1
// /cars/:CarID/status endpoint. The Cache type is the shared state: MQTT
// updates write into it, the HTTP handler reads from it.
package status

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"

	"github.com/tobiasehlert/teslamateapi/internal/config"
	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/metrics"
)

// Info holds the latest MQTT-derived snapshot for a car.
type Info struct {
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
	MQTTDataLocation                   InfoLocation
	MQTTDataActiveRoute                InfoActiveRoute
}

// InfoActiveRoute is the route-aware subset of Info.
type InfoActiveRoute struct {
	Destination         string
	EnergyAtArrival     int
	DistanceToArrival   float64
	MinutesToArrival    float64
	TrafficMinutesDelay float64
	Location            InfoLocation
}

// InfoLocation is a {lat,lon} pair.
type InfoLocation struct {
	Latitude  float64
	Longitude float64
}

// Cache is the shared MQTT-backed status cache.
type Cache struct {
	mqttDisabled bool
	// mqttConnected is flipped by the paho callback goroutines and read by
	// the HTTP request goroutine — must be accessed atomically.
	mqttConnected atomic.Bool

	topicScan string

	cache map[int]*Info
	mu    sync.Mutex

	ready *atomic.Value
}

// Disabled reports whether MQTT is configured off.
func (c *Cache) Disabled() bool { return c.mqttDisabled }

// Connected reports whether the MQTT broker is currently connected.
func (c *Cache) Connected() bool { return c.mqttConnected.Load() }

// Get returns a snapshot of the cached Info for the given car, or nil if
// no message has been received yet. Callers must not mutate the returned
// pointer.
func (c *Cache) Get(carID int) *Info {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache[carID]
}

func mqttNamespace(ns string) string {
	if len(ns) > 0 {
		return "/" + ns
	}
	return ""
}

// New starts the MQTT client (when enabled) and returns a Cache. The ready
// pointer is flipped to true once the broker connects (or immediately when
// MQTT is disabled, by the caller).
//
// When DISABLE_MQTT=true, returns a non-nil Cache with Disabled()==true and
// a nil error so the caller doesn't have to disambiguate "configured off"
// from "connection failed".
func New(cfg config.Config, ready *atomic.Value) (*Cache, error) {
	c := &Cache{
		cache:        make(map[int]*Info),
		mqttDisabled: cfg.MQTTDisabled,
		ready:        ready,
	}
	if c.mqttDisabled {
		return c, nil
	}

	mqttProtocol := "tcp"
	if cfg.MQTTTLS {
		mqttProtocol = "tls"
	}
	mqttURL := fmt.Sprintf("%s://%s:%d", mqttProtocol, cfg.MQTTHost, cfg.MQTTPort)

	clientID := cfg.MQTTClientID
	if clientID == "" {
		clientID = randstr.String(4)
	}

	opts := mqtt.NewClientOptions().AddBroker(mqttURL)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetDefaultPublishHandler(c.newMessage)
	opts.SetConnectionLostHandler(c.connectionLost)
	opts.SetReconnectingHandler(reconnectingHandler)
	opts.SetConnectionAttemptHandler(connectingHandler)
	opts.SetOnConnectHandler(c.connectedHandler)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetClientID("teslamateapi-" + clientID)
	opts.SetCleanSession(true)
	opts.SetOrderMatters(false)
	opts.SetAutoReconnect(true)
	opts.AutoReconnect = true
	if cfg.MQTTUsername != "" {
		opts.SetUsername(cfg.MQTTUsername)
	}
	if cfg.MQTTPassword != "" {
		opts.SetPassword(cfg.MQTTPassword)
	}

	m := mqtt.NewClient(opts)
	if token := m.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("[error] TeslaMateAPICarsStatusV1 failed to connect to MQTT: %w", token.Error())
	}

	if gin.IsDebugging() {
		log.Println("[debug] TeslaMateAPICarsStatusV1 successfully connected to mqtt.")
	}

	c.topicScan = fmt.Sprintf("teslamate%s/cars/%%d/%%s", mqttNamespace(cfg.MQTTNamespace))
	if c.ready != nil {
		c.ready.Store(true)
	}
	return c, nil
}

func reconnectingHandler(_ mqtt.Client, _ *mqtt.ClientOptions) {
	log.Println("[info] mqtt reconnecting...")
}

func connectingHandler(_ *url.URL, tlsCfg *tls.Config) *tls.Config {
	log.Println("[info] mqtt connecting...")
	return tlsCfg
}

func (c *Cache) connectedHandler(client mqtt.Client) {
	log.Println("[info] mqtt connected.")
	c.mqttConnected.Store(true)
	metrics.SetMQTTConnected(metrics.MQTTConnectedState)

	topic := strings.Replace(c.topicScan, "/cars/%d/%s", "/cars/#", 1)
	if token := client.Subscribe(topic, 0, c.newMessage); token.Wait() && token.Error() != nil {
		log.Panic(token.Error())
	}
	log.Println("[info] subscribed to: " + topic)

	if c.ready != nil {
		c.ready.Store(true)
	}
}

func (c *Cache) connectionLost(_ mqtt.Client, err error) {
	log.Println("[error] MQTT connection lost: " + err.Error())
	c.mqttConnected.Store(false)
	metrics.SetMQTTConnected(metrics.MQTTDisconnected)
	if c.ready != nil {
		c.ready.Store(false)
	}
}

func (c *Cache) newMessage(_ mqtt.Client, msg mqtt.Message) {
	var (
		carID     int
		mqttTopic string
	)
	_, err := fmt.Sscanf(msg.Topic(), c.topicScan, &carID, &mqttTopic)
	if err != nil {
		log.Printf("[warning] TeslaMateAPICarsStatusV1 unexpected topic format (%s) - ignoring message: %v", msg.Topic(), err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	stat := c.cache[carID]
	if stat == nil {
		stat = &Info{}
		c.cache[carID] = stat
	}

	switch mqttTopic {
	case "display_name":
		stat.MQTTDataDisplayName = string(msg.Payload())
	case "state":
		stat.MQTTDataState = string(msg.Payload())
	case "since":
		stat.MQTTDataStateSince = string(msg.Payload())
	case "healthy":
		stat.MQTTDataHealthy = convert.StrToBool(string(msg.Payload()))
	case "version":
		stat.MQTTDataVersion = string(msg.Payload())
	case "update_available":
		stat.MQTTDataUpdateAvailable = convert.StrToBool(string(msg.Payload()))
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
		stat.MQTTDataPower = convert.StrToInt(string(msg.Payload()))
	case "speed":
		stat.MQTTDataSpeed = convert.StrToInt(string(msg.Payload()))
	case "heading":
		stat.MQTTDataHeading = convert.StrToInt(string(msg.Payload()))
	case "elevation":
		stat.MQTTDataElevation = convert.StrToInt(string(msg.Payload()))
	case "locked":
		stat.MQTTDataLocked = convert.StrToBool(string(msg.Payload()))
	case "sentry_mode":
		stat.MQTTDataSentryMode = convert.StrToBool(string(msg.Payload()))
	case "windows_open":
		stat.MQTTDataWindowsOpen = convert.StrToBool(string(msg.Payload()))
	case "doors_open":
		stat.MQTTDataDoorsOpen = convert.StrToBool(string(msg.Payload()))
	case "driver_front_door_open":
		stat.MQTTDataDriverFrontDoorOpen = convert.StrToBool(string(msg.Payload()))
	case "driver_rear_door_open":
		stat.MQTTDataDriverRearDoorOpen = convert.StrToBool(string(msg.Payload()))
	case "passenger_front_door_open":
		stat.MQTTDataPassengerFrontDoorOpen = convert.StrToBool(string(msg.Payload()))
	case "passenger_rear_door_open":
		stat.MQTTDataPassengerRearDoorOpen = convert.StrToBool(string(msg.Payload()))
	case "trunk_open":
		stat.MQTTDataTrunkOpen = convert.StrToBool(string(msg.Payload()))
	case "frunk_open":
		stat.MQTTDataFrunkOpen = convert.StrToBool(string(msg.Payload()))
	case "is_user_present":
		stat.MQTTDataIsUserPresent = convert.StrToBool(string(msg.Payload()))
	case "center_display_state":
		stat.MQTTDataCenterDisplayState = convert.StrToInt(string(msg.Payload()))
	case "is_climate_on":
		stat.MQTTDataIsClimateOn = convert.StrToBool(string(msg.Payload()))
	case "inside_temp":
		stat.MQTTDataInsideTemp = convert.StrToFloat(string(msg.Payload()))
	case "outside_temp":
		stat.MQTTDataOutsideTemp = convert.StrToFloat(string(msg.Payload()))
	case "is_preconditioning":
		stat.MQTTDataIsPreconditioning = convert.StrToBool(string(msg.Payload()))
	case "climate_keeper_mode":
		stat.MQTTDataClimateKeeperMode = string(msg.Payload())
	case "odometer":
		stat.MQTTDataOdometer = convert.StrToFloat(string(msg.Payload()))
	case "est_battery_range_km":
		stat.MQTTDataEstBatteryRange = convert.StrToFloat(string(msg.Payload()))
	case "rated_battery_range_km":
		stat.MQTTDataRatedBatteryRange = convert.StrToFloat(string(msg.Payload()))
	case "ideal_battery_range_km":
		stat.MQTTDataIdealBatteryRange = convert.StrToFloat(string(msg.Payload()))
	case "battery_level":
		stat.MQTTDataBatteryLevel = convert.StrToInt(string(msg.Payload()))
	case "usable_battery_level":
		stat.MQTTDataUsableBatteryLevel = convert.StrToInt(string(msg.Payload()))
	case "plugged_in":
		stat.MQTTDataPluggedIn = convert.StrToBool(string(msg.Payload()))
	case "charging_state":
		stat.MQTTDataChargingState = strings.ToLower(string(msg.Payload()))
	case "charge_energy_added":
		stat.MQTTDataChargeEnergyAdded = convert.StrToFloat(string(msg.Payload()))
	case "charge_limit_soc":
		stat.MQTTDataChargeLimitSoc = convert.StrToInt(string(msg.Payload()))
	case "charge_port_door_open":
		stat.MQTTDataChargePortDoorOpen = convert.StrToBool(string(msg.Payload()))
	case "charger_actual_current":
		stat.MQTTDataChargerActualCurrent = convert.StrToFloat(string(msg.Payload()))
	case "charger_phases":
		stat.MQTTDataChargerPhases = convert.StrToInt(string(msg.Payload()))
	case "charger_power":
		stat.MQTTDataChargerPower = convert.StrToFloat(string(msg.Payload()))
	case "charger_voltage":
		stat.MQTTDataChargerVoltage = convert.StrToInt(string(msg.Payload()))
	case "charge_current_request":
		stat.MQTTDataChargeCurrentRequest = convert.StrToInt(string(msg.Payload()))
	case "charge_current_request_max":
		stat.MQTTDataChargeCurrentRequestMax = convert.StrToInt(string(msg.Payload()))
	case "scheduled_charging_start_time":
		stat.MQTTDataScheduledChargingStartTime = string(msg.Payload())
	case "time_to_full_charge":
		stat.MQTTDataTimeToFullCharge = convert.StrToFloat(string(msg.Payload()))
	case "tpms_pressure_fl":
		stat.MQTTDataTpmsPressureFL = convert.StrToFloat(string(msg.Payload()))
	case "tpms_pressure_fr":
		stat.MQTTDataTpmsPressureFR = convert.StrToFloat(string(msg.Payload()))
	case "tpms_pressure_rl":
		stat.MQTTDataTpmsPressureRL = convert.StrToFloat(string(msg.Payload()))
	case "tpms_pressure_rr":
		stat.MQTTDataTpmsPressureRR = convert.StrToFloat(string(msg.Payload()))
	case "tpms_soft_warning_fl":
		stat.MQTTDataTpmsSoftWarningFL = convert.StrToBool(string(msg.Payload()))
	case "tpms_soft_warning_fr":
		stat.MQTTDataTpmsSoftWarningFR = convert.StrToBool(string(msg.Payload()))
	case "tpms_soft_warning_rl":
		stat.MQTTDataTpmsSoftWarningRL = convert.StrToBool(string(msg.Payload()))
	case "tpms_soft_warning_rr":
		stat.MQTTDataTpmsSoftWarningRR = convert.StrToBool(string(msg.Payload()))

	case "location":
		var tmp struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		_ = json.Unmarshal(msg.Payload(), &tmp)
		stat.MQTTDataLocation = InfoLocation(tmp)

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
		stat.MQTTDataActiveRoute.DistanceToArrival = convert.MilesToKilometers(tmp.DistanceToArrival)
		stat.MQTTDataActiveRoute.MinutesToArrival = tmp.MinutesToArrival
		stat.MQTTDataActiveRoute.TrafficMinutesDelay = tmp.TrafficMinutesDelay
		stat.MQTTDataActiveRoute.Location = InfoLocation(tmp.Location)

	// deprecated
	case "latitude", "longitude", "active_route_destination", "active_route_latitude", "active_route_longitude":
		// no-op

	default:
		log.Printf("[warning] TeslaMateAPICarsStatusV1 mqtt.MessageHandler issue.. extraction of data for %s not implemented!", mqttTopic)
	}
}
