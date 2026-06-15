package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2TimeDistributionRange echoes the requested time window for a time
// distribution response. Empty values mean the request used the lifetime
// default on that side of the range.
type V2TimeDistributionRange struct {
	Start nullable.String `json:"start" swaggertype:"string" example:"2026-01-01T00:00:00+01:00"`
	End   nullable.String `json:"end" swaggertype:"string" example:"2026-06-01T00:00:00+01:00"`
}

// V2TimeDistributionSegment is one mutually-exclusive time bucket.
type V2TimeDistributionSegment struct {
	Key         string  `json:"key" example:"driving" enums:"driving,parked,charging"`
	Label       string  `json:"label" example:"Driving"`
	DurationMin int     `json:"duration_min" example:"4200"`
	Percent     float64 `json:"percent" example:"18.5"`
}

// V2TimeDistributionData is the `data` field for vehicle time distribution.
//
// Parking time excludes charging overlap so driving + parked + charging are
// mutually exclusive and can be used directly in a pie chart.
type V2TimeDistributionData struct {
	Car              Car                         `json:"car"`
	Range            V2TimeDistributionRange     `json:"range"`
	TotalDurationMin int                         `json:"total_duration_min" example:"22680"`
	Segments         []V2TimeDistributionSegment `json:"segments"`
}

// V2TimeDistributionResponse is the envelope for
// /api/v2/cars/{CarID}/stats/time-distribution.
type V2TimeDistributionResponse struct {
	Data V2TimeDistributionData `json:"data"`
}
