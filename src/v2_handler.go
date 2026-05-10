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
	BuildSummary(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2SummaryResponse, error)
}

type V2DrivingBuilder interface {
	BuildDriving(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2DrivingResponse, int64, error)
	BuildTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2DrivingTimeseriesResponse, int64, error)
	BuildDistribution(ctx context.Context, carIDParam string, timeRange V2TimeRange, dimension string) (V2DrivingDistributionResponse, int64, error)
	BuildRanking(ctx context.Context, carIDParam string, timeRange V2TimeRange, rankingType string, limit int) (V2DrivingRankingResponse, int64, error)
}

type V2ChargingBuilder interface {
	BuildCharging(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingResponse, int64, error)
	BuildChargingTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingTimeseriesResponse, int64, error)
	BuildChargingLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingLocationsResponse, int64, error)
	BuildChargingTypes(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingTypesResponse, int64, error)
	BuildChargingCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingCostResponse, int64, error)
}

type V2ParkingBuilder interface {
	BuildParking(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ParkingResponse, int64, error)
	BuildParkingLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ParkingLocationsResponse, int64, error)
	BuildParkingStates(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ParkingStatesResponse, int64, error)
}

type V2BatteryBuilder interface {
	BuildBattery(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2BatteryResponse, int64, error)
	BuildBatteryTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2BatteryTimeseriesResponse, int64, error)
	BuildBatteryDistribution(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2BatteryDistributionResponse, int64, error)
}

type V2EfficiencyBuilder interface {
	BuildEfficiency(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2EfficiencyResponse, int64, error)
	BuildEfficiencyFactors(ctx context.Context, carIDParam string, timeRange V2TimeRange, dimension string) (V2EfficiencyFactorsResponse, int64, error)
}

type V2CostBuilder interface {
	BuildCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2CostResponse, int64, error)
}

type V2LocationBuilder interface {
	BuildLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange, sort string) (V2LocationAnalyticsResponse, int64, error)
}

type V2UpdateBuilder interface {
	BuildUpdates(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2UpdateAnalyticsResponse, int64, error)
}

type V2LifecycleBuilder interface {
	BuildLifecycle(ctx context.Context, carIDParam string, asOf time.Time) (V2LifecycleResponse, error)
	BuildTimeline(ctx context.Context, carIDParam string, eventTypes []string, limit int, before, after *time.Time) (V2TimelineResponse, error)
}

type V2CalendarBuilder interface {
	BuildCalendar(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2CalendarResponse, int64, error)
}

type V2ReportBuilder interface {
	BuildReport(ctx context.Context, carIDParam string, timeRange V2TimeRange, include []string) (V2ReportResponse, error)
}

type V2InsightBuilder interface {
	BuildInsights(ctx context.Context, carIDParam string, timeRange V2TimeRange, category string, minSeverity string) (V2InsightResponse, error)
}

// @name V2Handlers
type V2Handlers struct {
	summaryBuilder    V2SummaryBuilder
	drivingBuilder    V2DrivingBuilder
	chargingBuilder   V2ChargingBuilder
	parkingBuilder    V2ParkingBuilder
	batteryBuilder    V2BatteryBuilder
	efficiencyBuilder V2EfficiencyBuilder
	costBuilder       V2CostBuilder
	locationBuilder   V2LocationBuilder
	updateBuilder     V2UpdateBuilder
	lifecycleBuilder  V2LifecycleBuilder
	calendarBuilder   V2CalendarBuilder
	reportBuilder     V2ReportBuilder
	insightBuilder    V2InsightBuilder
	now               func() time.Time
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
	var parkingRepository V2ParkingRepository
	var batteryRepository V2BatteryRepository
	var efficiencyRepository V2EfficiencyRepository
	var costRepository V2CostRepository
	var locationRepository V2LocationRepository
	var updateRepository V2UpdateRepository
	var lifecycleRepository V2LifecycleRepository
	var calendarRepository V2CalendarRepository
	if summaryRepository == nil && db != nil {
		repository := NewPostgresV2SummaryRepository(db)
		summaryRepository = repository
		drivingRepository = NewPostgresV2DrivingRepository(db)
		chargingRepository = NewPostgresV2ChargingRepository(db)
		parkingRepository = NewPostgresV2ParkingRepository(db)
		batteryRepository = NewPostgresV2BatteryRepository(db)
		efficiencyRepository = NewPostgresV2EfficiencyRepository(db)
		costRepository = NewPostgresV2CostRepository(db)
		locationRepository = NewPostgresV2LocationRepository(db)
		updateRepository = NewPostgresV2UpdateRepository(db)
		lifecycleRepository = NewPostgresV2LifecycleRepository(db)
		calendarRepository = NewPostgresV2CalendarRepository(db)
	}
	drivingService := NewV2DrivingService(drivingRepository)
	chargingService := NewV2ChargingService(chargingRepository)
	handlers := NewV2Handlers(NewV2SummaryService(summaryRepository), drivingService, chargingService)
	handlers.parkingBuilder = NewV2ParkingService(parkingRepository)
	handlers.batteryBuilder = NewV2BatteryService(batteryRepository)
	handlers.efficiencyBuilder = NewV2EfficiencyService(efficiencyRepository)
	handlers.costBuilder = NewV2CostService(costRepository)
	handlers.locationBuilder = NewV2LocationService(locationRepository)
	handlers.updateBuilder = NewV2UpdateService(updateRepository)
	handlers.lifecycleBuilder = NewV2LifecycleService(lifecycleRepository)
	handlers.calendarBuilder = NewV2CalendarService(calendarRepository)
	updateSvc := NewV2UpdateService(updateRepository)
	handlers.reportBuilder = NewV2ReportService(
		NewV2SummaryService(summaryRepository),
		drivingService,
		chargingService,
		updateSvc,
		handlers.parkingBuilder,
		handlers.batteryBuilder,
		handlers.efficiencyBuilder,
		handlers.costBuilder,
		handlers.locationBuilder,
		handlers.insightBuilder,
	)
	handlers.insightBuilder = NewV2InsightService(drivingService, chargingService)

	v2 := api.Group("/v2")
	{
		v2.GET("", handlers.Info)
		v2.GET("/", handlers.Info)
		// All car-scoped routes share a single car-validation middleware (one DB lookup per request).
		v2Cars := v2.Group("/cars/:CarID", v2CarValidationMiddleware())
		v2Cars.GET("/analytics/summary", handlers.Summary)
		v2Cars.GET("/analytics/driving", handlers.Driving)
		v2Cars.GET("/analytics/driving/timeseries", handlers.DrivingTimeseries)
		v2Cars.GET("/analytics/driving/distribution", handlers.DrivingDistribution)
		v2Cars.GET("/analytics/driving/ranking", handlers.DrivingRanking)
		v2Cars.GET("/analytics/charging", handlers.Charging)
		v2Cars.GET("/analytics/charging/timeseries", handlers.ChargingTimeseries)
		v2Cars.GET("/analytics/charging/locations", handlers.ChargingLocations)
		v2Cars.GET("/analytics/charging/types", handlers.ChargingTypes)
		v2Cars.GET("/analytics/charging/cost", handlers.ChargingCost)
		v2Cars.GET("/analytics/parking", handlers.Parking)
		v2Cars.GET("/analytics/parking/locations", handlers.ParkingLocations)
		v2Cars.GET("/analytics/parking/states", handlers.ParkingStates)
		v2Cars.GET("/analytics/battery", handlers.Battery)
		v2Cars.GET("/analytics/battery/timeseries", handlers.BatteryTimeseries)
		v2Cars.GET("/analytics/battery/distribution", handlers.BatteryDistribution)
		v2Cars.GET("/analytics/efficiency", handlers.Efficiency)
		v2Cars.GET("/analytics/efficiency/factors", handlers.EfficiencyFactors)
		v2Cars.GET("/analytics/cost", handlers.Cost)
		v2Cars.GET("/analytics/locations", handlers.Locations)
		v2Cars.GET("/analytics/updates", handlers.Updates)
		v2Cars.GET("/analytics/lifecycle", handlers.Lifecycle)
		v2Cars.GET("/timeline", handlers.Timeline)
		v2Cars.GET("/calendar", handlers.Calendar)
		v2Cars.GET("/reports", handlers.Reports)
		v2Cars.GET("/insights", handlers.Insights)
	}
}

// v2CarValidationMiddleware validates :CarID on every car-scoped V2 route.
// It performs a single DB lookup and aborts with 400/404 before reaching the handler.
// Car ID is stored in the gin context so handlers can retrieve it without re-parsing.
func v2CarValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		carIDStr := c.Param("CarID")
		carID, err := strconv.ParseInt(carIDStr, 10, 64)
		if err != nil || carID <= 0 {
			v2Error(c, http.StatusBadRequest, "INVALID_CAR_ID", "invalid car id", nil)
			c.Abort()
			return
		}
		if db != nil {
			var exists bool
			if err := db.QueryRowContext(c.Request.Context(),
				`SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID,
			).Scan(&exists); err != nil || !exists {
				v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "car not found", nil)
				c.Abort()
				return
			}
		}
		c.Set("v2CarID", carID)
		c.Next()
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
// @Param CarID path int true "Car ID" example(1)
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

	response, err := h.summaryBuilder.BuildSummary(c.Request.Context(), c.Param("CarID"), timeRange)
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
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// Driving godoc
//
// @Summary V2 driving analytics summary
// @Description Returns objective driving statistics and optional previous-period comparison for one car.
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
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
	response, carID, err := h.drivingBuilder.BuildDriving(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2DrivingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// DrivingTimeseries godoc
//
// @Summary V2 driving analytics timeseries
// @Description Returns driving metrics grouped by day, week, month, or year for charting.
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
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
	response, carID, err := h.drivingBuilder.BuildTimeseries(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		handleV2DrivingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// DrivingDistribution godoc
//
// @Summary V2 driving analytics distribution
// @Description Returns drive-count, distance, and duration distribution for a selected factual dimension.
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
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
	response, carID, err := h.drivingBuilder.BuildDistribution(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("dimension"))
	if err != nil {
		handleV2DrivingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// DrivingRanking godoc
//
// @Summary V2 driving analytics ranking
// @Description Returns objective top drives or top driving days by selected ranking type.
// @Tags V2 Driving Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
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
	response, carID, err := h.drivingBuilder.BuildRanking(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("type"), v2LimitFromQuery(c.Query("limit")))
	if err != nil {
		handleV2DrivingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
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
// @Param CarID path int true "Car ID" example(1)
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
	response, carID, err := h.chargingBuilder.BuildCharging(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ChargingTimeseries godoc
//
// @Summary V2 charging analytics timeseries
// @Description Returns charging metrics grouped by day, week, month, or year for charting.
// @Tags V2 Charging Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
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
	response, carID, err := h.chargingBuilder.BuildChargingTimeseries(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ChargingLocations godoc
//
// @Summary V2 charging analytics by location
// @Description Returns objective charging metrics grouped by geofence or address.
// @Tags V2 Charging Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
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
	response, carID, err := h.chargingBuilder.BuildChargingLocations(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ChargingTypes godoc
//
// @Summary V2 charging analytics by charger type
// @Description Returns objective charging metrics grouped by AC, DC, Tesla Supercharger, or unknown type.
// @Tags V2 Charging Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
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
	response, carID, err := h.chargingBuilder.BuildChargingTypes(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ChargingCost godoc
//
// @Summary V2 charging cost analytics
// @Description Returns objective charging cost, energy, and distance-normalized cost metrics.
// @Tags V2 Charging Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
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
	response, carID, err := h.chargingBuilder.BuildChargingCost(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		handleV2ChargingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
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

// Parking godoc
//
// @Summary V2 parking analytics summary
// @Description Returns objective parked duration, state duration, inferred parking sessions, and estimated parking drain for one car.
// @Tags V2 Parking Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param compare query string false "Comparison mode" Enums(none, previous_period, previous_year, lifetime_average)
// @Success 200 {object} V2ParkingAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/parking [get]
func (h V2Handlers) Parking(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.parkingBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 parking service is not configured.", nil)
		return
	}
	response, carID, err := h.parkingBuilder.BuildParking(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2ParkingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ParkingLocations godoc
//
// @Summary V2 parking analytics by location
// @Description Returns inferred parking sessions and parked duration grouped by geofence or address.
// @Tags V2 Parking Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} V2ParkingLocationsAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/parking/locations [get]
func (h V2Handlers) ParkingLocations(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, carID, err := h.parkingBuilder.BuildParkingLocations(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2ParkingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// ParkingStates godoc
//
// @Summary V2 parking state analytics
// @Description Returns online, asleep, offline, and unknown state durations, shares, and transition counts.
// @Tags V2 Parking Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} V2ParkingStatesAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/parking/states [get]
func (h V2Handlers) ParkingStates(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, carID, err := h.parkingBuilder.BuildParkingStates(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2ParkingError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

func handleV2ParkingError(c *gin.Context, err error, timeRange V2TimeRange) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
	case errors.Is(err, errV2CompareUnsupported):
		v2Error(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Requested comparison mode is not implemented for this API.", gin.H{"compare": timeRange.Compare})
	case err.Error() == "invalid car id":
		v2BadRequest(c, "Invalid car id.", nil)
	default:
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 parking analytics.", err.Error())
	}
}

// Battery godoc
//
// @Summary V2 battery analytics summary
// @Description Returns objective latest battery range samples, estimated full-range values, baseline range, and estimated range degradation. These estimates are not official state of health.
// @Tags V2 Battery Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param compare query string false "Comparison mode" Enums(none, previous_period, previous_year, lifetime_average)
// @Success 200 {object} V2BatteryAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/battery [get]
func (h V2Handlers) Battery(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.batteryBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 battery service is not configured.", nil)
		return
	}
	response, carID, err := h.batteryBuilder.BuildBattery(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2BatteryError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// BatteryTimeseries godoc
//
// @Summary V2 battery analytics timeseries
// @Description Returns estimated full rated and ideal range grouped by day, week, month, or year for trend charts.
// @Tags V2 Battery Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param group_by query string false "Timeseries grouping" Enums(day, week, month, year)
// @Success 200 {object} V2BatteryTimeseriesAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/battery/timeseries [get]
func (h V2Handlers) BatteryTimeseries(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, carID, err := h.batteryBuilder.BuildBatteryTimeseries(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		handleV2BatteryError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// BatteryDistribution godoc
//
// @Summary V2 battery level distribution
// @Description Returns battery_level sample counts grouped into 10 percent buckets from 0-10 through 90-100.
// @Tags V2 Battery Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} V2BatteryDistributionAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/battery/distribution [get]
func (h V2Handlers) BatteryDistribution(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, carID, err := h.batteryBuilder.BuildBatteryDistribution(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2BatteryError(c, err, timeRange)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

func handleV2BatteryError(c *gin.Context, err error, timeRange V2TimeRange) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
	case errors.Is(err, errV2CompareUnsupported):
		v2Error(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Requested comparison mode is not implemented for this API.", gin.H{"compare": timeRange.Compare})
	case errors.Is(err, errV2InvalidDrivingGroupBy):
		v2BadRequest(c, "Invalid battery group_by.", nil)
	case err.Error() == "invalid car id":
		v2BadRequest(c, "Invalid car id.", nil)
	default:
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 battery analytics.", err.Error())
	}
}

// Efficiency godoc
//
// @Summary V2 efficiency analytics summary
// @Description Returns objective drive efficiency metrics, including estimated energy consumption, average/best/worst consumption, average temperature, and average speed.
// @Tags V2 Efficiency Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
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
	response, carID, err := h.efficiencyBuilder.BuildEfficiency(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2EfficiencyError(c, err)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// EfficiencyFactors godoc
//
// @Summary V2 efficiency factor analytics
// @Description Returns factual efficiency metrics grouped by one selected dimension. Bucketed results do not imply causation.
// @Tags V2 Efficiency Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param dimension query string false "Factor dimension" Enums(temperature, speed, distance, elevation, location, hour_of_day, day_of_week)
// @Success 200 {object} V2EfficiencyFactorsAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/efficiency/factors [get]
func (h V2Handlers) EfficiencyFactors(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	response, carID, err := h.efficiencyBuilder.BuildEfficiencyFactors(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("dimension"))
	if err != nil {
		handleV2EfficiencyError(c, err)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

func handleV2EfficiencyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
	case errors.Is(err, errV2InvalidEfficiencyDimension):
		v2BadRequest(c, "Invalid efficiency dimension.", nil)
	case err.Error() == "invalid car id":
		v2BadRequest(c, "Invalid car id.", nil)
	default:
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 efficiency analytics.", err.Error())
	}
}

// Cost godoc
//
// @Summary V2 cost analytics
// @Description Returns objective charging-cost analytics. Current data scope includes charging_cost only and excludes insurance, maintenance, parking, depreciation, tire, and repair costs.
// @Tags V2 Cost Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param group_by query string false "Cost grouping" Enums(day, week, month, year)
// @Success 200 {object} V2CostAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/cost [get]
func (h V2Handlers) Cost(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.costBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 cost service is not configured.", nil)
		return
	}
	response, carID, err := h.costBuilder.BuildCost(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("group_by"))
	if err != nil {
		handleV2CostError(c, err)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

func handleV2CostError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
	case errors.Is(err, errV2InvalidDrivingGroupBy):
		v2BadRequest(c, "Invalid cost group_by.", nil)
	case err.Error() == "invalid car id":
		v2BadRequest(c, "Invalid car id.", nil)
	default:
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 cost analytics.", err.Error())
	}
}

// Locations godoc
//
// @Summary V2 location analytics
// @Description Returns objective usage metrics grouped by geofence or address, including drive starts, drive ends, charging, inferred parking, and estimated vampire drain.
// @Tags V2 Location Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param sort query string false "Sort mode" Enums(drive_start_count_desc, drive_end_count_desc, charging_session_count_desc, parking_duration_desc, charging_cost_desc)
// @Success 200 {object} V2LocationAnalyticsAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/locations [get]
func (h V2Handlers) Locations(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.locationBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 location service is not configured.", nil)
		return
	}
	response, carID, err := h.locationBuilder.BuildLocations(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("sort"))
	if err != nil {
		handleV2LocationError(c, err)
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

func handleV2LocationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
	case errors.Is(err, errV2InvalidLocationSort):
		v2BadRequest(c, "Invalid location sort.", nil)
	case err.Error() == "invalid car id":
		v2BadRequest(c, "Invalid car id.", nil)
	default:
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 location analytics.", err.Error())
	}
}

// Updates godoc
//
// @Summary V2 update analytics
// @Description Returns OTA update history statistics for a car in the selected period.
// @Tags V2 Update Analytics
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param compare query string false "Comparison mode (accepted for meta consistency; response has no comparison block)" Enums(none, previous_period, previous_year, lifetime_average)
// @Success 200 {object} V2UpdateAnalyticsAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/updates [get]
func (h V2Handlers) Updates(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.updateBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 update service is not configured.", nil)
		return
	}
	response, carID, err := h.updateBuilder.BuildUpdates(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2GenericError(c, err, "Unable to build V2 update analytics.")
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// Lifecycle godoc
//
// @Summary V2 lifetime cumulative analytics
// @Description Returns cumulative lifetime statistics for a car from the first recorded event up to as_of (defaults to now). Use as_of for historical snapshots.
// @Tags V2 Lifecycle
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param as_of query string false "Cutoff datetime in RFC3339 format. Defaults to now."
// @Success 200 {object} V2LifecycleAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/lifecycle [get]
func (h V2Handlers) Lifecycle(c *gin.Context) {
	if h.lifecycleBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 lifecycle service is not configured.", nil)
		return
	}
	var asOf time.Time
	if asOfStr := c.Query("as_of"); asOfStr != "" {
		parsed, err := time.Parse(time.RFC3339, asOfStr)
		if err != nil {
			v2BadRequest(c, "Invalid as_of parameter.", "as_of must be RFC3339 format, e.g. 2026-04-30T23:59:59+08:00")
			return
		}
		asOf = parsed.UTC()
	}
	response, err := h.lifecycleBuilder.BuildLifecycle(c.Request.Context(), c.Param("CarID"), asOf)
	if err != nil {
		handleV2GenericError(c, err, "Unable to build V2 lifecycle analytics.")
		return
	}
	carID, _ := strconv.ParseInt(c.Param("CarID"), 10, 64)
	meta := V2Meta{
		CarID:       carID,
		Period:      "lifetime",
		Timezone:    "UTC",
		GeneratedAt: h.now().UTC().Format(time.RFC3339),
	}
	v2JSON(c, http.StatusOK, response, meta)
}

// Timeline godoc
//
// @Summary V2 unified event timeline
// @Description Returns a cursor-paginated unified chronological timeline of drive, charging, and update events.
// @Tags V2 Lifecycle
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param type query string false "Comma-separated event types: drive,charging,update"
// @Param limit query int false "Max results per page" default(50)
// @Param before query string false "Return events before this RFC3339 timestamp (cursor, DESC order)"
// @Param after query string false "Return events after this RFC3339 timestamp (cursor, ASC order)"
// @Success 200 {object} V2TimelineAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/timeline [get]
func (h V2Handlers) Timeline(c *gin.Context) {
	if h.lifecycleBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 lifecycle service is not configured.", nil)
		return
	}
	eventTypes := parseEventTypes(c.Query("type"))
	limit := v2LimitFromQueryWithDefault(c.Query("limit"), 50, 200)

	var before, after *time.Time
	if beforeStr := c.Query("before"); beforeStr != "" {
		t, err := time.Parse(time.RFC3339, beforeStr)
		if err != nil {
			v2BadRequest(c, "Invalid before parameter.", "before must be RFC3339 format")
			return
		}
		before = &t
	}
	if afterStr := c.Query("after"); afterStr != "" {
		t, err := time.Parse(time.RFC3339, afterStr)
		if err != nil {
			v2BadRequest(c, "Invalid after parameter.", "after must be RFC3339 format")
			return
		}
		after = &t
	}

	response, err := h.lifecycleBuilder.BuildTimeline(c.Request.Context(), c.Param("CarID"), eventTypes, limit, before, after)
	if err != nil {
		handleV2GenericError(c, err, "Unable to build V2 timeline.")
		return
	}
	carID, _ := strconv.ParseInt(c.Param("CarID"), 10, 64)
	meta := V2Meta{
		CarID:       carID,
		Period:      "custom",
		Timezone:    "UTC",
		GeneratedAt: h.now().UTC().Format(time.RFC3339),
	}
	v2JSON(c, http.StatusOK, response, meta)
}

// Calendar godoc
//
// @Summary V2 calendar daily aggregation
// @Description Returns per-day aggregated statistics for calendar/heatmap views. All dates in the requested range are returned, with zeros for days without data.
// @Tags V2 Calendar
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start date (YYYY-MM-DD or RFC3339)"
// @Param end query string false "End date (YYYY-MM-DD or RFC3339)"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} V2CalendarAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/calendar [get]
func (h V2Handlers) Calendar(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.calendarBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 calendar service is not configured.", nil)
		return
	}
	response, carID, err := h.calendarBuilder.BuildCalendar(c.Request.Context(), c.Param("CarID"), timeRange)
	if err != nil {
		handleV2GenericError(c, err, "Unable to build V2 calendar.")
		return
	}
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// Reports godoc
//
// @Summary V2 period report
// @Description Returns a structured period report combining multiple analytics modules. Only sections with data are included.
// @Tags V2 Reports
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param include query string false "Comma-separated module list: summary,driving,charging,updates"
// @Success 200 {object} V2ReportAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/reports [get]
func (h V2Handlers) Reports(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.reportBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 report service is not configured.", nil)
		return
	}
	include := parseIncludeList(c.Query("include"))
	response, err := h.reportBuilder.BuildReport(c.Request.Context(), c.Param("CarID"), timeRange, include)
	if err != nil {
		handleV2GenericError(c, err, "Unable to build V2 report.")
		return
	}
	carID, _ := strconv.ParseInt(c.Param("CarID"), 10, 64)
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

// Insights godoc
//
// @Summary V2 objective insights
// @Description Returns factual, evidence-backed insights based on period comparison. No subjective evaluations.
// @Tags V2 Insights
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param category query string false "Filter by category: driving,charging,parking,battery,cost,lifecycle"
// @Param min_severity query string false "Minimum severity: info,warning" Enums(info, warning)
// @Success 200 {object} V2InsightAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/insights [get]
func (h V2Handlers) Insights(c *gin.Context) {
	_, timeRange, err := parseV2AnalyticsQuery(c, h.now())
	if err != nil {
		v2BadRequest(c, "Invalid analytics query.", err.Error())
		return
	}
	if h.insightBuilder == nil {
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "V2 insight service is not configured.", nil)
		return
	}
	response, err := h.insightBuilder.BuildInsights(c.Request.Context(), c.Param("CarID"), timeRange, c.Query("category"), c.Query("min_severity"))
	if err != nil {
		handleV2GenericError(c, err, "Unable to build V2 insights.")
		return
	}
	carID, _ := strconv.ParseInt(c.Param("CarID"), 10, 64)
	v2JSON(c, http.StatusOK, response, newV2Meta(carID, timeRange))
}

func handleV2GenericError(c *gin.Context, err error, msg string) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
	case err.Error() == "invalid car id":
		v2BadRequest(c, "Invalid car id.", nil)
	default:
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", msg, err.Error())
	}
}

func v2LimitFromQueryWithDefault(limitStr string, defaultVal, maxVal int) int {
	limit := v2LimitFromQuery(limitStr)
	if limit <= 0 {
		limit = defaultVal
	}
	if limit > maxVal {
		limit = maxVal
	}
	return limit
}
