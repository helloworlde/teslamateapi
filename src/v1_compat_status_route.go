package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var mqttStatusCache *statusCache

func TeslaMateAPICarsStatusRouteV1(c *gin.Context) {
	if mqttStatusCache == nil {
		TeslaMateAPIHandleOtherResponse(c, http.StatusNotImplemented, "TeslaMateAPICarsStatusRouteV1", gin.H{"error": "mqtt disabled.. status not accessible!"})
		return
	}

	mqttStatusCache.TeslaMateAPICarsStatusV1(c)
}
