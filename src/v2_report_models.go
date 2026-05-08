package main

type V2ReportSection struct {
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Metrics     map[string]interface{} `json:"metrics"`
	DataQuality V2DataQuality          `json:"data_quality"`
}

type V2ReportResponse struct {
	Title    string            `json:"title"`
	Period   string            `json:"period"`
	Sections []V2ReportSection `json:"sections"`
}

// V2ReportAPIResponse is the swagger wrapper for reports
type V2ReportAPIResponse struct {
	Data V2ReportResponse `json:"data"`
	Meta V2Meta           `json:"meta"`
}
