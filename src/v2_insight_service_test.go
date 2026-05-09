package main

import (
	"context"
	"testing"
	"time"
)

type mockDrivingBuilderForInsight struct {
	responses []V2DrivingResponse
	callCount int
}

func (m *mockDrivingBuilderForInsight) BuildDriving(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2DrivingResponse, int64, error) {
	idx := m.callCount
	m.callCount++
	if idx >= len(m.responses) {
		return V2DrivingResponse{}, 0, nil
	}
	return m.responses[idx], 1, nil
}

func (m *mockDrivingBuilderForInsight) BuildTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2DrivingTimeseriesResponse, int64, error) {
	return V2DrivingTimeseriesResponse{}, 0, nil
}

func (m *mockDrivingBuilderForInsight) BuildDistribution(ctx context.Context, carIDParam string, timeRange V2TimeRange, dimension string) (V2DrivingDistributionResponse, int64, error) {
	return V2DrivingDistributionResponse{}, 0, nil
}

func (m *mockDrivingBuilderForInsight) BuildRanking(ctx context.Context, carIDParam string, timeRange V2TimeRange, rankingType string, limit int) (V2DrivingRankingResponse, int64, error) {
	return V2DrivingRankingResponse{}, 0, nil
}

type mockChargingBuilderForInsight struct {
	responses []V2ChargingResponse
	callCount int
}

func (m *mockChargingBuilderForInsight) BuildCharging(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingResponse, int64, error) {
	idx := m.callCount
	m.callCount++
	if idx >= len(m.responses) {
		return V2ChargingResponse{}, 0, nil
	}
	return m.responses[idx], 1, nil
}

func (m *mockChargingBuilderForInsight) BuildChargingTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingTimeseriesResponse, int64, error) {
	return V2ChargingTimeseriesResponse{}, 0, nil
}

func (m *mockChargingBuilderForInsight) BuildChargingLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingLocationsResponse, int64, error) {
	return V2ChargingLocationsResponse{}, 0, nil
}

func (m *mockChargingBuilderForInsight) BuildChargingTypes(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingTypesResponse, int64, error) {
	return V2ChargingTypesResponse{}, 0, nil
}

func (m *mockChargingBuilderForInsight) BuildChargingCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingCostResponse, int64, error) {
	return V2ChargingCostResponse{}, 0, nil
}

func TestV2InsightServiceDrivingDistanceIncreased(t *testing.T) {
	consumption := 150.0
	service := NewV2InsightService(
		&mockDrivingBuilderForInsight{
			responses: []V2DrivingResponse{
				{Summary: V2DrivingAnalyticsSummary{DriveCount: 10, DistanceKM: 600, AvgConsumptionWhPerKM: &consumption}},
				{Summary: V2DrivingAnalyticsSummary{DriveCount: 8, DistanceKM: 500, AvgConsumptionWhPerKM: &consumption}},
			},
		},
		nil,
	)
	loc := time.UTC
	prevStart := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	prevEnd := time.Date(2024, 2, 1, 0, 0, 0, 0, loc)
	response, err := service.BuildInsights(context.Background(), "1", V2TimeRange{
		Start:         time.Date(2024, 2, 1, 0, 0, 0, 0, loc),
		End:           time.Date(2024, 3, 1, 0, 0, 0, 0, loc),
		PreviousStart: &prevStart,
		PreviousEnd:   &prevEnd,
		Compare:       "previous_period",
	}, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Total == 0 {
		t.Fatal("expected at least one insight")
	}
	found := false
	for _, ins := range response.Insights {
		if ins.ID == "driving_distance_increased" {
			found = true
			if ins.Category != "driving" {
				t.Errorf("expected category 'driving', got %q", ins.Category)
			}
		}
	}
	if !found {
		t.Fatalf("expected driving_distance_increased insight, got %+v", response.Insights)
	}
}

func TestV2InsightServiceInvalidCarID(t *testing.T) {
	service := NewV2InsightService(nil, nil)
	_, err := service.BuildInsights(context.Background(), "bad", V2TimeRange{}, "", "")
	if err == nil || err.Error() != "invalid car id" {
		t.Fatalf("expected invalid car id, got %v", err)
	}
}

func TestV2InsightServiceMinSeverityFilter(t *testing.T) {
	consumption := 150.0
	service := NewV2InsightService(
		&mockDrivingBuilderForInsight{
			responses: []V2DrivingResponse{
				// Small change < 10%, will be "info"
				{Summary: V2DrivingAnalyticsSummary{DriveCount: 10, DistanceKM: 505, AvgConsumptionWhPerKM: &consumption}},
				{Summary: V2DrivingAnalyticsSummary{DriveCount: 8, DistanceKM: 500, AvgConsumptionWhPerKM: &consumption}},
			},
		},
		nil,
	)
	loc := time.UTC
	prevStart := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	prevEnd := time.Date(2024, 2, 1, 0, 0, 0, 0, loc)
	response, err := service.BuildInsights(context.Background(), "1", V2TimeRange{
		Start:         time.Date(2024, 2, 1, 0, 0, 0, 0, loc),
		End:           time.Date(2024, 3, 1, 0, 0, 0, 0, loc),
		PreviousStart: &prevStart,
		PreviousEnd:   &prevEnd,
		Compare:       "previous_period",
	}, "", "warning")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Small change filtered out by min_severity=warning
	for _, ins := range response.Insights {
		if ins.Severity != "warning" {
			t.Errorf("expected only warning insights, got severity=%q", ins.Severity)
		}
	}
}

func TestV2InsightServiceNoData(t *testing.T) {
	service := NewV2InsightService(nil, nil)
	loc := time.UTC
	prevStart := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	prevEnd := time.Date(2024, 2, 1, 0, 0, 0, 0, loc)
	response, err := service.BuildInsights(context.Background(), "1", V2TimeRange{
		Start:         time.Date(2024, 2, 1, 0, 0, 0, 0, loc),
		End:           time.Date(2024, 3, 1, 0, 0, 0, 0, loc),
		PreviousStart: &prevStart,
		PreviousEnd:   &prevEnd,
	}, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Total != 0 {
		t.Fatalf("expected 0 insights, got %d", response.Total)
	}
}
