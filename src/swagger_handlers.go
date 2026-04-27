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
// @Description Original compatible battery-health endpoint. Use `/v1/cars/{CarID}/charts/battery/health` for chart-friendly series.
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
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} EnabledCommandsV1Envelope
// @Router /v1/cars/{CarID}/command [get]
func swaggerCommandCatalog() {}

// @Summary Execute command
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
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Success 200 {object} EnabledCommandsV1Envelope
// @Router /v1/cars/{CarID}/logging [get]
func swaggerLoggingGet() {}

// @Summary Update logging status
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

// @Summary Dashboard
// @Description Aggregated dashboard payload for app home. Returns statistics, calendar summary, optional series/distribution placeholders, and warnings in one request.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param period query string false "year|month|week|custom"
// @Param date query string false "YYYY-MM-DD or RFC3339"
// @Param startDate query string false "custom range start"
// @Param endDate query string false "custom range end"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} v1ObjectEnvelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/dashboard [get]
func swaggerDashboardV2() {}

// @Summary Calendar
// @Description Unified calendar endpoint for drives/charges metrics. Supports day, week, month buckets and returns summary + item list.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string true "range start"
// @Param endDate query string true "range end"
// @Param bucket query string false "day|week|month"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} v1ObjectEnvelope
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
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} v1ObjectEnvelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/statistics [get]
func swaggerStatisticsV2() {}

// @Summary Series
// @Description Unified time series endpoint with scope-aware metrics and chart metadata. Unsupported metric/scope pairs are skipped with warnings.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param scope query string false "drives|charges|battery|states|overview"
// @Param metrics query string false "comma separated metrics: distance,efficiency,speed,energy,cost,power,soc,range,regeneration,vampire_drain"
// @Param bucket query string false "raw|hour|day|week|month|year"
// @Param startDate query string true "range start"
// @Param endDate query string true "range end"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} v1ObjectEnvelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/series [get]
func swaggerSeriesV2() {}

// @Summary Distributions
// @Description Distribution buckets for drive and charge behavior. Supports hour, duration, distance, speed, energy and power distributions.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param metrics query string false "comma separated metrics: drive_start_hour,drive_duration,drive_distance,drive_speed,charge_start_hour,charge_duration,charge_energy,charge_power"
// @Param startDate query string true "range start"
// @Param endDate query string true "range end"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} v1ObjectEnvelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/distributions [get]
func swaggerDistributionsV2() {}

// @Summary Insights
// @Description Baseline-comparison insights generated from current range versus previous equivalent range, including efficiency, cost, charging, battery and anomaly signals.
// @Tags Extended API
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param startDate query string true "range start"
// @Param endDate query string true "range end"
// @Param types query string false "comma separated types: efficiency,cost,charging,driving,battery,anomaly"
// @Param limit query int false "insight count limit, 1-100, default 20"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} v1ObjectEnvelope
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
// @Param startDate query string true "range start"
// @Param endDate query string true "range end"
// @Param limit query int false "page size"
// @Param offset query int false "offset"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} v1ListEnvelope
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
// @Param startDate query string true "range start"
// @Param endDate query string true "range end"
// @Param timezone query string false "IANA timezone"
// @Success 200 {object} v1ObjectEnvelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/map/visited [get]
func swaggerVisitedMapV2() {}
