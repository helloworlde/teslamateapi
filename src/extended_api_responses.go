package main

type ExtendedResponseMeta struct {
	CarID       int    `json:"car_id,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Unit        string `json:"unit,omitempty"`
	GeneratedAt string `json:"generated_at,omitempty"`
	Version     string `json:"version,omitempty"`
}

type ExtendedWarning struct {
	Code    string `json:"code" example:"date_range_fallback"`
	Message string `json:"message" example:"invalid or missing date range, fallback to current month"`
	Metric  string `json:"metric,omitempty" example:"distance"`
	Scope   string `json:"scope,omitempty" example:"drives"`
}

type ExtendedRange struct {
	Start    string `json:"start" example:"2026-04-01T00:00:00+08:00"`
	End      string `json:"end" example:"2026-04-30T23:59:59+08:00"`
	Timezone string `json:"timezone" example:"Asia/Shanghai"`
}

type SummaryV2Envelope struct {
	Data     SummaryV2Data        `json:"data"`
	Meta     ExtendedResponseMeta `json:"meta"`
	Warnings []ExtendedWarning    `json:"warnings"`
}

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

type SummaryV2Car struct {
	CarID   int    `json:"car_id" example:"1"`
	CarName string `json:"car_name" example:"Model 3"`
}

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

type SummaryV2Duration struct {
	TotalMin   int      `json:"total_min"`
	AverageMin *float64 `json:"average_min"`
	LongestMin *int     `json:"longest_min"`
}

type SummaryV2Driving struct {
	Count       int                    `json:"count"`
	Distance    float64                `json:"distance"`
	Duration    SummaryV2Duration      `json:"duration"`
	Speed       SummaryV2Speed         `json:"speed"`
	Energy      SummaryV2DrivingEnergy `json:"energy"`
	Consumption SummaryV2Consumption   `json:"consumption"`
	Power       SummaryV2DrivingPower  `json:"power"`
}

type SummaryV2Speed struct {
	Average *float64 `json:"average"`
	Max     *int     `json:"max"`
}

type SummaryV2DrivingEnergy struct {
	Used              *float64 `json:"used"`
	Regenerated       *float64 `json:"regenerated"`
	RegenerationRatio *float64 `json:"regeneration_ratio"`
}

type SummaryV2Consumption struct {
	Average *float64 `json:"average"`
	Best    *float64 `json:"best"`
	Worst   *float64 `json:"worst"`
}

type SummaryV2DrivingPower struct {
	PeakDrive *int `json:"peak_drive"`
	PeakRegen *int `json:"peak_regen"`
}

type SummaryV2Charging struct {
	Count      int                     `json:"count"`
	Duration   SummaryV2Duration       `json:"duration"`
	Energy     SummaryV2ChargingEnergy `json:"energy"`
	Power      SummaryV2ChargingPower  `json:"power"`
	Efficiency *float64                `json:"efficiency"`
}

type SummaryV2ChargingEnergy struct {
	Added        float64  `json:"added"`
	ChargerUsed  *float64 `json:"charger_used"`
	LargestAdded *float64 `json:"largest_added"`
	AverageAdded *float64 `json:"average_added"`
}

type SummaryV2ChargingPower struct {
	Average *float64 `json:"average"`
	Max     *int     `json:"max"`
}

type SummaryV2Parking struct {
	Count          int                     `json:"count"`
	DurationMin    int                     `json:"duration_min"`
	AverageMin     *float64                `json:"average_min"`
	LongestMin     *int                    `json:"longest_min"`
	DominantState  *string                 `json:"dominant_state"`
	ParkedShare    *float64                `json:"parked_share"`
	StateBreakdown []ParkingStateBreakdown `json:"state_breakdown"`
}

type SummaryV2Battery struct {
	SOC                SummaryV2NullableRange `json:"soc"`
	RatedRange         SummaryV2NullableRange `json:"rated_range"`
	VampireDrainEnergy *float64               `json:"vampire_drain_energy"`
}

type SummaryV2NullableRange struct {
	Start *float64 `json:"start"`
	End   *float64 `json:"end"`
}

type SummaryV2Efficiency struct {
	DriveAverageConsumption *float64 `json:"drive_average_consumption"`
	DriveBestConsumption    *float64 `json:"drive_best_consumption"`
	DriveWorstConsumption   *float64 `json:"drive_worst_consumption"`
	GrossConsumption        *float64 `json:"gross_consumption"`
	ChargingEfficiency      *float64 `json:"charging_efficiency"`
	ConsumptionOverhead     *float64 `json:"consumption_overhead"`
	RegenerationRatio       *float64 `json:"regeneration_ratio"`
}

type SummaryV2Cost struct {
	Total          *float64 `json:"total"`
	Average        *float64 `json:"average"`
	Highest        *float64 `json:"highest"`
	AveragePerKwh  *float64 `json:"average_per_kwh"`
	Per100Distance *float64 `json:"per_100_distance"`
	Currency       string   `json:"currency"`
}

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

type SummaryV2VehicleState struct {
	Current         *string          `json:"current"`
	LastStateChange *string          `json:"last_state_change"`
	Breakdown       []StateBreakdown `json:"breakdown"`
}

type DashboardV2Envelope struct {
	Data     DashboardV2Data      `json:"data"`
	Meta     ExtendedResponseMeta `json:"meta"`
	Warnings []ExtendedWarning    `json:"warnings"`
}

type DashboardV2Data struct {
	CarID         int                      `json:"car_id"`
	Range         ExtendedRange            `json:"range"`
	Current       DashboardCurrentSnapshot `json:"current"`
	Statistics    StatisticsSummary        `json:"statistics"`
	Calendar      CalendarV2Data           `json:"calendar"`
	Series        []MetricSeriesV2         `json:"series"`
	Distributions []MetricDistributionV2   `json:"distributions"`
	Insights      []InsightV2Item          `json:"insights"`
	RecentDrives  []DashboardRecentDrive   `json:"recent_drives"`
	RecentCharges []DashboardRecentCharge  `json:"recent_charges"`
	RecentUpdates []DashboardRecentUpdate  `json:"recent_updates"`
}

type DashboardCurrentSnapshot struct {
	Position map[string]any `json:"position"`
	State    map[string]any `json:"state"`
	Charge   map[string]any `json:"charge"`
}

type DashboardRecentDrive struct {
	DriveID            int      `json:"drive_id"`
	StartTime          string   `json:"start_time"`
	EndTime            string   `json:"end_time"`
	StartAddress       *string  `json:"start_address"`
	EndAddress         *string  `json:"end_address"`
	Distance           float64  `json:"distance"`
	DurationSeconds    int      `json:"duration_seconds"`
	MaxSpeed           *int     `json:"max_speed"`
	AverageSpeed       *float64 `json:"average_speed"`
	EnergyUsed         *float64 `json:"energy_used"`
	ConsumptionNet     *float64 `json:"consumption_net"`
	OutsideTemperature *float64 `json:"outside_temperature"`
}

type DashboardRecentCharge struct {
	ChargeID           int      `json:"charge_id"`
	StartTime          string   `json:"start_time"`
	EndTime            string   `json:"end_time"`
	Location           *string  `json:"location"`
	ChargerType        *string  `json:"charger_type"`
	DurationSeconds    int      `json:"duration_seconds"`
	EnergyAdded        *float64 `json:"energy_added"`
	EnergyUsed         *float64 `json:"energy_used"`
	Cost               *float64 `json:"cost"`
	ChargingEfficiency *float64 `json:"charging_efficiency"`
	StartBatteryLevel  *int     `json:"start_battery_level"`
	EndBatteryLevel    *int     `json:"end_battery_level"`
	StartRatedRange    *float64 `json:"start_rated_range"`
	EndRatedRange      *float64 `json:"end_rated_range"`
	OutsideTemperature *float64 `json:"outside_temperature"`
}

type DashboardRecentUpdate struct {
	UpdateID  int     `json:"update_id"`
	StartTime string  `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Version   *string `json:"version"`
}

type CalendarV2Envelope struct {
	Data     CalendarV2Data       `json:"data"`
	Meta     ExtendedResponseMeta `json:"meta"`
	Warnings []ExtendedWarning    `json:"warnings"`
}

type CalendarV2Data struct {
	CarID   int                `json:"car_id"`
	Range   ExtendedRange      `json:"range"`
	Bucket  string             `json:"bucket" example:"day"`
	Summary CalendarSummaryV2  `json:"summary"`
	Items   []CalendarBucketV2 `json:"items"`
}

type CalendarSummaryV2 map[string]any

type CalendarBucketV2 map[string]any

type StatisticsV2Envelope struct {
	Data     StatisticsV2Data     `json:"data"`
	Meta     ExtendedResponseMeta `json:"meta"`
	Warnings []ExtendedWarning    `json:"warnings"`
}

type StatisticsV2Data struct {
	CarID    int                  `json:"car_id"`
	Period   string               `json:"period" example:"month"`
	Range    ExtendedRange        `json:"range"`
	Overview StatisticsOverviewV2 `json:"overview"`
	Drive    StatisticsDriveV2    `json:"drive"`
	Charge   StatisticsChargeV2   `json:"charge"`
	Battery  StatisticsBatteryV2  `json:"battery"`
}

type StatisticsOverviewV2 map[string]any
type StatisticsDriveV2 map[string]any
type StatisticsChargeV2 map[string]any
type StatisticsBatteryV2 map[string]any

type SeriesV2Envelope struct {
	Data     SeriesV2Data         `json:"data"`
	Meta     ExtendedResponseMeta `json:"meta"`
	Warnings []ExtendedWarning    `json:"warnings"`
}

type SeriesV2Data struct {
	CarID  int              `json:"car_id"`
	Scope  string           `json:"scope" example:"drives"`
	Bucket string           `json:"bucket" example:"day"`
	Range  ExtendedRange    `json:"range"`
	Series []MetricSeriesV2 `json:"series"`
}

type MetricSeriesV2 struct {
	Metric    string                `json:"metric" example:"distance"`
	Name      string                `json:"name" example:"distance"`
	Unit      string                `json:"unit" example:"km"`
	ChartType string                `json:"chart_type" example:"bar"`
	Points    []MetricSeriesPointV2 `json:"points"`
}

type MetricSeriesPointV2 struct {
	Time  string   `json:"time" example:"2026-04-01T00:00:00+08:00"`
	Value *float64 `json:"value"`
}

type DistributionsV2Envelope struct {
	Data     DistributionsV2Data  `json:"data"`
	Meta     ExtendedResponseMeta `json:"meta"`
	Warnings []ExtendedWarning    `json:"warnings"`
}

type DistributionsV2Data struct {
	CarID         int                    `json:"car_id"`
	Range         ExtendedRange          `json:"range"`
	Distributions []MetricDistributionV2 `json:"distributions"`
}

type MetricDistributionV2 struct {
	Metric    string                       `json:"metric" example:"drive_distance"`
	Name      string                       `json:"name" example:"drive_distance"`
	Unit      string                       `json:"unit" example:"count"`
	ChartType string                       `json:"chart_type" example:"bar"`
	Buckets   []MetricDistributionBucketV2 `json:"buckets"`
}

type MetricDistributionBucketV2 struct {
	Label string  `json:"label" example:"10-20"`
	From  *int    `json:"from,omitempty"`
	To    *int    `json:"to,omitempty"`
	Count int     `json:"count"`
	Value float64 `json:"value"`
}

type InsightsV2Envelope struct {
	Data     InsightsV2Data       `json:"data"`
	Meta     ExtendedResponseMeta `json:"meta"`
	Warnings []ExtendedWarning    `json:"warnings"`
}

type InsightsV2Data struct {
	CarID    int              `json:"car_id"`
	Range    ExtendedRange    `json:"range"`
	Summary  InsightSummaryV2 `json:"summary"`
	Insights []InsightV2Item  `json:"insights"`
}

type InsightSummaryV2 struct {
	PositiveCount int `json:"positive_count"`
	WarningCount  int `json:"warning_count"`
	InfoCount     int `json:"info_count"`
	TotalCount    int `json:"total_count"`
}

type InsightV2Item struct {
	Type         string         `json:"type" example:"efficiency"`
	Level        string         `json:"level" example:"info"`
	Title        string         `json:"title"`
	Message      string         `json:"message"`
	Metric       string         `json:"metric,omitempty"`
	Current      any            `json:"current,omitempty"`
	Baseline     any            `json:"baseline,omitempty"`
	DeltaPercent *float64       `json:"delta_percent,omitempty"`
	Related      map[string]any `json:"related,omitempty"`
}

type TimelineV2Envelope struct {
	Data       []TimelineEventV2    `json:"data"`
	Pagination v1Pagination         `json:"pagination"`
	Meta       ExtendedResponseMeta `json:"meta"`
	Warnings   []ExtendedWarning    `json:"warnings"`
}

type TimelineEventV2 struct {
	ID         string         `json:"id"`
	Type       string         `json:"type" example:"drive"`
	StartDate  string         `json:"start_date"`
	EndDate    *string        `json:"end_date"`
	Title      string         `json:"title"`
	Summary    map[string]any `json:"summary"`
	EntityType string         `json:"entity_type" example:"drive"`
	EntityID   int            `json:"entity_id"`
}

type VisitedMapV2Envelope struct {
	Data     VisitedMapV2Data     `json:"data"`
	Meta     ExtendedResponseMeta `json:"meta"`
	Warnings []ExtendedWarning    `json:"warnings"`
}

type VisitedMapV2Data struct {
	CarID         int                 `json:"car_id"`
	Range         ExtendedRange       `json:"range"`
	DistanceKm    *float64            `json:"distance_km"`
	DriveCount    *int                `json:"drive_count"`
	Bounds        *VisitedMapBoundsV2 `json:"bounds,omitempty"`
	VisitedPoints []VisitedPointV2    `json:"visited_points"`
	Heatmap       []any               `json:"heatmap"`
}

type VisitedMapBoundsV2 struct {
	North float64 `json:"north"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	West  float64 `json:"west"`
}

type VisitedPointV2 struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Count     int     `json:"count"`
}

type LocationsV2Envelope struct {
	Data     LocationsV2Data      `json:"data"`
	Meta     ExtendedResponseMeta `json:"meta"`
	Warnings []ExtendedWarning    `json:"warnings"`
}

type LocationsV2Data struct {
	CarID     int                   `json:"car_id"`
	Range     ExtendedRange         `json:"range"`
	Summary   LocationsSummaryV2    `json:"summary"`
	Locations []LocationAggregateV2 `json:"locations"`
}

type LocationsSummaryV2 struct {
	LocationCount       int      `json:"location_count"`
	ReturnedCount       int      `json:"returned_count"`
	DriveLocationCount  int      `json:"drive_location_count"`
	ChargeLocationCount int      `json:"charge_location_count"`
	DriveStartCount     int      `json:"drive_start_count"`
	DriveEndCount       int      `json:"drive_end_count"`
	ChargeCount         int      `json:"charge_count"`
	ChargeEnergyKwh     float64  `json:"charge_energy_kwh"`
	ChargeCost          *float64 `json:"charge_cost"`
}

type LocationAggregateV2 struct {
	Name            string   `json:"name"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	DriveStartCount int      `json:"drive_start_count"`
	DriveEndCount   int      `json:"drive_end_count"`
	DriveCount      int      `json:"drive_count"`
	ChargeCount     int      `json:"charge_count"`
	ChargeEnergyKwh *float64 `json:"charge_energy_kwh"`
	ChargeCost      *float64 `json:"charge_cost"`
	TotalEventCount int      `json:"total_event_count"`
	LastSeen        *string  `json:"last_seen"`
}
