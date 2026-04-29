package main

type CalendarV2Envelope struct {
	Data CalendarV2Data       `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

type CalendarV2Data struct {
	CarID   int                `json:"car_id"`
	Range   ExtendedRange      `json:"range"`
	Bucket  string             `json:"bucket" example:"day"`
	Summary CalendarSummaryV2  `json:"summary"`
	Items   []CalendarBucketV2 `json:"items"`
}

type CalendarSummaryV2 map[string]any

type CalendarBucketV2 map[string]any

type StatisticsV2Envelope struct {
	Data StatisticsV2Data     `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

type StatisticsV2Data struct {
	CarID    int                  `json:"car_id"`
	Period   string               `json:"period" example:"month"`
	Range    ExtendedRange        `json:"range"`
	Overview StatisticsOverviewV2 `json:"overview"`
	Drive    StatisticsDriveV2    `json:"drive"`
	Charge   StatisticsChargeV2   `json:"charge"`
	Battery  StatisticsBatteryV2  `json:"battery"`
}

type StatisticsOverviewV2 map[string]any
type StatisticsDriveV2 map[string]any
type StatisticsChargeV2 map[string]any
type StatisticsBatteryV2 map[string]any
