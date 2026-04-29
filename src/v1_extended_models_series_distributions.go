package main

type SeriesV2Envelope struct {
	Data SeriesV2Data         `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

type SeriesV2Data struct {
	CarID   int                   `json:"car_id"`
	Scope   string                `json:"scope" example:"drives"`
	Bucket  string                `json:"bucket" example:"day"`
	Range   ExtendedRange         `json:"range"`
	Metrics []MetricSeriesMetaV2  `json:"metrics"`
	Points  []MetricSeriesPointV2 `json:"points"`
}

type MetricSeriesMetaV2 struct {
	Metric    string `json:"metric" example:"distance"`
	Name      string `json:"name" example:"drive_distance"`
	Unit      string `json:"unit" example:"km"`
	ChartType string `json:"chart_type" example:"bar"`
}

type MetricSeriesPointV2 map[string]any

type DistributionsV2Envelope struct {
	Data DistributionsV2Data  `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

type DistributionsV2Data struct {
	CarID         int                    `json:"car_id"`
	Scope         string                 `json:"scope" example:"drives"`
	Range         ExtendedRange          `json:"range"`
	Distributions []MetricDistributionV2 `json:"distributions"`
}

type MetricDistributionV2 struct {
	Metric    string                       `json:"metric" example:"drive_distance"`
	Name      string                       `json:"name" example:"drive_distance"`
	Unit      string                       `json:"unit" example:"count"`
	ChartType string                       `json:"chart_type" example:"bar"`
	Buckets   []MetricDistributionBucketV2 `json:"buckets"`
}

type MetricDistributionBucketV2 struct {
	Label string  `json:"label" example:"10-20"`
	From  *int    `json:"from,omitempty"`
	To    *int    `json:"to,omitempty"`
	Count int     `json:"count"`
	Value float64 `json:"value"`
}
