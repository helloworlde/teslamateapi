package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const aggregateQueryTimeout = 1500 * time.Millisecond

func newAggregateQueryContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), aggregateQueryTimeout)
}

type v1Meta struct {
	CarID       int    `json:"car_id,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Unit        string `json:"unit,omitempty"`
	GeneratedAt string `json:"generated_at,omitempty"`
	Version     string `json:"version,omitempty"`
}

type v1Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type v1ObjectEnvelope struct {
	Data any    `json:"data"`
	Meta v1Meta `json:"meta"`
}

type v1ListEnvelope struct {
	Data       any          `json:"data"`
	Pagination v1Pagination `json:"pagination"`
	Meta       v1Meta       `json:"meta"`
}

type v1Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type v1ErrorEnvelope struct {
	Error v1Error `json:"error"`
}

type v1Range struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Timezone string `json:"timezone"`
}

type v1DateRange struct {
	Period   string
	Timezone *time.Location
	Start    time.Time
	End      time.Time
}

func writeV1Error(c *gin.Context, status int, code, message string, details map[string]any) {
	details = responseErrorDetails(c, status, code, message, details)
	c.JSON(status, v1ErrorEnvelope{
		Error: v1Error{
			Code:    strings.ToUpper(code),
			Message: message,
			Details: details,
		},
	})
}

func buildV1Meta(carID int, tzName string, unit string) v1Meta {
	return v1Meta{
		CarID:       carID,
		Timezone:    tzName,
		Unit:        unit,
		GeneratedAt: time.Now().In(appUsersTimezone).Format(time.RFC3339),
		Version:     "v1",
	}
}

func writeV1Object(c *gin.Context, data any, meta v1Meta) {
	c.JSON(http.StatusOK, v1ObjectEnvelope{
		Data: data,
		Meta: meta,
	})
}

func writeV1List(c *gin.Context, data any, pagination v1Pagination, meta v1Meta) {
	c.JSON(http.StatusOK, v1ListEnvelope{
		Data:       data,
		Pagination: pagination,
		Meta:       meta,
	})
}

func parseOffsetLimit(c *gin.Context, defaultLimit, maxLimit int) (int, int, error) {
	limit, err := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("limit", strconv.Itoa(defaultLimit))))
	if err != nil || limit <= 0 {
		return 0, 0, fmt.Errorf("limit must be positive integer")
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset, err := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("offset", "0")))
	if err != nil || offset < 0 {
		return 0, 0, fmt.Errorf("offset must be integer >= 0")
	}
	return offset, limit, nil
}

func parseTimezoneParam(c *gin.Context) (*time.Location, string, error) {
	defaultLoc := appUsersTimezone
	if defaultLoc == nil {
		defaultLoc = time.Local
	}
	return defaultLoc, defaultLoc.String(), nil
}

func parseFlexibleTime(raw string, loc *time.Location) (time.Time, error) {
	return parseAPITime(strings.TrimSpace(raw), loc)
}

func parseDateOnlyOrTime(raw string, loc *time.Location, endOfDay bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if len(raw) == len("2006-01-02") && strings.Count(raw, "-") == 2 && !strings.Contains(raw, "T") && !strings.Contains(raw, " ") {
		t, err := time.ParseInLocation("2006-01-02", raw, loc)
		if err != nil {
			return time.Time{}, err
		}
		if endOfDay {
			return t.Add(24*time.Hour - time.Second), nil
		}
		return t, nil
	}
	return parseFlexibleTime(raw, loc)
}

func parseDateRangeFromQuery(c *gin.Context, defaultPeriod string) (v1DateRange, error) {
	loc, _, err := parseTimezoneParam(c)
	if err != nil {
		return v1DateRange{}, err
	}
	period := strings.ToLower(strings.TrimSpace(c.DefaultQuery("period", defaultPeriod)))
	if period == "" {
		period = defaultPeriod
	}
	switch period {
	case "month", "week", "year", "custom":
	default:
		return v1DateRange{}, fmt.Errorf("period must be one of year|month|week|custom")
	}

	now := time.Now().In(loc)
	if period == "custom" {
		start, err := parseDateOnlyOrTime(c.Query("startDate"), loc, false)
		if err != nil {
			return v1DateRange{}, err
		}
		end, err := parseDateOnlyOrTime(c.Query("endDate"), loc, true)
		if err != nil {
			return v1DateRange{}, err
		}
		if start.IsZero() || end.IsZero() {
			return v1DateRange{}, fmt.Errorf("startDate and endDate are required when period=custom")
		}
		if end.Before(start) {
			return v1DateRange{}, fmt.Errorf("startDate must be before endDate")
		}
		return v1DateRange{Period: period, Timezone: loc, Start: start, End: end}, nil
	}

	refRaw := strings.TrimSpace(c.Query("date"))
	ref := now
	if refRaw != "" {
		ref, err = parseDateOnlyOrTime(refRaw, loc, false)
		if err != nil {
			return v1DateRange{}, err
		}
	}
	var start, end time.Time
	switch period {
	case "year":
		start = time.Date(ref.Year(), 1, 1, 0, 0, 0, 0, loc)
		end = start.AddDate(1, 0, 0).Add(-time.Second)
	case "month":
		start = time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, loc)
		end = start.AddDate(0, 1, 0).Add(-time.Second)
	case "week":
		weekday := int(ref.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(weekday - 1))
		end = start.AddDate(0, 0, 7).Add(-time.Second)
	}
	return v1DateRange{Period: period, Timezone: loc, Start: start, End: end}, nil
}

func buildRangeDTO(r v1DateRange) v1Range {
	return v1Range{
		Start:    r.Start.In(r.Timezone).Format(time.RFC3339),
		End:      r.End.In(r.Timezone).Format(time.RFC3339),
		Timezone: r.Timezone.String(),
	}
}

func dbTimeRange(r v1DateRange) (string, string) {
	// API ranges expose an inclusive, timezone-aware end timestamp. SQL filters use
	// a half-open range so adjacent local days/months do not double-count boundary rows.
	return r.Start.UTC().Format(dbTimestampFormat), r.End.Add(time.Second).UTC().Format(dbTimestampFormat)
}

func parseDateRangeStrictOrDefault(c *gin.Context, defaultPeriod string) (v1DateRange, error) {
	loc, _, err := parseTimezoneParam(c)
	if err != nil {
		return v1DateRange{}, err
	}
	startRaw := strings.TrimSpace(c.Query("startDate"))
	endRaw := strings.TrimSpace(c.Query("endDate"))
	if startRaw != "" || endRaw != "" {
		start, err := parseDateOnlyOrTime(startRaw, loc, false)
		if err != nil {
			return v1DateRange{}, err
		}
		end, err := parseDateOnlyOrTime(endRaw, loc, true)
		if err != nil {
			return v1DateRange{}, err
		}
		if start.IsZero() || end.IsZero() {
			return v1DateRange{}, fmt.Errorf("startDate and endDate are required together")
		}
		if end.Before(start) {
			return v1DateRange{}, fmt.Errorf("startDate must be before endDate")
		}
		return v1DateRange{Period: "custom", Timezone: loc, Start: start, End: end}, nil
	}
	return parseDateRangeFromQuery(c, defaultPeriod)
}
