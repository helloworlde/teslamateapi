package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @name V2IdlePeriodsAPIResponse
type V2IdlePeriodsAPIResponse struct {
	Data V2IdlePeriodsResponse `json:"data"`
	Meta V2Meta                `json:"meta"`
}

// @name V2IdlePeriodsResponse
type V2IdlePeriodsResponse struct {
	Items      []V2IdlePeriodItem `json:"items"`
	HasMore    bool               `json:"has_more"`
	NextCursor *string            `json:"next_cursor"`
}

// @name V2IdlePeriodItem
type V2IdlePeriodItem struct {
	ID                string         `json:"id"`
	StartDate         string         `json:"start_date"`
	EndDate           string         `json:"end_date"`
	Duration          float64        `json:"duration"`
	StartBatteryLevel *int           `json:"start_battery_level,omitempty"`
	EndBatteryLevel   *int           `json:"end_battery_level,omitempty"`
	SocDiff           *int           `json:"soc_diff,omitempty"`
	RangeLoss         *float64       `json:"range_loss,omitempty"`
	EnergyDrained     *float64       `json:"energy_drained,omitempty"`
	AvgPowerW         *float64       `json:"avg_power_w,omitempty"`
	RangeLossPerHour  *float64       `json:"range_loss_per_hour,omitempty"`
	StandbyRatio      *float64       `json:"standby_ratio,omitempty"`
	HasReducedRange   bool           `json:"has_reduced_range"`
	Geofence          *V2IdleGeofence `json:"geofence,omitempty"`
}

// @name V2IdleGeofence
type V2IdleGeofence struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// V2IdlePeriodsRepository fetches per-gap idle periods.
type V2IdlePeriodsRepository interface {
	IdlePeriods(ctx context.Context, carID int64, start, end timeBound, opts V2IdlePeriodsOptions) ([]V2IdlePeriodItem, error)
	CarExists(ctx context.Context, carID int64) (bool, error)
}

// V2IdlePeriodsOptions controls filtering / pagination.
type V2IdlePeriodsOptions struct {
	MinDurationHours float64
	GeofenceID       int64
	HasGeofenceID    bool
	Limit            int
	Offset           int
}

type PostgresV2IdlePeriodsRepository struct {
	db *sql.DB
}

func NewPostgresV2IdlePeriodsRepository(db *sql.DB) PostgresV2IdlePeriodsRepository {
	return PostgresV2IdlePeriodsRepository{db: db}
}

func (r PostgresV2IdlePeriodsRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2IdlePeriodsRepository) IdlePeriods(ctx context.Context, carID int64, start, end timeBound, opts V2IdlePeriodsOptions) ([]V2IdlePeriodItem, error) {
	// Adapted from Grafana vampire-drain.json: union drive+charge events, find gaps > min_duration,
	// per-gap pull battery_level / rated_range / standby seconds (asleep+offline) from states.
	// car.efficiency converts rated-range loss → kWh.
	rows, err := r.db.QueryContext(ctx, `
		WITH events AS (
			SELECT start_date, end_date, end_position_id AS pos, start_geofence_id AS gf_id
			FROM drives
			WHERE car_id = $1 AND end_date IS NOT NULL
			  AND end_date >= $2::timestamptz AND start_date < $3::timestamptz
			UNION ALL
			SELECT start_date, end_date, position_id AS pos, geofence_id AS gf_id
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL
			  AND end_date >= $2::timestamptz AND start_date < $3::timestamptz
		),
		o AS (
			SELECT
				start_date,
				end_date,
				pos,
				gf_id,
				LAG(end_date) OVER (ORDER BY start_date) AS prev_end,
				LAG(pos)      OVER (ORDER BY start_date) AS prev_pos
			FROM events
		),
		gaps AS (
			SELECT prev_end AS gap_start, start_date AS gap_end, prev_pos, pos, gf_id
			FROM o
			WHERE prev_end IS NOT NULL
			  AND start_date - prev_end > make_interval(hours => $4::int)
		),
		car_eff AS (
			SELECT COALESCE(efficiency, 0.15) AS efficiency FROM cars WHERE id = $1
		)
		SELECT
			g.gap_start,
			g.gap_end,
			EXTRACT(EPOCH FROM (g.gap_end - g.gap_start)) AS duration_seconds,
			p1.battery_level AS start_battery_level,
			p2.battery_level AS end_battery_level,
			p1.usable_battery_level AS start_usable,
			p2.usable_battery_level AS end_usable,
			p1.rated_battery_range_km AS start_rated_range,
			p2.rated_battery_range_km AS end_rated_range,
			(SELECT efficiency FROM car_eff) AS efficiency,
			(
				SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (LEAST(s.end_date, g.gap_end) - GREATEST(s.start_date, g.gap_start)))), 0)
				FROM states s
				WHERE s.car_id = $1
				  AND s.state IN ('asleep','offline')
				  AND s.start_date < g.gap_end
				  AND COALESCE(s.end_date, g.gap_end) > g.gap_start
			) AS standby_seconds,
			gf.id AS geofence_id,
			gf.name AS geofence_name
		FROM gaps g
		JOIN positions p1 ON p1.id = g.prev_pos
		JOIN positions p2 ON p2.id = g.pos
		LEFT JOIN geofences gf ON gf.id = g.gf_id
		WHERE
			(NOT $5::bool OR gf.id = $6::bigint)
		ORDER BY g.gap_start DESC
		LIMIT $7 OFFSET $8
	`,
		carID,
		start.Time,
		end.Time,
		int(opts.MinDurationHours),
		opts.HasGeofenceID,
		opts.GeofenceID,
		opts.Limit,
		opts.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []V2IdlePeriodItem{}
	idx := 0
	for rows.Next() {
		var (
			gapStart, gapEnd                                                  string
			durationSec                                                       float64
			startBL, endBL, startUsable, endUsable                            sql.NullInt64
			startRated, endRated                                              sql.NullFloat64
			efficiency                                                        sql.NullFloat64
			standbySec                                                        sql.NullFloat64
			geofenceID                                                        sql.NullInt64
			geofenceName                                                      sql.NullString
		)
		if err := rows.Scan(
			&gapStart, &gapEnd, &durationSec,
			&startBL, &endBL,
			&startUsable, &endUsable,
			&startRated, &endRated,
			&efficiency,
			&standbySec,
			&geofenceID, &geofenceName,
		); err != nil {
			return nil, err
		}
		idx++
		item := V2IdlePeriodItem{
			ID:        gapStart + "_" + gapEnd,
			StartDate: gapStart,
			EndDate:   gapEnd,
			Duration:  durationSec,
		}
		if startBL.Valid {
			v := int(startBL.Int64)
			item.StartBatteryLevel = &v
		}
		if endBL.Valid {
			v := int(endBL.Int64)
			item.EndBatteryLevel = &v
		}
		if startBL.Valid && endBL.Valid {
			diff := int(endBL.Int64 - startBL.Int64)
			item.SocDiff = &diff
		}
		if startRated.Valid && endRated.Valid {
			loss := startRated.Float64 - endRated.Float64
			item.RangeLoss = &loss
			if efficiency.Valid {
				kwh := loss * efficiency.Float64
				item.EnergyDrained = &kwh
				if durationSec > 0 {
					power := kwh * 1000.0 / (durationSec / 3600.0)
					item.AvgPowerW = &power
				}
			}
			if durationSec > 0 {
				perHour := loss / (durationSec / 3600.0)
				item.RangeLossPerHour = &perHour
			}
		}
		if standbySec.Valid && durationSec > 0 {
			ratio := standbySec.Float64 / durationSec
			item.StandbyRatio = &ratio
		}
		if startUsable.Valid && startBL.Valid && startBL.Int64 != startUsable.Int64 {
			item.HasReducedRange = true
		}
		if endUsable.Valid && endBL.Valid && endBL.Int64 != endUsable.Int64 {
			item.HasReducedRange = true
		}
		if geofenceID.Valid && geofenceName.Valid {
			item.Geofence = &V2IdleGeofence{ID: geofenceID.Int64, Name: geofenceName.String}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// V2IdlePeriodsService produces idle period responses.
type V2IdlePeriodsService struct {
	repository V2IdlePeriodsRepository
}

func NewV2IdlePeriodsService(repository V2IdlePeriodsRepository) V2IdlePeriodsService {
	return V2IdlePeriodsService{repository: repository}
}

type V2IdlePeriodsBuildOptions struct {
	V2IdlePeriodsOptions
	Cursor string
}

func (s V2IdlePeriodsService) BuildIdlePeriods(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2IdlePeriodsBuildOptions) (V2IdlePeriodsResponse, int64, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return V2IdlePeriodsResponse{}, 0, errors.New("invalid car id")
	}
	if s.repository != nil {
		exists, err := s.repository.CarExists(ctx, carID)
		if err != nil {
			return V2IdlePeriodsResponse{}, carID, err
		}
		if !exists {
			return V2IdlePeriodsResponse{}, carID, errV2CarNotFound
		}
	}
	if opts.MinDurationHours <= 0 {
		opts.MinDurationHours = 1
	}
	if opts.Limit <= 0 || opts.Limit > 500 {
		opts.Limit = 100
	}
	if opts.Cursor != "" {
		if decoded, err := base64.RawURLEncoding.DecodeString(opts.Cursor); err == nil {
			if n, err := strconv.Atoi(string(decoded)); err == nil && n >= 0 {
				opts.Offset = n
			}
		}
	}
	items, err := s.repository.IdlePeriods(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End), opts.V2IdlePeriodsOptions)
	if err != nil {
		return V2IdlePeriodsResponse{}, carID, err
	}
	resp := V2IdlePeriodsResponse{Items: items}
	if len(items) == opts.Limit {
		resp.HasMore = true
		next := base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(opts.Offset + opts.Limit)))
		resp.NextCursor = &next
	}
	return resp, carID, nil
}

// IdlePeriods godoc
//
// @Summary V2 parking idle periods
// @Description Returns per-gap idle periods (vampire drain candidates) with SoC diff, range loss, drained energy and standby ratio for one car.
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param min_duration_hours query number false "Minimum gap length in hours (default 1)"
// @Param geofence_id query int false "Filter by resolved geofence id"
// @Param limit query int false "Page size (default 100, max 500)"
// @Param cursor query string false "Pagination cursor returned by a previous response"
// @Success 200 {object} V2IdlePeriodsAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/parking/idle_periods [get]
func (h V2Handlers) IdlePeriods(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.idlePeriodsBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 idle periods service is not configured.", nil)
		return
	}
	opts := V2IdlePeriodsBuildOptions{}
	if v := c.Query("min_duration_hours"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 {
			opts.MinDurationHours = parsed
		}
	}
	if v := c.Query("geofence_id"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			opts.GeofenceID = parsed
			opts.HasGeofenceID = true
		}
	}
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			opts.Limit = parsed
		}
	}
	opts.Cursor = c.Query("cursor")
	response, carID, err := h.idlePeriodsBuilder.BuildIdlePeriods(c.Request.Context(), c.Param("CarID"), timeRange, opts)
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case err.Error() == "invalid car id":
			v2BadRequest(c, "Invalid car id.", nil)
		default:
			v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 idle periods.", err.Error())
		}
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}
