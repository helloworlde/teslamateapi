package main

type V2Insight struct {
	ID             string                 `json:"id"`
	Category       string                 `json:"category"`
	Severity       string                 `json:"severity"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	CurrentPeriod  string                 `json:"current_period"`
	BaselinePeriod string                 `json:"baseline_period"`
	Metrics        map[string]interface{} `json:"metrics"`
	Evidence       map[string]interface{} `json:"evidence"`
}

type V2InsightResponse struct {
	Insights []V2Insight `json:"insights"`
	Total    int         `json:"total"`
}

// V2InsightAPIResponse is the swagger wrapper for insights
type V2InsightAPIResponse struct {
	Data V2InsightResponse `json:"data"`
	Meta V2Meta            `json:"meta"`
}
