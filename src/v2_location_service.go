package main

import (
	"context"
	"errors"
)

var errV2InvalidLocationSort = errors.New("invalid location sort")

type V2LocationRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Locations(ctx context.Context, carID int64, timeRange V2TimeRange, sort string) ([]V2LocationAnalyticsItem, V2LocationStats, error)
}

// @name V2LocationStats
type V2LocationStats struct {
	DriveStartRows int64
	DriveEndRows   int64
	ChargingRows   int64
	ParkingRows    int64
}

// @name V2LocationService
type V2LocationService struct {
	repository V2LocationRepository
}

func NewV2LocationService(repository V2LocationRepository) V2LocationService {
	return V2LocationService{repository: repository}
}

func (s V2LocationService) BuildLocations(ctx context.Context, carIDParam string, timeRange V2TimeRange, sort string) (V2LocationAnalyticsResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2LocationAnalyticsResponse{}, 0, err
	}
	if sort == "" {
		sort = "parking_duration_desc"
	}
	if _, err := v2LocationOrderBy(sort); err != nil {
		return V2LocationAnalyticsResponse{}, 0, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2LocationAnalyticsResponse{}, 0, err
	}
	items, _, err := s.repository.Locations(ctx, carID, timeRange, sort)
	if err != nil {
		return V2LocationAnalyticsResponse{}, 0, err
	}
	return V2LocationAnalyticsResponse{Sort: sort, Items: items}, carID, nil
}

func (s V2LocationService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}
