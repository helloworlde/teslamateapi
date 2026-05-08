package main

import (
	"context"
	"strings"
)

type V2LifecycleRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Lifecycle(ctx context.Context, carID int64) (V2LifecycleResponse, error)
	Timeline(ctx context.Context, carID int64, start, end timeBound, eventTypes []string, limit int, order string) ([]V2TimelineEvent, int64, error)
}

type V2LifecycleService struct {
	repository V2LifecycleRepository
}

func NewV2LifecycleService(repository V2LifecycleRepository) V2LifecycleService {
	return V2LifecycleService{repository: repository}
}

func (s V2LifecycleService) BuildLifecycle(ctx context.Context, carIDParam string, timeRange V2TimeRange) (V2LifecycleResponse, V2DataQuality, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2LifecycleResponse{}, V2DataQuality{}, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2LifecycleResponse{}, V2DataQuality{}, err
	}
	response, err := s.repository.Lifecycle(ctx, carID)
	if err != nil {
		return V2LifecycleResponse{}, V2DataQuality{}, err
	}
	quality := V2DataQuality{
		Complete:    true,
		SampleCount: response.DriveCount + response.ChargingSessionCount + response.UpdateCount,
	}
	return response, quality, nil
}

func (s V2LifecycleService) BuildTimeline(ctx context.Context, carIDParam string, timeRange V2TimeRange, eventTypes []string, limit int, order string) (V2TimelineResponse, V2DataQuality, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2TimelineResponse{}, V2DataQuality{}, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2TimelineResponse{}, V2DataQuality{}, err
	}
	if order != "asc" && order != "desc" && order != "" {
		order = "desc"
	}
	if order == "" {
		order = "desc"
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	events, total, err := s.repository.Timeline(ctx, carID, asTimeBound(timeRange.Start), asTimeBound(timeRange.End), eventTypes, limit, order)
	if err != nil {
		return V2TimelineResponse{}, V2DataQuality{}, err
	}
	if events == nil {
		events = []V2TimelineEvent{}
	}
	quality := V2DataQuality{
		Complete:    true,
		SampleCount: total,
	}
	return V2TimelineResponse{Events: events, Total: total}, quality, nil
}

func (s V2LifecycleService) ensureCarExists(ctx context.Context, carID int64) error {
	exists, err := s.repository.CarExists(ctx, carID)
	if err != nil {
		return err
	}
	if !exists {
		return errV2CarNotFound
	}
	return nil
}

func parseEventTypes(param string) []string {
	if param == "" {
		return nil
	}
	parts := strings.Split(param, ",")
	var types []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			types = append(types, p)
		}
	}
	return types
}
