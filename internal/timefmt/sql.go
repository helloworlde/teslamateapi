package timefmt

import "fmt"

// UTCTimestampToLocalSQL converts TeslaMate's UTC timestamp-without-timezone
// columns into a local wall-clock timestamp for calendar bucketing.
func UTCTimestampToLocalSQL(tsExpr, tzParam string) string {
	return fmt.Sprintf("((%s AT TIME ZONE 'UTC') AT TIME ZONE %s)", tsExpr, tzParam)
}

// UTCNowSQL returns the current UTC instant as timestamp-without-timezone,
// matching TeslaMate timestamp columns for elapsed-time arithmetic.
func UTCNowSQL() string {
	return "(CURRENT_TIMESTAMP AT TIME ZONE 'UTC')"
}
