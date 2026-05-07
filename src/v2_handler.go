package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type V2SummaryBuilder interface {
	BuildSummary(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2SummaryResponse, V2DataQuality, error)
}

type V2DrivingBuilder interface {
	BuildDriving(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2DrivingResponse, V2DataQuality, int64, error)
	BuildTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2DrivingTimeseriesResponse, V2DataQuality, int64, error)
	BuildDistribution(ctx context.Context, carIDParam string, timeRange V2TimeRange, dimension string) (V2DrivingDistributionResponse, V2DataQuality, int64, error)
	BuildRanking(ctx context.Context, carIDParam string, timeRange V2TimeRange, rankingType string, limit int) (V2DrivingRankingResponse, V2DataQuality, int64, error)
}

type V2Handlers struct {
	summaryBuilder V2SummaryBuilder
	drivingBuilder V2DrivingBuilder
	now            func() time.Time
}

func NewV2Handlers(summaryBuilder V2SummaryBuilder, drivingBuilder V2DrivingBuilder) V2Handlers {
	return V2Handlers{
		summaryBuilder: summaryBuilder,
		drivingBuilder: drivingBuilder,
		now:            time.Now,
	}
}

func RegisterV2Routes(api *gin.RouterGroup, summaryRepository V2SummaryRepository) {
	var drivingRepository V2DrivingRepository
	if summaryRepository == nil && db != nil {
		repository := NewPostgresV2SummaryRepository(db)
		summaryRepository = repository
		drivingRepository = NewPostgresV2DrivingRepository(db)
	}
	handlers := NewV2Handlers(NewV2SummaryService(summaryRepository), NewV2DrivingService(drivingRepository))

	v2 := api.Group("/v2")
	{
		v2.GET("", handlers.Info)
		v2.GET("/", handlers.Info)
		v2.GET("/cars/:CarID/analytics/summary", handlers.Summary)
		v2.GET("/cars/:CarID/analytics/driving", handlers.Driving)
		v2.GET("/cars/:CarID/analytics/driving/timeseries", handlers.DrivingTimeseries)
		v2.GET("/cars/:CarID/analytics/driving/distribution", handlers.DrivingDistribution)
		v2.GET("/cars/:CarID/analytics/driving/ranking", handlers.DrivingRanking)
	}
}

// Info godoc
//
// @Summary V2 API capability information
// @Description Returns the V2 analytics API version, scope, and currently advertised feature groups.
// @Tags V2 Summary
// @Produce json
// @Success 200 {object} V2InfoAPIResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2 [get]
func (h V2Handlers) Info(c *gin.Context) {
	meta := V2Meta{
		Unit:        defaultV2Unit(),
		GeneratedAt: time.Now().In(timeRangeLocation(V2TimeRange{Timezone: defaultV2Timezone()})).Format(time.RFC3339),
	}
	v2JSON(c, http.StatusOK, V2InfoResponse{
		Version: "v2",
		Scope:   "analytics",
		Features: []string{
			"summary",
			"driving",
			"charging",
			"parking",
			"battery",
			"efficiency",
			"cost",
			"locations",
			"updates",
			"lifecycle",
			"calendar",
			"reports",
			"insights",
		},
	}, meta)
}

// Summary godoc
//
// @Summary V2 period summary analytics
// @Description Returns objective driving, charging, parking, battery, update, and charging-cost summary metrics for one car in a selected period.
// @Tags V2 Summary
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param compare query string false "Comparison mode" Enums(none, previous_period, previous_year, lifetime_average)
// @Success 200 {object} V2SummaryAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/summary [get]
func (h V2Handlers) Summary(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}

	if h.summaryBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 summary service is not configured.", nil)
		return
	}

	response, quality, err := h.summaryBuilder.BuildSummary(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		switch {
		case errors.Is(err, errV2CarNotFound):
			v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
		case errors.Is(err, errV2CompareUnsupported):
			v2Error(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Requested comparison mode is not implemented for this API.", gin.H{"compare": timeRange.Compare})
		case err.Error() == "invalid car id":
			v2BadRequest(c, "Invalid car id.", nil)
		default:
			v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 summary.", err.Error())
		}
		return
	}

	carID, _ := strconv.ParseInt(c.Param("CarID"), 10, 64)
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

// Driving godoc
//
// @Summary V2 driving analytics summary
// @Description Returns objective driving statistics and optional previous-period comparison for one car.
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param compare query string false "Comparison mode" Enums(none, previous_period, previous_year, lifetime_average)
// @Success 200 {object} V2DrivingAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/driving [get]
func (h V2Handlers) Driving(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.drivingBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 driving service is not configured.", nil)
		return
	}
	response, quality, carID, err := h.drivingBuilder.BuildDriving(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2DrivingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

// DrivingTimeseries godoc
//
// @Summary V2 driving analytics timeseries
// @Description Returns driving metrics grouped by day, week, month, or year for charting.
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param group_by query string false "Timeseries grouping" Enums(day, week, month, year)
// @Success 200 {object} V2DrivingTimeseriesAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/driving/timeseries [get]
func (h V2Handlers) DrivingTimeseries(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, quality, carID, err := h.drivingBuilder.BuildTimeseries(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		handleV2DrivingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

// DrivingDistribution godoc
//
// @Summary V2 driving analytics distribution
// @Description Returns drive-count, distance, and duration distribution for a selected factual dimension.
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param dimension query string false "Distribution dimension" Enums(hour_of_day, day_of_week, distance_bucket, duration_bucket, speed_bucket, consumption_bucket, temperature_bucket)
// @Success 200 {object} V2DrivingDistributionAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/driving/distribution [get]
func (h V2Handlers) DrivingDistribution(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, quality, carID, err := h.drivingBuilder.BuildDistribution(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("dimension"))
	if err != nil {
		handleV2DrivingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

// DrivingRanking godoc
//
// @Summary V2 driving analytics ranking
// @Description Returns objective top drives or top driving days by selected ranking type.
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param type query string false "Ranking type" Enums(longest_distance, longest_duration, highest_speed, lowest_consumption, highest_consumption, highest_distance_day)
// @Param limit query int false "Result limit from 1 to 100"
// @Success 200 {object} V2DrivingRankingAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/driving/ranking [get]
func (h V2Handlers) DrivingRanking(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, quality, carID, err := h.drivingBuilder.BuildRanking(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("type"), v2LimitFromQuery(c.Query("limit")))
	if err != nil {
		handleV2DrivingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

func handleV2DrivingError(c *gin.Context, err error, timeRange V2TimeRange) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
	case errors.Is(err, errV2CompareUnsupported):
		v2Error(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Requested comparison mode is not implemented for this API.", gin.H{"compare": timeRange.Compare})
	case errors.Is(err, errV2InvalidDrivingDimension):
		v2BadRequest(c, "Invalid driving distribution dimension.", nil)
	case errors.Is(err, errV2InvalidDrivingRanking):
		v2BadRequest(c, "Invalid driving ranking type.", nil)
	case errors.Is(err, errV2InvalidDrivingGroupBy):
		v2BadRequest(c, "Invalid driving group_by.", nil)
	case err.Error() == "invalid car id":
		v2BadRequest(c, "Invalid car id.", nil)
	default:
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 driving analytics.", err.Error())
	}
}
