package dto

// V1CommandList is the `enabled_commands` payload returned by
// GET /api/v1/cars/{CarID}/command(s).
type V1CommandList struct {
	EnabledCommands []string `json:"enabled_commands" example:"/wake_up,/command/auto_conditioning_start"`
}

// V1CommandResult is the passthrough body returned for command POSTs. Tesla
// owner-api responses vary in shape, so this is a free-form object.
type V1CommandResult map[string]any

// V1CommandRawResponse is the fallback shape when Tesla returns non-JSON
// (e.g. an HTML error page). The original status code is preserved.
type V1CommandRawResponse struct {
	Raw string `json:"raw" example:"<html>...</html>"`
}
