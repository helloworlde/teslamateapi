package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// GlobalsettingsV1AccountInfo 描述 Tesla 账号相关时间信息。
type GlobalsettingsV1AccountInfo struct {
	InsertedAt string `json:"inserted_at"`
	UpdatedAt  string `json:"updated_at"`
}

// GlobalsettingsV1Units 描述 TeslaMate 单位偏好。
type GlobalsettingsV1Units struct {
	UnitsLength      string `json:"unit_of_length"`
	UnitsTemperature string `json:"unit_of_temperature"`
}

// GlobalsettingsV1GUI 描述 TeslaMate 界面设置。
type GlobalsettingsV1GUI struct {
	PreferredRange string `json:"preferred_range"`
	Language       string `json:"language"`
}

// GlobalsettingsV1URLs 描述 TeslaMate 外部访问地址配置。
type GlobalsettingsV1URLs struct {
	BaseURL    string `json:"base_url"`
	GrafanaURL string `json:"grafana_url"`
}

// GlobalsettingsV1Settings 描述兼容全局设置对象。
type GlobalsettingsV1Settings struct {
	SettingID      int                         `json:"setting_id"`
	AccountInfo    GlobalsettingsV1AccountInfo `json:"account_info"`
	TeslaMateUnits GlobalsettingsV1Units       `json:"teslamate_units"`
	TeslaMateGUI   GlobalsettingsV1GUI         `json:"teslamate_webgui"`
	TeslaMateURLs  GlobalsettingsV1URLs        `json:"teslamate_urls"`
}

// GlobalsettingsV1Data 描述兼容全局设置响应的 data 对象。
type GlobalsettingsV1Data struct {
	Settings GlobalsettingsV1Settings `json:"settings"`
}

// GlobalsettingsV1Envelope 描述兼容全局设置接口响应。
type GlobalsettingsV1Envelope struct {
	Data GlobalsettingsV1Data `json:"data"`
}

// TeslaMateAPIGlobalsettingsV1 返回兼容 TeslaMate 全局设置。
// @Summary 全局设置
// @Tags 兼容 API
// @Produce json
// @Success 200 {object} GlobalsettingsV1Envelope
// @Router /v1/globalsettings [get]
func TeslaMateAPIGlobalsettingsV1(c *gin.Context) {

	// 定义错误消息。
	var CarsGlobalsettingsError1 = "Unable to load settings."

	// 创建响应对象。
	var globalSetting GlobalsettingsV1Settings

	// 从数据库读取全局设置。
	query := `
		SELECT
			id,
			inserted_at,
			updated_at,
			unit_of_length,
			unit_of_temperature,
			preferred_range,
			language,
			base_url,
			grafana_url
		FROM settings
		LIMIT 1;`
	row := db.QueryRow(query)

	// 将查询结果扫描到全局设置对象。
	err := row.Scan(
		&globalSetting.SettingID,
		&globalSetting.AccountInfo.InsertedAt,
		&globalSetting.AccountInfo.UpdatedAt,
		&globalSetting.TeslaMateUnits.UnitsLength,
		&globalSetting.TeslaMateUnits.UnitsTemperature,
		&globalSetting.TeslaMateGUI.PreferredRange,
		&globalSetting.TeslaMateGUI.Language,
		&globalSetting.TeslaMateURLs.BaseURL,
		&globalSetting.TeslaMateURLs.GrafanaURL,
	)

	switch err {
	case sql.ErrNoRows:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPIGlobalsettingsV1", "No rows were returned!", err.Error())
		return
	case nil:
		// 查询成功，继续组装响应。
		break
	default:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPIGlobalsettingsV1", CarsGlobalsettingsError1, err.Error())
		return
	}

	// 按用户配置时区转换时间字段。
	globalSetting.AccountInfo.InsertedAt = getTimeInTimeZone(globalSetting.AccountInfo.InsertedAt)
	globalSetting.AccountInfo.UpdatedAt = getTimeInTimeZone(globalSetting.AccountInfo.UpdatedAt)

	jsonData := GlobalsettingsV1Envelope{
		Data: GlobalsettingsV1Data{
			Settings: globalSetting,
		},
	}

	// 返回响应数据。
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPIGlobalsettingsV1", jsonData)
}
