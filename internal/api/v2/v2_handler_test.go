package v2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeV2SummaryBuilder struct {
	response V2SummaryResponse
	err      error
}

func (b fakeV2SummaryBuilder) BuildSummary(context.Context, string, V2TimeRange) (V2SummaryResponse, error) {
	return b.response, b.err
}

type fakeV2DrivingBuilder struct {
	drivingResponse    V2DrivingResponse
	timeseriesResponse V2DrivingTimeseriesResponse
	carID              int64
	err                error
}

func (b fakeV2DrivingBuilder) BuildDriving(context.Context, string, V2TimeRange) (V2DrivingResponse, int64, error) {
	return b.drivingResponse, b.carID, b.err
}

func (b fakeV2DrivingBuilder) BuildTimeseries(context.Context, string, V2TimeRange, string) (V2DrivingTimeseriesResponse, int64, error) {
	return b.timeseriesResponse, b.carID, b.err
}

type fakeV2ChargingBuilder struct {
	chargingResponse V2ChargingResponse
	carID            int64
	err              error
}

func (b fakeV2ChargingBuilder) BuildCharging(context.Context, string, V2TimeRange, V2ChargingBuildOptions) (V2ChargingResponse, int64, error) {
	return b.chargingResponse, b.carID, b.err
}

type fakeV2ParkingBuilder struct {
	parkingResponse V2ParkingResponse
	carID           int64
	err             error
}

func (b fakeV2ParkingBuilder) BuildParking(context.Context, string, V2TimeRange, V2ParkingBuildOptions) (V2ParkingResponse, int64, error) {
	return b.parkingResponse, b.carID, b.err
}

type fakeV2BatteryBuilder struct {
	batteryResponse    V2BatteryResponse
	timeseriesResponse V2BatteryTimeseriesResponse
	carID              int64
	err                error
}

func (b fakeV2BatteryBuilder) BuildBattery(context.Context, string, V2TimeRange) (V2BatteryResponse, int64, error) {
	return b.batteryResponse, b.carID, b.err
}

func (b fakeV2BatteryBuilder) BuildBatteryTimeseries(context.Context, string, V2TimeRange, string) (V2BatteryTimeseriesResponse, int64, error) {
	return b.timeseriesResponse, b.carID, b.err
}

type fakeV2CostBuilder struct {
	costResponse V2CostResponse
	carID        int64
	err          error
}

func (b fakeV2CostBuilder) BuildCost(context.Context, string, V2TimeRange, string) (V2CostResponse, int64, error) {
	return b.costResponse, b.carID, b.err
}

func TestV2CapabilitiesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	handlers := NewV2Handlers(fakeV2SummaryBuilder{}, nil)
	v2 := api.Group("/v2")
	v2.GET("/capabilities", handlers.Capabilities)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/capabilities", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var payload struct {
		Data V2CapabilitiesResponse `json:"data"` // 响应数据
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Data.Version != "v2" {
		t.Fatalf("unexpected version: %#v", payload.Data)
	}
	if len(payload.Data.Domains) == 0 {
		t.Fatalf("expected non-empty domains: %#v", payload.Data)
	}
	var summaryDomain *V2CapabilitiesDomain
	for i := range payload.Data.Domains {
		if payload.Data.Domains[i].Name == "summary" {
			summaryDomain = &payload.Data.Domains[i]
			break
		}
	}
	if summaryDomain == nil || !summaryDomain.SupportsCompare {
		t.Fatalf("expected summary domain to support compare: %#v", payload.Data.Domains)
	}
	if got := payload.Data.BreakdownOptions["charging"]; len(got) != 2 {
		t.Fatalf("expected charging breakdown options, got %#v", got)
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
				Driving: V2DrivingSummary{DriveCount: 1, Distance: 12.5},
			},
		},
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
			Summary: V2DrivingSummary{DriveCount: 2, Distance: 42},
		},
		carID: 1,
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

func TestV2ChargingHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := NewV2Handlers(nil, nil, fakeV2ChargingBuilder{
		chargingResponse: V2ChargingResponse{
			Summary: V2ChargingSummary{SessionCount: 2, EnergyAdded: 42},
		},
		carID: 1,
	})
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/charging", handlers.Charging)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/charging?period=custom&start=2026-05-01T00:00:00Z&end=2026-05-02T00:00:00Z", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload V2ChargingAPIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Data.Summary.SessionCount != 2 || payload.Meta.CarID != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestV2ParkingHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := NewV2Handlers(nil, nil)
	handlers.parkingBuilder = fakeV2ParkingBuilder{
		parkingResponse: V2ParkingResponse{
			Summary: V2ParkingSummary{ParkingSessionCount: 2, ParkedDuration: 7200},
		},
		carID: 1,
	}
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/parking", handlers.Parking)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/parking?period=custom&start=2026-05-01T00:00:00Z&end=2026-05-02T00:00:00Z", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload V2ParkingAPIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Data.Summary.ParkingSessionCount != 2 || payload.Meta.CarID != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestV2BatteryHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	latestLevel := int64(80)
	estimatedRated := 430.0
	handlers := NewV2Handlers(nil, nil)
	handlers.batteryBuilder = fakeV2BatteryBuilder{
		batteryResponse: V2BatteryResponse{
			Summary: V2BatterySummary{
				LatestLevel:       &latestLevel,
				RangeAtFullCharge: &V2BatteryRange{Rated: &estimatedRated},
			},
		},
		carID: 1,
	}
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/battery", handlers.Battery)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/battery?period=custom&start=2026-05-01T00:00:00Z&end=2026-05-02T00:00:00Z", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload V2BatteryAPIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Data.Summary.LatestLevel == nil || *payload.Data.Summary.LatestLevel != 80 || payload.Meta.CarID != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestV2CostHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	chargingCost := 30.0
	costPerEnergy := 1.5
	handlers := NewV2Handlers(nil, nil)
	handlers.costBuilder = fakeV2CostBuilder{
		costResponse: V2CostResponse{
			Summary: V2CostSummaryDetails{
				ChargingCost:  &chargingCost,
				EnergyUsed:    float64Ptr(20),
				Distance:      100,
				CostPerEnergy: &costPerEnergy,
			},
		},
		carID: 1,
	}
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/cost", handlers.Cost)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/cost?period=custom&start=2026-05-01T00:00:00Z&end=2026-05-02T00:00:00Z", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload V2CostAPIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Data.Summary.ChargingCost == nil || payload.Meta.CarID != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestV2CostRejectsInvalidGroupBy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := NewV2Handlers(nil, nil)
	handlers.costBuilder = fakeV2CostBuilder{err: errV2InvalidDrivingGroupBy}
	handlers.now = func() time.Time { return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC) }
	router.GET("/api/v2/cars/:CarID/analytics/cost", handlers.Cost)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v2/cars/1/analytics/cost?group_by=hour", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
}
