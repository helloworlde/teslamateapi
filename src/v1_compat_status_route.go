package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var mqttStatusCache *statusCache

// TeslaMateAPICarsStatusRouteV1 返回兼容响应结构的车辆状态。
// @Summary 当前车辆状态
// @Tags 兼容 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} CarStatusV1Envelope
// @Router /v1/cars/{CarID}/status [get]
func TeslaMateAPICarsStatusRouteV1(c *gin.Context) {
	if mqttStatusCache == nil {
		TeslaMateAPIHandleOtherResponse(c, http.StatusNotImplemented, "TeslaMateAPICarsStatusRouteV1", gin.H{"error": "mqtt disabled.. status not accessible!"})
		return
	}

	mqttStatusCache.TeslaMateAPICarsStatusV1(c)
}
