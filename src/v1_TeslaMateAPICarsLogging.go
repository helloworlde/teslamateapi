package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/tobiasehlert/teslamateapi/src/dto"
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
func TeslaMateAPICarsLoggingListV1(c *gin.Context) { TeslaMateAPICarsLoggingV1(c) }

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
func TeslaMateAPICarsLoggingExecV1(c *gin.Context) { TeslaMateAPICarsLoggingV1(c) }

// TeslaMateAPICarsLoggingV1 lists or invokes TeslaMate logging commands.
// Routed via the per-method wrappers above so swag can document the GET and
// PUT response shapes separately.
func TeslaMateAPICarsLoggingV1(c *gin.Context) {

	// creating required vars
	var (
		jsonData map[string]any
		err      error
	)

	// check if commands are enabled.. if not we need to abort
	if !getEnvAsBool("ENABLE_COMMANDS", false) {
		log.Println("[warning] TeslaMateAPICarsLoggingV1 ENABLE_COMMANDS is not true.. returning 403 forbidden.")
		TeslaMateAPIHandleOtherResponse(c, http.StatusForbidden, "TeslaMateAPICarsLoggingV1", gin.H{"error": "You are not allowed to access logging commands"})
		return
	}

	// if request method is GET return list of commands
	if c.Request.Method == http.MethodGet {
		TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsLoggingV1", dto.V1LoggingList{EnabledCommands: allowList})
		return
	}

	// authentication for the endpoint
	validToken, errorMessage := validateAuthToken(c)
	if !validToken {
		TeslaMateAPIHandleOtherResponse(c, http.StatusUnauthorized, "TeslaMateAPICarsLoggingV1", gin.H{"error": errorMessage})
		return
	}

	// getting CarID param from URL and validating that it's not zero
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		log.Println("[error] TeslaMateAPICarsLoggingV1 CarID is invalid (zero)!")
		TeslaMateAPIHandleOtherResponse(c, http.StatusBadRequest, "TeslaMateAPICarsLoggingV1", gin.H{"error": "CarID invalid"})
		return
	}

	// getting request body to pass to Tesla
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 error in first io.ReadAll", err)
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsLoggingV1", gin.H{"error": "internal io reading error"})
		return
	}

	// getting :Command
	command := ("/logging/" + c.Param("Command"))

	if !checkArrayContainsString(allowList, command) {
		log.Printf("[warning] TeslaMateAPICarsLoggingV1 command not allowed!")
		TeslaMateAPIHandleOtherResponse(c, http.StatusUnauthorized, "TeslaMateAPICarsLoggingV1", gin.H{"error": "unauthorized"})
		return
	}

	client := &http.Client{Timeout: teslaMateLoggingHTTPTimeout}
	putURL := ""
	if getEnvAsBool("TESLAMATE_SSL", false) {
		putURL = "https://"
	} else {
		putURL = "http://"
	}
	putURL = putURL + getEnv("TESLAMATE_HOST", "teslamate") + ":" + getEnv("TESLAMATE_PORT", "4000") + "/api/car/" + strconv.Itoa(CarID) + command
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPut, putURL, strings.NewReader(string(reqBody)))
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 http.NewRequestWithContext:", err)
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsLoggingV1", gin.H{"error": "internal http request error"})
		return
	}
	req.Header.Set("User-Agent", "TeslaMateApi/"+apiVersion+" https://github.com/tobiasehlert/teslamateapi")
	resp, err := client.Do(req)

	// check response error
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 error in http request to http://teslamate:", err)
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsLoggingV1", gin.H{"error": "internal http request error"})
		return
	}

	defer resp.Body.Close()
	defer client.CloseIdleConnections()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsLoggingV1 error in second io.ReadAll:", err)
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsLoggingV1", gin.H{"error": "internal io reading error"})
		return
	}
	if jsonErr := json.Unmarshal(respBody, &jsonData); jsonErr != nil {
		log.Println("[warning] TeslaMateAPICarsLoggingV1 non-JSON response from TeslaMate:", jsonErr)
		TeslaMateAPIHandleOtherResponse(c, resp.StatusCode, "TeslaMateAPICarsLoggingV1", dto.V1LoggingRawResponse{Raw: string(respBody)})
		return
	}

	// return jsonData
	// use TeslaMateAPIHandleOtherResponse since we use the statusCode from Tesla API
	TeslaMateAPIHandleOtherResponse(c, resp.StatusCode, "TeslaMateAPICarsLoggingV1", jsonData)
}
