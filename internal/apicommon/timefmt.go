package apicommon

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DBTimestampFormat is the timestamp layout used by Postgres drivers.
const DBTimestampFormat = "2006-01-02T15:04:05Z"

// GetTimeInTimeZone parses a UTC timestamp string in DBTimestampFormat
// and re-formats it in AppUsersTimezone using RFC3339.
func GetTimeInTimeZone(datestring string) string {
	t, _ := time.Parse(DBTimestampFormat, datestring)
	tz := AppUsersTimezone
	if tz == nil {
		tz = time.UTC
	}
	out := t.In(tz).Format(time.RFC3339)
	if gin.IsDebugging() {
		log.Println("[debug] GetTimeInTimeZone - UTC", t.Format(time.RFC3339), "time converted to", tz, "is", out)
	}
	return out
}

// ParseDateParam parses a user-supplied datestring into the DB timestamp format.
// Empty input returns ("", nil).
func ParseDateParam(datestring string) (string, error) {
	if datestring == "" {
		return "", nil
	}

	if t, err := time.Parse(time.RFC3339, datestring); err == nil {
		return t.UTC().Format(DBTimestampFormat), nil
	}

	tz := AppUsersTimezone
	if tz == nil {
		tz = time.UTC
	}
	normalized := strings.ReplaceAll(datestring, "T", " ")
	if t, err := time.ParseInLocation(time.DateTime, normalized, tz); err == nil {
		return t.UTC().Format(DBTimestampFormat), nil
	}

	sanitized := strings.NewReplacer("\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(datestring)
	return "", fmt.Errorf("invalid date format: %s, please use RFC3339 format", sanitized)
}
