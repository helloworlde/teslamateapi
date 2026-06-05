package v1

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/audit"
	"github.com/tobiasehlert/teslamateapi/internal/command"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// teslaCommandHTTPTimeout caps the outbound request to Tesla's owner-api so
// a hung peer can't pin a goroutine indefinitely.
const teslaCommandHTTPTimeout = 30 * time.Second

var teslaCommandHTTPClient = &http.Client{Timeout: teslaCommandHTTPTimeout}

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
//
// Every privileged invocation emits one [audit] log line — see
// internal/audit. GET (list) is not privileged and is not audited.
func (h *Handler) Command(c *gin.Context) {

	const handler = "TeslaMateAPICarsCommandV1"
	var (
		CarsCommandsError1                                 = "Unable to load cars."
		TeslaAccessToken, TeslaVehicleID, TeslaEndpointUrl string
		jsonData                                           map[string]any
		err                                                error
	)

	startedAt := time.Now()
	ev := audit.Event{
		Action:    "command_exec",
		Method:    c.Request.Method,
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
	emit := func() {
		ev.DurationMS = time.Since(startedAt).Milliseconds()
		audit.Log(ev)
	}

	// check if commands are enabled.. if not we need to abort
	if !h.cfg.CommandsEnabled {
		log.Println("[warning] TeslaMateAPICarsCommandV1 ENABLE_COMMANDS is not true.. returning 403 forbidden.")
		ev.Outcome = audit.OutcomeDenied
		ev.Reason = audit.ReasonCommandsDisabled
		emit()
		respond.HandleOther(c, http.StatusForbidden, handler, gin.H{"error": "You are not allowed to access commands"})
		return
	}

	// if request method is GET return list of commands
	if c.Request.Method == http.MethodGet {
		// Listing the allow-list is not a privileged action; skip audit to
		// avoid drowning the log when clients poll the catalog.
		respond.HandleSuccess(c, handler, dto.V1CommandList{EnabledCommands: h.allowListItems()})
		return
	}

	// authentication for the endpoint
	validToken, errorMessage := h.validateAuthToken(c)
	if !validToken {
		ev.Outcome = audit.OutcomeDenied
		ev.Reason = audit.ReasonUnauthorized
		ev.ErrDetail = errorMessage
		emit()
		respond.HandleOther(c, http.StatusUnauthorized, handler, gin.H{"error": errorMessage})
		return
	}

	// getting CarID param from URL and validating that it's not zero
	CarID, ok := requirePositiveIntParamStatus(c, handler, "CarID", c.Param("CarID"))
	if !ok {
		ev.Outcome = audit.OutcomeDenied
		ev.Reason = audit.ReasonInvalidCarID
		emit()
		return
	}
	ev.CarID = CarID

	// getting request body to pass to Tesla
	reqBody, err := readAllLimited(c.Request.Body, maxProxyRequestBodyBytes)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 error in first io.ReadAll", err)
		ev.Outcome = audit.OutcomeError
		ev.Reason = audit.ReasonReadBodyFail
		ev.ErrDetail = err.Error()
		emit()
		if bodyTooLarge(err) {
			respond.HandleOther(c, http.StatusRequestEntityTooLarge, handler, gin.H{"error": "request body too large"})
			return
		}
		respond.HandleOther(c, http.StatusInternalServerError, handler, gin.H{"error": "internal io reading error"})
		return
	}
	ev.ReqBytes = len(reqBody)

	// getting :Command
	cmdPath := ("/command/" + c.Param("Command"))
	// if command is /command/ or /command/wake_up, set to /wake_up only
	if cmdPath == "/command/" || cmdPath == "/command/wake_up" {
		cmdPath = "/wake_up"
	}
	ev.Command = cmdPath

	if !h.allowListContains(cmdPath) {
		log.Printf("[warning] TeslaMateAPICarsCommandV1 command not allowed: %s (car_id=%d)", cmdPath, CarID)
		ev.Outcome = audit.OutcomeDenied
		ev.Reason = audit.ReasonNotInAllowList
		emit()
		respond.HandleOther(c, http.StatusUnauthorized, handler, gin.H{"error": "unauthorized"})
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
		ev.Outcome = audit.OutcomeError
		ev.Reason = audit.ReasonDBLookupFailed
		ev.ErrDetail = "no rows"
		emit()
		respond.HandleError(c, handler, "No rows were returned!", err.Error())
		return
	case nil:
		// nothing wrong.. continuing
		break
	default:
		ev.Outcome = audit.OutcomeError
		ev.Reason = audit.ReasonDBLookupFailed
		ev.ErrDetail = err.Error()
		emit()
		respond.HandleError(c, handler, CarsCommandsError1, err.Error())
		return
	}

	// load ENCRYPTION_KEY environment variable
	teslaMateEncryptionKey := h.cfg.EncryptionKey
	if teslaMateEncryptionKey == "" {
		log.Println("[error] TeslaMateAPICarsCommandV1 can't get ENCRYPTION_KEY.. will fail to perform command.")
		ev.Outcome = audit.OutcomeError
		ev.Reason = audit.ReasonMissingEncKey
		emit()
		respond.HandleOther(c, http.StatusInternalServerError, handler, gin.H{"error": "missing ENCRYPTION_KEY env variable"})
		return
	}

	// decrypt access token
	TeslaAccessToken, err = command.DecryptAccessToken(TeslaAccessToken, teslaMateEncryptionKey)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 token decrypt failed:", err)
		ev.Outcome = audit.OutcomeError
		ev.Reason = audit.ReasonTokenDecryptFail
		ev.ErrDetail = err.Error()
		emit()
		respond.HandleOther(c, http.StatusInternalServerError, handler, gin.H{"error": "unable to decrypt access token"})
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

	upstreamURL := TeslaEndpointUrl + "/api/1/vehicles/" + TeslaVehicleID + cmdPath
	ev.UpstreamURL = upstreamURL

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 http.NewRequestWithContext:", err)
		ev.Outcome = audit.OutcomeError
		ev.Reason = audit.ReasonRequestBuildFail
		ev.ErrDetail = err.Error()
		emit()
		respond.HandleOther(c, http.StatusInternalServerError, handler, gin.H{"error": "internal http request error"})
		return
	}
	req.Header.Set("Authorization", "Bearer "+TeslaAccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TeslaMateApi/"+h.cfg.APIVersion+" (+https://github.com/tobiasehlert/teslamateapi)")
	resp, err := teslaCommandHTTPClient.Do(req)

	// check response error
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 error in http request to "+TeslaEndpointUrl, err)
		ev.Outcome = audit.OutcomeUpstream
		ev.Reason = audit.ReasonUpstreamReachFail
		ev.ErrDetail = err.Error()
		emit()
		respond.HandleOther(c, http.StatusInternalServerError, handler, gin.H{"error": "internal http request error"})
		return
	}

	defer resp.Body.Close()

	respBody, err := readAllLimited(resp.Body, maxProxyResponseBodyBytes)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 error in second io.ReadAll:", err)
		ev.Outcome = audit.OutcomeUpstream
		ev.Reason = audit.ReasonReadBodyFail
		ev.UpstreamCode = resp.StatusCode
		ev.ErrDetail = err.Error()
		emit()
		if errors.Is(err, errBodyTooLarge) {
			respond.HandleOther(c, http.StatusBadGateway, handler, gin.H{"error": "upstream response too large"})
			return
		}
		respond.HandleOther(c, http.StatusInternalServerError, handler, gin.H{"error": "internal io reading error"})
		return
	}
	ev.UpstreamCode = resp.StatusCode
	ev.RespBytes = len(respBody)

	// If Tesla returns non-JSON (HTML error page, empty body), pass the raw
	// payload through instead of silently coercing to `null`.
	if jsonErr := json.Unmarshal(respBody, &jsonData); jsonErr != nil {
		log.Println("[warning] TeslaMateAPICarsCommandV1 non-JSON response from Tesla:", jsonErr)
		ev.Outcome = audit.OutcomeUpstream
		ev.Reason = audit.ReasonUpstreamNonJSON
		ev.ErrDetail = jsonErr.Error()
		emit()
		respond.HandleOther(c, resp.StatusCode, handler, dto.V1CommandRawResponse{Raw: string(respBody)})
		return
	}

	// 2xx upstream → success; 4xx/5xx → upstream_error so monitors can split.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		ev.Outcome = audit.OutcomeSuccess
	} else {
		ev.Outcome = audit.OutcomeUpstream
	}
	emit()

	// return jsonData
	// use respond.HandleOther since we use the statusCode from Tesla API
	respond.HandleOther(c, resp.StatusCode, handler, jsonData)
}
