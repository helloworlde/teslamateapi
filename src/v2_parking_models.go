package main

// @name V2ParkingAPIResponse
type V2ParkingAPIResponse struct {
	Data V2ParkingResponse `json:"data"` // 响应数据
	Meta V2Meta `json:"meta"` // 响应元信息
}

// @name V2ParkingResponse
type V2ParkingResponse struct {
	Summary   V2ParkingSummary            `json:"summary"`
	Breakdown *V2ParkingBreakdownResponse `json:"breakdown,omitempty"` // 分项维度
}

// @name V2ParkingBreakdownResponse
type V2ParkingBreakdownResponse struct {
	By        string                  `json:"by"`
	Locations []V2ParkingLocationItem `json:"locations,omitempty"` // 地点列表
	States    []V2ParkingStateItem `json:"states,omitempty"` // 州/省分布
}

// V2ParkingStatesResponse is the internal shape returned by the parking
// repository's state breakdown query. It is not directly exposed via swagger
// after consolidation; values are folded into V2ParkingBreakdownResponse.
type V2ParkingStatesResponse struct {
	TotalDuration        float64              `json:"total_duration"`
	StateTransitionCount int64 `json:"state_transition_count"` // 状态切换次数
	Items                []V2ParkingStateItem `json:"items"` // 条目列表
}

// @name V2ParkingLocationItem
type V2ParkingLocationItem struct {
	LocationName        string `json:"location_name"` // 地点名称
	GeofenceID          *int64 `json:"geofence_id,omitempty"` // 围栏 ID
	AddressID           *int64 `json:"address_id,omitempty"` // 地址 ID
	ParkingSessionCount int64 `json:"parking_session_count"` // 驻车会话数
	ParkedDuration      float64 `json:"parked_duration"` // 驻车时长 (秒)
	AvgParkedDuration   *float64 `json:"avg_parked_duration,omitempty"` // 平均驻车时长 (秒)
}

// @name V2ParkingStateItem
type V2ParkingStateItem struct {
	State           string `json:"state"` // 状态
	Duration        float64 `json:"duration"` // 时长 (秒)
	Percent         float64 `json:"percent"`
	TransitionCount int64   `json:"transition_count"`
}
