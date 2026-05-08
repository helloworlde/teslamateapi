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

type V2ChargingBuilder interface {
	BuildCharging(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingResponse, V2DataQuality, int64, error)
	BuildChargingTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingTimeseriesResponse, V2DataQuality, int64, error)
	BuildChargingLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingLocationsResponse, V2DataQuality, int64, error)
	BuildChargingTypes(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingTypesResponse, V2DataQuality, int64, error)
	BuildChargingCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingCostResponse, V2DataQuality, int64, error)
}

type V2Handlers struct {
	summaryBuilder  V2SummaryBuilder
	drivingBuilder  V2DrivingBuilder
	chargingBuilder V2ChargingBuilder
	now             func() time.Time
}

func NewV2Handlers(summaryBuilder V2SummaryBuilder, drivingBuilder V2DrivingBuilder, chargingBuilder ...V2ChargingBuilder) V2Handlers {
	handlers := V2Handlers{
		summaryBuilder: summaryBuilder,
		drivingBuilder: drivingBuilder,
		now:            time.Now,
	}
	if len(chargingBuilder) > 0 {
		handlers.chargingBuilder = chargingBuilder[0]
	}
	return handlers
}

func RegisterV2Routes(api *gin.RouterGroup, summaryRepository V2SummaryRepository) {
	var drivingRepository V2DrivingRepository
	var chargingRepository V2ChargingRepository
	if summaryRepository == nil && db != nil {
		repository := NewPostgresV2SummaryRepository(db)
		summaryRepository = repository
		drivingRepository = NewPostgresV2DrivingRepository(db)
		chargingRepository = NewPostgresV2ChargingRepository(db)
	}
	handlers := NewV2Handlers(NewV2SummaryService(summaryRepository), NewV2DrivingService(drivingRepository), NewV2ChargingService(chargingRepository))

	v2 := api.Group("/v2")
	{
		v2.GET("", handlers.Info)
		v2.GET("/", handlers.Info)
		v2.GET("/cars/:CarID/analytics/summary", handlers.Summary)
		v2.GET("/cars/:CarID/analytics/driving", handlers.Driving)
		v2.GET("/cars/:CarID/analytics/driving/timeseries", handlers.DrivingTimeseries)
		v2.GET("/cars/:CarID/analytics/driving/distribution", handlers.DrivingDistribution)
		v2.GET("/cars/:CarID/analytics/driving/ranking", handlers.DrivingRanking)
		v2.GET("/cars/:CarID/analytics/charging", handlers.Charging)
		v2.GET("/cars/:CarID/analytics/charging/timeseries", handlers.ChargingTimeseries)
		v2.GET("/cars/:CarID/analytics/charging/locations", handlers.ChargingLocations)
		v2.GET("/cars/:CarID/analytics/charging/types", handlers.ChargingTypes)
		v2.GET("/cars/:CarID/analytics/charging/cost", handlers.ChargingCost)
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

// Charging godoc
//
// @Summary V2 charging analytics summary
// @Description Returns objective charging statistics and optional previous-period comparison for one car.
// @Tags V2 Charging Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param compare query string false "Comparison mode" Enums(none, previous_period, previous_year, lifetime_average)
// @Success 200 {object} V2ChargingAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/charging [get]
func (h V2Handlers) Charging(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.chargingBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 charging service is not configured.", nil)
		return
	}
	response, quality, carID, err := h.chargingBuilder.BuildCharging(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

// ChargingTimeseries godoc
//
// @Summary V2 charging analytics timeseries
// @Description Returns charging metrics grouped by day, week, month, or year for charting.
// @Tags V2 Charging Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param group_by query string false "Timeseries grouping" Enums(day, week, month, year)
// @Success 200 {object} V2ChargingTimeseriesAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/charging/timeseries [get]
func (h V2Handlers) ChargingTimeseries(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, quality, carID, err := h.chargingBuilder.BuildChargingTimeseries(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

// ChargingLocations godoc
//
// @Summary V2 charging analytics by location
// @Description Returns objective charging metrics grouped by geofence or address.
// @Tags V2 Charging Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} V2ChargingLocationsAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/charging/locations [get]
func (h V2Handlers) ChargingLocations(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, quality, carID, err := h.chargingBuilder.BuildChargingLocations(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

// ChargingTypes godoc
//
// @Summary V2 charging analytics by charger type
// @Description Returns objective charging metrics grouped by AC, DC, Tesla Supercharger, or unknown type.
// @Tags V2 Charging Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} V2ChargingTypesAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/charging/types [get]
func (h V2Handlers) ChargingTypes(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, quality, carID, err := h.chargingBuilder.BuildChargingTypes(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

// ChargingCost godoc
//
// @Summary V2 charging cost analytics
// @Description Returns objective charging cost, energy, and distance-normalized cost metrics.
// @Tags V2 Charging Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param group_by query string false "Cost grouping" Enums(day, week, month, year)
// @Success 200 {object} V2ChargingCostAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/charging/cost [get]
func (h V2Handlers) ChargingCost(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, quality, carID, err := h.chargingBuilder.BuildChargingCost(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange, &quality))
}

func handleV2ChargingError(c *gin.Context, err error, timeRange V2TimeRange) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
	case errors.Is(err, errV2CompareUnsupported):
		v2Error(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Requested comparison mode is not implemented for this API.", gin.H{"compare": timeRange.Compare})
	case errors.Is(err, errV2InvalidDrivingGroupBy):
		v2BadRequest(c, "Invalid charging group_by.", nil)
	case err.Error() == "invalid car id":
		v2BadRequest(c, "Invalid car id.", nil)
	default:
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 charging analytics.", err.Error())
	}
}
