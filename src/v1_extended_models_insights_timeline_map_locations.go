package main

type InsightsV2Envelope struct {
	Data InsightsV2Data       `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

type InsightsV2Data struct {
	CarID    int              `json:"car_id"`
	Range    ExtendedRange    `json:"range"`
	Summary  InsightSummaryV2 `json:"summary"`
	Insights []InsightV2Item  `json:"insights"`
}

type InsightSummaryV2 struct {
	PositiveCount int `json:"positive_count"`
	WarningCount  int `json:"warning_count"`
	InfoCount     int `json:"info_count"`
	TotalCount    int `json:"total_count"`
}

type InsightV2Item struct {
	Type         string         `json:"type" example:"efficiency"`
	Level        string         `json:"level" example:"info"`
	Title        string         `json:"title"`
	Message      string         `json:"message"`
	Metric       string         `json:"metric,omitempty"`
	Current      any            `json:"current,omitempty"`
	Baseline     any            `json:"baseline,omitempty"`
	DeltaPercent *float64       `json:"delta_percent,omitempty"`
	Related      map[string]any `json:"related,omitempty"`
}

type TimelineV2Envelope struct {
	Data       []TimelineEventV2    `json:"data"`
	Pagination v1Pagination         `json:"pagination"`
	Meta       ExtendedResponseMeta `json:"meta"`
}

type TimelineEventV2 struct {
	ID         string         `json:"id"`
	Type       string         `json:"type" example:"drive"`
	StartDate  string         `json:"start_date"`
	EndDate    *string        `json:"end_date"`
	Title      string         `json:"title"`
	Summary    map[string]any `json:"summary"`
	EntityType string         `json:"entity_type" example:"drive"`
	EntityID   int            `json:"entity_id"`
}

type VisitedMapV2Envelope struct {
	Data VisitedMapV2Data     `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

type VisitedMapV2Data struct {
	CarID         int                 `json:"car_id"`
	Range         ExtendedRange       `json:"range"`
	DistanceKm    *float64            `json:"distance_km"`
	DriveCount    *int                `json:"drive_count"`
	Bounds        *VisitedMapBoundsV2 `json:"bounds,omitempty"`
	VisitedPoints []VisitedPointV2    `json:"visited_points"`
	Heatmap       []any               `json:"heatmap"`
	Truncated     bool                `json:"truncated"`
}

type VisitedMapBoundsV2 struct {
	North float64 `json:"north"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	West  float64 `json:"west"`
}

type VisitedPointV2 struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Count     int     `json:"count"`
}

type LocationsV2Envelope struct {
	Data LocationsV2Data      `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

type LocationsV2Data struct {
	CarID     int                   `json:"car_id"`
	Range     ExtendedRange         `json:"range"`
	Summary   LocationsSummaryV2    `json:"summary"`
	Locations []LocationAggregateV2 `json:"locations"`
}

type LocationsSummaryV2 struct {
	LocationCount       int      `json:"location_count"`
	ReturnedCount       int      `json:"returned_count"`
	DriveLocationCount  int      `json:"drive_location_count"`
	ChargeLocationCount int      `json:"charge_location_count"`
	DriveStartCount     int      `json:"drive_start_count"`
	DriveEndCount       int      `json:"drive_end_count"`
	ChargeCount         int      `json:"charge_count"`
	ChargeEnergyKwh     float64  `json:"charge_energy_kwh"`
	ChargeCost          *float64 `json:"charge_cost"`
}

type LocationAggregateV2 struct {
	Name            string   `json:"name"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	DriveStartCount int      `json:"drive_start_count"`
	DriveEndCount   int      `json:"drive_end_count"`
	DriveCount      int      `json:"drive_count"`
	ChargeCount     int      `json:"charge_count"`
	ChargeEnergyKwh *float64 `json:"charge_energy_kwh"`
	ChargeCost      *float64 `json:"charge_cost"`
	TotalEventCount int      `json:"total_event_count"`
	LastSeen        *string  `json:"last_seen"`
}
