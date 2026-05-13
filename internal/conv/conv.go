// Package conv contains unit conversions used across v1 and v2 endpoints.
package conv

import "github.com/tobiasehlert/teslamateapi/internal/nullx"

// KmToMi converts kilometers to miles.
func KmToMi(km float64) float64 { return km * 0.62137119223733 }

// MiToKm converts miles to kilometers.
func MiToKm(mi float64) float64 { return mi * 1.609344 }

// KmToMiInt converts kilometers (int) to miles (int).
func KmToMiInt(km int) int { return int(float64(km) * 0.62137119223733) }

// KmToMiNullable converts kilometers to miles for null-safe values.
func KmToMiNullable(km nullx.Float64) nullx.Float64 {
	km.Float64 = km.Float64 * 0.62137119223733
	return km
}

// BarToPsi converts bar to PSI.
func BarToPsi(bar float64) float64 { return bar * 14.503773800722 }

// CelsiusToFahrenheit converts °C to °F.
func CelsiusToFahrenheit(c float64) float64 { return c*9/5 + 32 }

// CelsiusToFahrenheitNullable converts °C to °F for null-safe values.
func CelsiusToFahrenheitNullable(c nullx.Float64) nullx.Float64 {
	c.Float64 = c.Float64*9/5 + 32
	return c
}

// SlopeAdjustedConsumption returns the elevation-adjusted average consumption (Wh/km).
//
// Mirrors Grafana drives.json:
//
//	consumption + (ascent − descent) × g × m / (3600 × distance × 1000) × 1000 / regen.
//
// distance is kilometers, ascent/descent in meters, baseConsumption in Wh/km.
func SlopeAdjustedConsumption(distance, ascent, descent, baseConsumption float64) float64 {
	if distance <= 0 {
		return baseConsumption
	}
	const (
		gravity   = 9.81
		carMassKg = 2100.0
		regenEff  = 0.85
	)
	delta := (ascent - descent) * gravity * carMassKg / (3600.0 * distance * 1000.0) * 1000.0 / regenEff
	return baseConsumption + delta
}
