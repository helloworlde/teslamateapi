package dto

// V1BatteryHealth carries the derived range / capacity / efficiency metrics
// surfaced by GET /api/v1/cars/{CarID}/battery-health.
type V1BatteryHealth struct {
	MaxRange                float64 `json:"max_range" example:"450.2"`
	CurrentRange            float64 `json:"current_range" example:"432.7"`
	MaxCapacity             float64 `json:"max_capacity" example:"82.5"`
	CurrentCapacity         float64 `json:"current_capacity" example:"79.4"`
	RatedEfficiency         float64 `json:"rated_efficiency" example:"18.4"`
	BatteryHealthPercentage float64 `json:"battery_health_percentage" example:"96.24"`
}

// V1BatteryHealthData is the `data` field of V1BatteryHealthResponse.
type V1BatteryHealthData struct {
	Car            Car             `json:"car"`
	BatteryHealth  V1BatteryHealth `json:"battery_health"`
	TeslaMateUnits TeslaMateUnits  `json:"units"`
}

// V1BatteryHealthResponse is the envelope for /api/v1/cars/{CarID}/battery-health.
type V1BatteryHealthResponse struct {
	Data V1BatteryHealthData `json:"data"`
}
