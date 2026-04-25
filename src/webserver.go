package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	docs "github.com/tobiasehlert/teslamateapi/src/docs"
)

const (
	headerAPIVersion  = "API-Version"
	dbTimestampFormat = "2006-01-02T15:04:05Z" // format used in postgres for dates
)

var (
	// application readyz endpoint value for k8s
	isReady *atomic.Value

	// setting TeslaMateApi parameters
	apiVersion = "unspecified"

	// defining db var
	db *sql.DB

	// app-settings
	appUsersTimezone *time.Location
)

// main function
func main() {
	// setup of readiness endpoint code
	isReady = &atomic.Value{}
	isReady.Store(false)

	// setting log parameters
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	// setting application to ReleaseMode if DEBUG_MODE is false
	if !getEnvAsBool("DEBUG_MODE", false) {
		// setting GIN_MODE to ReleaseMode
		gin.SetMode(gin.ReleaseMode)
		log.Printf("[info] TeslaMateApi running in release mode.")
	} else {
		// setting GIN_MODE to DebugMode
		gin.SetMode(gin.DebugMode)
		log.Printf("[info] TeslaMateApi running in debug mode.")
	}

	// getting app-settings from environment
	appUsersTimezone, _ = time.LoadLocation(getEnv("TZ", "Europe/Berlin"))
	if gin.IsDebugging() {
		log.Println("[debug] TeslaMateApi appUsersTimezone:", appUsersTimezone)
	}

	// init of API with connection to database
	initDBconnection()
	defer db.Close()

	// run initAuthToken to validate environment vars
	initAuthToken()
	// initialize allowList stored for /command section
	initCommandAllowList()

	// Connect to the MQTT broker
	statusCache, err := startMQTT()
	mqttStatusCache = statusCache
	if getEnvAsBool("DISABLE_MQTT", false) {
		log.Printf("[info] TeslaMateApi MQTT connection not established.")
	} else {
		if err != nil {
			log.Fatalf("[error] TeslaMateApi MQTT connection failed: %s", err)
		}
	}

	if getEnvAsBool("API_TOKEN_DISABLE", false) {
		log.Println("[warning] validateAuthToken - header authorization bearer token disabled. Authorization: Bearer token will not be required for commands.")
	}

	if teslaApiHost := getEnv("TESLA_API_HOST", ""); teslaApiHost != "" {
		log.Printf("[info] TESLA_API_HOST is set: %s", teslaApiHost)
	}

	// kicking off Gin in value r
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/api"

	// gin middleware to enable GZIP support
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	r.Use(func(c *gin.Context) {
		c.Header(headerAPIVersion, apiVersion)
		c.Next()
	})

	// set 404 not found page
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	// disable proxy feature of gin
	_ = r.SetTrustedProxies(nil)

	// root endpoint telling API is running
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": r.BasePath()})
	})

	// TeslaMateApi /api endpoints
	api := r.Group("/api")
	BasePathV1 := api.BasePath() + "/v1"
	{
		// TeslaMateApi /api root
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": api.BasePath()})
		})

		// TeslaMateApi /api/v1 endpoints
		v1 := api.Group("/v1")
		{
			// TeslaMateApi /api/v1 root
			v1.GET("/", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi v1 running..", "path": v1.BasePath()})
			})
			v1.GET("/summaries/options", TeslaMateAPISummaryOptionsV1)
			v1.GET("/docs", serveScalarAPIReference)
			v1.GET("/docs/openapi.json", serveOpenAPIDocumentJSON)
			v1.GET("/docs/swagger", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+"/docs/swagger/index.html") })
			v1.GET("/docs/swagger/index.html", serveScalarAPIReference)
			v1.GET("/docs/swagger/doc.json", serveSwaggerDocJSON)

			// v1 /api/v1/cars endpoints
			v1.GET("/cars", TeslaMateAPICarsV1)
			v1.GET("/cars/:CarID", TeslaMateAPICarsV1)

			// v1 /api/v1/cars/:CarID/battery-health endpoints
			v1.GET("/cars/:CarID/battery-health", TeslaMateAPICarsBatteryHealthV1)

			// v1 /api/v1/cars/:CarID/summary endpoints
			v1.GET("/cars/:CarID/parking-sessions", TeslaMateAPICarsParkingV1)
			v1.GET("/cars/:CarID/summaries", TeslaMateAPICarsSummaryV1)
			v1.GET("/cars/:CarID/summaries/overview", TeslaMateAPICarsOverviewV1)
			v1.GET("/cars/:CarID/summaries/lifetime", TeslaMateAPICarsLifetimeSummaryV1)
			v1.GET("/cars/:CarID/summaries/drives", TeslaMateAPICarsDriveSummaryV1)
			v1.GET("/cars/:CarID/summaries/charges", TeslaMateAPICarsChargeSummaryV1)
			v1.GET("/cars/:CarID/summaries/parking", TeslaMateAPICarsParkingSummaryV1)
			v1.GET("/cars/:CarID/summaries/statistics", TeslaMateAPICarsStatisticsSummaryV1)
			v1.GET("/cars/:CarID/summaries/state-activity", TeslaMateAPICarsStateSummaryV1)
			v1.GET("/cars/:CarID/analytics/activity", TeslaMateAPICarsAnalyticsV1)
			v1.GET("/cars/:CarID/analytics/regeneration", TeslaMateAPICarsRegenerationInsightsV1)
			v1.GET("/cars/:CarID/activity-timeline", TeslaMateAPICarsStateTimelineV1)
			v1.GET("/cars/:CarID/dashboards/drives", TeslaMateAPICarsDriveDashboardsV1)
			v1.GET("/cars/:CarID/dashboards/charges", TeslaMateAPICarsChargeDashboardsV1)
			v1.GET("/cars/:CarID/insights", TeslaMateAPICarsInsightSummaryV1)
			v1.GET("/cars/:CarID/insights/events", TeslaMateAPICarsInsightEventsV1)
			v1.GET("/cars/:CarID/calendars/drives", TeslaMateAPICarsDriveCalendarV1)
			v1.GET("/cars/:CarID/charts/efficiency", TeslaMateAPICarsDashboardEfficiencySeriesV1)
			v1.GET("/cars/:CarID/charts/drives/monthly-distance", TeslaMateAPICarsDashboardMonthlyDistanceV1)
			v1.GET("/cars/:CarID/charts/drives/weekday-distance", TeslaMateAPICarsChartDriveWeekdayV1)
			v1.GET("/cars/:CarID/charts/drives/hourly-starts", TeslaMateAPICarsChartDriveHourlyV1)
			v1.GET("/cars/:CarID/charts/charges/monthly-energy", TeslaMateAPICarsDashboardMonthlyChargeEnergyV1)
			v1.GET("/cars/:CarID/charts/charges/location-energy", TeslaMateAPICarsDashboardChargeLocationsV1)
			v1.GET("/cars/:CarID/charts/charges/weekday-energy", TeslaMateAPICarsChartChargeWeekdayV1)
			v1.GET("/cars/:CarID/charts/charges/hourly-starts", TeslaMateAPICarsChartChargeHourlyV1)
			v1.GET("/cars/:CarID/charts/activity/duration", TeslaMateAPICarsChartStateDurationV1)

			// v1 /api/v1/cars/:CarID/charges endpoints
			v1.GET("/cars/:CarID/charges", TeslaMateAPICarsChargesV1)
			v1.GET("/cars/:CarID/charges/current", TeslaMateAPICarsChargesCurrentV1)
			v1.GET("/cars/:CarID/charges/:ChargeID/interval", TeslaMateAPICarsChargeIntervalV1)
			v1.GET("/cars/:CarID/charges/:ChargeID", TeslaMateAPICarsChargesDetailsV1)

			// v1 /api/v1/cars/:CarID/command endpoints
			v1.GET("/cars/:CarID/command", TeslaMateAPICarsCommandV1)
			v1.POST("/cars/:CarID/command/:Command", TeslaMateAPICarsCommandV1)

			// v1 /api/v1/cars/:CarID/drives endpoints
			v1.GET("/cars/:CarID/drives", TeslaMateAPICarsDrivesV1)
			v1.GET("/cars/:CarID/drives/:DriveID", TeslaMateAPICarsDrivesDetailsV1)

			// v1 /api/v1/cars/:CarID/logging endpoints
			v1.GET("/cars/:CarID/logging", TeslaMateAPICarsLoggingV1)
			v1.PUT("/cars/:CarID/logging/:Command", TeslaMateAPICarsLoggingV1)

			// v1 /api/v1/cars/:CarID/status endpoints
			v1.GET("/cars/:CarID/status", TeslaMateAPICarsStatusRouteV1)

			// v1 /api/v1/cars/:CarID/updates endpoints
			v1.GET("/cars/:CarID/updates", TeslaMateAPICarsUpdatesV1)

			// v1 /api/v1/cars/:CarID/wake_up endpoints
			v1.POST("/cars/:CarID/wake_up", TeslaMateAPICarsCommandV1)

			// v1 /api/v1/globalsettings endpoints
			v1.GET("/globalsettings", TeslaMateAPIGlobalsettingsV1)
		}

		// /api/ping endpoint
		api.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) })

		// health endpoints for kubernetes
		api.GET("/healthz", healthz)
		api.GET("/readyz", readyz)
	}

	// TeslaMateApi endpoints (before versioning)
	r.GET("/cars", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/parking-sessions", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/summaries", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/summaries/overview", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/summaries/lifetime", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/summaries/drives", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/summaries/charges", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/summaries/parking", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/summaries/statistics", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/summaries/state-activity", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/analytics/activity", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/analytics/regeneration", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/activity-timeline", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/dashboards/drives", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/dashboards/charges", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/insights", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/insights/events", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/calendars/drives", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charts/efficiency", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charts/drives/monthly-distance", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charts/drives/weekday-distance", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charts/drives/hourly-starts", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charts/charges/monthly-energy", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charts/charges/location-energy", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charts/charges/weekday-energy", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charts/charges/hourly-starts", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charts/activity/duration", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charges", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charges/:ChargeID/interval", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charges/:ChargeID", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/drives", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/drives/:DriveID", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/status", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/updates", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/globalsettings", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })

	// build the http server
	listenAddr := getEnv("TESLAMATEAPI_LISTEN_ADDR", ":8080")
	server := &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}

	// setting readyz endpoint to true (if not using MQTT)
	if getEnvAsBool("DISABLE_MQTT", false) {
		isReady.Store(true)
	}

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// we run a go routine that will receive the shutdown input
	go func() {
		<-quit
		log.Println("[info] TeslaMateAPI received shutdown input")
		if err := server.Close(); err != nil {
			log.Fatal("[error] TeslaMateAPI server close error:", err)
		}
	}()

	log.Printf("[info] TeslaMateAPI listening on %s", listenAddr)
	// run the server
	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("[info] TeslaMateAPI server gracefully shut down")
		} else {
			log.Fatal("[error] TeslaMateAPI server closed unexpectedly")
		}
	}
}

// initDBconnection func
func initDBconnection() {
	var err error

	// read environment variables with defaults for connection string
	dbhost := getEnv("DATABASE_HOST", "database")
	dbport := getEnvAsInt("DATABASE_PORT", 5432)
	dbuser := getEnv("DATABASE_USER", "teslamate")
	dbpass := getEnv("DATABASE_PASS", "secret")
	dbname := getEnv("DATABASE_NAME", "teslamate")
	dbtimeout := (getEnvAsInt("DATABASE_TIMEOUT", 60000) / 1000)
	dbsslmode := getEnv("DATABASE_SSL", "disable")
	dbsslrootcert := getEnv("DATABASE_SSL_CA_CERT_FILE", "")

	// convert boolean-like SSL mode for backwards compatibility
	switch dbsslmode {
	case "true", "noverify":
		dbsslmode = "require"
	case "false":
		dbsslmode = "disable"
	}

	// construct connection string
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d", dbhost, dbport, dbuser, dbpass, dbname, dbsslmode, dbtimeout)

	// add SSL certificate configuration if provided
	if dbsslrootcert != "" {
		psqlInfo += " sslrootcert=" + dbsslrootcert
	}

	// open database connection
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("[error] initDBconnection - database connection error: %v", err)
	}

	// test database connection
	if err = db.Ping(); err != nil {
		log.Fatalf("[error] initDBconnection - database ping error: %v", err)
	}

	// showing database successfully connected
	if gin.IsDebugging() {
		log.Println("[debug] initDBconnection - database connection established successfully.")
	}
}

func TeslaMateAPIHandleErrorResponse(c *gin.Context, s1 string, s2 string, s3 string) {
	log.Println("[error] " + s1 + " - (" + c.Request.RequestURI + "). " + s2 + "; " + s3)
	c.JSON(http.StatusOK, gin.H{"error": s2})
}

func TeslaMateAPIHandleOtherResponse(c *gin.Context, httpCode int, s string, j interface{}) {
	// return successful response
	log.Println("[info] " + s + " - (" + c.Request.RequestURI + ") executed successfully.")
	c.JSON(httpCode, j)
}

func TeslaMateAPIHandleSuccessResponse(c *gin.Context, s string, j interface{}) {
	// print to log about request
	if gin.IsDebugging() {
		log.Println("[debug] " + s + " - (" + c.Request.RequestURI + ") returned data:")
		js, _ := json.Marshal(j)
		log.Printf("[debug] %s\n", js)
	}

	// return successful response
	log.Println("[info] " + s + " - (" + c.Request.RequestURI + ") executed successfully.")
	c.JSON(http.StatusOK, j)
}

func getTimeInTimeZone(datestring string) string {
	// parsing datestring into dbTimestampFormat
	t, _ := time.Parse(dbTimestampFormat, datestring)

	// formatting in users location in RFC3339 format
	ReturnDate := t.In(appUsersTimezone).Format(time.RFC3339)

	// logging time conversion to log
	if gin.IsDebugging() {
		log.Println("[debug] getTimeInTimeZone - UTC", t.Format(time.RFC3339), "time converted to", appUsersTimezone, "is", ReturnDate)
	}

	return ReturnDate
}

func parseDateParam(datestring string) (string, error) {
	if datestring == "" {
		return "", nil
	}

	datestring = repairDateQueryParam(datestring)

	if t, err := time.Parse(time.RFC3339, datestring); err == nil {
		return t.UTC().Format(dbTimestampFormat), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, datestring); err == nil {
		return t.UTC().Format(dbTimestampFormat), nil
	}

	localDateTime := strings.Replace(datestring, "T", " ", 1)
	if t, err := time.ParseInLocation(time.DateTime, localDateTime, appUsersTimezone); err == nil {
		return t.UTC().Format(dbTimestampFormat), nil
	}

	sanitizedInput := strings.NewReplacer("\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(datestring)
	return "", fmt.Errorf("invalid date format: %q (use RFC3339, e.g. 2026-04-02T10:55:30+08:00; encode + as %%2B in query strings)", sanitizedInput)
}

// getEnv func - read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultVal
}

// getEnvAsBool func - read an environment variable into a bool or return default value
func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

// getEnvAsInt func - read an environment variable into integer or return a default value
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

// convertStringToBool func - converts a string to boolean, returning false on failure
func convertStringToBool(data string) bool {
	value, err := strconv.ParseBool(data)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] convertStringToBool: failed to parse '%s' as boolean - returning false", data)
		}
		return false
	}
	return value
}

// convertStringToFloat func - converts a string to float64, returning 0.0 on failure
func convertStringToFloat(data string) float64 {
	value, err := strconv.ParseFloat(data, 64)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] convertStringToFloat: failed to parse '%s' as float64 - returning 0.0", data)
		}
		return 0.0
	}
	return value
}

// convertStringToInteger func - converts a string to int, returning 0 on failure
func convertStringToInteger(data string) int {
	value, err := strconv.Atoi(data)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] convertStringToInteger: failed to parse '%s' as integer - returning 0", data)
		}
		return 0
	}
	return value
}

// kilometersToMiles func
func kilometersToMiles(km float64) float64 {
	return (km * 0.62137119223733)
}

// kilometersToMilesNilSupport func
func kilometersToMilesNilSupport(km NullFloat64) NullFloat64 {
	km.Float64 = (km.Float64 * 0.62137119223733)
	return (km)
}

// milesToKilometers func
func milesToKilometers(mi float64) float64 {
	return (mi * 1.609344)
}

// kilometersToMilesInteger func
func kilometersToMilesInteger(km int) int {
	return int(float64(km) * 0.62137119223733)
}

// barToPsi func
func barToPsi(bar float64) float64 {
	return (bar * 14.503773800722)
}

// celsiusToFahrenheit func
func celsiusToFahrenheit(c float64) float64 {
	return (c*9/5 + 32)
}

// celsiusToFahrenheitNilSupport func
func celsiusToFahrenheitNilSupport(c NullFloat64) NullFloat64 {
	c.Float64 = (c.Float64*9/5 + 32)
	return (c)
}

// checkArrayContainsString func - check if string is inside stringarray
func checkArrayContainsString(s []string, e string) bool {
	return slices.Contains(s, e)
}

// healthz godoc
// @Summary Health check
// @Tags System
// @Produce json
// @Success 200 {object} SwaggerMessageResponse
// @Router /healthz [get]
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": http.StatusText(http.StatusOK)})
}

// readyz godoc
// @Summary Readiness check
// @Tags System
// @Produce json
// @Success 200 {object} SwaggerMessageResponse
// @Failure 503 {object} SwaggerErrorResponse
// @Router /readyz [get]
func readyz(c *gin.Context) {
	if isReady == nil || !isReady.Load().(bool) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": http.StatusText(http.StatusServiceUnavailable)})
		return
	}
	TeslaMateAPIHandleSuccessResponse(c, "webserver", gin.H{"status": http.StatusText(http.StatusOK)})
}
