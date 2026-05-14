package v2

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
)

// ----------------- /v2/cars/{CarID}/lifecycle/odometer_series -----------------

// @name V2OdometerSeriesAPIResponse
type V2OdometerSeriesAPIResponse struct {
	Data V2OdometerSeriesResponse `json:"data"` // 响应数据
	Meta V2Meta                   `json:"meta"` // 响应元信息
}

// @name V2OdometerSeriesResponse
type V2OdometerSeriesResponse struct {
	Items      []V2OdometerSeriesItem `json:"items"`                 // 条目列表
	HasMore    bool                   `json:"has_more"`              // 是否还有更多
	NextCursor *string                `json:"next_cursor,omitempty"` // 下一页游标
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
// @Summary V2 累计里程序列
// @Description 返回该车在所选窗口内每日一条里程读数。
// @Tags v2
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param period query string false "聚合周期" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "起始时间（RFC3339）"
// @Param end query string false "结束时间（RFC3339）"
// @Param timezone query string false "IANA 时区"
// @Param limit query int false "每页大小（默认 500，最大 1000）"
// @Param cursor query string false "由上一次响应返回的分页游标"
// @Success 200 {object} V2OdometerSeriesAPIResponse
// @Failure 400 {object} apicommon.APIErrorResponse
// @Failure 404 {object} apicommon.APIErrorResponse
// @Failure 500 {object} apicommon.APIErrorResponse
// @Router /v2/cars/{CarID}/lifecycle/odometer_series [get]
func (h V2Handlers) OdometerSeries(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		apicommon.V2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.odometerSeriesBuilder == nil {
		apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 odometer series service is not configured.", nil)
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
			apicommon.V2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case err.Error() == "invalid car id":
			apicommon.V2BadRequest(c, "Invalid car id.", nil)
		default:
			apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 odometer series.", err.Error())
		}
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ----------------- /v2/cars/{CarID}/summary/by_period -----------------

// @name V2SummaryByPeriodAPIResponse
type V2SummaryByPeriodAPIResponse struct {
	Data V2SummaryByPeriodResponse `json:"data"` // 响应数据
	Meta V2Meta                    `json:"meta"` // 响应元信息
}

// @name V2SummaryByPeriodResponse
type V2SummaryByPeriodResponse struct {
	GroupBy string                  `json:"group_by"` // 聚合粒度
	Items   []V2SummaryByPeriodItem `json:"items"`    // 条目列表
}

// @name V2SummaryByPeriodItem
type V2SummaryByPeriodItem struct {
	PeriodStart      string   `json:"period_start"` // 周期起始
	DriveCount       int64    `json:"drive_count"`  // 行程数
	Distance         float64  `json:"distance"`     // 距离 (km)
	DriveDuration    float64  `json:"drive_duration"`
	NetEnergy        *float64 `json:"net_energy,omitempty"`      // 净能量 (kWh)
	AvgConsumption   *float64 `json:"avg_consumption,omitempty"` // 平均能耗 (Wh/km)
	ChargingSessions int64    `json:"charging_sessions"`         // 充电会话数
	EnergyAdded      float64  `json:"energy_added"`              // 充入电池能量 (kWh)
	EnergyUsed       float64  `json:"energy_used"`               // 墙端用电 (kWh)
	ChargingCost     *float64 `json:"charging_cost,omitempty"`   // 充电费用
	DataComplete     bool     `json:"data_complete"`             // 数据是否完整
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
			periodStart                           string
			driveCount, sessionCount              sql.NullInt64
			distance, driveDuration, netEnergy    sql.NullFloat64
			energyAdded, energyUsed, chargingCost sql.NullFloat64
			complete                              sql.NullBool
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
// @Summary V2 周期扁平汇总
// @Description 返回按周期扁平化的行驶 + 充电 + 费用指标（每周期一行）。
// @Tags v2
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param period query string false "聚合周期" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "起始时间（RFC3339）"
// @Param end query string false "结束时间（RFC3339）"
// @Param timezone query string false "IANA 时区"
// @Param group_by query string false "周期聚合粒度" Enums(day, week, month, year)
// @Success 200 {object} V2SummaryByPeriodAPIResponse
// @Failure 400 {object} apicommon.APIErrorResponse
// @Failure 404 {object} apicommon.APIErrorResponse
// @Failure 500 {object} apicommon.APIErrorResponse
// @Router /v2/cars/{CarID}/summary/by_period [get]
func (h V2Handlers) SummaryByPeriod(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		apicommon.V2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.summaryByPeriodBuilder == nil {
		apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 summary-by-period service is not configured.", nil)
		return
	}
	response, carID, err := h.summaryByPeriodBuilder.BuildByPeriod(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			apicommon.V2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case errors.Is(err, errV2InvalidDrivingGroupBy):
			apicommon.V2BadRequest(c, "Invalid summary group_by.", nil)
		case err.Error() == "invalid car id":
			apicommon.V2BadRequest(c, "Invalid car id.", nil)
		default:
			apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 by_period summary.", err.Error())
		}
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ----------------- /v2/cars/{CarID}/lifecycle/places -----------------

// @name V2PlacesAPIResponse
type V2PlacesAPIResponse struct {
	Data V2PlacesResponse `json:"data"` // 响应数据
	Meta V2Meta           `json:"meta"` // 响应元信息
}

// @name V2PlacesResponse
type V2PlacesResponse struct {
	Cities    []V2PlaceBucket `json:"cities"`    // 城市分布
	States    []V2PlaceBucket `json:"states"`    // 州/省分布
	Countries []V2PlaceBucket `json:"countries"` // 国家分布
}

// @name V2PlaceBucket
type V2PlaceBucket struct {
	Name        string  `json:"name"`                   // 名称
	VisitCount  int64   `json:"visit_count"`            // 访问次数
	LastVisited *string `json:"last_visited,omitempty"` // 最后访问时间
}

type V2PlacesService struct {
	db *sql.DB
}

func NewV2PlacesService(db *sql.DB) V2PlacesService {
	return V2PlacesService{db: db}
}

// V2PlacesOptions captures the optional date-range filter for the places
// endpoint. When both Start and End are nil the service returns the lifetime
// distribution (audit §2.5: clients want to ask "where did I drive last
// summer?", which the lifetime view cannot answer).
type V2PlacesOptions struct {
	Start *time.Time
	End   *time.Time
}

type V2PlacesBuilder interface {
	BuildPlaces(ctx context.Context, carIDParam string, topN int, opts V2PlacesOptions) (V2PlacesResponse, int64, error)
}

func (s V2PlacesService) BuildPlaces(ctx context.Context, carIDParam string, topN int, opts V2PlacesOptions) (V2PlacesResponse, int64, error) {
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
	loc := timeRangeLocation(V2TimeRange{Timezone: defaultV2Timezone()})
	queryFor := func(col string) ([]V2PlaceBucket, error) {
		// Bind the optional date range as $3/$4 so we don't have to splice the
		// SQL based on which side is set; nil sql.NullTime acts as an open bound.
		rows, err := s.db.QueryContext(ctx, `
			SELECT a.`+col+` AS name,
			       COUNT(*) AS visit_count,
			       MAX(d.end_date) AS last_visited
			FROM drives d
			JOIN addresses a ON a.id = d.end_address_id
			WHERE d.car_id = $1
			  AND d.end_date IS NOT NULL
			  AND ($3::timestamptz IS NULL OR d.start_date >= $3)
			  AND ($4::timestamptz IS NULL OR d.start_date < $4)
			  AND a.`+col+` IS NOT NULL AND a.`+col+` <> ''
			GROUP BY 1
			ORDER BY visit_count DESC
			LIMIT $2
		`, carID, topN, nullTimePtr(opts.Start), nullTimePtr(opts.End))
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []V2PlaceBucket{}
		for rows.Next() {
			var (
				name        string
				count       int64
				lastVisited sql.NullTime
			)
			if err := rows.Scan(&name, &count, &lastVisited); err != nil {
				return nil, err
			}
			b := V2PlaceBucket{Name: name, VisitCount: count}
			if lastVisited.Valid {
				formatted := lastVisited.Time.In(loc).Format(time.RFC3339)
				b.LastVisited = &formatted
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

// nullTimePtr converts an optional time.Time into a sql.NullTime so the same
// SQL can serve "lifetime" and "ranged" queries without branching.
func nullTimePtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// Places godoc
//
// @Summary V2 生命周期地点分布（城市/州/国家）
// @Description 返回该车访问最多的前 N 个城市/州/国家，含最后访问时间。可通过 start/end (RFC3339 或 YYYY-MM-DD) 指定时间范围；不传则返回全生命周期数据。
// @Tags v2
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param top_n query int false "每个维度的 Top N（默认 20，最大 100）"
// @Param start query string false "起始时间 (RFC3339 或 YYYY-MM-DD)"
// @Param end query string false "结束时间 (RFC3339 或 YYYY-MM-DD)"
// @Param timezone query string false "解析 start/end 时使用的时区，默认服务器配置"
// @Success 200 {object} V2PlacesAPIResponse
// @Failure 400 {object} apicommon.APIErrorResponse
// @Failure 404 {object} apicommon.APIErrorResponse
// @Failure 500 {object} apicommon.APIErrorResponse
// @Router /v2/cars/{CarID}/lifecycle/places [get]
func (h V2Handlers) Places(c *gin.Context) {
	if h.placesBuilder == nil {
		apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 places service is not configured.", nil)
		return
	}
	topN := 20
	if v := c.Query("top_n"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			topN = parsed
		}
	}

	// audit §2.5: optional start/end so callers can scope to a season /
	// vacation / month. Reuses parseV2ClientTime so RFC3339 + YYYY-MM-DD work
	// the same way as on the analytics endpoints.
	tz := v2QueryDefault(c, "timezone", defaultV2Timezone())
	location, err := time.LoadLocation(tz)
	if err != nil {
		apicommon.V2BadRequest(c, "Invalid timezone.", err.Error())
		return
	}
	var opts V2PlacesOptions
	if raw := strings.TrimSpace(c.Query("start")); raw != "" {
		t, perr := parseV2ClientTime(raw, location)
		if perr != nil {
			apicommon.V2BadRequest(c, "Invalid start.", perr.Error())
			return
		}
		utc := t.UTC()
		opts.Start = &utc
	}
	if raw := strings.TrimSpace(c.Query("end")); raw != "" {
		t, perr := parseV2ClientTime(raw, location)
		if perr != nil {
			apicommon.V2BadRequest(c, "Invalid end.", perr.Error())
			return
		}
		utc := t.UTC()
		opts.End = &utc
	}
	if opts.Start != nil && opts.End != nil && !opts.End.After(*opts.Start) {
		apicommon.V2BadRequest(c, "end must be after start.", nil)
		return
	}

	response, carID, err := h.placesBuilder.BuildPlaces(c.Request.Context(), c.Param("CarID"), topN, opts)
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			apicommon.V2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case err.Error() == "invalid car id":
			apicommon.V2BadRequest(c, "Invalid car id.", nil)
		default:
			apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 lifecycle places.", err.Error())
		}
		return
	}
	loc := timeRangeLocation(V2TimeRange{Timezone: tz})
	period := "lifetime"
	if opts.Start != nil || opts.End != nil {
		period = "custom"
	}
	meta := V2Meta{
		CarID:       carID,
		Period:      period,
		Timezone:    tz,
		Unit:        defaultV2Unit(),
		GeneratedAt: h.now().In(loc).Format(time.RFC3339),
	}
	if opts.Start != nil {
		meta.Start = opts.Start.In(loc).Format(time.RFC3339)
	}
	if opts.End != nil {
		meta.End = opts.End.In(loc).Format(time.RFC3339)
	}
	// Echo back the resolved filters so callers can confirm the server picked
	// up start/end/timezone/top_n exactly as expected (spec §5.2).
	filters := map[string]string{
		"timezone": tz,
		"top_n":    strconv.Itoa(topN),
	}
	if opts.Start != nil {
		filters["start"] = opts.Start.In(loc).Format(time.RFC3339)
	}
	if opts.End != nil {
		filters["end"] = opts.End.In(loc).Format(time.RFC3339)
	}
	meta.AppliedFilters = filters
	v2JSON(c, http.StatusOK, response, meta)
}

// ----------------- /v2/geofences (global) -----------------

// @name V2GeofencesAPIResponse
type V2GeofencesAPIResponse struct {
	Data V2GeofencesResponse `json:"data"` // 响应数据
	Meta V2Meta              `json:"meta"` // 响应元信息
}

// @name V2GeofencesResponse
type V2GeofencesResponse struct {
	Items []V2Geofence `json:"items"` // 条目列表
}

// @name V2Geofence
type V2Geofence struct {
	ID          int64    `json:"id"`                      // ID
	Name        string   `json:"name"`                    // 名称
	Latitude    *float64 `json:"latitude,omitempty"`      // 纬度
	Longitude   *float64 `json:"longitude,omitempty"`     // 经度
	Radius      *float64 `json:"radius,omitempty"`        // 半径 (米)
	CostPerUnit *float64 `json:"cost_per_unit,omitempty"` // 单位费用
	BillingType *string  `json:"billing_type,omitempty"`  // 计费类型
	SessionFee  *float64 `json:"session_fee,omitempty"`   // 每次会话固定费用
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
			id          int64
			name        string
			lat, lng    sql.NullFloat64
			radius      sql.NullFloat64
			costPerUnit sql.NullFloat64
			billingType sql.NullString
			sessionFee  sql.NullFloat64
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
// @Summary V2 围栏列表（含计费规则）
// @Description 返回全部围栏：位置、半径、计费类型、单位费用、每次会话固定费用。
// @Tags v2
// @Produce json
// @Success 200 {object} V2GeofencesAPIResponse
// @Failure 500 {object} apicommon.APIErrorResponse
// @Router /v2/geofences [get]
func (h V2Handlers) Geofences(c *gin.Context) {
	if h.geofencesBuilder == nil {
		apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 geofences service is not configured.", nil)
		return
	}
	response, err := h.geofencesBuilder.BuildGeofences(c.Request.Context())
	if err != nil {
		apicommon.V2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 geofences.", err.Error())
		return
	}
	tz := defaultV2Timezone()
	loc := timeRangeLocation(V2TimeRange{Timezone: tz})
	meta := V2Meta{
		Period:      "lifetime",
		Timezone:    tz,
		Unit:        defaultV2Unit(),
		GeneratedAt: h.now().In(loc).Format(time.RFC3339),
	}
	v2JSON(c, http.StatusOK, response, meta)
}
