package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSwaggerSpecFromAnnotationsIncludesV2Analytics(t *testing.T) {
	spec := swaggerSpecForDocs()
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("paths missing from spec")
	}
	if _, ok := paths["/v2/cars/{CarID}/analytics/summary"]; !ok {
		t.Fatal("v2 summary path missing from spec")
	}
	for _, path := range []string{
		"/v2/cars/{CarID}/analytics/driving",
		"/v2/cars/{CarID}/analytics/driving/timeseries",
		"/v2/cars/{CarID}/analytics/charging",
		"/v2/cars/{CarID}/analytics/parking",
		"/v2/cars/{CarID}/analytics/battery",
		"/v2/cars/{CarID}/analytics/battery/timeseries",
		"/v2/cars/{CarID}/analytics/cost",
	} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("analytics path missing from spec: %s", path)
		}
	}
}

func TestDocsRoutesServeSwaggerAndScalar(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	RegisterRoutes(api)

	swaggerRecorder := httptest.NewRecorder()
	router.ServeHTTP(swaggerRecorder, httptest.NewRequest(http.MethodGet, "/api/docs/swagger.json", nil))
	if swaggerRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected swagger status: %d", swaggerRecorder.Code)
	}

	scalarRecorder := httptest.NewRecorder()
	router.ServeHTTP(scalarRecorder, httptest.NewRequest(http.MethodGet, "/api/docs/scalar", nil))
	if scalarRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected scalar status: %d body=%s", scalarRecorder.Code, scalarRecorder.Body.String())
	}
	if body := scalarRecorder.Body.String(); !strings.Contains(body, "TeslaMateApi Reference") || !strings.Contains(body, "/v2/cars/{CarID}/analytics/summary") || !strings.Contains(body, "/v2/cars/{CarID}/analytics/charging") || !strings.Contains(body, "/v2/cars/{CarID}/analytics/parking") || !strings.Contains(body, "/v2/cars/{CarID}/analytics/cost") {
		t.Fatalf("scalar body does not include expected content")
	}
	if body := scalarRecorder.Body.String(); strings.Contains(body, "cdn.jsdelivr.net") || !strings.Contains(body, scalarLocalScriptPath) {
		t.Fatalf("scalar body does not use the local script asset")
	}

	scriptRecorder := httptest.NewRecorder()
	router.ServeHTTP(scriptRecorder, httptest.NewRequest(http.MethodGet, scalarLocalScriptPath, nil))
	if scriptRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected scalar script status: %d", scriptRecorder.Code)
	}
	if contentType := scriptRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/javascript") {
		t.Fatalf("unexpected script content type: %s", contentType)
	}
	if body := scriptRecorder.Body.String(); strings.Contains(body, "https://cdn.jsdelivr.net/npm/@scalar/api-reference") || strings.Contains(body, "https://fonts.scalar.com") {
		t.Fatalf("local scalar script still references external UI assets")
	}

	fontRecorder := httptest.NewRecorder()
	router.ServeHTTP(fontRecorder, httptest.NewRequest(http.MethodGet, "/api/docs/assets/fonts/inter-latin.woff2", nil))
	if fontRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected scalar font status: %d", fontRecorder.Code)
	}
	if contentType := fontRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "font/woff2") {
		t.Fatalf("unexpected font content type: %s", contentType)
	}
}
