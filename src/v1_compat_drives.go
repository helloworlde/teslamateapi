package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsDrivesV1 返回兼容响应结构的行程历史。
// @Summary 行程列表
// @Description 兼容接口：保持历史行程列表响应封装。
// @Tags 兼容 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Param page query int false "页码"
// @Param show query int false "每页数量"
// @Param limit query int false "每页数量别名"
// @Param offset query int false "偏移量别名"
// @Param sort query string false "排序表达式"
// @Success 200 {object} DrivesListV1Envelope
// @Router /v1/cars/{CarID}/drives [get]
func TeslaMateAPICarsDrivesV1(c *gin.Context) {

	// 定义错误消息。
	var CarsDrivesError1 = "Unable to load drives."
	var CarsDrivesError2 = "Invalid date format."

	// 从 URL 读取车辆 ID。
	CarID := convertStringToInteger(c.Param("CarID"))
	// 读取分页和排序查询参数。
	ResultPage := convertStringToInteger(c.DefaultQuery("page", "1"))
	ResultShow := convertStringToInteger(c.DefaultQuery("show", "100"))
	limit := convertStringToInteger(c.DefaultQuery("limit", "0"))
	offset := convertStringToInteger(c.DefaultQuery("offset", "0"))
	sortRaw := strings.TrimSpace(c.DefaultQuery("sort", "-start_date"))
	_ = c.Query("include")

	// 从查询参数读取开始和结束时间。
	parsedStartDate, err := parseDateParam(c.Query("startDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError2, err.Error())
		return
	}
	parsedEndDate, err := parseDateParam(c.Query("endDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError2, err.Error())
		return
	}
	// 从查询参数读取可选的最小和最大距离过滤条件。
	minDistanceParam := c.Query("minDistance")
	minDistance := 0.0
	if minDistanceParam != "" {
		minDistance = convertStringToFloat(minDistanceParam)
		if minDistance < 0 {
			minDistance = 0
		}
	}
	maxDistanceParam := c.Query("maxDistance")
	maxDistance := 0.0
	if maxDistanceParam != "" {
		maxDistance = convertStringToFloat(maxDistanceParam)
		if maxDistance < 0 {
			maxDistance = 0
		}
	}

	var (
		CarName                       NullString
		DrivesData                    []DriveListItemV1
		UnitsLength, UnitsTemperature string
	)

	if limit > 0 {
		ResultShow = limit
		ResultPage = 0
		if offset > 0 {
			ResultPage = offset
		}
	} else {
		// 基于页码计算偏移量；页码最小为 1。
		if ResultPage > 0 {
			ResultPage--
		} else {
			ResultPage = 0
		}
		ResultPage = (ResultPage * ResultShow)
	}

	orderBy := "start_date DESC"
	switch sortRaw {
	case "start_date":
		orderBy = "start_date ASC"
	case "-start_date":
		orderBy = "start_date DESC"
	case "distance":
		orderBy = "distance ASC"
	case "-distance":
		orderBy = "distance DESC"
	case "duration":
		orderBy = "duration_min ASC"
	case "-duration":
		orderBy = "duration_min DESC"
	case "efficiency":
		orderBy = "consumption_net ASC"
	case "-efficiency":
		orderBy = "consumption_net DESC"
	}

	// 从数据库读取行程数据。
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
		WHERE drives.car_id=$1 AND end_date IS NOT NULL`

	// 查询参数列表。
	var queryParams []any
	queryParams = append(queryParams, CarID)
	paramIndex := 2

	// 按需追加日期过滤。
	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND drives.start_date >= $%d", paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND drives.end_date <= $%d", paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}

	// 按需追加最小和最大距离过滤。
	if minDistance > 0 || maxDistance > 0 {
		var unitsLength string
		err = db.QueryRow("SELECT unit_of_length FROM settings LIMIT 1").Scan(&unitsLength)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(
				c,
				"TeslaMateAPICarsDrivesV1",
				CarsDrivesError1,
				fmt.Sprintf("unable to retrieve unit_of_length from settings table: %v", err),
			)
			return
		}
		if unitsLength == "mi" {
			if minDistance > 0 {
				minDistance = milesToKilometers(minDistance)
			}
			if maxDistance > 0 {
				maxDistance = milesToKilometers(maxDistance)
			}
		}

		if minDistance > 0 {
			query += fmt.Sprintf(" AND distance >= $%d", paramIndex)
			queryParams = append(queryParams, minDistance)
			paramIndex++
		}
		if maxDistance > 0 {
			query += fmt.Sprintf(" AND distance <= $%d", paramIndex)
			queryParams = append(queryParams, maxDistance)
			paramIndex++
		}
	}

	query += fmt.Sprintf(`
        ORDER BY %s
        LIMIT $%d OFFSET $%d;`, orderBy, paramIndex, paramIndex+1)

	queryParams = append(queryParams, ResultShow, ResultPage)

	rows, err := db.Query(query, queryParams...)

	// 检查查询错误。
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError1, err.Error())
		return
	}

	// 延迟关闭结果集。
	defer rows.Close()

	// 遍历查询结果。
	for rows.Next() {

		// 创建行程记录对象。
		drive := DriveListItemV1{}

		// 将当前行扫描到行程记录。
		err = rows.Scan(
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

		// 检查扫描错误。
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError1, err.Error())
			return
		}

		// 追加行程记录到响应列表。
		DrivesData = append(DrivesData, drive)
	}

	// 检查结果集遍历错误。
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError1, err.Error())
		return
	}

	jsonData := DrivesListV1Envelope{
		Data: DrivesListV1Data{
			Car: CarRefV1{
				CarID:   CarID,
				CarName: CarName,
			},
			Drives: DrivesData,
			TeslaMateUnits: UnitsLengthTempV1{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// 返回响应数据。
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsDrivesV1", jsonData)
}
