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
}

type V2ChargingBuilder interface {
	BuildCharging(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2ChargingBuildOptions) (V2ChargingResponse, int64, error)
}

type V2ParkingBuilder interface {
	BuildParking(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2ParkingBuildOptions) (V2ParkingResponse, int64, error)
}

type V2BatteryBuilder interface {
	BuildBattery(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2BatteryResponse, int64, error)
	BuildBatteryTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2BatteryTimeseriesResponse, int64, error)
}

type V2CostBuilder interface {
	BuildCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2CostResponse, int64, error)
}

type V2UpdateBuilder interface {
	BuildUpdates(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2UpdateAnalyticsResponse, int64, error)
}

type V2LifecycleBuilder interface {
	BuildLifecycle(ctx context.Context, carIDParam string, asOf time.Time) (V2LifecycleResponse, error)
	BuildTimeline(ctx context.Context, carIDParam string, eventTypes []string, limit int, before, after *time.Time) (V2TimelineResponse, error)
}

// V2ChargingCurveBuilder builds aggregated DC curve responses.
type V2ChargingCurveBuilder interface {
	BuildCurve(ctx context.Context, carIDParam string, timeRange V2TimeRange, minSessions int) (V2ChargingCurveResponse, int64, error)
}

// V2EfficiencyBuilder builds efficiency analytics responses.
type V2EfficiencyBuilder interface {
	BuildEfficiency(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2EfficiencyBuildOptions) (V2EfficiencyResponse, int64, error)
}

// V2IdlePeriodsBuilder builds parking idle period responses.
type V2IdlePeriodsBuilder interface {
	BuildIdlePeriods(ctx context.Context, carIDParam string, timeRange V2TimeRange, opts V2IdlePeriodsBuildOptions) (V2IdlePeriodsResponse, int64, error)
}

// V2CapacityByMileageBuilder builds half-month capacity bucket responses.
type V2CapacityByMileageBuilder interface {
	BuildCapacityByMileage(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2CapacityByMileageResponse, int64, error)
}

// (V2EnvironmentalBuilder is declared alongside the environmental service.)

// @name V2Handlers
type V2Handlers struct {
	summaryBuilder       V2SummaryBuilder
	drivingBuilder       V2DrivingBuilder
	chargingBuilder      V2ChargingBuilder
	chargingCurveBuilder     V2ChargingCurveBuilder
	efficiencyBuilder        V2EfficiencyBuilder
	idlePeriodsBuilder       V2IdlePeriodsBuilder
	capacityByMileageBuilder V2CapacityByMileageBuilder
	environmentalBuilder     V2EnvironmentalBuilder
	odometerSeriesBuilder    V2OdometerSeriesBuilder
	summaryByPeriodBuilder   V2SummaryByPeriodBuilder
	placesBuilder            V2PlacesBuilder
	geofencesBuilder         V2GeofencesBuilder
	parkingBuilder       V2ParkingBuilder
	batteryBuilder       V2BatteryBuilder
	costBuilder          V2CostBuilder
	updateBuilder        V2UpdateBuilder
	lifecycleBuilder     V2LifecycleBuilder
	now                  func() time.Time
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
	var costRepository V2CostRepository
	var updateRepository V2UpdateRepository
	var lifecycleRepository V2LifecycleRepository
	if summaryRepository == nil && db != nil {
		repository := NewPostgresV2SummaryRepository(db)
		summaryRepository = repository
		drivingRepository = NewPostgresV2DrivingRepository(db)
		chargingRepository = NewPostgresV2ChargingRepository(db)
		parkingRepository = NewPostgresV2ParkingRepository(db)
		batteryRepository = NewPostgresV2BatteryRepository(db)
		costRepository = NewPostgresV2CostRepository(db)
		updateRepository = NewPostgresV2UpdateRepository(db)
		lifecycleRepository = NewPostgresV2LifecycleRepository(db)
	}
	drivingService := NewV2DrivingService(drivingRepository)
	chargingService := NewV2ChargingService(chargingRepository)
	handlers := NewV2Handlers(NewV2SummaryService(summaryRepository), drivingService, chargingService)
	handlers.parkingBuilder = NewV2ParkingService(parkingRepository)
	handlers.batteryBuilder = NewV2BatteryService(batteryRepository)
	handlers.costBuilder = NewV2CostService(costRepository)
	handlers.updateBuilder = NewV2UpdateService(updateRepository)
	handlers.lifecycleBuilder = NewV2LifecycleService(lifecycleRepository)
	if db != nil {
		handlers.chargingCurveBuilder = NewV2ChargingCurveService(NewPostgresV2ChargingCurveRepository(db))
		handlers.efficiencyBuilder = NewV2EfficiencyService(NewPostgresV2EfficiencyRepository(db))
		handlers.idlePeriodsBuilder = NewV2IdlePeriodsService(NewPostgresV2IdlePeriodsRepository(db))
		handlers.capacityByMileageBuilder = NewV2CapacityByMileageService(NewPostgresV2CapacityByMileageRepository(db))
		handlers.environmentalBuilder = NewV2EnvironmentalService(NewPostgresV2EnvironmentalRepository(db))
		handlers.odometerSeriesBuilder = NewV2OdometerSeriesService(db)
		handlers.summaryByPeriodBuilder = NewV2SummaryByPeriodService(db)
		handlers.placesBuilder = NewV2PlacesService(db)
		handlers.geofencesBuilder = NewV2GeofencesService(db)
	}

	v2 := api.Group("/v2")
	{
		v2.GET("/capabilities", handlers.Capabilities)
		// All car-scoped routes share a single car-validation middleware (one DB lookup per request).
		v2Cars := v2.Group("/cars/:CarID", v2CarValidationMiddleware())
		v2Cars.GET("/analytics/summary", handlers.Summary)
		v2Cars.GET("/analytics/driving", handlers.Driving)
		v2Cars.GET("/analytics/driving/timeseries", handlers.DrivingTimeseries)
		v2Cars.GET("/analytics/charging", handlers.Charging)
		v2Cars.GET("/analytics/charging/curve", handlers.ChargingCurve)
		v2Cars.GET("/analytics/efficiency", handlers.Efficiency)
		v2Cars.GET("/analytics/parking", handlers.Parking)
		v2Cars.GET("/parking/idle_periods", handlers.IdlePeriods)
		v2Cars.GET("/analytics/battery", handlers.Battery)
		v2Cars.GET("/analytics/battery/timeseries", handlers.BatteryTimeseries)
		v2Cars.GET("/battery/capacity_by_mileage", handlers.CapacityByMileage)
		v2Cars.GET("/analytics/environmental", handlers.Environmental)
		v2Cars.GET("/lifecycle/odometer_series", handlers.OdometerSeries)
		v2Cars.GET("/lifecycle/places", handlers.Places)
		v2Cars.GET("/summary/by_period", handlers.SummaryByPeriod)
		v2.GET("/geofences", handlers.Geofences)
		v2Cars.GET("/analytics/cost", handlers.Cost)
		v2Cars.GET("/updates", handlers.Updates)
		v2Cars.GET("/lifecycle", handlers.Lifecycle)
		v2Cars.GET("/timeline", handlers.Timeline)
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

// Capabilities godoc
//
// @Summary V2 API capabilities
// @Description Returns the V2 analytics API version, the per-domain feature flags (compare/timeseries/breakdown), and the allowed breakdown values per domain. Clients use this to decide which query parameters to send instead of probing each endpoint.
// @Tags v2
// @Produce json
// @Success 200 {object} V2CapabilitiesAPIResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/capabilities [get]
func (h V2Handlers) Capabilities(c *gin.Context) {
	meta := V2Meta{
		Unit:        defaultV2Unit(),
		GeneratedAt: time.Now().In(timeRangeLocation(V2TimeRange{Timezone: defaultV2Timezone()})).Format(time.RFC3339),
	}
	response := V2CapabilitiesResponse{
		Version: "v2",
		Domains: []V2CapabilitiesDomain{
			{Name: "summary", Path: "/v2/cars/{car_id}/analytics/summary", SupportsCompare: true},
			{Name: "driving", Path: "/v2/cars/{car_id}/analytics/driving", SupportsTimeseries: true},
			{Name: "charging", Path: "/v2/cars/{car_id}/analytics/charging", SupportsTimeseries: true, SupportsBreakdown: true},
			{Name: "parking", Path: "/v2/cars/{car_id}/analytics/parking", SupportsBreakdown: true},
			{Name: "battery", Path: "/v2/cars/{car_id}/analytics/battery", SupportsTimeseries: true},
			{Name: "cost", Path: "/v2/cars/{car_id}/analytics/cost", SupportsTimeseries: true},
			{Name: "updates", Path: "/v2/cars/{car_id}/updates"},
			{Name: "lifecycle", Path: "/v2/cars/{car_id}/lifecycle"},
			{Name: "timeline", Path: "/v2/cars/{car_id}/timeline"},
		},
		BreakdownOptions: map[string][]string{
			"charging": {"location", "charger_type"},
			"parking":  {"location", "state"},
		},
	}
	v2JSON(c, http.StatusOK, response, meta)
}

// Summary godoc
//
// @Summary V2 period summary analytics
// @Description Returns objective driving, charging, parking, battery, update, and charging-cost summary metrics for one car in a selected period.
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param compare query string false "Comparison mode" Enums(none, previous_period)
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
// @Description Returns objective driving statistics for one car.
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
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
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
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

func handleV2DrivingError(c *gin.Context, err error, timeRange V2TimeRange) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
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
// @Description Returns objective charging statistics for one car.
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param include query string false "Comma-separated extras: timeseries,breakdown" example("timeseries,breakdown")
// @Param group_by query string false "Timeseries grouping (used when include=timeseries)" Enums(day, week, month, year)
// @Param breakdown query string false "Breakdown dimension (used when include=breakdown)" Enums(location, charger_type)
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
	include := parseV2IncludeSet(c.Query("include"))
	opts := V2ChargingBuildOptions{
		IncludeTimeseries: include["timeseries"],
		IncludeBreakdown:  include["breakdown"],
		GroupBy:           c.Query("group_by"),
		BreakdownBy:       c.Query("breakdown"),
	}
	response, carID, err := h.chargingBuilder.BuildCharging(c.Request.Context(), c.Param("CarID"), timeRange, opts)
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
	case errors.Is(err, errV2InvalidDrivingGroupBy):
		v2BadRequest(c, "Invalid charging group_by.", nil)
	case errors.Is(err, errV2InvalidChargingBreakdown):
		v2BadRequest(c, "Invalid charging breakdown.", "breakdown must be one of: location, charger_type")
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
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param include query string false "Comma-separated extras: breakdown" example("breakdown")
// @Param breakdown query string false "Breakdown dimension (used when include=breakdown)" Enums(location, state)
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
	include := parseV2IncludeSet(c.Query("include"))
	opts := V2ParkingBuildOptions{
		IncludeBreakdown: include["breakdown"],
		BreakdownBy:      c.Query("breakdown"),
	}
	response, carID, err := h.parkingBuilder.BuildParking(c.Request.Context(), c.Param("CarID"), timeRange, opts)
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
	case errors.Is(err, errV2InvalidParkingBreakdown):
		v2BadRequest(c, "Invalid parking breakdown.", "breakdown must be one of: location, state")
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
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
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
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
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

func handleV2BatteryError(c *gin.Context, err error, timeRange V2TimeRange) {
	switch {
	case errors.Is(err, errV2CarNotFound):
		v2Error(c, http.StatusNotFound, "CAR_NOT_FOUND", "Car was not found.", nil)
	case errors.Is(err, errV2InvalidDrivingGroupBy):
		v2BadRequest(c, "Invalid battery group_by.", nil)
	case err.Error() == "invalid car id":
		v2BadRequest(c, "Invalid car id.", nil)
	default:
		v2Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to build V2 battery analytics.", err.Error())
	}
}

// Cost godoc
//
// @Summary V2 cost analytics
// @Description Returns objective charging-cost analytics. Current data scope includes charging_cost only and excludes insurance, maintenance, parking, depreciation, tire, and repair costs.
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
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

// Updates godoc
//
// @Summary V2 update analytics
// @Description Returns OTA update history statistics for a car in the selected period.
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} V2UpdateAnalyticsAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/updates [get]
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
// @Tags v2
// @Produce json
// @Param CarID path int true "Car ID" example(1)
// @Param as_of query string false "Cutoff datetime in RFC3339 format. Defaults to now."
// @Success 200 {object} V2LifecycleAPIResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v2/cars/{CarID}/lifecycle [get]
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
// @Tags v2
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
