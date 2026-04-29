package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type apiCarContext struct {
	CarID            int
	UnitsLength      string
	UnitsTemperature string
	CarName          NullString
}

type chartRecord struct {
	Bucket string
	Value  *float64
}

type chargeCalendarDay struct {
	Date             string   `json:"date"`
	ChargeCount      int      `json:"charge_count"`
	DurationMin      int      `json:"duration_min"`
	EnergyAdded      float64  `json:"energy_added"`
	Cost             *float64 `json:"cost,omitempty"`
	FirstChargeStart *string  `json:"first_charge_start,omitempty"`
	LastChargeEnd    *string  `json:"last_charge_end,omitempty"`
	Day              int      `json:"day"`
	Weekday          int      `json:"weekday"`
	IsCurrentMonth   bool     `json:"is_current_month"`
	IsToday          bool     `json:"is_today"`
}

type chargeCalendarMonth struct {
	Year      int                 `json:"year"`
	Month     int                 `json:"month"`
	MonthName string              `json:"month_name"`
	StartDate string              `json:"start_date"`
	EndDate   string              `json:"end_date"`
	Days      []chargeCalendarDay `json:"days"`
}

type visitedBounds struct {
	MinLatitude  float64 `json:"min_latitude"`
	MaxLatitude  float64 `json:"max_latitude"`
	MinLongitude float64 `json:"min_longitude"`
	MaxLongitude float64 `json:"max_longitude"`
}

type visitedPoint struct {
	Time      string  `json:"time"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func loadAPICarContext(c *gin.Context, actionName string) (*apiCarContext, bool) {
	carID, err := parseCarID(c)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_car_id", "invalid CarID, expected positive integer", map[string]any{"car_id": c.Param("CarID")})
		return nil, false
	}
	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(carID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAPIError(c, http.StatusNotFound, "car_not_found", "car not found", map[string]any{"car_id": carID})
			return nil, false
		}
		if strings.Contains(err.Error(), "out of range for type smallint") {
			writeAPIError(c, http.StatusBadRequest, "invalid_car_id", "invalid CarID, expected value within TeslaMate car id range", map[string]any{"car_id": carID})
			return nil, false
		}
		writeAPIError(c, http.StatusInternalServerError, "metadata_error", "unable to load car metadata", map[string]any{"reason": err.Error(), "action": actionName})
		return nil, false
	}
	return &apiCarContext{CarID: carID, UnitsLength: unitsLength, UnitsTemperature: unitsTemperature, CarName: carName}, true
}

func parseAPIRangeWithDefault(c *gin.Context, defaultDays int) (string, string, error) {
	startRaw := c.Query("startDate")
	endRaw := c.Query("endDate")
	startUTC, endUTC, err := parseDateRangeValues(startRaw, endRaw, appUsersTimezone)
	if err != nil {
		return "", "", err
	}
	if defaultDays <= 0 {
		return startUTC, endUTC, nil
	}
	now := time.Now().In(appUsersTimezone).Truncate(time.Second)
	if endUTC == "" {
		endUTC = now.UTC().Format(dbTimestampFormat)
	}
	if startUTC == "" {
		endTime, err := time.Parse(dbTimestampFormat, endUTC)
		if err != nil {
			return "", "", err
		}
		startUTC = endTime.AddDate(0, 0, -defaultDays).Format(dbTimestampFormat)
	}
	return startUTC, endUTC, nil
}

func bucketDateExpr(bucket string, column string, tzParam int) string {
	return fmt.Sprintf("TO_CHAR(date_trunc('%s', timezone($%d, %s)), 'YYYY-MM-DD HH24:MI:SS')", bucket, tzParam, column)
}

func toChartTime(value string) string {
	if value == "" {
		return ""
	}
	if t, err := time.ParseInLocation(time.DateTime, value, appUsersTimezone); err == nil {
		return t.Format(time.RFC3339)
	}
	if t, err := time.Parse(dbTimestampFormat, value); err == nil {
		return t.In(appUsersTimezone).Format(time.RFC3339)
	}
	return value
}

func floatPtr(v float64) *float64 { return &v }

func toAnySlice[T any](items []T) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func TeslaMateAPICarsSummaryV2(c *gin.Context) {
	dr, err := parseSummaryRangeStrict(c)
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date", "invalid summary range", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsSummaryV2")
	if !ok {
		return
	}
	data, err := buildUnifiedSummary(ctx, dr)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load summary", map[string]any{"reason": err.Error()})
		return
	}
	writeV1Object(c, data, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"))
}

func TeslaMateAPICarsStatisticsV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsStatisticsV2")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	driveSummary, err := fetchDriveHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load drive summary", map[string]any{"reason": err.Error()})
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load charge summary", map[string]any{"reason": err.Error()})
		return
	}
	statistics, err := fetchStatisticsSummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, ctx.UnitsTemperature, driveSummary, chargeSummary)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load statistics", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIObjectResponse(ctx.CarID, buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature), startUTC, endUTC, map[string]any{"statistics": statistics}, nil))
}

func TeslaMateAPICarsInsightsV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsInsightsV2")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	summary, err := fetchInsightSummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load insights", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIObjectResponse(ctx.CarID, buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature), startUTC, endUTC, map[string]any{"insights": summary}, nil))
}

func TeslaMateAPICarsInsightEventsV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsInsightEventsV2")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	page, show, err := parsePaginationParams(c, 1, 100, 500)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_pagination", "invalid pagination parameters", map[string]any{"reason": err.Error()})
		return
	}
	types, err := parseInsightTypes(c.Query("types"))
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_types", "invalid insight types", map[string]any{"reason": err.Error()})
		return
	}
	allEvents, err := fetchInsightEvents(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, types, 1, 100000)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load insight events", map[string]any{"reason": err.Error()})
		return
	}
	total := len(allEvents)
	offset := (page - 1) * show
	if offset > total {
		offset = total
	}
	end := offset + show
	if end > total {
		end = total
	}
	items := toAnySlice(allEvents[offset:end])
	writeAPISuccess(c, newAPIListResponse(ctx.CarID, startUTC, endUTC, items, page, show, total, map[string]any{"types": types}))
}

func TeslaMateAPICarsAnalyticsActivityV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsAnalyticsActivityV2")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	driveSummary, err := fetchDriveHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load drive summary", map[string]any{"reason": err.Error()})
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load charge summary", map[string]any{"reason": err.Error()})
		return
	}
	parkingSummary, err := fetchParkingHistorySummary(ctx.CarID, startUTC, endUTC, nil)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load parking summary", map[string]any{"reason": err.Error()})
		return
	}
	analysis, err := fetchAnalysisSummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, driveSummary, chargeSummary, parkingSummary)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load activity analytics", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIObjectResponse(ctx.CarID, buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature), startUTC, endUTC, map[string]any{"activity": analysis}, nil))
}

func TeslaMateAPICarsAnalyticsRegenerationV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsAnalyticsRegenerationV2")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	driveSummary, err := fetchDriveHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load drive summary", map[string]any{"reason": err.Error()})
		return
	}
	regeneration, err := fetchRegenerationSummary(ctx.CarID, startUTC, endUTC, driveSummary, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load regeneration analytics", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIObjectResponse(ctx.CarID, buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature), startUTC, endUTC, map[string]any{"regeneration": regeneration}, map[string]any{"estimated": regeneration.MetricsEstimated}))
}

func TeslaMateAPICarsTimelineV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsTimelineV2")
	if !ok {
		return
	}
	page, show, err := parsePaginationParams(c, 1, 100, 500)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_pagination", "invalid pagination parameters", map[string]any{"reason": err.Error()})
		return
	}
	_, order, err := parseSortOrder(c, map[string]string{"startDate": "start_date", "type": "type"}, "startDate")
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_sort", "invalid timeline sort", map[string]any{"reason": err.Error()})
		return
	}
	startUTC, endUTC, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	items, total, err := fetchTimelineEvents(ctx.CarID, startUTC, endUTC, page, show, order)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load timeline", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIListResponse(ctx.CarID, startUTC, endUTC, toAnySlice(items), page, show, total, map[string]any{"order": order}))
}

func TeslaMateAPICarsDriveCalendarV2(c *gin.Context) {
	if _, _, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone); err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsDriveCalendarV2")
	if !ok {
		return
	}
	year, err := parseSummaryPositiveIntParam(c.Query("year"), time.Now().In(appUsersTimezone).Year(), 2012, 2100)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_calendar", "invalid year parameter", map[string]any{"reason": err.Error()})
		return
	}
	month, err := parseSummaryPositiveIntParam(c.Query("month"), int(time.Now().In(appUsersTimezone).Month()), 1, 12)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_calendar", "invalid month parameter", map[string]any{"reason": err.Error()})
		return
	}
	calendar, err := fetchDriveCalendarMonth(ctx.CarID, year, month, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load drive calendar", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIObjectResponse(ctx.CarID, buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature), calendar.StartDate, calendar.EndDate, map[string]any{"calendar": calendar}, nil))
}

func TeslaMateAPICarsChargeCalendarV2(c *gin.Context) {
	if _, _, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone); err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsChargeCalendarV2")
	if !ok {
		return
	}
	year, err := parseSummaryPositiveIntParam(c.Query("year"), time.Now().In(appUsersTimezone).Year(), 2012, 2100)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_calendar", "invalid year parameter", map[string]any{"reason": err.Error()})
		return
	}
	month, err := parseSummaryPositiveIntParam(c.Query("month"), int(time.Now().In(appUsersTimezone).Month()), 1, 12)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_calendar", "invalid month parameter", map[string]any{"reason": err.Error()})
		return
	}
	calendar, err := fetchChargeCalendarMonth(ctx.CarID, year, month)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load charge calendar", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIObjectResponse(ctx.CarID, buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature), calendar.StartDate, calendar.EndDate, map[string]any{"calendar": calendar}, nil))
}

func TeslaMateAPICarsMapVisitedV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsMapVisitedV2")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 90)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	points, bounds, truncated, err := fetchVisitedMap(ctx.CarID, startUTC, endUTC, 5000)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load visited map", map[string]any{"reason": err.Error()})
		return
	}
	meta := map[string]any{"truncated": truncated, "point_limit": 5000}
	writeAPISuccess(c, newAPIObjectResponse(ctx.CarID, buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature), startUTC, endUTC, map[string]any{"points": points, "bounds": bounds, "clusters": []any{}}, meta))
}

func TeslaMateAPICarsDriveDetailsV2(c *gin.Context) {
	if _, _, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone); err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsDriveDetailsV2")
	if !ok {
		return
	}
	driveID, err := strconv.Atoi(strings.TrimSpace(c.Param("DriveID")))
	if err != nil || driveID <= 0 {
		writeAPIError(c, http.StatusBadRequest, "invalid_drive_id", "invalid DriveID, expected positive integer", map[string]any{"drive_id": c.Param("DriveID")})
		return
	}
	details, err := fetchDriveDetailsPayload(ctx.CarID, driveID, ctx.UnitsLength, ctx.UnitsTemperature)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAPIError(c, http.StatusNotFound, "drive_not_found", "drive not found", map[string]any{"drive_id": driveID, "car_id": ctx.CarID})
			return
		}
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load drive details", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIObjectResponse(ctx.CarID, buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature), "", "", details, nil))
}

func TeslaMateAPICarsChargeDetailsV2(c *gin.Context) {
	if _, _, err := parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone); err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsChargeDetailsV2")
	if !ok {
		return
	}
	chargeID, err := strconv.Atoi(strings.TrimSpace(c.Param("ChargeID")))
	if err != nil || chargeID <= 0 {
		writeAPIError(c, http.StatusBadRequest, "invalid_charge_id", "invalid ChargeID, expected positive integer", map[string]any{"charge_id": c.Param("ChargeID")})
		return
	}
	details, err := fetchChargeDetailsPayload(ctx.CarID, chargeID, ctx.UnitsLength, ctx.UnitsTemperature)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAPIError(c, http.StatusNotFound, "charge_not_found", "charge not found", map[string]any{"charge_id": chargeID, "car_id": ctx.CarID})
			return
		}
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load charge details", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIObjectResponse(ctx.CarID, buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature), "", "", details, nil))
}

func TeslaMateAPICarsChartsOverviewV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsChartsOverviewV2")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 30)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	series, meta, err := fetchOverviewChartSeries(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, ctx.UnitsTemperature)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load overview charts", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, "day", series, meta))
}

func TeslaMateAPICarsDriveDistanceChartV2(c *gin.Context) {
	handleDriveAggregateChart(c, "distance", distanceUnitFromContext, fetchDriveDistanceSeries)
}
func TeslaMateAPICarsDriveEnergyChartV2(c *gin.Context) {
	handleDriveAggregateChart(c, "energy", func(u string) string { return "kWh" }, fetchDriveEnergySeries)
}
func TeslaMateAPICarsDriveEfficiencyChartV2(c *gin.Context) {
	handleDriveAggregateChart(c, "efficiency", consumptionUnitFromContext, fetchDriveEfficiencySeries)
}
func TeslaMateAPICarsDriveSpeedChartV2(c *gin.Context)       { handleDriveSpeedChart(c) }
func TeslaMateAPICarsDriveTemperatureChartV2(c *gin.Context) { handleDriveTemperatureChart(c) }
func TeslaMateAPICarsChargeEnergyChartV2(c *gin.Context) {
	handleChargeAggregateChart(c, "energy", "kWh", fetchChargeEnergySeries)
}
func TeslaMateAPICarsChargeCostChartV2(c *gin.Context) {
	handleChargeAggregateChart(c, "cost", "currency", fetchChargeCostSeries)
}
func TeslaMateAPICarsChargeEfficiencyChartV2(c *gin.Context) {
	handleChargeAggregateChart(c, "efficiency", "ratio", fetchChargeEfficiencySeries)
}
func TeslaMateAPICarsChargePowerChartV2(c *gin.Context)    { handleChargePowerChart(c) }
func TeslaMateAPICarsChargeLocationChartV2(c *gin.Context) { handleChargeLocationChart(c) }
func TeslaMateAPICarsChargeSOCChartV2(c *gin.Context)      { handleChargeSOCChart(c) }
func TeslaMateAPICarsBatteryRangeChartV2(c *gin.Context)   { handleBatteryRangeChart(c) }
func TeslaMateAPICarsBatteryHealthChartV2(c *gin.Context)  { handleBatteryHealthChart(c) }
func TeslaMateAPICarsStateDurationChartV2(c *gin.Context)  { handleStateDurationChart(c) }
func TeslaMateAPICarsVampireDrainChartV2(c *gin.Context)   { handleVampireDrainChart(c) }
func TeslaMateAPICarsMileageChartV2(c *gin.Context)        { handleMileageChart(c) }

func handleDriveAggregateChart(c *gin.Context, name string, unitFn func(string) string, fetcher func(int, string, string, string, string) ([]APIChartPoint, error)) {
	ctx, ok := loadAPICarContext(c, "drive_chart_"+name)
	if !ok {
		return
	}
	bucket, err := parseBucketParam(c, "month")
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_bucket", "invalid bucket", map[string]any{"reason": err.Error()})
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	points, err := fetcher(ctx.CarID, startUTC, endUTC, bucket, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load drive chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, bucket, []APIChartSeries{{Name: name, Unit: unitFn(ctx.UnitsLength), Points: points}}, nil))
}

func handleChargeAggregateChart(c *gin.Context, name, unit string, fetcher func(int, string, string, string) ([]APIChartPoint, error)) {
	ctx, ok := loadAPICarContext(c, "charge_chart_"+name)
	if !ok {
		return
	}
	bucket, err := parseBucketParam(c, "month")
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_bucket", "invalid bucket", map[string]any{"reason": err.Error()})
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	points, err := fetcher(ctx.CarID, startUTC, endUTC, bucket)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load charge chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, bucket, []APIChartSeries{{Name: name, Unit: unit, Points: points}}, nil))
}

func handleDriveTemperatureChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "drive_temperature_chart")
	if !ok {
		return
	}
	bucket, err := parseBucketParam(c, "month")
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_bucket", "invalid bucket", map[string]any{"reason": err.Error()})
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	tempPoints, consumptionPoints, err := fetchDriveTemperatureSeries(ctx.CarID, startUTC, endUTC, bucket, ctx.UnitsLength, ctx.UnitsTemperature)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load drive temperature chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, bucket, []APIChartSeries{
		{Name: "outside_temperature", Unit: temperatureUnit(ctx.UnitsTemperature), Points: tempPoints},
		{Name: "consumption", Unit: consumptionUnit(ctx.UnitsLength), Points: consumptionPoints},
	}, nil))
}

func handleDriveSpeedChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "drive_speed_chart")
	if !ok {
		return
	}
	bucket, err := parseBucketParam(c, "month")
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_bucket", "invalid bucket", map[string]any{"reason": err.Error()})
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	avgPoints, maxPoints, err := fetchDriveSpeedSeries(ctx.CarID, startUTC, endUTC, bucket, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load drive speed chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, bucket, []APIChartSeries{
		{Name: "average_speed", Unit: buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature).Speed, Points: avgPoints},
		{Name: "max_speed", Unit: buildAPIUnits(ctx.UnitsLength, ctx.UnitsTemperature).Speed, Points: maxPoints},
	}, nil))
}

func handleChargePowerChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "charge_power_chart")
	if !ok {
		return
	}
	bucket, err := parseBucketParam(c, "month")
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_bucket", "invalid bucket", map[string]any{"reason": err.Error()})
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	avgPoints, maxPoints, err := fetchChargePowerSeries(ctx.CarID, startUTC, endUTC, bucket)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load charge power chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, bucket, []APIChartSeries{
		{Name: "average_power", Unit: "kW", Points: avgPoints},
		{Name: "max_power", Unit: "kW", Points: maxPoints},
	}, nil))
}

func handleChargeLocationChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "charge_location_chart")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	limit, err := parseSeriesLimitParam(c, "show", 20, 100)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_show", "invalid show parameter", map[string]any{"reason": err.Error()})
		return
	}
	points, err := fetchChargeLocationSeries(ctx.CarID, startUTC, endUTC, limit)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load charge location chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, "", []APIChartSeries{{Name: "location_energy", Unit: "kWh", Points: points}}, map[string]any{"show": limit}))
}

func handleChargeSOCChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "charge_soc_chart")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	startPoints, endPoints, err := fetchChargeSOCDistribution(ctx.CarID, startUTC, endUTC)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load charge soc chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, "", []APIChartSeries{{Name: "start_soc", Unit: "%", Points: startPoints}, {Name: "end_soc", Unit: "%", Points: endPoints}}, nil))
}

func handleBatteryRangeChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "battery_range_chart")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	series, meta, err := fetchBatteryRangeChartSeries(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load battery range chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, "day", series, meta))
}

func handleBatteryHealthChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "battery_health_chart")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	series, meta, err := fetchBatteryHealthChartSeries(ctx.CarID, startUTC, endUTC)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load battery health chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, "day", series, meta))
}

func handleStateDurationChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "state_duration_chart")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 180)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	breakdown, _, _, err := fetchStateBreakdown(ctx.CarID, startUTC, endUTC)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load state duration chart", map[string]any{"reason": err.Error()})
		return
	}
	points := make([]APIChartPoint, 0, len(breakdown))
	for _, item := range breakdown {
		value := float64(item.DurationMin)
		points = append(points, APIChartPoint{Label: item.State, Value: &value})
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, "", []APIChartSeries{{Name: "duration", Unit: "minutes", Points: points}}, nil))
}

func handleVampireDrainChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "vampire_drain_chart")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 180)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	series, meta := fetchVampireDrainPlaceholder()
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, "session", series, meta))
}

func handleMileageChart(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "mileage_chart")
	if !ok {
		return
	}
	startUTC, endUTC, err := parseAPIRangeWithDefault(c, 365)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_date", "invalid startDate or endDate, expected RFC3339 or local date format", map[string]any{"reason": err.Error()})
		return
	}
	points, err := fetchMileageSeries(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "query_error", "unable to load mileage chart", map[string]any{"reason": err.Error()})
		return
	}
	writeAPISuccess(c, newAPIChartResponse(ctx.CarID, startUTC, endUTC, "month", []APIChartSeries{{Name: "odometer", Unit: distanceUnit(ctx.UnitsLength), Points: points}}, nil))
}

func buildEfficiencySummary(driveSummary *DriveHistorySummary, chargeSummary *ChargeHistorySummary) map[string]any {
	result := map[string]any{}
	if driveSummary != nil {
		result["drive_average_consumption"] = driveSummary.AverageConsumption
		result["drive_best_consumption"] = driveSummary.BestConsumption
		result["drive_worst_consumption"] = driveSummary.WorstConsumption
	}
	if chargeSummary != nil {
		result["charging_efficiency"] = chargeSummary.ChargingEfficiency
	}
	return result
}

func buildCostSummary(chargeSummary *ChargeHistorySummary) map[string]any {
	if chargeSummary == nil {
		return map[string]any{}
	}
	return map[string]any{
		"total_cost":           chargeSummary.TotalCost,
		"average_cost":         chargeSummary.AverageCost,
		"highest_cost":         chargeSummary.HighestCost,
		"average_cost_per_kwh": chargeSummary.AverageCostPerKwh,
		"cost_per_distance":    chargeSummary.CostPer100Distance,
	}
}

func fetchLatestOdometer(carID int, unitsLength string) (*float64, error) {
	var odometer sql.NullFloat64
	if err := db.QueryRow(`SELECT odometer FROM positions WHERE car_id = $1 AND odometer IS NOT NULL ORDER BY date DESC LIMIT 1`, carID).Scan(&odometer); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if !odometer.Valid {
		return nil, nil
	}
	value := odometer.Float64
	if strings.EqualFold(unitsLength, "mi") {
		value = kilometersToMiles(value)
	}
	return &value, nil
}

func distanceUnit(unitsLength string) string {
	if strings.EqualFold(unitsLength, "mi") {
		return "mi"
	}
	return "km"
}
func distanceUnitFromContext(unitsLength string) string { return distanceUnit(unitsLength) }
func temperatureUnit(unitsTemperature string) string {
	if strings.EqualFold(unitsTemperature, "f") {
		return "F"
	}
	return "C"
}
func consumptionUnit(unitsLength string) string {
	if strings.EqualFold(unitsLength, "mi") {
		return "Wh/mi"
	}
	return "Wh/km"
}
func consumptionUnitFromContext(unitsLength string) string { return consumptionUnit(unitsLength) }

func fetchOverviewChartSeries(carID int, startUTC, endUTC, unitsLength, unitsTemperature string) ([]APIChartSeries, map[string]any, error) {
	driveDistance, err := fetchDriveDistanceSeries(carID, startUTC, endUTC, "day", unitsLength)
	if err != nil {
		return nil, nil, err
	}
	chargeEnergy, err := fetchChargeEnergySeries(carID, startUTC, endUTC, "day")
	if err != nil {
		return nil, nil, err
	}
	efficiency, err := fetchDriveEfficiencySeries(carID, startUTC, endUTC, "day", unitsLength)
	if err != nil {
		return nil, nil, err
	}
	stateBreakdown, _, _, err := fetchStateBreakdown(carID, startUTC, endUTC)
	if err != nil {
		return nil, nil, err
	}
	statePoints := make([]APIChartPoint, 0, len(stateBreakdown))
	for _, item := range stateBreakdown {
		value := float64(item.DurationMin)
		statePoints = append(statePoints, APIChartPoint{Label: item.State, Value: &value})
	}
	return []APIChartSeries{
		{Name: "drive_distance", Unit: distanceUnit(unitsLength), Points: driveDistance},
		{Name: "charge_energy", Unit: "kWh", Points: chargeEnergy},
		{Name: "drive_efficiency", Unit: consumptionUnit(unitsLength), Points: efficiency},
		{Name: "state_duration", Unit: "minutes", Points: statePoints},
	}, map[string]any{"temperature_unit": temperatureUnit(unitsTemperature)}, nil
}

func fetchAggregateChartPoints(query string, args []any) ([]APIChartPoint, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	points := make([]APIChartPoint, 0)
	for rows.Next() {
		var bucket string
		var value sql.NullFloat64
		if err := rows.Scan(&bucket, &value); err != nil {
			return nil, err
		}
		points = append(points, APIChartPoint{Time: toChartTime(bucket), Value: floatPointer(value)})
	}
	return points, rows.Err()
}

func fetchDriveDistanceSeries(carID int, startUTC, endUTC, bucket, unitsLength string) ([]APIChartPoint, error) {
	tzParam := 4
	query := fmt.Sprintf(`
		SELECT %s AS bucket, COALESCE(SUM(GREATEST(COALESCE(drives.distance, 0), 0)), 0)::float8 AS value
		FROM drives
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
		GROUP BY bucket
		ORDER BY bucket DESC`, bucketDateExpr(bucket, "drives.start_date", tzParam))
	points, err := fetchAggregateChartPoints(query, []any{carID, startUTC, endUTC, appUsersTimezone.String()})
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(unitsLength, "mi") {
		for i := range points {
			if points[i].Value != nil {
				v := kilometersToMiles(*points[i].Value)
				points[i].Value = &v
			}
		}
	}
	return points, nil
}

func fetchDriveEnergySeries(carID int, startUTC, endUTC, bucket, unitsLength string) ([]APIChartPoint, error) {
	_ = unitsLength
	tzParam := 4
	query := fmt.Sprintf(`
		SELECT %s AS bucket,
			COALESCE(SUM(CASE WHEN (drives.start_rated_range_km - drives.end_rated_range_km) > 0 THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency ELSE 0 END), 0)::float8 AS value
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
		GROUP BY bucket
		ORDER BY bucket DESC`, bucketDateExpr(bucket, "drives.start_date", tzParam))
	return fetchAggregateChartPoints(query, []any{carID, startUTC, endUTC, appUsersTimezone.String()})
}

func fetchDriveEfficiencySeries(carID int, startUTC, endUTC, bucket, unitsLength string) ([]APIChartPoint, error) {
	tzParam := 4
	query := fmt.Sprintf(`
		SELECT %s AS bucket,
			CASE WHEN SUM(GREATEST(COALESCE(drives.distance, 0), 0)) > 0 THEN
				SUM(CASE WHEN (drives.start_rated_range_km - drives.end_rated_range_km) > 0 THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency ELSE 0 END)
				/ SUM(GREATEST(COALESCE(drives.distance, 0), 0)) * 1000.0
			ELSE NULL END::float8 AS value
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
		GROUP BY bucket
		ORDER BY bucket DESC`, bucketDateExpr(bucket, "drives.start_date", tzParam))
	points, err := fetchAggregateChartPoints(query, []any{carID, startUTC, endUTC, appUsersTimezone.String()})
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(unitsLength, "mi") {
		for i := range points {
			if points[i].Value != nil {
				v := whPerKmToWhPerMi(*points[i].Value)
				points[i].Value = &v
			}
		}
	}
	return points, nil
}

func fetchDriveTemperatureSeries(carID int, startUTC, endUTC, bucket, unitsLength, unitsTemperature string) ([]APIChartPoint, []APIChartPoint, error) {
	tzParam := 4
	query := fmt.Sprintf(`
		SELECT %s AS bucket,
			AVG(drives.outside_temp_avg)::float8 AS avg_temp,
			CASE WHEN SUM(GREATEST(COALESCE(drives.distance, 0), 0)) > 0 THEN
				SUM(CASE WHEN (drives.start_rated_range_km - drives.end_rated_range_km) > 0 THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency ELSE 0 END)
				/ SUM(GREATEST(COALESCE(drives.distance, 0), 0)) * 1000.0
			ELSE NULL END::float8 AS avg_consumption
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
		GROUP BY bucket
		ORDER BY bucket DESC`, bucketDateExpr(bucket, "drives.start_date", tzParam))
	rows, err := db.Query(query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	temps := []APIChartPoint{}
	cons := []APIChartPoint{}
	for rows.Next() {
		var bucketValue string
		var temp sql.NullFloat64
		var consumption sql.NullFloat64
		if err := rows.Scan(&bucketValue, &temp, &consumption); err != nil {
			return nil, nil, err
		}
		if temp.Valid {
			val := temp.Float64
			if strings.EqualFold(unitsTemperature, "f") {
				val = celsiusToFahrenheit(val)
			}
			temps = append(temps, APIChartPoint{Time: toChartTime(bucketValue), Value: &val})
		} else {
			temps = append(temps, APIChartPoint{Time: toChartTime(bucketValue)})
		}
		if consumption.Valid {
			val := consumption.Float64
			if strings.EqualFold(unitsLength, "mi") {
				val = whPerKmToWhPerMi(val)
			}
			cons = append(cons, APIChartPoint{Time: toChartTime(bucketValue), Value: &val})
		} else {
			cons = append(cons, APIChartPoint{Time: toChartTime(bucketValue)})
		}
	}
	return temps, cons, rows.Err()
}

func fetchDriveSpeedSeries(carID int, startUTC, endUTC, bucket, unitsLength string) ([]APIChartPoint, []APIChartPoint, error) {
	tzParam := 4
	query := fmt.Sprintf(`
		SELECT %s AS bucket,
			CASE WHEN SUM(GREATEST(COALESCE(drives.duration_min, 0), 0)) > 0 THEN
				SUM(GREATEST(COALESCE(drives.distance, 0), 0)) / (SUM(GREATEST(COALESCE(drives.duration_min, 0), 0))::float8 / 60.0)
			ELSE NULL END::float8 AS avg_speed,
			MAX(NULLIF(drives.speed_max, 0))::float8 AS max_speed
		FROM drives
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
		GROUP BY bucket
		ORDER BY bucket DESC`, bucketDateExpr(bucket, "drives.start_date", tzParam))
	rows, err := db.Query(query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	avgPoints := []APIChartPoint{}
	maxPoints := []APIChartPoint{}
	for rows.Next() {
		var bucketValue string
		var avgSpeed sql.NullFloat64
		var maxSpeed sql.NullFloat64
		if err := rows.Scan(&bucketValue, &avgSpeed, &maxSpeed); err != nil {
			return nil, nil, err
		}
		if strings.EqualFold(unitsLength, "mi") {
			avgSpeed = kmhToMphNull(avgSpeed)
			if maxSpeed.Valid {
				maxSpeed.Float64 = kilometersToMiles(maxSpeed.Float64)
			}
		}
		avgPoints = append(avgPoints, APIChartPoint{Time: toChartTime(bucketValue), Value: floatPointer(avgSpeed)})
		maxPoints = append(maxPoints, APIChartPoint{Time: toChartTime(bucketValue), Value: floatPointer(maxSpeed)})
	}
	return avgPoints, maxPoints, rows.Err()
}

func fetchChargeEnergySeries(carID int, startUTC, endUTC, bucket string) ([]APIChartPoint, error) {
	tzParam := 4
	query := fmt.Sprintf(`
		SELECT %s AS bucket, COALESCE(SUM(GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0)), 0)::float8 AS value
		FROM charging_processes
		WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date < $3
		GROUP BY bucket
		ORDER BY bucket DESC`, bucketDateExpr(bucket, "charging_processes.start_date", tzParam))
	return fetchAggregateChartPoints(query, []any{carID, startUTC, endUTC, appUsersTimezone.String()})
}

func fetchChargeCostSeries(carID int, startUTC, endUTC, bucket string) ([]APIChartPoint, error) {
	tzParam := 4
	query := fmt.Sprintf(`
		SELECT %s AS bucket, COALESCE(SUM(CASE WHEN charging_processes.cost > 0 THEN charging_processes.cost ELSE 0 END), 0)::float8 AS value
		FROM charging_processes
		WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date < $3
		GROUP BY bucket
		ORDER BY bucket DESC`, bucketDateExpr(bucket, "charging_processes.start_date", tzParam))
	return fetchAggregateChartPoints(query, []any{carID, startUTC, endUTC, appUsersTimezone.String()})
}

func fetchChargeEfficiencySeries(carID int, startUTC, endUTC, bucket string) ([]APIChartPoint, error) {
	tzParam := 4
	query := fmt.Sprintf(`
		SELECT %s AS bucket,
			CASE WHEN SUM(GREATEST(COALESCE(charging_processes.charge_energy_used, charging_processes.charge_energy_added, 0), 0)) > 0 THEN
				SUM(GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0)) / SUM(GREATEST(COALESCE(charging_processes.charge_energy_used, charging_processes.charge_energy_added, 0), 0))
			ELSE NULL END::float8 AS value
		FROM charging_processes
		WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date < $3
		GROUP BY bucket
		ORDER BY bucket DESC`, bucketDateExpr(bucket, "charging_processes.start_date", tzParam))
	return fetchAggregateChartPoints(query, []any{carID, startUTC, endUTC, appUsersTimezone.String()})
}

func fetchChargePowerSeries(carID int, startUTC, endUTC, bucket string) ([]APIChartPoint, []APIChartPoint, error) {
	tzParam := 4
	query := fmt.Sprintf(`
		SELECT %s AS bucket,
			AVG(NULLIF(charges.charger_power, 0))::float8 AS avg_power,
			MAX(NULLIF(charges.charger_power, 0))::float8 AS max_power
		FROM charging_processes
		LEFT JOIN charges ON charges.charging_process_id = charging_processes.id
		WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date < $3
		GROUP BY bucket
		ORDER BY bucket DESC`, bucketDateExpr(bucket, "charging_processes.start_date", tzParam))
	rows, err := db.Query(query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	avgPoints := []APIChartPoint{}
	maxPoints := []APIChartPoint{}
	for rows.Next() {
		var bucketValue string
		var avgValue sql.NullFloat64
		var maxValue sql.NullFloat64
		if err := rows.Scan(&bucketValue, &avgValue, &maxValue); err != nil {
			return nil, nil, err
		}
		avgPoints = append(avgPoints, APIChartPoint{Time: toChartTime(bucketValue), Value: floatPointer(avgValue)})
		maxPoints = append(maxPoints, APIChartPoint{Time: toChartTime(bucketValue), Value: floatPointer(maxValue)})
	}
	return avgPoints, maxPoints, rows.Err()
}

func fetchChargeLocationSeries(carID int, startUTC, endUTC string, limit int) ([]APIChartPoint, error) {
	items, err := fetchDashboardChargeLocations(carID, startUTC, endUTC, limit)
	if err != nil {
		return nil, err
	}
	points := make([]APIChartPoint, 0, len(items))
	for _, item := range items {
		value := item.Value
		points = append(points, APIChartPoint{Label: item.Label, Value: &value})
	}
	return points, nil
}

func fetchChargeSOCDistribution(carID int, startUTC, endUTC string) ([]APIChartPoint, []APIChartPoint, error) {
	query := `
		SELECT CONCAT(bucket, '-', LEAST(bucket + 9, 100)) AS label,
			SUM(start_count)::int AS start_count,
			SUM(end_count)::int AS end_count
		FROM (
			SELECT (FLOOR(GREATEST(COALESCE(start_battery_level, 0), 0) / 10) * 10)::int AS bucket, 1 AS start_count, 0 AS end_count
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
			UNION ALL
			SELECT (FLOOR(GREATEST(COALESCE(end_battery_level, 0), 0) / 10) * 10)::int AS bucket, 0 AS start_count, 1 AS end_count
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
		) buckets
		GROUP BY label, bucket
		ORDER BY bucket DESC`
	rows, err := db.Query(query, carID, startUTC, endUTC)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	startPoints := []APIChartPoint{}
	endPoints := []APIChartPoint{}
	for rows.Next() {
		var label string
		var startCount int
		var endCount int
		if err := rows.Scan(&label, &startCount, &endCount); err != nil {
			return nil, nil, err
		}
		sc := float64(startCount)
		ec := float64(endCount)
		startPoints = append(startPoints, APIChartPoint{Label: label, Value: &sc})
		endPoints = append(endPoints, APIChartPoint{Label: label, Value: &ec})
	}
	return startPoints, endPoints, rows.Err()
}

func fetchBatteryRangeChartSeries(carID int, startUTC, endUTC, unitsLength string) ([]APIChartSeries, map[string]any, error) {
	query := `
		WITH latest_daily AS (
			SELECT DISTINCT ON (timezone($4, positions.date)::date)
				timezone($4, positions.date)::date AS local_day,
				positions.date,
				positions.odometer,
				positions.rated_battery_range_km,
				positions.ideal_battery_range_km,
				positions.usable_battery_level,
				cars.efficiency
			FROM positions
			LEFT JOIN cars ON cars.id = positions.car_id
			WHERE positions.car_id = $1
				AND positions.date >= $2
				AND positions.date < $3
				AND positions.rated_battery_range_km IS NOT NULL
				AND positions.ideal_battery_range_km IS NOT NULL
			ORDER BY timezone($4, positions.date)::date, positions.date DESC
		)
		SELECT TO_CHAR(local_day::timestamp, 'YYYY-MM-DD HH24:MI:SS') AS bucket,
			odometer,
			rated_battery_range_km,
			ideal_battery_range_km,
			CASE WHEN usable_battery_level > 0 THEN rated_battery_range_km * efficiency / usable_battery_level * 100.0 ELSE NULL END AS usable_battery_estimate
		FROM latest_daily
		ORDER BY local_day DESC`
	rows, err := db.Query(query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	odometerSeries := APIChartSeries{Name: "odometer", Unit: distanceUnit(unitsLength), Points: []APIChartPoint{}}
	ratedSeries := APIChartSeries{Name: "rated_range", Unit: distanceUnit(unitsLength), Points: []APIChartPoint{}}
	idealSeries := APIChartSeries{Name: "ideal_range", Unit: distanceUnit(unitsLength), Points: []APIChartPoint{}}
	usableSeries := APIChartSeries{Name: "usable_battery_estimate", Unit: "kWh", Points: []APIChartPoint{}}
	maxRated := 0.0
	degradationSeries := APIChartSeries{Name: "degradation_estimate", Unit: "%", Points: []APIChartPoint{}}
	for rows.Next() {
		var bucket string
		var odometer sql.NullFloat64
		var rated sql.NullFloat64
		var ideal sql.NullFloat64
		var usable sql.NullFloat64
		if err := rows.Scan(&bucket, &odometer, &rated, &ideal, &usable); err != nil {
			return nil, nil, err
		}
		when := toChartTime(bucket)
		if rated.Valid && rated.Float64 > maxRated {
			maxRated = rated.Float64
		}
		if odometer.Valid {
			value := odometer.Float64
			if strings.EqualFold(unitsLength, "mi") {
				value = kilometersToMiles(value)
			}
			odometerSeries.Points = append(odometerSeries.Points, APIChartPoint{Time: when, Value: &value})
		}
		if rated.Valid {
			value := rated.Float64
			if strings.EqualFold(unitsLength, "mi") {
				value = kilometersToMiles(value)
			}
			ratedSeries.Points = append(ratedSeries.Points, APIChartPoint{Time: when, Value: &value})
		} else {
			ratedSeries.Points = append(ratedSeries.Points, APIChartPoint{Time: when})
		}
		if ideal.Valid {
			value := ideal.Float64
			if strings.EqualFold(unitsLength, "mi") {
				value = kilometersToMiles(value)
			}
			idealSeries.Points = append(idealSeries.Points, APIChartPoint{Time: when, Value: &value})
		} else {
			idealSeries.Points = append(idealSeries.Points, APIChartPoint{Time: when})
		}
		usableSeries.Points = append(usableSeries.Points, APIChartPoint{Time: when, Value: floatPointer(usable)})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	for _, point := range ratedSeries.Points {
		if point.Value != nil && maxRated > 0 {
			orig := *point.Value
			if strings.EqualFold(unitsLength, "mi") {
				orig = milesToKilometers(orig)
			}
			value := math.Max(0, (1-orig/maxRated)*100)
			degradationSeries.Points = append(degradationSeries.Points, APIChartPoint{Time: point.Time, Value: &value})
		}
	}
	return []APIChartSeries{odometerSeries, ratedSeries, idealSeries, usableSeries, degradationSeries}, map[string]any{"max_rated_range_km": maxRated}, nil
}

func fetchBatteryHealthChartSeries(carID int, startUTC, endUTC string) ([]APIChartSeries, map[string]any, error) {
	query := `
		WITH daily AS (
			SELECT DISTINCT ON (timezone($4, positions.date)::date)
				timezone($4, positions.date)::date AS local_day,
				positions.date,
				positions.rated_battery_range_km
			FROM positions
			WHERE positions.car_id = $1
				AND positions.date >= $2
				AND positions.date < $3
				AND positions.rated_battery_range_km IS NOT NULL
			ORDER BY timezone($4, positions.date)::date, positions.date DESC
		)
		SELECT TO_CHAR(local_day::timestamp, 'YYYY-MM-DD HH24:MI:SS') AS bucket, rated_battery_range_km
		FROM daily
		ORDER BY local_day DESC`
	rows, err := db.Query(query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	points := []APIChartPoint{}
	maxRange := 0.0
	raw := []struct {
		time  string
		value float64
	}{}
	for rows.Next() {
		var bucket string
		var rated sql.NullFloat64
		if err := rows.Scan(&bucket, &rated); err != nil {
			return nil, nil, err
		}
		if rated.Valid && rated.Float64 > maxRange {
			maxRange = rated.Float64
		}
		raw = append(raw, struct {
			time  string
			value float64
		}{time: toChartTime(bucket), value: rated.Float64})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	for _, item := range raw {
		if maxRange <= 0 {
			continue
		}
		value := item.value / maxRange * 100.0
		points = append(points, APIChartPoint{Time: item.time, Value: &value})
	}
	return []APIChartSeries{{Name: "battery_health", Unit: "%", Points: points}}, map[string]any{"reference_max_rated_range_km": maxRange}, nil
}

func fetchVampireDrainPlaceholder() ([]APIChartSeries, map[string]any) {
	return []APIChartSeries{{Name: "vampire_drain", Unit: "%", Points: []APIChartPoint{}}}, map[string]any{"limitations": []string{"vampire drain needs reliable state-to-battery correlation; current schema support is insufficient for a safe calculation, so this endpoint returns an empty structure instead of guessed data"}}
}

func fetchMileageSeries(carID int, startUTC, endUTC, unitsLength string) ([]APIChartPoint, error) {
	query := `
		WITH monthly AS (
			SELECT date_trunc('month', timezone($4, positions.date)) AS month_bucket, MAX(positions.odometer) AS odometer
			FROM positions
			WHERE positions.car_id = $1 AND positions.date >= $2 AND positions.date < $3 AND positions.odometer IS NOT NULL
			GROUP BY month_bucket
		)
		SELECT TO_CHAR(month_bucket, 'YYYY-MM-DD HH24:MI:SS') AS bucket, odometer FROM monthly ORDER BY month_bucket DESC`
	points, err := fetchAggregateChartPoints(query, []any{carID, startUTC, endUTC, appUsersTimezone.String()})
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(unitsLength, "mi") {
		for i := range points {
			if points[i].Value != nil {
				v := kilometersToMiles(*points[i].Value)
				points[i].Value = &v
			}
		}
	}
	return points, nil
}

func fetchTimelineEvents(carID int, startUTC, endUTC string, page, show int, order string) ([]ActivityTimelineEvent, int, error) {
	baseQuery, params := buildStateTimelineBaseQuery(carID, startUTC, endUTC)
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	var total int
	if err := db.QueryRowContext(queryCtx, baseQuery+` SELECT COUNT(*)::int FROM timeline;`, params...).Scan(&total); err != nil {
		return nil, 0, err
	}
	items, err := fetchStateTimeline(carID, startUTC, endUTC, page, show)
	if err != nil {
		return nil, 0, err
	}
	events := mapStateTimelineToActivityEvents(items)
	if order == "asc" {
		sort.SliceStable(events, func(i, j int) bool { return events[i].StartDate < events[j].StartDate })
	}
	return events, total, nil
}

func fetchChargeCalendarMonth(carID int, year int, month int) (chargeCalendarMonth, error) {
	startLocal := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, appUsersTimezone)
	endLocal := startLocal.AddDate(0, 1, 0)
	startUTC := startLocal.UTC().Format(dbTimestampFormat)
	endUTC := endLocal.UTC().Format(dbTimestampFormat)
	query := `
		SELECT timezone($4, charging_processes.start_date)::date AS local_date,
			COUNT(*)::int AS charge_count,
			COALESCE(SUM(GREATEST(COALESCE(charging_processes.duration_min, 0), 0)), 0)::int AS total_duration_min,
			COALESCE(SUM(GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0)), 0)::float8 AS total_energy_added,
			NULLIF(SUM(CASE WHEN charging_processes.cost > 0 THEN charging_processes.cost ELSE 0 END), 0) AS total_cost,
			MIN(charging_processes.start_date) AS first_charge_at,
			MAX(charging_processes.end_date) AS last_charge_at
		FROM charging_processes
		WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL AND charging_processes.start_date >= $2 AND charging_processes.end_date < $3
		GROUP BY local_date ORDER BY local_date DESC`
	rows, err := db.Query(query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return chargeCalendarMonth{}, err
	}
	defer rows.Close()
	aggregates := map[string]chargeCalendarDay{}
	for rows.Next() {
		var day chargeCalendarDay
		var localDate time.Time
		var cost sql.NullFloat64
		var firstAt sql.NullString
		var lastAt sql.NullString
		if err := rows.Scan(&localDate, &day.ChargeCount, &day.DurationMin, &day.EnergyAdded, &cost, &firstAt, &lastAt); err != nil {
			return chargeCalendarMonth{}, err
		}
		dateKey := localDate.Format("2006-01-02")
		day.Date = dateKey
		day.Cost = floatPointer(cost)
		day.FirstChargeStart = timeZoneStringPointer(firstAt)
		day.LastChargeEnd = timeZoneStringPointer(lastAt)
		aggregates[dateKey] = day
	}
	if err := rows.Err(); err != nil {
		return chargeCalendarMonth{}, err
	}
	days := make([]chargeCalendarDay, 0, endLocal.Day()-1)
	nowLocal := time.Now().In(appUsersTimezone)
	for current := startLocal; current.Before(endLocal); current = current.AddDate(0, 0, 1) {
		key := current.Format("2006-01-02")
		day, ok := aggregates[key]
		if !ok {
			day = chargeCalendarDay{Date: key}
		}
		day.Day = current.Day()
		day.Weekday = int(current.Weekday())
		day.IsCurrentMonth = true
		day.IsToday = current.Year() == nowLocal.Year() && current.YearDay() == nowLocal.YearDay()
		days = append(days, day)
	}
	return chargeCalendarMonth{
		Year:      year,
		Month:     month,
		MonthName: startLocal.Month().String(),
		StartDate: startLocal.Format(time.RFC3339),
		EndDate:   endLocal.Add(-time.Second).Format(time.RFC3339),
		Days:      days,
	}, nil
}

func fetchVisitedMap(carID int, startUTC, endUTC string, limit int) ([]visitedPoint, *visitedBounds, bool, error) {
	query := `
		SELECT positions.date, positions.latitude, positions.longitude
		FROM positions
		INNER JOIN drives ON drives.id = positions.drive_id
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
			AND positions.latitude IS NOT NULL AND positions.longitude IS NOT NULL
		ORDER BY positions.date DESC
		LIMIT $4`
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, carID, startUTC, endUTC, limit+1)
	if err != nil {
		return nil, nil, false, err
	}
	defer rows.Close()
	points := []visitedPoint{}
	bounds := &visitedBounds{MinLatitude: 999, MaxLatitude: -999, MinLongitude: 999, MaxLongitude: -999}
	for rows.Next() {
		var date string
		var p visitedPoint
		if err := rows.Scan(&date, &p.Latitude, &p.Longitude); err != nil {
			return nil, nil, false, err
		}
		p.Time = getTimeInTimeZone(date)
		points = append(points, p)
		if p.Latitude < bounds.MinLatitude {
			bounds.MinLatitude = p.Latitude
		}
		if p.Latitude > bounds.MaxLatitude {
			bounds.MaxLatitude = p.Latitude
		}
		if p.Longitude < bounds.MinLongitude {
			bounds.MinLongitude = p.Longitude
		}
		if p.Longitude > bounds.MaxLongitude {
			bounds.MaxLongitude = p.Longitude
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, false, err
	}
	truncated := len(points) > limit
	if truncated {
		points = points[:limit]
	}
	if len(points) == 0 {
		return points, nil, false, nil
	}
	return points, bounds, truncated, nil
}
