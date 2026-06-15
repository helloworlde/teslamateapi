package timefmt

import "testing"

func TestUTCTimestampToLocalSQL(t *testing.T) {
	got := UTCTimestampToLocalSQL("start_date", "$2")
	want := "((start_date AT TIME ZONE 'UTC') AT TIME ZONE $2)"
	if got != want {
		t.Fatalf("UTCTimestampToLocalSQL() = %q, want %q", got, want)
	}
}

func TestUTCNowSQL(t *testing.T) {
	got := UTCNowSQL()
	want := "(CURRENT_TIMESTAMP AT TIME ZONE 'UTC')"
	if got != want {
		t.Fatalf("UTCNowSQL() = %q, want %q", got, want)
	}
}
