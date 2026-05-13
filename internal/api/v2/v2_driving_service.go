package v2

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

var (
	errV2InvalidDrivingGroupBy = errors.New("invalid driving group_by")
)

var v2AllowedDrivingGroupBy = map[string]bool{
	"day":   true,
	"week":  true,
	"month": true,
	"year":  true,
}

type V2DrivingRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Summary(ctx context.Context, carID int64, start timeBound, end timeBound) (V2DrivingSummary, V2DrivingStats, error)
	Timeseries(ctx context.Context, carID int64, timeRange V2TimeRange, groupBy string) ([]V2DrivingTimeseriesItem, V2DrivingStats, error)
}

// @name V2DrivingStats
type V2DrivingStats struct {
	DriveRows          int64
	EnergyEstimateRows int64
	TemperatureRows    int64
}

// @name V2DrivingService
type V2DrivingService struct {
	repository V2DrivingRepository
}

func NewV2DrivingService(repository V2DrivingRepository) V2DrivingService {
	return V2DrivingService{repository: repository}
}

func (s V2DrivingService) BuildDriving(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2DrivingResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2DrivingResponse{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2DrivingResponse{}, 0, err
	}

	summary, _, err := s.repository.Summary(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End))
	if err != nil {
		return V2DrivingResponse{}, 0, err
	}
	return V2DrivingResponse{Summary: summary}, carID, nil
}

func (s V2DrivingService) BuildTimeseries(ctx context.Context, carIDParam string, timeRange V2TimeRange, groupBy string) (V2DrivingTimeseriesResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2DrivingTimeseriesResponse{}, 0, err
	}
	if groupBy == "" {
		groupBy = defaultV2DrivingGroupBy(timeRange.Period)
	}
	if !v2AllowedDrivingGroupBy[groupBy] {
		return V2DrivingTimeseriesResponse{}, 0, errV2InvalidDrivingGroupBy
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2DrivingTimeseriesResponse{}, 0, err
	}

	items, _, err := s.repository.Timeseries(ctx, carID, timeRange, groupBy)
	if err != nil {
		return V2DrivingTimeseriesResponse{}, 0, err
	}
	return V2DrivingTimeseriesResponse{GroupBy: groupBy, Items: items}, carID, nil
}

func parseV2CarID(carIDParam string) (int64, error) {
	carID, err := strconv.ParseInt(carIDParam, 10, 64)
	if err != nil || carID <= 0 {
		return 0, fmt.Errorf("invalid car id")
	}
	return carID, nil
}

func (s V2DrivingService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}

func defaultV2DrivingGroupBy(period string) string {
	switch period {
	case "year":
		return "month"
	case "quarter":
		return "week"
	default:
		return "day"
	}
}

func v2LimitFromQuery(value string) int {
	if value == "" {
		return 10
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 10
	}
	return limit
}
