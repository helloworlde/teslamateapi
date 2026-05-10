package main

import "context"

type V2ParkingRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, timeRange V2TimeRange) (V2ParkingAnalyticsSummary, V2ParkingStats, error)
	Locations(ctx context.Context, carID int64, timeRange V2TimeRange) ([]V2ParkingLocationItem, V2ParkingStats, error)
	StateBreakdown(ctx context.Context, carID int64, timeRange V2TimeRange) (V2ParkingStatesResponse, V2ParkingStats, error)
}

type V2ParkingStats struct {
	StateRows    int64
	SessionRows  int64
	PositionRows int64
}

type V2ParkingService struct {
	repository V2ParkingRepository
}

func NewV2ParkingService(repository V2ParkingRepository) V2ParkingService {
	return V2ParkingService{repository: repository}
}

func (s V2ParkingService) BuildParking(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ParkingResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ParkingResponse{}, 0, err
	}
	if timeRange.Compare == "previous_year" || timeRange.Compare == "lifetime_average" {
		return V2ParkingResponse{}, 0, errV2CompareUnsupported
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ParkingResponse{}, 0, err
	}
	summary, _, err := s.repository.Summary(ctx, carID, timeRange)
	if err != nil {
		return V2ParkingResponse{}, 0, err
	}
	response := V2ParkingResponse{Summary: summary}
	if timeRange.Compare == "previous_period" && timeRange.PreviousStart != nil && timeRange.PreviousEnd != nil {
		previousRange := timeRange
		previousRange.Start = *timeRange.PreviousStart
		previousRange.End = *timeRange.PreviousEnd
		previous, _, err := s.repository.Summary(ctx, carID, previousRange)
		if err != nil {
			return V2ParkingResponse{}, 0, err
		}
		response.Comparison = buildV2ParkingComparison(summary, previous)
	}
	return response, carID, nil
}

func (s V2ParkingService) BuildParkingLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ParkingLocationsResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ParkingLocationsResponse{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ParkingLocationsResponse{}, 0, err
	}
	items, _, err := s.repository.Locations(ctx, carID, timeRange)
	if err != nil {
		return V2ParkingLocationsResponse{}, 0, err
	}
	return V2ParkingLocationsResponse{Items: items}, carID, nil
}

func (s V2ParkingService) BuildParkingStates(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ParkingStatesResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ParkingStatesResponse{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ParkingStatesResponse{}, 0, err
	}
	response, _, err := s.repository.StateBreakdown(ctx, carID, timeRange)
	if err != nil {
		return V2ParkingStatesResponse{}, 0, err
	}
	return response, carID, nil
}

func (s V2ParkingService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}

func buildV2ParkingComparison(current V2ParkingAnalyticsSummary, previous V2ParkingAnalyticsSummary) map[string]V2ComparisonValue {
	return map[string]V2ComparisonValue{
		"parking_session_count": compareFloat(float64(current.ParkingSessionCount), float64(previous.ParkingSessionCount)),
		"parked_duration_min":   compareFloat(current.ParkedDurationMin, previous.ParkedDurationMin),
		"asleep_duration_min":   compareFloat(current.AsleepDurationMin, previous.AsleepDurationMin),
		"online_duration_min":   compareFloat(current.OnlineDurationMin, previous.OnlineDurationMin),
		"offline_duration_min":  compareFloat(current.OfflineDurationMin, previous.OfflineDurationMin),
	}
}
