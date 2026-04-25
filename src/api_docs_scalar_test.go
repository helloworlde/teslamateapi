package main

import (
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
		"/v1/cars/{CarID}/summaries/statistics",
		"/v1/cars/{CarID}/insights/events",
	} {
		if !strings.Contains(s, sub) {
			t.Fatalf("OpenAPI doc missing %q", sub)
		}
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

func TestSummariesOptionsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/summaries/options", TeslaMateAPISummaryOptionsV1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/summaries/options", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"endpoints"`) || !strings.Contains(w.Body.String(), `/api/v1/summaries/options`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}
