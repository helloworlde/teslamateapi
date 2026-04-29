package main

// @Summary API root
// @Tags System
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router / [get]
func swaggerAPIRoot() {}

// @Summary API v1 root
// @Tags System
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router /v1 [get]
func swaggerAPIV1Root() {}

// @Summary Ping
// @Tags System
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router /ping [get]
func swaggerPing() {}

// @Summary List cars
// @Description Returns all cars. Nullable DB fields may appear as empty string or 0 in JSON. Legacy: some failures still return HTTP 200 with JSON body containing only an error string field.
// @Tags Compatible API
// @Produce json
// @Success 200 {object} CarsV1Envelope
// @Router /v1/cars [get]
func swaggerCars() {}

// @Summary Get car
// @Description Returns the matching car in data.cars (usually one item). Legacy: some failures still return HTTP 200 with JSON body containing only an error string field.
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} CarsV1Envelope
// @Router /v1/cars/{CarID} [get]
func swaggerCar() {}

// @Summary Battery health
// @Description Original compatible battery-health endpoint. Use `/v1/cars/{CarID}/series/battery?metrics=range,soc` for chart-friendly battery series.
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} BatteryHealthV1Envelope
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
// @Param limit query int false "Page size alias"
// @Param offset query int false "Offset alias"
// @Param sort query string false "start_date|-start_date|duration|-duration|cost|-cost|energy|-energy"
// @Param include query string false "summary,location,energy,cost"
// @Success 200 {object} ChargesListV1Envelope
// @Router /v1/cars/{CarID}/charges [get]
func swaggerCharges() {}

// @Summary Current charge
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} CurrentChargeV1Envelope
// @Router /v1/cars/{CarID}/charges/current [get]
func swaggerCurrentCharge() {}

// @Summary Charge details
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param ChargeID path int true "Charge ID"
// @Success 200 {object} ChargeDetailsV1Envelope
// @Router /v1/cars/{CarID}/charges/{ChargeID} [get]
func swaggerChargeDetails() {}

// @Summary List command options
// @Description Registered only when ENABLE_COMMANDS=true. Tesla account access and refresh tokens are never exposed in API responses.
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} EnabledCommandsV1Envelope
// @Router /v1/cars/{CarID}/command [get]
func swaggerCommandCatalog() {}

// @Summary Execute command
// @Description Registered only when ENABLE_COMMANDS=true and the specific command is allowlisted. Tesla account access and refresh tokens are never exposed in API responses.
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param Command path string true "Command name"
// @Success 200 {object} TeslaPassthroughJSONBody
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
// @Param limit query int false "Page size alias"
// @Param offset query int false "Offset alias"
// @Param sort query string false "start_date|-start_date|distance|-distance|duration|-duration|efficiency|-efficiency"
// @Param include query string false "summary,locations,energy"
// @Success 200 {object} DrivesListV1Envelope
// @Router /v1/cars/{CarID}/drives [get]
func swaggerDrives() {}

// @Summary Drive details
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param DriveID path int true "Drive ID"
// @Success 200 {object} DriveDetailsV1Envelope
// @Router /v1/cars/{CarID}/drives/{DriveID} [get]
func swaggerDriveDetails() {}

// @Summary Get logging status
// @Description Registered only when ENABLE_COMMANDS=true. Returns allowlisted logging commands only.
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} EnabledCommandsV1Envelope
// @Router /v1/cars/{CarID}/logging [get]
func swaggerLoggingGet() {}

// @Summary Update logging status
// @Description Registered only when ENABLE_COMMANDS=true and the logging command is allowlisted.
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param Command path string true "Logging command"
// @Success 200 {object} TeslaPassthroughJSONBody
// @Router /v1/cars/{CarID}/logging/{Command} [put]
func swaggerLoggingPut() {}

// @Summary Current vehicle status
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} CarStatusV1Envelope
// @Router /v1/cars/{CarID}/status [get]
func swaggerStatus() {}

// @Summary List updates
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param page query int false "Page number"
// @Param show query int false "Page size"
// @Success 200 {object} UpdatesListV1Envelope
// @Router /v1/cars/{CarID}/updates [get]
func swaggerUpdates() {}

// @Summary Wake up vehicle
// @Description Registered only when ENABLE_COMMANDS=true and COMMANDS_WAKE, COMMANDS_ALL, or COMMANDS_ALLOWLIST allows /wake_up. Tesla account access and refresh tokens are never exposed in API responses.
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} TeslaPassthroughJSONBody
// @Router /v1/cars/{CarID}/wake_up [post]
func swaggerWakeUp() {}

// @Summary Global settings
// @Tags Compatible API
// @Produce json
// @Success 200 {object} GlobalsettingsV1Envelope
// @Router /v1/globalsettings [get]
func swaggerGlobalSettings() {}

// @Summary Summary
// @Description Canonical range summary for a car. Returns stable overview, driving, charging, parking, battery, efficiency, cost, quality and state sections without legacy include-driven sparse fields.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param period query string false "year|month|week|custom, default month"
// @Param date query string false "reference date for period mode"
// @Param startDate query string false "custom range start; when present endDate is required"
// @Param endDate query string false "custom range end; when present startDate is required"
// @Success 200 {object} SummaryV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/summary [get]
func swaggerSummaryV2() {}

// @Summary Dashboard
// @Description Vehicle-level dashboard statistics for the selected range. Realtime state, chart series, distributions, insights, timeline, drives and charges are exposed by dedicated endpoints.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param period query string false "year|month|week|custom"
// @Param date query string false "YYYY-MM-DD or RFC3339"
// @Param startDate query string false "custom range start"
// @Param endDate query string false "custom range end"
// @Success 200 {object} DashboardV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/dashboard [get]
func swaggerDashboardV2() {}

// @Summary Realtime vehicle snapshot
// @Description Current vehicle snapshot derived from latest position, latest state and latest charging process.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} RealtimeV2Envelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/realtime [get]
func swaggerRealtimeV2() {}

// @Summary Calendar
// @Description Unified calendar endpoint for drives/charges metrics. Supports day, week, month buckets and returns summary + item list.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Param bucket query string false "day|week|month"
// @Success 200 {object} CalendarV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/calendar [get]
func swaggerCalendarV2() {}

// @Summary Statistics
// @Description Unified period statistics with explicit overview/drive/charge/battery sections, including efficiency, cost, energy and parking metrics.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param period query string false "year|month|week|custom"
// @Param date query string false "YYYY-MM-DD or RFC3339"
// @Param startDate query string false "custom range start"
// @Param endDate query string false "custom range end"
// @Success 200 {object} StatisticsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/statistics [get]
func swaggerStatisticsV2() {}

// @Summary Drive series
// @Description Drive time series grouped by bucket. Response merges selected metrics into one point object per timestamp, sorted newest first.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param metrics query string false "comma separated metrics: distance,efficiency,speed,max_speed,motor_power,regen_power,elevation,outside_temp,inside_temp,energy,regeneration"
// @Param bucket query string false "raw|hour|day|week|month|year"
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Success 200 {object} SeriesV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/series/drives [get]
func swaggerDriveSeriesV2() {}

// @Summary Charge series
// @Description Charge time series grouped by bucket. Response includes start_soc and end_soc and merges selected metrics into one point object per timestamp.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param metrics query string false "comma separated metrics: energy,power,cost,start_soc,end_soc"
// @Param bucket query string false "raw|hour|day|week|month|year"
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Success 200 {object} SeriesV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/series/charges [get]
func swaggerChargeSeriesV2() {}

// @Summary Battery series
// @Description Battery time series for state of charge and rated range.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param metrics query string false "comma separated metrics: soc,range"
// @Param bucket query string false "raw|hour|day|week|month|year"
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Success 200 {object} SeriesV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/series/battery [get]
func swaggerBatterySeriesV2() {}

// @Summary State series
// @Description State-derived time series for state duration and parking energy / vampire drain.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param metrics query string false "comma separated metrics: duration,vampire_drain"
// @Param bucket query string false "raw|hour|day|week|month|year"
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Success 200 {object} SeriesV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/series/states [get]
func swaggerStateSeriesV2() {}

// @Summary Drive distributions
// @Description Drive distribution buckets for start hour, weekday, distance, duration, speed and efficiency. Buckets are ordered and include zero-count gaps.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param metrics query string false "comma separated metrics: start_hour,weekday,distance,duration,speed,efficiency"
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Success 200 {object} DistributionsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/distributions/drives [get]
func swaggerDriveDistributionsV2() {}

// @Summary Charge distributions
// @Description Charge distribution buckets for start hour, weekday, energy, duration, power and cost. Buckets are ordered and include zero-count gaps.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param metrics query string false "comma separated metrics: start_hour,weekday,energy,duration,power,cost"
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Success 200 {object} DistributionsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/distributions/charges [get]
func swaggerChargeDistributionsV2() {}

// @Summary Insights
// @Description Baseline-comparison insights generated from current range versus previous equivalent range, including efficiency, cost, charging, battery and anomaly signals.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Param types query string false "comma separated types: efficiency,cost,charging,driving,battery,anomaly"
// @Param limit query int false "insight count limit, 1-100, default 20"
// @Success 200 {object} InsightsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/insights [get]
func swaggerInsightsV2() {}

// @Summary Timeline
// @Description Unified timeline feed for drive/charge/state activities with pagination.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Param limit query int false "page size"
// @Param offset query int false "offset"
// @Success 200 {object} TimelineV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/timeline [get]
func swaggerTimelineV2() {}

// @Summary Visited map
// @Description Visited geo points and map bounds for heatmap/coverage rendering.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Success 200 {object} VisitedMapV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/map/visited [get]
func swaggerVisitedMapV2() {}

// @Summary Locations
// @Description Aggregated locations payload for TeslaMate location dashboards. Combines drive start/end places and charging places with event counts, coordinates, charge energy, cost, and last-seen timestamps.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string false "range start"
// @Param endDate query string false "range end"
// @Param limit query int false "maximum locations, capped at 100"
// @Success 200 {object} LocationsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/locations [get]
func swaggerLocationsV2() {}
