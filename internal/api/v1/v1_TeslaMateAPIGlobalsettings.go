package v1

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
)

// TeslaMateAPIGlobalsettingsV1 godoc
//
// @Summary TeslaMate 全局设置
// @Tags v1
// @Produce json
// @Success 200 {object} V1JSONEnvelope
// @Failure 200 {object} V1ErrorEnvelope
// @Router /v1/globalsettings [get]
func TeslaMateAPIGlobalsettingsV1(c *gin.Context) {

	// define error messages
	var CarsGlobalsettingsError1 = "Unable to load settings."

	// creating structs for /globalsettings
	// AccountInfo struct - child of GlobalSettings
	type AccountInfo struct {
		InsertedAt string `json:"inserted_at"` // string
		UpdatedAt  string `json:"updated_at"`  // string
	}
	// TeslaMateUnits struct - child of GlobalSettings
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`             // string
		UnitsTemperature string `json:"unit_of_temperature"`        // string
		UnitsPressure    string `json:"unit_of_pressure,omitempty"` // string (added)
	}
	// TeslaMateGUI struct - child of GlobalSettings
	type TeslaMateGUI struct {
		PreferredRange string `json:"preferred_range"`      // string
		Language       string `json:"language"`             // string
		ThemeMode      string `json:"theme_mode,omitempty"` // string (added)
	}
	// TeslaMateURLs struct - child of GlobalSettings
	type TeslaMateURLs struct {
		BaseURL    string `json:"base_url"`    // string
		GrafanaURL string `json:"grafana_url"` // string
	}
	// GlobalSettings struct - child of Data
	type GlobalSettings struct {
		SettingID      int            `json:"setting_id"`       // smallint
		AccountInfo    AccountInfo    `json:"account_info"`     // struct
		TeslaMateUnits TeslaMateUnits `json:"teslamate_units"`  // struct
		TeslaMateGUI   TeslaMateGUI   `json:"teslamate_webgui"` // struct
		TeslaMateURLs  TeslaMateURLs  `json:"teslamate_urls"`   // struct
	}
	// Data struct - child of JSONData
	type Data struct {
		GlobalSettings GlobalSettings `json:"settings"`
	}
	// JSONData struct - main
	type JSONData struct {
		Data Data `json:"data"` // 响应数据
	}

	// creating required vars
	var globalSetting GlobalSettings

	// getting data from database
	query := `
		SELECT
			id,
			inserted_at,
			updated_at,
			unit_of_length,
			unit_of_temperature,
			COALESCE(unit_of_pressure, '') as unit_of_pressure,
			preferred_range,
			language,
			COALESCE(theme_mode, '') as theme_mode,
			base_url,
			grafana_url
		FROM settings
		LIMIT 1;`
	row := apicommon.DB.QueryRow(query)

	// scanning row and putting values into the globalSetting
	err := row.Scan(
		&globalSetting.SettingID,
		&globalSetting.AccountInfo.InsertedAt,
		&globalSetting.AccountInfo.UpdatedAt,
		&globalSetting.TeslaMateUnits.UnitsLength,
		&globalSetting.TeslaMateUnits.UnitsTemperature,
		&globalSetting.TeslaMateUnits.UnitsPressure,
		&globalSetting.TeslaMateGUI.PreferredRange,
		&globalSetting.TeslaMateGUI.Language,
		&globalSetting.TeslaMateGUI.ThemeMode,
		&globalSetting.TeslaMateURLs.BaseURL,
		&globalSetting.TeslaMateURLs.GrafanaURL,
	)

	switch err {
	case sql.ErrNoRows:
		apicommon.HandleErrorResponse(c, "TeslaMateAPIGlobalsettingsV1", "No rows were returned!", err.Error())
		return
	case nil:
		// nothing wrong.. continuing
		break
	default:
		apicommon.HandleErrorResponse(c, "TeslaMateAPIGlobalsettingsV1", CarsGlobalsettingsError1, err.Error())
		return
	}

	// adjusting to timezone differences from UTC to be userspecific
	globalSetting.AccountInfo.InsertedAt = apicommon.GetTimeInTimeZone(globalSetting.AccountInfo.InsertedAt)
	globalSetting.AccountInfo.UpdatedAt = apicommon.GetTimeInTimeZone(globalSetting.AccountInfo.UpdatedAt)

	//
	// build the data-blob
	jsonData := JSONData{
		Data{
			GlobalSettings: globalSetting,
		},
	}

	// return jsonData
	apicommon.HandleSuccessResponse(c, "TeslaMateAPIGlobalsettingsV1", jsonData)
}
