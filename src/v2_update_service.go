package main

import (
	"context"
)

type V2UpdateRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Updates(ctx context.Context, carID int64, start, end timeBound) (V2UpdateAnalyticsResponse, int64, error)
}

type V2UpdateService struct {
	repository V2UpdateRepository
}

func NewV2UpdateService(repository V2UpdateRepository) V2UpdateService {
	return V2UpdateService{repository: repository}
}

func (s V2UpdateService) BuildUpdates(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2UpdateAnalyticsResponse, int64, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2UpdateAnalyticsResponse{}, 0, err
	}

	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return V2UpdateAnalyticsResponse{}, 0, err
	}
	if !exists {
		return V2UpdateAnalyticsResponse{}, 0, errV2CarNotFound
	}

	response, _, err := s.repository.Updates(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End))
	if err != nil {
		return V2UpdateAnalyticsResponse{}, 0, err
	}

	if response.Versions == nil {
		response.Versions = []V2UpdateVersion{}
	}

	return response, carID, nil
}
