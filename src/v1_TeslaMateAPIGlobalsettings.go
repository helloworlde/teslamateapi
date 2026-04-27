package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type GlobalsettingsV1AccountInfo struct {
	InsertedAt string `json:"inserted_at"`
	UpdatedAt  string `json:"updated_at"`
}

type GlobalsettingsV1Units struct {
	UnitsLength      string `json:"unit_of_length"`
	UnitsTemperature string `json:"unit_of_temperature"`
}

type GlobalsettingsV1GUI struct {
	PreferredRange string `json:"preferred_range"`
	Language       string `json:"language"`
}

type GlobalsettingsV1URLs struct {
	BaseURL    string `json:"base_url"`
	GrafanaURL string `json:"grafana_url"`
}

type GlobalsettingsV1Settings struct {
	SettingID      int                         `json:"setting_id"`
	AccountInfo    GlobalsettingsV1AccountInfo `json:"account_info"`
	TeslaMateUnits GlobalsettingsV1Units       `json:"teslamate_units"`
	TeslaMateGUI   GlobalsettingsV1GUI         `json:"teslamate_webgui"`
	TeslaMateURLs  GlobalsettingsV1URLs        `json:"teslamate_urls"`
}

type GlobalsettingsV1Data struct {
	Settings GlobalsettingsV1Settings `json:"settings"`
}

type GlobalsettingsV1Envelope struct {
	Data GlobalsettingsV1Data `json:"data"`
}

// TeslaMateAPIGlobalsettingsV1 func
func TeslaMateAPIGlobalsettingsV1(c *gin.Context) {

	// define error messages
	var CarsGlobalsettingsError1 = "Unable to load settings."

	// creating required vars
	var globalSetting GlobalsettingsV1Settings

	// getting data from database
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

	// scanning row and putting values into the globalSetting
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
		// nothing wrong.. continuing
		break
	default:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPIGlobalsettingsV1", CarsGlobalsettingsError1, err.Error())
		return
	}

	// adjusting to timezone differences from UTC to be userspecific
	globalSetting.AccountInfo.InsertedAt = getTimeInTimeZone(globalSetting.AccountInfo.InsertedAt)
	globalSetting.AccountInfo.UpdatedAt = getTimeInTimeZone(globalSetting.AccountInfo.UpdatedAt)

	jsonData := GlobalsettingsV1Envelope{
		Data: GlobalsettingsV1Data{
			Settings: globalSetting,
		},
	}

	// return jsonData
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPIGlobalsettingsV1", jsonData)
}
