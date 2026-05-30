package v1

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
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
func (h *Handler) Globalsettings(c *gin.Context) {

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
	row := h.db.QueryRowContext(c.Request.Context(), query)

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
		respond.HandleError(c, "TeslaMateAPIGlobalsettingsV1", "No rows were returned!", err.Error())
		return
	case nil:
		// nothing wrong.. continuing
		break
	default:
		respond.HandleError(c, "TeslaMateAPIGlobalsettingsV1", CarsGlobalsettingsError1, err.Error())
		return
	}

	// adjusting to timezone differences from UTC to be userspecific
	globalSetting.AccountInfo.InsertedAt = h.timeInTZ(globalSetting.AccountInfo.InsertedAt)
	globalSetting.AccountInfo.UpdatedAt = h.timeInTZ(globalSetting.AccountInfo.UpdatedAt)

	//
	// build the data-blob
	jsonData := dto.V1GlobalSettingsResponse{
		Data: dto.V1GlobalSettingsData{
			GlobalSettings: globalSetting,
		},
	}

	// return jsonData
	respond.HandleSuccess(c, "TeslaMateAPIGlobalsettingsV1", jsonData)
}
