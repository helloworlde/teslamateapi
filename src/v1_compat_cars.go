package main

import (
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsListV1 返回兼容 v1 响应结构的全部车辆。
// @Summary 车辆列表
// @Description 兼容接口：使用历史响应封装返回全部车辆。
// @Tags 兼容 API
// @Produce json
// @Success 200 {object} CarsV1Envelope
// @Router /v1/cars [get]
func TeslaMateAPICarsListV1(c *gin.Context) {
	handleCarsV1(c)
}

// TeslaMateAPICarByIDV1 返回兼容 v1 响应结构的单辆车。
// @Summary 车辆详情
// @Description 兼容接口：在 data.cars 中返回匹配车辆。
// @Tags 兼容 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} CarsV1Envelope
// @Router /v1/cars/{CarID} [get]
func TeslaMateAPICarByIDV1(c *gin.Context) {
	handleCarsV1(c)
}

func handleCarsV1(c *gin.Context) {

	// 定义错误消息。
	var CarsError1 = "Unable to load cars."

	// 从 URL 读取车辆 ID。
	ParamCarID := c.Param("CarID")
	var CarID int
	if ParamCarID != "" {
		CarID = convertStringToInteger(ParamCarID)
	}

	// 创建响应数据容器。
	var CarsData []CarsV1Car

	// 从数据库读取车辆数据。
	query := `
		SELECT
			cars.id,
			eid,
			vid,
			model,
			efficiency,
			inserted_at,
			updated_at,
			vin,
			name,
			trim_badging,
			exterior_color,
			spoiler_type,
			wheel_type,
			suspend_min,
			suspend_after_idle_min,
			req_not_unlocked,
			free_supercharging,
			use_streaming_api,
			(SELECT COUNT(*) FROM charging_processes WHERE car_id=cars.id) as total_charges,
			(SELECT COUNT(*) FROM drives WHERE car_id=cars.id) as total_drives,
			(SELECT COUNT(*) FROM updates WHERE car_id=cars.id) as total_charges
		FROM cars
		LEFT JOIN car_settings ON cars.id = car_settings.id
		ORDER BY id;`
	rows, err := db.Query(query)

	// 检查查询错误。
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsV1", CarsError1, err.Error())
		return
	}

	// 延迟关闭结果集。
	defer rows.Close()

	// 遍历查询结果。
	for rows.Next() {

		car := CarsV1Car{}

		// 将当前行扫描到车辆对象。
		err = rows.Scan(
			&car.CarID,
			&car.CarDetails.EID,
			&car.CarDetails.VID,
			&car.CarDetails.Model,
			&car.CarDetails.Efficiency,
			&car.TeslaMateDetails.InsertedAt,
			&car.TeslaMateDetails.UpdatedAt,
			&car.CarDetails.Vin,
			&car.Name,
			&car.CarDetails.TrimBadging,
			&car.CarExterior.ExteriorColor,
			&car.CarExterior.SpoilerType,
			&car.CarExterior.WheelType,
			&car.CarSettings.SuspendMin,
			&car.CarSettings.SuspendAfterIdleMin,
			&car.CarSettings.ReqNotUnlocked,
			&car.CarSettings.FreeSupercharging,
			&car.CarSettings.UseStreamingAPI,
			&car.TeslaMateStats.TotalCharges,
			&car.TeslaMateStats.TotalDrives,
			&car.TeslaMateStats.TotalUpdates,
		)

		// 检查扫描错误。
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsV1", CarsError1, err.Error())
			return
		}

		// 未指定车辆 ID 时返回全部车辆；指定时只返回匹配车辆。
		if CarID == 0 && len(ParamCarID) == 0 || CarID != 0 && CarID == car.CarID {

			// 按用户配置时区转换时间字段。
			car.TeslaMateDetails.InsertedAt = getTimeInTimeZone(car.TeslaMateDetails.InsertedAt)
			car.TeslaMateDetails.UpdatedAt = getTimeInTimeZone(car.TeslaMateDetails.UpdatedAt)

			CarsData = append(CarsData, car)
		}
	}

	// 检查结果集遍历错误。
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsV1", CarsError1, err.Error())
		return
	}

	jsonData := CarsV1Envelope{
		Data: CarsV1Data{Cars: CarsData},
	}

	// 返回响应数据。
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsV1", jsonData)

}
