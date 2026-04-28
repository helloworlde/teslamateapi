package main

import (
	"database/sql"
	"errors"
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

func nonFatalWarning(code, message string, fields map[string]any, err error) map[string]any {
	if err != nil {
		log.Printf("[warning] nonfatal_warning - code=%s message=%s details=%v", code, message, err)
	}
	warning := map[string]any{
		"code":    code,
		"message": message,
	}
	for key, value := range fields {
		warning[key] = value
	}
	return warning
}

func TeslaMateAPIHandleErrorResponseWithStatus(c *gin.Context, status int, logPrefix, message, detail string) {
	log.Println("[error] " + logPrefix + " - (" + sanitizedRequestURI(c) + "). " + message + "; " + detail)
	body := gin.H{"error": message}
	if detail != "" {
		body["detail"] = detail
	}
	c.JSON(status, body)
}

func respondSummaryMetadataError(c *gin.Context, actionName string, err error, unableMsg string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusNotFound, actionName, "Car not found.", "")
		return true
	}
	TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, unableMsg, err.Error())
	return true
}
