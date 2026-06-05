package timefmt

import (
	"testing"
	"time"
)

func TestGetTimeInTimeZoneInvalidInputReturnsOriginal(t *testing.T) {
	got := GetTimeInTimeZone("not-a-date", time.UTC)
	if got != "not-a-date" {
		t.Fatalf("GetTimeInTimeZone invalid input = %q", got)
	}
}

func TestGetTimeInTimeZoneEmptyInputReturnsEmpty(t *testing.T) {
	got := GetTimeInTimeZone("", time.UTC)
	if got != "" {
		t.Fatalf("GetTimeInTimeZone empty input = %q", got)
	}
}
