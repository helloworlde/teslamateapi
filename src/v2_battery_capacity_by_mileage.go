package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @name V2CapacityByMileageAPIResponse
type V2CapacityByMileageAPIResponse struct {
	Data V2CapacityByMileageResponse `json:"data"` // 响应数据
	Meta V2Meta `json:"meta"` // 响应元信息
}

// @name V2CapacityByMileageResponse
type V2CapacityByMileageResponse struct {
	Items []V2CapacityByMileageItem `json:"items"` // 条目列表
}

// @name V2CapacityByMileageItem
type V2CapacityByMileageItem struct {
	Bucket       string  `json:"bucket"`
	OdometerAvg  float64 `json:"odometer_avg"`
	CapacityKwh  float64 `json:"capacity_kwh_p50"`
	SampleCount  int64   `json:"sample_count"`
}

// V2CapacityByMileageRepository fetches half-month capacity buckets.
type V2CapacityByMileageRepository interface {
	CapacityByMileage(ctx context.Context, carID int64, start, end timeBound) ([]V2CapacityByMileageItem, error)
	CarExists(ctx context.Context, carID int64) (bool, error)
}

type PostgresV2CapacityByMileageRepository struct {
	db *sql.DB
}

func NewPostgresV2CapacityByMileageRepository(db *sql.DB) PostgresV2CapacityByMileageRepository {
	return PostgresV2CapacityByMileageRepository{db: db}
}

func (r PostgresV2CapacityByMileageRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

func (r PostgresV2CapacityByMileageRepository) CapacityByMileage(ctx context.Context, carID int64, start, end timeBound) ([]V2CapacityByMileageItem, error) {
	// Half-month bucket label: YYYYMM + '1' (days 1-15) or '2' (16-end).
	// Capacity per session: rated_battery_range_km × car.efficiency / usable_battery_level on the
	// last sample of each charging_processes row.
	rows, err := r.db.QueryContext(ctx, `
		WITH car_eff AS (
			SELECT COALESCE(efficiency, 0.15) AS efficiency FROM cars WHERE id = $1
		),
		last_sample AS (
			SELECT DISTINCT ON (cp.id)
				cp.id AS cp_id,
				cp.end_date,
				c.rated_battery_range_km,
				c.usable_battery_level,
				cp.end_km AS odo
			FROM charging_processes cp
			JOIN charges c ON c.charging_process_id = cp.id
			WHERE cp.car_id = $1
			  AND cp.end_date IS NOT NULL
			  AND cp.end_date >= $2::timestamptz
			  AND cp.end_date <  $3::timestamptz
			  AND c.usable_battery_level > 0
			  AND c.rated_battery_range_km IS NOT NULL
			ORDER BY cp.id, c.date DESC
		)
		SELECT
			to_char(end_date, 'YYYYMM') || CASE WHEN EXTRACT(DAY FROM end_date)::int <= 15 THEN '1' ELSE '2' END AS bucket,
			AVG(odo)::float8 AS odometer_avg,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY rated_battery_range_km * (SELECT efficiency FROM car_eff) / NULLIF(usable_battery_level, 0) * 100) AS capacity_kwh_p50,
			COUNT(*) AS sample_count
		FROM last_sample
		GROUP BY 1
		ORDER BY 1 ASC
	`, carID, start.Time, end.Time)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []V2CapacityByMileageItem{}
	for rows.Next() {
		var (
			bucket  string
			odo     sql.NullFloat64
			cap50   sql.NullFloat64
			samples sql.NullInt64
		)
		if err := rows.Scan(&bucket, &odo, &cap50, &samples); err != nil {
			return nil, err
		}
		item := V2CapacityByMileageItem{Bucket: bucket}
		if odo.Valid {
			item.OdometerAvg = odo.Float64
		}
		if cap50.Valid {
			item.CapacityKwh = cap50.Float64
		}
		if samples.Valid {
			item.SampleCount = samples.Int64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// V2CapacityByMileageService produces half-month capacity buckets for one car.
type V2CapacityByMileageService struct {
	repository V2CapacityByMileageRepository
}

func NewV2CapacityByMileageService(repository V2CapacityByMileageRepository) V2CapacityByMileageService {
	return V2CapacityByMileageService{repository: repository}
}

func (s V2CapacityByMileageService) BuildCapacityByMileage(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2CapacityByMileageResponse, int64, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return V2CapacityByMileageResponse{}, 0, errors.New("invalid car id")
	}
	if s.repository != nil {
		exists, err := s.repository.CarExists(ctx, carID)
		if err != nil {
			return V2CapacityByMileageResponse{}, carID, err
		}
		if !exists {
			return V2CapacityByMileageResponse{}, carID, errV2CarNotFound
		}
	}
	items, err := s.repository.CapacityByMileage(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End))
	if err != nil {
		return V2CapacityByMileageResponse{}, carID, err
	}
	return V2CapacityByMileageResponse{Items: items}, carID, nil
}

// CapacityByMileage godoc
//
// @Summary V2 电池容量随里程趋势
// @Description 按半月分桶返回该车的中位电池容量 (kWh) 与里程关系。
// @Tags v2
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param period query string false "聚合周期" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "起始时间（RFC3339）"
// @Param end query string false "结束时间（RFC3339）"
// @Param timezone query string false "IANA 时区"
// @Success 200 {object} V2CapacityByMileageAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/battery/capacity_by_mileage [get]
func (h V2Handlers) CapacityByMileage(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.capacityByMileageBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 capacity-by-mileage service is not configured.", nil)
		return
	}
	response, carID, err := h.capacityByMileageBuilder.BuildCapacityByMileage(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case err.Error() == "invalid car id":
			v2BadRequest(c, "Invalid car id.", nil)
		default:
			v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 capacity by mileage.", err.Error())
		}
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}
