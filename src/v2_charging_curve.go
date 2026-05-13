package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @name V2ChargingCurveAPIResponse
type V2ChargingCurveAPIResponse struct {
	Data V2ChargingCurveResponse `json:"data"` // 响应数据
	Meta V2Meta `json:"meta"` // 响应元信息
}

// @name V2ChargingCurveResponse
type V2ChargingCurveResponse struct {
	MinSessions int                       `json:"min_sessions"`
	Samples     []V2ChargingCurveSample `json:"samples"` // 样本
}

// @name V2ChargingCurveSample
type V2ChargingCurveSample struct {
	BatteryLevel int `json:"battery_level"` // 电量百分比
	SessionCount int64 `json:"session_count"` // 充电会话数
	MedianPower  float64 `json:"median_power"`
	P25Power     float64 `json:"p25_power"`
	P75Power     float64 `json:"p75_power"`
	AvgVoltage   float64 `json:"avg_voltage,omitempty"`
}

// V2ChargingCurveRepository fetches DC charging curve aggregations.
type V2ChargingCurveRepository interface {
	Curve(ctx context.Context, carID int64, start timeBound, end timeBound, minSessions int) ([]V2ChargingCurveSample, error)
	CarExists(ctx context.Context, carID int64) (bool, error)
}

// PostgresV2ChargingCurveRepository implements V2ChargingCurveRepository against Postgres.
type PostgresV2ChargingCurveRepository struct {
	db *sql.DB
}

func NewPostgresV2ChargingCurveRepository(db *sql.DB) PostgresV2ChargingCurveRepository {
	return PostgresV2ChargingCurveRepository{db: db}
}

func (r PostgresV2ChargingCurveRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2ChargingCurveRepository) Curve(ctx context.Context, carID int64, start timeBound, end timeBound, minSessions int) ([]V2ChargingCurveSample, error) {
	// Restrict to DC sessions: charger_phases IS NULL marks DC fast-charging in TeslaMate.
	// Aggregate (battery_level, session) → median charger_power; then per-level percentiles
	// across sessions so a long session does not dominate.
	rows, err := r.db.QueryContext(ctx, `
		WITH per_session AS (
			SELECT
				c.battery_level,
				c.charging_process_id,
				PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY c.charger_power) AS session_power,
				AVG(NULLIF(c.charger_voltage, 0)) AS session_voltage
			FROM charges c
			JOIN charging_processes cp ON cp.id = c.charging_process_id
			WHERE cp.car_id = $1
				AND cp.start_date >= $2::timestamptz
				AND cp.start_date < $3::timestamptz
				AND c.charger_power IS NOT NULL
				AND c.battery_level IS NOT NULL
				AND c.charger_phases IS NULL
			GROUP BY c.battery_level, c.charging_process_id
		)
		SELECT
			battery_level,
			COUNT(*) AS session_count,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY session_power) AS median_power,
			PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY session_power) AS p25_power,
			PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY session_power) AS p75_power,
			AVG(session_voltage) AS avg_voltage
		FROM per_session
		GROUP BY battery_level
		HAVING COUNT(*) >= $4
		ORDER BY battery_level ASC
	`, carID, start.Time, end.Time, minSessions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := []V2ChargingCurveSample{}
	for rows.Next() {
		var sample V2ChargingCurveSample
		var avgVoltage sql.NullFloat64
		if err := rows.Scan(&sample.BatteryLevel, &sample.SessionCount, &sample.MedianPower, &sample.P25Power, &sample.P75Power, &avgVoltage); err != nil {
			return nil, err
		}
		if avgVoltage.Valid {
			sample.AvgVoltage = avgVoltage.Float64
		}
		samples = append(samples, sample)
	}
	return samples, rows.Err()
}

// V2ChargingCurveService produces the curve response for one car.
type V2ChargingCurveService struct {
	repository V2ChargingCurveRepository
}

func NewV2ChargingCurveService(repository V2ChargingCurveRepository) V2ChargingCurveService {
	return V2ChargingCurveService{repository: repository}
}

func (s V2ChargingCurveService) BuildCurve(ctx context.Context, carIDParam string, timeRange V2TimeRange, minSessions int) (V2ChargingCurveResponse, int64, error) {
	if minSessions <= 0 {
		minSessions = 5
	}
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return V2ChargingCurveResponse{}, 0, errors.New("invalid car id")
	}
	if s.repository != nil {
		exists, err := s.repository.CarExists(ctx, carID)
		if err != nil {
			return V2ChargingCurveResponse{}, carID, err
		}
		if !exists {
			return V2ChargingCurveResponse{}, carID, errV2CarNotFound
		}
	}
	samples, err := s.repository.Curve(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End), minSessions)
	if err != nil {
		return V2ChargingCurveResponse{}, carID, err
	}
	return V2ChargingCurveResponse{MinSessions: minSessions, Samples: samples}, carID, nil
}

// ChargingCurve godoc
//
// @Summary V2 直流充电曲线（聚合）
// @Description 返回该车直流充电曲线的聚合样本（按电量分桶：会话数、功率中位数 / P25 / P75）。
// @Tags v2
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param period query string false "聚合周期" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "起始时间（RFC3339）"
// @Param end query string false "结束时间（RFC3339）"
// @Param timezone query string false "IANA 时区"
// @Param min_sessions query int false "每个电量分桶的最少会话数（默认 5）"
// @Success 200 {object} V2ChargingCurveAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/charging/curve [get]
func (h V2Handlers) ChargingCurve(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.chargingCurveBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 charging curve service is not configured.", nil)
		return
	}
	minSessions := 5
	if v := c.Query("min_sessions"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			minSessions = parsed
		}
	}
	response, carID, err := h.chargingCurveBuilder.BuildCurve(c.Request.Context(), c.Param("CarID"), timeRange, minSessions)
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case err.Error() == "invalid car id":
			v2BadRequest(c, "Invalid car id.", nil)
		default:
			v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 charging curve.", err.Error())
		}
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}
