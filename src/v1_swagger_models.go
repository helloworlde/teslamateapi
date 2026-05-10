package main

// V1JSONEnvelope documents handlers that return TeslaMate-style `{ "data": ... }` payloads.
// @name V1JSONEnvelope
type V1JSONEnvelope struct {
	Data interface{} `json:"data"`
}

// V1ErrorEnvelope matches TeslaMateAPIHandleErrorResponse (HTTP 200 with body `{ "error": "<message>" }`).
// @name V1ErrorEnvelope
type V1ErrorEnvelope struct {
	Error string `json:"error"`
}

// V1EnabledCommandsResponse is returned by GET command/logging when ENABLE_COMMANDS is true.
// @name V1EnabledCommandsResponse
type V1EnabledCommandsResponse struct {
	EnabledCommands []string `json:"enabled_commands"`
}
