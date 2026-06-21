package v2

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// tripDistEdgesKm are the lower bounds of the trip-length histogram bands in
// kilometres. The final band (100+) is open-ended. Kept in km because the
// underlying drives.distance is metric; edges are converted to the user's unit
// for display, mirroring how consumption renders temperature bands.
var tripDistEdgesKm = []float64{0, 5, 10, 20, 50, 100}

// TeslaMateAPICarsStatsBehaviorV2 returns three objective usage distributions
// in one response: a weekday×hour activity heatmap, a charge-level histogram,
// and a trip-length histogram. It is the behaviour-profile screen's data
// source.
//
// Everything here is a direct count or a sum over per-session rows — no
// thresholds, estimates, or assumptions. Energy in the trip-type section uses
// the same rated-range-drop × efficiency model as lifetime / consumption.
//
// Performance: three small aggregations over `drives` / `charging_processes`
// (one row per session). It never scans the multi-million-row `positions`
// table. Heatmap timestamps are bucketed in the user's timezone.
//
// @Summary      Behaviour profile
// @Description  Weekday×hour heatmap, charge-level histogram, and trip-length histogram. Objective data only.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path  int  true  "TeslaMate cars.id"
// @Success      200  {object}  dto.V2BehaviorResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/stats/behavior [get]
func (h *Handler) StatsBehavior(c *gin.Context) {

	const handler = "TeslaMateAPICarsStatsBehaviorV2"
	var ErrMsg = "Unable to load behavior stats."

	CarID, ok := respond.RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}

	ctx := c.Request.Context()
	tzName := h.tz.String()

	// --- Heatmap: weekday × hour activity, drives and charges combined ---
	// DOW/HOUR are extracted after converting TeslaMate's UTC timestamp columns
	// into the user's local wall-clock. Drives contribute count, distance, and
	// duration; charges contribute count, energy, and duration.
	heatmap := make([]dto.V2BehaviorHeatmapCell, 0, 24)
	localStart := localTimestampSQL("start_date", "$2")
	heatRows, err := h.db.QueryContext(ctx, fmt.Sprintf(`
		WITH cells AS (
			SELECT
				EXTRACT(DOW  FROM %[1]s)::int AS wd,
				EXTRACT(HOUR FROM %[1]s)::int AS hr,
				1 AS is_drive,
				0 AS is_charge,
				COALESCE(distance, 0) AS drive_dist,
				COALESCE(duration_min, 0) AS drive_dur,
				0::double precision AS charge_added,
				0::double precision AS charge_used,
				0 AS charge_dur
			FROM drives
			WHERE car_id = $1 AND end_date IS NOT NULL
			UNION ALL
			SELECT
				EXTRACT(DOW  FROM %[1]s)::int,
				EXTRACT(HOUR FROM %[1]s)::int,
				0,
				1,
				0,
				0,
				COALESCE(charge_energy_added, 0),
				COALESCE(GREATEST(charge_energy_used, charge_energy_added), 0),
				COALESCE(duration_min, 0)
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL
		)
		SELECT
			wd, hr,
			SUM(is_drive)::int AS drives_count,
			COALESCE(SUM(drive_dist), 0) AS drives_distance,
			COALESCE(SUM(drive_dur), 0)::int AS drives_duration_min,
			SUM(is_charge)::int AS charges_count,
			COALESCE(SUM(charge_added), 0) AS charges_energy_added_kwh,
			COALESCE(SUM(charge_used), 0) AS charges_energy_used_kwh,
			COALESCE(SUM(charge_dur), 0)::int AS charges_duration_min
		FROM cells
		GROUP BY wd, hr
		ORDER BY wd, hr`, localStart), CarID, tzName)
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}
	for heatRows.Next() {
		var cell dto.V2BehaviorHeatmapCell
		if err = heatRows.Scan(
			&cell.Weekday,
			&cell.Hour,
			&cell.DrivesCount,
			&cell.DrivesDistance,
			&cell.DrivesDurationMin,
			&cell.ChargesCount,
			&cell.ChargesEnergyAddedKWh,
			&cell.ChargesEnergyUsedKWh,
			&cell.ChargesDurationMin,
		); err != nil {
			heatRows.Close()
			respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
			return
		}
		heatmap = append(heatmap, cell)
	}
	if err = heatRows.Err(); err != nil {
		heatRows.Close()
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}
	heatRows.Close()

	// --- Charge-level histogram: start/end SOC bucketed into 10-point bands ---
	// A level of 100 floors to band 90 (the [90,100] band) via LEAST. Both
	// counts come from one scan; the generated 0…90 series fills empty bands.
	chargeLevels := make([]dto.V2BehaviorChargeLevelBucket, 0, 10)
	lvlRows, err := h.db.QueryContext(ctx, `
		WITH lv AS (
			SELECT
				LEAST(floor(start_battery_level / 10) * 10, 90) AS sb,
				LEAST(floor(end_battery_level   / 10) * 10, 90) AS eb
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL
				AND start_battery_level IS NOT NULL AND end_battery_level IS NOT NULL
		),
		buckets AS (SELECT generate_series(0, 90, 10) AS low)
		SELECT
			b.low::int,
			(SELECT COUNT(*) FROM lv WHERE lv.sb = b.low)::int AS start_count,
			(SELECT COUNT(*) FROM lv WHERE lv.eb = b.low)::int AS end_count
		FROM buckets b
		ORDER BY b.low`, CarID)
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}
	for lvlRows.Next() {
		var b dto.V2BehaviorChargeLevelBucket
		if err = lvlRows.Scan(&b.BucketLow, &b.StartCount, &b.EndCount); err != nil {
			lvlRows.Close()
			respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
			return
		}
		b.BucketHigh = b.BucketLow + 10
		chargeLevels = append(chargeLevels, b)
	}
	if err = lvlRows.Err(); err != nil {
		lvlRows.Close()
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}
	lvlRows.Close()

	// --- Trip-length histogram: drives bucketed by distance band ---
	// Band low-bound (km) is the group key; energy uses the rated-range-drop
	// model. Only qualifying drives (positive distance, known range endpoints)
	// contribute energy, matching consumption/lifetime semantics.
	type rawTrip struct {
		low   float64
		trips int
		dist  float64
		kwh   float64
	}
	tripRows, err := h.db.QueryContext(ctx, `
		SELECT
			CASE
				WHEN d.distance <   5 THEN 0
				WHEN d.distance <  10 THEN 5
				WHEN d.distance <  20 THEN 10
				WHEN d.distance <  50 THEN 20
				WHEN d.distance < 100 THEN 50
				ELSE 100
			END AS low,
			COUNT(*)::int AS trips,
			COALESCE(SUM(d.distance), 0) AS dist,
			COALESCE(SUM(
				CASE WHEN d.start_rated_range_km IS NOT NULL AND d.end_rated_range_km IS NOT NULL
				THEN GREATEST(d.start_rated_range_km - d.end_rated_range_km, 0) * cars.efficiency
				ELSE 0 END
			), 0) AS kwh
		FROM drives d
		LEFT JOIN cars ON cars.id = d.car_id
		WHERE d.car_id = $1 AND d.end_date IS NOT NULL AND d.distance > 0
		GROUP BY low
		ORDER BY low`, CarID)
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}
	var rawTrips []rawTrip
	for tripRows.Next() {
		var t rawTrip
		if err = tripRows.Scan(&t.low, &t.trips, &t.dist, &t.kwh); err != nil {
			tripRows.Close()
			respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
			return
		}
		rawTrips = append(rawTrips, t)
	}
	if err = tripRows.Err(); err != nil {
		tripRows.Close()
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}
	tripRows.Close()

	// --- Units + car name (one small lookup, mirrors the other stats handlers) ---
	var UnitsLength, UnitsTemperature string
	var CarName NullString
	if err = h.db.QueryRowContext(ctx,
		`SELECT (SELECT unit_of_length FROM settings LIMIT 1),
		        (SELECT unit_of_temperature FROM settings LIMIT 1),
		        (SELECT name FROM cars WHERE id = $1)`, CarID).
		Scan(&UnitsLength, &UnitsTemperature, &CarName); err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	// Build trip-type buckets, resolving each band's high edge from the km
	// edge table (null for the open-ended top band).
	tripTypes := make([]dto.V2BehaviorTripTypeBucket, 0, len(rawTrips))
	for _, t := range rawTrips {
		bucket := dto.V2BehaviorTripTypeBucket{
			Key:        strconv.Itoa(int(t.low)),
			DistLow:    t.low,
			TripsCount: t.trips,
			Distance:   t.dist,
			EnergyKWh:  t.kwh,
		}
		if high, ok := nextTripEdge(t.low); ok {
			bucket.DistHigh = NullFloat64{NullFloat64: sql.NullFloat64{Float64: high, Valid: true}}
		}
		tripTypes = append(tripTypes, bucket)
	}

	// Distance is the only unit-dependent figure; convert km→mi for display.
	// Bucket edges convert too, so a "5–10 km" band reads "3.1–6.2 mi" — the
	// same trade-off consumption makes for temperature bands.
	if UnitsLength == "mi" {
		for i := range heatmap {
			heatmap[i].DrivesDistance = convert.KilometersToMiles(heatmap[i].DrivesDistance)
		}
		for i := range tripTypes {
			tripTypes[i].Distance = convert.KilometersToMiles(tripTypes[i].Distance)
			tripTypes[i].DistLow = convert.KilometersToMiles(tripTypes[i].DistLow)
			if tripTypes[i].DistHigh.Valid {
				tripTypes[i].DistHigh.Float64 = convert.KilometersToMiles(tripTypes[i].DistHigh.Float64)
			}
		}
	}

	respond.HandleSuccess(c, handler, dto.V2BehaviorResponse{
		Data: dto.V2BehaviorData{
			Car:          dto.Car{CarID: CarID, CarName: CarName},
			Heatmap:      heatmap,
			ChargeLevels: chargeLevels,
			TripTypes:    tripTypes,
			Units: dto.TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	})
}

// nextTripEdge returns the upper bound for a trip band whose lower bound is
// low, and false for the open-ended top band.
func nextTripEdge(low float64) (float64, bool) {
	for i, e := range tripDistEdgesKm {
		if e == low && i+1 < len(tripDistEdgesKm) {
			return tripDistEdgesKm[i+1], true
		}
	}
	return 0, false
}
