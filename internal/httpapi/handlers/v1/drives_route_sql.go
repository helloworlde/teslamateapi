package v1

import (
	"fmt"
	"strconv"
	"strings"
)

// Route sampling for the drives list endpoint.
//
// Why it lives on the list rather than a dedicated endpoint: a client drawing
// a month of driving on one map already asks this endpoint for exactly that
// window's drives. Without opt-in routes it would have to follow up with one
// /drives/{id} call per drive — 40-80 round trips for a month, several hundred
// for a year. `include_route=true` lets the same request carry the paths.
//
// Sampling mirrors the per-drive rule in v1_detail_sampling.go, applied
// PARTITION BY drive_id so each drive is sampled independently: without the
// partition one long drive would consume the whole point budget and starve
// every other route.

// drivesRouteMaxPoints is the default per-drive sampling target for
// include_route. Deliberately far below the 800 used by the detail endpoint —
// a route drawn among dozens of others needs shape, not fidelity.
const drivesRouteMaxPoints = 120

// drivesRouteMaxPointsRange bounds the client-supplied override.
const (
	drivesRouteMinPointsPerDrive = 20
	drivesRouteMaxPointsPerDrive = 800
)

// drivesRouteMaxDrives caps how many drives one request may pull routes for.
// The sampling target bounds the *response*, not the work: `raw` below reads
// every position of every drive on the page whatever the target is, so without
// this cap `show=10000&include_route=true` would sort millions of positions
// rows in one query. Clients wanting a longer window paginate.
const drivesRouteMaxDrives = 200

// drivesRouteTotalPointBudget bounds the points returned across the whole page,
// so `max_points_per_drive` and `show` cannot multiply into a payload no map
// view can use. The per-drive target is reduced to fit (never below
// drivesRouteMinPointsPerDrive, which is cheap enough to ignore the budget for).
const drivesRouteTotalPointBudget = 40000

// drivesRoutePointsPerDrive spreads drivesRouteTotalPointBudget over the page.
// A page of few drives gets the full requested fidelity; a crowded page trades
// per-route detail for a bounded response.
func drivesRoutePointsPerDrive(requested, driveCount int) int {
	if driveCount <= 0 {
		return requested
	}
	share := drivesRouteTotalPointBudget / driveCount
	if share >= requested {
		return requested
	}
	if share < drivesRouteMinPointsPerDrive {
		return drivesRouteMinPointsPerDrive
	}
	return share
}

// drivesRouteQuery samples the positions of the supplied drives.
//
// driveIDs come from the already-executed page query (ints straight out of the
// DB) and maxPointsPerDrive is range-validated before it gets here, so both are
// interpolated as literals. Interpolating maxPointsPerDrive is not just
// convenience: it appears in both a bigint comparison and a numeric division,
// and Postgres unifies a placeholder's deduced type across all its uses, so
// binding it once would risk "inconsistent types deduced for parameter".
//
// Kept per drive: an evenly strided subset, the first and last point, and the
// four latitude/longitude extrema. Keeping the extrema is not cosmetic —
// clients frame the map camera from the north/south/east/west bounds of the
// points they receive, so dropping an extremum would silently crop the route
// out of view.
//
// The extrema are found via per-drive MIN/MAX (a hash aggregate, no sort) and
// then resolved to one point id per bound. The obvious
// `(array_agg(point_id ORDER BY latitude))[1]` spelling is avoided on purpose:
// over a whole page of drives it sorts every position four times, which is the
// antipattern AGENTS.md calls out. Resolving ids matters too — matching rows by
// coordinate *value* would return every point of a drive that sat still at its
// northernmost coordinate, blowing the point budget.
func drivesRouteQuery(driveIDs []int, maxPointsPerDrive int) string {
	ids := make([]string, 0, len(driveIDs))
	for _, id := range driveIDs {
		ids = append(ids, strconv.Itoa(id))
	}

	return fmt.Sprintf(`
		WITH raw AS (
			SELECT
				p.drive_id,
				p.id AS point_id,
				p.date,
				p.latitude,
				p.longitude,
				ROW_NUMBER() OVER (PARTITION BY p.drive_id ORDER BY p.date ASC, p.id ASC) AS rn,
				COUNT(*) OVER (PARTITION BY p.drive_id) AS total
			FROM positions p
			WHERE p.drive_id IN (%s)
				AND p.latitude IS NOT NULL
				AND p.longitude IS NOT NULL
		),
		route_bounds AS (
			SELECT
				drive_id,
				MIN(latitude) AS min_latitude,
				MAX(latitude) AS max_latitude,
				MIN(longitude) AS min_longitude,
				MAX(longitude) AS max_longitude
			FROM raw
			GROUP BY drive_id
		),
		route_extrema AS (
			(SELECT DISTINCT ON (r.drive_id) r.point_id
				FROM raw r JOIN route_bounds b ON b.drive_id = r.drive_id
				WHERE r.latitude = b.min_latitude
				ORDER BY r.drive_id, r.point_id)
			UNION
			(SELECT DISTINCT ON (r.drive_id) r.point_id
				FROM raw r JOIN route_bounds b ON b.drive_id = r.drive_id
				WHERE r.latitude = b.max_latitude
				ORDER BY r.drive_id, r.point_id)
			UNION
			(SELECT DISTINCT ON (r.drive_id) r.point_id
				FROM raw r JOIN route_bounds b ON b.drive_id = r.drive_id
				WHERE r.longitude = b.min_longitude
				ORDER BY r.drive_id, r.point_id)
			UNION
			(SELECT DISTINCT ON (r.drive_id) r.point_id
				FROM raw r JOIN route_bounds b ON b.drive_id = r.drive_id
				WHERE r.longitude = b.max_longitude
				ORDER BY r.drive_id, r.point_id)
		)
		SELECT r.drive_id, r.latitude, r.longitude
		FROM raw r
		WHERE r.total <= %d
			OR r.rn = 1
			OR r.rn = r.total
			OR ((r.rn - 1) %% GREATEST(1, CEIL(r.total::numeric / %d)::bigint)) = 0
			OR r.point_id IN (SELECT point_id FROM route_extrema)
		ORDER BY r.drive_id ASC, r.date ASC, r.point_id ASC;`,
		strings.Join(ids, ","), maxPointsPerDrive, maxPointsPerDrive)
}
