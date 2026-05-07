package main

import (
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsUpdatesV1 返回兼容响应结构的车辆软件更新历史。
// @Summary 软件更新列表
// @Tags 兼容 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Param page query int false "页码"
// @Param show query int false "每页数量"
// @Success 200 {object} UpdatesListV1Envelope
// @Router /v1/cars/{CarID}/updates [get]
func TeslaMateAPICarsUpdatesV1(c *gin.Context) {

	// 定义错误消息。
	var CarsUpdatesError1 = "Unable to load updates."

	// 从 URL 读取车辆 ID。
	CarID := convertStringToInteger(c.Param("CarID"))
	// 读取分页查询参数。
	ResultPage := convertStringToInteger(c.DefaultQuery("page", "1"))
	ResultShow := convertStringToInteger(c.DefaultQuery("show", "100"))

	var (
		UpdatesData []UpdatesListItemV1
		CarData     CarRefV1
	)

	// 基于页码计算偏移量；页码最小为 1。
	if ResultPage > 0 {
		ResultPage--
	} else {
		ResultPage = 0
	}
	ResultPage = (ResultPage * ResultShow)

	// 从数据库读取软件更新数据。
	query := `
		SELECT
			updates.id,
			cars.name,
			start_date,
			end_date,
			version
		FROM updates
		LEFT JOIN cars ON car_id = cars.id
		WHERE car_id = $1 AND end_date IS NOT NULL AND version IS NOT NULL
		ORDER BY start_date DESC
		LIMIT $2 OFFSET $3;`
	rows, err := db.Query(query, CarID, ResultShow, ResultPage)

	// 检查查询错误。
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
		return

	}

	// 延迟关闭结果集。
	defer rows.Close()

	// 遍历查询结果。
	for rows.Next() {

		// 创建软件更新记录对象。
		update := UpdatesListItemV1{}

		// 将当前行扫描到软件更新记录。
		err = rows.Scan(
			&update.UpdateID,
			&CarData.CarName,
			&update.StartDate,
			&update.EndDate,
			&update.Version,
		)

		// 检查扫描错误。
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
			return
		}

		// 按用户配置时区转换时间字段。
		update.StartDate = getTimeInTimeZone(update.StartDate)
		update.EndDate = getTimeInTimeZone(update.EndDate)

		// 追加软件更新记录到响应列表。
		UpdatesData = append(UpdatesData, update)
		CarData.CarID = CarID
	}

	// 检查结果集遍历错误。
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
		return
	}

	jsonData := UpdatesListV1Envelope{
		Data: UpdatesListV1Data{
			Car:     CarData,
			Updates: UpdatesData,
		},
	}

	// 返回响应数据。
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsUpdatesV1", jsonData)
}
