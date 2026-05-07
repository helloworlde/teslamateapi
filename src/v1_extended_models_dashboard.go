package main

// StatsV2Envelope 描述 v2 周期统计接口响应。
type StatsV2Envelope struct {
	Data StatsV2Data          `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// StatsV2Data 描述 v2 周期统计接口的 data 对象。
type StatsV2Data struct {
	Period      string         `json:"period" example:"month"`
	Range       ExtendedRange  `json:"range"`
	Drives      StatsSectionV2 `json:"drives"`
	Charges     StatsSectionV2 `json:"charges"`
	Battery     StatsSectionV2 `json:"battery"`
	Parking     StatsSectionV2 `json:"parking"`
	Odometer    *float64       `json:"odometer"`
	GeneratedAt string         `json:"generated_at"`
}

// StatsSectionV2 描述可扩展的统计分区。
type StatsSectionV2 map[string]any

// DashboardV2Envelope 描述为 schema 兼容保留的旧 dashboard 响应。
type DashboardV2Envelope struct {
	Data DashboardV2Data      `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// DashboardV2Data 描述为 schema 兼容保留的旧 dashboard 载荷。
type DashboardV2Data struct {
	CarID      int               `json:"car_id"`
	Range      ExtendedRange     `json:"range"`
	Overview   DashboardOverview `json:"overview"`
	Statistics StatisticsSummary `json:"statistics"`
}

// DashboardOverview 描述可扩展的 dashboard 概览指标。
type DashboardOverview map[string]any

// RealtimeV2Envelope 描述为 schema 兼容保留的旧 realtime 响应。
type RealtimeV2Envelope struct {
	Data RealtimeV2Data       `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// RealtimeV2Data 描述为 schema 兼容保留的旧 realtime 载荷。
type RealtimeV2Data struct {
	CarID   int                      `json:"car_id"`
	Current DashboardCurrentSnapshot `json:"current"`
}

// DashboardCurrentSnapshot 描述当前车辆快照。
type DashboardCurrentSnapshot struct {
	Position map[string]any `json:"position"`
	State    map[string]any `json:"state"`
	Charge   map[string]any `json:"charge"`
}
