package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ----------------- /v2/cars/{CarID}/lifecycle/odometer_series -----------------

// @name V2OdometerSeriesAPIResponse
type V2OdometerSeriesAPIResponse struct {
	Data V2OdometerSeriesResponse `json:"data"`
	Meta V2Meta                   `json:"meta"`
}

// @name V2OdometerSeriesResponse
type V2OdometerSeriesResponse struct {
	Items      []V2OdometerSeriesItem `json:"items"`
	HasMore    bool                   `json:"has_more"`
	NextCursor *string                `json:"next_cursor,omitempty"`
}

// @name V2OdometerSeriesItem
type V2OdometerSeriesItem struct {
	Date     string  `json:"date"`
	Odometer float64 `json:"odometer"`
}

// V2OdometerSeriesBuilder builds cumulative odometer series.
type V2OdometerSeriesBuilder interface {
	BuildOdometerSeries(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2CursorPagination) (V2OdometerSeriesResponse, int64, error)
}

// V2CursorPagination is a simple base64(int) cursor scheme reused across phase Q endpoints.
type V2CursorPagination struct {
	Limit  int
	Cursor string
	Offset int
}

func decodeOffsetCursor(c string) int {
	if c == "" {
		return 0
	}
	if dec, err := base64.RawURLEncoding.DecodeString(c); err == nil {
		if n, err := strconv.Atoi(string(dec)); err == nil && n >= 0 {
			return n
		}
	}
	return 0
}

func encodeOffsetCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

type V2OdometerSeriesService struct {
	db *sql.DB
}

func NewV2OdometerSeriesService(db *sql.DB) V2OdometerSeriesService {
	return V2OdometerSeriesService{db: db}
}

func (s V2OdometerSeriesService) BuildOdometerSeries(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2CursorPagination) (V2OdometerSeriesResponse, int64, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return V2OdometerSeriesResponse{}, 0, errors.New("invalid car id")
	}
	if s.db == nil {
		return V2OdometerSeriesResponse{}, carID, errors.New("db not configured")
	}
	limit := opts.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	offset := decodeOffsetCursor(opts.Cursor)

	rows, err := s.db.QueryContext(ctx, `
		SELECT to_char(date_trunc('day', end_date), 'YYYY-MM-DD') AS day,
		       MAX(end_km)::float8 AS odometer
		FROM drives
		WHERE car_id = $1
		  AND end_date IS NOT NULL
		  AND end_date >= $2::timestamptz
		  AND end_date <  $3::timestamptz
		GROUP BY 1
		ORDER BY 1 ASC
		LIMIT $4 OFFSET $5
	`, carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, limit, offset)
	if err != nil {
		return V2OdometerSeriesResponse{}, carID, err
	}
	defer rows.Close()

	items := []V2OdometerSeriesItem{}
	for rows.Next() {
		var (
			day string
			odo sql.NullFloat64
		)
		if err := rows.Scan(&day, &odo); err != nil {
			return V2OdometerSeriesResponse{}, carID, err
		}
		item := V2OdometerSeriesItem{Date: day}
		if odo.Valid {
			item.Odometer = odo.Float64
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return V2OdometerSeriesResponse{}, carID, err
	}
	resp := V2OdometerSeriesResponse{Items: items}
	if len(items) == limit {
		resp.HasMore = true
		next := encodeOffsetCursor(offset + limit)
		resp.NextCursor = &next
	}
	return resp, carID, nil
}

// OdometerSeries godoc
//
// @Summary V2 cumulative odometer series
// @Description Returns one odometer reading per day in the requested window for one car.
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param limit query int false "Page size (default 500, max 1000)"
// @Param cursor query string false "Pagination cursor returned by a previous response"
// @Success 200 {object} V2OdometerSeriesAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/lifecycle/odometer_series [get]
func (h V2Handlers) OdometerSeries(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.odometerSeriesBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 odometer series service is not configured.", nil)
		return
	}
	opts := V2CursorPagination{}
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			opts.Limit = parsed
		}
	}
	opts.Cursor = c.Query("cursor")
	response, carID, err := h.odometerSeriesBuilder.BuildOdometerSeries(c.Request.Context(), c.Param("CarID"), timeRange, opts)
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case err.Error() == "invalid car id":
			v2BadRequest(c, "Invalid car id.", nil)
		default:
			v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 odometer series.", err.Error())
		}
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ----------------- /v2/cars/{CarID}/summary/by_period -----------------

// @name V2SummaryByPeriodAPIResponse
type V2SummaryByPeriodAPIResponse struct {
	Data V2SummaryByPeriodResponse `json:"data"`
	Meta V2Meta                    `json:"meta"`
}

// @name V2SummaryByPeriodResponse
type V2SummaryByPeriodResponse struct {
	GroupBy string                  `json:"group_by"`
	Items   []V2SummaryByPeriodItem `json:"items"`
}

// @name V2SummaryByPeriodItem
type V2SummaryByPeriodItem struct {
	PeriodStart      string   `json:"period_start"`
	DriveCount       int64    `json:"drive_count"`
	Distance         float64  `json:"distance"`
	DriveDuration    float64  `json:"drive_duration"`
	NetEnergy        *float64 `json:"net_energy,omitempty"`
	AvgConsumption   *float64 `json:"avg_consumption,omitempty"`
	ChargingSessions int64    `json:"charging_sessions"`
	EnergyAdded      float64  `json:"energy_added"`
	EnergyUsed       float64  `json:"energy_used"`
	ChargingCost     *float64 `json:"charging_cost,omitempty"`
	DataComplete     bool     `json:"data_complete"`
}

type V2SummaryByPeriodService struct {
	db *sql.DB
}

func NewV2SummaryByPeriodService(db *sql.DB) V2SummaryByPeriodService {
	return V2SummaryByPeriodService{db: db}
}

type V2SummaryByPeriodBuilder interface {
	BuildByPeriod(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2SummaryByPeriodResponse, int64, error)
}

func (s V2SummaryByPeriodService) BuildByPeriod(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2SummaryByPeriodResponse, int64, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return V2SummaryByPeriodResponse{}, 0, errors.New("invalid car id")
	}
	if s.db == nil {
		return V2SummaryByPeriodResponse{}, carID, errors.New("db not configured")
	}
	switch groupBy {
	case "", "day", "week", "month", "year":
	default:
		return V2SummaryByPeriodResponse{}, carID, errV2InvalidDrivingGroupBy
	}
	if groupBy == "" {
		groupBy = "day"
	}
	truncUnit := postgresDateTruncUnit(groupBy)

	rows, err := s.db.QueryContext(ctx, `
		WITH drv AS (
			SELECT
				date_trunc('`+truncUnit+`', drives.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
				COUNT(*) AS drive_count,
				COALESCE(SUM(drives.distance), 0) AS distance,
				COALESCE(SUM(drives.duration_min) * 60, 0) AS drive_duration,
				SUM(
					CASE
						WHEN drives.start_rated_range_km IS NOT NULL
							AND drives.end_rated_range_km IS NOT NULL
							AND cars.efficiency IS NOT NULL
							AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0
						THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency
					END
				) AS net_energy,
				BOOL_AND(drives.end_date IS NOT NULL) AS drive_complete
			FROM drives
			LEFT JOIN cars ON cars.id = drives.car_id
			WHERE drives.car_id = $1
			  AND drives.start_date >= $2 AND drives.start_date < $3
			GROUP BY 1
		),
		chg AS (
			SELECT
				date_trunc('`+truncUnit+`', cp.start_date AT TIME ZONE 'UTC' AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
				COUNT(*) AS charging_sessions,
				COALESCE(SUM(cp.charge_energy_added), 0) AS energy_added,
				COALESCE(SUM(GREATEST(COALESCE(cp.charge_energy_used, 0), COALESCE(cp.charge_energy_added, 0))), 0) AS energy_used,
				SUM(cp.cost) AS charging_cost,
				BOOL_AND(cp.end_date IS NOT NULL) AS charge_complete
			FROM charging_processes cp
			WHERE cp.car_id = $1 AND cp.start_date >= $2 AND cp.start_date < $3
			GROUP BY 1
		)
		SELECT
			COALESCE(drv.period_start, chg.period_start) AS period_start,
			COALESCE(drv.drive_count, 0)        AS drive_count,
			COALESCE(drv.distance, 0)           AS distance,
			COALESCE(drv.drive_duration, 0)     AS drive_duration,
			drv.net_energy,
			COALESCE(chg.charging_sessions, 0)  AS charging_sessions,
			COALESCE(chg.energy_added, 0)       AS energy_added,
			COALESCE(chg.energy_used, 0)        AS energy_used,
			chg.charging_cost,
			COALESCE(drv.drive_complete, true) AND COALESCE(chg.charge_complete, true) AS data_complete
		FROM drv
		FULL OUTER JOIN chg USING (period_start)
		ORDER BY period_start ASC
	`, carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, timeRange.Timezone)
	if err != nil {
		return V2SummaryByPeriodResponse{}, carID, err
	}
	defer rows.Close()

	items := []V2SummaryByPeriodItem{}
	for rows.Next() {
		var (
			periodStart                          string
			driveCount, sessionCount             sql.NullInt64
			distance, driveDuration, netEnergy   sql.NullFloat64
			energyAdded, energyUsed, chargingCost sql.NullFloat64
			complete                             sql.NullBool
		)
		if err := rows.Scan(
			&periodStart,
			&driveCount,
			&distance,
			&driveDuration,
			&netEnergy,
			&sessionCount,
			&energyAdded,
			&energyUsed,
			&chargingCost,
			&complete,
		); err != nil {
			return V2SummaryByPeriodResponse{}, carID, err
		}
		item := V2SummaryByPeriodItem{
			PeriodStart:      periodStart,
			DriveCount:       driveCount.Int64,
			Distance:         distance.Float64,
			DriveDuration:    driveDuration.Float64,
			ChargingSessions: sessionCount.Int64,
			EnergyAdded:      energyAdded.Float64,
			EnergyUsed:       energyUsed.Float64,
			DataComplete:     complete.Valid && complete.Bool,
		}
		if netEnergy.Valid {
			v := netEnergy.Float64
			item.NetEnergy = &v
			if distance.Valid && distance.Float64 > 0 {
				cons := netEnergy.Float64 / distance.Float64 * 1000
				item.AvgConsumption = &cons
			}
		}
		if chargingCost.Valid {
			v := chargingCost.Float64
			item.ChargingCost = &v
		}
		items = append(items, item)
	}
	return V2SummaryByPeriodResponse{GroupBy: groupBy, Items: items}, carID, rows.Err()
}

// SummaryByPeriod godoc
//
// @Summary V2 flat per-period summary
// @Description Returns drive + charging + cost metrics flattened per period (one row per period).
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param group_by query string false "Period grouping" Enums(day, week, month, year)
// @Success 200 {object} V2SummaryByPeriodAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/summary/by_period [get]
func (h V2Handlers) SummaryByPeriod(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.summaryByPeriodBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 summary-by-period service is not configured.", nil)
		return
	}
	response, carID, err := h.summaryByPeriodBuilder.BuildByPeriod(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case errors.Is(err, errV2InvalidDrivingGroupBy):
			v2BadRequest(c, "Invalid summary group_by.", nil)
		case err.Error() == "invalid car id":
			v2BadRequest(c, "Invalid car id.", nil)
		default:
			v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 by_period summary.", err.Error())
		}
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ----------------- /v2/cars/{CarID}/lifecycle/places -----------------

// @name V2PlacesAPIResponse
type V2PlacesAPIResponse struct {
	Data V2PlacesResponse `json:"data"`
	Meta V2Meta           `json:"meta"`
}

// @name V2PlacesResponse
type V2PlacesResponse struct {
	Cities    []V2PlaceBucket `json:"cities"`
	States    []V2PlaceBucket `json:"states"`
	Countries []V2PlaceBucket `json:"countries"`
}

// @name V2PlaceBucket
type V2PlaceBucket struct {
	Name        string  `json:"name"`
	VisitCount  int64   `json:"visit_count"`
	LastVisited *string `json:"last_visited,omitempty"`
}

type V2PlacesService struct {
	db *sql.DB
}

func NewV2PlacesService(db *sql.DB) V2PlacesService {
	return V2PlacesService{db: db}
}

type V2PlacesBuilder interface {
	BuildPlaces(ctx context.Context, carIDParam string, topN int) (V2PlacesResponse, int64, error)
}

func (s V2PlacesService) BuildPlaces(ctx context.Context, carIDParam string, topN int) (V2PlacesResponse, int64, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return V2PlacesResponse{}, 0, errors.New("invalid car id")
	}
	if s.db == nil {
		return V2PlacesResponse{}, carID, errors.New("db not configured")
	}
	if topN <= 0 || topN > 100 {
		topN = 20
	}
	queryFor := func(col string) ([]V2PlaceBucket, error) {
		rows, err := s.db.QueryContext(ctx, `
			SELECT a.`+col+` AS name,
			       COUNT(*) AS visit_count,
			       MAX(d.end_date)::text AS last_visited
			FROM drives d
			JOIN addresses a ON a.id = d.end_address_id
			WHERE d.car_id = $1
			  AND d.end_date IS NOT NULL
			  AND a.`+col+` IS NOT NULL AND a.`+col+` <> ''
			GROUP BY 1
			ORDER BY visit_count DESC
			LIMIT $2
		`, carID, topN)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []V2PlaceBucket{}
		for rows.Next() {
			var (
				name        string
				count       int64
				lastVisited sql.NullString
			)
			if err := rows.Scan(&name, &count, &lastVisited); err != nil {
				return nil, err
			}
			b := V2PlaceBucket{Name: name, VisitCount: count}
			if lastVisited.Valid && lastVisited.String != "" {
				s := lastVisited.String
				b.LastVisited = &s
			}
			out = append(out, b)
		}
		return out, rows.Err()
	}
	cities, err := queryFor("city")
	if err != nil {
		return V2PlacesResponse{}, carID, err
	}
	states, err := queryFor("state")
	if err != nil {
		return V2PlacesResponse{}, carID, err
	}
	countries, err := queryFor("country")
	if err != nil {
		return V2PlacesResponse{}, carID, err
	}
	return V2PlacesResponse{Cities: cities, States: states, Countries: countries}, carID, nil
}

// Places godoc
//
// @Summary V2 lifecycle places (city / state / country breakdown)
// @Description Returns the top-N most visited cities, states and countries for one car, with last-visited timestamps.
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param top_n query int false "Top N per dimension (default 20, max 100)"
// @Success 200 {object} V2PlacesAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/lifecycle/places [get]
func (h V2Handlers) Places(c *gin.Context) {
	if h.placesBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 places service is not configured.", nil)
		return
	}
	topN := 20
	if v := c.Query("top_n"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			topN = parsed
		}
	}
	response, carID, err := h.placesBuilder.BuildPlaces(c.Request.Context(), c.Param("CarID"), topN)
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case err.Error() == "invalid car id":
			v2BadRequest(c, "Invalid car id.", nil)
		default:
			v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 lifecycle places.", err.Error())
		}
		return
	}
	meta := V2Meta{
		CarID:       carID,
		Period:      "lifetime",
		Timezone:    "UTC",
		Unit:        defaultV2Unit(),
		GeneratedAt: h.now().UTC().Format(time.RFC3339),
	}
	v2JSON(c, http.StatusOK, response, meta)
}

// ----------------- /v2/geofences (global) -----------------

// @name V2GeofencesAPIResponse
type V2GeofencesAPIResponse struct {
	Data V2GeofencesResponse `json:"data"`
	Meta V2Meta              `json:"meta"`
}

// @name V2GeofencesResponse
type V2GeofencesResponse struct {
	Items []V2Geofence `json:"items"`
}

// @name V2Geofence
type V2Geofence struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	Radius      *float64 `json:"radius,omitempty"`
	CostPerUnit *float64 `json:"cost_per_unit,omitempty"`
	BillingType *string  `json:"billing_type,omitempty"`
	SessionFee  *float64 `json:"session_fee,omitempty"`
}

type V2GeofencesService struct {
	db *sql.DB
}

func NewV2GeofencesService(db *sql.DB) V2GeofencesService {
	return V2GeofencesService{db: db}
}

type V2GeofencesBuilder interface {
	BuildGeofences(ctx context.Context) (V2GeofencesResponse, error)
}

func (s V2GeofencesService) BuildGeofences(ctx context.Context) (V2GeofencesResponse, error) {
	if s.db == nil {
		return V2GeofencesResponse{}, errors.New("db not configured")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, latitude::float8, longitude::float8, radius::float8,
		       cost_per_unit::float8, billing_type::text, session_fee::float8
		FROM geofences
		ORDER BY name ASC
	`)
	if err != nil {
		return V2GeofencesResponse{}, err
	}
	defer rows.Close()
	items := []V2Geofence{}
	for rows.Next() {
		var (
			id            int64
			name          string
			lat, lng      sql.NullFloat64
			radius        sql.NullFloat64
			costPerUnit   sql.NullFloat64
			billingType   sql.NullString
			sessionFee    sql.NullFloat64
		)
		if err := rows.Scan(&id, &name, &lat, &lng, &radius, &costPerUnit, &billingType, &sessionFee); err != nil {
			return V2GeofencesResponse{}, err
		}
		gf := V2Geofence{ID: id, Name: name}
		if lat.Valid {
			gf.Latitude = &lat.Float64
		}
		if lng.Valid {
			gf.Longitude = &lng.Float64
		}
		if radius.Valid {
			gf.Radius = &radius.Float64
		}
		if costPerUnit.Valid {
			gf.CostPerUnit = &costPerUnit.Float64
		}
		if billingType.Valid {
			s := billingType.String
			gf.BillingType = &s
		}
		if sessionFee.Valid {
			gf.SessionFee = &sessionFee.Float64
		}
		items = append(items, gf)
	}
	return V2GeofencesResponse{Items: items}, rows.Err()
}

// Geofences godoc
//
// @Summary V2 list of geofences with billing rules
// @Description Returns all geofences with their location, radius, billing type, cost per unit, and per-session fee.
// @Tags v2
// @Produce json
// @Success 200 {object} V2GeofencesAPIResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/geofences [get]
func (h V2Handlers) Geofences(c *gin.Context) {
	if h.geofencesBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 geofences service is not configured.", nil)
		return
	}
	response, err := h.geofencesBuilder.BuildGeofences(c.Request.Context())
	if err != nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 geofences.", err.Error())
		return
	}
	meta := V2Meta{
		Period:      "lifetime",
		Timezone:    "UTC",
		Unit:        defaultV2Unit(),
		GeneratedAt: h.now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	v2JSON(c, http.StatusOK, response, meta)
}
