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

type V2Handlers struct {
	summaryBuilder V2SummaryBuilder
	now            func() time.Time
}

func NewV2Handlers(summaryBuilder V2SummaryBuilder) V2Handlers {
	return V2Handlers{
		summaryBuilder: summaryBuilder,
		now:            time.Now,
	}
}

func RegisterV2Routes(api *gin.RouterGroup, summaryRepository V2SummaryRepository) {
	if summaryRepository == nil && db != nil {
		repository := NewPostgresV2SummaryRepository(db)
		summaryRepository = repository
	}
	handlers := NewV2Handlers(NewV2SummaryService(summaryRepository))

	v2 := api.Group("/v2")
	{
		v2.GET("", handlers.Info)
		v2.GET("/", handlers.Info)
		v2.GET("/cars/:CarID/analytics/summary", handlers.Summary)
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
