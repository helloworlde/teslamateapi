package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	errV2CarNotFound        = errors.New("car not found")
	errV2CompareUnsupported = errors.New("compare mode is not implemented")
)

type V2SummaryService struct {
	repository V2SummaryRepository
}

func NewV2SummaryService(repository V2SummaryRepository) V2SummaryService {
	return V2SummaryService{repository: repository}
}

func (s V2SummaryService) BuildSummary(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2SummaryResponse, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return V2SummaryResponse{}, fmt.Errorf("invalid car id")
	}

	if timeRange.Compare == "previous_year" || timeRange.Compare == "lifetime_average" {
		return V2SummaryResponse{}, errV2CompareUnsupported
	}

	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return V2SummaryResponse{}, err
	}
	if !exists {
		return V2SummaryResponse{}, errV2CarNotFound
	}

	summary, _, err := s.repository.Summary(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End))
	if err != nil {
		return V2SummaryResponse{}, err
	}

	response := V2SummaryResponse{Summary: summary}

	if timeRange.Compare == "previous_period" && timeRange.PreviousStart != nil && timeRange.PreviousEnd != nil {
		previousSummary, _, err := s.repository.Summary(ctx, carID, asTimeBound(*timeRange.PreviousStart), asTimeBound(*timeRange.PreviousEnd))
		if err != nil {
			return V2SummaryResponse{}, err
		}
		response.Comparison = buildV2SummaryComparison(summary, previousSummary)
	}

	return response, nil
}

func asTimeBound(t time.Time) timeBound {
	return timeBound{Time: t.Format(time.RFC3339)}
}

func buildV2SummaryComparison(current V2Summary, previous V2Summary) map[string]V2ComparisonValue {
	return map[string]V2ComparisonValue{
		"driving.drive_count":       compareFloat(float64(current.Driving.DriveCount), float64(previous.Driving.DriveCount)),
		"driving.distance_km":       compareFloat(current.Driving.DistanceKM, previous.Driving.DistanceKM),
		"driving.duration_min":      compareFloat(current.Driving.DurationMin, previous.Driving.DurationMin),
		"charging.session_count":    compareFloat(float64(current.Charging.SessionCount), float64(previous.Charging.SessionCount)),
		"charging.energy_added_kwh": compareFloat(current.Charging.EnergyAddedKWh, previous.Charging.EnergyAddedKWh),
		"charging.energy_used_kwh":  compareFloat(current.Charging.EnergyUsedKWh, previous.Charging.EnergyUsedKWh),
		"charging.cost":             compareFloat(current.Charging.Cost, previous.Charging.Cost),
		"cost.charging_cost":        compareFloat(current.Cost.ChargingCost, previous.Cost.ChargingCost),
	}
}
