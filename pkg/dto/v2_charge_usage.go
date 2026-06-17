package dto

import "github.com/tobiasehlert/teslamateapi/pkg/nullable"

// V2ChargeUsageCycle identifies the charge-to-next-charge analysis window.
type V2ChargeUsageCycle struct {
	ChargeID          int            `json:"charge_id" example:"1234"`
	NextChargeID      nullable.Int64 `json:"next_charge_id" swaggertype:"integer" example:"1235"`
	ChargeStartDate   string         `json:"charge_start_date" example:"2024-01-01T20:00:00+01:00"`
	ChargeEndDate     string         `json:"charge_end_date" example:"2024-01-01T22:30:00+01:00"`
	AnalysisStartDate string         `json:"analysis_start_date" example:"2024-01-01T22:30:00+01:00"`
	AnalysisEndDate   string         `json:"analysis_end_date" example:"2024-01-05T08:15:00+01:00"`
	IsComplete        bool           `json:"is_complete" example:"true"`
	DurationMin       int            `json:"duration_min" example:"4890"`
	DurationStr       string         `json:"duration_str" example:"81:30"`
}

// V2ChargeUsageBattery carries SOC and battery-inventory snapshots for a
// single charge usage cycle. Energy figures are kWh.
type V2ChargeUsageBattery struct {
	StartBatteryLevel      nullable.Int64   `json:"start_battery_level" swaggertype:"integer" example:"28"`
	PostChargeBatteryLevel nullable.Int64   `json:"post_charge_battery_level" swaggertype:"integer" example:"82"`
	EndBatteryLevel        nullable.Int64   `json:"end_battery_level" swaggertype:"integer" example:"35"`
	AddedBatteryLevel      nullable.Int64   `json:"added_battery_level" swaggertype:"integer" example:"54"`
	UsedBatteryLevel       nullable.Int64   `json:"used_battery_level" swaggertype:"integer" example:"47"`
	StartBatteryEnergyKWh  nullable.Float64 `json:"start_battery_energy_kwh" swaggertype:"number" example:"18.2"`
	PostChargeEnergyKWh    nullable.Float64 `json:"post_charge_battery_energy_kwh" swaggertype:"number" example:"53.3"`
	EndBatteryEnergyKWh    nullable.Float64 `json:"end_battery_energy_kwh" swaggertype:"number" example:"22.8"`
}

// V2ChargeUsageMetrics contains the accounting numbers for one charge usage
// cycle. "Available" means battery inventory at analysis start, i.e. when the
// selected charging session ended.
type V2ChargeUsageMetrics struct {
	WallEnergyKWh                float64          `json:"wall_energy_kwh" example:"40.1"`
	ChargeEnergyAddedKWh         float64          `json:"charge_energy_added_kwh" example:"38.4"`
	ChargingLossEnergyKWh        float64          `json:"charging_loss_energy_kwh" example:"1.7"`
	ChargeCost                   float64          `json:"charge_cost" example:"12.35"`
	WallCostPerKWh               float64          `json:"wall_cost_per_kwh" example:"0.30798"`
	ChargeCostPerKWh             float64          `json:"charge_cost_per_kwh" example:"0.32161"`
	InventoryExpectedEnergyKWh   nullable.Float64 `json:"inventory_expected_energy_kwh" swaggertype:"number" example:"56.6"`
	InventoryReconciliationKWh   nullable.Float64 `json:"inventory_reconciliation_kwh" swaggertype:"number" example:"-1.2"`
	VehicleAvailableEnergyKWh    nullable.Float64 `json:"vehicle_available_energy_kwh" swaggertype:"number" example:"56.6"`
	DrivingEnergyKWh             nullable.Float64 `json:"driving_energy_kwh" swaggertype:"number" example:"22.4"`
	ParkingEnergyKWh             nullable.Float64 `json:"parking_energy_kwh" swaggertype:"number" example:"2.1"`
	EndBatteryEnergyKWh          nullable.Float64 `json:"end_battery_energy_kwh" swaggertype:"number" example:"22.8"`
	UntrackedEnergyKWh           nullable.Float64 `json:"untracked_energy_kwh" swaggertype:"number" example:"9.3"`
	UnmatchedUsageKWh            nullable.Float64 `json:"unmatched_usage_kwh" swaggertype:"number" example:"0"`
	BatteryUsedEnergyKWh         nullable.Float64 `json:"battery_used_energy_kwh" swaggertype:"number" example:"33.8"`
	BatteryUsedRatePct           nullable.Float64 `json:"battery_used_rate_pct" swaggertype:"number" example:"59.72"`
	TrackedUsageRatePct          nullable.Float64 `json:"tracked_usage_rate_pct" swaggertype:"number" example:"43.29"`
	DrivingShareOfAvailablePct   nullable.Float64 `json:"driving_share_of_available_pct" swaggertype:"number" example:"39.58"`
	ParkingShareOfAvailablePct   nullable.Float64 `json:"parking_share_of_available_pct" swaggertype:"number" example:"3.71"`
	RemainingShareOfAvailablePct nullable.Float64 `json:"remaining_share_of_available_pct" swaggertype:"number" example:"40.28"`
	UntrackedShareOfAvailablePct nullable.Float64 `json:"untracked_share_of_available_pct" swaggertype:"number" example:"16.43"`
	TotalDistance                float64          `json:"total_distance" example:"185.2"`
	DriveCount                   int              `json:"drive_count" example:"5"`
	DriveDurationMin             int              `json:"drive_duration_min" example:"260"`
	DrivingEnergyPerDistanceWh   nullable.Float64 `json:"driving_energy_per_distance_wh" swaggertype:"number" example:"120.95"`
}

// V2ChargeUsageDataQuality tells callers whether kWh figures were backed by
// complete rated-range snapshots. When false, SOC fields are still useful but
// kWh accounting should be treated as partial.
type V2ChargeUsageDataQuality struct {
	AccountingStatus           string   `json:"accounting_status" example:"complete" enums:"complete,partial"`
	RangeDataComplete          bool     `json:"range_data_complete" example:"true"`
	ReconciliationDataComplete bool     `json:"reconciliation_data_complete" example:"true"`
	HasPreChargeRangeData      bool     `json:"has_pre_charge_range_data" example:"true"`
	HasPostChargeRangeData     bool     `json:"has_post_charge_range_data" example:"true"`
	HasEndRangeData            bool     `json:"has_end_range_data" example:"true"`
	DrivesRangeDataComplete    bool     `json:"drives_range_data_complete" example:"true"`
	PartialFields              []string `json:"partial_fields" example:"start_battery_energy_kwh"`
}

// V2ChargeUsageNode is one node in the per-charge battery accounting model.
type V2ChargeUsageNode struct {
	ID                 string  `json:"id" example:"charge_added"`
	Label              string  `json:"label" example:"Charge added"`
	EnergyKWh          float64 `json:"energy_kwh" example:"38.4"`
	PercentOfAvailable float64 `json:"percent_of_available" example:"67.84"`
}

// V2ChargeUsageLink is one directed energy flow in the per-charge accounting
// model. PercentOfSource is relative to the source node's energy.
type V2ChargeUsageLink struct {
	Source          string  `json:"source" example:"cycle_available"`
	Target          string  `json:"target" example:"driving_usage"`
	EnergyKWh       float64 `json:"energy_kwh" example:"22.4"`
	PercentOfSource float64 `json:"percent_of_source" example:"39.58"`
}

// V2ChargeUsageData is the `data` field of V2ChargeUsageResponse.
type V2ChargeUsageData struct {
	Car           Car                      `json:"car"`
	Cycle         V2ChargeUsageCycle       `json:"cycle"`
	Battery       V2ChargeUsageBattery     `json:"battery"`
	Metrics       V2ChargeUsageMetrics     `json:"metrics"`
	Nodes         []V2ChargeUsageNode      `json:"nodes"`
	Links         []V2ChargeUsageLink      `json:"links"`
	Units         TeslaMateUnits           `json:"units"`
	DataQuality   V2ChargeUsageDataQuality `json:"data_quality"`
	BalanceStatus string                   `json:"balance_status" example:"balanced"`
}

// V2ChargeUsageResponse is the envelope for
// /api/v2/cars/{CarID}/charges/{ChargeID}/usage.
type V2ChargeUsageResponse struct {
	Data V2ChargeUsageData `json:"data"`
}
