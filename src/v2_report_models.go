package main

// @name V2ReportSection
type V2ReportSection struct {
	Type    string                 `json:"type"`
	Title   string                 `json:"title"`
	Metrics map[string]interface{} `json:"metrics"`
}

// @name V2ReportResponse
type V2ReportResponse struct {
	Title    string            `json:"title"`
	Period   string            `json:"period"`
	Sections []V2ReportSection `json:"sections"`
}

// V2ReportAPIResponse is the swagger wrapper for reports
// @name V2ReportAPIResponse
type V2ReportAPIResponse struct {
	Data V2ReportResponse `json:"data"`
	Meta V2Meta           `json:"meta"`
}
