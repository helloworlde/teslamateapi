package v1

import (
	"strings"
	"testing"
)

func TestDetailMaxPointsClamp(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "default", raw: "", want: defaultDetailMaxPoints},
		{name: "invalid", raw: "abc", want: defaultDetailMaxPoints},
		{name: "min", raw: "10", want: minDetailMaxPoints},
		{name: "max", raw: "99999", want: maxDetailMaxPoints},
		{name: "valid", raw: "1200", want: 1200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detailMaxPoints(tt.raw); got != tt.want {
				t.Fatalf("detailMaxPoints(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestDriveDetailsQueryDefaultsToAuto(t *testing.T) {
	query, args := driveDetailsQuery(42, "", 800)
	if len(args) != 3 || args[0] != 42 || args[1] != 800 || args[2] != 80 {
		t.Fatalf("unexpected args: %#v", args)
	}
	if !strings.Contains(query, "width_bucket") {
		t.Fatalf("default drive query should use auto width-bucket sampling: %s", query)
	}
	for _, expected := range []string{
		"bounds.raw_count <= $2",
		"bounds.raw_count > $2",
		"ORDER BY latitude ASC NULLS LAST",
		"ORDER BY latitude DESC NULLS LAST",
		"ORDER BY longitude ASC NULLS LAST",
		"ORDER BY longitude DESC NULLS LAST",
		"battery_level IS DISTINCT FROM",
	} {
		if !strings.Contains(query, expected) {
			t.Fatalf("default drive query missing auto compatibility rule %q: %s", expected, query)
		}
	}
}

func TestDriveDetailsQueryExplicitEvery5sKeepsLegacyModeWithRouteExtrema(t *testing.T) {
	query, args := driveDetailsQuery(42, "every_5s", 800)
	if len(args) != 1 || args[0] != 42 {
		t.Fatalf("unexpected args: %#v", args)
	}
	if !strings.Contains(query, "DISTINCT ON") {
		t.Fatalf("explicit every_5s drive query should use legacy time bucket sampling: %s", query)
	}
	if strings.Contains(query, "width_bucket") {
		t.Fatalf("explicit every_5s drive query should not use auto width-bucket sampling")
	}
	for _, expected := range []string{
		"route_extrema_ids",
		"ORDER BY latitude ASC NULLS LAST",
		"ORDER BY latitude DESC NULLS LAST",
		"ORDER BY longitude ASC NULLS LAST",
		"ORDER BY longitude DESC NULLS LAST",
	} {
		if !strings.Contains(query, expected) {
			t.Fatalf("explicit every_5s drive query missing route extrema preservation %q: %s", expected, query)
		}
	}
}

func TestDriveDetailsQueryAutoUsesBoundedSampling(t *testing.T) {
	query, args := driveDetailsQuery(42, "auto", 800)
	if len(args) != 3 || args[0] != 42 || args[1] != 800 || args[2] != 80 {
		t.Fatalf("unexpected args: %#v", args)
	}
	for _, expected := range []string{
		"WITH raw AS",
		"width_bucket",
		"bounds.raw_count <= $2",
		"bounds.raw_count > $2",
		"ORDER BY latitude ASC NULLS LAST",
		"ORDER BY latitude DESC NULLS LAST",
		"ORDER BY longitude ASC NULLS LAST",
		"ORDER BY longitude DESC NULLS LAST",
		"battery_level IS DISTINCT FROM",
	} {
		if !strings.Contains(query, expected) {
			t.Fatalf("auto drive query missing %q: %s", expected, query)
		}
	}
}

func TestChargeDetailsQueryDefaultsToAutoAndFullOptOut(t *testing.T) {
	autoQuery, autoArgs := chargeDetailsQuery(99, "", 800)
	if len(autoArgs) != 3 || autoArgs[0] != 99 || autoArgs[1] != 800 || autoArgs[2] != 80 {
		t.Fatalf("unexpected auto args: %#v", autoArgs)
	}
	if !strings.Contains(autoQuery, "fast_charger_present IS DISTINCT FROM") {
		t.Fatalf("auto charge query should preserve state changes: %s", autoQuery)
	}
	if !strings.Contains(autoQuery, "bounds.raw_count > $2") {
		t.Fatalf("auto charge query should skip bucket sampling for small result sets: %s", autoQuery)
	}

	fullQuery, fullArgs := chargeDetailsQuery(99, "full", 800)
	if len(fullArgs) != 1 || fullArgs[0] != 99 {
		t.Fatalf("unexpected full args: %#v", fullArgs)
	}
	if strings.Contains(fullQuery, "width_bucket") {
		t.Fatalf("full charge query should not downsample")
	}
	if !strings.Contains(fullQuery, "ORDER BY charges.id ASC") {
		t.Fatalf("full charge query should preserve original ordering: %s", fullQuery)
	}
}
