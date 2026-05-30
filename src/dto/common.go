// Package dto defines the request/response shapes the teslamateapi HTTP
// surface produces. Types here are pure data carriers — no behavior, no
// imports from main — so swaggo/swag can introspect them and emit a complete
// OpenAPI document via `swag init`.
//
// Conventions:
//   - JSON tags mirror the legacy in-handler structs byte-for-byte. The
//     handlers convert from internal representations into these types only
//     where helpful; in many cases the types are already drop-in aliases of
//     what the handlers built.
//   - Nullable scalars use the `nullable.*` wrappers, which marshal to
//     scalar-or-null. A `swaggertype` tag on each nullable field tells swag
//     to emit `{"type":"string"}` (etc.) instead of dumping the {Valid, X}
//     internal shape.
//   - Each endpoint owns one `*Response` envelope under `Data: {...}`. Even
//     when two endpoints share a logical shape, we keep them in distinct
//     types so future field drift on one endpoint can't bleed into the
//     other.
package dto

import "github.com/tobiasehlert/teslamateapi/src/nullable"

// Car identifies the vehicle scope of a response. Re-used by every
// per-car endpoint.
type Car struct {
	CarID   int             `json:"car_id" example:"1"`
	CarName nullable.String `json:"car_name" swaggertype:"string" example:"Blue Thunder"`
}

// TeslaMateUnits captures the user's preferred display units for length and
// temperature. Used by every endpoint that surfaces distances or temps.
type TeslaMateUnits struct {
	UnitsLength      string `json:"unit_of_length" example:"km" enums:"km,mi"`
	UnitsTemperature string `json:"unit_of_temperature" example:"C" enums:"C,F"`
}

// TeslaMateUnitsWithPressure extends TeslaMateUnits with the pressure unit.
// Only the /status endpoint surfaces tire-pressure values, so only it
// includes this field.
type TeslaMateUnitsWithPressure struct {
	UnitsLength      string `json:"unit_of_length" example:"km" enums:"km,mi"`
	UnitsPressure    string `json:"unit_of_pressure" example:"bar" enums:"bar,psi"`
	UnitsTemperature string `json:"unit_of_temperature" example:"C" enums:"C,F"`
}

// Pagination describes the page metadata returned by v2 list endpoints.
type Pagination struct {
	Page       int  `json:"page" example:"1"`
	Show       int  `json:"show" example:"100"`
	TotalCount int  `json:"total_count" example:"382"`
	HasNext    bool `json:"has_next" example:"false"`
}

// ErrorEnvelope is the JSON shape every error response uses (both v1's
// 200+envelope quirk and v2's real-status-code error path).
type ErrorEnvelope struct {
	Error string `json:"error" example:"Unable to load cars."`
}

// MessageEnvelope is returned by inline status / docs / health probes.
type MessageEnvelope struct {
	Message string `json:"message" example:"TeslaMateApi container running.."`
	Path    string `json:"path,omitempty" example:"/api"`
}

// PongResponse is returned by /api/ping.
type PongResponse struct {
	Message string `json:"message" example:"pong"`
}

// HealthResponse is returned by /api/healthz.
type HealthResponse struct {
	Status string `json:"status" example:"OK"`
}

// ReadyResponse is returned by /api/readyz when the API is ready to accept
// traffic.
type ReadyResponse struct {
	Status string `json:"status" example:"OK"`
}

// NotFoundResponse is the JSON body returned by the 404 handler.
type NotFoundResponse struct {
	Code    string `json:"code" example:"PAGE_NOT_FOUND"`
	Message string `json:"message" example:"Page not found"`
}
