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

	"github.com/tobiasehlert/teslamateapi/internal/config"
)

// Token validates incoming Authorization headers (or ?token= query) against
// the configured API_TOKEN. Stateless and safe for concurrent use.
type Token struct {
	value   string
	disable bool
}

// New constructs a Token from cfg, logging the same warnings as the legacy
// initAuthToken func.
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

// Configured reports whether an API_TOKEN was set.
func (t *Token) Configured() bool { return t.value != "" }

// Disabled reports whether API_TOKEN_DISABLE was set.
func (t *Token) Disabled() bool { return t.disable }

// Validate checks the request for a valid Authorization: Bearer header or
// ?token= query value. Returns (false, message) on failure.
func (t *Token) Validate(c *gin.Context) (bool, string) {
	if t.disable {
		log.Println("[debug] validateAuthToken - header authorization bearer token disabled.")
		return true, ""
	}

	reqHeaderToken := c.Request.Header.Get("Authorization")
	if len(reqHeaderToken) > 0 {
		splitToken := strings.Split(reqHeaderToken, "Bearer")
		if len(splitToken) != 2 {
			log.Println("[info] validateAuthToken - header authorization bearer token is not proper formatted.. returning 401")
			return false, "header authorization bearer token is not proper formatted"
		} else if strings.TrimSpace(splitToken[1]) == "" {
			log.Println("[info] validateAuthToken - header authorization bearer token is empty.. returning 401")
			return false, "header authorization bearer token is empty"
		} else if t.check(strings.TrimSpace(splitToken[1])) {
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}
		c.Next()
	}
}
