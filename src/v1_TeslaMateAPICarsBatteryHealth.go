package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsBatteryHealthV1 godoc
//
// @Summary 获取车辆电池健康度
// @Tags v1
// @Produce json
// @Param CarID path int true "车辆 ID"
// @Success 200 {object} V1JSONEnvelope
// @Failure 200 {object} V1ErrorEnvelope
// @Router /v1/cars/{CarID}/battery-health [get]
func TeslaMateAPICarsBatteryHealthV1(c *gin.Context) {
	var CarsBatteryHealthError1 = "Unable to load battery health data."
	CarID := convertStringToInteger(c.Param("CarID"))

	// creating structs for /cars/<CarID>/battery-health
	// Car struct - child of Data
	type Car struct {
		CarID   int        `json:"car_id"`   // smallint
		CarName NullString `json:"car_name"` // text (nullable)
	}
	// EstimatedCapacityAtFullCharge struct - child of BatteryHealth (added)
	type EstimatedCapacityAtFullCharge struct {
		Rated float64 `json:"rated"`
		Ideal float64 `json:"ideal"`
	}
	// BatteryHealth struct - child of Data
	type BatteryHealth struct {
		MaxRange                      float64                       `json:"max_range"`                                  // float64
		CurrentRange                  float64                       `json:"current_range"`                              // float64
		MaxCapacity                   float64                       `json:"max_capacity"`                               // float64
		CurrentCapacity               float64                       `json:"current_capacity"`                           // float64
		RatedEfficiency               float64                       `json:"rated_efficiency"`                           // float64
		BatteryHealthPercentage       float64                       `json:"battery_health_percentage"`                  // float64
		Cycles                        *float64                      `json:"cycles,omitempty"`                           // (added)
		DataLost                      *float64                      `json:"data_lost,omitempty"`                        // (added)
		LfpBattery                    *bool                         `json:"lfp_battery,omitempty"`                      // (added)
		EstimatedCapacityAtFullCharge EstimatedCapacityAtFullCharge `json:"estimated_capacity_at_full_charge"`          // (added)
		DerivedEfficiency             *float64                      `json:"derived_efficiency,omitempty"`               // (added)
		SamplesUsed                   *int64                        `json:"samples_used,omitempty"`                     // (added)
	}
	// TeslaMateUnits struct - child of Data
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`      // string
		UnitsTemperature string `json:"unit_of_temperature"` // string
	}
	// Data struct - child of JSONData
	type Data struct {
		Car            Car `json:"car"` // 车辆
		BatteryHealth  BatteryHealth  `json:"battery_health"`
		TeslaMateUnits TeslaMateUnits `json:"units"` // 单位
	}
	// JSONData struct - main
	type JSONData struct {
		Data Data `json:"data"` // 响应数据
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
	MaxRangeRated AS (
		SELECT
			CASE
				WHEN sum(usable_battery_level) = 0 THEN sum(rated_battery_range_km) * 100
				ELSE sum(rated_battery_range_km) / sum(usable_battery_level) * 100
			END AS range
		FROM (
			SELECT
				battery_level,
				usable_battery_level,
				date,
				rated_battery_range_km
			FROM charges c 
				INNER JOIN charging_processes p ON p.id = c.charging_process_id 
			WHERE
				p.car_id = $1
				AND usable_battery_level IS NOT NULL
		) AS data
		GROUP BY date_trunc('day', date)
		ORDER BY range DESC
		LIMIT 1
	),
	MaxRangeIdeal AS (
		SELECT
			CASE
				WHEN sum(usable_battery_level) = 0 THEN sum(ideal_battery_range_km) * 100
				ELSE sum(ideal_battery_range_km) / sum(usable_battery_level) * 100
			END AS range
		FROM (
			SELECT
				battery_level,
				usable_battery_level,
				date,
				ideal_battery_range_km
			FROM charges c 
				INNER JOIN charging_processes p ON p.id = c.charging_process_id 
			WHERE
				p.car_id = $1
				AND usable_battery_level IS NOT NULL
		) AS data
		GROUP BY date_trunc('day', date)
		ORDER BY range DESC
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
	WHERE cars.id = $1;`

	// execute query
	err := db.QueryRow(query, CarID).Scan(
		&MaxRangeRated,
		&MaxRangeIdeal,
		&CurrentRangeRated,
		&CurrentRangeIdeal,
		&MaxCapacity,
		&CurrentCapacity,
		&Efficiency,
		&PreferredRange,
		&UnitsLength,
		&UnitsTemperature,
		&CarName,
	)

	// checking for errors in query
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsBatteryHealthV1", CarsBatteryHealthError1, err.Error())
		return
	}

	// Create battery health object
	batteryHealth := BatteryHealth{
		CurrentCapacity:         CurrentCapacity,
		MaxCapacity:             MaxCapacity,
		RatedEfficiency:         Efficiency,
		BatteryHealthPercentage: 0,
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

	// Estimated capacity at full charge (rated/ideal): max_range × efficiency / 100.
	// efficiency here is in percent (Wh/km × 100), so divide by 100 to get kWh.
	if Efficiency > 0 {
		batteryHealth.EstimatedCapacityAtFullCharge.Rated = MaxRangeRated * Efficiency / 100.0 / 100.0
		batteryHealth.EstimatedCapacityAtFullCharge.Ideal = MaxRangeIdeal * Efficiency / 100.0 / 100.0
	}

	// Augment with cycles / data_lost / lfp_battery / derived_efficiency / samples_used.
	var (
		totalChargeEnergy sql.NullFloat64
		distanceSum       sql.NullFloat64
		odoMax            sql.NullFloat64
		odoMin            sql.NullFloat64
		lfpFlag           sql.NullBool
		samplesUsed       sql.NullInt64
	)
	auxErr := db.QueryRow(`
		SELECT
			(SELECT COALESCE(SUM(charge_energy_added), 0) FROM charging_processes WHERE car_id = $1 AND end_date IS NOT NULL),
			(SELECT COALESCE(SUM(distance), 0) FROM drives WHERE car_id = $1 AND end_date IS NOT NULL),
			(SELECT MAX(end_km) FROM drives WHERE car_id = $1 AND end_date IS NOT NULL),
			(SELECT MIN(start_km) FROM drives WHERE car_id = $1 AND end_date IS NOT NULL),
			(SELECT lfp_battery FROM car_settings WHERE id = $1),
			(
				SELECT COUNT(*) FROM charging_processes
				WHERE car_id = $1 AND end_date IS NOT NULL
				  AND duration_min > 10
				  AND end_battery_level <= 95
				  AND start_rated_range_km IS NOT NULL
				  AND end_rated_range_km IS NOT NULL
				  AND charge_energy_added > 0
			)
	`, CarID).Scan(&totalChargeEnergy, &distanceSum, &odoMax, &odoMin, &lfpFlag, &samplesUsed)
	if auxErr == nil {
		if MaxCapacity > 0 && totalChargeEnergy.Valid {
			cycles := totalChargeEnergy.Float64 / MaxCapacity
			batteryHealth.Cycles = &cycles
		}
		if odoMax.Valid && odoMin.Valid && distanceSum.Valid {
			lost := (odoMax.Float64 - odoMin.Float64) - distanceSum.Float64
			if lost < 0 {
				lost = 0
			}
			batteryHealth.DataLost = &lost
		}
		if lfpFlag.Valid {
			b := lfpFlag.Bool
			batteryHealth.LfpBattery = &b
		}
		if samplesUsed.Valid {
			n := samplesUsed.Int64
			batteryHealth.SamplesUsed = &n
		}
		if Efficiency > 0 {
			eff := Efficiency
			batteryHealth.DerivedEfficiency = &eff
		}
	}

	// converting values based on settings UnitsLength
	if UnitsLength == "mi" {
		batteryHealth.MaxRange = kilometersToMiles(batteryHealth.MaxRange)
		batteryHealth.CurrentRange = kilometersToMiles(batteryHealth.CurrentRange)
		if batteryHealth.DataLost != nil {
			conv := kilometersToMiles(*batteryHealth.DataLost)
			batteryHealth.DataLost = &conv
		}
	}

	jsonData := JSONData{
		Data{
			Car: Car{
				CarID:   CarID,
				CarName: CarName,
			},
			BatteryHealth: batteryHealth,
			TeslaMateUnits: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsBatteryHealthV1", jsonData)
}
