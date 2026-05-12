package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// @name V2EfficiencyAPIResponse
type V2EfficiencyAPIResponse struct {
	Data V2EfficiencyResponse `json:"data"`
	Meta V2Meta               `json:"meta"`
}

// @name V2EfficiencyResponse
type V2EfficiencyResponse struct {
	Summary    V2EfficiencySummary           `json:"summary"`
	Timeseries *V2EfficiencyTimeseries       `json:"timeseries,omitempty"`
	Buckets    *V2EfficiencyBuckets          `json:"buckets,omitempty"`
}

// @name V2EfficiencySummary
type V2EfficiencySummary struct {
	NetConsumption       *float64 `json:"net_consumption,omitempty"`
	GrossConsumption     *float64 `json:"gross_consumption,omitempty"`
	ConsumptionOverhead  *float64 `json:"consumption_overhead,omitempty"`
	DriveDistance        float64  `json:"drive_distance"`
	DriveDuration        float64  `json:"drive_duration"`
	EnergyConsumedDrives float64  `json:"energy_consumed_drives"`
	EnergyConsumedIdle   float64  `json:"energy_consumed_idle"`
	AvgOutsideTemp       *float64 `json:"avg_outside_temp,omitempty"`
}

// @name V2EfficiencyTimeseries
type V2EfficiencyTimeseries struct {
	GroupBy string                       `json:"group_by"`
	Items   []V2EfficiencyTimeseriesItem `json:"items"`
}

// @name V2EfficiencyTimeseriesItem
type V2EfficiencyTimeseriesItem struct {
	PeriodStart         string   `json:"period_start"`
	NetConsumption      *float64 `json:"net_consumption,omitempty"`
	GrossConsumption    *float64 `json:"gross_consumption,omitempty"`
	ConsumptionOverhead *float64 `json:"consumption_overhead,omitempty"`
	Distance            float64  `json:"distance"`
}

// @name V2EfficiencyBuckets
type V2EfficiencyBuckets struct {
	Temperature5C []V2EfficiencyTempBucket  `json:"temperature_5c,omitempty"`
	Speed10Kmh    []V2EfficiencySpeedBucket `json:"speed_10kmh,omitempty"`
}

// @name V2EfficiencyTempBucket
type V2EfficiencyTempBucket struct {
	Bucket      float64  `json:"bucket"`
	DriveCount  int64    `json:"drive_count"`
	Distance    float64  `json:"distance"`
	Consumption *float64 `json:"consumption,omitempty"`
	AvgSpeed    *float64 `json:"avg_speed,omitempty"`
}

// @name V2EfficiencySpeedBucket
type V2EfficiencySpeedBucket struct {
	Bucket      float64  `json:"bucket"`
	DriveCount  int64    `json:"drive_count"`
	Distance    float64  `json:"distance"`
	Consumption *float64 `json:"consumption,omitempty"`
}

// V2EfficiencyRepository fetches efficiency aggregations.
type V2EfficiencyRepository interface {
	Summary(ctx context.Context, carID int64, start, end timeBound) (V2EfficiencySummary, error)
	TempBuckets(ctx context.Context, carID int64, start, end timeBound) ([]V2EfficiencyTempBucket, error)
	SpeedBuckets(ctx context.Context, carID int64, start, end timeBound) ([]V2EfficiencySpeedBucket, error)
	CarExists(ctx context.Context, carID int64) (bool, error)
}

type PostgresV2EfficiencyRepository struct {
	db *sql.DB
}

func NewPostgresV2EfficiencyRepository(db *sql.DB) PostgresV2EfficiencyRepository {
	return PostgresV2EfficiencyRepository{db: db}
}

func (r PostgresV2EfficiencyRepository) CarExists(ctx context.Context, carID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID).Scan(&exists)
	return exists, err
}

// Summary computes net + gross consumption + drives/idle energy from drives in the window.
// gross_consumption = (drive_energy + idle_energy) / drive_distance, where idle_energy is
// derived from end_rated_range gaps between consecutive drives × car efficiency.
func (r PostgresV2EfficiencyRepository) Summary(ctx context.Context, carID int64, start, end timeBound) (V2EfficiencySummary, error) {
	var summary V2EfficiencySummary
	var (
		distance        sql.NullFloat64
		duration        sql.NullFloat64
		driveEnergy     sql.NullFloat64
		idleEnergy      sql.NullFloat64
		netConsumption  sql.NullFloat64
		grossConsumption sql.NullFloat64
		avgOutsideTemp  sql.NullFloat64
	)

	err := r.db.QueryRowContext(ctx, `
		WITH d AS (
			SELECT
				dr.id,
				dr.start_date,
				dr.end_date,
				dr.distance,
				dr.duration_min,
				dr.start_rated_range_km,
				dr.end_rated_range_km,
				dr.outside_temp_avg,
				LAG(dr.end_rated_range_km) OVER (ORDER BY dr.start_date) AS prev_end_rated_range_km
			FROM drives dr
			WHERE dr.car_id = $1
			  AND dr.start_date >= $2::timestamptz
			  AND dr.start_date < $3::timestamptz
			  AND dr.end_date IS NOT NULL
			  AND dr.distance IS NOT NULL
		),
		car_eff AS (
			SELECT COALESCE(efficiency, 0.15) AS efficiency FROM cars WHERE id = $1
		)
		SELECT
			COALESCE(SUM(d.distance), 0) AS distance,
			COALESCE(SUM(d.duration_min) * 60, 0) AS duration_seconds,
			COALESCE(SUM(GREATEST((d.start_rated_range_km - d.end_rated_range_km), 0) * (SELECT efficiency FROM car_eff)), 0) AS drive_energy,
			COALESCE(SUM(GREATEST((COALESCE(d.prev_end_rated_range_km, d.start_rated_range_km) - d.start_rated_range_km), 0) * (SELECT efficiency FROM car_eff)), 0) AS idle_energy,
			CASE WHEN SUM(d.distance) > 0
			     THEN (SUM(GREATEST((d.start_rated_range_km - d.end_rated_range_km), 0) * (SELECT efficiency FROM car_eff)) * 1000) / SUM(d.distance)
			END AS net_consumption,
			CASE WHEN SUM(d.distance) > 0
			     THEN (SUM(GREATEST((d.start_rated_range_km - d.end_rated_range_km), 0) * (SELECT efficiency FROM car_eff))
			          + SUM(GREATEST((COALESCE(d.prev_end_rated_range_km, d.start_rated_range_km) - d.start_rated_range_km), 0) * (SELECT efficiency FROM car_eff))) * 1000 / SUM(d.distance)
			END AS gross_consumption,
			AVG(d.outside_temp_avg) AS avg_outside_temp
		FROM d
	`, carID, start.Time, end.Time).Scan(&distance, &duration, &driveEnergy, &idleEnergy, &netConsumption, &grossConsumption, &avgOutsideTemp)
	if err != nil {
		return summary, err
	}

	if distance.Valid {
		summary.DriveDistance = distance.Float64
	}
	if duration.Valid {
		summary.DriveDuration = duration.Float64
	}
	if driveEnergy.Valid {
		summary.EnergyConsumedDrives = driveEnergy.Float64
	}
	if idleEnergy.Valid {
		summary.EnergyConsumedIdle = idleEnergy.Float64
	}
	if netConsumption.Valid {
		v := netConsumption.Float64
		summary.NetConsumption = &v
	}
	if grossConsumption.Valid {
		v := grossConsumption.Float64
		summary.GrossConsumption = &v
		// consumption_overhead = (gross − net) / gross
		if netConsumption.Valid && grossConsumption.Float64 > 0 {
			oh := (grossConsumption.Float64 - netConsumption.Float64) / grossConsumption.Float64
			summary.ConsumptionOverhead = &oh
		}
	}
	if avgOutsideTemp.Valid {
		v := avgOutsideTemp.Float64
		summary.AvgOutsideTemp = &v
	}
	return summary, nil
}

func (r PostgresV2EfficiencyRepository) TempBuckets(ctx context.Context, carID int64, start, end timeBound) ([]V2EfficiencyTempBucket, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH car_eff AS (
			SELECT COALESCE(efficiency, 0.15) AS efficiency FROM cars WHERE id = $1
		)
		SELECT
			ROUND(dr.outside_temp_avg / 5.0) * 5.0 AS bucket,
			COUNT(*) AS drive_count,
			SUM(dr.distance) AS distance,
			CASE WHEN SUM(dr.distance) > 0
			     THEN (SUM(GREATEST((dr.start_rated_range_km - dr.end_rated_range_km), 0) * (SELECT efficiency FROM car_eff)) * 1000) / SUM(dr.distance)
			END AS consumption,
			CASE WHEN SUM(dr.duration_min) > 0
			     THEN SUM(dr.distance) / (SUM(dr.duration_min) / 60.0)
			END AS avg_speed
		FROM drives dr
		WHERE dr.car_id = $1
		  AND dr.start_date >= $2::timestamptz
		  AND dr.start_date < $3::timestamptz
		  AND dr.end_date IS NOT NULL
		  AND dr.distance IS NOT NULL
		  AND dr.outside_temp_avg IS NOT NULL
		GROUP BY ROUND(dr.outside_temp_avg / 5.0) * 5.0
		ORDER BY bucket
	`, carID, start.Time, end.Time)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buckets := []V2EfficiencyTempBucket{}
	for rows.Next() {
		var b V2EfficiencyTempBucket
		var consumption, avgSpeed sql.NullFloat64
		if err := rows.Scan(&b.Bucket, &b.DriveCount, &b.Distance, &consumption, &avgSpeed); err != nil {
			return nil, err
		}
		if consumption.Valid {
			v := consumption.Float64
			b.Consumption = &v
		}
		if avgSpeed.Valid {
			v := avgSpeed.Float64
			b.AvgSpeed = &v
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

func (r PostgresV2EfficiencyRepository) SpeedBuckets(ctx context.Context, carID int64, start, end timeBound) ([]V2EfficiencySpeedBucket, error) {
	// Use per-drive avg speed (distance / duration_h). Bucket by 10 km/h. Position-level speed
	// histograms are heavier; this gives the same shape at much lower cost.
	rows, err := r.db.QueryContext(ctx, `
		WITH car_eff AS (
			SELECT COALESCE(efficiency, 0.15) AS efficiency FROM cars WHERE id = $1
		),
		per_drive AS (
			SELECT
				dr.distance,
				dr.duration_min,
				CASE WHEN dr.duration_min > 0 THEN dr.distance / (dr.duration_min / 60.0) ELSE NULL END AS avg_speed,
				GREATEST((dr.start_rated_range_km - dr.end_rated_range_km), 0) AS rated_loss
			FROM drives dr
			WHERE dr.car_id = $1
			  AND dr.start_date >= $2::timestamptz
			  AND dr.start_date < $3::timestamptz
			  AND dr.end_date IS NOT NULL
			  AND dr.distance IS NOT NULL
			  AND dr.duration_min > 0
		)
		SELECT
			FLOOR(avg_speed / 10.0) * 10.0 AS bucket,
			COUNT(*) AS drive_count,
			SUM(distance) AS distance,
			CASE WHEN SUM(distance) > 0
			     THEN (SUM(rated_loss) * (SELECT efficiency FROM car_eff) * 1000) / SUM(distance)
			END AS consumption
		FROM per_drive
		WHERE avg_speed IS NOT NULL
		GROUP BY FLOOR(avg_speed / 10.0) * 10.0
		ORDER BY bucket
	`, carID, start.Time, end.Time)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buckets := []V2EfficiencySpeedBucket{}
	for rows.Next() {
		var b V2EfficiencySpeedBucket
		var consumption sql.NullFloat64
		if err := rows.Scan(&b.Bucket, &b.DriveCount, &b.Distance, &consumption); err != nil {
			return nil, err
		}
		if consumption.Valid {
			v := consumption.Float64
			b.Consumption = &v
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

// V2EfficiencyService composes summary + bucket responses.
type V2EfficiencyService struct {
	repository V2EfficiencyRepository
}

func NewV2EfficiencyService(repository V2EfficiencyRepository) V2EfficiencyService {
	return V2EfficiencyService{repository: repository}
}

type V2EfficiencyBuildOptions struct {
	IncludeBuckets bool
	GroupBy        string // "temperature_5c" | "speed_10kmh" | ""
}

func (s V2EfficiencyService) BuildEfficiency(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2EfficiencyBuildOptions) (V2EfficiencyResponse, int64, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return V2EfficiencyResponse{}, 0, errors.New("invalid car id")
	}
	if s.repository != nil {
		exists, err := s.repository.CarExists(ctx, carID)
		if err != nil {
			return V2EfficiencyResponse{}, carID, err
		}
		if !exists {
			return V2EfficiencyResponse{}, carID, errV2CarNotFound
		}
	}
	start, end := asTimeBound(timeRange.Start), asTimeBound(timeRange.End)
	summary, err := s.repository.Summary(ctx, carID, start, end)
	if err != nil {
		return V2EfficiencyResponse{}, carID, err
	}
	resp := V2EfficiencyResponse{Summary: summary}
	if opts.IncludeBuckets {
		groupBy := strings.ToLower(opts.GroupBy)
		buckets := V2EfficiencyBuckets{}
		switch groupBy {
		case "", "temperature_5c":
			items, err := s.repository.TempBuckets(ctx, carID, start, end)
			if err != nil {
				return V2EfficiencyResponse{}, carID, err
			}
			buckets.Temperature5C = items
			if groupBy != "" {
				resp.Buckets = &buckets
				return resp, carID, nil
			}
		}
		if groupBy == "" || groupBy == "speed_10kmh" {
			items, err := s.repository.SpeedBuckets(ctx, carID, start, end)
			if err != nil {
				return V2EfficiencyResponse{}, carID, err
			}
			buckets.Speed10Kmh = items
		}
		resp.Buckets = &buckets
	}
	return resp, carID, nil
}

// Efficiency godoc
//
// @Summary V2 efficiency analytics
// @Description Returns net + gross consumption, consumption overhead, plus optional temperature / speed buckets for one car.
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param include query string false "Comma-separated extras: buckets" example("buckets")
// @Param group_by query string false "Bucket grouping (used when include=buckets)" Enums(temperature_5c, speed_10kmh)
// @Success 200 {object} V2EfficiencyAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/efficiency [get]
func (h V2Handlers) Efficiency(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.efficiencyBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 efficiency service is not configured.", nil)
		return
	}
	include := parseV2IncludeSet(c.Query("include"))
	opts := V2EfficiencyBuildOptions{
		IncludeBuckets: include["buckets"],
		GroupBy:        c.Query("group_by"),
	}
	response, carID, err := h.efficiencyBuilder.BuildEfficiency(c.Request.Context(), c.Param("CarID"), timeRange, opts)
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case err.Error() == "invalid car id":
			v2BadRequest(c, "Invalid car id.", nil)
		default:
			v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 efficiency analytics.", err.Error())
		}
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}
