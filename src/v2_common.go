package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func parseV2AnalyticsQuery(c *gin.Context, now time.Time) (V2AnalyticsQuery, V2TimeRange, error) {
	query := V2AnalyticsQuery{
		Period:   strings.ToLower(v2QueryDefault(c, "period", v2DefaultPeriod)),
		Start:    strings.TrimSpace(c.Query("start")),
		End:      strings.TrimSpace(c.Query("end")),
		Timezone: v2QueryDefault(c, "timezone", defaultV2Timezone()),
		Compare:  strings.ToLower(v2QueryDefault(c, "compare", v2DefaultCompare)),
		GroupBy:  strings.TrimSpace(c.Query("group_by")),
		Metrics:  strings.TrimSpace(c.Query("metrics")),
		Include:  strings.TrimSpace(c.Query("include")),
	}

	if !v2AllowedPeriods[query.Period] {
		return query, V2TimeRange{}, fmt.Errorf("invalid period %q", query.Period)
	}
	if !v2AllowedCompares[query.Compare] {
		return query, V2TimeRange{}, fmt.Errorf("invalid compare %q", query.Compare)
	}

	location, err := time.LoadLocation(query.Timezone)
	if err != nil {
		return query, V2TimeRange{}, fmt.Errorf("invalid timezone %q", query.Timezone)
	}

	start, end, err := resolveV2Range(query, location, now)
	if err != nil {
		return query, V2TimeRange{}, err
	}

	result := V2TimeRange{
		Period:   query.Period,
		Timezone: query.Timezone,
		Compare:  query.Compare,
		Start:    start.UTC(),
		End:      end.UTC(),
	}

	if query.Compare == "previous_period" {
		duration := result.End.Sub(result.Start)
		previousEnd := result.Start
		previousStart := previousEnd.Add(-duration)
		result.PreviousStart = &previousStart
		result.PreviousEnd = &previousEnd
	}

	return query, result, nil
}

func v2QueryDefault(c *gin.Context, key string, fallback string) string {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return fallback
	}
	return value
}

func defaultV2Timezone() string {
	return defaultV2Location().String()
}

func defaultV2Location() *time.Location {
	if appUsersTimezone != nil {
		return appUsersTimezone
	}
	location, err := time.LoadLocation(getEnv("TZ", "Europe/Berlin"))
	if err != nil {
		return time.UTC
	}
	return location
}

func resolveV2Range(query V2AnalyticsQuery, location *time.Location, now time.Time) (time.Time, time.Time, error) {
	now = now.In(location)
	if !now.After(time.Time{}) {
		now = time.Now().In(location)
	}

	var start time.Time
	var end time.Time
	var err error

	switch query.Period {
	case "day":
		start = beginningOfDay(now)
		end = start.AddDate(0, 0, 1)
	case "week":
		weekdayOffset := (int(now.Weekday()) + 6) % 7
		start = beginningOfDay(now).AddDate(0, 0, -weekdayOffset)
		end = start.AddDate(0, 0, 7)
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
		end = start.AddDate(0, 1, 0)
	case "quarter":
		quarterMonth := time.Month(((int(now.Month())-1)/3)*3 + 1)
		start = time.Date(now.Year(), quarterMonth, 1, 0, 0, 0, 0, location)
		end = start.AddDate(0, 3, 0)
	case "year":
		start = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, location)
		end = start.AddDate(1, 0, 0)
	case "custom":
		if query.Start == "" || query.End == "" {
			return time.Time{}, time.Time{}, fmt.Errorf("custom period requires both start and end")
		}
	}

	if query.Start != "" {
		start, err = parseV2ClientTime(query.Start, location)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start: %w", err)
		}
	}
	if query.End != "" {
		end, err = parseV2ClientTime(query.End, location)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end: %w", err)
		}
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end must be after start")
	}

	return start, end, nil
}

func beginningOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func parseV2ClientTime(value string, location *time.Location) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", value, location); err == nil {
		return t, nil
	}
	normalized := strings.ReplaceAll(value, "T", " ")
	return time.ParseInLocation(time.DateTime, normalized, location)
}

func newV2Meta(carID int64, timeRange V2TimeRange) V2Meta {
	location := timeRangeLocation(timeRange)
	return V2Meta{
		CarID:       carID,
		Period:      timeRange.Period,
		Timezone:    timeRange.Timezone,
		Start:       timeRange.Start.In(location).Format(time.RFC3339),
		End:         timeRange.End.In(location).Format(time.RFC3339),
		Compare:     timeRange.Compare,
		Unit:        defaultV2Unit(),
		GeneratedAt: time.Now().In(location).Format(time.RFC3339),
	}
}

func timeRangeLocation(timeRange V2TimeRange) *time.Location {
	if timeRange.Timezone == "" {
		return defaultV2Location()
	}
	location, err := time.LoadLocation(timeRange.Timezone)
	if err != nil {
		return defaultV2Location()
	}
	return location
}

func defaultV2Unit() V2Unit {
	return V2Unit{
		Distance:    "km",
		Energy:      "kWh",
		Power:       "kW",
		Temperature: "C",
		Currency:    "CNY",
	}
}

func v2JSON(c *gin.Context, status int, data interface{}, meta V2Meta) {
	c.JSON(status, V2APIResponse{Data: data, Meta: meta})
}

func v2Error(c *gin.Context, status int, code string, message string, details interface{}) {
	c.JSON(status, APIErrorResponse{
		Error: APIErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func v2BadRequest(c *gin.Context, message string, details interface{}) {
	v2Error(c, http.StatusBadRequest, "BAD_REQUEST", message, details)
}

// parseV2IncludeSet splits the include query parameter into a lower-cased set.
// Empty input returns an empty set; callers decide which keys are required.
func parseV2IncludeSet(raw string) map[string]bool {
	out := map[string]bool{}
	if raw == "" {
		return out
	}
	for _, part := range strings.Split(raw, ",") {
		key := strings.ToLower(strings.TrimSpace(part))
		if key != "" {
			out[key] = true
		}
	}
	return out
}

func float64Ptr(value float64) *float64 {
	return &value
}

func compareFloat(current float64, previous float64) V2ComparisonValue {
	comparison := V2ComparisonValue{
		Current:  float64Ptr(current),
		Previous: float64Ptr(previous),
	}
	delta := current - previous
	comparison.Delta = &delta
	if previous != 0 {
		deltaPercent := delta / previous * 100
		comparison.DeltaPercent = &deltaPercent
	}
	return comparison
}
