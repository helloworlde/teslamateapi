// Package httpparams contains shared HTTP parameter parsing helpers.
package httpparams

import (
	"fmt"
	"strconv"
)

// PositiveInt parses raw as an integer greater than zero.
func PositiveInt(raw string) (int, error) {
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return 0, fmt.Errorf("must be a positive integer: %q", raw)
	}
	return v, nil
}

// OptionalIntInRange parses raw as an integer in [min, max].
// Empty raw values return def.
func OptionalIntInRange(raw string, def, min, max int) (int, error) {
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < min || v > max {
		return 0, fmt.Errorf("must be an integer in [%d, %d]: %q", min, max, raw)
	}
	return v, nil
}

// OptionalFloatMin parses raw as a float >= min.
// Empty raw values return 0.
func OptionalFloatMin(raw string, min float64) (float64, error) {
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < min {
		return 0, fmt.Errorf("must be a number >= %.0f: %q", min, raw)
	}
	return v, nil
}

// PageOffset converts a 1-indexed page + page size into a SQL offset.
func PageOffset(page, pageSize int) int {
	if page < 1 {
		return 0
	}
	return (page - 1) * pageSize
}
