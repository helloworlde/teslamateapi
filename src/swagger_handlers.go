package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TeslaMateAPIAPIRootV1 godoc
// @Summary API root
// @Tags System
// @Produce json
// @Success 200 {object} SwaggerMessageResponse
// @Router / [get]
func TeslaMateAPIAPIRootV1(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": "/api"})
}

// TeslaMateAPIVersionRootV1 godoc
// @Summary API v1 root
// @Tags System
// @Produce json
// @Success 200 {object} SwaggerMessageResponse
// @Router /v1 [get]
func TeslaMateAPIVersionRootV1(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi v1 running..", "path": "/api/v1"})
}

// TeslaMateAPIPingV1 godoc
// @Summary Ping
// @Tags System
// @Produce json
// @Success 200 {object} SwaggerMessageResponse
// @Router /ping [get]
func TeslaMateAPIPingV1(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

// TeslaMateAPICarsListV1 godoc
// @Summary List cars
// @Tags Cars
// @Produce json
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars [get]
func TeslaMateAPICarsListV1(c *gin.Context) {
	TeslaMateAPICarsV1(c)
}

// TeslaMateAPICarByIDV1 godoc
// @Summary Get car
// @Tags Cars
// @Produce json
// @Param CarID path int true "Car ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID} [get]
func TeslaMateAPICarByIDV1(c *gin.Context) {
	TeslaMateAPICarsV1(c)
}

// TeslaMateAPICarsBatteryHealthDocV1 godoc
// @Summary Battery health
// @Tags Cars
// @Produce json
// @Param CarID path int true "Car ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/battery-health [get]
func TeslaMateAPICarsBatteryHealthDocV1(c *gin.Context) {
	TeslaMateAPICarsBatteryHealthV1(c)
}

// TeslaMateAPICarsChargesDocV1 godoc
// @Summary List charges
// @Tags Charges
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charges [get]
func TeslaMateAPICarsChargesDocV1(c *gin.Context) {
	TeslaMateAPICarsChargesV1(c)
}

// TeslaMateAPICarsCurrentChargeDocV1 godoc
// @Summary Current charge
// @Tags Charges
// @Produce json
// @Param CarID path int true "Car ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charges/current [get]
func TeslaMateAPICarsCurrentChargeDocV1(c *gin.Context) {
	TeslaMateAPICarsChargesCurrentV1(c)
}

// TeslaMateAPICarsChargeDetailDocV1 godoc
// @Summary Charge details
// @Tags Charges
// @Produce json
// @Param CarID path int true "Car ID"
// @Param ChargeID path int true "Charge ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charges/{ChargeID} [get]
func TeslaMateAPICarsChargeDetailDocV1(c *gin.Context) {
	TeslaMateAPICarsChargesDetailsV1(c)
}

// TeslaMateAPICarsChargeIntervalsDocV1 godoc
// @Summary Charge intervals
// @Tags Charges
// @Produce json
// @Param CarID path int true "Car ID"
// @Param ChargeID path int true "Charge ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charges/{ChargeID}/interval [get]
func TeslaMateAPICarsChargeIntervalsDocV1(c *gin.Context) {
	TeslaMateAPICarsChargeIntervalV1(c)
}

// TeslaMateAPICarsCommandOptionsDocV1 godoc
// @Summary List command options
// @Tags Commands
// @Produce json
// @Security BearerAuth
// @Param CarID path int true "Car ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/command [get]
func TeslaMateAPICarsCommandOptionsDocV1(c *gin.Context) {
	TeslaMateAPICarsCommandV1(c)
}

// TeslaMateAPICarsCommandExecuteDocV1 godoc
// @Summary Execute command
// @Tags Commands
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param CarID path int true "Car ID"
// @Param Command path string true "Command name"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/command/{Command} [post]
func TeslaMateAPICarsCommandExecuteDocV1(c *gin.Context) {
	TeslaMateAPICarsCommandV1(c)
}

// TeslaMateAPICarsDrivesDocV1 godoc
// @Summary List drives
// @Tags Drives
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param minDistance query number false "Minimum drive distance"
// @Param maxDistance query number false "Maximum drive distance"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/drives [get]
func TeslaMateAPICarsDrivesDocV1(c *gin.Context) {
	TeslaMateAPICarsDrivesV1(c)
}

// TeslaMateAPICarsDriveDetailDocV1 godoc
// @Summary Drive details
// @Tags Drives
// @Produce json
// @Param CarID path int true "Car ID"
// @Param DriveID path int true "Drive ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/drives/{DriveID} [get]
func TeslaMateAPICarsDriveDetailDocV1(c *gin.Context) {
	TeslaMateAPICarsDrivesDetailsV1(c)
}

// TeslaMateAPICarsLoggingGetDocV1 godoc
// @Summary Get logging status
// @Tags Logging
// @Produce json
// @Security BearerAuth
// @Param CarID path int true "Car ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/logging [get]
func TeslaMateAPICarsLoggingGetDocV1(c *gin.Context) {
	TeslaMateAPICarsLoggingV1(c)
}

// TeslaMateAPICarsLoggingPutDocV1 godoc
// @Summary Update logging status
// @Tags Logging
// @Produce json
// @Security BearerAuth
// @Param CarID path int true "Car ID"
// @Param Command path string true "Logging command"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/logging/{Command} [put]
func TeslaMateAPICarsLoggingPutDocV1(c *gin.Context) {
	TeslaMateAPICarsLoggingV1(c)
}

// TeslaMateAPICarsStatusDocV1 godoc
// @Summary Current vehicle status
// @Tags Status
// @Produce json
// @Param CarID path int true "Car ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/status [get]
func TeslaMateAPICarsStatusDocV1(c *gin.Context) {
	TeslaMateAPICarsStatusRouteV1(c)
}

// TeslaMateAPICarsUpdatesDocV1 godoc
// @Summary List updates
// @Tags Updates
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/updates [get]
func TeslaMateAPICarsUpdatesDocV1(c *gin.Context) {
	TeslaMateAPICarsUpdatesV1(c)
}

// TeslaMateAPICarsWakeUpDocV1 godoc
// @Summary Wake up vehicle
// @Tags Commands
// @Produce json
// @Security BearerAuth
// @Param CarID path int true "Car ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/wake_up [post]
func TeslaMateAPICarsWakeUpDocV1(c *gin.Context) {
	TeslaMateAPICarsCommandV1(c)
}

// TeslaMateAPIGlobalsettingsDocV1 godoc
// @Summary Global settings
// @Tags Settings
// @Produce json
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/globalsettings [get]
func TeslaMateAPIGlobalsettingsDocV1(c *gin.Context) {
	TeslaMateAPIGlobalsettingsV1(c)
}

// TeslaMateAPICarsSummariesDocV1 godoc
// @Summary Combined summaries
// @Tags Summaries
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param include query string false "all, overview, lifetime, drives, charges, parking, analysis, statistics, states, series"
// @Param seriesLimit query int false "Series item limit"
// @Param seriesMonths query int false "Series month bucket count"
// @Param locationLimit query int false "Location bucket limit"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/summaries [get]
func TeslaMateAPICarsSummariesDocV1(c *gin.Context) {
	TeslaMateAPICarsSummaryV1(c)
}

// TeslaMateAPICarsOverviewDocV1 godoc
// @Summary Overview summary
// @Tags Summaries
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/summaries/overview [get]
func TeslaMateAPICarsOverviewDocV1(c *gin.Context) {
	TeslaMateAPICarsOverviewV1(c)
}

// TeslaMateAPICarsLifetimeSummaryDocV1 godoc
// @Summary Lifetime summary
// @Tags Summaries
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/summaries/lifetime [get]
func TeslaMateAPICarsLifetimeSummaryDocV1(c *gin.Context) {
	TeslaMateAPICarsLifetimeSummaryV1(c)
}

// TeslaMateAPICarsDriveSummaryDocV1 godoc
// @Summary Drive summary
// @Tags Summaries
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/summaries/drives [get]
func TeslaMateAPICarsDriveSummaryDocV1(c *gin.Context) {
	TeslaMateAPICarsDriveSummaryV1(c)
}

// TeslaMateAPICarsChargeSummaryDocV1 godoc
// @Summary Charge summary
// @Tags Summaries
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/summaries/charges [get]
func TeslaMateAPICarsChargeSummaryDocV1(c *gin.Context) {
	TeslaMateAPICarsChargeSummaryV1(c)
}

// TeslaMateAPICarsParkingSummaryDocV1 godoc
// @Summary Parking summary
// @Tags Summaries
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/summaries/parking [get]
func TeslaMateAPICarsParkingSummaryDocV1(c *gin.Context) {
	TeslaMateAPICarsParkingSummaryV1(c)
}

// TeslaMateAPICarsStatisticsSummaryDocV1 godoc
// @Summary Statistics summary
// @Tags Summaries
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/summaries/statistics [get]
func TeslaMateAPICarsStatisticsSummaryDocV1(c *gin.Context) {
	TeslaMateAPICarsStatisticsSummaryV1(c)
}

// TeslaMateAPICarsStateSummaryDocV1 godoc
// @Summary State activity summary
// @Tags Summaries
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/summaries/state-activity [get]
func TeslaMateAPICarsStateSummaryDocV1(c *gin.Context) {
	TeslaMateAPICarsStateSummaryV1(c)
}

// TeslaMateAPICarsParkingSessionsDocV1 godoc
// @Summary Parking sessions
// @Tags Parking
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param states query string false "online,offline,asleep"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/parking-sessions [get]
func TeslaMateAPICarsParkingSessionsDocV1(c *gin.Context) {
	TeslaMateAPICarsParkingV1(c)
}

// TeslaMateAPICarsActivityAnalyticsDocV1 godoc
// @Summary Activity analytics
// @Tags Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/analytics/activity [get]
func TeslaMateAPICarsActivityAnalyticsDocV1(c *gin.Context) {
	TeslaMateAPICarsAnalyticsV1(c)
}

// TeslaMateAPICarsRegenerationDocV1 godoc
// @Summary Regeneration analytics
// @Tags Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/analytics/regeneration [get]
func TeslaMateAPICarsRegenerationDocV1(c *gin.Context) {
	TeslaMateAPICarsRegenerationInsightsV1(c)
}

// TeslaMateAPICarsActivityTimelineDocV1 godoc
// @Summary Activity timeline
// @Tags Activity
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/activity-timeline [get]
func TeslaMateAPICarsActivityTimelineDocV1(c *gin.Context) {
	TeslaMateAPICarsStateTimelineV1(c)
}

// TeslaMateAPICarsDriveDashboardsDocV1 godoc
// @Summary Drive dashboards
// @Tags Dashboards
// @Produce json
// @Param CarID path int true "Car ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/dashboards/drives [get]
func TeslaMateAPICarsDriveDashboardsDocV1(c *gin.Context) {
	TeslaMateAPICarsDriveDashboardsV1(c)
}

// TeslaMateAPICarsChargeDashboardsDocV1 godoc
// @Summary Charge dashboards
// @Tags Dashboards
// @Produce json
// @Param CarID path int true "Car ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/dashboards/charges [get]
func TeslaMateAPICarsChargeDashboardsDocV1(c *gin.Context) {
	TeslaMateAPICarsChargeDashboardsV1(c)
}

// TeslaMateAPICarsInsightsDocV1 godoc
// @Summary Insight summary
// @Tags Insights
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/insights [get]
func TeslaMateAPICarsInsightsDocV1(c *gin.Context) {
	TeslaMateAPICarsInsightSummaryV1(c)
}

// TeslaMateAPICarsInsightEventsDocV1 godoc
// @Summary Insight events
// @Tags Insights
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param types query string false "harsh_brake,charge_power_drop,sleep_interruption"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/insights/events [get]
func TeslaMateAPICarsInsightEventsDocV1(c *gin.Context) {
	TeslaMateAPICarsInsightEventsV1(c)
}

// TeslaMateAPICarsDriveCalendarDocV1 godoc
// @Summary Drive calendar
// @Tags Calendar
// @Produce json
// @Param CarID path int true "Car ID"
// @Param year query int false "Calendar year"
// @Param month query int false "Calendar month"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/calendars/drives [get]
func TeslaMateAPICarsDriveCalendarDocV1(c *gin.Context) {
	TeslaMateAPICarsDriveCalendarV1(c)
}

// TeslaMateAPICarsChartEfficiencyDocV1 godoc
// @Summary Efficiency chart
// @Tags Charts
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param limit query int false "Point limit"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charts/efficiency [get]
func TeslaMateAPICarsChartEfficiencyDocV1(c *gin.Context) {
	TeslaMateAPICarsDashboardEfficiencySeriesV1(c)
}

// TeslaMateAPICarsChartDriveMonthlyDistanceDocV1 godoc
// @Summary Drive monthly distance chart
// @Tags Charts
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param months query int false "Month bucket count"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charts/drives/monthly-distance [get]
func TeslaMateAPICarsChartDriveMonthlyDistanceDocV1(c *gin.Context) {
	TeslaMateAPICarsDashboardMonthlyDistanceV1(c)
}

// TeslaMateAPICarsChartDriveWeekdayDocV1 godoc
// @Summary Drive weekday chart
// @Tags Charts
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charts/drives/weekday-distance [get]
func TeslaMateAPICarsChartDriveWeekdayDocV1(c *gin.Context) {
	TeslaMateAPICarsChartDriveWeekdayV1(c)
}

// TeslaMateAPICarsChartDriveHourlyDocV1 godoc
// @Summary Drive hourly chart
// @Tags Charts
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charts/drives/hourly-starts [get]
func TeslaMateAPICarsChartDriveHourlyDocV1(c *gin.Context) {
	TeslaMateAPICarsChartDriveHourlyV1(c)
}

// TeslaMateAPICarsChartChargeMonthlyEnergyDocV1 godoc
// @Summary Charge monthly energy chart
// @Tags Charts
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param months query int false "Month bucket count"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/monthly-energy [get]
func TeslaMateAPICarsChartChargeMonthlyEnergyDocV1(c *gin.Context) {
	TeslaMateAPICarsDashboardMonthlyChargeEnergyV1(c)
}

// TeslaMateAPICarsChartChargeLocationDocV1 godoc
// @Summary Charge location chart
// @Tags Charts
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Param limit query int false "Location bucket limit"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/location-energy [get]
func TeslaMateAPICarsChartChargeLocationDocV1(c *gin.Context) {
	TeslaMateAPICarsDashboardChargeLocationsV1(c)
}

// TeslaMateAPICarsChartChargeWeekdayDocV1 godoc
// @Summary Charge weekday chart
// @Tags Charts
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/weekday-energy [get]
func TeslaMateAPICarsChartChargeWeekdayDocV1(c *gin.Context) {
	TeslaMateAPICarsChartChargeWeekdayV1(c)
}

// TeslaMateAPICarsChartChargeHourlyDocV1 godoc
// @Summary Charge hourly chart
// @Tags Charts
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/hourly-starts [get]
func TeslaMateAPICarsChartChargeHourlyDocV1(c *gin.Context) {
	TeslaMateAPICarsChartChargeHourlyV1(c)
}

// TeslaMateAPICarsChartActivityDurationDocV1 godoc
// @Summary Activity duration chart
// @Tags Charts
// @Produce json
// @Param CarID path int true "Car ID"
// @Param startDate query string false "RFC3339 start date"
// @Param endDate query string false "RFC3339 end date"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charts/activity/duration [get]
func TeslaMateAPICarsChartActivityDurationDocV1(c *gin.Context) {
	TeslaMateAPICarsChartStateDurationV1(c)
}
