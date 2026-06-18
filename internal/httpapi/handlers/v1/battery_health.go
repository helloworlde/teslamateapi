package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// TeslaMateAPICarsBatteryHealthV1 returns derived battery-health metrics.
//
// @Summary      Battery health
// @Description  Returns max/current range and capacity along with a derived health %.
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path      int  true  "TeslaMate cars.id"
// @Success      200    {object}  dto.V1BatteryHealthResponse
// @Failure      401    {object}  dto.ErrorEnvelope
// @Router       /api/v1/cars/{CarID}/battery-health [get]
func (h *Handler) BatteryHealth(c *gin.Context) {
	const handler = "TeslaMateAPICarsBatteryHealthV1"
	var CarsBatteryHealthError1 = "Unable to load battery health data."
	CarID, ok := requirePositiveIntParam(c, handler, "CarID", c.Param("CarID"))
	if !ok {
		return
	}

	// creating required vars
	var (
		CarName                       NullString
		Efficiency                    float64
		MaxRangeRated                 float64
		MaxRangeIdeal                 float64
		CurrentRangeRated             float64
		CurrentRangeIdeal             float64
		MaxCapacity                   float64
		CurrentCapacity               float64
		CurrentBatteryLevel           float64
		PreferredRange                string
		UnitsLength, UnitsTemperature string
	)

	query := `
	WITH Aux as (
		SELECT 
			car_id,
			COALESCE(derived_efficiency, car_efficiency) AS efficiency
		FROM (
			SELECT
				ROUND((charge_energy_added / NULLIF(end_rated_range_km - start_rated_range_km, 0))::numeric, 3) * 100 AS derived_efficiency,
				COUNT(*) as count,
				cars.id as car_id,
				cars.efficiency * 100 AS car_efficiency
			FROM cars
				LEFT JOIN charging_processes ON
					cars.id = charging_processes.car_id 
					AND duration_min > 10
					AND end_battery_level <= 95
					AND start_rated_range_km IS NOT NULL
					AND end_rated_range_km IS NOT NULL
					AND charge_energy_added > 0
			WHERE cars.id = $1
			GROUP BY 1, 3, 4
			ORDER BY 2 DESC
			LIMIT 1
		) AS Efficiency
	),
	CurrentCapacity AS (
		SELECT
			AVG(Capacity) AS Capacity
		FROM (
			SELECT 
				c.rated_battery_range_km * aux.efficiency / c.usable_battery_level AS Capacity
			FROM charging_processes cp
				INNER JOIN charges c ON c.charging_process_id = cp.id 
				INNER JOIN aux ON cp.car_id = aux.car_id
			WHERE
				cp.car_id = $1
				AND cp.end_date IS NOT NULL
				AND cp.charge_energy_added >= aux.efficiency
				AND c.usable_battery_level > 0
			ORDER BY cp.end_date DESC, c.date desc
			LIMIT 100
		) AS lastCharges
	),
	MaxCapacity AS (
		SELECT 
			MAX(c.rated_battery_range_km * aux.efficiency / c.usable_battery_level) AS Capacity
		FROM charging_processes cp
			INNER JOIN (
				SELECT
					charging_process_id,
					MAX(date) as date FROM charges WHERE usable_battery_level > 0 GROUP BY charging_process_id
			) AS gcharges ON
				cp.id = gcharges.charging_process_id
			INNER JOIN charges c ON
				c.charging_process_id = cp.id
				AND c.date = gcharges.date
			INNER JOIN aux ON cp.car_id = aux.car_id
		WHERE
			cp.car_id = $1
			AND cp.end_date IS NOT NULL
			AND cp.charge_energy_added >= aux.efficiency
	),
	CurrentRangeRated AS (
		SELECT
			(range * 100.0 / usable_battery_level) AS range
		FROM (
			(
				SELECT
					date,
					rated_battery_range_km AS range,
					usable_battery_level AS usable_battery_level
				FROM positions
				WHERE
					car_id = $1
					AND ideal_battery_range_km IS NOT NULL
					AND usable_battery_level > 0 
				ORDER BY date DESC
				LIMIT 1
			)
			UNION ALL
			(
				SELECT date,
					rated_battery_range_km AS range,
					usable_battery_level as usable_battery_level
				FROM charges c
					INNER JOIN charging_processes p ON p.id = c.charging_process_id
				WHERE
					p.car_id = $1
					AND usable_battery_level > 0
				ORDER BY date DESC
				LIMIT 1
			)
		) AS data
		ORDER BY date DESC
		LIMIT 1
	),
	CurrentRangeIdeal AS (
		SELECT
			(range * 100.0 / usable_battery_level) AS range
		FROM (
			(
				SELECT
					date,
					ideal_battery_range_km AS range,
					usable_battery_level AS usable_battery_level
				FROM positions
				WHERE
					car_id = $1
					AND ideal_battery_range_km IS NOT NULL
					AND usable_battery_level > 0 
				ORDER BY date DESC
				LIMIT 1
			)
			UNION ALL
			(
				SELECT date,
					ideal_battery_range_km AS range,
					usable_battery_level as usable_battery_level
				FROM charges c
					INNER JOIN charging_processes p ON p.id = c.charging_process_id
				WHERE
					p.car_id = $1
					AND usable_battery_level > 0
				ORDER BY date DESC
				LIMIT 1
			)
		) AS data
		ORDER BY date DESC
		LIMIT 1
	),
	MaxRangeDaily AS (
		-- Per-day SOC-normalised rated and ideal range, computed in a single
		-- pass over this car's charges. MaxRangeRated and MaxRangeIdeal used to
		-- scan this (large) charges ⋈ charging_processes join twice; they now
		-- share one scan and each picks its own maximum from the tiny per-day
		-- result. Result-identical: same WHERE/GROUP BY, and each maximum is
		-- still "ORDER BY <range> DESC LIMIT 1" over the same per-day values
		-- (preserving NULLS-FIRST ordering, so edge-case NULL days behave as
		-- before). Referenced twice, so Postgres materialises it once.
		SELECT
			CASE
				WHEN sum(usable_battery_level) = 0 THEN sum(rated_battery_range_km) * 100
				ELSE sum(rated_battery_range_km) / sum(usable_battery_level) * 100
			END AS rated_range,
			CASE
				WHEN sum(usable_battery_level) = 0 THEN sum(ideal_battery_range_km) * 100
				ELSE sum(ideal_battery_range_km) / sum(usable_battery_level) * 100
			END AS ideal_range
		FROM charges c
			INNER JOIN charging_processes p ON p.id = c.charging_process_id
		WHERE
			p.car_id = $1
			AND usable_battery_level IS NOT NULL
		GROUP BY date_trunc('day', date)
	),
	MaxRangeRated AS (
		SELECT rated_range AS range
		FROM MaxRangeDaily
		ORDER BY rated_range DESC
		LIMIT 1
	),
	MaxRangeIdeal AS (
		SELECT ideal_range AS range
		FROM MaxRangeDaily
		ORDER BY ideal_range DESC
		LIMIT 1
	),
	CurrentBatteryLevel AS (
		SELECT usable_battery_level
		FROM (
			(
				SELECT date, usable_battery_level
				FROM positions
				WHERE car_id = $1 AND usable_battery_level > 0
				ORDER BY date DESC
				LIMIT 1
			)
			UNION ALL
			(
				SELECT c.date, c.usable_battery_level
				FROM charges c
					INNER JOIN charging_processes p ON p.id = c.charging_process_id
				WHERE p.car_id = $1 AND c.usable_battery_level > 0
				ORDER BY c.date DESC
				LIMIT 1
			)
		) AS data
		ORDER BY date DESC
		LIMIT 1
	)
	SELECT
		COALESCE(MaxRangeRated.range, 0) as max_range_rated,
		COALESCE(MaxRangeIdeal.range, 0) as max_range_ideal,
		COALESCE(CurrentRangeRated.range, 0) as current_range_rated,
		COALESCE(CurrentRangeIdeal.range, 0) as current_range_ideal,
		COALESCE(MaxCapacity.Capacity, 0) as max_capacity,
		COALESCE(CurrentCapacity.Capacity, 0) as current_capacity,
		COALESCE(aux.efficiency, 0) as efficiency,
		COALESCE(CurrentBatteryLevel.usable_battery_level, 0) as current_battery_level,
		(SELECT preferred_range FROM settings LIMIT 1) as preferred_range,
		(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
		(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
		cars.name
	FROM cars
		LEFT JOIN MaxRangeRated ON true
		LEFT JOIN MaxRangeIdeal ON true
		LEFT JOIN CurrentRangeRated ON true
		LEFT JOIN CurrentRangeIdeal ON true
		LEFT JOIN Aux ON cars.id = aux.car_id
		LEFT JOIN MaxCapacity ON true
		LEFT JOIN CurrentCapacity ON true
		LEFT JOIN CurrentBatteryLevel ON true
	WHERE cars.id = $1;`

	// execute query
	err := h.db.QueryRowContext(c.Request.Context(), query, CarID).Scan(
		&MaxRangeRated,
		&MaxRangeIdeal,
		&CurrentRangeRated,
		&CurrentRangeIdeal,
		&MaxCapacity,
		&CurrentCapacity,
		&Efficiency,
		&CurrentBatteryLevel,
		&PreferredRange,
		&UnitsLength,
		&UnitsTemperature,
		&CarName,
	)

	// checking for errors in query
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsBatteryHealthV1", CarsBatteryHealthError1, err.Error())
		return
	}

	// Create battery health object
	batteryHealth := dto.V1BatteryHealth{
		CurrentCapacity:         CurrentCapacity,
		MaxCapacity:             MaxCapacity,
		RatedEfficiency:         Efficiency,
		BatteryHealthPercentage: 0,
		CurrentBatteryLevel:     CurrentBatteryLevel,
	}

	// Select the correct range based on preferred_range setting
	if PreferredRange == "ideal" {
		batteryHealth.MaxRange = MaxRangeIdeal
		batteryHealth.CurrentRange = CurrentRangeIdeal
	} else {
		batteryHealth.MaxRange = MaxRangeRated
		batteryHealth.CurrentRange = CurrentRangeRated
	}

	// Calculate battery health percentage
	if MaxCapacity > 0 {
		batteryHealth.BatteryHealthPercentage = (CurrentCapacity / MaxCapacity) * 100
	}

	// converting values based on settings UnitsLength
	if UnitsLength == "mi" {
		batteryHealth.MaxRange = convert.KilometersToMiles(batteryHealth.MaxRange)
		batteryHealth.CurrentRange = convert.KilometersToMiles(batteryHealth.CurrentRange)
	}

	// Predicted (currently drivable) range is the SOC-normalised current range
	// scaled back down to the latest battery level. Computed after unit
	// conversion so it inherits CurrentRange's units.
	batteryHealth.PredictedRange = batteryHealth.CurrentRange * CurrentBatteryLevel / 100

	jsonData := dto.V1BatteryHealthResponse{
		Data: dto.V1BatteryHealthData{
			Car: dto.Car{
				CarID:   CarID,
				CarName: CarName,
			},
			BatteryHealth: batteryHealth,
			TeslaMateUnits: dto.TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	respond.HandleSuccess(c, "TeslaMateAPICarsBatteryHealthV1", jsonData)
}
