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

func (s V2ParkingService) BuildParking(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ParkingResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ParkingResponse{}, V2DataQuality{}, 0, err
	}
	if timeRange.Compare == "previous_year" || timeRange.Compare == "lifetime_average" {
		return V2ParkingResponse{}, V2DataQuality{}, 0, errV2CompareUnsupported
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ParkingResponse{}, V2DataQuality{}, 0, err
	}
	summary, stats, err := s.repository.Summary(ctx, carID, timeRange)
	if err != nil {
		return V2ParkingResponse{}, V2DataQuality{}, 0, err
	}
	response := V2ParkingResponse{Summary: summary}
	if timeRange.Compare == "previous_period" && timeRange.PreviousStart != nil && timeRange.PreviousEnd != nil {
		previousRange := timeRange
		previousRange.Start = *timeRange.PreviousStart
		previousRange.End = *timeRange.PreviousEnd
		previous, _, err := s.repository.Summary(ctx, carID, previousRange)
		if err != nil {
			return V2ParkingResponse{}, V2DataQuality{}, 0, err
		}
		response.Comparison = buildV2ParkingComparison(summary, previous)
	}
	return response, buildV2ParkingDataQuality(stats), carID, nil
}

func (s V2ParkingService) BuildParkingLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ParkingLocationsResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ParkingLocationsResponse{}, V2DataQuality{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ParkingLocationsResponse{}, V2DataQuality{}, 0, err
	}
	items, stats, err := s.repository.Locations(ctx, carID, timeRange)
	if err != nil {
		return V2ParkingLocationsResponse{}, V2DataQuality{}, 0, err
	}
	quality := V2DataQuality{
		Complete:    false,
		SampleCount: stats.SessionRows,
	}
	if stats.SessionRows == 0 {
		quality.MissingFields = append(quality.MissingFields, "drives", "charging_processes")
	}
	quality.Warnings = append(quality.Warnings, "Parking locations are inferred from the previous drive or charging location when available.")
	quality.Warnings = append(quality.Warnings, "State durations and vampire drain cannot always be precisely attributed to individual locations.")
	return V2ParkingLocationsResponse{Items: items}, quality, carID, nil
}

func (s V2ParkingService) BuildParkingStates(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2ParkingStatesResponse, V2DataQuality, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2ParkingStatesResponse{}, V2DataQuality{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2ParkingStatesResponse{}, V2DataQuality{}, 0, err
	}
	response, stats, err := s.repository.StateBreakdown(ctx, carID, timeRange)
	if err != nil {
		return V2ParkingStatesResponse{}, V2DataQuality{}, 0, err
	}
	quality := V2DataQuality{Complete: true, SampleCount: stats.StateRows}
	if stats.StateRows == 0 {
		quality.Complete = false
		quality.MissingFields = append(quality.MissingFields, "states")
	}
	return response, quality, carID, nil
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

func buildV2ParkingDataQuality(stats V2ParkingStats) V2DataQuality {
	quality := V2DataQuality{Complete: true, SampleCount: stats.StateRows + stats.SessionRows}
	if stats.StateRows == 0 {
		quality.MissingFields = append(quality.MissingFields, "states")
	}
	if stats.SessionRows == 0 {
		quality.Warnings = append(quality.Warnings, "parking_session_count is inferred from gaps between drives and charging_processes and may be unavailable for sparse periods.")
	}
	if stats.PositionRows < 2 {
		quality.MissingFields = append(quality.MissingFields, "positions.battery_level", "positions.rated_battery_range_km")
		quality.Warnings = append(quality.Warnings, "vampire drain fields are only returned when non-driving, non-charging position samples exist at both ends of the period.")
	} else {
		quality.Warnings = append(quality.Warnings, "estimated_vampire_drain_kwh is derived from rated range loss and vehicle efficiency when available.")
	}
	if len(quality.MissingFields) > 0 || len(quality.Warnings) > 0 {
		quality.Complete = false
	}
	return quality
}
