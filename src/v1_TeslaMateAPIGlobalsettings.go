package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/tobiasehlert/teslamateapi/src/dto"
)

// TeslaMateAPIGlobalsettingsV1 returns the single TeslaMate settings row.
//
// @Summary      Global settings
// @Description  Returns the TeslaMate single-row settings (units, GUI, URLs).
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dto.V1GlobalSettingsResponse
// @Failure      401  {object}  dto.ErrorEnvelope
// @Router       /api/v1/globalsettings [get]
func TeslaMateAPIGlobalsettingsV1(c *gin.Context) {

	// define error messages
	var CarsGlobalsettingsError1 = "Unable to load settings."

	// creating required vars
	var globalSetting dto.V1GlobalSettings

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
	row := db.QueryRowContext(c.Request.Context(), query)

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

	//
	// build the data-blob
	jsonData := dto.V1GlobalSettingsResponse{
		Data: dto.V1GlobalSettingsData{
			GlobalSettings: globalSetting,
		},
	}

	// return jsonData
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPIGlobalsettingsV1", jsonData)
}
