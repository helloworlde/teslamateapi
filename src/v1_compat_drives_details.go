package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsDrivesDetailsV1 返回兼容响应结构的行程详情。
// @Summary 行程详情
// @Tags 兼容 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param DriveID path int true "行程 ID"
// @Success 200 {object} DriveDetailsV1Envelope
// @Router /v1/cars/{CarID}/drives/{DriveID} [get]
func TeslaMateAPICarsDrivesDetailsV1(c *gin.Context) {

	// 定义错误消息。
	var (
		CarsDrivesDetailsError1 = "Unable to load drive."
		CarsDrivesDetailsError2 = "Unable to load drive details."
	)

	// 从 URL 读取车辆 ID 和行程 ID。
	CarID := convertStringToInteger(c.Param("CarID"))
	DriveID := convertStringToInteger(c.Param("DriveID"))

	var (
		CarName                       NullString
		drive                         DriveDetailFullV1
		DriveDetailsData              []DrivePositionRowV1
		UnitsLength, UnitsTemperature string
	)

	// 从数据库读取行程主记录。
	query := `
		SELECT
			drives.id AS drive_id,
			start_date,
			end_date,
			COALESCE(start_geofence.name, CONCAT_WS(', ', COALESCE(start_address.name, nullif(CONCAT_WS(' ', start_address.road, start_address.house_number), '')), start_address.city)) AS start_address,
			COALESCE(end_geofence.name, CONCAT_WS(', ', COALESCE(end_address.name, nullif(CONCAT_WS(' ', end_address.road, end_address.house_number), '')), end_address.city)) AS end_address,
			start_km,
			end_km,
			distance,
			duration_min,
			TO_CHAR((duration_min * INTERVAL '1 minute'), 'HH24:MI') as duration_str,
			speed_max,
			COALESCE(distance / NULLIF(duration_min, 0) * 60, 0) AS speed_avg,
			power_max,
			power_min,
			COALESCE(start_position.usable_battery_level, start_position.battery_level) as start_usable_battery_level,
			start_position.battery_level as start_battery_level,
			COALESCE(end_position.usable_battery_level, end_position.battery_level) as end_usable_battery_level,
			end_position.battery_level as end_battery_level,
			case when ( start_position.battery_level != start_position.usable_battery_level OR end_position.battery_level != end_position.usable_battery_level ) = true then true else false end  as reduced_range,
			duration_min > 1 AND distance > 1 AND ( start_position.usable_battery_level IS NULL OR end_position.usable_battery_level IS NULL OR ( end_position.battery_level - end_position.usable_battery_level ) = 0 ) as is_sufficiently_precise,
			start_ideal_range_km,
			end_ideal_range_km,
			COALESCE( NULLIF ( GREATEST ( start_ideal_range_km - end_ideal_range_km, 0 ), 0 ),0 ) as range_diff_ideal_km,
			start_rated_range_km,
			end_rated_range_km,
			COALESCE( NULLIF ( GREATEST ( start_rated_range_km - end_rated_range_km, 0 ), 0 ),0 ) as range_diff_rated_km,
			outside_temp_avg,
			inside_temp_avg,
			CASE 
				WHEN (start_rated_range_km - end_rated_range_km) > 0 
				THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency 
				ELSE NULL 
			END as energy_consumed_net,
			CASE 
				WHEN (duration_min > 1 AND distance > 1 AND ( start_position.usable_battery_level IS NULL OR end_position.usable_battery_level IS NULL OR ( end_position.battery_level - end_position.usable_battery_level ) = 0 )) AND NULLIF(distance, 0) IS NOT NULL
				THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / NULLIF(distance, 0) * 1000
				ELSE NULL 
			END as consumption_net,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name
		FROM drives
		LEFT JOIN cars ON car_id = cars.id
		LEFT JOIN addresses start_address ON start_address_id = start_address.id
		LEFT JOIN addresses end_address ON end_address_id = end_address.id
		LEFT JOIN positions start_position ON start_position_id = start_position.id
		LEFT JOIN positions end_position ON end_position_id = end_position.id
		LEFT JOIN geofences start_geofence ON start_geofence_id = start_geofence.id
		LEFT JOIN geofences end_geofence ON end_geofence_id = end_geofence.id
		WHERE drives.car_id=$1 AND end_date IS NOT NULL AND drives.id = $2;`
	row := db.QueryRow(query, CarID, DriveID)

	// 将当前行扫描到行程详情对象。
	err := row.Scan(
		&drive.DriveID,
		&drive.StartDate,
		&drive.EndDate,
		&drive.StartAddress,
		&drive.EndAddress,
		&drive.OdometerDetails.OdometerStart,
		&drive.OdometerDetails.OdometerEnd,
		&drive.OdometerDetails.OdometerDistance,
		&drive.DurationMin,
		&drive.DurationStr,
		&drive.SpeedMax,
		&drive.SpeedAvg,
		&drive.PowerMax,
		&drive.PowerMin,
		&drive.BatteryDetails.StartUsableBatteryLevel,
		&drive.BatteryDetails.StartBatteryLevel,
		&drive.BatteryDetails.EndUsableBatteryLevel,
		&drive.BatteryDetails.EndBatteryLevel,
		&drive.BatteryDetails.ReducedRange,
		&drive.BatteryDetails.IsSufficientlyPrecise,
		&drive.RangeIdeal.StartRange,
		&drive.RangeIdeal.EndRange,
		&drive.RangeIdeal.RangeDiff,
		&drive.RangeRated.StartRange,
		&drive.RangeRated.EndRange,
		&drive.RangeRated.RangeDiff,
		&drive.OutsideTempAvg,
		&drive.InsideTempAvg,
		&drive.EnergyConsumedNet,
		&drive.ConsumptionNet,
		&UnitsLength,
		&UnitsTemperature,
		&CarName,
	)

	switch err {
	case sql.ErrNoRows:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", "No rows were returned!", err.Error())
		return
	case nil:
		// 查询成功，继续组装响应。
		break
	default:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError1, err.Error())
		return
	}

	// 根据长度单位设置转换数值。
	if UnitsLength == "mi" {
		drive.OdometerDetails.OdometerStart = kilometersToMiles(drive.OdometerDetails.OdometerStart)
		drive.OdometerDetails.OdometerEnd = kilometersToMiles(drive.OdometerDetails.OdometerEnd)
		drive.OdometerDetails.OdometerDistance = kilometersToMiles(drive.OdometerDetails.OdometerDistance)
		drive.SpeedMax = int(kilometersToMiles(float64(drive.SpeedMax)))
		drive.SpeedAvg = kilometersToMiles(drive.SpeedAvg)
		drive.RangeIdeal.StartRange = kilometersToMiles(drive.RangeIdeal.StartRange)
		drive.RangeIdeal.EndRange = kilometersToMiles(drive.RangeIdeal.EndRange)
		drive.RangeIdeal.RangeDiff = kilometersToMiles(drive.RangeIdeal.RangeDiff)
		drive.RangeRated.StartRange = kilometersToMiles(drive.RangeRated.StartRange)
		drive.RangeRated.EndRange = kilometersToMiles(drive.RangeRated.EndRange)
		drive.RangeRated.RangeDiff = kilometersToMiles(drive.RangeRated.RangeDiff)
		if drive.ConsumptionNet != nil {
			*drive.ConsumptionNet = whPerKmToWhPerMi(*drive.ConsumptionNet)
		}
	}
	// 根据温度单位设置转换数值。
	if UnitsTemperature == "F" {
		drive.OutsideTempAvg = celsiusToFahrenheit(drive.OutsideTempAvg)
		drive.InsideTempAvg = celsiusToFahrenheit(drive.InsideTempAvg)
	}
	// 按用户配置时区转换时间字段。
	drive.StartDate = getTimeInTimeZone(drive.StartDate)
	drive.EndDate = getTimeInTimeZone(drive.EndDate)

	// 从数据库读取行程位置采样明细。
	query = `
		 			SELECT
						id AS detail_id,
						date,
						latitude,
						longitude,
						COALESCE(speed, 0) AS speed,
						power,
						odometer,
						battery_level,
						usable_battery_level,
						elevation,
						inside_temp,
						outside_temp,
						is_climate_on,
						fan_status,
						driver_temp_setting,
						passenger_temp_setting,
						is_rear_defroster_on,
						is_front_defroster_on,
						est_battery_range_km,
						ideal_battery_range_km,
						rated_battery_range_km,
						battery_heater,
						battery_heater_on,
						battery_heater_no_power
		 			FROM positions
		 			WHERE drive_id = $1
		 			ORDER BY id ASC;`
	rows, err := db.Query(query, DriveID)

	// 检查查询错误。
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
		return
	}

	// 延迟关闭结果集。
	defer rows.Close()

	// 遍历查询结果。
	for rows.Next() {

		// 创建行程采样对象。
		drivedetails := DrivePositionRowV1{}

		// 将当前行扫描到行程采样对象。
		err = rows.Scan(
			&drivedetails.DetailID,
			&drivedetails.Date,
			&drivedetails.Latitude,
			&drivedetails.Longitude,
			&drivedetails.Speed,
			&drivedetails.Power,
			&drivedetails.Odometer,
			&drivedetails.BatteryLevel,
			&drivedetails.UsableBatteryLevel,
			&drivedetails.Elevation,
			&drivedetails.ClimateInfo.InsideTemp,
			&drivedetails.ClimateInfo.OutsideTemp,
			&drivedetails.ClimateInfo.IsClimateOn,
			&drivedetails.ClimateInfo.FanStatus,
			&drivedetails.ClimateInfo.DriverTempSetting,
			&drivedetails.ClimateInfo.PassengerTempSetting,
			&drivedetails.ClimateInfo.IsRearDefrosterOn,
			&drivedetails.ClimateInfo.IsFrontDefrosterOn,
			&drivedetails.BatteryInfo.EstBatteryRange,
			&drivedetails.BatteryInfo.IdealBatteryRange,
			&drivedetails.BatteryInfo.RatedBatteryRange,
			&drivedetails.BatteryInfo.BatteryHeater,
			&drivedetails.BatteryInfo.BatteryHeaterOn,
			&drivedetails.BatteryInfo.BatteryHeaterNoPower,
		)

		// 根据长度单位设置转换数值。
		if UnitsLength == "mi" {
			drivedetails.Odometer = kilometersToMiles(drivedetails.Odometer)
			drivedetails.Speed = int(kilometersToMiles(float64(drivedetails.Speed)))
			drivedetails.BatteryInfo.EstBatteryRange = kilometersToMilesNilSupport(drivedetails.BatteryInfo.EstBatteryRange)
			drivedetails.BatteryInfo.IdealBatteryRange = kilometersToMilesNilSupport(drivedetails.BatteryInfo.IdealBatteryRange)
			drivedetails.BatteryInfo.RatedBatteryRange = kilometersToMilesNilSupport(drivedetails.BatteryInfo.RatedBatteryRange)
		}
		// 根据温度单位设置转换数值。
		if UnitsTemperature == "F" {
			drivedetails.ClimateInfo.InsideTemp = celsiusToFahrenheitNilSupport(drivedetails.ClimateInfo.InsideTemp)
			drivedetails.ClimateInfo.OutsideTemp = celsiusToFahrenheitNilSupport(drivedetails.ClimateInfo.OutsideTemp)
			drivedetails.ClimateInfo.DriverTempSetting = celsiusToFahrenheitNilSupport(drivedetails.ClimateInfo.DriverTempSetting)
			drivedetails.ClimateInfo.PassengerTempSetting = celsiusToFahrenheitNilSupport(drivedetails.ClimateInfo.PassengerTempSetting)
		}
		// 按用户配置时区转换时间字段。
		drivedetails.Date = getTimeInTimeZone(drivedetails.Date)

		// 检查扫描错误。
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
			return
		}

		// 追加行程采样到响应列表。
		DriveDetailsData = append(DriveDetailsData, drivedetails)
		drive.DriveDetails = DriveDetailsData
	}

	// 检查结果集遍历错误。
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
		return
	}

	jsonData := DriveDetailsV1Envelope{
		Data: DriveDetailsV1Data{
			Car: CarRefV1{
				CarID:   CarID,
				CarName: CarName,
			},
			Drive: drive,
			TeslaMateUnits: UnitsLengthTempV1{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// 返回响应数据。
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsDrivesDetailsV1", jsonData)
}
