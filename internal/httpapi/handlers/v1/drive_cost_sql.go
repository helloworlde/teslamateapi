package v1

// v1DriveEnergyConsumedNetSQL is the row-level drive energy displayed by v1
// drive list/detail. It intentionally keeps the legacy rated-range basis
// because estimated_usage_cost must match the "energy_consumed_net" field shown
// on the same row.
const v1DriveEnergyConsumedNetSQL = `CASE
				WHEN (start_rated_range_km - end_rated_range_km) > 0
				THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency
				ELSE NULL
			END`

// v1EstimatedUsageCostSQL prices the row-level v1 energy with the average
// charge price. Do not use integer battery_level/SOC deltas here: short trips
// with the same 1% drop can have materially different rated-range energy.
const v1EstimatedUsageCostSQL = `CASE
				WHEN (start_rated_range_km - end_rated_range_km) > 0
					AND charge_price.cost_per_kwh IS NOT NULL
				THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency * charge_price.cost_per_kwh
				ELSE NULL
			END`
