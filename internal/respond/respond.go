// Package respond holds shared HTTP response helpers used across all
// handler packages. Lives in its own package to break the cycle between
// the router (httpapi) and the handlers it wires up.
package respond

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/httpparams"
)

// HandleSuccess emits a 200 with j and, in debug mode, logs the request
// URI and payload.
func HandleSuccess(c *gin.Context, handler string, j any) {
	if gin.IsDebugging() {
		log.Println("[debug] " + handler + " - (" + SafeRequestURI(c) + ") returned data:")
		js, _ := json.Marshal(j)
		log.Printf("[debug] %s\n", js)
	}
	c.JSON(http.StatusOK, j)
}

// HandleOther emits an arbitrary status with j; used for command/logging
// pass-through and for the legacy 200+error envelope variants.
func HandleOther(c *gin.Context, httpCode int, j any) {
	c.JSON(httpCode, j)
}

// HandleError emits the legacy v1 error envelope: HTTP 200 with `{"error": s2}`.
// v1 handlers must keep this shape for backwards compatibility.
func HandleError(c *gin.Context, handler, message, detail string) {
	log.Println("[error] " + handler + " - (" + SafeRequestURI(c) + "). " + message + "; " + detail)
	c.JSON(http.StatusOK, gin.H{"error": message})
}

// HandleErrorV2 emits a real HTTP status code instead of the upstream-compat
// 200+{error} envelope. v2 handlers should use this.
func HandleErrorV2(c *gin.Context, handler string, httpCode int, message, detail string) {
	log.Println("[error] " + handler + " - (" + SafeRequestURI(c) + "). " + message + "; " + detail)
	c.JSON(httpCode, gin.H{"error": message})
}

// SafeRequestURI returns the request URI with sensitive query values redacted.
func SafeRequestURI(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	return SafeRequestURIFromURL(c.Request.URL)
}

// SafeRequestURIFromURL returns the request URI with sensitive query values redacted.
func SafeRequestURIFromURL(rawURL *url.URL) string {
	if rawURL == nil {
		return ""
	}
	u := *rawURL
	q := u.Query()
	for key := range q {
		lowerKey := strings.ToLower(key)
		if lowerKey == "token" || strings.Contains(lowerKey, "token") {
			q.Set(key, "[REDACTED]")
		}
	}
	u.RawQuery = q.Encode()
	return u.RequestURI()
}

// RequirePositiveIntParam parses a path/query integer with strict
// validation: non-numeric or non-positive values respond 400 and return
// ok=false. Used by every v2 handler to avoid silently coercing garbage to
// 0.
func RequirePositiveIntParam(c *gin.Context, handler, name, raw string) (int, bool) {
	v, err := httpparams.PositiveInt(raw)
	if err != nil {
		HandleErrorV2(c, handler, http.StatusBadRequest, name+" is required and must be a positive integer.", "got: "+raw)
		return 0, false
	}
	return v, true
}

// OptionalIntInRange parses a query integer with a default + min/max
// clamp. Empty string yields the default. Non-numeric or out-of-range
// values respond 400.
func OptionalIntInRange(c *gin.Context, handler, name, raw string, def, min, max int) (int, bool) {
	v, err := httpparams.OptionalIntInRange(raw, def, min, max)
	if err != nil {
		HandleErrorV2(c, handler, http.StatusBadRequest,
			fmt.Sprintf("%s must be an integer in [%d, %d].", name, min, max), "got: "+raw)
		return 0, false
	}
	return v, true
}
