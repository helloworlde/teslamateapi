package v2

import "testing"

func TestLocalTimestampSQLDeclaresUTCBeforeUserTimezone(t *testing.T) {
	got := localTimestampSQL("start_date", "$2")
	want := "((start_date AT TIME ZONE 'UTC') AT TIME ZONE $2)"
	if got != want {
		t.Fatalf("localTimestampSQL() = %q; want %q", got, want)
	}
}

func TestBucketStepSQLUsesPeriodParameter(t *testing.T) {
	got := bucketStepSQL("$2")
	want := "CASE $2 WHEN 'day' THEN INTERVAL '1 day' WHEN 'week' THEN INTERVAL '1 week' WHEN 'month' THEN INTERVAL '1 month' WHEN 'year' THEN INTERVAL '1 year' END"
	if got != want {
		t.Fatalf("bucketStepSQL() = %q; want %q", got, want)
	}
}
