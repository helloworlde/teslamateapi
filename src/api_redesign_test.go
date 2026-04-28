package main

import (
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

func routeSet(r *gin.Engine) map[string]bool {
	routes := map[string]bool{}
	for _, route := range r.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	return routes
}

func restoreEnv(t *testing.T, key string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
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
	routes := routeSet(r)
	for _, key := range []string{
		"GET /api/v1/cars/:CarID/summary",
		"GET /api/v1/cars/:CarID/dashboard",
		"GET /api/v1/cars/:CarID/calendar",
		"GET /api/v1/cars/:CarID/statistics",
		"GET /api/v1/cars/:CarID/series",
		"GET /api/v1/cars/:CarID/distributions",
		"GET /api/v1/cars/:CarID/insights",
		"GET /api/v1/cars/:CarID/timeline",
		"GET /api/v1/cars/:CarID/map/visited",
		"GET /api/v1/cars/:CarID/locations",
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
		"GET /api/v1/cars/:CarID/charts/drives/distance",
		"GET /api/v1/cars/:CarID/calendar/drives",
		"GET /api/v1/cars/:CarID/calendar/charges",
	} {
		if routes[key] {
			t.Fatalf("unexpected legacy route %s", key)
		}
	}
}

func TestCommandRoutesAreNotRegisteredByDefault(t *testing.T) {
	restoreEnv(t, "ENABLE_COMMANDS")
	_ = os.Unsetenv("ENABLE_COMMANDS")

	r := buildTestRouter()
	routes := routeSet(r)
	for _, key := range []string{
		"GET /api/v1/cars/:CarID/command",
		"POST /api/v1/cars/:CarID/command/:Command",
		"GET /api/v1/cars/:CarID/logging",
		"PUT /api/v1/cars/:CarID/logging/:Command",
		"POST /api/v1/cars/:CarID/wake_up",
	} {
		if routes[key] {
			t.Fatalf("command route should be disabled by default: %s", key)
		}
	}

	legacy := gin.New()
	registerLegacyRedirects(legacy, "/api/v1")
	legacyRoutes := routeSet(legacy)
	for _, key := range []string{
		"GET /cars/:CarID/command",
		"POST /cars/:CarID/command/:Command",
		"GET /cars/:CarID/logging",
		"PUT /cars/:CarID/logging/:Command",
		"POST /cars/:CarID/wake_up",
	} {
		if legacyRoutes[key] {
			t.Fatalf("legacy command redirect should be disabled by default: %s", key)
		}
	}
}

func TestCommandRoutesAreRegisteredWhenExplicitlyEnabled(t *testing.T) {
	restoreEnv(t, "ENABLE_COMMANDS")
	_ = os.Setenv("ENABLE_COMMANDS", "true")

	r := buildTestRouter()
	routes := routeSet(r)
	for _, key := range []string{
		"GET /api/v1/cars/:CarID/command",
		"POST /api/v1/cars/:CarID/command/:Command",
		"GET /api/v1/cars/:CarID/logging",
		"PUT /api/v1/cars/:CarID/logging/:Command",
		"POST /api/v1/cars/:CarID/wake_up",
	} {
		if !routes[key] {
			t.Fatalf("command route should be enabled when ENABLE_COMMANDS=true: %s", key)
		}
	}
}

func TestDashboardInvalidDateFallback(t *testing.T) {
	oldTZ := appUsersTimezone
	appUsersTimezone = time.FixedZone("CST", 8*3600)
	defer func() { appUsersTimezone = oldTZ }()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cars/1/dashboard?period=custom&startDate=not-a-date&endDate=2026-04-01", nil)
	dr, warnings := parseDateRangeWithMonthFallback(c, "month")
	if dr.Period != "month" {
		t.Fatalf("expected month fallback, got %s", dr.Period)
	}
	if len(warnings) == 0 {
		t.Fatal("expected fallback warning")
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
		"/api/v1/cars/%d/dashboard",
		"/api/v1/cars/%d/calendar?startDate=2026-04-01&endDate=2026-04-30",
		"/api/v1/cars/%d/statistics",
		"/api/v1/cars/%d/series?startDate=2026-04-01&endDate=2026-04-30&metrics=distance,speed",
		"/api/v1/cars/%d/distributions?startDate=2026-04-01&endDate=2026-04-30&metrics=drive_start_hour",
		"/api/v1/cars/%d/insights?startDate=2026-04-01&endDate=2026-04-30",
		"/api/v1/cars/%d/timeline?startDate=2026-04-01&endDate=2026-04-30",
		"/api/v1/cars/%d/map/visited",
		"/api/v1/cars/%d/locations?startDate=2026-04-01&endDate=2026-04-30",
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
