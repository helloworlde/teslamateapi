package main

import (
	"github.com/gin-gonic/gin"
)

func registerCompatibleV1Routes(v1 *gin.RouterGroup) {
	// 兼容接口保持原 TeslaMateApi 路由和响应结构，避免影响已有客户端。
	v1.GET("/cars", TeslaMateAPICarsV1)
	v1.GET("/cars/:CarID", TeslaMateAPICarsV1)
	v1.GET("/cars/:CarID/battery-health", TeslaMateAPICarsBatteryHealthV1)
	v1.GET("/cars/:CarID/charges", TeslaMateAPICarsChargesV1)
	v1.GET("/cars/:CarID/charges/current", TeslaMateAPICarsChargesCurrentV1)
	v1.GET("/cars/:CarID/charges/:ChargeID", TeslaMateAPICarsChargesDetailsV1)
	v1.GET("/cars/:CarID/drives", TeslaMateAPICarsDrivesV1)
	v1.GET("/cars/:CarID/drives/:DriveID", TeslaMateAPICarsDrivesDetailsV1)
	v1.GET("/cars/:CarID/status", TeslaMateAPICarsStatusRouteV1)
	v1.GET("/cars/:CarID/updates", TeslaMateAPICarsUpdatesV1)
	v1.GET("/globalsettings", TeslaMateAPIGlobalsettingsV1)

	registerCommandV1Routes(v1)
}

func commandRoutesEnabled() bool {
	return getEnvAsBool("ENABLE_COMMANDS", false)
}

func registerCommandV1Routes(v1 *gin.RouterGroup) {
	// 车辆命令具备外部副作用，必须通过环境变量显式启用才注册路由。
	if !commandRoutesEnabled() {
		return
	}
	v1.GET("/cars/:CarID/command", TeslaMateAPICarsCommandV1)
	v1.POST("/cars/:CarID/command/:Command", TeslaMateAPICarsCommandV1)
	v1.GET("/cars/:CarID/logging", TeslaMateAPICarsLoggingV1)
	v1.PUT("/cars/:CarID/logging/:Command", TeslaMateAPICarsLoggingV1)
	v1.POST("/cars/:CarID/wake_up", TeslaMateAPICarsCommandV1)
}

func registerExtendedV1Routes(v1 *gin.RouterGroup) {
	// Real-time vehicle state — no period param, always fresh.
	v1.GET("/cars/:CarID/status", TeslaMateAPICarsStatusV2)

	// Period KPIs — unified stats replacing the old summary/statistics/dashboard trio.
	v1.GET("/cars/:CarID/stats", TeslaMateAPICarsStatsV2)

	// Calendar-style activity view.
	v1.GET("/cars/:CarID/activity", TeslaMateAPICarsActivityV2)

	// Unified time series with ?scope=drives|charges|battery|states.
	v1.GET("/cars/:CarID/series", TeslaMateAPICarsSeriesV2)

	// Unified distribution histograms with ?scope=drives|charges.
	v1.GET("/cars/:CarID/distributions", TeslaMateAPICarsDistributionsV2)

	// Paginated event history, one resource per event type.
	v1.GET("/cars/:CarID/drives", TeslaMateAPICarsDrivesHistoryV2)
	v1.GET("/cars/:CarID/charges", TeslaMateAPICarsChargesHistoryV2)

	// Spatial data.
	v1.GET("/cars/:CarID/locations", TeslaMateAPICarsLocationsV2)
	v1.GET("/cars/:CarID/locations/heatmap", TeslaMateAPICarsLocationsHeatmapV2)

	// Analytical features grouped under /analysis/.
	v1.GET("/cars/:CarID/analysis/insights", TeslaMateAPICarsAnalysisInsightsV2)
	v1.GET("/cars/:CarID/analysis/trends", TeslaMateAPICarsAnalysisTrendsV2)
	v1.GET("/cars/:CarID/analysis/records", TeslaMateAPICarsAnalysisRecordsV2)
}
