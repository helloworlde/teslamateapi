package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	docs "github.com/tobiasehlert/teslamateapi/src/docs"
	"github.com/tobiasehlert/teslamateapi/src/internal/docsui"
)

func TestOpenAPIDocumentContainsStatisticsAndInsights(t *testing.T) {
	s := docs.SwaggerInfo.ReadDoc()
	for _, sub := range []string{
		"/v1/cars/{CarID}/summary",
		"/v1/cars/{CarID}/dashboard",
		"/v1/cars/{CarID}/realtime",
		"/v1/cars/{CarID}/calendar",
		"/v1/cars/{CarID}/statistics",
		"/v1/cars/{CarID}/series/drives",
		"/v1/cars/{CarID}/series/charges",
		"/v1/cars/{CarID}/series/battery",
		"/v1/cars/{CarID}/series/states",
		"/v1/cars/{CarID}/distributions/drives",
		"/v1/cars/{CarID}/distributions/charges",
		"/v1/cars/{CarID}/timeline",
		"/v1/cars/{CarID}/insights",
		"/v1/cars/{CarID}/locations",
	} {
		if !strings.Contains(s, sub) {
			t.Fatalf("OpenAPI doc missing %q", sub)
		}
	}
}

func TestOpenAPIDocumentDoesNotExposeTeslaAccountTokens(t *testing.T) {
	s := strings.ToLower(docs.SwaggerInfo.ReadDoc())
	for _, forbidden := range []string{"access_token", "refresh_token"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("OpenAPI doc exposes forbidden Tesla account token field %q", forbidden)
		}
	}
}

func TestOpenAPIGetCar200SchemaIsCarsEnvelope(t *testing.T) {
	raw := docs.SwaggerInfo.ReadDoc()
	var root map[string]any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		t.Fatalf("parse openapi: %v", err)
	}
	paths, _ := root["paths"].(map[string]any)
	for _, p := range []string{"/v1/cars", "/v1/cars/{CarID}"} {
		node, ok := paths[p].(map[string]any)
		if !ok {
			t.Fatalf("missing path %q", p)
		}
		get, ok := node["get"].(map[string]any)
		if !ok {
			t.Fatalf("path %q missing get", p)
		}
		responses, ok := get["responses"].(map[string]any)
		if !ok {
			t.Fatalf("path %q missing responses", p)
		}
		r200, ok := responses["200"].(map[string]any)
		if !ok {
			t.Fatalf("path %q missing 200", p)
		}
		schema, ok := r200["schema"].(map[string]any)
		if !ok {
			t.Fatalf("path %q 200 missing schema", p)
		}
		ref, _ := schema["$ref"].(string)
		if want := "#/definitions/main.CarsV1Envelope"; ref != want {
			t.Fatalf("path %q 200 schema ref = %q want %q", p, ref, want)
		}
	}
	defs, _ := root["definitions"].(map[string]any)
	def, ok := defs["main.CarsV1Envelope"].(map[string]any)
	if !ok {
		t.Fatal("missing definitions.main.CarsV1Envelope")
	}
	props, _ := def["properties"].(map[string]any)
	if props["data"] == nil {
		t.Fatal("CarsV1Envelope.properties.data missing")
	}
}

func TestOpenAPIExtendedRoutesUseConcreteResponseModels(t *testing.T) {
	raw := docs.SwaggerInfo.ReadDoc()
	var root map[string]any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		t.Fatalf("parse openapi: %v", err)
	}
	paths, _ := root["paths"].(map[string]any)
	expected := map[string]string{
		"/v1/cars/{CarID}/summary":               "#/definitions/main.SummaryV2Envelope",
		"/v1/cars/{CarID}/dashboard":             "#/definitions/main.DashboardV2Envelope",
		"/v1/cars/{CarID}/realtime":              "#/definitions/main.RealtimeV2Envelope",
		"/v1/cars/{CarID}/calendar":              "#/definitions/main.CalendarV2Envelope",
		"/v1/cars/{CarID}/statistics":            "#/definitions/main.StatisticsV2Envelope",
		"/v1/cars/{CarID}/series/drives":         "#/definitions/main.SeriesV2Envelope",
		"/v1/cars/{CarID}/series/charges":        "#/definitions/main.SeriesV2Envelope",
		"/v1/cars/{CarID}/series/battery":        "#/definitions/main.SeriesV2Envelope",
		"/v1/cars/{CarID}/series/states":         "#/definitions/main.SeriesV2Envelope",
		"/v1/cars/{CarID}/distributions/drives":  "#/definitions/main.DistributionsV2Envelope",
		"/v1/cars/{CarID}/distributions/charges": "#/definitions/main.DistributionsV2Envelope",
		"/v1/cars/{CarID}/insights":              "#/definitions/main.InsightsV2Envelope",
		"/v1/cars/{CarID}/timeline":              "#/definitions/main.TimelineV2Envelope",
		"/v1/cars/{CarID}/map/visited":           "#/definitions/main.VisitedMapV2Envelope",
		"/v1/cars/{CarID}/locations":             "#/definitions/main.LocationsV2Envelope",
	}
	for path, want := range expected {
		node, ok := paths[path].(map[string]any)
		if !ok {
			t.Fatalf("missing path %q", path)
		}
		get, ok := node["get"].(map[string]any)
		if !ok {
			t.Fatalf("path %q missing get", path)
		}
		responses, _ := get["responses"].(map[string]any)
		r200, _ := responses["200"].(map[string]any)
		schema, _ := r200["schema"].(map[string]any)
		ref, _ := schema["$ref"].(string)
		if ref != want {
			t.Fatalf("path %q 200 schema ref = %q want %q", path, ref, want)
		}
	}
}

func TestOpenAPIExtendedModelsExposeExpectedDataSections(t *testing.T) {
	raw := docs.SwaggerInfo.ReadDoc()
	var root map[string]any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		t.Fatalf("parse openapi: %v", err)
	}
	defs, _ := root["definitions"].(map[string]any)
	expectProps := map[string][]string{
		"main.SummaryV2Data":       {"schema_version", "car", "range", "units", "overview", "driving", "charging", "parking", "battery", "efficiency", "cost", "quality", "state", "generated_at"},
		"main.DashboardV2Data":     {"car_id", "range", "overview", "statistics"},
		"main.RealtimeV2Data":      {"car_id", "current"},
		"main.SeriesV2Data":        {"car_id", "scope", "bucket", "range", "metrics", "points"},
		"main.LocationsV2Data":     {"car_id", "range", "summary", "locations"},
		"main.LocationAggregateV2": {"name", "latitude", "longitude", "drive_start_count", "drive_end_count", "drive_count", "charge_count", "charge_energy_kwh", "charge_cost", "total_event_count", "last_seen"},
	}
	for defName, props := range expectProps {
		def, ok := defs[defName].(map[string]any)
		if !ok {
			t.Fatalf("missing definition %s", defName)
		}
		gotProps, _ := def["properties"].(map[string]any)
		for _, prop := range props {
			if gotProps[prop] == nil {
				t.Fatalf("%s missing property %q", defName, prop)
			}
		}
	}
}

func TestDocsRoutesReturnContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	docsui.RegisterRoutes(r.Group("/api/v1"), "/api/v1")

	for _, path := range []string{"/api/v1/docs/openapi.json", "/api/v1/docs/swagger/doc.json"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, w.Code)
		}
		if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("%s content-type %q", path, w.Header().Get("Content-Type"))
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("docs html: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	body := w.Body.String()
	if !strings.Contains(body, "TeslaMateApi") || !strings.Contains(body, "api-reference") {
		t.Fatal("scalar html missing expected markers")
	}
}

func TestDocsRouteAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	docsui.RegisterRoutes(r.Group("/api/v1"), "/api/v1")

	for _, path := range []string{"/api/v1/docs", "/api/v1/docs/openapi.json", "/api/v1/docs/swagger/doc.json"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, w.Code)
		}
	}
}
