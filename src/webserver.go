package main

import (
	"database/sql"
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
	"github.com/tobiasehlert/teslamateapi/src/internal/docsui"
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
	if commandRoutesEnabled() {
		initAuthToken()
	}
	// initialize allowList stored for /command section
	if commandRoutesEnabled() {
		initCommandAllowList()
	}

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

	if commandRoutesEnabled() && getEnvAsBool("API_TOKEN_DISABLE", false) {
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
			docsui.RegisterRoutes(v1, BasePathV1)
			registerCompatibleV1Routes(v1)
			registerExtendedV1Routes(v1)
		}

		// /api/ping endpoint
		api.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) })

		// health endpoints for kubernetes
		api.GET("/healthz", healthz)
		api.GET("/readyz", readyz)
	}

	registerLegacyRedirects(r, BasePathV1)

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
	log.Println("[error] " + s1 + " - (" + sanitizedRequestURI(c) + "). " + s2 + "; " + s3)
	c.JSON(http.StatusOK, gin.H{"error": s2})
}

func TeslaMateAPIHandleOtherResponse(c *gin.Context, httpCode int, s string, j interface{}) {
	if httpCode >= http.StatusInternalServerError {
		log.Println("[error] " + s + " - (" + sanitizedRequestURI(c) + ") returned status " + strconv.Itoa(httpCode) + ".")
	} else if httpCode >= http.StatusBadRequest {
		log.Println("[warning] " + s + " - (" + sanitizedRequestURI(c) + ") returned status " + strconv.Itoa(httpCode) + ".")
	} else {
		log.Println("[info] " + s + " - (" + sanitizedRequestURI(c) + ") executed successfully.")
	}
	c.JSON(httpCode, j)
}

func TeslaMateAPIHandleSuccessResponse(c *gin.Context, s string, j interface{}) {
	if gin.IsDebugging() {
		log.Println("[debug] " + s + " - (" + sanitizedRequestURI(c) + ") returned a successful response.")
	}

	log.Println("[info] " + s + " - (" + sanitizedRequestURI(c) + ") executed successfully.")
	c.JSON(http.StatusOK, j)
}

func getTimeInTimeZone(datestring string) string {
	datestring = strings.TrimSpace(datestring)
	if datestring == "" {
		return ""
	}

	// Parse RFC3339/db timestamps first; fallback to API parser.
	t, err := time.Parse(dbTimestampFormat, datestring)
	if err != nil {
		t, err = parseAPITime(datestring, time.UTC)
		if err != nil {
			if gin.IsDebugging() {
				log.Println("[warning] getTimeInTimeZone - unable to parse", datestring, "returning raw value")
			}
			return datestring
		}
	}

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
	t, err := parseAPITime(datestring, appUsersTimezone)
	if err != nil {
		return "", err
	}
	return t.UTC().Format(dbTimestampFormat), nil
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
// @Success 200 {object} APISystemMessageResponse
// @Router /healthz [get]
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": http.StatusText(http.StatusOK)})
}

// readyz godoc
// @Summary Readiness check
// @Tags System
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Failure 503 {object} APISystemErrorBody
// @Router /readyz [get]
func readyz(c *gin.Context) {
	if isReady == nil || !isReady.Load().(bool) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": http.StatusText(http.StatusServiceUnavailable)})
		return
	}
	TeslaMateAPIHandleSuccessResponse(c, "webserver", gin.H{"status": http.StatusText(http.StatusOK)})
}
