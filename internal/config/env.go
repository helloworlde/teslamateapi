// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// getEnv reads an environment variable; returns defaultVal when unset or empty.
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultVal
}

// getEnvAsBool reads an environment variable as a bool; returns defaultVal on parse failure.
func getEnvAsBool(name string, defaultVal bool) bool {
	if val, err := strconv.ParseBool(getEnv(name, "")); err == nil {
		return val
	}
	return defaultVal
}

// getEnvAsInt reads an environment variable as an int; returns defaultVal on parse failure.
func getEnvAsInt(name string, defaultVal int) int {
	if value, err := strconv.Atoi(getEnv(name, "")); err == nil {
		return value
	}
	return defaultVal
}
