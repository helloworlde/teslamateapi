package main

// ActivityV2Envelope 描述 v2 活动日历接口响应。
type ActivityV2Envelope struct {
	Data ActivityV2Data       `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// ActivityV2Data 描述 v2 活动日历接口的 data 对象。
type ActivityV2Data struct {
	CarID   int                `json:"car_id"`
	Range   ExtendedRange      `json:"range"`
	Bucket  string             `json:"bucket" example:"day"`
	Summary ActivitySummaryV2  `json:"summary"`
	Items   []ActivityBucketV2 `json:"items"`
}

// ActivitySummaryV2 描述可扩展的活动汇总指标。
type ActivitySummaryV2 map[string]any

// ActivityBucketV2 描述一个活动日历桶。
type ActivityBucketV2 map[string]any

// CalendarV2Envelope 描述为 schema 兼容保留的旧 calendar 响应。
type CalendarV2Envelope struct {
	Data CalendarV2Data       `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// CalendarV2Data 描述为 schema 兼容保留的旧 calendar 载荷。
type CalendarV2Data struct {
	CarID   int                `json:"car_id"`
	Range   ExtendedRange      `json:"range"`
	Bucket  string             `json:"bucket" example:"day"`
	Summary CalendarSummaryV2  `json:"summary"`
	Items   []CalendarBucketV2 `json:"items"`
}

// CalendarSummaryV2 描述可扩展的日历汇总指标。
type CalendarSummaryV2 map[string]any

// CalendarBucketV2 描述一个可扩展的日历桶。
type CalendarBucketV2 map[string]any

// StatisticsV2Envelope 描述为 schema 兼容保留的旧 statistics 响应。
type StatisticsV2Envelope struct {
	Data StatisticsV2Data     `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// StatisticsV2Data 描述为 schema 兼容保留的旧 statistics 载荷。
type StatisticsV2Data struct {
	CarID    int                  `json:"car_id"`
	Period   string               `json:"period" example:"month"`
	Range    ExtendedRange        `json:"range"`
	Overview StatisticsOverviewV2 `json:"overview"`
	Drive    StatisticsDriveV2    `json:"drive"`
	Charge   StatisticsChargeV2   `json:"charge"`
	Battery  StatisticsBatteryV2  `json:"battery"`
}

// StatisticsOverviewV2 描述可扩展的统计概览指标。
type StatisticsOverviewV2 map[string]any

// StatisticsDriveV2 描述可扩展的行程统计指标。
type StatisticsDriveV2 map[string]any

// StatisticsChargeV2 描述可扩展的充电统计指标。
type StatisticsChargeV2 map[string]any

// StatisticsBatteryV2 描述可扩展的电池统计指标。
type StatisticsBatteryV2 map[string]any
