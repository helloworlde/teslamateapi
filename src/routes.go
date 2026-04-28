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
	v1.GET("/cars/:CarID/summary", TeslaMateAPICarsSummaryV2)
	v1.GET("/cars/:CarID/dashboard", TeslaMateAPICarsDashboardV2)
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
	redirect("/cars/:CarID/drives")
	redirect("/cars/:CarID/drives/:DriveID")
	redirect("/cars/:CarID/status")
	redirect("/cars/:CarID/updates")
	redirect("/globalsettings")

	if commandRoutesEnabled() {
		redirect("/cars/:CarID/command")
		r.POST("/cars/:CarID/command/:Command", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePathV1+c.Request.RequestURI) })
		redirect("/cars/:CarID/logging")
		r.PUT("/cars/:CarID/logging/:Command", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePathV1+c.Request.RequestURI) })
		r.POST("/cars/:CarID/wake_up", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, basePathV1+c.Request.RequestURI) })
	}
}
