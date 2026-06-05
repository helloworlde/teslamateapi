// Package convert holds tiny pure conversion helpers shared by handlers.
//
// Functions here have no side effects and no globals; they're safe to call
// from anywhere.
package convert

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/pkg/nullable"
)

// StrToBool parses data as a bool, returning false on parse failure.
func StrToBool(data string) bool {
	value, err := strconv.ParseBool(data)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] convertStringToBool: failed to parse '%s' as boolean - returning false", data)
		}
		return false
	}
	return value
}

// StrToFloat parses data as a float64, returning 0.0 on parse failure.
func StrToFloat(data string) float64 {
	value, err := strconv.ParseFloat(data, 64)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] convertStringToFloat: failed to parse '%s' as float64 - returning 0.0", data)
		}
		return 0.0
	}
	return value
}

// StrToInt parses data as an int, returning 0 on parse failure.
func StrToInt(data string) int {
	value, err := strconv.Atoi(data)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] convertStringToInteger: failed to parse '%s' as integer - returning 0", data)
		}
		return 0
	}
	return value
}

// KilometersToMiles converts km to miles.
func KilometersToMiles(km float64) float64 { return km * 0.62137119223733 }

// KilometersToMilesNullable converts a nullable kilometre value to miles in place.
func KilometersToMilesNullable(km nullable.Float64) nullable.Float64 {
	km.Float64 = km.Float64 * 0.62137119223733
	return km
}

// MilesToKilometers converts miles to km.
func MilesToKilometers(mi float64) float64 { return mi * 1.609344 }

// KilometersToMilesInteger converts a kilometre integer to miles, truncated.
func KilometersToMilesInteger(km int) int { return int(float64(km) * 0.62137119223733) }

// WhPerKmToWhPerMile converts consumption from Wh/km to Wh/mi.
func WhPerKmToWhPerMile(whPerKm float64) float64 { return whPerKm * 1.609344 }

// BarToPsi converts bar to psi.
func BarToPsi(bar float64) float64 { return bar * 14.503773800722 }

// CelsiusToFahrenheit converts c to fahrenheit.
func CelsiusToFahrenheit(c float64) float64 { return c*9/5 + 32 }

// CelsiusToFahrenheitNullable converts a nullable celsius value to fahrenheit in place.
func CelsiusToFahrenheitNullable(c nullable.Float64) nullable.Float64 {
	c.Float64 = c.Float64*9/5 + 32
	return c
}
