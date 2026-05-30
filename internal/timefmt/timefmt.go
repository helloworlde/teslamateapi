// Package timefmt converts between TeslaMate's Postgres timestamp format
// and the user-facing RFC3339 representation in a configured timezone.
package timefmt

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DBTimestampFormat is the format Postgres returns timestamps in for this
// codebase.
const DBTimestampFormat = "2006-01-02T15:04:05Z"

// GetTimeInTimeZone converts a DBTimestampFormat datestring into RFC3339
// in tz.
func GetTimeInTimeZone(datestring string, tz *time.Location) string {
	t, _ := time.Parse(DBTimestampFormat, datestring)
	out := t.In(tz).Format(time.RFC3339)
	if gin.IsDebugging() {
		log.Println("[debug] getTimeInTimeZone - UTC", t.Format(time.RFC3339), "time converted to", tz, "is", out)
	}
	return out
}

// ParseDateParam normalises a user-supplied date filter into the
// DBTimestampFormat. Empty input returns "" with no error.
//
// Accepts RFC3339 (with timezone offset/Z) directly. Falls back to the
// `2006-01-02 15:04:05` form, which is interpreted in tz.
func ParseDateParam(datestring string, tz *time.Location) (string, error) {
	if datestring == "" {
		return "", nil
	}

	if t, err := time.Parse(time.RFC3339, datestring); err == nil {
		return t.UTC().Format(DBTimestampFormat), nil
	}

	normalizedDateString := strings.ReplaceAll(datestring, "T", " ")
	if t, err := time.ParseInLocation(time.DateTime, normalizedDateString, tz); err == nil {
		return t.UTC().Format(DBTimestampFormat), nil
	}

	sanitizedInput := strings.NewReplacer("\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(datestring)
	return "", fmt.Errorf("invalid date format: %s, please use RFC3339 format", sanitizedInput)
}
