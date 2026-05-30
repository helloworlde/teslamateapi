package v1

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// teslaMateLoggingHTTPTimeout caps the outbound PUT to TeslaMate so a hung
// upstream can't pin a goroutine indefinitely.
const teslaMateLoggingHTTPTimeout = 30 * time.Second

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
func (h *Handler) Logging(c *gin.Context) {

	// creating required vars
	var (
		jsonData map[string]any
		err      error
	)

	// check if commands are enabled.. if not we need to abort
	if !h.cfg.CommandsEnabled {
		log.Println("[warning] TeslaMateAPICarsLoggingV1 ENABLE_COMMANDS is not true.. returning 403 forbidden.")
		respond.HandleOther(c, http.StatusForbidden, "TeslaMateAPICarsLoggingV1", gin.H{"error": "You are not allowed to access logging commands"})
		return
	}

	// if request method is GET return list of commands
	if c.Request.Method == http.MethodGet {
		respond.HandleSuccess(c, "TeslaMateAPICarsLoggingV1", dto.V1LoggingList{EnabledCommands: h.allowListItems()})
		return
	}

	// authentication for the endpoint
	validToken, errorMessage := h.validateAuthToken(c)
	if !validToken {
		respond.HandleOther(c, http.StatusUnauthorized, "TeslaMateAPICarsLoggingV1", gin.H{"error": errorMessage})
		return
	}

	// getting CarID param from URL and validating that it's not zero
	CarID := convert.StrToInt(c.Param("CarID"))
	if CarID == 0 {
		log.Println("[error] TeslaMateAPICarsLoggingV1 CarID is invalid (zero)!")
		respond.HandleOther(c, http.StatusBadRequest, "TeslaMateAPICarsLoggingV1", gin.H{"error": "CarID invalid"})
		return
	}

	// getting request body to pass to Tesla
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 error in first io.ReadAll", err)
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsLoggingV1", gin.H{"error": "internal io reading error"})
		return
	}

	// getting :Command
	command := ("/logging/" + c.Param("Command"))

	if !h.allowListContains(command) {
		log.Printf("[warning] TeslaMateAPICarsLoggingV1 command not allowed!")
		respond.HandleOther(c, http.StatusUnauthorized, "TeslaMateAPICarsLoggingV1", gin.H{"error": "unauthorized"})
		return
	}

	client := &http.Client{Timeout: teslaMateLoggingHTTPTimeout}
	putURL := ""
	if h.cfg.TeslaMateSSL {
		putURL = "https://"
	} else {
		putURL = "http://"
	}
	putURL = putURL + h.cfg.TeslaMateHost + ":" + h.cfg.TeslaMatePort + "/api/car/" + strconv.Itoa(CarID) + command
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPut, putURL, strings.NewReader(string(reqBody)))
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 http.NewRequestWithContext:", err)
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsLoggingV1", gin.H{"error": "internal http request error"})
		return
	}
	req.Header.Set("User-Agent", "TeslaMateApi/"+h.cfg.APIVersion+" https://github.com/tobiasehlert/teslamateapi")
	resp, err := client.Do(req)

	// check response error
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 error in http request to http://teslamate:", err)
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsLoggingV1", gin.H{"error": "internal http request error"})
		return
	}

	defer resp.Body.Close()
	defer client.CloseIdleConnections()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 error in second io.ReadAll:", err)
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsLoggingV1", gin.H{"error": "internal io reading error"})
		return
	}
	if jsonErr := json.Unmarshal(respBody, &jsonData); jsonErr != nil {
		log.Println("[warning] TeslaMateAPICarsLoggingV1 non-JSON response from TeslaMate:", jsonErr)
		respond.HandleOther(c, resp.StatusCode, "TeslaMateAPICarsLoggingV1", dto.V1LoggingRawResponse{Raw: string(respBody)})
		return
	}

	// return jsonData
	// use respond.HandleOther since we use the statusCode from Tesla API
	respond.HandleOther(c, resp.StatusCode, "TeslaMateAPICarsLoggingV1", jsonData)
}
