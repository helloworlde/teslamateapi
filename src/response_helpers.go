package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func TeslaMateAPIHandleErrorResponseWithStatus(c *gin.Context, status int, logPrefix, message, detail string) {
	log.Println("[error] " + logPrefix + " - (" + c.Request.RequestURI + "). " + message + "; " + detail)
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
