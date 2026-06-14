package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V1DriveDetailClimateInfo carries the per-sample climate values.
type V1DriveDetailClimateInfo struct {
	InsideTemp           nullable.Float64 `json:"inside_temp" swaggertype:"number" example:"21.0"`
	OutsideTemp          nullable.Float64 `json:"outside_temp" swaggertype:"number" example:"15.3"`
	IsClimateOn          nullable.Bool    `json:"is_climate_on" swaggertype:"boolean"`
	FanStatus            nullable.Int64   `json:"fan_status" swaggertype:"integer" example:"3"`
	DriverTempSetting    nullable.Float64 `json:"driver_temp_setting" swaggertype:"number" example:"21.0"`
	PassengerTempSetting nullable.Float64 `json:"passenger_temp_setting" swaggertype:"number" example:"21.0"`
	IsRearDefrosterOn    nullable.Bool    `json:"is_rear_defroster_on" swaggertype:"boolean"`
	IsFrontDefrosterOn   nullable.Bool    `json:"is_front_defroster_on" swaggertype:"boolean"`
}

// V1DriveDetailBatteryInfo carries per-sample range and heater state.
type V1DriveDetailBatteryInfo struct {
	EstBatteryRange      nullable.Float64 `json:"est_battery_range" swaggertype:"number" example:"310.0"`
	IdealBatteryRange    nullable.Float64 `json:"ideal_battery_range" swaggertype:"number" example:"320.0"`
	RatedBatteryRange    nullable.Float64 `json:"rated_battery_range" swaggertype:"number" example:"305.0"`
	BatteryHeater        nullable.Bool    `json:"battery_heater" swaggertype:"boolean"`
	BatteryHeaterOn      nullable.Bool    `json:"battery_heater_on" swaggertype:"boolean"`
	BatteryHeaterNoPower nullable.Bool    `json:"battery_heater_no_power" swaggertype:"boolean"`
}

// V1DriveDetailPoint represents a single position sample inside a drive.
type V1DriveDetailPoint struct {
	DetailID           int                      `json:"detail_id" example:"123"`
	Date               string                   `json:"date" example:"2024-01-01T08:30:00+01:00"`
	Latitude           float64                  `json:"latitude" example:"52.5200"`
	Longitude          float64                  `json:"longitude" example:"13.4050"`
	Speed              int                      `json:"speed" example:"60"`
	Power              int                      `json:"power" example:"120"`
	Odometer           float64                  `json:"odometer" example:"42050.5"`
	BatteryLevel       int                      `json:"battery_level" example:"68"`
	UsableBatteryLevel nullable.Int64           `json:"usable_battery_level" swaggertype:"integer" example:"67"`
	Elevation          nullable.Int64           `json:"elevation" swaggertype:"integer" example:"55"`
	ClimateInfo        V1DriveDetailClimateInfo `json:"climate_info"`
	BatteryInfo        V1DriveDetailBatteryInfo `json:"battery_info"`
}

// V1DriveDetail is the full per-drive blob for /drives/{DriveID}.
type V1DriveDetail struct {
	DriveID           int                     `json:"drive_id" example:"1234"`
	StartDate         string                  `json:"start_date" example:"2024-01-01T08:00:00+01:00"`
	EndDate           string                  `json:"end_date" example:"2024-01-01T09:30:00+01:00"`
	StartAddress      string                  `json:"start_address" example:"Home"`
	EndAddress        string                  `json:"end_address" example:"Office"`
	OdometerDetails   V1DrivesOdometerDetails `json:"odometer_details"`
	DurationMin       int                     `json:"duration_min" example:"90"`
	DurationStr       string                  `json:"duration_str" example:"01:30"`
	SpeedMax          int                     `json:"speed_max" example:"130"`
	SpeedAvg          float64                 `json:"speed_avg" example:"66.7"`
	PowerMax          int                     `json:"power_max" example:"250"`
	PowerMin          int                     `json:"power_min" example:"-50"`
	BatteryDetails    V1DrivesBatteryDetails  `json:"battery_details"`
	RangeIdeal        V1DrivesPreferredRange  `json:"range_ideal"`
	RangeRated        V1DrivesPreferredRange  `json:"range_rated"`
	OutsideTempAvg    float64                 `json:"outside_temp_avg" example:"15.3"`
	InsideTempAvg     float64                 `json:"inside_temp_avg" example:"21.0"`
	EnergyConsumedNet *float64                `json:"energy_consumed_net" example:"22.5"`
	ConsumptionNet    *float64                `json:"consumption_net" example:"225.0"`
	// RangeAchievementPct = distance / rated-range drop × 100 (objective).
	// Null when the rated-range drop is non-positive (charging mid-drive,
	// missing range readings). Unit-independent: a ratio of two distances.
	RangeAchievementPct *float64 `json:"range_achievement_pct" example:"92.5"`
	// EstimatedUsageCost = (lifetime charging cost / lifetime distance) × this
	// drive's distance. Semi-objective: amortises a global per-distance rate
	// onto one trip. 0 when no charging cost is configured; null when there is
	// no lifetime distance yet. In the same currency as charging_processes.cost.
	EstimatedUsageCost *float64             `json:"estimated_usage_cost" example:"3.42"`
	DriveDetails       []V1DriveDetailPoint `json:"drive_details"`
}

// V1DriveDetailData is the `data` field of V1DriveDetailResponse.
type V1DriveDetailData struct {
	Car            Car            `json:"car"`
	Drive          V1DriveDetail  `json:"drive"`
	TeslaMateUnits TeslaMateUnits `json:"units"`
}

// V1DriveDetailResponse is the envelope for /api/v1/cars/{CarID}/drives/{DriveID}.
type V1DriveDetailResponse struct {
	Data V1DriveDetailData `json:"data"`
}
