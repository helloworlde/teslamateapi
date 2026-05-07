package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsChargesDetailsV1 返回兼容响应结构的充电详情。
// @Summary 充电详情
// @Tags 兼容 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param ChargeID path int true "充电 ID"
// @Success 200 {object} ChargeDetailsV1Envelope
// @Router /v1/cars/{CarID}/charges/{ChargeID} [get]
func TeslaMateAPICarsChargesDetailsV1(c *gin.Context) {

	// 定义错误消息。
	var (
		CarsChargesDetailsError1 = "Unable to load charge."
		CarsChargesDetailsError2 = "Unable to load charge details."
	)

	// 从 URL 读取车辆 ID 和充电 ID。
	CarID := convertStringToInteger(c.Param("CarID"))
	ChargeID := convertStringToInteger(c.Param("ChargeID"))

	var (
		CarName                       NullString
		charge                        ChargeDetailFullV1
		ChargeDetailsData             []ChargeDetailRowV1
		UnitsLength, UnitsTemperature string
	)

	// 从数据库读取充电主记录。
	query := `
		SELECT
			charging_processes.id AS charge_id,
			charging_processes.start_date,
			charging_processes.end_date,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, nullif(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS address,
			COALESCE(charging_processes.charge_energy_added, 0) AS charge_energy_added,
			COALESCE(GREATEST(charge_energy_used, charging_processes.charge_energy_added), 0) AS charge_energy_used,
			COALESCE(cost, 0) AS cost,
			start_ideal_range_km AS start_ideal_range,
			end_ideal_range_km AS end_ideal_range,
			start_rated_range_km AS start_rated_range,
			end_rated_range_km AS end_rated_range,
			start_battery_level,
			end_battery_level,
			duration_min,
			TO_CHAR((duration_min * INTERVAL '1 minute'), 'HH24:MI') as duration_str,
			outside_temp_avg,
			position.odometer as odometer,
			position.latitude,
			position.longitude,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name
		FROM charging_processes
		LEFT JOIN cars ON car_id = cars.id
		LEFT JOIN addresses address ON address_id = address.id
		LEFT JOIN positions position ON position_id = position.id
		LEFT JOIN geofences geofence ON geofence_id = geofence.id
		LEFT JOIN charges ON charging_processes.id = charges.id
		WHERE charging_processes.car_id=$1 AND charging_processes.id=$2 AND charging_processes.end_date IS NOT NULL
		ORDER BY start_date DESC;`
	row := db.QueryRow(query, CarID, ChargeID)

	// 将当前行扫描到充电详情对象。
	err := row.Scan(
		&charge.ChargeID,
		&charge.StartDate,
		&charge.EndDate,
		&charge.Address,
		&charge.ChargeEnergyAdded,
		&charge.ChargeEnergyUsed,
		&charge.Cost,
		&charge.RangeIdeal.StartRange,
		&charge.RangeIdeal.EndRange,
		&charge.RangeRated.StartRange,
		&charge.RangeRated.EndRange,
		&charge.BatteryDetails.StartBatteryLevel,
		&charge.BatteryDetails.EndBatteryLevel,
		&charge.DurationMin,
		&charge.DurationStr,
		&charge.OutsideTempAvg,
		&charge.Odometer,
		&charge.Latitude,
		&charge.Longitude,
		&UnitsLength,
		&UnitsTemperature,
		&CarName,
	)

	switch err {
	case sql.ErrNoRows:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesDetailsV1", "No rows were returned!", err.Error())
		return
	case nil:
		// 查询成功，继续组装响应。
		break
	default:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesDetailsV1", CarsChargesDetailsError1, err.Error())
		return
	}

	// 根据长度单位设置转换数值。
	if UnitsLength == "mi" {
		charge.RangeIdeal.StartRange = kilometersToMiles(charge.RangeIdeal.StartRange)
		charge.RangeIdeal.EndRange = kilometersToMiles(charge.RangeIdeal.EndRange)
		charge.RangeRated.StartRange = kilometersToMiles(charge.RangeRated.StartRange)
		charge.RangeRated.EndRange = kilometersToMiles(charge.RangeRated.EndRange)
		charge.Odometer = kilometersToMiles(charge.Odometer)
	}
	// 根据温度单位设置转换数值。
	if UnitsTemperature == "F" {
		charge.OutsideTempAvg = celsiusToFahrenheit(charge.OutsideTempAvg)
	}

	// 按用户配置时区转换时间字段。
	charge.StartDate = getTimeInTimeZone(charge.StartDate)
	charge.EndDate = getTimeInTimeZone(charge.EndDate)

	// 从数据库读取充电采样明细。
	query = `
 			SELECT
				id AS detail_id,
				date,
				battery_level,
				usable_battery_level,
				charge_energy_added,
				not_enough_power_to_heat,
				COALESCE(charger_actual_current, 0) as charger_actual_current,
				COALESCE(charger_phases, 0) AS charger_phases,
				COALESCE(charger_pilot_current, 0) as charger_pilot_current,
				COALESCE(charger_power, 0) as charger_power,
				COALESCE(charger_voltage, 0) as charger_voltage,
				ideal_battery_range_km AS ideal_battery_range,
				rated_battery_range_km AS rated_battery_range,
				battery_heater,
				battery_heater_on,
				battery_heater_no_power,
				conn_charge_cable,
				fast_charger_present,
				fast_charger_brand,
				fast_charger_type,
				outside_temp
			FROM charges
			WHERE charging_process_id=$1
			ORDER BY id ASC;`
	rows, err := db.Query(query, ChargeID)

	// 检查查询错误。
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesDetailsV1", CarsChargesDetailsError2, err.Error())
		return
	}

	// 延迟关闭结果集。
	defer rows.Close()

	// 遍历查询结果。
	for rows.Next() {

		// 创建充电采样对象。
		chargedetails := ChargeDetailRowV1{}

		// 将当前行扫描到充电采样对象。
		err = rows.Scan(
			&chargedetails.DetailID,
			&chargedetails.Date,
			&chargedetails.BatteryLevel,
			&chargedetails.UsableBatteryLevel,
			&chargedetails.ChargeEnergyAdded,
			&chargedetails.NotEnoughPowerToHeat,
			&chargedetails.ChargerDetails.ChargerActualCurrent,
			&chargedetails.ChargerDetails.ChargerPhases,
			&chargedetails.ChargerDetails.ChargerPilotCurrent,
			&chargedetails.ChargerDetails.ChargerPower,
			&chargedetails.ChargerDetails.ChargerVoltage,
			&chargedetails.BatteryInfo.IdealBatteryRange,
			&chargedetails.BatteryInfo.RatedBatteryRange,
			&chargedetails.BatteryInfo.BatteryHeater,
			&chargedetails.BatteryInfo.BatteryHeaterOn,
			&chargedetails.BatteryInfo.BatteryHeaterNoPower,
			&chargedetails.ConnChargeCable,
			&chargedetails.FastChargerInfo.FastChargerPresent,
			&chargedetails.FastChargerInfo.FastChargerBrand,
			&chargedetails.FastChargerInfo.FastChargerType,
			&chargedetails.OutsideTemp,
		)

		// 根据长度单位设置转换数值。
		if UnitsLength == "mi" {
			chargedetails.BatteryInfo.IdealBatteryRange = kilometersToMiles(chargedetails.BatteryInfo.IdealBatteryRange)
			chargedetails.BatteryInfo.RatedBatteryRange = kilometersToMiles(chargedetails.BatteryInfo.RatedBatteryRange)

		}
		// 根据温度单位设置转换数值。
		if UnitsTemperature == "F" {
			chargedetails.OutsideTemp = celsiusToFahrenheit(chargedetails.OutsideTemp)
		}
		// 按用户配置时区转换时间字段。
		chargedetails.Date = getTimeInTimeZone(chargedetails.Date)

		// 检查扫描错误。
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesDetailsV1", CarsChargesDetailsError2, err.Error())
			return
		}

		// 追加充电采样到响应列表。
		ChargeDetailsData = append(ChargeDetailsData, chargedetails)
		charge.ChargeDetails = ChargeDetailsData
	}

	// 检查结果集遍历错误。
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesDetailsV1", CarsChargesDetailsError2, err.Error())
		return
	}

	jsonData := ChargeDetailsV1Envelope{
		Data: ChargeDetailsV1Data{
			Car: CarRefV1{
				CarID:   CarID,
				CarName: CarName,
			},
			Charge: charge,
			TeslaMateUnits: UnitsLengthTempV1{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// 返回响应数据。
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsChargesDetailsV1", jsonData)
}
