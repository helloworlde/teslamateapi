package main

// @Summary API root
// @Tags System
// @Produce json
// @Success 200 {object} SwaggerMessageResponse
// @Router / [get]
func swaggerAPIRoot() {}

// @Summary API v1 root
// @Tags System
// @Produce json
// @Success 200 {object} SwaggerMessageResponse
// @Router /v1 [get]
func swaggerAPIV1Root() {}

// @Summary Ping
// @Tags System
// @Produce json
// @Success 200 {object} SwaggerMessageResponse
// @Router /ping [get]
func swaggerPing() {}

// @Summary List cars
// @Tags Compatible API
// @Produce json
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars [get]
func swaggerCars() {}

// @Summary Get car
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID} [get]
func swaggerCar() {}

// @Summary Battery health
// @Description Original compatible battery-health endpoint. Use `/v1/cars/{CarID}/charts/battery/health` for chart-friendly series.
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/battery-health [get]
func swaggerBatteryHealth() {}

// @Summary List charges
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Supports RFC3339, offset values, decoded-space offsets, local datetime, and date-only values"
// @Param endDate query string false "Supports RFC3339, offset values, decoded-space offsets, local datetime, and date-only values"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charges [get]
func swaggerCharges() {}

// @Summary Current charge
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charges/current [get]
func swaggerCurrentCharge() {}

// @Summary Charge details
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param ChargeID path int true "Charge ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/charges/{ChargeID} [get]
func swaggerChargeDetails() {}

// @Summary List command options
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/command [get]
func swaggerCommandCatalog() {}

// @Summary Execute command
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param Command path string true "Command name"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/command/{Command} [post]
func swaggerExecuteCommand() {}

// @Summary List drives
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Supports RFC3339, offset values, decoded-space offsets, local datetime, and date-only values"
// @Param endDate query string false "Supports RFC3339, offset values, decoded-space offsets, local datetime, and date-only values"
// @Param minDistance query number false "Minimum drive distance"
// @Param maxDistance query number false "Maximum drive distance"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/drives [get]
func swaggerDrives() {}

// @Summary Drive details
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param DriveID path int true "Drive ID"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/drives/{DriveID} [get]
func swaggerDriveDetails() {}

// @Summary Get logging status
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/logging [get]
func swaggerLoggingGet() {}

// @Summary Update logging status
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param Command path string true "Logging command"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/logging/{Command} [put]
func swaggerLoggingPut() {}

// @Summary Current vehicle status
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/status [get]
func swaggerStatus() {}

// @Summary List updates
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/updates [get]
func swaggerUpdates() {}

// @Summary Wake up vehicle
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/cars/{CarID}/wake_up [post]
func swaggerWakeUp() {}

// @Summary Global settings
// @Tags Compatible API
// @Produce json
// @Success 200 {object} SwaggerDataResponse
// @Failure 200 {object} SwaggerErrorResponse
// @Router /v1/globalsettings [get]
func swaggerGlobalSettings() {}

// @Summary Summary overview
// @Description New summary endpoint for third-party apps. Returns drive, charge, parking, efficiency, cost, mileage, state snapshot, and vampire-drain availability in one payload.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "RFC3339, offset values, decoded-space offsets, local datetime, or date-only. Encode + as %2B in URLs when possible."
// @Param endDate query string false "RFC3339, offset values, decoded-space offsets, local datetime, or date-only. Date-only endDate is expanded to local end-of-day."
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/summary [get]
func swaggerSummaryV2() {}

// @Summary Statistics dashboard summary
// @Description TeslaMate Statistics dashboard aligned aggregate response.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/statistics [get]
func swaggerStatisticsV2() {}

// @Summary Overview charts
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start; defaults to the last 30 days"
// @Param endDate query string false "Date range end; defaults to now"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/overview [get]
func swaggerChartsOverview() {}

// @Summary Drive distance chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param bucket query string false "day|week|month|year"
// @Param startDate query string false "Date range start; defaults to the last 365 days"
// @Param endDate query string false "Date range end; defaults to now"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/drives/distance [get]
func swaggerDriveDistanceChart() {}

// @Summary Drive energy chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param bucket query string false "day|week|month|year"
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/drives/energy [get]
func swaggerDriveEnergyChart() {}

// @Summary Drive efficiency chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param bucket query string false "day|week|month|year"
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/drives/efficiency [get]
func swaggerDriveEfficiencyChart() {}

// @Summary Drive speed chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param bucket query string false "day|week|month|year"
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/drives/speed [get]
func swaggerDriveSpeedChart() {}

// @Summary Drive temperature chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param bucket query string false "day|week|month|year"
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/drives/temperature [get]
func swaggerDriveTemperatureChart() {}

// @Summary Charge energy chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param bucket query string false "day|week|month|year"
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/energy [get]
func swaggerChargeEnergyChart() {}

// @Summary Charge cost chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param bucket query string false "day|week|month|year"
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/cost [get]
func swaggerChargeCostChart() {}

// @Summary Charge efficiency chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param bucket query string false "day|week|month|year"
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/efficiency [get]
func swaggerChargeEfficiencyChart() {}

// @Summary Charge power chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param bucket query string false "day|week|month|year"
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/power [get]
func swaggerChargePowerChart() {}

// @Summary Charge location chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Param show query int false "Location bucket count"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/location [get]
func swaggerChargeLocationChart() {}

// @Summary Charge SOC distribution chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/charges/soc [get]
func swaggerChargeSOCChart() {}

// @Summary Battery range chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/battery/range [get]
func swaggerBatteryRangeChart() {}

// @Summary Battery health chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/battery/health [get]
func swaggerBatteryHealthChart() {}

// @Summary State duration chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/states/duration [get]
func swaggerStateDurationChart() {}

// @Summary Vampire drain chart
// @Description Returns an explicit empty structure when TeslaMate schema does not allow a reliable vampire-drain calculation.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/vampire-drain [get]
func swaggerVampireDrainChart() {}

// @Summary Mileage chart
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIChartResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charts/mileage [get]
func swaggerMileageChart() {}

// @Summary Drive details context
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param DriveID path int true "Drive ID"
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/drives/{DriveID}/details [get]
func swaggerDriveDetailsV2() {}

// @Summary Charge details context
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param ChargeID path int true "Charge ID"
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/charges/{ChargeID}/details [get]
func swaggerChargeDetailsV2() {}

// @Summary Unified timeline
// @Description Returns drives, charges, states, and updates in one timeline ordered by time.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Param sort query string false "startDate|type"
// @Param order query string false "asc|desc"
// @Success 200 {object} APIListResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/timeline [get]
func swaggerTimelineV2() {}

// @Summary Drive calendar
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param year query int false "Calendar year"
// @Param month query int false "Calendar month"
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/calendar/drives [get]
func swaggerDriveCalendarV2() {}

// @Summary Charge calendar
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param year query int false "Calendar year"
// @Param month query int false "Calendar month"
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/calendar/charges [get]
func swaggerChargeCalendarV2() {}

// @Summary Visited map
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start; defaults to the last 90 days"
// @Param endDate query string false "Date range end; defaults to now"
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/map/visited [get]
func swaggerVisitedMapV2() {}

// @Summary Insights summary
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/insights [get]
func swaggerInsightsV2() {}

// @Summary Insight events
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Param types query string false "Comma-separated insight types"
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} APIListResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/insights/events [get]
func swaggerInsightEventsV2() {}

// @Summary Activity analytics
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/analytics/activity [get]
func swaggerActivityAnalyticsV2() {}

// @Summary Regeneration analytics
// @Description Returns estimated regeneration metrics. Meta flags when values are estimated from available drive/position data.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "Date range start"
// @Param endDate query string false "Date range end"
// @Success 200 {object} APIObjectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /v1/cars/{CarID}/analytics/regeneration [get]
func swaggerRegenerationAnalyticsV2() {}
