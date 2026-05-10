package main

import (
	"context"
	"strings"
	"time"
)

type V2LifecycleRepository interface {
	CarExists(ctx context.Context, carID int64) (bool, error)
	Lifecycle(ctx context.Context, carID int64, asOf time.Time) (V2LifecycleResponse, error)
	Timeline(ctx context.Context, carID int64, eventTypes []string, limit int, before, after *time.Time) ([]V2TimelineEvent, bool, *time.Time, error)
}

// @name V2LifecycleService
type V2LifecycleService struct {
	repository V2LifecycleRepository
}

func NewV2LifecycleService(repository V2LifecycleRepository) V2LifecycleService {
	return V2LifecycleService{repository: repository}
}

// BuildLifecycle returns cumulative stats up to asOf (defaults to now if zero).
func (s V2LifecycleService) BuildLifecycle(ctx context.Context, carIDParam string, asOf time.Time) (V2LifecycleResponse, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2LifecycleResponse{}, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2LifecycleResponse{}, err
	}
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	response, err := s.repository.Lifecycle(ctx, carID, asOf)
	if err != nil {
		return V2LifecycleResponse{}, err
	}
	asOfStr := asOf.Format(time.RFC3339)
	response.AsOf = &asOfStr
	return response, nil
}

// BuildTimeline returns cursor-paginated events. before/after are cursor timestamps.
func (s V2LifecycleService) BuildTimeline(ctx context.Context, carIDParam string, eventTypes []string, limit int, before, after *time.Time) (V2TimelineResponse, error) {
	carID, err := parseV2CarID(carIDParam)
	if err != nil {
		return V2TimelineResponse{}, err
	}
	if err := s.ensureCarExists(ctx, carID); err != nil {
		return V2TimelineResponse{}, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	events, hasMore, nextCursor, err := s.repository.Timeline(ctx, carID, eventTypes, limit, before, after)
	if err != nil {
		return V2TimelineResponse{}, err
	}
	if events == nil {
		events = []V2TimelineEvent{}
	}

	resp := V2TimelineResponse{
		Events:  events,
		Total:   int64(len(events)),
		HasMore: hasMore,
	}
	if nextCursor != nil {
		s := nextCursor.Format(time.RFC3339)
		resp.NextCursor = &s
	}

	return resp, nil
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
