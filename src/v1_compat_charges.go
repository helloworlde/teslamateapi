package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsChargesV1 返回兼容响应结构的车辆充电会话列表。
// @Summary 充电列表
// @Description 兼容接口：保持历史充电列表响应封装。
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
// @Param include query string false "逗号分隔的可选分区"
// @Success 200 {object} ChargesListV1Envelope
// @Router /v1/cars/{CarID}/charges [get]
func TeslaMateAPICarsChargesV1(c *gin.Context) {

	// 定义错误消息。
	var CarsChargesError1 = "Unable to load charges."
	var CarsChargesError2 = "Invalid date format."

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
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError2, err.Error())
		return
	}
	parsedEndDate, err := parseDateParam(c.Query("endDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError2, err.Error())
		return
	}

	var (
		CarName                       NullString
		ChargesData                   []ChargeListItemV1
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
	case "duration":
		orderBy = "duration_min ASC"
	case "-duration":
		orderBy = "duration_min DESC"
	case "cost":
		orderBy = "cost ASC"
	case "-cost":
		orderBy = "cost DESC"
	case "energy":
		orderBy = "charge_energy_added ASC"
	case "-energy":
		orderBy = "charge_energy_added DESC"
	}

	// 从数据库读取充电数据。
	query := `
		SELECT
			charging_processes.id AS charge_id,
			charging_processes.start_date,
			charging_processes.end_date,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, nullif(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS address,
			COALESCE(charge_energy_added, 0) AS charge_energy_added,
			COALESCE(GREATEST(charge_energy_used, charge_energy_added), 0) AS charge_energy_used,
			COALESCE(cost, 0) AS cost,
			start_ideal_range_km AS start_ideal_range,
			end_ideal_range_km AS end_ideal_range,
			start_rated_range_km AS start_rated_range,
			end_rated_range_km AS end_rated_range,
			start_battery_level,
			end_battery_level,
			duration_min,
			TO_CHAR((duration_min * INTERVAL '1 minute'), 'HH24:MI') AS duration_str,
			outside_temp_avg,
			position.odometer AS odometer,
			position.latitude,
			position.longitude,
			(SELECT unit_of_length FROM settings LIMIT 1) AS unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) AS unit_of_temperature,
			cars.name,
			charges.conn_charge_cable,
			charges.fast_charger_brand
		FROM charging_processes
		LEFT JOIN cars ON car_id = cars.id
		LEFT JOIN addresses address ON address_id = address.id
		LEFT JOIN positions position ON position_id = position.id
		LEFT JOIN geofences geofence ON geofence_id = geofence.id
		LEFT JOIN LATERAL (
			SELECT conn_charge_cable, fast_charger_brand
			FROM charges
			WHERE charging_process_id = charging_processes.id
			LIMIT 1
		) charges ON true
		WHERE charging_processes.car_id = $1
		  AND charging_processes.end_date IS NOT NULL`

	// 查询参数列表。
	var queryParams []any
	queryParams = append(queryParams, CarID)
	paramIndex := 2

	// 按需追加日期过滤。
	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND charging_processes.start_date >= $%d", paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND charging_processes.end_date <= $%d", paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}

	query += fmt.Sprintf(`
        ORDER BY %s
        LIMIT $%d OFFSET $%d;`, orderBy, paramIndex, paramIndex+1)

	queryParams = append(queryParams, ResultShow, ResultPage)

	rows, err := db.Query(query, queryParams...)

	// 检查查询错误。
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
		return
	}

	// 延迟关闭结果集。
	defer rows.Close()

	// 遍历查询结果。
	for rows.Next() {

		// 创建充电记录对象。
		charge := ChargeListItemV1{}
		var connCable, fastBrand sql.NullString

		// 将当前行扫描到充电记录。
		err = rows.Scan(
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
			&connCable,
			&fastBrand,
		)

		if connCable.Valid && connCable.String != "" {
			s := connCable.String
			charge.ConnChargeCable = &s
		}
		if fastBrand.Valid && fastBrand.String != "" && fastBrand.String != "<invalid>" {
			s := fastBrand.String
			charge.FastChargerBrand = &s
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

		// 检查扫描错误。
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
			return
		}

		// 追加充电记录到响应列表。
		ChargesData = append(ChargesData, charge)
	}

	// 检查结果集遍历错误。
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
		return
	}

	jsonData := ChargesListV1Envelope{
		Data: ChargesListV1Data{
			Car: CarRefV1{
				CarID:   CarID,
				CarName: CarName,
			},
			Charges: ChargesData,
			TeslaMateUnits: UnitsLengthTempV1{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// 返回响应数据。
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsChargesV1", jsonData)
}
