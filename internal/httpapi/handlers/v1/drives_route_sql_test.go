package v1

import (
	"strings"
	"testing"
)

func TestDrivesRouteQueryRestrictsToRequestedDrives(t *testing.T) {
	query := drivesRouteQuery([]int{7, 11, 13}, drivesRouteMaxPoints)

	if !strings.Contains(query, "p.drive_id IN (7,11,13)") {
		t.Fatalf("query should restrict to the page's drives; got %q", query)
	}
}

// Sampling must be per drive. Without the partition, one long drive would
// consume the whole point budget and starve every other route on the page.
func TestDrivesRouteQueryPartitionsPerDrive(t *testing.T) {
	query := drivesRouteQuery([]int{1}, 120)

	for _, needle := range []string{
		"ROW_NUMBER() OVER (PARTITION BY p.drive_id",
		"COUNT(*) OVER (PARTITION BY p.drive_id)",
		"GROUP BY drive_id",
	} {
		if !strings.Contains(query, needle) {
			t.Fatalf("query missing %q", needle)
		}
	}
}

// Clients frame the map camera from the bounds of the points they receive, so
// dropping an extremum would silently crop the route out of view.
func TestDrivesRouteQueryKeepsEndpointsAndExtrema(t *testing.T) {
	query := drivesRouteQuery([]int{1}, 120)

	for _, needle := range []string{
		"r.rn = 1",
		"r.rn = r.total",
		"MIN(latitude) AS min_latitude",
		"MAX(latitude) AS max_latitude",
		"MIN(longitude) AS min_longitude",
		"MAX(longitude) AS max_longitude",
		"r.point_id IN (SELECT point_id FROM route_extrema)",
	} {
		if !strings.Contains(query, needle) {
			t.Fatalf("query missing %q", needle)
		}
	}
}

// The extrema must resolve to point ids, not to coordinate values: a drive that
// sat still at its northernmost coordinate has many rows equal to max(latitude),
// and matching on the value would return all of them and blow the point budget.
// DISTINCT ON keeps one row per drive per bound.
func TestDrivesRouteQueryResolvesExtremaToSinglePoints(t *testing.T) {
	query := drivesRouteQuery([]int{1}, 120)

	if got := strings.Count(query, "SELECT DISTINCT ON (r.drive_id) r.point_id"); got != 4 {
		t.Fatalf("expected one DISTINCT ON branch per bound, got %d; query %q", got, query)
	}
}

// AGENTS.md names `array_agg(x ORDER BY y)[1]` as an antipattern: over a page of
// drives it sorts every position once per extremum.
func TestDrivesRouteQueryAvoidsArrayAggExtrema(t *testing.T) {
	query := drivesRouteQuery([]int{1}, 120)

	if strings.Contains(query, "array_agg") {
		t.Fatalf("extrema should come from MIN/MAX aggregates, not array_agg; got %q", query)
	}
}

// The stride expression is built with fmt, so a mis-escaped percent would emit
// a literal Postgres rejects.
func TestDrivesRouteQueryEmitsLiteralModulo(t *testing.T) {
	query := drivesRouteQuery([]int{1}, 200)

	if !strings.Contains(query, "((r.rn - 1) % GREATEST(1, CEIL(r.total::numeric / 200)::bigint)) = 0") {
		t.Fatalf("stride expression malformed; got %q", query)
	}
	if strings.Contains(query, "%!") {
		t.Fatalf("format verb error in query; got %q", query)
	}
	if !strings.Contains(query, "r.total <= 200") {
		t.Fatalf("short routes should bypass sampling; got %q", query)
	}
}

// max_points_per_drive is interpolated rather than bound precisely because it
// appears in both a bigint comparison and a numeric division; binding it once
// would make Postgres deduce conflicting types for the placeholder.
func TestDrivesRouteQueryBindsNoParameters(t *testing.T) {
	query := drivesRouteQuery([]int{1, 2}, 120)

	if strings.Contains(query, "$1") {
		t.Fatalf("route query should carry no placeholders; got %q", query)
	}
}

// Ordering by date keeps the polyline drawable end to end; ordering by drive
// first lets the handler attach rows in a single pass.
func TestDrivesRouteQueryOrdersByDriveThenTime(t *testing.T) {
	query := drivesRouteQuery([]int{1}, 120)

	if !strings.Contains(query, "ORDER BY r.drive_id ASC, r.date ASC, r.point_id ASC") {
		t.Fatalf("unexpected ordering; got %q", query)
	}
}

func TestDrivesRouteQuerySkipsNullCoordinates(t *testing.T) {
	query := drivesRouteQuery([]int{1}, 120)

	if !strings.Contains(query, "p.latitude IS NOT NULL") || !strings.Contains(query, "p.longitude IS NOT NULL") {
		t.Fatalf("query should skip coordinate-less positions; got %q", query)
	}
}

// The per-drive target and `show` must not multiply into an unbounded payload:
// a crowded page trades per-route detail for a bounded response, and a sparse
// one keeps the fidelity the client asked for.
func TestDrivesRoutePointsPerDriveRespectsBudget(t *testing.T) {
	cases := []struct {
		name       string
		requested  int
		driveCount int
		want       int
	}{
		{"few drives keep the requested target", 800, 10, 800},
		{"default page stays at its target", drivesRouteMaxPoints, 100, drivesRouteMaxPoints},
		{"crowded page is downsampled to its share", 800, 200, drivesRouteTotalPointBudget / 200},
		{"never below the floor", 800, 100000, drivesRouteMinPointsPerDrive},
		{"empty page is a no-op", 120, 0, 120},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := drivesRoutePointsPerDrive(tc.requested, tc.driveCount); got != tc.want {
				t.Fatalf("drivesRoutePointsPerDrive(%d, %d) = %d, want %d", tc.requested, tc.driveCount, got, tc.want)
			}
		})
	}
}

// The budget must never raise the client's requested fidelity, only lower it.
func TestDrivesRoutePointsPerDriveNeverExceedsRequest(t *testing.T) {
	for driveCount := 1; driveCount <= drivesRouteMaxDrives; driveCount++ {
		got := drivesRoutePointsPerDrive(drivesRouteMinPointsPerDrive, driveCount)
		if got != drivesRouteMinPointsPerDrive {
			t.Fatalf("driveCount=%d raised the target to %d", driveCount, got)
		}
	}
}

// A full page at the maximum per-drive target must still fit the budget, or the
// cap on `show` and the budget disagree about what "bounded" means.
func TestDrivesRouteBudgetCoversAFullPage(t *testing.T) {
	perDrive := drivesRoutePointsPerDrive(drivesRouteMaxPointsPerDrive, drivesRouteMaxDrives)

	if total := perDrive * drivesRouteMaxDrives; total > drivesRouteTotalPointBudget {
		t.Fatalf("a full page yields %d points, over the %d budget", total, drivesRouteTotalPointBudget)
	}
}
