package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestParseV2AnalyticsQueryRejectsInvalidPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v2?period=hour", nil)

	_, _, err := parseV2AnalyticsQuery(c, time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected invalid period error")
	}
}

func TestParseV2AnalyticsQueryPreviousPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v2?period=custom&start=2026-05-01T00:00:00Z&end=2026-05-08T00:00:00Z&compare=previous_period", nil)

	_, timeRange, err := parseV2AnalyticsQuery(c, time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if timeRange.PreviousStart == nil || timeRange.PreviousEnd == nil {
		t.Fatal("expected previous period range")
	}
	if got := timeRange.PreviousStart.Format(time.RFC3339); got != "2026-04-24T00:00:00Z" {
		t.Fatalf("unexpected previous start: %s", got)
	}
	if got := timeRange.PreviousEnd.Format(time.RFC3339); got != "2026-05-01T00:00:00Z" {
		t.Fatalf("unexpected previous end: %s", got)
	}
}

func TestParseV2AnalyticsQueryDefaultsTimezoneFromTZEnv(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldLocation := appUsersTimezone
	appUsersTimezone = nil
	t.Cleanup(func() {
		appUsersTimezone = oldLocation
	})
	t.Setenv("TZ", "Asia/Shanghai")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v2?period=day&timezone=", nil)

	_, timeRange, err := parseV2AnalyticsQuery(c, time.Date(2026, 5, 7, 16, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if timeRange.Timezone != "Asia/Shanghai" {
		t.Fatalf("unexpected timezone: %s", timeRange.Timezone)
	}
	if got := timeRange.Start.Format(time.RFC3339); got != "2026-05-07T16:00:00Z" {
		t.Fatalf("unexpected UTC start: %s", got)
	}
	if got := timeRange.End.Format(time.RFC3339); got != "2026-05-08T16:00:00Z" {
		t.Fatalf("unexpected UTC end: %s", got)
	}

	meta := newV2Meta(1, timeRange, nil)
	if meta.Start != "2026-05-08T00:00:00+08:00" {
		t.Fatalf("unexpected localized meta start: %s", meta.Start)
	}
	if meta.End != "2026-05-09T00:00:00+08:00" {
		t.Fatalf("unexpected localized meta end: %s", meta.End)
	}
}
