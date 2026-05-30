package v1

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/command"
	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// teslaCommandHTTPTimeout caps the outbound request to Tesla's owner-api so
// a hung peer can't pin a goroutine indefinitely.
const teslaCommandHTTPTimeout = 30 * time.Second

// TeslaMateAPICarsCommandListV1 returns the allow-listed Tesla command names.
//
// @Summary      List Tesla commands
// @Description  Returns the allow-list. Requires ENABLE_COMMANDS=true.
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path      int  true  "TeslaMate cars.id"
// @Success      200    {object}  dto.V1CommandList
// @Failure      401    {object}  dto.ErrorEnvelope
// @Failure      403    {object}  dto.ErrorEnvelope  "ENABLE_COMMANDS=false"
// @Router       /api/v1/cars/{CarID}/command  [get]
// @Router       /api/v1/cars/{CarID}/commands [get]
func (h *Handler) CommandList(c *gin.Context) { h.Command(c) }

// TeslaMateAPICarsCommandExecV1 proxies a command call to Tesla owner-api.
//
// @Summary      Execute Tesla command
// @Description  Proxies to Tesla owner-api with the car's stored token. Status code and body mirror Tesla's. Requires ENABLE_COMMANDS=true.
// @Tags         v1
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        CarID    path      int     true  "TeslaMate cars.id"
// @Param        Command  path      string  true  "Command name"
// @Success      200      {object}  dto.V1CommandResult  "Tesla passthrough JSON"
// @Failure      400      {object}  dto.ErrorEnvelope
// @Failure      401      {object}  dto.ErrorEnvelope
// @Failure      403      {object}  dto.ErrorEnvelope    "ENABLE_COMMANDS=false"
// @Failure      500      {object}  dto.ErrorEnvelope
// @Router       /api/v1/cars/{CarID}/command/{Command} [post]
// @Router       /api/v1/cars/{CarID}/wake_up           [post]
func (h *Handler) CommandExec(c *gin.Context) { h.Command(c) }

// TeslaMateAPICarsCommandV1 lists or dispatches Tesla owner-api commands.
// Routed via the per-method wrappers above so swag can document the GET
// list-response and the POST passthrough-response separately.
func (h *Handler) Command(c *gin.Context) {

	// creating required vars
	var (
		CarsCommandsError1                                 = "Unable to load cars."
		TeslaAccessToken, TeslaVehicleID, TeslaEndpointUrl string
		jsonData                                           map[string]any
		err                                                error
	)

	// check if commands are enabled.. if not we need to abort
	if !h.cfg.CommandsEnabled {
		log.Println("[warning] TeslaMateAPICarsCommandV1 ENABLE_COMMANDS is not true.. returning 403 forbidden.")
		respond.HandleOther(c, http.StatusForbidden, "TeslaMateAPICarsCommandV1", gin.H{"error": "You are not allowed to access commands"})
		return
	}

	// if request method is GET return list of commands
	if c.Request.Method == http.MethodGet {
		respond.HandleSuccess(c, "TeslaMateAPICarsCommandV1", dto.V1CommandList{EnabledCommands: h.allowListItems()})
		return
	}

	// authentication for the endpoint
	validToken, errorMessage := h.validateAuthToken(c)
	if !validToken {
		respond.HandleOther(c, http.StatusUnauthorized, "TeslaMateAPICarsCommandV1", gin.H{"error": errorMessage})
		return
	}

	// getting CarID param from URL and validating that it's not zero
	CarID := convert.StrToInt(c.Param("CarID"))
	if CarID == 0 {
		log.Println("[error] TeslaMateAPICarsCommandV1 CarID is invalid (zero)!")
		respond.HandleOther(c, http.StatusBadRequest, "TeslaMateAPICarsCommandV1", gin.H{"error": "CarID invalid"})
		return
	}

	// getting request body to pass to Tesla
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 error in first io.ReadAll", err)
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", gin.H{"error": "internal io reading error"})
		return
	}

	// getting :Command
	cmdPath := ("/command/" + c.Param("Command"))
	// if command is /command/ or /command/wake_up, set to /wake_up only
	if cmdPath == "/command/" || cmdPath == "/command/wake_up" {
		cmdPath = "/wake_up"
	}

	if !h.allowListContains(cmdPath) {
		log.Println("[warning] TeslaMateAPICarsCommandV1 command not allowed!")
		respond.HandleOther(c, http.StatusUnauthorized, "TeslaMateAPICarsCommandV1", gin.H{"error": "unauthorized"})
		return
	}

	// get TeslaVehicleID and TeslaAccessToken
	query := `
		SELECT
			eid as TeslaVehicleID,
			(SELECT access FROM private.tokens LIMIT 1) as TeslaAccessToken
		FROM cars
		WHERE id = $1
		LIMIT 1;`
	row := h.db.QueryRowContext(c.Request.Context(), query, CarID)

	err = row.Scan(
		&TeslaVehicleID,
		&TeslaAccessToken,
	)

	switch err {
	case sql.ErrNoRows:
		respond.HandleError(c, "TeslaMateAPICarsCommandV1", "No rows were returned!", err.Error())
		return
	case nil:
		// nothing wrong.. continuing
		break
	default:
		respond.HandleError(c, "TeslaMateAPICarsCommandV1", CarsCommandsError1, err.Error())
		return
	}

	// load ENCRYPTION_KEY environment variable
	teslaMateEncryptionKey := h.cfg.EncryptionKey
	if teslaMateEncryptionKey == "" {
		log.Println("[error] TeslaMateAPICarsCommandV1 can't get ENCRYPTION_KEY.. will fail to perform command.")
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", gin.H{"error": "missing ENCRYPTION_KEY env variable"})
		return
	}

	// decrypt access token
	TeslaAccessToken, err = command.DecryptAccessToken(TeslaAccessToken, teslaMateEncryptionKey)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 token decrypt failed:", err)
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", gin.H{"error": "unable to decrypt access token"})
		return
	}

	// Region default applies only when TESLA_API_HOST is unset; explicit
	// override wins for both China and Global so deployments can pin to a
	// proxy or test endpoint regardless of region detection.
	switch command.GetCarRegionAPI(TeslaAccessToken) {
	case command.ChinaAPI:
		if h.cfg.TeslaAPIHost != "" {
			TeslaEndpointUrl = h.cfg.TeslaAPIHost
		} else {
			TeslaEndpointUrl = "https://owner-api.vn.cloud.tesla.cn"
		}
	default:
		if h.cfg.TeslaAPIHost != "" {
			TeslaEndpointUrl = h.cfg.TeslaAPIHost
		} else {
			TeslaEndpointUrl = "https://owner-api.teslamotors.com"
		}
	}

	client := &http.Client{Timeout: teslaCommandHTTPTimeout}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, TeslaEndpointUrl+"/api/1/vehicles/"+TeslaVehicleID+cmdPath, strings.NewReader(string(reqBody)))
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 http.NewRequestWithContext:", err)
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", gin.H{"error": "internal http request error"})
		return
	}
	req.Header.Set("Authorization", "Bearer "+TeslaAccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TeslaMateApi/"+h.cfg.APIVersion+" (+https://github.com/tobiasehlert/teslamateapi)")
	resp, err := client.Do(req)

	// check response error
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 error in http request to "+TeslaEndpointUrl, err)
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", gin.H{"error": "internal http request error"})
		return
	}

	defer resp.Body.Close()
	defer client.CloseIdleConnections()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 error in second io.ReadAll:", err)
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", gin.H{"error": "internal io reading error"})
		return
	}
	// If Tesla returns non-JSON (HTML error page, empty body), pass the raw
	// payload through instead of silently coercing to `null`.
	if jsonErr := json.Unmarshal(respBody, &jsonData); jsonErr != nil {
		log.Println("[warning] TeslaMateAPICarsCommandV1 non-JSON response from Tesla:", jsonErr)
		respond.HandleOther(c, resp.StatusCode, "TeslaMateAPICarsCommandV1", dto.V1CommandRawResponse{Raw: string(respBody)})
		return
	}

	// return jsonData
	// use respond.HandleOther since we use the statusCode from Tesla API
	respond.HandleOther(c, resp.StatusCode, "TeslaMateAPICarsCommandV1", jsonData)

}
