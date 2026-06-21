package v2

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
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
// Drives and charges are bucketed by their `start_date`. Parkings are time
// intervals: each parking is sliced across every bucket it overlaps with,
// and duration + vampire-drain energy are allocated pro-rata. Buckets that
// only carry parking activity still appear in the response (drives_count /
// charges_count = 0). The `start_date` / `end_date` filter clamps both the
// drive/charge events and the parking bucket series, so a parking that
// straddles the window edge only contributes inside the requested range.
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
func (h *Handler) StatsSummary(c *gin.Context) {

	const handler = "TeslaMateAPICarsStatsSummaryV2"
	var ErrMsg = "Unable to load summary stats."
	var ErrDate = "Invalid date format."

	CarID, ok := respond.RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
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
		respond.HandleErrorV2(c, handler, http.StatusBadRequest, "Invalid period.",
			"period must be one of: day, week, month, year")
		return
	}

	parsedStartDate, err := h.parseDate(c.Query("start_date"))
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusBadRequest, ErrDate, err.Error())
		return
	}
	parsedEndDate, err := h.parseDate(c.Query("end_date"))
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusBadRequest, ErrDate, err.Error())
		return
	}

	type Bucket struct {
		BucketStart                     NullString `json:"bucket_start"`
		BucketEnd                       NullString `json:"bucket_end"`
		DrivesCount                     int        `json:"drives_count"`
		DrivesDistance                  float64    `json:"drives_distance"`
		DrivesDurationMin               int        `json:"drives_duration_min"`
		DrivesEnergyConsumedKWh         float64    `json:"drives_energy_consumed_kwh"`
		DrivesAvgConsumption            float64    `json:"drives_avg_consumption"`
		DrivesLongestDistance           float64    `json:"drives_longest_distance"`
		DrivesLongestDurationMin        int        `json:"drives_longest_duration_min"`
		DrivesMaxSpeed                  int        `json:"drives_max_speed"`
		DrivesBestConsumption           float64    `json:"drives_best_consumption"`
		DrivesWorstConsumption          float64    `json:"drives_worst_consumption"`
		DrivesPeakDrivePowerKW          int        `json:"drives_peak_drive_power_kw"`
		DrivesPeakRegenPowerKW          int        `json:"drives_peak_regen_power_kw"`
		DrivesLongestDistanceStartDate  NullString `json:"drives_longest_distance_start_date"`
		DrivesLongestDistanceEndDate    NullString `json:"drives_longest_distance_end_date"`
		DrivesLongestDurationStartDate  NullString `json:"drives_longest_duration_start_date"`
		DrivesLongestDurationEndDate    NullString `json:"drives_longest_duration_end_date"`
		DrivesMaxSpeedStartDate         NullString `json:"drives_max_speed_start_date"`
		DrivesMaxSpeedEndDate           NullString `json:"drives_max_speed_end_date"`
		DrivesBestConsumptionStartDate  NullString `json:"drives_best_consumption_start_date"`
		DrivesBestConsumptionEndDate    NullString `json:"drives_best_consumption_end_date"`
		DrivesWorstConsumptionStartDate NullString `json:"drives_worst_consumption_start_date"`
		DrivesWorstConsumptionEndDate   NullString `json:"drives_worst_consumption_end_date"`
		DrivesPeakDrivePowerStartDate   NullString `json:"drives_peak_drive_power_start_date"`
		DrivesPeakDrivePowerEndDate     NullString `json:"drives_peak_drive_power_end_date"`
		DrivesPeakRegenPowerStartDate   NullString `json:"drives_peak_regen_power_start_date"`
		DrivesPeakRegenPowerEndDate     NullString `json:"drives_peak_regen_power_end_date"`
		ChargesCount                    int        `json:"charges_count"`
		ChargesEnergyAddedKWh           float64    `json:"charges_energy_added_kwh"`
		ChargesEnergyUsedKWh            float64    `json:"charges_energy_used_kwh"`
		ChargesDurationMin              int        `json:"charges_duration_min"`
		ChargesCost                     float64    `json:"charges_cost"`
		ChargesACCount                  int        `json:"charges_ac_count"`
		ChargesDCCount                  int        `json:"charges_dc_count"`
		ChargesACEnergyAddedKWh         float64    `json:"charges_ac_energy_added_kwh"`
		ChargesDCEnergyAddedKWh         float64    `json:"charges_dc_energy_added_kwh"`
		ChargesACEnergyUsedKWh          float64    `json:"charges_ac_energy_used_kwh"`
		ChargesDCEnergyUsedKWh          float64    `json:"charges_dc_energy_used_kwh"`
		ChargesACDurationMin            int        `json:"charges_ac_duration_min"`
		ChargesDCDurationMin            int        `json:"charges_dc_duration_min"`
		ChargesACCost                   float64    `json:"charges_ac_cost"`
		ChargesDCCost                   float64    `json:"charges_dc_cost"`
		ChargesAvgDurationACMin         float64    `json:"charges_avg_duration_ac_min"`
		ChargesAvgDurationDCMin         float64    `json:"charges_avg_duration_dc_min"`
		ChargesAvgEnergyPerACSessionKWh float64    `json:"charges_avg_energy_per_ac_session_kwh"`
		ChargesAvgEnergyPerDCSessionKWh float64    `json:"charges_avg_energy_per_dc_session_kwh"`
		ChargesAvgSessionCostAC         float64    `json:"charges_avg_session_cost_ac"`
		ChargesAvgSessionCostDC         float64    `json:"charges_avg_session_cost_dc"`
		ChargesAvgCostPerKWhAC          float64    `json:"charges_avg_cost_per_kwh_ac"`
		ChargesAvgCostPerKWhDC          float64    `json:"charges_avg_cost_per_kwh_dc"`
		FastChargeRatio                 float64    `json:"fast_charge_ratio"`
		ChargesLongestSessionMin        int        `json:"charges_longest_session_duration_min"`
		ChargesLargestSessionKWh        float64    `json:"charges_largest_session_kwh"`
		ChargesMaxSessionCost           float64    `json:"charges_max_session_cost"`
		ChargesMaxPowerKW               int        `json:"charges_max_power_kw"`
		ChargesLongestSessionStartDate  NullString `json:"charges_longest_session_start_date"`
		ChargesLongestSessionEndDate    NullString `json:"charges_longest_session_end_date"`
		ChargesLargestSessionStartDate  NullString `json:"charges_largest_session_start_date"`
		ChargesLargestSessionEndDate    NullString `json:"charges_largest_session_end_date"`
		ChargesMaxSessionCostStartDate  NullString `json:"charges_max_session_cost_start_date"`
		ChargesMaxSessionCostEndDate    NullString `json:"charges_max_session_cost_end_date"`
		ChargesMaxPowerDate             NullString `json:"charges_max_power_date"`
		ChargesAvgPowerACKW             float64    `json:"charges_avg_power_ac_kw"`
		ChargesAvgPowerDCKW             float64    `json:"charges_avg_power_dc_kw"`
		ParkingsTotalDurationMin        int        `json:"parkings_total_duration_min"`
		VampireDrainKWh                 float64    `json:"vampire_drain_kwh"`
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
	// Bucket key uses date_trunc(unit, ts_local), where ts_local converts the
	// stored UTC timestamp into the user's timezone. Drives/charges are
	// single-point events bucketed by start; parkings are intervals sliced
	// across every bucket they overlap, with duration and rated-range drop
	// allocated pro-rata.
	tzName := h.tz.String()

	var args []any
	args = append(args, CarID, pgUnit, tzName)
	paramIdx := 4

	dateFilterDrives := ""
	dateFilterCharges := ""
	dateFilterParkings := ""
	startIdx, endIdx := 0, 0
	if parsedStartDate != "" {
		startIdx = paramIdx
		dateFilterDrives += fmt.Sprintf(" AND d.start_date >= $%d", paramIdx)
		dateFilterCharges += fmt.Sprintf(" AND cp.start_date >= $%d", paramIdx)
		dateFilterParkings += fmt.Sprintf(" AND COALESCE(dp.park_end, %s) > $%d::timestamp", utcNowTimestampSQL(), paramIdx)
		args = append(args, parsedStartDate)
		paramIdx++
	}
	if parsedEndDate != "" {
		endIdx = paramIdx
		dateFilterDrives += fmt.Sprintf(" AND d.start_date <= $%d", paramIdx)
		dateFilterCharges += fmt.Sprintf(" AND cp.start_date <= $%d", paramIdx)
		dateFilterParkings += fmt.Sprintf(" AND dp.park_start <= $%d::timestamp", paramIdx)
		args = append(args, parsedEndDate)
		paramIdx++
	}

	localDriveStart := localTimestampSQL("d.start_date", "$3")
	localChargeStart := localTimestampSQL("cp.start_date", "$3")
	localParkStart := localTimestampSQL("p.park_start", "$3")
	localParkEndExclusive := localTimestampSQL("(p.park_end - INTERVAL '1 microsecond')", "$3")
	step := bucketStepSQL("$2")
	utcNow := utcNowTimestampSQL()

	// Clamp the parking bucket series to the requested [startDate, endDate]
	// window. Without this clamp, a parking that straddles the window edge
	// (e.g. started before startDate, ends after endDate) would generate `pk`
	// rows for every bucket it touches — including buckets outside the
	// requested range — and they would leak into the final UNION-of-keys.
	seriesStart := fmt.Sprintf("date_trunc($2, %s)", localParkStart)
	seriesEnd := fmt.Sprintf("date_trunc($2, %s)", localParkEndExclusive)
	if startIdx > 0 {
		startLocal := localTimestampSQL(fmt.Sprintf("$%d::timestamp", startIdx), "$3")
		seriesStart = fmt.Sprintf("GREATEST(%s, date_trunc($2, %s))", seriesStart, startLocal)
	}
	if endIdx > 0 {
		endLocal := localTimestampSQL(fmt.Sprintf("$%d::timestamp", endIdx), "$3")
		seriesEnd = fmt.Sprintf("LEAST(%s, date_trunc($2, %s))", seriesEnd, endLocal)
	}
	query := fmt.Sprintf(`
		WITH drv AS (
			SELECT
				date_trunc($2, %[1]s) AS bk,
				COUNT(*) AS cnt,
				SUM(distance) AS dist,
				SUM(duration_min) AS dur,
				SUM(
					CASE WHEN start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency
					ELSE 0 END
				) AS kwh,
				SUM(distance) AS dist_for_avg,
				COALESCE(MAX(distance), 0) AS longest_dist,
				COALESCE(MAX(duration_min), 0) AS longest_dur,
				COALESCE(MAX(speed_max), 0) AS max_speed,
				COALESCE(MAX(power_max), 0) AS peak_drive_power,
				COALESCE(-MIN(power_min), 0) AS peak_regen_power,
				MIN(
					CASE WHEN distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency / distance * 1000
					ELSE NULL END
				) AS best_consumption,
				MAX(
					CASE WHEN distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency / distance * 1000
					ELSE NULL END
				) AS worst_consumption,
				(array_agg(d.start_date ORDER BY distance DESC NULLS LAST, d.start_date ASC) FILTER (WHERE distance IS NOT NULL))[1] AS longest_distance_start_date,
				(array_agg(d.end_date ORDER BY distance DESC NULLS LAST, d.start_date ASC) FILTER (WHERE distance IS NOT NULL))[1] AS longest_distance_end_date,
				(array_agg(d.start_date ORDER BY duration_min DESC NULLS LAST, d.start_date ASC) FILTER (WHERE duration_min IS NOT NULL))[1] AS longest_duration_start_date,
				(array_agg(d.end_date ORDER BY duration_min DESC NULLS LAST, d.start_date ASC) FILTER (WHERE duration_min IS NOT NULL))[1] AS longest_duration_end_date,
				(array_agg(d.start_date ORDER BY speed_max DESC NULLS LAST, d.start_date ASC) FILTER (WHERE speed_max IS NOT NULL))[1] AS max_speed_start_date,
				(array_agg(d.end_date ORDER BY speed_max DESC NULLS LAST, d.start_date ASC) FILTER (WHERE speed_max IS NOT NULL))[1] AS max_speed_end_date,
				(array_agg(d.start_date ORDER BY (
					CASE WHEN distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency / distance * 1000
					ELSE NULL END
				) ASC NULLS LAST, d.start_date ASC) FILTER (
					WHERE distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
				))[1] AS best_consumption_start_date,
				(array_agg(d.end_date ORDER BY (
					CASE WHEN distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency / distance * 1000
					ELSE NULL END
				) ASC NULLS LAST, d.start_date ASC) FILTER (
					WHERE distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
				))[1] AS best_consumption_end_date,
				(array_agg(d.start_date ORDER BY (
					CASE WHEN distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency / distance * 1000
					ELSE NULL END
				) DESC NULLS LAST, d.start_date ASC) FILTER (
					WHERE distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
				))[1] AS worst_consumption_start_date,
				(array_agg(d.end_date ORDER BY (
					CASE WHEN distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
					THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency / distance * 1000
					ELSE NULL END
				) DESC NULLS LAST, d.start_date ASC) FILTER (
					WHERE distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL
					AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0
				))[1] AS worst_consumption_end_date,
				(array_agg(d.start_date ORDER BY power_max DESC NULLS LAST, d.start_date ASC) FILTER (WHERE power_max IS NOT NULL))[1] AS peak_drive_power_start_date,
				(array_agg(d.end_date ORDER BY power_max DESC NULLS LAST, d.start_date ASC) FILTER (WHERE power_max IS NOT NULL))[1] AS peak_drive_power_end_date,
				(array_agg(d.start_date ORDER BY power_min ASC NULLS LAST, d.start_date ASC) FILTER (WHERE power_min IS NOT NULL))[1] AS peak_regen_power_start_date,
				(array_agg(d.end_date ORDER BY power_min ASC NULLS LAST, d.start_date ASC) FILTER (WHERE power_min IS NOT NULL))[1] AS peak_regen_power_end_date
			FROM drives d
			LEFT JOIN cars ON cars.id = d.car_id
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL %[7]s
			GROUP BY 1
		),
		ch AS (
			SELECT
				bk,
				COUNT(*) AS cnt,
				SUM(charge_energy_added) AS added,
				SUM(GREATEST(charge_energy_used, charge_energy_added)) AS used,
				SUM(duration_min)::int AS dur,
				SUM(cost) AS cost,
				COUNT(*) FILTER (WHERE NOT fast_present) AS ac_count,
				COUNT(*) FILTER (WHERE fast_present) AS dc_count,
				COALESCE(SUM(charge_energy_added) FILTER (WHERE NOT fast_present), 0) AS ac_added,
				COALESCE(SUM(charge_energy_added) FILTER (WHERE fast_present), 0) AS dc_added,
				COALESCE(SUM(GREATEST(charge_energy_used, charge_energy_added)) FILTER (WHERE NOT fast_present), 0) AS ac_used,
				COALESCE(SUM(GREATEST(charge_energy_used, charge_energy_added)) FILTER (WHERE fast_present), 0) AS dc_used,
				COALESCE(SUM(duration_min) FILTER (WHERE NOT fast_present), 0)::int AS ac_dur,
				COALESCE(SUM(duration_min) FILTER (WHERE fast_present), 0)::int AS dc_dur,
				COALESCE(SUM(cost) FILTER (WHERE NOT fast_present), 0) AS ac_cost,
				COALESCE(SUM(cost) FILTER (WHERE fast_present), 0) AS dc_cost,
				COALESCE(AVG(duration_min) FILTER (WHERE NOT fast_present), 0) AS avg_duration_ac,
				COALESCE(AVG(duration_min) FILTER (WHERE fast_present), 0) AS avg_duration_dc,
				COALESCE(AVG(charge_energy_added) FILTER (WHERE NOT fast_present), 0) AS avg_energy_ac,
				COALESCE(AVG(charge_energy_added) FILTER (WHERE fast_present), 0) AS avg_energy_dc,
				COALESCE(AVG(cost) FILTER (WHERE NOT fast_present AND cost IS NOT NULL), 0) AS avg_session_cost_ac,
				COALESCE(AVG(cost) FILTER (WHERE fast_present AND cost IS NOT NULL), 0) AS avg_session_cost_dc,
				CASE WHEN SUM(charge_energy_added) FILTER (WHERE NOT fast_present) > 0
					THEN COALESCE(SUM(cost) FILTER (WHERE NOT fast_present), 0)
						/ NULLIF(SUM(charge_energy_added) FILTER (WHERE NOT fast_present), 0)
					ELSE 0 END AS avg_cost_per_kwh_ac,
				CASE WHEN SUM(charge_energy_added) FILTER (WHERE fast_present) > 0
					THEN COALESCE(SUM(cost) FILTER (WHERE fast_present), 0)
						/ NULLIF(SUM(charge_energy_added) FILTER (WHERE fast_present), 0)
					ELSE 0 END AS avg_cost_per_kwh_dc,
				COALESCE(
					SUM(charge_energy_added) FILTER (WHERE fast_present)
					/ NULLIF(SUM(charge_energy_added), 0),
					0
				) AS fast_ratio,
				COALESCE(MAX(duration_min), 0)::int AS longest_session_dur,
				COALESCE(MAX(charge_energy_added), 0) AS largest_session_energy,
				COALESCE(MAX(cost), 0) AS max_session_cost,
				COALESCE(MAX(peak_pw), 0)::int AS max_power,
				(array_agg(start_date ORDER BY duration_min DESC NULLS LAST, start_date ASC) FILTER (WHERE duration_min IS NOT NULL))[1] AS longest_session_start_date,
				(array_agg(end_date ORDER BY duration_min DESC NULLS LAST, start_date ASC) FILTER (WHERE duration_min IS NOT NULL))[1] AS longest_session_end_date,
				(array_agg(start_date ORDER BY charge_energy_added DESC NULLS LAST, start_date ASC) FILTER (WHERE charge_energy_added IS NOT NULL))[1] AS largest_session_start_date,
				(array_agg(end_date ORDER BY charge_energy_added DESC NULLS LAST, start_date ASC) FILTER (WHERE charge_energy_added IS NOT NULL))[1] AS largest_session_end_date,
				(array_agg(start_date ORDER BY cost DESC NULLS LAST, start_date ASC) FILTER (WHERE cost IS NOT NULL))[1] AS max_session_cost_start_date,
				(array_agg(end_date ORDER BY cost DESC NULLS LAST, start_date ASC) FILTER (WHERE cost IS NOT NULL))[1] AS max_session_cost_end_date,
				(array_agg(peak_pw_date ORDER BY peak_pw DESC NULLS LAST, start_date ASC) FILTER (WHERE peak_pw IS NOT NULL))[1] AS max_power_date,
				COALESCE(AVG(peak_pw) FILTER (WHERE NOT fast_present), 0) AS avg_power_ac,
				COALESCE(AVG(peak_pw) FILTER (WHERE fast_present), 0) AS avg_power_dc
			FROM (
				SELECT
					date_trunc($2, %[2]s) AS bk,
					cp.charge_energy_added,
					cp.charge_energy_used,
					cp.duration_min,
					cp.cost,
					cp.start_date,
					cp.end_date,
					COALESCE(chg.fast_present, false) AS fast_present,
					chg.peak_pw,
					chg.peak_pw_date
				FROM charging_processes cp
				LEFT JOIN LATERAL (
					SELECT
						bool_or(fast_charger_present) AS fast_present,
						MAX(charger_power) AS peak_pw,
						(array_agg(date ORDER BY charger_power DESC NULLS LAST, date ASC) FILTER (WHERE charger_power IS NOT NULL))[1] AS peak_pw_date
					FROM charges c
					WHERE c.charging_process_id = cp.id
				) chg ON true
				WHERE cp.car_id = $1 AND cp.end_date IS NOT NULL %[8]s
			) sub
			GROUP BY bk
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
		park_intervals AS (
			SELECT
				dp.park_start,
				COALESCE(dp.park_end, %[6]s) AS park_end,
				dp.end_position_id,
				dp.next_start_position_id,
				CASE
					WHEN sp.rated_battery_range_km IS NOT NULL AND ep.rated_battery_range_km IS NOT NULL
					AND sp.rated_battery_range_km > ep.rated_battery_range_km
					AND NOT EXISTS(SELECT 1 FROM charging_processes cp
						WHERE cp.car_id = $1 AND cp.start_date >= dp.park_start
						AND (dp.park_end IS NULL OR cp.start_date < dp.park_end))
					THEN (sp.rated_battery_range_km - ep.rated_battery_range_km) * cars.efficiency
					WHEN sp.rated_battery_range_km IS NOT NULL
					AND COALESCE(sp.usable_battery_level, sp.battery_level) > 0
					AND NOT EXISTS(SELECT 1 FROM charging_processes cp
						WHERE cp.car_id = $1 AND cp.start_date >= dp.park_start
						AND (dp.park_end IS NULL OR cp.start_date < dp.park_end))
					THEN GREATEST(
						COALESCE(sp.usable_battery_level, sp.battery_level)
						- COALESCE(ep.usable_battery_level, ep.battery_level),
						0
					) * sp.rated_battery_range_km * cars.efficiency
						/ COALESCE(sp.usable_battery_level, sp.battery_level)
					ELSE 0
				END AS drop_kwh
			FROM dp
			LEFT JOIN cars ON cars.id = $1
			LEFT JOIN positions sp ON sp.id = dp.end_position_id
			LEFT JOIN positions ep ON ep.id = dp.next_start_position_id
			WHERE COALESCE(dp.park_end, %[6]s) > dp.park_start %[9]s
		),
		park_slices AS (
			SELECT
				bucket.bk,
				GREATEST(p.park_start, ((bucket.bk AT TIME ZONE $3) AT TIME ZONE 'UTC')) AS slice_start,
				LEAST(p.park_end, (((bucket.bk + %[5]s) AT TIME ZONE $3) AT TIME ZONE 'UTC')) AS slice_end,
				p.drop_kwh,
				EXTRACT(EPOCH FROM (p.park_end - p.park_start)) AS total_seconds
			FROM park_intervals p
			CROSS JOIN LATERAL generate_series(
				%[3]s,
				%[4]s,
				%[5]s
			) AS bucket(bk)
		),
		pk AS (
			SELECT
				bk,
				SUM(EXTRACT(EPOCH FROM (slice_end - slice_start)) / 60)::int AS dur,
				SUM(
					CASE WHEN total_seconds > 0
					THEN drop_kwh * EXTRACT(EPOCH FROM (slice_end - slice_start)) / total_seconds
					ELSE 0 END
				) AS drop_kwh
			FROM park_slices
			WHERE slice_end > slice_start
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
			-- a real UTC instant. h.timeInTZ then formats it in user tz exactly once.
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
			COALESCE(drv.longest_dist, 0),
			COALESCE(drv.longest_dur, 0),
			COALESCE(drv.max_speed, 0),
			COALESCE(drv.best_consumption, 0),
			COALESCE(drv.worst_consumption, 0),
			COALESCE(drv.peak_drive_power, 0)::int,
			COALESCE(drv.peak_regen_power, 0)::int,
			drv.longest_distance_start_date, drv.longest_distance_end_date,
			drv.longest_duration_start_date, drv.longest_duration_end_date,
			drv.max_speed_start_date, drv.max_speed_end_date,
			drv.best_consumption_start_date, drv.best_consumption_end_date,
			drv.worst_consumption_start_date, drv.worst_consumption_end_date,
			drv.peak_drive_power_start_date, drv.peak_drive_power_end_date,
			drv.peak_regen_power_start_date, drv.peak_regen_power_end_date,
			COALESCE(ch.cnt, 0),
			COALESCE(ch.added, 0),
			COALESCE(ch.used, 0),
			COALESCE(ch.dur, 0),
			COALESCE(ch.cost, 0),
			COALESCE(ch.ac_count, 0),
			COALESCE(ch.dc_count, 0),
			COALESCE(ch.ac_added, 0),
			COALESCE(ch.dc_added, 0),
			COALESCE(ch.ac_used, 0),
			COALESCE(ch.dc_used, 0),
			COALESCE(ch.ac_dur, 0),
			COALESCE(ch.dc_dur, 0),
			COALESCE(ch.ac_cost, 0),
			COALESCE(ch.dc_cost, 0),
			COALESCE(ch.avg_duration_ac, 0),
			COALESCE(ch.avg_duration_dc, 0),
			COALESCE(ch.avg_energy_ac, 0),
			COALESCE(ch.avg_energy_dc, 0),
			COALESCE(ch.avg_session_cost_ac, 0),
			COALESCE(ch.avg_session_cost_dc, 0),
			COALESCE(ch.avg_cost_per_kwh_ac, 0),
			COALESCE(ch.avg_cost_per_kwh_dc, 0),
			COALESCE(ch.fast_ratio, 0),
			COALESCE(ch.longest_session_dur, 0),
			COALESCE(ch.largest_session_energy, 0),
			COALESCE(ch.max_session_cost, 0),
			COALESCE(ch.max_power, 0),
			ch.longest_session_start_date, ch.longest_session_end_date,
			ch.largest_session_start_date, ch.largest_session_end_date,
			ch.max_session_cost_start_date, ch.max_session_cost_end_date,
			ch.max_power_date,
			COALESCE(ch.avg_power_ac, 0),
			COALESCE(ch.avg_power_dc, 0),
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
		localDriveStart,
		localChargeStart,
		seriesStart,
		seriesEnd,
		step,
		utcNow,
		dateFilterDrives,
		dateFilterCharges,
		dateFilterParkings,
	)

	rows, err := h.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
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
			&b.DrivesLongestDistance,
			&b.DrivesLongestDurationMin,
			&b.DrivesMaxSpeed,
			&b.DrivesBestConsumption,
			&b.DrivesWorstConsumption,
			&b.DrivesPeakDrivePowerKW,
			&b.DrivesPeakRegenPowerKW,
			&b.DrivesLongestDistanceStartDate, &b.DrivesLongestDistanceEndDate,
			&b.DrivesLongestDurationStartDate, &b.DrivesLongestDurationEndDate,
			&b.DrivesMaxSpeedStartDate, &b.DrivesMaxSpeedEndDate,
			&b.DrivesBestConsumptionStartDate, &b.DrivesBestConsumptionEndDate,
			&b.DrivesWorstConsumptionStartDate, &b.DrivesWorstConsumptionEndDate,
			&b.DrivesPeakDrivePowerStartDate, &b.DrivesPeakDrivePowerEndDate,
			&b.DrivesPeakRegenPowerStartDate, &b.DrivesPeakRegenPowerEndDate,
			&b.ChargesCount,
			&b.ChargesEnergyAddedKWh,
			&b.ChargesEnergyUsedKWh,
			&b.ChargesDurationMin,
			&b.ChargesCost,
			&b.ChargesACCount,
			&b.ChargesDCCount,
			&b.ChargesACEnergyAddedKWh,
			&b.ChargesDCEnergyAddedKWh,
			&b.ChargesACEnergyUsedKWh,
			&b.ChargesDCEnergyUsedKWh,
			&b.ChargesACDurationMin,
			&b.ChargesDCDurationMin,
			&b.ChargesACCost,
			&b.ChargesDCCost,
			&b.ChargesAvgDurationACMin,
			&b.ChargesAvgDurationDCMin,
			&b.ChargesAvgEnergyPerACSessionKWh,
			&b.ChargesAvgEnergyPerDCSessionKWh,
			&b.ChargesAvgSessionCostAC,
			&b.ChargesAvgSessionCostDC,
			&b.ChargesAvgCostPerKWhAC,
			&b.ChargesAvgCostPerKWhDC,
			&b.FastChargeRatio,
			&b.ChargesLongestSessionMin,
			&b.ChargesLargestSessionKWh,
			&b.ChargesMaxSessionCost,
			&b.ChargesMaxPowerKW,
			&b.ChargesLongestSessionStartDate, &b.ChargesLongestSessionEndDate,
			&b.ChargesLargestSessionStartDate, &b.ChargesLargestSessionEndDate,
			&b.ChargesMaxSessionCostStartDate, &b.ChargesMaxSessionCostEndDate,
			&b.ChargesMaxPowerDate,
			&b.ChargesAvgPowerACKW,
			&b.ChargesAvgPowerDCKW,
			&b.ParkingsTotalDurationMin,
			&b.VampireDrainKWh,
			&UnitsLength,
			&UnitsTemperature,
			&CarName,
		); err != nil {
			respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
			return
		}

		// avg_consumption / best_consumption / worst_consumption are Wh/(distance);
		// converting Wh/km → Wh/mi means dividing by 0.62137... (distance unit
		// stretches, energy stays).
		if UnitsLength == "mi" {
			b.DrivesDistance = convert.KilometersToMiles(b.DrivesDistance)
			b.DrivesLongestDistance = convert.KilometersToMiles(b.DrivesLongestDistance)
			b.DrivesMaxSpeed = convert.KilometersToMilesInteger(b.DrivesMaxSpeed)
			if b.DrivesAvgConsumption > 0 {
				b.DrivesAvgConsumption = b.DrivesAvgConsumption / 0.62137119223733
			}
			if b.DrivesBestConsumption > 0 {
				b.DrivesBestConsumption = b.DrivesBestConsumption / 0.62137119223733
			}
			if b.DrivesWorstConsumption > 0 {
				b.DrivesWorstConsumption = b.DrivesWorstConsumption / 0.62137119223733
			}
		}

		// NullString.Scan rendered the bucket timestamps as dbTimestampFormat
		// (UTC); re-format into the user's timezone for display.
		localize := func(value *NullString) {
			if len(*value) > 0 {
				*value = NullString(h.timeInTZ(string(*value)))
			}
		}
		localize(&b.BucketStart)
		localize(&b.BucketEnd)
		localize(&b.DrivesLongestDistanceStartDate)
		localize(&b.DrivesLongestDistanceEndDate)
		localize(&b.DrivesLongestDurationStartDate)
		localize(&b.DrivesLongestDurationEndDate)
		localize(&b.DrivesMaxSpeedStartDate)
		localize(&b.DrivesMaxSpeedEndDate)
		localize(&b.DrivesBestConsumptionStartDate)
		localize(&b.DrivesBestConsumptionEndDate)
		localize(&b.DrivesWorstConsumptionStartDate)
		localize(&b.DrivesWorstConsumptionEndDate)
		localize(&b.DrivesPeakDrivePowerStartDate)
		localize(&b.DrivesPeakDrivePowerEndDate)
		localize(&b.DrivesPeakRegenPowerStartDate)
		localize(&b.DrivesPeakRegenPowerEndDate)
		localize(&b.ChargesLongestSessionStartDate)
		localize(&b.ChargesLongestSessionEndDate)
		localize(&b.ChargesLargestSessionStartDate)
		localize(&b.ChargesLargestSessionEndDate)
		localize(&b.ChargesMaxSessionCostStartDate)
		localize(&b.ChargesMaxSessionCostEndDate)
		localize(&b.ChargesMaxPowerDate)
		buckets = append(buckets, b)
	}
	if err = rows.Err(); err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	respond.HandleSuccess(c, handler, JSONData{
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
