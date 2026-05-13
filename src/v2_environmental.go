package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @name V2EnvironmentalAPIResponse
type V2EnvironmentalAPIResponse struct {
	Data V2EnvironmentalResponse `json:"data"` // 响应数据
	Meta V2Meta `json:"meta"` // 响应元信息
}

// @name V2EnvironmentalResponse
type V2EnvironmentalResponse struct {
	Summary    V2EnvironmentalSummary       `json:"summary"`
	Timeseries *V2EnvironmentalTimeseries `json:"timeseries,omitempty"` // 时序
}

// @name V2EnvironmentalSummary
type V2EnvironmentalSummary struct {
	OutsideTempMin       *float64 `json:"outside_temp_min,omitempty"` // 车外最低温 (°C)
	OutsideTempMax       *float64 `json:"outside_temp_max,omitempty"` // 车外最高温 (°C)
	OutsideTempAvg       *float64 `json:"outside_temp_avg,omitempty"` // 车外平均温 (°C)
	InsideTempAvg        *float64 `json:"inside_temp_avg,omitempty"` // 车内平均温 (°C)
	ClimateOnMinutes     *float64 `json:"climate_on_minutes,omitempty"` // 空调开启时长 (分)
	BatteryHeaterMinutes *float64 `json:"battery_heater_minutes,omitempty"` // 电池加热器时长 (分)
	DefrosterMinutes     *float64 `json:"defroster_minutes,omitempty"` // 除霜时长 (分)
	ElevationMin         *float64 `json:"elevation_min,omitempty"` // 最低海拔 (米)
	ElevationMax         *float64 `json:"elevation_max,omitempty"` // 最高海拔 (米)
	ElevationGainTotal   *float64 `json:"elevation_gain_total,omitempty"` // 累计爬升 (米)
	ElevationLossTotal   *float64 `json:"elevation_loss_total,omitempty"` // 累计下降 (米)
}

// @name V2EnvironmentalTimeseries
type V2EnvironmentalTimeseries struct {
	GroupBy string `json:"group_by"` // 聚合粒度
	Items   []V2EnvironmentalTimeseriesItem `json:"items"` // 条目列表
}

// @name V2EnvironmentalTimeseriesItem
type V2EnvironmentalTimeseriesItem struct {
	PeriodStart    string `json:"period_start"` // 周期起始
	OutsideTempAvg *float64 `json:"outside_temp_avg,omitempty"` // 车外平均温 (°C)
	InsideTempAvg  *float64 `json:"inside_temp_avg,omitempty"` // 车内平均温 (°C)
	ElevationAvg   *float64 `json:"elevation_avg,omitempty"`
	ClimateOnRatio *float64 `json:"climate_on_ratio,omitempty"`
}

// V2EnvironmentalRepository fetches HVAC / temp / elevation aggregations.
type V2EnvironmentalRepository interface {
	Summary(ctx context.Context, carID int64, start, end timeBound) (V2EnvironmentalSummary, error)
	Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2EnvironmentalTimeseriesItem, error)
	CarExists(ctx context.Context, carID int64) (bool, error)
}

type PostgresV2EnvironmentalRepository struct {
	db *sql.DB
}

func NewPostgresV2EnvironmentalRepository(db *sql.DB) PostgresV2EnvironmentalRepository {
	return PostgresV2EnvironmentalRepository{db: db}
}

func (r PostgresV2EnvironmentalRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

// Summary aggregates positions (temps / HVAC / elevation) and drives (ascent/descent) within the window.
// climate_on_minutes / battery_heater_minutes / defroster_minutes are derived by sampling positions ordered
// by date and summing per-row time deltas where the corresponding flag is true (clamped to ≤ 5 minutes per
// gap to avoid over-counting long sleeps).
func (r PostgresV2EnvironmentalRepository) Summary(ctx context.Context, carID int64, start, end timeBound) (V2EnvironmentalSummary, error) {
	var summary V2EnvironmentalSummary
	var (
		oMin, oMax, oAvg, iAvg sql.NullFloat64
		eMin, eMax             sql.NullFloat64
		ascentSum, descentSum  sql.NullFloat64
		climateMin, heaterMin  sql.NullFloat64
		defrosterMin           sql.NullFloat64
	)
	err := r.db.QueryRowContext(ctx, `
		WITH p AS (
			SELECT
				date,
				outside_temp::float8 AS outside_temp,
				inside_temp::float8 AS inside_temp,
				elevation::float8 AS elevation,
				is_climate_on,
				battery_heater_on,
				is_front_defroster_on,
				is_rear_defroster_on,
				LEAST(
					EXTRACT(EPOCH FROM (date - LAG(date) OVER (ORDER BY date))) / 60.0,
					5.0
				) AS minutes
			FROM positions
			WHERE car_id = $1 AND date >= $2::timestamptz AND date < $3::timestamptz
		),
		drv AS (
			SELECT
				COALESCE(SUM(ascent), 0)  AS ascent_total,
				COALESCE(SUM(descent), 0) AS descent_total
			FROM drives
			WHERE car_id = $1
			  AND end_date IS NOT NULL
			  AND end_date   >= $2::timestamptz
			  AND start_date <  $3::timestamptz
		)
		SELECT
			MIN(p.outside_temp), MAX(p.outside_temp), AVG(p.outside_temp),
			AVG(p.inside_temp),
			MIN(p.elevation), MAX(p.elevation),
			(SELECT ascent_total FROM drv),
			(SELECT descent_total FROM drv),
			SUM(CASE WHEN p.is_climate_on AND p.minutes IS NOT NULL THEN p.minutes ELSE 0 END),
			SUM(CASE WHEN p.battery_heater_on AND p.minutes IS NOT NULL THEN p.minutes ELSE 0 END),
			SUM(CASE WHEN (p.is_front_defroster_on OR p.is_rear_defroster_on) AND p.minutes IS NOT NULL THEN p.minutes ELSE 0 END)
		FROM p
	`, carID, start.Time, end.Time).Scan(
		&oMin, &oMax, &oAvg, &iAvg,
		&eMin, &eMax,
		&ascentSum, &descentSum,
		&climateMin, &heaterMin, &defrosterMin,
	)
	if err != nil {
		return summary, err
	}
	if oMin.Valid {
		summary.OutsideTempMin = &oMin.Float64
	}
	if oMax.Valid {
		summary.OutsideTempMax = &oMax.Float64
	}
	if oAvg.Valid {
		summary.OutsideTempAvg = &oAvg.Float64
	}
	if iAvg.Valid {
		summary.InsideTempAvg = &iAvg.Float64
	}
	if eMin.Valid {
		summary.ElevationMin = &eMin.Float64
	}
	if eMax.Valid {
		summary.ElevationMax = &eMax.Float64
	}
	if ascentSum.Valid {
		summary.ElevationGainTotal = &ascentSum.Float64
	}
	if descentSum.Valid {
		summary.ElevationLossTotal = &descentSum.Float64
	}
	if climateMin.Valid {
		summary.ClimateOnMinutes = &climateMin.Float64
	}
	if heaterMin.Valid {
		summary.BatteryHeaterMinutes = &heaterMin.Float64
	}
	if defrosterMin.Valid {
		summary.DefrosterMinutes = &defrosterMin.Float64
	}
	return summary, nil
}

func (r PostgresV2EnvironmentalRepository) Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2EnvironmentalTimeseriesItem, error) {
	truncUnit := postgresDateTruncUnit(groupBy)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		WITH p AS (
			SELECT
				GREATEST(date_trunc('%s', date AT TIME ZONE 'UTC' AT TIME ZONE $4), $2 AT TIME ZONE $4) AT TIME ZONE $4 AS period_start,
				outside_temp::float8 AS outside_temp,
				inside_temp::float8 AS inside_temp,
				elevation::float8 AS elevation,
				is_climate_on,
				LEAST(
					EXTRACT(EPOCH FROM (date - LAG(date) OVER (ORDER BY date))) / 60.0,
					5.0
				) AS minutes
			FROM positions
			WHERE car_id = $1 AND date >= $2 AND date < $3
		)
		SELECT
			period_start,
			AVG(outside_temp) AS outside_temp_avg,
			AVG(inside_temp)  AS inside_temp_avg,
			AVG(elevation)    AS elevation_avg,
			CASE
				WHEN SUM(minutes) > 0 THEN SUM(CASE WHEN is_climate_on AND minutes IS NOT NULL THEN minutes ELSE 0 END) / SUM(minutes)
				ELSE NULL
			END AS climate_on_ratio
		FROM p
		GROUP BY 1
		ORDER BY 1`, truncUnit),
		carID, asTimeBound(timeRange.Start).Time, asTimeBound(timeRange.End).Time, timeRange.Timezone,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []V2EnvironmentalTimeseriesItem{}
	for rows.Next() {
		var (
			periodStart string
			oAvg, iAvg  sql.NullFloat64
			eAvg, ratio sql.NullFloat64
		)
		if err := rows.Scan(&periodStart, &oAvg, &iAvg, &eAvg, &ratio); err != nil {
			return nil, err
		}
		item := V2EnvironmentalTimeseriesItem{PeriodStart: periodStart}
		if oAvg.Valid {
			item.OutsideTempAvg = &oAvg.Float64
		}
		if iAvg.Valid {
			item.InsideTempAvg = &iAvg.Float64
		}
		if eAvg.Valid {
			item.ElevationAvg = &eAvg.Float64
		}
		if ratio.Valid {
			item.ClimateOnRatio = &ratio.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// V2EnvironmentalBuilder builds environmental analytics responses.
type V2EnvironmentalBuilder interface {
	BuildEnvironmental(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2EnvironmentalBuildOptions) (V2EnvironmentalResponse, int64, error)
}

// V2EnvironmentalBuildOptions controls optional sections.
type V2EnvironmentalBuildOptions struct {
	IncludeTimeseries bool
	GroupBy           string
}

// V2EnvironmentalService produces environmental analytics responses.
type V2EnvironmentalService struct {
	repository V2EnvironmentalRepository
}

func NewV2EnvironmentalService(repository V2EnvironmentalRepository) V2EnvironmentalService {
	return V2EnvironmentalService{repository: repository}
}

func (s V2EnvironmentalService) BuildEnvironmental(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2EnvironmentalBuildOptions) (V2EnvironmentalResponse, int64, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return V2EnvironmentalResponse{}, 0, errors.New("invalid car id")
	}
	if s.repository != nil {
		exists, err := s.repository.CarExists(ctx, carID)
		if err != nil {
			return V2EnvironmentalResponse{}, carID, err
		}
		if !exists {
			return V2EnvironmentalResponse{}, carID, errV2CarNotFound
		}
	}
	start, end := asTimeBound(timeRange.Start), asTimeBound(timeRange.End)
	summary, err := s.repository.Summary(ctx, carID, start, end)
	if err != nil {
		return V2EnvironmentalResponse{}, carID, err
	}
	response := V2EnvironmentalResponse{Summary: summary}
	if opts.IncludeTimeseries {
		groupBy := opts.GroupBy
		switch groupBy {
		case "", "day", "week", "month", "year":
		default:
			return V2EnvironmentalResponse{}, carID, errV2InvalidDrivingGroupBy
		}
		if groupBy == "" {
			groupBy = "day"
		}
		items, err := s.repository.Timeseries(ctx, carID, timeRange, groupBy)
		if err != nil {
			return V2EnvironmentalResponse{}, carID, err
		}
		response.Timeseries = &V2EnvironmentalTimeseries{
			GroupBy: groupBy,
			Items:   items,
		}
	}
	return response, carID, nil
}

// Environmental godoc
//
// @Summary V2 环境分析（温度/HVAC/海拔）
// @Description 返回该车的车外/车内温度聚合、HVAC 活动时长、海拔统计。
// @Tags v2
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param period query string false "聚合周期" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "起始时间（RFC3339）"
// @Param end query string false "结束时间（RFC3339）"
// @Param timezone query string false "IANA 时区"
// @Param include query string false "逗号分隔的扩展项，如 timeseries" example("timeseries")
// @Param group_by query string false "时序聚合粒度（仅在 include=timeseries 时生效）" Enums(day, week, month, year)
// @Success 200 {object} V2EnvironmentalAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/environmental [get]
func (h V2Handlers) Environmental(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.environmentalBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 environmental service is not configured.", nil)
		return
	}
	include := parseV2IncludeSet(c.Query("include"))
	opts := V2EnvironmentalBuildOptions{
		IncludeTimeseries: include["timeseries"],
		GroupBy:           c.Query("group_by"),
	}
	response, carID, err := h.environmentalBuilder.BuildEnvironmental(c.Request.Context(), c.Param("CarID"), timeRange, opts)
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case errors.Is(err, errV2InvalidDrivingGroupBy):
			v2BadRequest(c, "Invalid environmental group_by.", nil)
		case err.Error() == "invalid car id":
			v2BadRequest(c, "Invalid car id.", nil)
		default:
			v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 environmental analytics.", err.Error())
		}
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}
