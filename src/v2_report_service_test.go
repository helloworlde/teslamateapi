package main

import (
	"context"
	"testing"
)

// Mock builders for report tests

type mockSummaryBuilderForReport struct {
	response V2SummaryResponse
	quality  V2DataQuality
	err      error
}

func (m *mockSummaryBuilderForReport) BuildSummary(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2SummaryResponse, V2DataQuality, error) {
	return m.response, m.quality, m.err
}

type mockDrivingBuilderForReport struct {
	response V2DrivingResponse
	quality  V2DataQuality
	carID    int64
	err      error
}

func (m *mockDrivingBuilderForReport) BuildDriving(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2DrivingResponse, V2DataQuality, int64, error) {
	return m.response, m.quality, m.carID, m.err
}

func (m *mockDrivingBuilderForReport) BuildTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2DrivingTimeseriesResponse, V2DataQuality, int64, error) {
	return V2DrivingTimeseriesResponse{}, V2DataQuality{}, 0, nil
}

func (m *mockDrivingBuilderForReport) BuildDistribution(ctx context.Context, carIDParam string, timeRange V2TimeRange, dimension string) (V2DrivingDistributionResponse, V2DataQuality, int64, error) {
	return V2DrivingDistributionResponse{}, V2DataQuality{}, 0, nil
}

func (m *mockDrivingBuilderForReport) BuildRanking(ctx context.Context, carIDParam string, timeRange V2TimeRange, rankingType string, limit int) (V2DrivingRankingResponse, V2DataQuality, int64, error) {
	return V2DrivingRankingResponse{}, V2DataQuality{}, 0, nil
}

type mockChargingBuilderForReport struct {
	response V2ChargingResponse
	quality  V2DataQuality
	carID    int64
	err      error
}

func (m *mockChargingBuilderForReport) BuildCharging(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingResponse, V2DataQuality, int64, error) {
	return m.response, m.quality, m.carID, m.err
}

func (m *mockChargingBuilderForReport) BuildChargingTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingTimeseriesResponse, V2DataQuality, int64, error) {
	return V2ChargingTimeseriesResponse{}, V2DataQuality{}, 0, nil
}

func (m *mockChargingBuilderForReport) BuildChargingLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingLocationsResponse, V2DataQuality, int64, error) {
	return V2ChargingLocationsResponse{}, V2DataQuality{}, 0, nil
}

func (m *mockChargingBuilderForReport) BuildChargingTypes(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ChargingTypesResponse, V2DataQuality, int64, error) {
	return V2ChargingTypesResponse{}, V2DataQuality{}, 0, nil
}

func (m *mockChargingBuilderForReport) BuildChargingCost(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2ChargingCostResponse, V2DataQuality, int64, error) {
	return V2ChargingCostResponse{}, V2DataQuality{}, 0, nil
}

type mockUpdateBuilderForReport struct {
	response V2UpdateAnalyticsResponse
	quality  V2DataQuality
	carID    int64
	err      error
}

func (m *mockUpdateBuilderForReport) BuildUpdates(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2UpdateAnalyticsResponse, V2DataQuality, int64, error) {
	return m.response, m.quality, m.carID, m.err
}

func TestV2ReportServiceBuildReport(t *testing.T) {
	version := "2024.12.1"
	service := NewV2ReportService(
		&mockSummaryBuilderForReport{
			response: V2SummaryResponse{
				Summary: V2Summary{
					Driving:  V2DrivingSummary{DriveCount: 10, DistanceKM: 500},
					Charging: V2ChargingSummary{SessionCount: 5, EnergyAddedKWh: 100},
				},
			},
			quality: V2DataQuality{Complete: true, SampleCount: 15},
		},
		&mockDrivingBuilderForReport{
			response: V2DrivingResponse{
				Summary: V2DrivingAnalyticsSummary{DriveCount: 10, DistanceKM: 500},
			},
			quality: V2DataQuality{Complete: true, SampleCount: 10},
			carID:   1,
		},
		&mockChargingBuilderForReport{
			response: V2ChargingResponse{
				Summary: V2ChargingAnalyticsSummary{SessionCount: 5, EnergyAddedKWh: 100},
			},
			quality: V2DataQuality{Complete: true, SampleCount: 5},
			carID:   1,
		},
		&mockUpdateBuilderForReport{
			response: V2UpdateAnalyticsResponse{
				UpdateCount:   2,
				LatestVersion: &version,
				Versions:      []V2UpdateVersion{},
			},
			quality: V2DataQuality{Complete: true, SampleCount: 2},
			carID:   1,
		},
		nil, nil, nil, nil, nil, nil,
	)

	response, quality, err := service.BuildReport(context.Background(), "1", V2TimeRange{Period: "month"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Period != "month" {
		t.Fatalf("expected period 'month', got %q", response.Period)
	}
	// All 4 modules with data
	if len(response.Sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(response.Sections))
	}
	if quality.SampleCount != 32 { // 15+10+5+2
		t.Fatalf("expected sample_count 32, got %d", quality.SampleCount)
	}
}

func TestV2ReportServiceSkipsEmptyModules(t *testing.T) {
	service := NewV2ReportService(
		&mockSummaryBuilderForReport{
			response: V2SummaryResponse{Summary: V2Summary{}},
			quality:  V2DataQuality{Complete: true},
		},
		&mockDrivingBuilderForReport{
			// DriveCount = 0, will be skipped
			response: V2DrivingResponse{Summary: V2DrivingAnalyticsSummary{DriveCount: 0}},
			quality:  V2DataQuality{Complete: true},
		},
		&mockChargingBuilderForReport{
			// SessionCount = 0, will be skipped
			response: V2ChargingResponse{Summary: V2ChargingAnalyticsSummary{SessionCount: 0}},
			quality:  V2DataQuality{Complete: true},
		},
		&mockUpdateBuilderForReport{
			// UpdateCount = 0, will be skipped
			response: V2UpdateAnalyticsResponse{UpdateCount: 0, Versions: []V2UpdateVersion{}},
			quality:  V2DataQuality{Complete: true},
		},
		nil, nil, nil, nil, nil, nil,
	)

	response, _, err := service.BuildReport(context.Background(), "1", V2TimeRange{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only summary section (always included), driving/charging/updates skipped (zero data)
	if len(response.Sections) != 1 {
		t.Fatalf("expected 1 section (summary only), got %d", len(response.Sections))
	}
}

func TestV2ReportServiceIncludeFilter(t *testing.T) {
	service := NewV2ReportService(
		&mockSummaryBuilderForReport{
			response: V2SummaryResponse{Summary: V2Summary{Driving: V2DrivingSummary{DriveCount: 5}}},
			quality:  V2DataQuality{Complete: true, SampleCount: 5},
		},
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	response, _, err := service.BuildReport(context.Background(), "1", V2TimeRange{}, []string{"summary"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Sections) != 1 || response.Sections[0].Type != "summary" {
		t.Fatalf("expected only summary section, got %d sections", len(response.Sections))
	}
}

func TestV2ReportServiceInvalidCarID(t *testing.T) {
	service := NewV2ReportService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, _, err := service.BuildReport(context.Background(), "bad", V2TimeRange{}, nil)
	if err == nil || err.Error() != "invalid car id" {
		t.Fatalf("expected invalid car id, got %v", err)
	}
}
