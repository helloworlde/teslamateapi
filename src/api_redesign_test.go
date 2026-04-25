package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func buildTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	v1 := api.Group("/v1")
	registerDocsRoutes(v1, "/api/v1")
	registerCompatibleV1Routes(v1)
	registerExtendedV1Routes(v1)
	api.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) })
	api.GET("/healthz", healthz)
	api.GET("/readyz", readyz)
	return r
}

func TestParseAPITimeFormats(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	cases := []string{
		"2026-04-02T10:55:30+08:00",
		"2026-04-02T10:55:30Z",
		"2026-04-02T10:55:30 08:00",
		"2026-04-02 10:55:30",
		"2026-04-02",
	}
	for _, input := range cases {
		if _, err := parseAPITime(input, loc); err != nil {
			t.Fatalf("parseAPITime(%q): %v", input, err)
		}
	}
}

func TestParseDateRangeValuesDateOnlyEnd(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	start, end, err := parseDateRangeValues("2026-04-02", "2026-04-02", loc)
	if err != nil {
		t.Fatal(err)
	}
	if start == "" || end == "" {
		t.Fatalf("empty range: %q %q", start, end)
	}
	if end <= start {
		t.Fatalf("expected end > start, got start=%q end=%q", start, end)
	}
}

func TestRouteRegistryContainsNewRoutes(t *testing.T) {
	r := buildTestRouter()
	routes := map[string]bool{}
	for _, route := range r.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	for _, key := range []string{
		"GET /api/v1/cars/:CarID/summary",
		"GET /api/v1/cars/:CarID/statistics",
		"GET /api/v1/cars/:CarID/timeline",
		"GET /api/v1/cars/:CarID/calendar/drives",
		"GET /api/v1/cars/:CarID/calendar/charges",
		"GET /api/v1/cars/:CarID/map/visited",
		"GET /api/v1/cars/:CarID/charts/drives/distance",
		"GET /api/v1/cars/:CarID/charts/charges/location",
		"GET /api/v1/cars/:CarID/charts/battery/range",
		"GET /api/v1/cars/:CarID/charts/vampire-drain",
		"GET /api/v1/cars/:CarID/drives/:DriveID/details",
		"GET /api/v1/cars/:CarID/charges/:ChargeID/details",
	} {
		if !routes[key] {
			t.Fatalf("missing route %s", key)
		}
	}
	for _, key := range []string{
		"GET /api/v1/cars/:CarID/summaries",
		"GET /api/v1/cars/:CarID/activity-timeline",
		"GET /api/v1/cars/:CarID/dashboards/drives",
		"GET /api/v1/cars/:CarID/parking-sessions",
	} {
		if routes[key] {
			t.Fatalf("unexpected legacy route %s", key)
		}
	}
}

func TestSummaryInvalidDateResponse(t *testing.T) {
	oldTZ := appUsersTimezone
	appUsersTimezone = time.FixedZone("CST", 8*3600)
	defer func() { appUsersTimezone = oldTZ }()
	r := buildTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cars/1/summary?startDate=not-a-date", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body APIErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "invalid_date" {
		t.Fatalf("unexpected code: %+v", body.Error)
	}
}

func TestIntegrationRedesignedEndpoints(t *testing.T) {
	if os.Getenv("TESLAMATEAPI_ENDPOINT_CHECK") != "1" {
		t.Skip("set TESLAMATEAPI_ENDPOINT_CHECK=1 with DATABASE_* and TZ to run redesigned endpoint integration tests")
	}
	var err error
	appUsersTimezone, err = time.LoadLocation(getEnv("TZ", "Europe/Berlin"))
	if err != nil {
		t.Fatal(err)
	}
	initDBconnection()
	defer func() {
		if db != nil {
			_ = db.Close()
			db = nil
		}
	}()
	r := buildTestRouter()
	carID := getEnvAsInt("TESLAMATEAPI_ENDPOINT_CAR_ID", 1)
	paths := []string{
		"/api/v1/cars/%d/summary",
		"/api/v1/cars/%d/statistics",
		"/api/v1/cars/%d/charts/overview",
		"/api/v1/cars/%d/charts/drives/distance?bucket=month",
		"/api/v1/cars/%d/charts/drives/energy?bucket=month",
		"/api/v1/cars/%d/charts/drives/efficiency?bucket=month",
		"/api/v1/cars/%d/charts/drives/speed?bucket=month",
		"/api/v1/cars/%d/charts/drives/temperature?bucket=month",
		"/api/v1/cars/%d/charts/charges/energy?bucket=month",
		"/api/v1/cars/%d/charts/charges/cost?bucket=month",
		"/api/v1/cars/%d/charts/charges/efficiency?bucket=month",
		"/api/v1/cars/%d/charts/charges/power?bucket=month",
		"/api/v1/cars/%d/charts/charges/location",
		"/api/v1/cars/%d/charts/charges/soc",
		"/api/v1/cars/%d/charts/battery/range",
		"/api/v1/cars/%d/charts/battery/health",
		"/api/v1/cars/%d/charts/states/duration",
		"/api/v1/cars/%d/charts/vampire-drain",
		"/api/v1/cars/%d/charts/mileage",
		"/api/v1/cars/%d/timeline",
		"/api/v1/cars/%d/calendar/drives",
		"/api/v1/cars/%d/calendar/charges",
		"/api/v1/cars/%d/map/visited",
		"/api/v1/cars/%d/insights",
		"/api/v1/cars/%d/insights/events",
		"/api/v1/cars/%d/analytics/activity",
		"/api/v1/cars/%d/analytics/regeneration",
	}
	for _, pattern := range paths {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(pattern, carID), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Fatalf("endpoint %s returned %d body=%s", pattern, w.Code, w.Body.String())
		}
	}
}
