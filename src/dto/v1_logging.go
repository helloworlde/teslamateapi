package dto

// V1LoggingList is the `enabled_commands` payload returned by
// GET /api/v1/cars/{CarID}/logging.
type V1LoggingList struct {
	EnabledCommands []string `json:"enabled_commands" example:"/logging/resume,/logging/suspend"`
}

// V1LoggingResult is the passthrough body returned for logging PUTs.
type V1LoggingResult map[string]any

// V1LoggingRawResponse is the fallback shape when TeslaMate returns non-JSON
// (e.g. an HTML error page). The original status code is preserved.
type V1LoggingRawResponse struct {
	Raw string `json:"raw" example:"<html>...</html>"`
}
