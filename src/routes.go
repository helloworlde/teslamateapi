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
	// 扩展接口按使用场景拆分：摘要、仪表盘、实时、时序、分布、洞察、地图等职责分离。
	v1.GET("/cars/:CarID/summary", TeslaMateAPICarsSummaryV2)
	v1.GET("/cars/:CarID/dashboard", TeslaMateAPICarsDashboardV2)
	v1.GET("/cars/:CarID/realtime", TeslaMateAPICarsRealtimeV2)
	v1.GET("/cars/:CarID/calendar", TeslaMateAPICarsCalendarV2)
	v1.GET("/cars/:CarID/statistics", TeslaMateAPICarsUnifiedStatisticsV2)
	v1.GET("/cars/:CarID/series/drives", TeslaMateAPICarsDriveSeriesV2)
	v1.GET("/cars/:CarID/series/charges", TeslaMateAPICarsChargeSeriesV2)
	v1.GET("/cars/:CarID/series/battery", TeslaMateAPICarsBatterySeriesV2)
	v1.GET("/cars/:CarID/series/states", TeslaMateAPICarsStateSeriesV2)
	v1.GET("/cars/:CarID/distributions/drives", TeslaMateAPICarsDriveDistributionsV2)
	v1.GET("/cars/:CarID/distributions/charges", TeslaMateAPICarsChargeDistributionsV2)
	v1.GET("/cars/:CarID/insights", TeslaMateAPICarsUnifiedInsightsV2)
	v1.GET("/cars/:CarID/timeline", TeslaMateAPICarsUnifiedTimelineV2)
	v1.GET("/cars/:CarID/map/visited", TeslaMateAPICarsMapVisitedUnifiedV2)
	v1.GET("/cars/:CarID/locations", TeslaMateAPICarsLocationsV2)
}
