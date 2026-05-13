package v2

// @name V2CostAPIResponse
type V2CostAPIResponse struct {
	Data V2CostResponse `json:"data"` // 响应数据
	Meta V2Meta         `json:"meta"` // 响应元信息
}

// @name V2CostResponse
type V2CostResponse struct {
	Summary        V2CostSummaryDetails `json:"summary"`
	CostByPeriod   []V2CostPeriodItem   `json:"cost_by_period,omitempty"`
	CostByLocation []V2CostLocationItem `json:"cost_by_location,omitempty"`
}

// @name V2CostSummaryDetails
type V2CostSummaryDetails struct {
	ChargingCost    *float64 `json:"charging_cost,omitempty"` // 充电费用
	EnergyUsed      *float64 `json:"energy_used,omitempty"`   // 墙端用电 (kWh)
	Distance        float64  `json:"distance"`                // 距离 (km)
	CostPerEnergy   *float64 `json:"cost_per_energy,omitempty"`
	CostPerDistance *float64 `json:"cost_per_distance,omitempty"` // 单位里程费用
}

// @name V2CostPeriodItem
type V2CostPeriodItem struct {
	PeriodStart  string   `json:"period_start"`            // 周期起始
	ChargingCost *float64 `json:"charging_cost,omitempty"` // 充电费用
	EnergyUsed   *float64 `json:"energy_used,omitempty"`   // 墙端用电 (kWh)
}

// @name V2CostLocationItem
type V2CostLocationItem struct {
	LocationName string   `json:"location_name"`           // 地点名称
	ChargingCost *float64 `json:"charging_cost,omitempty"` // 充电费用
	EnergyUsed   *float64 `json:"energy_used,omitempty"`   // 墙端用电 (kWh)
}
