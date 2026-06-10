// Package config loads runtime configuration from environment variables.
package config

import (
	"log"
	"time"
)

// Config holds all settings parsed from the process environment.
//
// Fields are grouped by subsystem so handlers can carry only the slice they
// need rather than the whole struct.
type Config struct {
	// Process / runtime
	DebugMode bool
	TZName    string
	Language  string

	// Database
	DBHost        string
	DBPort        int
	DBUser        string
	DBPass        string
	DBName        string
	DBTimeoutMS   int
	DBSSLMode     string
	DBSSLRootCert string

	// Auth
	APIToken        string
	APITokenDisable bool

	// Commands
	CommandsEnabled        bool
	CommandsAll            bool
	CommandsAllowList      string
	CommandsLogging        bool
	CommandsWake           bool
	CommandsAlert          bool
	CommandsRemotestart    bool
	CommandsHomelink       bool
	CommandsSpeedlimit     bool
	CommandsValet          bool
	CommandsSentryMode     bool
	CommandsDoors          bool
	CommandsTrunk          bool
	CommandsWindows        bool
	CommandsSunroof        bool
	CommandsCharging       bool
	CommandsClimate        bool
	CommandsMedia          bool
	CommandsSharing        bool
	CommandsSoftwareupdate bool
	CommandsUnknown        bool

	// Tesla / TeslaMate proxies
	TeslaAPIHost  string
	TeslaMateHost string
	TeslaMatePort string
	TeslaMateSSL  bool
	EncryptionKey string

	// MQTT
	MQTTDisabled  bool
	MQTTTLS       bool
	MQTTHost      string
	MQTTPort      int
	MQTTUsername  string
	MQTTPassword  string
	MQTTClientID  string
	MQTTNamespace string
}

// Load reads every supported environment variable once and returns a Config.
//
// Defaults match the historical behaviour from src/webserver.go and friends:
// nothing in the new layout silently changes a default.
func Load() Config {
	mqttPortDefault := 1883
	if getEnvAsBool("MQTT_TLS", false) {
		mqttPortDefault = 8883
	}
	return Config{
		DebugMode: getEnvAsBool("DEBUG_MODE", false),
		TZName:    getEnv("TZ", "Europe/Berlin"),
		Language:  getEnv("LANGUAGE", getEnv("LANG", "en")),

		DBHost:        getEnv("DATABASE_HOST", "database"),
		DBPort:        getEnvAsInt("DATABASE_PORT", 5432),
		DBUser:        getEnv("DATABASE_USER", "teslamate"),
		DBPass:        getEnv("DATABASE_PASS", "secret"),
		DBName:        getEnv("DATABASE_NAME", "teslamate"),
		DBTimeoutMS:   getEnvAsInt("DATABASE_TIMEOUT", 60000),
		DBSSLMode:     getEnv("DATABASE_SSL", "disable"),
		DBSSLRootCert: getEnv("DATABASE_SSL_CA_CERT_FILE", ""),

		APIToken:        getEnv("API_TOKEN", ""),
		APITokenDisable: getEnvAsBool("API_TOKEN_DISABLE", false),

		CommandsEnabled:        getEnvAsBool("ENABLE_COMMANDS", false),
		CommandsAll:            getEnvAsBool("COMMANDS_ALL", false),
		CommandsAllowList:      getEnv("COMMANDS_ALLOWLIST", "allow_list.json"),
		CommandsLogging:        getEnvAsBool("COMMANDS_LOGGING", false),
		CommandsWake:           getEnvAsBool("COMMANDS_WAKE", false),
		CommandsAlert:          getEnvAsBool("COMMANDS_ALERT", false),
		CommandsRemotestart:    getEnvAsBool("COMMANDS_REMOTESTART", false),
		CommandsHomelink:       getEnvAsBool("COMMANDS_HOMELINK", false),
		CommandsSpeedlimit:     getEnvAsBool("COMMANDS_SPEEDLIMIT", false),
		CommandsValet:          getEnvAsBool("COMMANDS_VALET", false),
		CommandsSentryMode:     getEnvAsBool("COMMANDS_SENTRYMODE", false),
		CommandsDoors:          getEnvAsBool("COMMANDS_DOORS", false),
		CommandsTrunk:          getEnvAsBool("COMMANDS_TRUNK", false),
		CommandsWindows:        getEnvAsBool("COMMANDS_WINDOWS", false),
		CommandsSunroof:        getEnvAsBool("COMMANDS_SUNROOF", false),
		CommandsCharging:       getEnvAsBool("COMMANDS_CHARGING", false),
		CommandsClimate:        getEnvAsBool("COMMANDS_CLIMATE", false),
		CommandsMedia:          getEnvAsBool("COMMANDS_MEDIA", false),
		CommandsSharing:        getEnvAsBool("COMMANDS_SHARING", false),
		CommandsSoftwareupdate: getEnvAsBool("COMMANDS_SOFTWAREUPDATE", false),
		CommandsUnknown:        getEnvAsBool("COMMANDS_UNKNOWN", false),

		TeslaAPIHost:  getEnv("TESLA_API_HOST", ""),
		TeslaMateHost: getEnv("TESLAMATE_HOST", "teslamate"),
		TeslaMatePort: getEnv("TESLAMATE_PORT", "4000"),
		TeslaMateSSL:  getEnvAsBool("TESLAMATE_SSL", false),
		EncryptionKey: getEnv("ENCRYPTION_KEY", ""),

		MQTTDisabled:  getEnvAsBool("DISABLE_MQTT", false),
		MQTTTLS:       getEnvAsBool("MQTT_TLS", false),
		MQTTHost:      getEnv("MQTT_HOST", "mosquitto"),
		MQTTPort:      getEnvAsInt("MQTT_PORT", mqttPortDefault),
		MQTTUsername:  getEnv("MQTT_USERNAME", ""),
		MQTTPassword:  getEnv("MQTT_PASSWORD", ""),
		MQTTClientID:  getEnv("MQTT_CLIENTID", ""),
		MQTTNamespace: getEnv("MQTT_NAMESPACE", ""),
	}
}

// LoadTZ parses tzName into a *time.Location, falling back to UTC with a
// warning log if the name is unrecognised. Mirrors the original behaviour
// of webserver.go which guarded against `time.Time.In(nil)` panics.
func LoadTZ(tzName string) *time.Location {
	loc, err := time.LoadLocation(tzName)
	if err != nil || loc == nil {
		log.Printf("[warning] TZ=%q not loadable (%v); falling back to UTC", tzName, err)
		return time.UTC
	}
	return loc
}
