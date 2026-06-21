package v2

// Shared SQL fragments for drive consumption/energy math, kept in one place so
// the lifetime and summary stats queries stay in lock-step. These expressions
// reference the unqualified `drives` columns (distance, duration_min,
// start_rated_range_km, end_rated_range_km) and `cars.efficiency`, so they only
// compose inside a `FROM drives ... JOIN cars` scope.

// driveConsumptionFilter is the row predicate for a usable consumption sample:
// a real trip (>1 distance unit, >1 minute) with a measured positive
// rated-range drop. Used both as the CASE guard below and as array_agg FILTER.
const driveConsumptionFilter = `distance > 1 AND duration_min > 1 AND start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL AND GREATEST(start_rated_range_km - end_rated_range_km, 0) > 0`

// driveConsumptionWhPerKm is a single drive's consumption in Wh/km
// (rated-range drop × efficiency / distance), NULL for rows failing the filter.
const driveConsumptionWhPerKm = `CASE WHEN ` + driveConsumptionFilter + ` THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency / distance * 1000 ELSE NULL END`

// driveEnergyKWh is a single drive's consumed energy (rated-range drop ×
// efficiency, floored at 0). Unlike driveConsumptionWhPerKm it carries no
// trip-validity guard because it feeds raw energy/distance sums.
const driveEnergyKWh = `CASE WHEN start_rated_range_km IS NOT NULL AND end_rated_range_km IS NOT NULL THEN GREATEST(start_rated_range_km - end_rated_range_km, 0) * cars.efficiency ELSE 0 END`
