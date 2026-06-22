package v2

// Shared SQL fragments for parking (vampire-drain) math, kept in one place so
// the lifetime and summary stats queries derive parking intervals and energy
// drop identically. Both fragments are bound to car_id $1.

// drivePairsCTE is the body of the `dp` CTE: consecutive completed drives
// paired via LEAD so each row spans one parking interval — from the end of a
// drive (park_start) to the start of the next (park_end). park_end is NULL for
// the still-open final park. Composed as `dp AS (` + drivePairsCTE + `)`.
const drivePairsCTE = `
			SELECT
				d.id AS drive_id,
				d.end_date AS park_start,
				LEAD(d.start_date) OVER w AS park_end,
				d.end_position_id,
				LEAD(d.start_position_id) OVER w AS next_start_position_id
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			WINDOW w AS (PARTITION BY d.car_id ORDER BY d.start_date ASC)`

// parkingEnergyDropKWh estimates a single parking interval's vampire drain in
// kWh from a `dp` row joined to its start/end positions (sp = position at end
// of the prior drive, ep = position at start of the next). It only attributes
// drain to intervals with no charging session inside them. Two branches:
//   - rated-range delta available → (range drop) × efficiency;
//   - else fall back to usable-battery-level delta scaled into range × efficiency.
// Composes where sp, ep, cars.efficiency and dp.park_start/park_end are in
// scope, bound to car_id $1.
const parkingEnergyDropKWh = `CASE
					WHEN sp.rated_battery_range_km IS NOT NULL AND ep.rated_battery_range_km IS NOT NULL
					AND sp.rated_battery_range_km > ep.rated_battery_range_km
					AND NOT EXISTS(SELECT 1 FROM charging_processes cp
						WHERE cp.car_id = $1 AND cp.start_date >= dp.park_start
						AND (dp.park_end IS NULL OR cp.start_date < dp.park_end))
					THEN (sp.rated_battery_range_km - ep.rated_battery_range_km) * cars.efficiency
					WHEN sp.rated_battery_range_km IS NOT NULL
					AND COALESCE(sp.usable_battery_level, sp.battery_level) > 0
					AND NOT EXISTS(SELECT 1 FROM charging_processes cp
						WHERE cp.car_id = $1 AND cp.start_date >= dp.park_start
						AND (dp.park_end IS NULL OR cp.start_date < dp.park_end))
					THEN GREATEST(
						COALESCE(sp.usable_battery_level, sp.battery_level)
						- COALESCE(ep.usable_battery_level, ep.battery_level),
						0
					) * sp.rated_battery_range_km * cars.efficiency
						/ COALESCE(sp.usable_battery_level, sp.battery_level)
					ELSE 0
				END`
