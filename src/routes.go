package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerDocsRoutes(v1 *gin.RouterGroup, basePathV1 string) {
	v1.GET("/docs", serveScalarAPIReference)
	v1.GET("/docs/openapi.json", serveOpenAPIDocumentJSON)
	v1.GET("/docs/swagger", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePathV1+"/docs/swagger/index.html") })
	v1.GET("/docs/swagger/index.html", serveScalarAPIReference)
	v1.GET("/docs/swagger/doc.json", serveSwaggerDocJSON)
}

func registerCompatibleV1Routes(v1 *gin.RouterGroup) {
	v1.GET("/cars", TeslaMateAPICarsV1)
	v1.GET("/cars/:CarID", TeslaMateAPICarsV1)
	v1.GET("/cars/:CarID/battery-health", TeslaMateAPICarsBatteryHealthV1)
	v1.GET("/cars/:CarID/charges", TeslaMateAPICarsChargesV1)
	v1.GET("/cars/:CarID/charges/current", TeslaMateAPICarsChargesCurrentV1)
	v1.GET("/cars/:CarID/charges/:ChargeID", TeslaMateAPICarsChargesDetailsV1)
	v1.GET("/cars/:CarID/command", TeslaMateAPICarsCommandV1)
	v1.POST("/cars/:CarID/command/:Command", TeslaMateAPICarsCommandV1)
	v1.GET("/cars/:CarID/drives", TeslaMateAPICarsDrivesV1)
	v1.GET("/cars/:CarID/drives/:DriveID", TeslaMateAPICarsDrivesDetailsV1)
	v1.GET("/cars/:CarID/logging", TeslaMateAPICarsLoggingV1)
	v1.PUT("/cars/:CarID/logging/:Command", TeslaMateAPICarsLoggingV1)
	v1.GET("/cars/:CarID/status", TeslaMateAPICarsStatusRouteV1)
	v1.GET("/cars/:CarID/updates", TeslaMateAPICarsUpdatesV1)
	v1.POST("/cars/:CarID/wake_up", TeslaMateAPICarsCommandV1)
	v1.GET("/globalsettings", TeslaMateAPIGlobalsettingsV1)
}

func registerExtendedV1Routes(v1 *gin.RouterGroup) {
	v1.GET("/cars/:CarID/summary", TeslaMateAPICarsSummaryV2)
	v1.GET("/cars/:CarID/statistics", TeslaMateAPICarsStatisticsV2)
	v1.GET("/cars/:CarID/charts/overview", TeslaMateAPICarsChartsOverviewV2)
	v1.GET("/cars/:CarID/charts/drives/distance", TeslaMateAPICarsDriveDistanceChartV2)
	v1.GET("/cars/:CarID/charts/drives/energy", TeslaMateAPICarsDriveEnergyChartV2)
	v1.GET("/cars/:CarID/charts/drives/efficiency", TeslaMateAPICarsDriveEfficiencyChartV2)
	v1.GET("/cars/:CarID/charts/drives/speed", TeslaMateAPICarsDriveSpeedChartV2)
	v1.GET("/cars/:CarID/charts/drives/temperature", TeslaMateAPICarsDriveTemperatureChartV2)
	v1.GET("/cars/:CarID/charts/charges/energy", TeslaMateAPICarsChargeEnergyChartV2)
	v1.GET("/cars/:CarID/charts/charges/cost", TeslaMateAPICarsChargeCostChartV2)
	v1.GET("/cars/:CarID/charts/charges/efficiency", TeslaMateAPICarsChargeEfficiencyChartV2)
	v1.GET("/cars/:CarID/charts/charges/power", TeslaMateAPICarsChargePowerChartV2)
	v1.GET("/cars/:CarID/charts/charges/location", TeslaMateAPICarsChargeLocationChartV2)
	v1.GET("/cars/:CarID/charts/charges/soc", TeslaMateAPICarsChargeSOCChartV2)
	v1.GET("/cars/:CarID/charts/battery/range", TeslaMateAPICarsBatteryRangeChartV2)
	v1.GET("/cars/:CarID/charts/battery/health", TeslaMateAPICarsBatteryHealthChartV2)
	v1.GET("/cars/:CarID/charts/states/duration", TeslaMateAPICarsStateDurationChartV2)
	v1.GET("/cars/:CarID/charts/vampire-drain", TeslaMateAPICarsVampireDrainChartV2)
	v1.GET("/cars/:CarID/charts/mileage", TeslaMateAPICarsMileageChartV2)
	v1.GET("/cars/:CarID/drives/:DriveID/details", TeslaMateAPICarsDriveDetailsV2)
	v1.GET("/cars/:CarID/charges/:ChargeID/details", TeslaMateAPICarsChargeDetailsV2)
	v1.GET("/cars/:CarID/timeline", TeslaMateAPICarsTimelineV2)
	v1.GET("/cars/:CarID/calendar/drives", TeslaMateAPICarsDriveCalendarV2)
	v1.GET("/cars/:CarID/calendar/charges", TeslaMateAPICarsChargeCalendarV2)
	v1.GET("/cars/:CarID/map/visited", TeslaMateAPICarsMapVisitedV2)
	v1.GET("/cars/:CarID/insights", TeslaMateAPICarsInsightsV2)
	v1.GET("/cars/:CarID/insights/events", TeslaMateAPICarsInsightEventsV2)
	v1.GET("/cars/:CarID/analytics/activity", TeslaMateAPICarsAnalyticsActivityV2)
	v1.GET("/cars/:CarID/analytics/regeneration", TeslaMateAPICarsAnalyticsRegenerationV2)
}

func registerLegacyRedirects(r *gin.Engine, basePathV1 string) {
	redirect := func(pattern string) {
		r.GET(pattern, func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePathV1+c.Request.RequestURI) })
	}
	redirect("/cars")
	redirect("/cars/:CarID")
	redirect("/cars/:CarID/battery-health")
	redirect("/cars/:CarID/charges")
	redirect("/cars/:CarID/charges/current")
	redirect("/cars/:CarID/charges/:ChargeID")
	redirect("/cars/:CarID/command")
	r.POST("/cars/:CarID/command/:Command", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePathV1+c.Request.RequestURI) })
	redirect("/cars/:CarID/drives")
	redirect("/cars/:CarID/drives/:DriveID")
	redirect("/cars/:CarID/logging")
	r.PUT("/cars/:CarID/logging/:Command", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePathV1+c.Request.RequestURI) })
	redirect("/cars/:CarID/status")
	redirect("/cars/:CarID/updates")
	r.POST("/cars/:CarID/wake_up", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePathV1+c.Request.RequestURI) })
	redirect("/globalsettings")
}
