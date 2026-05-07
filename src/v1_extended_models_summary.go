package main

// SummaryV2Envelope 描述为 schema 兼容保留的旧 summary 响应。
type SummaryV2Envelope struct {
	Data SummaryV2Data        `json:"data"`
	Meta ExtendedResponseMeta `json:"meta"`
}

// SummaryV2Data 描述为 schema 兼容保留的旧 summary 载荷。
type SummaryV2Data struct {
	SchemaVersion string                `json:"schema_version" example:"summary.v1"`
	Car           SummaryV2Car          `json:"car"`
	Range         ExtendedRange         `json:"range"`
	Units         TeslaMateSummaryUnits `json:"units"`
	Overview      SummaryV2Overview     `json:"overview"`
	Driving       SummaryV2Driving      `json:"driving"`
	Charging      SummaryV2Charging     `json:"charging"`
	Parking       SummaryV2Parking      `json:"parking"`
	Battery       SummaryV2Battery      `json:"battery"`
	Efficiency    SummaryV2Efficiency   `json:"efficiency"`
	Cost          SummaryV2Cost         `json:"cost"`
	Quality       SummaryV2Quality      `json:"quality"`
	State         SummaryV2VehicleState `json:"state"`
	GeneratedAt   string                `json:"generated_at"`
}

// SummaryV2Car 描述摘要中的车辆身份字段。
type SummaryV2Car struct {
	CarID   int    `json:"car_id" example:"1"`
	CarName string `json:"car_name" example:"Model 3"`
}

// SummaryV2Overview 描述摘要中的高层计数指标。
type SummaryV2Overview struct {
	DriveCount         int      `json:"drive_count"`
	ChargeCount        int      `json:"charge_count"`
	ParkingCount       int      `json:"parking_count"`
	Distance           float64  `json:"distance"`
	DriveDurationMin   int      `json:"drive_duration_min"`
	ChargeDurationMin  int      `json:"charge_duration_min"`
	ParkingDurationMin int      `json:"parking_duration_min"`
	EnergyUsed         *float64 `json:"energy_used"`
	EnergyAdded        float64  `json:"energy_added"`
	Cost               *float64 `json:"cost"`
	LatestOdometer     *float64 `json:"latest_odometer"`
}

// SummaryV2Duration 描述总时长、平均时长和最长时长指标。
type SummaryV2Duration struct {
	TotalMin   int      `json:"total_min"`
	AverageMin *float64 `json:"average_min"`
	LongestMin *int     `json:"longest_min"`
}

// SummaryV2Driving 描述摘要中的行程指标。
type SummaryV2Driving struct {
	Count       int                    `json:"count"`
	Distance    float64                `json:"distance"`
	Duration    SummaryV2Duration      `json:"duration"`
	Speed       SummaryV2Speed         `json:"speed"`
	Energy      SummaryV2DrivingEnergy `json:"energy"`
	Consumption SummaryV2Consumption   `json:"consumption"`
	Power       SummaryV2DrivingPower  `json:"power"`
}

// SummaryV2Speed 描述平均速度和最高速度指标。
type SummaryV2Speed struct {
	Average *float64 `json:"average"`
	Max     *int     `json:"max"`
}

// SummaryV2DrivingEnergy 描述行程能耗指标。
type SummaryV2DrivingEnergy struct {
	Used              *float64 `json:"used"`
	Regenerated       *float64 `json:"regenerated"`
	RegenerationRatio *float64 `json:"regeneration_ratio"`
}

// SummaryV2Consumption 描述能耗效率指标。
type SummaryV2Consumption struct {
	Average *float64 `json:"average"`
	Best    *float64 `json:"best"`
	Worst   *float64 `json:"worst"`
}

// SummaryV2DrivingPower 描述驱动功率和动能回收功率峰值。
type SummaryV2DrivingPower struct {
	PeakDrive *int `json:"peak_drive"`
	PeakRegen *int `json:"peak_regen"`
}

// SummaryV2Charging 描述摘要中的充电指标。
type SummaryV2Charging struct {
	Count      int                     `json:"count"`
	Duration   SummaryV2Duration       `json:"duration"`
	Energy     SummaryV2ChargingEnergy `json:"energy"`
	Power      SummaryV2ChargingPower  `json:"power"`
	Efficiency *float64                `json:"efficiency"`
}

// SummaryV2ChargingEnergy 描述充电能量指标。
type SummaryV2ChargingEnergy struct {
	Added        float64  `json:"added"`
	ChargerUsed  *float64 `json:"charger_used"`
	LargestAdded *float64 `json:"largest_added"`
	AverageAdded *float64 `json:"average_added"`
}

// SummaryV2ChargingPower 描述充电功率指标。
type SummaryV2ChargingPower struct {
	Average *float64 `json:"average"`
	Max     *int     `json:"max"`
}

// SummaryV2Parking 描述停车摘要指标。
type SummaryV2Parking struct {
	Count          int                     `json:"count"`
	DurationMin    int                     `json:"duration_min"`
	AverageMin     *float64                `json:"average_min"`
	LongestMin     *int                    `json:"longest_min"`
	DominantState  *string                 `json:"dominant_state"`
	ParkedShare    *float64                `json:"parked_share"`
	StateBreakdown []ParkingStateBreakdown `json:"state_breakdown"`
}

// SummaryV2Battery 描述电池摘要指标。
type SummaryV2Battery struct {
	SOC                SummaryV2NullableRange `json:"soc"`
	RatedRange         SummaryV2NullableRange `json:"rated_range"`
	VampireDrainEnergy *float64               `json:"vampire_drain_energy"`
}

// SummaryV2NullableRange 描述可空的起止数值。
type SummaryV2NullableRange struct {
	Start *float64 `json:"start"`
	End   *float64 `json:"end"`
}

// SummaryV2Efficiency 描述效率摘要指标。
type SummaryV2Efficiency struct {
	DriveAverageConsumption *float64 `json:"drive_average_consumption"`
	DriveBestConsumption    *float64 `json:"drive_best_consumption"`
	DriveWorstConsumption   *float64 `json:"drive_worst_consumption"`
	GrossConsumption        *float64 `json:"gross_consumption"`
	ChargingEfficiency      *float64 `json:"charging_efficiency"`
	ConsumptionOverhead     *float64 `json:"consumption_overhead"`
	RegenerationRatio       *float64 `json:"regeneration_ratio"`
}

// SummaryV2Cost 描述充电成本摘要指标。
type SummaryV2Cost struct {
	Total          *float64 `json:"total"`
	Average        *float64 `json:"average"`
	Highest        *float64 `json:"highest"`
	AveragePerKwh  *float64 `json:"average_per_kwh"`
	Per100Distance *float64 `json:"per_100_distance"`
	Currency       string   `json:"currency"`
}

// SummaryV2Quality 描述数据质量标记和计数。
type SummaryV2Quality struct {
	DataComplete              *bool `json:"data_complete"`
	RegenerationEstimated     bool  `json:"regeneration_estimated"`
	LowSpeedDriveCount        int   `json:"low_speed_drive_count"`
	CongestionLikeDriveCount  int   `json:"congestion_like_drive_count"`
	HighConsumptionDriveCount int   `json:"high_consumption_drive_count"`
	LowEfficiencyChargeCount  int   `json:"low_efficiency_charge_count"`
	AbnormalChargeCount       int   `json:"abnormal_charge_count"`
	CostAvailable             bool  `json:"cost_available"`
	GrossConsumptionAvailable bool  `json:"gross_consumption_available"`
	BatterySnapshotAvailable  bool  `json:"battery_snapshot_available"`
}

// SummaryV2VehicleState 描述当前状态和历史状态汇总字段。
type SummaryV2VehicleState struct {
	Current         *string          `json:"current"`
	LastStateChange *string          `json:"last_state_change"`
	Breakdown       []StateBreakdown `json:"breakdown"`
}
