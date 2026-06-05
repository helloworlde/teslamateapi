// Package system holds the unauthenticated banner and probe handlers
// (root, /api, /api/v1, /api/v2, ping, healthz, readyz, 404).
package system

import (
	"log"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/respond"
)

// Handler bundles the readiness flag for the probe handlers. All other
// handlers in this package are stateless.
type Handler struct {
	Ready *atomic.Value
}

// New returns a Handler wired with the given readiness flag.
func New(ready *atomic.Value) *Handler { return &Handler{Ready: ready} }

// Root returns the handler for GET / — a tiny liveness banner that echoes
// the configured base path.
//
// @Summary      Root banner
// @Description  Returns a small banner confirming the API process is running.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.MessageEnvelope
// @Router       / [get]
func Root(r *gin.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": r.BasePath()})
	}
}

// APIRoot returns the banner handler for GET /api.
//
// @Summary      /api banner
// @Description  Banner for the /api root.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.MessageEnvelope
// @Router       /api/ [get]
func APIRoot(api *gin.RouterGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": api.BasePath()})
	}
}

// APIV1Root returns the banner handler for GET /api/v1.
//
// @Summary      /api/v1 banner
// @Description  Banner for the /api/v1 root.
// @Tags         system
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dto.MessageEnvelope
// @Router       /api/v1/ [get]
func APIV1Root(v1 *gin.RouterGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi v1 running..", "path": v1.BasePath()})
	}
}

// APIV2Root returns the banner handler for GET /api/v2.
//
// @Summary      /api/v2 banner
// @Description  Banner for the /api/v2 root.
// @Tags         system
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dto.MessageEnvelope
// @Router       /api/v2/ [get]
func APIV2Root(v2 *gin.RouterGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi v2 running..", "path": v2.BasePath()})
	}
}

// Ping answers the unauthenticated /api/ping liveness check.
//
// @Summary      Ping
// @Description  Returns {"message":"pong"} for simple uptime checks. No auth required.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.PongResponse
// @Router       /api/ping [get]
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

// Healthz is a liveness probe.
//
// @Summary      Liveness probe
// @Description  Returns 200 as long as the process is up. No auth required.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.HealthResponse
// @Router       /api/healthz [get]
func Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": http.StatusText(http.StatusOK)})
}

// Readyz is a readiness probe.
//
// @Summary      Readiness probe
// @Description  Returns 200 when MQTT is connected (or DISABLE_MQTT=true). 503 otherwise. No auth required.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.ReadyResponse
// @Failure      503  {object}  dto.ErrorEnvelope
// @Router       /api/readyz [get]
func (h *Handler) Readyz(c *gin.Context) {
	if h.Ready == nil || !h.Ready.Load().(bool) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": http.StatusText(http.StatusServiceUnavailable)})
		return
	}
	log.Println("[info] webserver - (" + respond.SafeRequestURI(c) + ") executed successfully.")
	c.JSON(http.StatusOK, gin.H{"status": http.StatusText(http.StatusOK)})
}

// NotFound renders a JSON 404 envelope for routes that don't match.
//
// @Summary      404 fallback
// @Description  JSON 404 envelope returned for any unmatched route.
// @Tags         system
// @Produce      json
// @Success      404  {object}  dto.NotFoundResponse
func NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
}
