package main

import (
	"context"
	"time"
)

type V2CalendarRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	DailyDriving(ctx context.Context, carID int64, start, end timeBound, location *time.Location) (map[string]V2CalendarDay, error)
	DailyCharging(ctx context.Context, carID int64, start, end timeBound, location *time.Location) (map[string]V2CalendarDay, error)
	DailyUpdates(ctx context.Context, carID int64, start, end timeBound, location *time.Location) (map[string]int64, error)
}

// @name V2CalendarService
type V2CalendarService struct {
	repository V2CalendarRepository
}

func NewV2CalendarService(repository V2CalendarRepository) V2CalendarService {
	return V2CalendarService{repository: repository}
}

func (s V2CalendarService) BuildCalendar(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2CalendarResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2CalendarResponse{}, 0, err
	}

	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return V2CalendarResponse{}, 0, err
	}
	if !exists {
		return V2CalendarResponse{}, 0, errV2CarNotFound
	}

	location := timeRangeLocation(timeRange)
	start := asTimeBound(timeRange.Start)
	end := asTimeBound(timeRange.End)

	drivingData, err := s.repository.DailyDriving(ctx, carID, start, end, location)
	if err != nil {
		return V2CalendarResponse{}, 0, err
	}
	chargingData, err := s.repository.DailyCharging(ctx, carID, start, end, location)
	if err != nil {
		return V2CalendarResponse{}, 0, err
	}
	updatesData, err := s.repository.DailyUpdates(ctx, carID, start, end, location)
	if err != nil {
		return V2CalendarResponse{}, 0, err
	}

	// Generate all dates in range
	days := generateDateRange(timeRange.Start, timeRange.End, location)
	var calendarDays []V2CalendarDay

	for _, dateStr := range days {
		day := V2CalendarDay{Date: dateStr}

		if d, ok := drivingData[dateStr]; ok {
			day.DriveCount = d.DriveCount
			day.DistanceKM = d.DistanceKM
			day.DriveDurationMin = d.DriveDurationMin
		}
		if c, ok := chargingData[dateStr]; ok {
			day.ChargingSessionCount = c.ChargingSessionCount
			day.EnergyAddedKWh = c.EnergyAddedKWh
			day.ChargingCost = c.ChargingCost
		}
		if count, ok := updatesData[dateStr]; ok {
			day.UpdateCount = count
		}

		day.ActivityLevel = computeActivityLevel(day)
		calendarDays = append(calendarDays, day)
	}

	return V2CalendarResponse{Days: calendarDays}, carID, nil
}

func generateDateRange(start, end time.Time, location *time.Location) []string {
	var dates []string
	// Iterate day by day in local timezone
	current := time.Date(start.In(location).Year(), start.In(location).Month(), start.In(location).Day(), 0, 0, 0, 0, location)
	endLocal := end.In(location)

	for current.Before(endLocal) {
		dates = append(dates, current.Format("2006-01-02"))
		current = current.AddDate(0, 0, 1)
	}
	return dates
}

func computeActivityLevel(day V2CalendarDay) V2ActivityLevel {
	return V2ActivityLevel{
		Driving:      drivingActivityLevel(day.DistanceKM),
		Charging:     chargingActivityLevel(day.ChargingSessionCount),
		ParkingDrain: parkingDrainActivityLevel(day.VampireDrainPercent),
	}
}

func drivingActivityLevel(distanceKM float64) int {
	switch {
	case distanceKM <= 0:
		return 0
	case distanceKM <= 30:
		return 1
	case distanceKM <= 100:
		return 2
	case distanceKM <= 200:
		return 3
	default:
		return 4
	}
}

func chargingActivityLevel(sessions int64) int {
	switch {
	case sessions == 0:
		return 0
	case sessions == 1:
		return 1
	case sessions == 2:
		return 2
	case sessions == 3:
		return 3
	default:
		return 4
	}
}

func parkingDrainActivityLevel(drainPercent float64) int {
	switch {
	case drainPercent <= 0:
		return 0
	case drainPercent < 1:
		return 1
	case drainPercent < 2:
		return 2
	case drainPercent < 5:
		return 3
	default:
		return 4
	}
}
