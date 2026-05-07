package main

// SeriesV2Envelope 描述 v2 时序接口响应。
type SeriesV2Envelope struct {
	Data SeriesV2Data         `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// SeriesV2Data 描述 v2 时序接口的 data 对象。
type SeriesV2Data struct {
	CarID   int                   `json:"car_id"`
	Scope   string                `json:"scope" example:"drives"`
	Bucket  string                `json:"bucket" example:"day"`
	Range   ExtendedRange         `json:"range"`
	Metrics []MetricSeriesMetaV2  `json:"metrics"`
	Points  []MetricSeriesPointV2 `json:"points"`
}

// MetricSeriesMetaV2 描述一个返回时序指标的元数据。
type MetricSeriesMetaV2 struct {
	Metric    string `json:"metric" example:"distance"`
	Name      string `json:"name" example:"drive_distance"`
	Unit      string `json:"unit" example:"km"`
	ChartType string `json:"chart_type" example:"bar"`
}

// MetricSeriesPointV2 描述一条宽表形式的时序数据点。
type MetricSeriesPointV2 map[string]any

// DistributionsV2Envelope 描述 v2 分布接口响应。
type DistributionsV2Envelope struct {
	Data DistributionsV2Data  `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// DistributionsV2Data 描述 v2 分布接口的 data 对象。
type DistributionsV2Data struct {
	CarID         int                    `json:"car_id"`
	Scope         string                 `json:"scope" example:"drives"`
	Range         ExtendedRange          `json:"range"`
	Distributions []MetricDistributionV2 `json:"distributions"`
}

// MetricDistributionV2 描述一个指标直方图。
type MetricDistributionV2 struct {
	Metric    string                       `json:"metric" example:"drive_distance"`
	Name      string                       `json:"name" example:"drive_distance"`
	Unit      string                       `json:"unit" example:"count"`
	ChartType string                       `json:"chart_type" example:"bar"`
	Buckets   []MetricDistributionBucketV2 `json:"buckets"`
}

// MetricDistributionBucketV2 描述一个直方图分桶。
type MetricDistributionBucketV2 struct {
	Label string  `json:"label" example:"10-20"`
	From  *int    `json:"from,omitempty"`
	To    *int    `json:"to,omitempty"`
	Count int     `json:"count"`
	Value float64 `json:"value"`
}
