package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	docs "github.com/tobiasehlert/teslamateapi/src/docs"
)

func TestOpenAPIDocumentContainsStatisticsAndInsights(t *testing.T) {
	s := docs.SwaggerInfo.ReadDoc()
	for _, sub := range []string{
		"/v1/cars/{CarID}/dashboard",
		"/v1/cars/{CarID}/calendar",
		"/v1/cars/{CarID}/statistics",
		"/v1/cars/{CarID}/series",
		"/v1/cars/{CarID}/distributions",
		"/v1/cars/{CarID}/timeline",
		"/v1/cars/{CarID}/insights",
	} {
		if !strings.Contains(s, sub) {
			t.Fatalf("OpenAPI doc missing %q", sub)
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

func TestDocsRoutesReturnContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/docs", serveScalarAPIReference)
	r.GET("/api/v1/docs/openapi.json", serveOpenAPIDocumentJSON)
	r.GET("/api/v1/docs/swagger/doc.json", serveSwaggerDocJSON)

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
	registerDocsRoutes(r.Group("/api/v1"), "/api/v1")

	for _, path := range []string{"/api/v1/docs", "/api/v1/docs/openapi.json", "/api/v1/docs/swagger/doc.json"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, w.Code)
		}
	}
}
