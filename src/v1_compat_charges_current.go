package main

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

const (
	maxChargeInactivityThresholdMinutes = 15
)

// TeslaMateAPICarsChargesCurrentV1 返回兼容响应结构的当前充电数据。
// @Summary 当前充电
// @Tags 兼容 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} CurrentChargeV1Envelope
// @Router /v1/cars/{CarID}/charges/current [get]
func TeslaMateAPICarsChargesCurrentV1(c *gin.Context) {

	// 定义错误消息。
	var (
		CarsChargesCurrentError1 = "Unable to load current charge."
		CarsChargesCurrentError2 = "Unable to load current charge details."
		CarsChargesCurrentError3 = "No active charging in progress."
	)

	// 从 URL 读取车辆 ID。
	CarID := convertStringToInteger(c.Param("CarID"))

	var (
		CarName                       NullString
		charge                        CurrentChargeV1
		ChargeDetailsData             []CurrentChargeDetailRowV1
		UnitsLength, UnitsTemperature string
		isCharging                    bool
	)

	// 创建临时变量以处理数据库 NULL 值。
	var (
		startRatedRange, currentRatedRange     sql.NullFloat64
		startBatteryLevel, currentBatteryLevel sql.NullInt64
		chargeEnergyAdded, cost                sql.NullFloat64
		outsideTempAvg                         sql.NullFloat64
		odometer                               sql.NullFloat64
		durationMin                            sql.NullFloat64
		durationStr, address                   sql.NullString
	)

	// 构建带首选续航设置的查询。
	query := `
		SELECT
			charging_processes.id AS charge_id,
			start_date,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, nullif(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS address,
			(SELECT charge_energy_added FROM charges WHERE charging_process_id = charging_processes.id ORDER BY id DESC LIMIT 1) AS charge_energy_added,
			COALESCE(cost, 0) AS cost,
	        (SELECT rated_battery_range_km FROM charges WHERE charging_process_id = charging_processes.id ORDER BY id ASC LIMIT 1) AS start_rated_range,
			(SELECT rated_battery_range_km FROM charges WHERE charging_process_id = charging_processes.id ORDER BY id DESC LIMIT 1) AS current_rated_range,
			(SELECT battery_level FROM charges WHERE charging_process_id = charging_processes.id ORDER BY date ASC LIMIT 1) AS start_battery_level,
			(SELECT battery_level FROM charges WHERE charging_process_id = charging_processes.id ORDER BY id DESC LIMIT 1) AS current_battery_level,
			EXTRACT(EPOCH FROM (COALESCE(end_date, NOW()) - start_date))/60 AS duration_min,
			TO_CHAR((EXTRACT(EPOCH FROM (COALESCE(end_date, NOW()) - start_date))/60 * INTERVAL '1 minute'), 'HH24:MI') as duration_str,
			(SELECT outside_temp FROM charges WHERE charging_process_id = charging_processes.id ORDER BY id DESC LIMIT 1) AS outside_temp_avg,
			position.odometer as odometer,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name,
			end_date IS NULL AS is_charging
		FROM charging_processes
		LEFT JOIN cars ON car_id = cars.id
		LEFT JOIN addresses address ON address_id = address.id
		LEFT JOIN positions position ON position_id = position.id
		LEFT JOIN geofences geofence ON geofence_id = geofence.id
		WHERE charging_processes.car_id=$1
		ORDER BY end_date IS NULL DESC, start_date DESC
		LIMIT 1;`

	row := db.QueryRow(query, CarID)

	// 将当前行扫描到临时变量以处理 NULL 值。
	err := row.Scan(
		&charge.ChargeID,
		&charge.StartDate,
		&address,
		&chargeEnergyAdded,
		&cost,
		&startRatedRange,
		&currentRatedRange,
		&startBatteryLevel,
		&currentBatteryLevel,
		&durationMin,
		&durationStr,
		&outsideTempAvg,
		&odometer,
		&UnitsLength,
		&UnitsTemperature,
		&CarName,
		&isCharging,
	)

	switch err {
	case sql.ErrNoRows:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", "No current charge found.", "No rows were returned")
		return
	case nil:
		// 查询成功，继续组装响应。
		break
	default:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError1, err.Error())
		return
	}

	charge.IsCharging = isCharging
	if address.Valid {
		charge.Address = address.String
	} else {
		charge.Address = "Unknown"
	}

	if chargeEnergyAdded.Valid {
		charge.ChargeEnergyAdded = chargeEnergyAdded.Float64
	}

	if cost.Valid {
		charge.Cost = cost.Float64
	}

	if startRatedRange.Valid {
		charge.RatedRange.StartRange = startRatedRange.Float64
	}

	if currentRatedRange.Valid {
		charge.RatedRange.CurrentRange = currentRatedRange.Float64
	}

	if startBatteryLevel.Valid {
		charge.BatteryDetails.StartBatteryLevel = int(startBatteryLevel.Int64)
	}

	if currentBatteryLevel.Valid {
		charge.BatteryDetails.CurrentBatteryLevel = int(currentBatteryLevel.Int64)
	}

	if durationMin.Valid {
		charge.DurationMin = int(durationMin.Float64) // 将 float64 转换为 int。
	}

	if durationStr.Valid {
		charge.DurationStr = durationStr.String
	}

	if outsideTempAvg.Valid {
		charge.OutsideTempAvg = outsideTempAvg.Float64
	}

	if odometer.Valid {
		charge.Odometer = odometer.Float64
	}

	// 根据长度单位设置转换数值。
	if UnitsLength == "mi" {
		charge.RatedRange.StartRange = kilometersToMiles(charge.RatedRange.StartRange)
		charge.RatedRange.CurrentRange = kilometersToMiles(charge.RatedRange.CurrentRange)
		charge.RatedRange.AddedRange = kilometersToMiles(charge.RatedRange.AddedRange)
		charge.Odometer = kilometersToMiles(charge.Odometer)
	}
	// 根据温度单位设置转换数值。
	if UnitsTemperature == "F" && outsideTempAvg.Valid {
		charge.OutsideTempAvg = celsiusToFahrenheit(charge.OutsideTempAvg)
	}

	// 按用户配置时区转换时间字段。
	charge.StartDate = getTimeInTimeZone(charge.StartDate)

	// 从数据库读取充电采样明细。
	detailsQuery := `
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
		ORDER BY id DESC;`
	rows, err := db.Query(detailsQuery, charge.ChargeID)

	// 检查查询错误。
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError2, err.Error())
		return
	}

	// 延迟关闭结果集。
	defer rows.Close()

	// 遍历查询结果。
	for rows.Next() {
		// 创建临时变量以处理 NULL 值。
		var (
			detailBatteryLevel, detailUsableBatteryLevel                        sql.NullInt64
			detailChargeEnergyAdded, detailRatedBatteryRange, detailOutsideTemp sql.NullFloat64
			detailConnChargeCable, detailFastChargerType                        sql.NullString
			detailFastChargerBrand                                              sql.NullString
		)

		// 创建充电采样对象。
		chargedetails := CurrentChargeDetailRowV1{}

		// 将当前行扫描到临时变量。
		err = rows.Scan(
			&chargedetails.DetailID,
			&chargedetails.Date,
			&detailBatteryLevel,
			&detailUsableBatteryLevel,
			&detailChargeEnergyAdded,
			&chargedetails.NotEnoughPowerToHeat,
			&chargedetails.ChargerDetails.ChargerActualCurrent,
			&chargedetails.ChargerDetails.ChargerPhases,
			&chargedetails.ChargerDetails.ChargerPilotCurrent,
			&chargedetails.ChargerDetails.ChargerPower,
			&chargedetails.ChargerDetails.ChargerVoltage,
			&detailRatedBatteryRange,
			&chargedetails.BatteryInfo.BatteryHeater,
			&chargedetails.BatteryInfo.BatteryHeaterOn,
			&chargedetails.BatteryInfo.BatteryHeaterNoPower,
			&detailConnChargeCable,
			&chargedetails.FastChargerInfo.FastChargerPresent,
			&detailFastChargerBrand,
			&detailFastChargerType,
			&detailOutsideTemp,
		)

		// 处理 NULL 值。
		if detailBatteryLevel.Valid {
			chargedetails.BatteryLevel = int(detailBatteryLevel.Int64)
		}

		if detailUsableBatteryLevel.Valid {
			chargedetails.UsableBatteryLevel = int(detailUsableBatteryLevel.Int64)
		}

		if detailChargeEnergyAdded.Valid {
			chargedetails.ChargeEnergyAdded = detailChargeEnergyAdded.Float64
		}

		if detailRatedBatteryRange.Valid {
			chargedetails.BatteryInfo.RatedBatteryRange = detailRatedBatteryRange.Float64
		}

		// 使用 interface{} 正确表达字符串字段的 NULL 值。
		if detailConnChargeCable.Valid {
			chargedetails.ConnChargeCable = detailConnChargeCable.String
		} else {
			chargedetails.ConnChargeCable = nil
		}

		// 过滤 fast_charger_brand 和 fast_charger_type 的无效占位值。
		if detailFastChargerBrand.Valid && detailFastChargerBrand.String != "<invalid>" {
			chargedetails.FastChargerInfo.FastChargerBrand = &detailFastChargerBrand.String
		} else {
			chargedetails.FastChargerInfo.FastChargerBrand = nil
		}

		if detailFastChargerType.Valid && detailFastChargerType.String != "<invalid>" {
			chargedetails.FastChargerInfo.FastChargerType = &detailFastChargerType.String
		} else {
			chargedetails.FastChargerInfo.FastChargerType = nil
		}

		if detailOutsideTemp.Valid {
			chargedetails.OutsideTemp = detailOutsideTemp.Float64
		}

		// 根据长度单位设置转换数值。
		if UnitsLength == "mi" && detailRatedBatteryRange.Valid {
			chargedetails.BatteryInfo.RatedBatteryRange = kilometersToMiles(chargedetails.BatteryInfo.RatedBatteryRange)
		}

		// 根据温度单位设置转换数值。
		if UnitsTemperature == "F" && detailOutsideTemp.Valid {
			chargedetails.OutsideTemp = celsiusToFahrenheit(chargedetails.OutsideTemp)
		}

		// 按用户配置时区转换时间字段。
		chargedetails.Date = getTimeInTimeZone(chargedetails.Date)

		chargedetails.ChargerDetails.ChargerPhases = normalizeChargerPhases(chargedetails.ChargerDetails.ChargerPhases)

		// 检查扫描错误。
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError2, err.Error())
			return
		}

		// 追加充电采样到响应列表。
		ChargeDetailsData = append(ChargeDetailsData, chargedetails)
	}

	// 检查结果集遍历错误。
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError2, err.Error())
		return
	}

	// 检查充电采样列表是否包含记录。
	if len(ChargeDetailsData) > 0 {
		// 解析最近一条充电采样时间。
		latestDetailDate, err := time.Parse(time.RFC3339, ChargeDetailsData[0].Date)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError2, "Error parsing charge detail date")
			return
		}

		// 计算最近采样距当前的时间。
		timeElapsed := time.Since(latestDetailDate)

		// 最近采样超过阈值时认为当前充电已不活跃。
		if timeElapsed.Minutes() > maxChargeInactivityThresholdMinutes {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError3, "No active charging in progress. There are incomplete charges but last update was more than 15 minutes ago.")
			return
		}
	}

	// 将充电采样写入充电对象。
	charge.ChargeDetails = ChargeDetailsData

	if charge.RatedRange.StartRange == 0 && len(ChargeDetailsData) > 0 {
		charge.RatedRange.StartRange = ChargeDetailsData[len(ChargeDetailsData)-1].BatteryInfo.RatedBatteryRange
	}

	if addedRange := charge.RatedRange.CurrentRange - charge.RatedRange.StartRange; addedRange > 0 {
		charge.RatedRange.AddedRange = addedRange
	} else {
		charge.RatedRange.AddedRange = 0
	}

	jsonData := CurrentChargeV1Envelope{
		Data: CurrentChargeV1Data{
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
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsChargesCurrentV1", jsonData)
}

// normalizeChargerPhases 将相位值归一化为有效配置。
// 2 相和 3 相都归一化为 3。
// 其他值默认按单相处理。
func normalizeChargerPhases(phases int) int {
	switch phases {
	case 2, 3:
		return 3
	default:
		return 1
	}
}
