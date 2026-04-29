package main

type DashboardV2Envelope struct {
	Data DashboardV2Data      `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

type DashboardV2Data struct {
	CarID      int               `json:"car_id"`
	Range      ExtendedRange     `json:"range"`
	Overview   DashboardOverview `json:"overview"`
	Statistics StatisticsSummary `json:"statistics"`
}

type DashboardOverview map[string]any

type RealtimeV2Envelope struct {
	Data RealtimeV2Data       `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

type RealtimeV2Data struct {
	CarID   int                      `json:"car_id"`
	Current DashboardCurrentSnapshot `json:"current"`
}

type DashboardCurrentSnapshot struct {
	Position map[string]any `json:"position"`
	State    map[string]any `json:"state"`
	Charge   map[string]any `json:"charge"`
}
