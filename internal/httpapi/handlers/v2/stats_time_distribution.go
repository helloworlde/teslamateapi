package v2

import (
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// TeslaMateAPICarsStatsTimeDistributionV2 returns server-side aggregates for
// mutually-exclusive driving, parked, and charging time.
//
// Query params:
//
//	startDate  RFC3339 (or YYYY-MM-DD HH:MM:SS in user TZ)
//	endDate    RFC3339
//
// Parking intervals are the gaps between completed drives, clamped to the
// requested window and current time. Charging overlap is subtracted from
// parking so the returned segments can be rendered as a pie chart without
// double-counting.
//
// @Summary      Vehicle time distribution
// @Description  Aggregates driving, parked, and charging time for a car. Parked excludes charging overlap.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID       path   int     true   "TeslaMate cars.id"
// @Param        start_date  query  string  false  "RFC3339 lower bound"
// @Param        end_date    query  string  false  "RFC3339 upper bound"
// @Success      200  {object}  dto.V2TimeDistributionResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/stats/time-distribution [get]
func (h *Handler) StatsTimeDistribution(c *gin.Context) {
	const handler = "TeslaMateAPICarsStatsTimeDistributionV2"
	var ErrMsg = "Unable to load time distribution stats."
	var ErrDate = "Invalid date format."

	CarID, ok := respond.RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
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

	utcNow := utcNowTimestampSQL()
	query := fmt.Sprintf(`
		WITH bounds AS (
			SELECT
				NULLIF($2, '')::timestamp AS start_at,
				NULLIF($3, '')::timestamp AS requested_end_at,
				%[1]s AS now_at
		),
		time_window AS (
			SELECT
				start_at,
				COALESCE(requested_end_at, now_at) AS end_at,
				now_at
			FROM bounds
		),
		drive_slices AS (
			SELECT
				GREATEST(d.start_date, COALESCE(w.start_at, d.start_date)) AS slice_start,
				LEAST(d.end_date, w.end_at) AS slice_end
			FROM drives d
			CROSS JOIN time_window w
			WHERE d.car_id = $1
				AND d.end_date IS NOT NULL
				AND d.end_date > COALESCE(w.start_at, d.start_date)
				AND d.start_date < w.end_at
		),
		charge_slices AS (
			SELECT
				GREATEST(cp.start_date, COALESCE(w.start_at, cp.start_date)) AS slice_start,
				LEAST(cp.end_date, w.end_at) AS slice_end
			FROM charging_processes cp
			CROSS JOIN time_window w
			WHERE cp.car_id = $1
				AND cp.end_date IS NOT NULL
				AND cp.end_date > COALESCE(w.start_at, cp.start_date)
				AND cp.start_date < w.end_at
		),
		drive_pairs AS (
			SELECT
				d.end_date AS park_start,
				LEAD(d.start_date) OVER (PARTITION BY d.car_id ORDER BY d.start_date ASC) AS next_drive_start
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
		),
		parking_slices AS (
			SELECT
				GREATEST(dp.park_start, COALESCE(w.start_at, dp.park_start)) AS slice_start,
				LEAST(COALESCE(dp.next_drive_start, w.now_at), w.end_at) AS slice_end
			FROM drive_pairs dp
			CROSS JOIN time_window w
			WHERE COALESCE(dp.next_drive_start, w.now_at) > COALESCE(w.start_at, dp.park_start)
				AND dp.park_start < w.end_at
		),
		parking_seconds AS (
			SELECT
				COALESCE(SUM(GREATEST(
					EXTRACT(EPOCH FROM (p.slice_end - p.slice_start)) - COALESCE(overlap.seconds, 0),
					0
				)), 0) AS seconds
			FROM parking_slices p
			LEFT JOIN LATERAL (
				SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (
					LEAST(cp.end_date, p.slice_end) - GREATEST(cp.start_date, p.slice_start)
				))), 0) AS seconds
				FROM charging_processes cp
				WHERE cp.car_id = $1
					AND cp.end_date IS NOT NULL
					AND cp.end_date > p.slice_start
					AND cp.start_date < p.slice_end
					AND LEAST(cp.end_date, p.slice_end) > GREATEST(cp.start_date, p.slice_start)
			) overlap ON true
			WHERE p.slice_end > p.slice_start
		),
		totals AS (
			SELECT
				COALESCE((SELECT SUM(EXTRACT(EPOCH FROM (slice_end - slice_start))) FROM drive_slices WHERE slice_end > slice_start), 0) AS driving_seconds,
				(SELECT seconds FROM parking_seconds) AS parked_seconds,
				COALESCE((SELECT SUM(EXTRACT(EPOCH FROM (slice_end - slice_start))) FROM charge_slices WHERE slice_end > slice_start), 0) AS charging_seconds
		)
		SELECT
			(SELECT name FROM cars WHERE id = $1),
			COALESCE(driving_seconds, 0),
			COALESCE(parked_seconds, 0),
			COALESCE(charging_seconds, 0)
		FROM totals;`, utcNow)

	var (
		carName         NullString
		drivingSeconds  float64
		parkedSeconds   float64
		chargingSeconds float64
	)
	err = h.db.QueryRowContext(c.Request.Context(), query, CarID, parsedStartDate, parsedEndDate).Scan(
		&carName,
		&drivingSeconds,
		&parkedSeconds,
		&chargingSeconds,
	)
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	totalSeconds := math.Max(drivingSeconds, 0) + math.Max(parkedSeconds, 0) + math.Max(chargingSeconds, 0)
	segments := []dto.V2TimeDistributionSegment{
		timeDistributionSegment("driving", "Driving", drivingSeconds, totalSeconds),
		timeDistributionSegment("parked", "Parked", parkedSeconds, totalSeconds),
		timeDistributionSegment("charging", "Charging", chargingSeconds, totalSeconds),
	}

	start := NullString(parsedStartDate)
	end := NullString(parsedEndDate)
	if len(start) > 0 {
		start = NullString(h.timeInTZ(string(start)))
	}
	if len(end) > 0 {
		end = NullString(h.timeInTZ(string(end)))
	}

	respond.HandleSuccess(c, handler, dto.V2TimeDistributionResponse{
		Data: dto.V2TimeDistributionData{
			Car: dto.Car{
				CarID:   CarID,
				CarName: carName,
			},
			Range: dto.V2TimeDistributionRange{
				Start: start,
				End:   end,
			},
			TotalDurationMin: secondsToMinutes(totalSeconds),
			Segments:         segments,
		},
	})
}

func timeDistributionSegment(key, label string, seconds, totalSeconds float64) dto.V2TimeDistributionSegment {
	seconds = math.Max(seconds, 0)
	percent := 0.0
	if totalSeconds > 0 {
		percent = seconds / totalSeconds * 100
	}
	return dto.V2TimeDistributionSegment{
		Key:         key,
		Label:       label,
		DurationMin: secondsToMinutes(seconds),
		Percent:     percent,
	}
}

func secondsToMinutes(seconds float64) int {
	if seconds <= 0 {
		return 0
	}
	return int(math.Round(seconds / 60))
}
