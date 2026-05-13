package v2

import (
	"time"
)

const (
	v2DefaultPeriod  = "month"
	v2DefaultCompare = "none"
)

var (
	v2AllowedPeriods = map[string]bool{
		"day":     true,
		"week":    true,
		"month":   true,
		"quarter": true,
		"year":    true,
		"custom":  true,
	}
	v2AllowedCompares = map[string]bool{
		"none":            true,
		"previous_period": true,
	}
)

// @name V2AnalyticsQuery
type V2AnalyticsQuery struct {
	Period   string `form:"period" json:"period" example:"month"`                         // 聚合周期
	Start    string `form:"start" json:"start" example:"2026-05-01T00:00:00+08:00"`       // 起始时间
	End      string `form:"end" json:"end" example:"2026-05-31T23:59:59+08:00"`           // 结束时间
	Timezone string `form:"timezone" json:"timezone" example:"Asia/Shanghai"`             // 时区
	Compare  string `form:"compare" json:"compare" example:"previous_period"`             // 对比模式
	GroupBy  string `form:"group_by" json:"group_by,omitempty" example:"day"`             // 聚合粒度
	Metrics  string `form:"metrics" json:"metrics,omitempty" example:"distance,duration"` // 指标
	Include  string `form:"include" json:"include,omitempty" example:"summary,comparison,timeseries"`
}

// @name V2TimeRange
type V2TimeRange struct {
	Period        string
	Timezone      string
	Compare       string
	Start         time.Time
	End           time.Time
	PreviousStart *time.Time
	PreviousEnd   *time.Time
}

// @name V2Unit
type V2Unit struct {
	Distance    string `json:"distance" example:"km"` // 距离 (km)
	Energy      string `json:"energy" example:"kWh"`
	Power       string `json:"power" example:"kW"`
	Speed       string `json:"speed" example:"km/h"`
	Duration    string `json:"duration" example:"seconds"` // 时长 (秒)
	Elevation   string `json:"elevation" example:"m"`
	Consumption string `json:"consumption" example:"Wh/km"`
	Temperature string `json:"temperature" example:"C"`
	Currency    string `json:"currency" example:"CNY"`
}

// @name V2Meta
type V2Meta struct {
	CarID       int64  `json:"car_id,omitempty" example:"1"`                                         // 车辆 ID
	Period      string `json:"period,omitempty" enums:"day,week,month,quarter,year,custom,lifetime"` // 聚合周期
	Timezone    string `json:"timezone,omitempty" example:"Asia/Shanghai"`                           // 时区
	Start       string `json:"start,omitempty" format:"date-time"`                                   // 起始时间
	End         string `json:"end,omitempty" format:"date-time"`                                     // 结束时间
	Compare     string `json:"compare,omitempty" enums:"none,previous_period"`                       // 对比模式
	Unit        V2Unit `json:"unit"`                                                                 // 单位
	GeneratedAt string `json:"generated_at" format:"date-time"`
}

// @name V2APIResponse
type V2APIResponse struct {
	Data interface{} `json:"data"`
	Meta V2Meta      `json:"meta"` // 响应元信息
}

// @name V2ComparisonValue
type V2ComparisonValue struct {
	Current      *float64 `json:"current,omitempty"`
	Previous     *float64 `json:"previous,omitempty"`
	Delta        *float64 `json:"delta,omitempty"`
	DeltaPercent *float64 `json:"delta_percent,omitempty"`
}

// @name V2CapabilitiesResponse
type V2CapabilitiesResponse struct {
	Version          string                 `json:"version" example:"v2"` // 版本
	Domains          []V2CapabilitiesDomain `json:"domains"`
	BreakdownOptions map[string][]string    `json:"breakdown_options"`
}

// @name V2CapabilitiesDomain
type V2CapabilitiesDomain struct {
	Name               string `json:"name" example:"driving"` // 名称
	Path               string `json:"path" example:"/v2/cars/{car_id}/analytics/driving"`
	SupportsCompare    bool   `json:"supports_compare"`
	SupportsTimeseries bool   `json:"supports_timeseries"`
	SupportsBreakdown  bool   `json:"supports_breakdown"`
}

// @name V2CapabilitiesAPIResponse
type V2CapabilitiesAPIResponse struct {
	Data V2CapabilitiesResponse `json:"data"` // 响应数据
	Meta V2Meta                 `json:"meta"` // 响应元信息
}

// @name V2SummaryAPIResponse
type V2SummaryAPIResponse struct {
	Data V2SummaryResponse `json:"data"` // 响应数据
	Meta V2Meta            `json:"meta"` // 响应元信息
}

// @name V2SummaryResponse
type V2SummaryResponse struct {
	Summary V2Summary `json:"summary"`
	// Comparison keys: driving/charging/parking/battery/updates/vehicle/cost field paths; values are current vs previous period.
	Comparison map[string]V2ComparisonValue `json:"comparison,omitempty"`
}

// @name V2Summary
type V2Summary struct {
	Driving  V2DrivingSummary  `json:"driving"`  // 行驶指标
	Charging V2ChargingSummary `json:"charging"` // 充电指标
	Parking  V2ParkingSummary  `json:"parking"`
	Battery  V2BatterySummary  `json:"battery"` // 电池指标
	Updates  V2UpdateSummary   `json:"updates"`
	Vehicle  V2VehicleSummary  `json:"vehicle"` // 车辆信息
	Cost     V2CostSummary     `json:"cost"`    // 费用
}

// V2DrivingSummary is the canonical driving aggregate; consumed by both
// /analytics/summary and /analytics/driving.
//
// @name V2DrivingSummary
type V2DrivingSummary struct {
	DriveCount              int64    `json:"drive_count"`            // 行程数
	Distance                float64  `json:"distance"`               // 距离 (km)
	Duration                float64  `json:"duration"`               // 时长 (秒)
	AvgDistance             *float64 `json:"avg_distance,omitempty"` // 平均距离 (km)
	AvgDuration             *float64 `json:"avg_duration,omitempty"` // 平均时长 (秒)
	LongestDriveDuration    *float64 `json:"longest_drive_duration,omitempty"`
	MaxSpeed                *float64 `json:"max_speed,omitempty"`
	AvgSpeed                *float64 `json:"avg_speed,omitempty"` // 平均速度 (km/h)
	PeakDrivePower          *float64 `json:"peak_drive_power,omitempty"`
	PeakRegenPower          *float64 `json:"peak_regen_power,omitempty"`
	EstimatedEnergyConsumed *float64 `json:"estimated_energy_consumed,omitempty"`
	EstimatedEnergyRegen    *float64 `json:"estimated_energy_regen,omitempty"`
	NetEnergy               *float64 `json:"net_energy,omitempty"`        // 净能量 (kWh)
	AvgConsumption          *float64 `json:"avg_consumption,omitempty"`   // 平均能耗 (Wh/km)
	BestConsumption         *float64 `json:"best_consumption,omitempty"`  // 最佳能耗 (Wh/km)
	WorstConsumption        *float64 `json:"worst_consumption,omitempty"` // 最差能耗 (Wh/km)
	RangeLoss               float64  `json:"range_loss"`                  // 续航损失 (km)
	BatteryLevelUsed        *float64 `json:"battery_level_used,omitempty"`
	AvgOutsideTemp          *float64 `json:"avg_outside_temp,omitempty"`
}

// V2ChargingSummary is the canonical charging aggregate; consumed by both
// /analytics/summary and /analytics/charging.
//
// @name V2ChargingSummary
type V2ChargingSummary struct {
	SessionCount           int64    `json:"session_count"`          // 充电会话数
	EnergyAdded            float64  `json:"energy_added"`           // 充入电池能量 (kWh)
	EnergyUsed             float64  `json:"energy_used"`            // 墙端用电 (kWh)
	Duration               float64  `json:"duration"`               // 时长 (秒)
	AvgDuration            *float64 `json:"avg_duration,omitempty"` // 平均时长 (秒)
	LongestSessionDuration *float64 `json:"longest_session_duration,omitempty"`
	AvgEnergyAdded         *float64 `json:"avg_energy_added,omitempty"`
	LargestSession         *float64 `json:"largest_session,omitempty"`
	AvgPower               *float64 `json:"avg_power,omitempty"` // 平均功率 (kW)
	MaxPower               *float64 `json:"max_power,omitempty"`
	ChargingEfficiency     *float64 `json:"charging_efficiency,omitempty"` // 充电效率
	Cost                   float64  `json:"cost"`                          // 费用
	AvgCost                *float64 `json:"avg_cost,omitempty"`
	MaxCost                *float64 `json:"max_cost,omitempty"`
	AvgCostPerEnergy       *float64 `json:"avg_cost_per_energy,omitempty"`
	StartBatteryAvg        *float64 `json:"start_battery_avg,omitempty"`
	EndBatteryAvg          *float64 `json:"end_battery_avg,omitempty"`
	ACSessionCount         int64    `json:"ac_session_count"`
	DCSessionCount         int64    `json:"dc_session_count"`
	ACEnergy               float64  `json:"ac_energy"`
	DCEnergy               float64  `json:"dc_energy"`
}

// @name V2VehicleSummary
type V2VehicleSummary struct {
	Odometer             *float64 `json:"odometer,omitempty"`
	RatedEfficiency      *float64 `json:"rated_efficiency,omitempty"`
	TrackedConsumption   *float64 `json:"tracked_consumption,omitempty"`
	TrackedWall          *float64 `json:"tracked_wall,omitempty"`
	ChargeEfficiency     *float64 `json:"charge_efficiency,omitempty"`
	OdometerCoverage     *float64 `json:"odometer_coverage,omitempty"`
	OdometerTracked      *float64 `json:"odometer_tracked,omitempty"`
	OdometerTotal        *float64 `json:"odometer_total,omitempty"`
	TrackedDistance      float64  `json:"tracked_distance"`
	TrackedDrives        int64    `json:"tracked_drives"`
	TrackedCharges       int64    `json:"tracked_charges"`
}

// V2ParkingSummary is the canonical parking aggregate; consumed by both
// /analytics/summary and /analytics/parking.
//
// @name V2ParkingSummary
type V2ParkingSummary struct {
	ParkingSessionCount   int64    `json:"parking_session_count"`         // 驻车会话数
	ParkedDuration        float64  `json:"parked_duration"`               // 驻车时长 (秒)
	AvgParkedDuration     *float64 `json:"avg_parked_duration,omitempty"` // 平均驻车时长 (秒)
	AsleepDuration        float64  `json:"asleep_duration"`
	OnlineDuration        float64  `json:"online_duration"`
	OfflineDuration       float64  `json:"offline_duration"`
	VampireDrainPercent   *float64 `json:"vampire_drain_percent,omitempty"` // 吸血式漏电比例 (%)
	EstimatedVampireDrain *float64 `json:"estimated_vampire_drain,omitempty"`
	AvgDrainPercentPerDay *float64 `json:"avg_drain_percent_per_day,omitempty"`
	StateTransitionCount  int64    `json:"state_transition_count"` // 状态切换次数
}

// @name V2BatterySummary
type V2BatterySummary struct {
	LatestLevel               *int64          `json:"latest_level,omitempty"`
	LatestRatedRange          *float64        `json:"latest_rated_range,omitempty"`   // 最近额定续航 (km)
	LatestIdealRange          *float64        `json:"latest_ideal_range,omitempty"`   // 最近理想续航 (km)
	RangeAtFullCharge         *V2BatteryRange `json:"range_at_full_charge,omitempty"` // 满电续航 (km)
	BaselineRangeAtFullCharge *V2BatteryRange `json:"baseline_range_at_full_charge,omitempty"`
	EstimatedRangeDegradation *float64        `json:"estimated_range_degradation,omitempty"`
}

// @name V2BatteryRange
type V2BatteryRange struct {
	Rated *float64 `json:"rated,omitempty"`
	Ideal *float64 `json:"ideal,omitempty"`
}

// @name V2UpdateSummary
type V2UpdateSummary struct {
	UpdateCount   int64   `json:"update_count"`             // OTA 更新次数
	LatestVersion *string `json:"latest_version,omitempty"` // 最近版本
}

// @name V2CostSummary
type V2CostSummary struct {
	ChargingCost    float64  `json:"charging_cost"`               // 充电费用
	CostPerDistance *float64 `json:"cost_per_distance,omitempty"` // 单位里程费用
}
