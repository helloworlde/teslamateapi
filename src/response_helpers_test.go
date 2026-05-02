package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSanitizedRequestURIRedactsTokenQueryValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cars/1/command?token=secret&access_token=a&refresh_token=r&keep=value", nil)

	uri := sanitizedRequestURI(c)
	if strings.Contains(uri, "secret") || strings.Contains(uri, "access_token=a") || strings.Contains(uri, "refresh_token=r") {
		t.Fatalf("expected token values to be redacted, got %q", uri)
	}
	if !strings.Contains(uri, "keep=value") {
		t.Fatalf("expected unrelated query value to be preserved, got %q", uri)
	}
}

func TestWriteV1ErrorHidesInternalDetailsForServerErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cars/1/summary", nil)

	writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load summary", map[string]any{"reason": "select * from private.tokens failed"})

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	var body v1ErrorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if body.Error.Details != nil {
		t.Fatalf("expected internal details to be omitted, got %#v", body.Error.Details)
	}
}
