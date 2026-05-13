package config

import (
	"os"
	"strconv"
)

// APIVersion is overridden at link time via -ldflags
// "-X github.com/tobiasehlert/teslamateapi/internal/config.APIVersion=...".
var APIVersion = "unspecified"

// HeaderAPIVersion is the response header key carrying the running API version.
const HeaderAPIVersion = "X-API-Version"

// Env reads a string env var or returns defaultVal when unset/empty.
func Env(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultVal
}

// EnvAsBool reads a bool env var or returns defaultVal when unset/invalid.
func EnvAsBool(name string, defaultVal bool) bool {
	valStr := Env(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

// EnvAsInt reads an int env var or returns defaultVal when unset/invalid.
func EnvAsInt(name string, defaultVal int) int {
	valueStr := Env(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}
