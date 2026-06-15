package v2

// localTimestampSQL converts TeslaMate's UTC timestamp-without-time-zone
// columns into a local wall-clock timestamp before extracting calendar parts.
// In Postgres, `ts AT TIME ZONE zone` on a timestamp first interprets `ts` in
// that zone. TeslaMate stores these timestamps as UTC wall-clock values, so the
// conversion has to declare UTC first, then render in the user's timezone.
func localTimestampSQL(column, tzParam string) string {
	return "((" + column + " AT TIME ZONE 'UTC') AT TIME ZONE " + tzParam + ")"
}

func utcNowTimestampSQL() string {
	return "(CURRENT_TIMESTAMP AT TIME ZONE 'UTC')"
}

func bucketStepSQL(periodParam string) string {
	return "CASE " + periodParam + " WHEN 'day' THEN INTERVAL '1 day' WHEN 'week' THEN INTERVAL '1 week' WHEN 'month' THEN INTERVAL '1 month' WHEN 'year' THEN INTERVAL '1 year' END"
}
