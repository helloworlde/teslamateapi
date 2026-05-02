package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func sanitizedRequestURI(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	u := *c.Request.URL
	query := u.Query()
	for key := range query {
		switch strings.ToLower(key) {
		case "token", "api_token", "access_token", "refresh_token", "authorization":
			query.Set(key, "[REDACTED]")
		}
	}
	u.RawQuery = query.Encode()
	return u.RequestURI()
}

func responseErrorDetails(c *gin.Context, status int, code, message string, details map[string]any) map[string]any {
	if status >= http.StatusInternalServerError {
		log.Printf("[error] api_error - (%s) status=%d code=%s message=%s details=%v", sanitizedRequestURI(c), status, code, message, details)
		return nil
	}
	if status >= http.StatusBadRequest {
		log.Printf("[warning] api_error - (%s) status=%d code=%s message=%s details=%v", sanitizedRequestURI(c), status, code, message, details)
	}
	return details
}
