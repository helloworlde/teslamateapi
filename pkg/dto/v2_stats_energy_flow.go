package dto

// V2EnergyFlowNode is one node in the energy/cost Sankey model.
type V2EnergyFlowNode struct {
	ID        string  `json:"id" example:"wall_input"`
	Label     string  `json:"label" example:"Wall input"`
	EnergyKWh float64 `json:"energy_kwh" example:"17200.0"`
	Cost      float64 `json:"cost" example:"4321.50"`
}

// V2EnergyFlowLink is one directed flow between Sankey nodes.
type V2EnergyFlowLink struct {
	Source          string  `json:"source" example:"wall_input"`
	Target          string  `json:"target" example:"vehicle_added"`
	EnergyKWh       float64 `json:"energy_kwh" example:"16500.0"`
	Cost            float64 `json:"cost" example:"4145.74"`
	PercentOfSource float64 `json:"percent_of_source" example:"95.93"`
}

// V2EnergyFlowMetrics contains the canonical accounting figures behind the
// Sankey model. Energy figures are kWh; cost figures use charging_processes.cost
// currency; distance cost follows the user's unit_of_length.
type V2EnergyFlowMetrics struct {
	WallEnergyKWh         float64 `json:"wall_energy_kwh" example:"17200.0"`
	VehicleEnergyAddedKWh float64 `json:"vehicle_energy_added_kwh" example:"16500.0"`
	// VehicleAvailableEnergyKWh is the vehicle-side pool that destinations draw
	// from: vehicle_energy_added_kwh + start_battery_energy_kwh, widened to the
	// recorded consumption when that is larger (incomplete history).
	VehicleAvailableEnergyKWh    float64 `json:"vehicle_available_energy_kwh" example:"16620.0"`
	StartBatteryEnergyKWh        float64 `json:"start_battery_energy_kwh" example:"120.0"`
	EndBatteryEnergyKWh          float64 `json:"end_battery_energy_kwh" example:"95.0"`
	BatteryInventoryDeltaKWh     float64 `json:"battery_inventory_delta_kwh" example:"-25.0"`
	ChargingLossEnergyKWh        float64 `json:"charging_loss_energy_kwh" example:"700.0"`
	DrivingEnergyKWh             float64 `json:"driving_energy_kwh" example:"7825.0"`
	ParkingEnergyKWh             float64 `json:"parking_energy_kwh" example:"125.0"`
	UnattributedVehicleEnergyKWh float64 `json:"unattributed_vehicle_energy_kwh" example:"8455.0"`
	UnmatchedVehicleUsageKWh     float64 `json:"unmatched_vehicle_usage_kwh" example:"0"`
	// TotalChargingCost is the summed charging_processes.cost over all charges.
	TotalChargingCost float64 `json:"total_charging_cost" example:"4321.50"`
	// WallCostPerKWh is total cost / wall-side energy, where wall-side energy is
	// GREATEST(charge_energy_used, charge_energy_added). This is the plug-side
	// price (charging loss is in the denominator). Because TeslaMate's
	// charge_energy_used is often missing/unreliable, the GREATEST fallback can
	// make this collapse to the battery-side rate (charging_cost_per_kwh).
	WallCostPerKWh float64 `json:"wall_cost_per_kwh" example:"0.25125"`
	// ChargingCostPerKWh is total cost / charge_energy_added (battery-side): the
	// cost of each kWh that actually entered the pack. Charging loss is baked in,
	// so it reads higher than wall_cost_per_kwh.
	ChargingCostPerKWh float64 `json:"charging_cost_per_kwh" example:"0.26191"`
	// VehicleAccountingCostPerKWh is the price used to value the vehicle-side
	// energy buckets below: vehicle_added cost spread across
	// vehicle_available_energy_kwh. Because free starting inventory (and any
	// zero-cost usage gap) sit in that denominator, this reads BELOW
	// wall_cost_per_kwh. It is used for the conserved flow graph node/link costs;
	// the metrics-level driving_cost uses charging_cost_per_kwh.
	VehicleAccountingCostPerKWh float64 `json:"vehicle_accounting_cost_per_kwh" example:"0.24943"`
	// DrivingCost is driving energy valued at charging_cost_per_kwh
	// (SUM(cost) / SUM(charge_energy_added)). It counts only the energy that
	// moved the car; charging loss is NOT folded into this drive-cost estimate.
	DrivingCost float64 `json:"driving_cost" example:"1965.0"`
	// DrivingCostPerDistance is driving_cost / distance (per km, or per mile
	// when unit_of_length is mi). Same battery-side charge-price basis as
	// driving_cost.
	DrivingCostPerDistance float64 `json:"driving_cost_per_distance" example:"0.04624"`
	// ParkingCost is parking/idle drain energy valued at the accounting price.
	ParkingCost float64 `json:"parking_cost" example:"31.41"`
	// ChargingLossCost is the charging-loss energy (wall − vehicle) valued at the
	// wall price; carried separately so it is not amortised into driving_cost.
	ChargingLossCost float64 `json:"charging_loss_cost" example:"175.87"`
	// EndBatteryCost is the energy still stored in the pack at window end, valued
	// at the accounting price (paid for but not yet consumed).
	EndBatteryCost            float64 `json:"end_battery_cost" example:"23.72"`
	ActualDrivingUsageRatePct float64 `json:"actual_driving_usage_rate_pct" example:"45.49"`
	ActualLossRatePct         float64 `json:"actual_loss_rate_pct" example:"4.07"`
	ChargingEfficiencyPct     float64 `json:"charging_efficiency_pct" example:"95.93"`
	VehicleDrivingSharePct    float64 `json:"vehicle_driving_share_pct" example:"47.42"`
}

// V2EnergyFlowData is the `data` field of V2EnergyFlowResponse.
type V2EnergyFlowData struct {
	Car     Car                 `json:"car"`
	Nodes   []V2EnergyFlowNode  `json:"nodes"`
	Links   []V2EnergyFlowLink  `json:"links"`
	Metrics V2EnergyFlowMetrics `json:"metrics"`
	Units   TeslaMateUnits      `json:"units"`
	// BalanceStatus compares recorded consumption to the vehicle-available pool
	// (vehicle_added + starting inventory). One of: "balanced",
	// "usage_exceeds_supply" (consumption exceeds supply on incomplete history;
	// shortfall shown as the zero-cost usage gap), or
	// "has_unattributed_vehicle_energy" (supply exceeds consumption; surplus
	// shown as the unmetered vehicle-loss sink).
	BalanceStatus string `json:"balance_status" example:"balanced"`
}

// V2EnergyFlowResponse is the envelope for
// /api/v2/cars/{CarID}/stats/energy-flow.
type V2EnergyFlowResponse struct {
	Data V2EnergyFlowData `json:"data"`
}
