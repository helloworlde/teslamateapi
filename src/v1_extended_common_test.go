package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestParseDateRangeFromQueryMonthWeekYearCustom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := appUsersTimezone
	appUsersTimezone = time.FixedZone("CST", 8*3600)
	t.Cleanup(func() { appUsersTimezone = old })

	makeCtx := func(raw string) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, raw, nil)
		c.Request = req
		return c
	}

	if _, err := parseDateRangeFromQuery(makeCtx("/?period=month&date=2026-04-12"), "month"); err != nil {
		t.Fatal(err)
	}
	if _, err := parseDateRangeFromQuery(makeCtx("/?period=week&date=2026-04-12"), "month"); err != nil {
		t.Fatal(err)
	}
	if _, err := parseDateRangeFromQuery(makeCtx("/?period=year&date=2026-04-12"), "month"); err != nil {
		t.Fatal(err)
	}
	if _, err := parseDateRangeFromQuery(makeCtx("/?period=custom&startDate=2026-04-01&endDate=2026-04-30"), "month"); err != nil {
		t.Fatal(err)
	}
}

func TestParseDateRangeFromQueryUsesConfiguredTimezoneAndInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := appUsersTimezone
	appUsersTimezone = time.FixedZone("CST", 8*3600)
	t.Cleanup(func() { appUsersTimezone = old })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?period=month&date=2026-04-01", nil)
	dr, err := parseDateRangeFromQuery(c, "month")
	if err != nil {
		t.Fatal(err)
	}
	if dr.Timezone.String() != "CST" {
		t.Fatalf("expected configured timezone, got %s", dr.Timezone.String())
	}

	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest(http.MethodGet, "/?period=custom&startDate=bad&endDate=2026-04-01", nil)
	if _, err := parseDateRangeFromQuery(c2, "custom"); err == nil {
		t.Fatal("expected invalid date error")
	}

	c3, _ := gin.CreateTestContext(httptest.NewRecorder())
	c3.Request = httptest.NewRequest(http.MethodGet, "/?period=custom&startDate=2026-04-30&endDate=2026-04-01", nil)
	if _, err := parseDateRangeFromQuery(c3, "custom"); err == nil {
		t.Fatal("expected end before start error")
	}
}

func TestDBTimeRangeUsesLocalTimezoneAndExclusiveEnd(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	dr := v1DateRange{
		Period:   "custom",
		Timezone: loc,
		Start:    time.Date(2026, 4, 1, 0, 0, 0, 0, loc),
		End:      time.Date(2026, 4, 30, 23, 59, 59, 0, loc),
	}

	startUTC, endUTC := dbTimeRange(dr)
	if startUTC != "2026-03-31T16:00:00Z" {
		t.Fatalf("unexpected start UTC: %s", startUTC)
	}
	if endUTC != "2026-04-30T16:00:00Z" {
		t.Fatalf("unexpected exclusive end UTC: %s", endUTC)
	}
}

func TestStatisticsCalcNullAndZeroHandling(t *testing.T) {
	d := 20.0
	dur := 3600.0
	e := 3.0
	ce := 5.0

	if v := calcAvgSpeed(&d, &dur); v == nil || *v != 20 {
		t.Fatalf("avg speed: %+v", v)
	}
	if v := calcEfficiencyWhPerKm(&e, &d); v == nil || *v != 150 {
		t.Fatalf("efficiency: %+v", v)
	}
	if v := calcChargeEfficiencyPercent(&e, &ce); v == nil || *v != 60 {
		t.Fatalf("charge efficiency: %+v", v)
	}
	zero := 0.0
	if v := calcAvgSpeed(&d, &zero); v != nil {
		t.Fatalf("expected nil on zero duration: %+v", *v)
	}
	if v := calcEfficiencyWhPerKm(&e, &zero); v != nil {
		t.Fatalf("expected nil on zero distance: %+v", *v)
	}
	if v := calcChargeEfficiencyPercent(&e, &zero); v != nil {
		t.Fatalf("expected nil on zero charger energy: %+v", *v)
	}
}
