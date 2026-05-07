package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeV2SummaryBuilder struct {
	response V2SummaryResponse
	quality  V2DataQuality
	err      error
}

func (b fakeV2SummaryBuilder) BuildSummary(context.Context, string, V2TimeRange) (V2SummaryResponse, V2DataQuality, error) {
	return b.response, b.quality, b.err
}

type fakeV2DrivingBuilder struct {
	drivingResponse      V2DrivingResponse
	timeseriesResponse   V2DrivingTimeseriesResponse
	distributionResponse V2DrivingDistributionResponse
	rankingResponse      V2DrivingRankingResponse
	quality              V2DataQuality
	carID                int64
	err                  error
}

func (b fakeV2DrivingBuilder) BuildDriving(context.Context, string, V2TimeRange) (V2DrivingResponse, V2DataQuality, int64, error) {
	return b.drivingResponse, b.quality, b.carID, b.err
}

func (b fakeV2DrivingBuilder) BuildTimeseries(context.Context, string, V2TimeRange, string) (V2DrivingTimeseriesResponse, V2DataQuality, int64, error) {
	return b.timeseriesResponse, b.quality, b.carID, b.err
}

func (b fakeV2DrivingBuilder) BuildDistribution(context.Context, string, V2TimeRange, string) (V2DrivingDistributionResponse, V2DataQuality, int64, error) {
	return b.distributionResponse, b.quality, b.carID, b.err
}

func (b fakeV2DrivingBuilder) BuildRanking(context.Context, string, V2TimeRange, string, int) (V2DrivingRankingResponse, V2DataQuality, int64, error) {
	return b.rankingResponse, b.quality, b.carID, b.err
}

func TestV2InfoHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	handlers := NewV2Handlers(fakeV2SummaryBuilder{}, nil)
	v2 := api.Group("/v2")
	v2.GET("", handlers.Info)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var payload struct {
		Data V2InfoResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Data.Version != "v2" || payload.Data.Scope != "analytics" {
		t.Fatalf("unexpected payload: %#v", payload.Data)
	}
}

func TestV2SummaryHandlerRejectsInvalidPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := NewV2Handlers(fakeV2SummaryBuilder{}, nil)
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/summary", handlers.Summary)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/summary?period=hour", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}

func TestV2SummaryHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := NewV2Handlers(fakeV2SummaryBuilder{
		response: V2SummaryResponse{
			Summary: V2Summary{
				Driving: V2DrivingSummary{DriveCount: 1, DistanceKM: 12.5},
			},
		},
		quality: V2DataQuality{Complete: true, SampleCount: 1},
	}, nil)
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/summary", handlers.Summary)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/summary?period=custom&start=2026-05-01T00:00:00Z&end=2026-05-02T00:00:00Z", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload V2SummaryAPIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Data.Summary.Driving.DriveCount != 1 || payload.Meta.CarID != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestV2DrivingHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := NewV2Handlers(nil, fakeV2DrivingBuilder{
		drivingResponse: V2DrivingResponse{
			Summary: V2DrivingAnalyticsSummary{DriveCount: 2, DistanceKM: 42},
		},
		quality: V2DataQuality{Complete: true, SampleCount: 2},
		carID:   1,
	})
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/driving", handlers.Driving)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/driving?period=custom&start=2026-05-01T00:00:00Z&end=2026-05-02T00:00:00Z", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload V2DrivingAPIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Data.Summary.DriveCount != 2 || payload.Meta.CarID != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestV2DrivingDistributionHandlerRejectsInvalidDimension(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := NewV2Handlers(nil, fakeV2DrivingBuilder{err: errV2InvalidDrivingDimension})
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/driving/distribution", handlers.DrivingDistribution)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/driving/distribution?dimension=bad", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestV2DrivingRankingHandlerRejectsInvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := NewV2Handlers(nil, fakeV2DrivingBuilder{err: errV2InvalidDrivingRanking})
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/driving/ranking", handlers.DrivingRanking)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/driving/ranking?type=bad", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestBuildOpenAPISpecIncludesV2Summary(t *testing.T) {
	spec := buildOpenAPISpec()
	paths, ok := spec["paths"].(gin.H)
	if !ok {
		t.Fatal("paths missing from spec")
	}
	if _, ok := paths["/v2/cars/{CarID}/analytics/summary"]; !ok {
		t.Fatal("v2 summary path missing from spec")
	}
	for _, path := range []string{
		"/v2/cars/{CarID}/analytics/driving",
		"/v2/cars/{CarID}/analytics/driving/timeseries",
		"/v2/cars/{CarID}/analytics/driving/distribution",
		"/v2/cars/{CarID}/analytics/driving/ranking",
	} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("driving path missing from spec: %s", path)
		}
	}
}

func TestDocsRoutesServeSwaggerAndScalar(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	RegisterDocsRoutes(api)

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
	if body := scalarRecorder.Body.String(); !strings.Contains(body, "TeslaMateApi Reference") || !strings.Contains(body, "/v2/cars/{CarID}/analytics/summary") || !strings.Contains(body, "/v2/cars/{CarID}/analytics/driving/ranking") {
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
