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
