package main

import (
	"github.com/gin-gonic/gin"
)

func registerCompatibleV1Routes(v1 *gin.RouterGroup) {
	// 兼容接口保持原 TeslaMateApi 路由和响应结构，避免影响已有客户端。
	v1.GET("/cars", TeslaMateAPICarsListV1)
	v1.GET("/cars/:CarID", TeslaMateAPICarByIDV1)
	v1.GET("/cars/:CarID/battery-health", TeslaMateAPICarsBatteryHealthV1)
	v1.GET("/cars/:CarID/charges", TeslaMateAPICarsChargesV1)
	v1.GET("/cars/:CarID/charges/current", TeslaMateAPICarsChargesCurrentV1)
	v1.GET("/cars/:CarID/charges/:ChargeID", TeslaMateAPICarsChargesDetailsV1)
	v1.GET("/cars/:CarID/drives", TeslaMateAPICarsDrivesV1)
	v1.GET("/cars/:CarID/drives/:DriveID", TeslaMateAPICarsDrivesDetailsV1)
	v1.GET("/cars/:CarID/status", TeslaMateAPICarsStatusRouteV1)
	v1.GET("/cars/:CarID/updates", TeslaMateAPICarsUpdatesV1)
	v1.GET("/globalsettings", TeslaMateAPIGlobalsettingsV1)
}

func registerExtendedV2Routes(group *gin.RouterGroup) {
	// 周期核心指标，统一替代旧的 summary、statistics、dashboard 组合。
	group.GET("/cars/:CarID/stats", TeslaMateAPICarsStatsV2)

	// 日历视角的活动聚合。
	group.GET("/cars/:CarID/activity", TeslaMateAPICarsActivityV2)

	// 统一时序接口，通过 scope 选择 drives、charges、battery 或 states。
	group.GET("/cars/:CarID/series", TeslaMateAPICarsSeriesV2)

	// 统一分布直方图接口，通过 scope 选择 drives 或 charges。
	group.GET("/cars/:CarID/distributions", TeslaMateAPICarsDistributionsV2)

	// 空间位置数据。
	group.GET("/cars/:CarID/locations", TeslaMateAPICarsLocationsV2)
	group.GET("/cars/:CarID/locations/heatmap", TeslaMateAPICarsLocationsHeatmapV2)

	// 分析能力统一放在 /analysis/ 下。
	group.GET("/cars/:CarID/analysis/insights", TeslaMateAPICarsAnalysisInsightsV2)
	group.GET("/cars/:CarID/analysis/trends", TeslaMateAPICarsAnalysisTrendsV2)
	group.GET("/cars/:CarID/analysis/records", TeslaMateAPICarsAnalysisRecordsV2)
}
