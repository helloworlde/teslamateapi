// Package respond holds shared HTTP response helpers used across all
// handler packages. Lives in its own package to break the cycle between
// the router (httpapi) and the handlers it wires up.
package respond

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// HandleSuccess emits a 200 with j and logs the request URI. Mirrors the
// legacy TeslaMateAPIHandleSuccessResponse helper.
func HandleSuccess(c *gin.Context, handler string, j any) {
	if gin.IsDebugging() {
		log.Println("[debug] " + handler + " - (" + c.Request.RequestURI + ") returned data:")
		js, _ := json.Marshal(j)
		log.Printf("[debug] %s\n", js)
	}
	log.Println("[info] " + handler + " - (" + c.Request.RequestURI + ") executed successfully.")
	c.JSON(http.StatusOK, j)
}

// HandleOther emits an arbitrary status with j; used for command/logging
// pass-through and for the legacy 200+error envelope variants. Mirrors
// TeslaMateAPIHandleOtherResponse.
func HandleOther(c *gin.Context, httpCode int, handler string, j any) {
	log.Println("[info] " + handler + " - (" + c.Request.RequestURI + ") executed successfully.")
	c.JSON(httpCode, j)
}

// HandleError emits the legacy v1 error envelope: HTTP 200 with `{"error": s2}`.
// v1 handlers must keep this shape for backwards compatibility.
func HandleError(c *gin.Context, handler, message, detail string) {
	log.Println("[error] " + handler + " - (" + c.Request.RequestURI + "). " + message + "; " + detail)
	c.JSON(http.StatusOK, gin.H{"error": message})
}

// HandleErrorV2 emits a real HTTP status code instead of the upstream-compat
// 200+{error} envelope. v2 handlers should use this.
func HandleErrorV2(c *gin.Context, handler string, httpCode int, message, detail string) {
	log.Println("[error] " + handler + " - (" + c.Request.RequestURI + "). " + message + "; " + detail)
	c.JSON(httpCode, gin.H{"error": message})
}

// RequirePositiveIntParam parses a path/query integer with strict
// validation: non-numeric or non-positive values respond 400 and return
// ok=false. Used by every v2 handler to avoid silently coercing garbage to
// 0.
func RequirePositiveIntParam(c *gin.Context, handler, name, raw string) (int, bool) {
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		HandleErrorV2(c, handler, http.StatusBadRequest, name+" is required and must be a positive integer.", "got: "+raw)
		return 0, false
	}
	return v, true
}

// OptionalIntInRange parses a query integer with a default + min/max
// clamp. Empty string yields the default. Non-numeric or out-of-range
// values respond 400.
func OptionalIntInRange(c *gin.Context, handler, name, raw string, def, min, max int) (int, bool) {
	if raw == "" {
		return def, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < min || v > max {
		HandleErrorV2(c, handler, http.StatusBadRequest,
			fmt.Sprintf("%s must be an integer in [%d, %d].", name, min, max), "got: "+raw)
		return 0, false
	}
	return v, true
}
