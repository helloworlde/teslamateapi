package main

// @name V2ChargingAPIResponse
type V2ChargingAPIResponse struct {
	Data V2ChargingResponse `json:"data"` // 响应数据
	Meta V2Meta `json:"meta"` // 响应元信息
}

// @name V2ChargingResponse
type V2ChargingResponse struct {
	Summary    V2ChargingSummary             `json:"summary"`
	Timeseries *V2ChargingTimeseriesResponse `json:"timeseries,omitempty"` // 时序
	Breakdown  *V2ChargingBreakdownResponse `json:"breakdown,omitempty"` // 分项维度
}

// @name V2ChargingBreakdownResponse
type V2ChargingBreakdownResponse struct {
	By        string                   `json:"by"`
	Locations []V2ChargingLocationItem `json:"locations,omitempty"` // 地点列表
	Types     []V2ChargingTypeItem     `json:"types,omitempty"`
}

// @name V2ChargingTimeseriesResponse
type V2ChargingTimeseriesResponse struct {
	GroupBy string `json:"group_by"` // 聚合粒度
	Items   []V2ChargingTimeseriesItem `json:"items"` // 条目列表
}

// @name V2ChargingTimeseriesItem
type V2ChargingTimeseriesItem struct {
	PeriodStart  string `json:"period_start"` // 周期起始
	SessionCount int64 `json:"session_count"` // 充电会话数
	EnergyAdded  float64 `json:"energy_added"` // 充入电池能量 (kWh)
	EnergyUsed   *float64 `json:"energy_used,omitempty"` // 墙端用电 (kWh)
	Duration     float64 `json:"duration"` // 时长 (秒)
	Cost         *float64 `json:"cost,omitempty"` // 费用
	AvgPower     *float64 `json:"avg_power,omitempty"` // 平均功率 (kW)
}

// @name V2ChargingLocationItem
type V2ChargingLocationItem struct {
	LocationName       string `json:"location_name"` // 地点名称
	GeofenceID         *int64 `json:"geofence_id,omitempty"` // 围栏 ID
	AddressID          *int64 `json:"address_id,omitempty"` // 地址 ID
	SessionCount       int64 `json:"session_count"` // 充电会话数
	EnergyAdded        float64 `json:"energy_added"` // 充入电池能量 (kWh)
	EnergyUsed         *float64 `json:"energy_used,omitempty"` // 墙端用电 (kWh)
	Cost               *float64 `json:"cost,omitempty"` // 费用
	ChargingEfficiency *float64 `json:"charging_efficiency,omitempty"` // 充电效率
}

// @name V2ChargingTypeItem
type V2ChargingTypeItem struct {
	ChargingType string   `json:"charging_type"`
	SessionCount int64 `json:"session_count"` // 充电会话数
	EnergyAdded  float64 `json:"energy_added"` // 充入电池能量 (kWh)
	EnergyUsed   *float64 `json:"energy_used,omitempty"` // 墙端用电 (kWh)
	Cost         *float64 `json:"cost,omitempty"` // 费用
}
