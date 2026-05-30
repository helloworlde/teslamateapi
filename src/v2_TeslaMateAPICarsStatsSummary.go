package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsStatsSummaryV2 returns aggregates bucketed by day / week
// / month / year. Buckets land in the user's configured timezone (the same
// `appUsersTimezone` used elsewhere) so a "month" really is a month in the
// user's calendar.
//
// Query params:
//
//	period     day | week | month | year   (default: month)
//	startDate  RFC3339 (or YYYY-MM-DD HH:MM:SS in user TZ)
//	endDate    RFC3339
//
// Each bucket carries drive / charge / parking aggregates so the client can
// render a single bar / line chart without further joins. Empty buckets are
// omitted — clients that need a dense series should fill in zero buckets
// from the timeline they want to display.
//
// @Summary      Period summary stats
// @Description  Aggregates bucketed by day / week / month / year in the user's timezone.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID       path   int     true   "TeslaMate cars.id"
// @Param        period      query  string  false  "bucket size"  Enums(day,week,month,year)  default(month)
// @Param        start_date  query  string  false  "RFC3339 lower bound on bucket start"
// @Param        end_date    query  string  false  "RFC3339 upper bound on bucket start"
// @Success      200  {object}  dto.V2SummaryResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/stats/summary [get]
func TeslaMateAPICarsStatsSummaryV2(c *gin.Context) {

	const handler = "TeslaMateAPICarsStatsSummaryV2"
	var ErrMsg = "Unable to load summary stats."
	var ErrDate = "Invalid date format."

	CarID, ok := v2RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}

	period := c.DefaultQuery("period", "month")
	pgUnit := ""
	switch period {
	case "day":
		pgUnit = "day"
	case "week":
		pgUnit = "week"
	case "month":
		pgUnit = "month"
	case "year":
		pgUnit = "year"
	default:
		v2HandleErrorResponse(c, handler, http.StatusBadRequest, "Invalid period.",
			"period must be one of: day, week, month, year")
		return
	}

	parsedStartDate, err := parseDateParam(c.Query("start_date"))
	if err != nil {
		v2HandleErrorResponse(c, handler, http.StatusBadRequest, ErrDate, err.Error())
		return
	}
	parsedEndDate, err := parseDateParam(c.Query("end_date"))
	if err != nil {
		v2HandleErrorResponse(c, handler, http.StatusBadRequest, ErrDate, err.Error())
		return
	}

	type Bucket struct {
		BucketStart                NullString `json:"bucket_start"`
		BucketEnd                  NullString `json:"bucket_end"`
		DrivesCount                int     `json:"drives_count"`
		DrivesDistance             float64 `json:"drives_distance"`
		DrivesDurationMin          int     `json:"drives_duration_min"`
		DrivesEnergyConsumedKWh    float64 `json:"drives_energy_consumed_kwh"`
		DrivesAvgConsumption       float64 `json:"drives_avg_consumption"`
		ChargesCount               int     `json:"charges_count"`
		ChargesEnergyAddedKWh      float64 `json:"charges_energy_added_kwh"`
		ChargesCost                float64 `json:"charges_cost"`
		FastChargeRatio            float64 `json:"fast_charge_ratio"`
		ParkingsTotalDurationMin   int     `json:"parkings_total_duration_min"`
		VampireDrainKWh            float64 `json:"vampire_drain_kwh"`
	}
	type Car struct {
		CarID   int        `json:"car_id"`
		CarName NullString `json:"car_name"`
	}
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`
		UnitsTemperature string `json:"unit_of_temperature"`
	}
	type Data struct {
		Car     Car            `json:"car"`
		Period  string         `json:"period"`
		Buckets []Bucket       `json:"buckets"`
		Units   TeslaMateUnits `json:"units"`
	}
	type JSONData struct {
		Data Data `json:"data"`
	}

	// We collect bucket keys from drives, charges, and parking-pairs; full-outer
	// join via a UNION-of-keys CTE.
	//
	// Parameters: $1=car_id, $2=period_unit, $3=tz_name, $4=startDate?, $5=endDate?
	//
	// Bucket key uses date_trunc(unit, date AT TIME ZONE tz). We coalesce
	// drive timestamps into the start_date for bucketing.
	tzName := appUsersTimezone.String()

	var args []any
	args = append(args, CarID, pgUnit, tzName)
	paramIdx := 4

	dateFilterDrives := ""
	dateFilterCharges := ""
	dateFilterParkings := ""
	if parsedStartDate != "" {
		dateFilterDrives += fmt.Sprintf(" AND d.start_date >= $%d", paramIdx)
		dateFilterCharges += fmt.Sprintf(" AND cp.start_date >= $%d", paramIdx)
		dateFilterParkings += fmt.Sprintf(" AND park.park_start >= $%d", paramIdx)
		args = append(args, parsedStartDate)
		paramIdx++
	}
	if parsedEndDate != "" {
		dateFilterDrives += fmt.Sprintf(" AND d.start_date <= $%d", paramIdx)
		dateFilterCharges += fmt.Sprintf(" AND cp.start_date <= $%d", paramIdx)
		dateFilterParkings += fmt.Sprintf(" AND park.park_start <= $%d", paramIdx)
		args = append(args, parsedEndDate)
		paramIdx++
	}

	query := fmt.Sprintf(`
		WITH drv AS (
			SELECT
				date_trunc($2, d.start_date AT TIME ZONE $3) AS bk,
				COUNT(*) AS cnt,
				SUM(distance) AS dist,
				SUM(duration_min) AS dur,
				SUM(
					CASE WHEN start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency
					ELSE 0 END
				) AS kwh,
				SUM(distance) AS dist_for_avg
			FROM drives d
			LEFT JOIN cars ON cars.id = d.car_id
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL %s
			GROUP BY 1
		),
		ch AS (
			SELECT
				date_trunc($2, cp.start_date AT TIME ZONE $3) AS bk,
				COUNT(*) AS cnt,
				SUM(charge_energy_added) AS added,
				SUM(cost) AS cost,
				-- ratio is energy-weighted: fast-charged kWh / total kWh in the bucket
				COALESCE(
					SUM(charge_energy_added) FILTER (
						WHERE EXISTS(SELECT 1 FROM charges c WHERE c.charging_process_id = cp.id AND c.fast_charger_present)
					) / NULLIF(SUM(charge_energy_added), 0),
					0
				) AS fast_ratio
			FROM charging_processes cp
			WHERE cp.car_id = $1 AND cp.end_date IS NOT NULL %s
			GROUP BY 1
		),
		dp AS (
			SELECT
				d.id AS drive_id,
				d.end_date AS park_start,
				LEAD(d.start_date) OVER w AS park_end,
				d.end_position_id,
				LEAD(d.start_position_id) OVER w AS next_start_position_id
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			WINDOW w AS (PARTITION BY d.car_id ORDER BY d.start_date ASC)
		),
		park AS (
			SELECT
				dp.park_start,
				dp.park_end,
				dp.end_position_id,
				dp.next_start_position_id,
				COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60, EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60)::int AS dur,
				CASE
					WHEN sp.rated_battery_range_km IS NOT NULL AND ep.rated_battery_range_km IS NOT NULL
					AND NOT EXISTS(SELECT 1 FROM charging_processes cp
						WHERE cp.car_id = $1 AND cp.start_date >= dp.park_start
						AND (dp.park_end IS NULL OR cp.start_date < dp.park_end))
					THEN GREATEST(sp.rated_battery_range_km - ep.rated_battery_range_km, 0) * cars.efficiency
					ELSE 0
				END AS drop_kwh
			FROM dp
			LEFT JOIN cars ON cars.id = $1
			LEFT JOIN positions sp ON sp.id = dp.end_position_id
			LEFT JOIN positions ep ON ep.id = dp.next_start_position_id
		),
		pk AS (
			SELECT
				date_trunc($2, park.park_start AT TIME ZONE $3) AS bk,
				SUM(park.dur) AS dur,
				SUM(park.drop_kwh) AS drop_kwh
			FROM park
			WHERE 1=1 %s
			GROUP BY 1
		),
		keys AS (
			SELECT bk FROM drv
			UNION
			SELECT bk FROM ch
			UNION
			SELECT bk FROM pk
		)
		SELECT
			-- date_trunc on (timestamptz AT TIME ZONE tz) returns a tz-naive timestamp
			-- representing wall-clock in user tz; cast back AT TIME ZONE tz so pq scans
			-- a real UTC instant. getTimeInTimeZone then formats it in user tz exactly once.
			(k.bk AT TIME ZONE $3) AS bucket_start,
			((k.bk + (CASE $2
				WHEN 'day' THEN INTERVAL '1 day'
				WHEN 'week' THEN INTERVAL '1 week'
				WHEN 'month' THEN INTERVAL '1 month'
				WHEN 'year' THEN INTERVAL '1 year'
			END)) AT TIME ZONE $3) AS bucket_end,
			COALESCE(drv.cnt, 0),
			COALESCE(drv.dist, 0),
			COALESCE(drv.dur, 0),
			COALESCE(drv.kwh, 0),
			CASE WHEN COALESCE(drv.dist_for_avg, 0) > 0 THEN drv.kwh / drv.dist_for_avg * 1000 ELSE 0 END AS avg_consumption,
			COALESCE(ch.cnt, 0),
			COALESCE(ch.added, 0),
			COALESCE(ch.cost, 0),
			COALESCE(ch.fast_ratio, 0),
			COALESCE(pk.dur, 0)::int,
			COALESCE(pk.drop_kwh, 0),
			(SELECT unit_of_length FROM settings LIMIT 1),
			(SELECT unit_of_temperature FROM settings LIMIT 1),
			(SELECT name FROM cars WHERE id = $1)
		FROM keys k
		LEFT JOIN drv ON drv.bk = k.bk
		LEFT JOIN ch  ON ch.bk  = k.bk
		LEFT JOIN pk  ON pk.bk  = k.bk
		ORDER BY k.bk ASC;`,
		dateFilterDrives, dateFilterCharges, dateFilterParkings,
	)

	rows, err := db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}
	defer rows.Close()

	var (
		buckets                       []Bucket
		UnitsLength, UnitsTemperature string
		CarName                       NullString
	)

	for rows.Next() {
		b := Bucket{}
		if err = rows.Scan(
			&b.BucketStart,
			&b.BucketEnd,
			&b.DrivesCount,
			&b.DrivesDistance,
			&b.DrivesDurationMin,
			&b.DrivesEnergyConsumedKWh,
			&b.DrivesAvgConsumption,
			&b.ChargesCount,
			&b.ChargesEnergyAddedKWh,
			&b.ChargesCost,
			&b.FastChargeRatio,
			&b.ParkingsTotalDurationMin,
			&b.VampireDrainKWh,
			&UnitsLength,
			&UnitsTemperature,
			&CarName,
		); err != nil {
			v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
			return
		}

		if UnitsLength == "mi" {
			b.DrivesDistance = kilometersToMiles(b.DrivesDistance)
			if b.DrivesAvgConsumption > 0 {
				b.DrivesAvgConsumption = b.DrivesAvgConsumption / 0.62137119223733
			}
		}

		// NullString.Scan rendered the bucket timestamps as dbTimestampFormat
		// (UTC); re-format into the user's timezone for display.
		b.BucketStart = NullString(getTimeInTimeZone(string(b.BucketStart)))
		b.BucketEnd = NullString(getTimeInTimeZone(string(b.BucketEnd)))
		buckets = append(buckets, b)
	}
	if err = rows.Err(); err != nil {
		v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, handler, JSONData{
		Data: Data{
			Car:     Car{CarID: CarID, CarName: CarName},
			Period:  period,
			Buckets: buckets,
			Units: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	})
}
