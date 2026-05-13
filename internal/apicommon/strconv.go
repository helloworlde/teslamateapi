package apicommon

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ConvertStringToBool parses s as bool, returning false on error.
func ConvertStringToBool(s string) bool {
	v, err := strconv.ParseBool(s)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] ConvertStringToBool: failed to parse '%s' as boolean - returning false", s)
		}
		return false
	}
	return v
}

// ConvertStringToFloat parses s as float64, returning 0.0 on error.
func ConvertStringToFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] ConvertStringToFloat: failed to parse '%s' as float64 - returning 0.0", s)
		}
		return 0.0
	}
	return v
}

// ConvertStringToInteger parses s as int, returning 0 on error.
func ConvertStringToInteger(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] ConvertStringToInteger: failed to parse '%s' as integer - returning 0", s)
		}
		return 0
	}
	return v
}
