package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/audit"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// teslaMateLoggingHTTPTimeout caps the outbound PUT to TeslaMate so a hung
// upstream can't pin a goroutine indefinitely.
const teslaMateLoggingHTTPTimeout = 30 * time.Second

var teslaMateLoggingHTTPClient = &http.Client{Timeout: teslaMateLoggingHTTPTimeout}

// TeslaMateAPICarsLoggingListV1 returns the allow-listed logging commands.
//
// @Summary      List logging commands
// @Description  Returns the allow-list. Requires ENABLE_COMMANDS=true.
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path      int  true  "TeslaMate cars.id"
// @Success      200    {object}  dto.V1LoggingList
// @Failure      401    {object}  dto.ErrorEnvelope
// @Failure      403    {object}  dto.ErrorEnvelope  "ENABLE_COMMANDS=false"
// @Router       /api/v1/cars/{CarID}/logging [get]
func (h *Handler) LoggingList(c *gin.Context) { h.Logging(c) }

// TeslaMateAPICarsLoggingExecV1 invokes a TeslaMate logging command.
//
// @Summary      Invoke logging command
// @Description  Proxies to TeslaMate /api/car/{id}/logging/{Command}. Status code and body mirror TeslaMate's. Requires ENABLE_COMMANDS=true.
// @Tags         v1
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        CarID    path      int     true  "TeslaMate cars.id"
// @Param        Command  path      string  true  "Logging command"
// @Success      200      {object}  dto.V1LoggingResult  "TeslaMate passthrough JSON"
// @Failure      400      {object}  dto.ErrorEnvelope
// @Failure      401      {object}  dto.ErrorEnvelope
// @Failure      403      {object}  dto.ErrorEnvelope  "ENABLE_COMMANDS=false"
// @Failure      500      {object}  dto.ErrorEnvelope
// @Router       /api/v1/cars/{CarID}/logging/{Command} [put]
func (h *Handler) LoggingExec(c *gin.Context) { h.Logging(c) }

// TeslaMateAPICarsLoggingV1 lists or invokes TeslaMate logging commands.
// Routed via the per-method wrappers above so swag can document the GET and
// PUT response shapes separately.
//
// Every PUT invocation emits one [audit] log line — see internal/audit. GET
// (list) is not privileged and is not audited.
func (h *Handler) Logging(c *gin.Context) {

	const handler = "TeslaMateAPICarsLoggingV1"
	var (
		jsonData map[string]any
		err      error
	)

	startedAt := time.Now()
	ev := audit.Event{
		Action:    "logging_exec",
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
		log.Println("[warning] TeslaMateAPICarsLoggingV1 ENABLE_COMMANDS is not true.. returning 403 forbidden.")
		ev.Outcome = audit.OutcomeDenied
		ev.Reason = audit.ReasonCommandsDisabled
		emit()
		respond.HandleOther(c, http.StatusForbidden, handler, gin.H{"error": "You are not allowed to access logging commands"})
		return
	}

	// if request method is GET return list of commands
	if c.Request.Method == http.MethodGet {
		respond.HandleSuccess(c, handler, dto.V1LoggingList{EnabledCommands: h.allowListItems()})
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
		log.Println("[error] TeslaMateAPICarsLoggingV1 error in first io.ReadAll", err)
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
	command := ("/logging/" + c.Param("Command"))
	ev.Command = command

	if !h.allowListContains(command) {
		log.Printf("[warning] TeslaMateAPICarsLoggingV1 command not allowed: %s (car_id=%d)", command, CarID)
		ev.Outcome = audit.OutcomeDenied
		ev.Reason = audit.ReasonNotInAllowList
		emit()
		respond.HandleOther(c, http.StatusUnauthorized, handler, gin.H{"error": "unauthorized"})
		return
	}

	putURL := ""
	if h.cfg.TeslaMateSSL {
		putURL = "https://"
	} else {
		putURL = "http://"
	}
	putURL = putURL + h.cfg.TeslaMateHost + ":" + h.cfg.TeslaMatePort + "/api/car/" + strconv.Itoa(CarID) + command
	ev.UpstreamURL = putURL
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPut, putURL, bytes.NewReader(reqBody))
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 http.NewRequestWithContext:", err)
		ev.Outcome = audit.OutcomeError
		ev.Reason = audit.ReasonRequestBuildFail
		ev.ErrDetail = err.Error()
		emit()
		respond.HandleOther(c, http.StatusInternalServerError, handler, gin.H{"error": "internal http request error"})
		return
	}
	req.Header.Set("User-Agent", "TeslaMateApi/"+h.cfg.APIVersion+" https://github.com/tobiasehlert/teslamateapi")
	resp, err := teslaMateLoggingHTTPClient.Do(req)

	// check response error
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 error in http request to http://teslamate:", err)
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
		log.Println("[error] TeslaMateAPICarsLoggingV1 error in second io.ReadAll:", err)
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

	if jsonErr := json.Unmarshal(respBody, &jsonData); jsonErr != nil {
		log.Println("[warning] TeslaMateAPICarsLoggingV1 non-JSON response from TeslaMate:", jsonErr)
		ev.Outcome = audit.OutcomeUpstream
		ev.Reason = audit.ReasonUpstreamNonJSON
		ev.ErrDetail = jsonErr.Error()
		emit()
		respond.HandleOther(c, resp.StatusCode, handler, dto.V1LoggingRawResponse{Raw: string(respBody)})
		return
	}

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
