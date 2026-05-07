package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/src/internal/docsui"
)

func buildTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	v1 := api.Group("/v1")
	docsui.RegisterRoutes(v1, "/api/v1")
	registerCompatibleV1Routes(v1)
	v2 := api.Group("/v2")
	docsui.RegisterRoutes(v2, "/api/v2")
	registerExtendedV2Routes(v2)
	api.GET("/ping", apiPing)
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
		"GET /api/v1/cars",
		"GET /api/v1/cars/:CarID",
		"GET /api/v1/cars/:CarID/battery-health",
		"GET /api/v1/cars/:CarID/charges",
		"GET /api/v1/cars/:CarID/charges/current",
		"GET /api/v1/cars/:CarID/charges/:ChargeID",
		"GET /api/v1/cars/:CarID/drives",
		"GET /api/v1/cars/:CarID/drives/:DriveID",
		"GET /api/v1/cars/:CarID/status",
		"GET /api/v1/cars/:CarID/updates",
		"GET /api/v1/globalsettings",
		"GET /api/v2/cars/:CarID/stats",
		"GET /api/v2/cars/:CarID/activity",
		"GET /api/v2/cars/:CarID/series",
		"GET /api/v2/cars/:CarID/distributions",
		"GET /api/v2/cars/:CarID/locations",
		"GET /api/v2/cars/:CarID/locations/heatmap",
		"GET /api/v2/cars/:CarID/analysis/insights",
		"GET /api/v2/cars/:CarID/analysis/trends",
		"GET /api/v2/cars/:CarID/analysis/records",
	} {
		if !routes[key] {
			t.Fatalf("missing route %s", key)
		}
	}
	for _, key := range []string{
		"GET /api/v1/cars/:CarID/stats",
		"GET /api/v1/cars/:CarID/activity",
		"GET /api/v1/cars/:CarID/series",
		"GET /api/v1/cars/:CarID/distributions",
		"GET /api/v1/cars/:CarID/locations",
		"GET /api/v1/cars/:CarID/locations/heatmap",
		"GET /api/v1/cars/:CarID/analysis/insights",
		"GET /api/v1/cars/:CarID/analysis/trends",
		"GET /api/v1/cars/:CarID/analysis/records",
		"GET /api/v2/cars/:CarID/status",
		"GET /api/v2/cars/:CarID/drives",
		"GET /api/v2/cars/:CarID/charges",
	} {
		if routes[key] {
			t.Fatalf("route must not be registered: %s", key)
		}
	}
}

func TestDashboardInvalidDateReturnsError(t *testing.T) {
	oldTZ := appUsersTimezone
	appUsersTimezone = time.FixedZone("CST", 8*3600)
	defer func() { appUsersTimezone = oldTZ }()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/stats?period=custom&startDate=not-a-date&endDate=2026-04-01", nil)
	if _, err := parseDateRangeStrictOrDefault(c, "month"); err == nil {
		t.Fatal("expected invalid date error")
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
		"/api/v2/cars/%d/stats",
		"/api/v2/cars/%d/activity?startDate=2026-04-01&endDate=2026-04-30",
		"/api/v2/cars/%d/series?scope=drives&startDate=2026-04-01&endDate=2026-04-30&metrics=distance,speed",
		"/api/v2/cars/%d/series?scope=charges&startDate=2026-04-01&endDate=2026-04-30&metrics=energy,power",
		"/api/v2/cars/%d/series?scope=battery&startDate=2026-04-01&endDate=2026-04-30",
		"/api/v2/cars/%d/distributions?scope=drives&startDate=2026-04-01&endDate=2026-04-30&metrics=start_hour",
		"/api/v2/cars/%d/distributions?scope=charges&startDate=2026-04-01&endDate=2026-04-30&metrics=energy",
		"/api/v2/cars/%d/analysis/insights?startDate=2026-04-01&endDate=2026-04-30",
		"/api/v2/cars/%d/analysis/trends?startDate=2026-04-01&endDate=2026-04-30",
		"/api/v2/cars/%d/analysis/records",
		"/api/v2/cars/%d/locations?startDate=2026-04-01&endDate=2026-04-30",
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
