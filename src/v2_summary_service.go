package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var errV2CarNotFound = errors.New("car not found")

// @name V2SummaryService
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
		"driving.distance":          compareFloat(current.Driving.Distance, previous.Driving.Distance),
		"driving.duration":          compareFloat(current.Driving.Duration, previous.Driving.Duration),
		"charging.session_count":    compareFloat(float64(current.Charging.SessionCount), float64(previous.Charging.SessionCount)),
		"charging.energy_added":     compareFloat(current.Charging.EnergyAdded, previous.Charging.EnergyAdded),
		"charging.energy_used":      compareFloat(current.Charging.EnergyUsed, previous.Charging.EnergyUsed),
		"charging.cost":                  compareFloat(current.Charging.Cost, previous.Charging.Cost),
		"vehicle.tracked_consumption":    compareFloatPtr(current.Vehicle.TrackedConsumption, previous.Vehicle.TrackedConsumption),
		"vehicle.tracked_wall":           compareFloatPtr(current.Vehicle.TrackedWall, previous.Vehicle.TrackedWall),
		"vehicle.charge_efficiency":      compareFloatPtr(current.Vehicle.ChargeEfficiency, previous.Vehicle.ChargeEfficiency),
		"cost.charging_cost":             compareFloat(current.Cost.ChargingCost, previous.Cost.ChargingCost),
	}
}
