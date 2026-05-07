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
		"/v1/cars",
		"/v1/cars/{CarID}/charges",
		"/v1/cars/{CarID}/drives",
		"/v1/cars/{CarID}/status",
		"/v2/cars/{CarID}/stats",
		"/v2/cars/{CarID}/activity",
		"/v2/cars/{CarID}/series",
		"/v2/cars/{CarID}/distributions",
		"/v2/cars/{CarID}/locations",
		"/v2/cars/{CarID}/locations/heatmap",
		"/v2/cars/{CarID}/analysis/insights",
		"/v2/cars/{CarID}/analysis/trends",
		"/v2/cars/{CarID}/analysis/records",
	} {
		if !strings.Contains(s, sub) {
			t.Fatalf("OpenAPI doc missing %q", sub)
		}
	}
}

func TestOpenAPIDocumentDoesNotExposeExtendedV1Routes(t *testing.T) {
	s := docs.SwaggerInfo.ReadDoc()
	for _, sub := range []string{
		"/v1/cars/{CarID}/stats",
		"/v1/cars/{CarID}/activity",
		"/v1/cars/{CarID}/series",
		"/v1/cars/{CarID}/distributions",
		"/v1/cars/{CarID}/analysis/insights",
		"/v1/cars/{CarID}/analysis/trends",
		"/v1/cars/{CarID}/analysis/records",
		"/v2/cars/{CarID}/status",
		"/v2/cars/{CarID}/drives",
		"/v2/cars/{CarID}/charges",
	} {
		if strings.Contains(s, sub) {
			t.Fatalf("OpenAPI doc exposes removed route %q", sub)
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
		"/v2/cars/{CarID}/stats":             "#/definitions/main.StatsV2Envelope",
		"/v2/cars/{CarID}/activity":          "#/definitions/main.ActivityV2Envelope",
		"/v2/cars/{CarID}/series":            "#/definitions/main.SeriesV2Envelope",
		"/v2/cars/{CarID}/distributions":     "#/definitions/main.DistributionsV2Envelope",
		"/v2/cars/{CarID}/locations":         "#/definitions/main.LocationsV2Envelope",
		"/v2/cars/{CarID}/locations/heatmap": "#/definitions/main.VisitedMapV2Envelope",
		"/v2/cars/{CarID}/analysis/insights": "#/definitions/main.InsightsV2Envelope",
		"/v2/cars/{CarID}/analysis/trends":   "#/definitions/main.TrendsV2Envelope",
		"/v2/cars/{CarID}/analysis/records":  "#/definitions/main.RecordsV2Envelope",
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
		"main.StatsV2Data":         {"period", "range", "drives", "charges", "battery", "parking", "odometer", "generated_at"},
		"main.ActivityV2Data":      {"car_id", "range", "bucket", "summary", "items"},
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
	docsui.RegisterRoutes(r.Group("/api/v2"), "/api/v2")

	for _, path := range []string{"/api/v1/docs/openapi.json", "/api/v1/docs/swagger/doc.json", "/api/v2/docs/openapi.json", "/api/v2/docs/swagger/doc.json"} {
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
	docsui.RegisterRoutes(r.Group("/api/v2"), "/api/v2")

	for _, path := range []string{"/api/v1/docs", "/api/v1/docs/openapi.json", "/api/v1/docs/swagger/doc.json", "/api/v2/docs", "/api/v2/docs/openapi.json", "/api/v2/docs/swagger/doc.json"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, w.Code)
		}
	}
}
