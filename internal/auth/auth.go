// Package auth implements API_TOKEN bearer-token validation and Gin
// middleware for the public /api allow-list.
package auth

import (
	"crypto/subtle"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/audit"
	"github.com/tobiasehlert/teslamateapi/internal/config"
)

// Token validates incoming Authorization headers (or ?token= query) against
// the configured API_TOKEN. Stateless and safe for concurrent use.
type Token struct {
	value   string
	disable bool
}

// New constructs a Token from cfg, logging a warning when the token is
// missing or too short.
func New(cfg config.Config) *Token {
	if cfg.APIToken == "" {
		log.Println("[warning] initAuthToken - environment variable API_TOKEN not set or is empty.")
	} else if len(cfg.APIToken) < 32 {
		log.Println("[warning] initAuthToken - environment variable API_TOKEN too short.. should be 32 or longer.")
	} else {
		log.Println("[info] initAuthToken - environment variable API_TOKEN is set and good.")
	}
	return &Token{value: cfg.APIToken, disable: cfg.APITokenDisable}
}

// Validate checks the request for a valid Authorization: Bearer header or
// ?token= query value. Returns (false, message) on failure.
func (t *Token) Validate(c *gin.Context) (bool, string) {
	if t.disable {
		log.Println("[debug] validateAuthToken - header authorization bearer token disabled.")
		return true, ""
	}

	reqHeaderToken := c.Request.Header.Get("Authorization")
	if len(reqHeaderToken) > 0 {
		// Strict prefix match: the scheme must lead the header. The previous
		// strings.Split(header, "Bearer") accepted the scheme anywhere in the
		// value (e.g. "foo Bearer <token>").
		rest, isBearer := strings.CutPrefix(reqHeaderToken, "Bearer")
		if !isBearer {
			log.Println("[info] validateAuthToken - header authorization bearer token is not proper formatted.. returning 401")
			return false, "header authorization bearer token is not proper formatted"
		}
		token := strings.TrimSpace(rest)
		if token == "" {
			log.Println("[info] validateAuthToken - header authorization bearer token is empty.. returning 401")
			return false, "header authorization bearer token is empty"
		}
		if t.check(token) {
			log.Println("[debug] validateAuthToken - header authorization bearer token valid.")
			return true, ""
		}
		log.Println("[info] validateAuthToken - header authorization bearer token invalid.. returning 401")
		return false, "header authorization bearer token invalid"
	}

	tokenParamsValue := c.DefaultQuery("token", "")
	if len(tokenParamsValue) > 0 {
		if t.check(tokenParamsValue) {
			log.Println("[debug] validateAuthToken - param token valid.")
			return true, ""
		}
		log.Println("[info] validateAuthToken - param token invalid.. returning 401")
		return false, "param token invalid"
	}

	return false, "failed validation"
}

// check compares token against the configured value in constant time.
func (t *Token) check(token string) bool {
	if len(t.value) == 0 {
		log.Println("[warning] checkAuthToken - returning false (API_TOKEN is not set or empty)")
		return false
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(t.value)) != 1 {
		return false
	}
	if gin.IsDebugging() {
		log.Println("[debug] checkAuthToken - returning true")
	}
	return true
}

// publicSuffixes lists every /api/* path that bypasses bearer-auth: health,
// readiness, docs, openapi spec, ping, and the /api root itself. These have
// to be reachable from a browser or kubelet.
var publicSuffixes = []string{
	"/api",
	"/api/",
	"/api/ping",
	"/api/healthz",
	"/api/readyz",
	"/api/docs",
	"/api/openapi.yaml",
}

// Middleware enforces API_TOKEN on /api/* with the public allow-list. Auth
// is opt-in: when API_TOKEN is unset, traffic is allowed through (matches
// the legacy behaviour so deployments without a token aren't broken).
func Middleware(t *Token) gin.HandlerFunc {
	return func(c *gin.Context) {
		if t.disable {
			c.Next()
			return
		}
		if t.value == "" {
			c.Next()
			return
		}
		if slices.Contains(publicSuffixes, c.Request.URL.Path) {
			c.Next()
			return
		}
		ok, msg := t.Validate(c)
		if !ok {
			// Auth failures on privileged endpoints (Tesla command proxy /
			// TeslaMate logging proxy) are recorded here because the handler
			// never runs. We classify by path prefix rather than parsing the
			// full route so /command, /commands, /command/<x>, /wake_up, and
			// /logging/<x> all fall under the same audit umbrella.
			if action := privilegedAuditAction(c.Request.URL.Path); action != "" {
				audit.Log(audit.Event{
					Action:    action,
					Method:    c.Request.Method,
					ClientIP:  c.ClientIP(),
					UserAgent: c.Request.UserAgent(),
					Outcome:   audit.OutcomeDenied,
					Reason:    audit.ReasonUnauthorized,
					ErrDetail: msg,
				})
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}
		c.Next()
	}
}

// privilegedAuditAction returns the audit action name for paths that proxy
// to Tesla owner-api or TeslaMate logging, or "" for everything else. We
// match by substring so the same logic covers /command, /commands,
// /wake_up, and /logging/... without duplicating route patterns.
func privilegedAuditAction(p string) string {
	switch {
	case strings.Contains(p, "/logging"):
		return "logging_exec"
	case strings.Contains(p, "/command"), strings.HasSuffix(p, "/wake_up"):
		return "command_exec"
	}
	return ""
}
