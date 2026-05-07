package main

// InsightsV2Envelope 描述 v2 洞察接口响应。
type InsightsV2Envelope struct {
	Data InsightsV2Data       `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// InsightsV2Data 描述 v2 洞察接口的 data 对象。
type InsightsV2Data struct {
	CarID    int              `json:"car_id"`
	Range    ExtendedRange    `json:"range"`
	Summary  InsightSummaryV2 `json:"summary"`
	Insights []InsightV2Item  `json:"insights"`
}

// InsightSummaryV2 描述洞察严重级别计数。
type InsightSummaryV2 struct {
	PositiveCount int `json:"positive_count"`
	WarningCount  int `json:"warning_count"`
	InfoCount     int `json:"info_count"`
	TotalCount    int `json:"total_count"`
}

// InsightV2Item 描述一张洞察卡片。
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

// TrendsV2Envelope 描述 v2 趋势接口响应。
type TrendsV2Envelope struct {
	Data TrendsV2Data         `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// TrendsV2Data 描述 v2 趋势接口的 data 对象。
type TrendsV2Data struct {
	CarID           int           `json:"car_id"`
	Period          string        `json:"period" example:"month"`
	Range           ExtendedRange `json:"range"`
	ComparisonRange ExtendedRange `json:"comparison_range"`
	Trends          []TrendV2Item `json:"trends"`
}

// TrendV2Item 描述一个环比指标对比。
type TrendV2Item struct {
	Metric         string   `json:"metric" example:"distance"`
	Name           string   `json:"name" example:"Distance Driven"`
	Unit           string   `json:"unit" example:"km"`
	Current        any      `json:"current"`
	Previous       any      `json:"previous"`
	ChangePercent  *float64 `json:"change_percent"`
	Direction      string   `json:"direction" example:"better"`
	HigherIsBetter bool     `json:"higher_is_better"`
}

// RecordsV2Envelope 描述 v2 纪录接口响应。
type RecordsV2Envelope struct {
	Data RecordsV2Data        `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// RecordsV2Data 描述 v2 纪录接口的 data 对象。
type RecordsV2Data struct {
	CarID      int                 `json:"car_id"`
	YearFilter any                 `json:"year_filter"`
	Records    map[string]RecordV2 `json:"records"`
}

// RecordV2 描述一项个人最佳纪录。
type RecordV2 struct {
	Value      any    `json:"value"`
	Unit       string `json:"unit"`
	Date       any    `json:"date"`
	EntityType string `json:"entity_type" example:"drive"`
	EntityID   any    `json:"entity_id"`
}

// TimelineV2Envelope 描述为 schema 兼容保留的旧 timeline 响应。
type TimelineV2Envelope struct {
	Data       []TimelineEventV2    `json:"data"`
	Pagination v1Pagination         `json:"pagination"`
	Meta       ExtendedResponseMeta `json:"meta"`
}

// TimelineEventV2 描述一个时间线事件。
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

// VisitedMapV2Envelope 描述 v2 访问地图接口响应。
type VisitedMapV2Envelope struct {
	Data VisitedMapV2Data     `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// VisitedMapV2Data 描述 v2 访问地图接口的 data 对象。
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

// VisitedMapBoundsV2 描述访问地图坐标边界。
type VisitedMapBoundsV2 struct {
	North float64 `json:"north"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	West  float64 `json:"west"`
}

// VisitedPointV2 描述一个访问地图点位。
type VisitedPointV2 struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Count     int     `json:"count"`
}

// LocationsV2Envelope 描述 v2 位置聚合接口响应。
type LocationsV2Envelope struct {
	Data LocationsV2Data      `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// LocationsV2Data 描述 v2 位置聚合接口的 data 对象。
type LocationsV2Data struct {
	CarID     int                   `json:"car_id"`
	Range     ExtendedRange         `json:"range"`
	Summary   LocationsSummaryV2    `json:"summary"`
	Locations []LocationAggregateV2 `json:"locations"`
}

// LocationsSummaryV2 描述位置聚合计数。
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

// LocationAggregateV2 描述一个聚合后的行程或充电地点。
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
