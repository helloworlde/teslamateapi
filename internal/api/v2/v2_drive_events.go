package v2

import (
	"context"
	"database/sql"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
)

// @name V2DriveEventsAPIResponse
type V2DriveEventsAPIResponse struct {
	Data V2DriveEventsResponse `json:"data"`
}

// @name V2DriveEventsResponse
type V2DriveEventsResponse struct {
	DriveID    int64                  `json:"drive_id"`
	Score      int                    `json:"score"`
	Thresholds V2DriveEventThresholds `json:"thresholds"`
	Summary    V2DriveEventsSummary   `json:"summary"`
	Events     []V2DriveEvent         `json:"events"`
}

// @name V2DriveEventThresholds
type V2DriveEventThresholds struct {
	HardBrakeKmhS float64 `json:"hard_brake_kmh_s"`
	HardAccelKmhS float64 `json:"hard_accel_kmh_s"`
	OverspeedKmh  int     `json:"overspeed_kmh"`
	HighPowerKW   int     `json:"high_power_kw"`
	HighRegenKW   int     `json:"high_regen_kw"`
}

// @name V2DriveEventsSummary
type V2DriveEventsSummary struct {
	HardBrakeCount int `json:"hard_brake_count"`
	HardAccelCount int `json:"hard_accel_count"`
	OverspeedCount int `json:"overspeed_count"`
	HighPowerCount int `json:"high_power_count"`
	HighRegenCount int `json:"high_regen_count"`
	TotalEvents    int `json:"total_events"`
}

// @name V2DriveEvent
type V2DriveEvent struct {
	Type      string   `json:"type"`
	Severity  string   `json:"severity"`
	Date      string   `json:"date"`
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Speed     int      `json:"speed"`
	SpeedPrev *int     `json:"speed_before,omitempty"`
	Power     int      `json:"power"`
	Rate      *float64 `json:"rate,omitempty"`
}

type v2DriveEventsConfig struct {
	hardBrakeKmhS float64
	hardAccelKmhS float64
	overspeedKmh  int
	highPowerKW   int
	highRegenKW   int
}

func defaultDriveEventsConfig() v2DriveEventsConfig {
	return v2DriveEventsConfig{
		hardBrakeKmhS: 8,
		hardAccelKmhS: 8,
		overspeedKmh:  120,
		highPowerKW:   80,
		highRegenKW:   50,
	}
}

type rawEvent struct {
	date      time.Time
	lat, lng  float64
	speed     int
	speedPrev int
	power     int
	rate      float64
	eventType string
}

func queryDriveEvents(ctx context.Context, db *sql.DB, carID, driveID int64, cfg v2DriveEventsConfig) ([]V2DriveEvent, error) {
	rows, err := db.QueryContext(ctx, `
		WITH pts AS (
			SELECT
				date, latitude, longitude, speed, power,
				LAG(speed, 5) OVER w AS speed_5ago,
				LAG(date, 5)  OVER w AS date_5ago
			FROM positions
			WHERE drive_id = $2 AND car_id = $1
			WINDOW w AS (ORDER BY date)
		),
		with_rate AS (
			SELECT
				date, latitude, longitude, speed, power, speed_5ago,
				EXTRACT(EPOCH FROM (date - date_5ago)) AS window_dt,
				CASE WHEN EXTRACT(EPOCH FROM (date - date_5ago)) > 0
				     THEN (speed - speed_5ago)::float8 / EXTRACT(EPOCH FROM (date - date_5ago))
				END AS rate_kmh_s
			FROM pts
		)
		SELECT date, latitude, longitude, speed, COALESCE(speed_5ago, speed), power,
		       COALESCE(rate_kmh_s, 0), window_dt,
			   CASE
				 WHEN rate_kmh_s < $3 AND speed_5ago > 20 THEN 'hard_brake'
				 WHEN rate_kmh_s > $4 AND speed > 20      THEN 'hard_accel'
				 ELSE NULL
			   END AS rate_event,
			   CASE WHEN speed >= $5 THEN true ELSE false END AS is_overspeed,
			   CASE WHEN power >= $6 THEN true ELSE false END AS is_high_power,
			   CASE WHEN power <= $7 THEN true ELSE false END AS is_high_regen
		FROM with_rate
		WHERE (
			(rate_kmh_s < $3 AND speed_5ago > 20)
			OR (rate_kmh_s > $4 AND speed > 20)
			OR speed >= $5
			OR power >= $6
			OR power <= $7
		)
		AND (window_dt IS NULL OR window_dt BETWEEN 0.5 AND 10)
		ORDER BY date ASC`,
		carID, driveID,
		-cfg.hardBrakeKmhS, cfg.hardAccelKmhS,
		cfg.overspeedKmh, cfg.highPowerKW, -cfg.highRegenKW,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var raw []rawEvent
	for rows.Next() {
		var (
			date                     time.Time
			lat, lng, rate, windowDt float64
			speed, speedPrev, power  int
			rateEvent                sql.NullString
			isOverspeed, isHP, isHR  bool
		)
		if err := rows.Scan(&date, &lat, &lng, &speed, &speedPrev, &power,
			&rate, &windowDt, &rateEvent, &isOverspeed, &isHP, &isHR); err != nil {
			return nil, err
		}
		if rateEvent.Valid {
			raw = append(raw, rawEvent{date: date, lat: lat, lng: lng, speed: speed, speedPrev: speedPrev, power: power, rate: rate, eventType: rateEvent.String})
		}
		if isOverspeed {
			raw = append(raw, rawEvent{date: date, lat: lat, lng: lng, speed: speed, power: power, eventType: "overspeed"})
		}
		if isHP {
			raw = append(raw, rawEvent{date: date, lat: lat, lng: lng, speed: speed, power: power, eventType: "high_power"})
		}
		if isHR {
			raw = append(raw, rawEvent{date: date, lat: lat, lng: lng, speed: speed, power: power, eventType: "high_regen"})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mergeEvents(raw, cfg), nil
}

func mergeWindowFor(eventType string) time.Duration {
	switch eventType {
	case "overspeed":
		return 30 * time.Second
	case "high_power", "high_regen":
		return 10 * time.Second
	default:
		return 5 * time.Second
	}
}

func mergeEvents(raw []rawEvent, cfg v2DriveEventsConfig) []V2DriveEvent {
	var merged []V2DriveEvent
	used := make([]bool, len(raw))

	for i := range raw {
		if used[i] {
			continue
		}
		best := raw[i]
		lastDate := raw[i].date
		used[i] = true
		window := mergeWindowFor(best.eventType)
		for j := i + 1; j < len(raw); j++ {
			if used[j] || raw[j].eventType != best.eventType {
				continue
			}
			if raw[j].date.Sub(lastDate) > window {
				break
			}
			used[j] = true
			lastDate = raw[j].date
			if isMoreExtreme(raw[j], best) {
				best = raw[j]
			}
		}
		merged = append(merged, toV2Event(best, cfg))
	}
	return merged
}

func isMoreExtreme(a, b rawEvent) bool {
	switch a.eventType {
	case "hard_brake":
		return a.rate < b.rate
	case "hard_accel":
		return a.rate > b.rate
	case "overspeed":
		return a.speed > b.speed
	case "high_power":
		return a.power > b.power
	case "high_regen":
		return a.power < b.power
	}
	return false
}

func toV2Event(r rawEvent, cfg v2DriveEventsConfig) V2DriveEvent {
	e := V2DriveEvent{
		Type:      r.eventType,
		Date:      r.date.Format(time.RFC3339),
		Latitude:  r.lat,
		Longitude: r.lng,
		Speed:     r.speed,
		Power:     r.power,
	}
	switch r.eventType {
	case "hard_brake":
		e.SpeedPrev = &r.speedPrev
		rate := r.rate
		e.Rate = &rate
		if math.Abs(r.rate) > cfg.hardBrakeKmhS*1.5 {
			e.Severity = "severe"
		} else {
			e.Severity = "moderate"
		}
	case "hard_accel":
		e.SpeedPrev = &r.speedPrev
		rate := r.rate
		e.Rate = &rate
		if r.rate > cfg.hardAccelKmhS*1.5 {
			e.Severity = "severe"
		} else {
			e.Severity = "moderate"
		}
	case "overspeed":
		if r.speed >= cfg.overspeedKmh+20 {
			e.Severity = "severe"
		} else {
			e.Severity = "moderate"
		}
	case "high_power":
		if r.power >= cfg.highPowerKW+30 {
			e.Severity = "severe"
		} else {
			e.Severity = "moderate"
		}
	case "high_regen":
		if r.power <= -(cfg.highRegenKW + 20) {
			e.Severity = "severe"
		} else {
			e.Severity = "moderate"
		}
	}
	return e
}

func computeScore(events []V2DriveEvent) int {
	score := 100
	for _, e := range events {
		switch e.Type {
		case "hard_brake":
			if e.Severity == "severe" {
				score -= 8
			} else {
				score -= 5
			}
		case "hard_accel":
			if e.Severity == "severe" {
				score -= 5
			} else {
				score -= 3
			}
		case "overspeed":
			if e.Severity == "severe" {
				score -= 10
			} else {
				score -= 5
			}
		case "high_power":
			score -= 2
		case "high_regen":
			score -= 1
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

func countByType(events []V2DriveEvent) V2DriveEventsSummary {
	var s V2DriveEventsSummary
	for _, e := range events {
		switch e.Type {
		case "hard_brake":
			s.HardBrakeCount++
		case "hard_accel":
			s.HardAccelCount++
		case "overspeed":
			s.OverspeedCount++
		case "high_power":
			s.HighPowerCount++
		case "high_regen":
			s.HighRegenCount++
		}
	}
	s.TotalEvents = len(events)
	return s
}

func parseDriveEventsConfig(c *gin.Context) v2DriveEventsConfig {
	cfg := defaultDriveEventsConfig()
	if v := c.Query("hard_brake_threshold"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.hardBrakeKmhS = f
		}
	}
	if v := c.Query("hard_accel_threshold"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.hardAccelKmhS = f
		}
	}
	if v := c.Query("overspeed_threshold"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			cfg.overspeedKmh = i
		}
	}
	if v := c.Query("high_power_threshold"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			cfg.highPowerKW = i
		}
	}
	if v := c.Query("high_regen_threshold"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			cfg.highRegenKW = i
		}
	}
	return cfg
}

// DriveEvents godoc
//
// @Summary Detect abnormal driving events in a single drive
// @Description Analyzes position data within a drive to detect hard braking, hard acceleration, overspeeding, high power, and high regen events. Returns event list and a safety score (100 = perfect).
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param DriveID path int true "Drive ID" example(798)
// @Param hard_brake_threshold query number false "Hard brake threshold in km/h/s (default 8)"
// @Param hard_accel_threshold query number false "Hard accel threshold in km/h/s (default 8)"
// @Param overspeed_threshold query int false "Overspeed threshold in km/h (default 120)"
// @Param high_power_threshold query int false "High power threshold in kW (default 80)"
// @Param high_regen_threshold query int false "High regen threshold in kW (default 50)"
// @Success 200 {object} V2DriveEventsAPIResponse
// @Failure 400 {object} apicommon.APIErrorResponse
// @Failure 404 {object} apicommon.APIErrorResponse
// @Failure 500 {object} apicommon.APIErrorResponse
// @Router /v2/cars/{CarID}/drives/{DriveID}/events [get]
func (h V2Handlers) DriveEvents(c *gin.Context) {
	carID, err := strconv.ParseInt(c.Param("CarID"), 10, 64)
	if err != nil || carID <= 0 {
		apicommon.V2BadRequest(c, "Invalid car id.", nil)
		return
	}
	driveID, err := strconv.ParseInt(c.Param("DriveID"), 10, 64)
	if err != nil || driveID <= 0 {
		apicommon.V2BadRequest(c, "Invalid drive id.", nil)
		return
	}

	if apicommon.DB == nil {
		apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Database not available.", nil)
		return
	}

	var exists bool
	if err := apicommon.DB.QueryRowContext(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM drives WHERE id = $1 AND car_id = $2)`,
		driveID, carID).Scan(&exists); err != nil || !exists {
		apicommon.V2Error(c, http.StatusNotFound, "DRIVE_NOT_FOUND", "Drive not found.", nil)
		return
	}

	cfg := parseDriveEventsConfig(c)
	events, err := queryDriveEvents(c.Request.Context(), apicommon.DB, carID, driveID, cfg)
	if err != nil {
		apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to analyze drive events.", err.Error())
		return
	}

	response := V2DriveEventsResponse{
		DriveID: driveID,
		Score:   computeScore(events),
		Thresholds: V2DriveEventThresholds{
			HardBrakeKmhS: cfg.hardBrakeKmhS,
			HardAccelKmhS: cfg.hardAccelKmhS,
			OverspeedKmh:  cfg.overspeedKmh,
			HighPowerKW:   cfg.highPowerKW,
			HighRegenKW:   cfg.highRegenKW,
		},
		Summary: countByType(events),
		Events:  events,
	}
	if response.Events == nil {
		response.Events = []V2DriveEvent{}
	}
	c.JSON(http.StatusOK, V2DriveEventsAPIResponse{Data: response})
}
