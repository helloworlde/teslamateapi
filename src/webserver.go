// Package main provides the TeslaMateApi HTTP server.
//
// @title                       TeslaMateApi
// @version                     1.0
// @description                 REST API in front of TeslaMate's Postgres database, the MQTT status feed, and (optionally) Tesla owner-api / TeslaMate logging command relays.
// @BasePath                    /
// @schemes                     http https
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 API_TOKEN bearer credential. Required for every /api/* route except /api, /api/, /api/ping, /api/healthz, /api/readyz, /api/docs, /api/openapi.yaml.
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

	// getting app-settings from environment. A bad TZ name returns nil and we'd
	// later panic on time.Time.In(nil); fall back to UTC and log loudly instead.
	tzName := getEnv("TZ", "Europe/Berlin")
	loc, err := time.LoadLocation(tzName)
	if err != nil || loc == nil {
		log.Printf("[warning] TZ=%q not loadable (%v); falling back to UTC", tzName, err)
		loc = time.UTC
	}
	appUsersTimezone = loc
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

	// Commands hit Tesla's owner-api / TeslaMate's logging endpoint with the
	// car's stored access token. Exposing them without auth turns the API into
	// an open relay for anyone who can reach the port. Refuse to start in that
	// configuration instead of silently registering unauthenticated routes.
	commandsEnabled := getEnvAsBool("ENABLE_COMMANDS", false)
	if commandsEnabled {
		if getEnvAsBool("API_TOKEN_DISABLE", false) {
			log.Fatal("[error] ENABLE_COMMANDS=true requires authentication; refusing to start with API_TOKEN_DISABLE=true.")
		}
		if envToken == "" {
			log.Fatal("[error] ENABLE_COMMANDS=true requires API_TOKEN to be set; refusing to start without it.")
		}
		if len(envToken) < 32 {
			log.Fatal("[error] ENABLE_COMMANDS=true requires API_TOKEN of at least 32 characters; refusing to start.")
		}
	}

	// Connect to the MQTT broker
	statusCache, err := startMQTT()
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

	// gin middleware to enable GZIP support
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	r.Use(func(c *gin.Context) {
		c.Header(headerAPIVersion, apiVersion)
		c.Next()
	})

	// set 404 not found page
	r.NoRoute(notFoundHandler)

	// disable proxy feature of gin
	_ = r.SetTrustedProxies(nil)

	// root endpoint telling API is running
	r.GET("/", rootRoot(r))

	// TeslaMateApi /api endpoints. The group-level auth middleware gates every
	// /api/* route behind API_TOKEN. Unauthenticated probes (health/readiness,
	// docs, openapi spec, ping) and the /api root itself are whitelisted —
	// these have to be reachable from a browser or a kubelet.
	//
	// Per-handler validateAuthToken calls (in command/logging) still run after
	// this middleware; the second check is a no-op for valid tokens and keeps
	// behavior identical when API_TOKEN_DISABLE is set.
	api := r.Group("/api", apiAuthMiddleware())
	{
		// TeslaMateApi /api root
		api.GET("/", apiRoot(api))

		// TeslaMateApi /api/v1 endpoints
		v1 := api.Group("/v1")
		{
			// TeslaMateApi /api/v1 root
			v1.GET("/", apiV1Root(v1))

			// v1 /api/v1/cars endpoints
			v1.GET("/cars", TeslaMateAPICarsV1)
			v1.GET("/cars/:CarID", TeslaMateAPICarsV1)

			// v1 /api/v1/cars/:CarID/battery-health endpoints
			v1.GET("/cars/:CarID/battery-health", TeslaMateAPICarsBatteryHealthV1)

			// v1 /api/v1/cars/:CarID/charges endpoints
			v1.GET("/cars/:CarID/charges", TeslaMateAPICarsChargesV1)
			v1.GET("/cars/:CarID/charges/current", TeslaMateAPICarsChargesCurrentV1)
			v1.GET("/cars/:CarID/charges/:ChargeID", TeslaMateAPICarsChargesDetailsV1)

			// v1 /api/v1/cars/:CarID/command + /logging + /wake_up endpoints —
			// only registered when ENABLE_COMMANDS is true. Skipping registration
			// (vs. handler-level 403) means the routes literally don't exist on
			// disabled deployments: scanners get 404, not a hint that command
			// machinery is present.
			if commandsEnabled {
				v1.GET("/cars/:CarID/command", TeslaMateAPICarsCommandListV1)
				v1.GET("/cars/:CarID/commands", TeslaMateAPICarsCommandListV1)
				v1.POST("/cars/:CarID/command/:Command", TeslaMateAPICarsCommandExecV1)

				v1.GET("/cars/:CarID/logging", TeslaMateAPICarsLoggingListV1)
				v1.PUT("/cars/:CarID/logging/:Command", TeslaMateAPICarsLoggingExecV1)

				v1.POST("/cars/:CarID/wake_up", TeslaMateAPICarsCommandExecV1)
			}

			// v1 /api/v1/cars/:CarID/drives endpoints
			v1.GET("/cars/:CarID/drives", TeslaMateAPICarsDrivesV1)
			v1.GET("/cars/:CarID/drives/:DriveID", TeslaMateAPICarsDrivesDetailsV1)

			// v1 /api/v1/cars/:CarID/status endpoints
			v1.GET("/cars/:CarID/status", statusCache.TeslaMateAPICarsStatusV1)

			// v1 /api/v1/cars/:CarID/updates endpoints
			v1.GET("/cars/:CarID/updates", TeslaMateAPICarsUpdatesV1)

			// v1 /api/v1/globalsettings endpoints
			v1.GET("/globalsettings", TeslaMateAPIGlobalsettingsV1)
		}

		// TeslaMateApi /api/v2 endpoints — additive only; v1 routes above are
		// frozen for backwards compatibility. v2 supplements v1 with
		// parking sessions and lifetime/period/by-geofence aggregates.
		// (Charges-list charger-shape fields used to live here as a
		// separate endpoint; they are now folded into v1
		// /api/v1/cars/:CarID/charges directly.)
		v2 := api.Group("/v2")
		{
			// TeslaMateApi /api/v2 root
			v2.GET("/", apiV2Root(v2))

			// /api/v2/cars/:CarID/parkings — parking sessions derived from
			// gaps between adjacent drives. Has list + detail with optional
			// SOC time-series.
			v2.GET("/cars/:CarID/parkings", TeslaMateAPICarsParkingsV2)
			v2.GET("/cars/:CarID/parkings/:PrecedingDriveID", TeslaMateAPICarsParkingsDetailsV2)

			// /api/v2/cars/:CarID/stats/* — aggregates that v1 forces clients
			// to compute by walking every drive/charge.
			v2.GET("/cars/:CarID/stats/lifetime", TeslaMateAPICarsStatsLifetimeV2)
			v2.GET("/cars/:CarID/stats/summary", TeslaMateAPICarsStatsSummaryV2)
			v2.GET("/cars/:CarID/stats/by-geofence", TeslaMateAPICarsStatsByGeofenceV2)
		}

		// /api/ping endpoint
		api.GET("/ping", apiPing)

		// health endpoints for kubernetes
		api.GET("/healthz", healthz)
		api.GET("/readyz", readyz)

		// /api/docs renders the Scalar UI; /api/openapi.yaml serves the raw
		// spec embedded into the binary. Both intentionally bypass the bearer
		// auth middleware so they're reachable from a browser without a token.
		api.GET("/docs", scalarDocs)
		api.GET("/openapi.yaml", openapiYAML)
	}

	// TeslaMateApi endpoints (before versioning)
	BasePathV1 := api.BasePath() + "/v1"
	r.GET("/cars", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charges", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charges/:ChargeID", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/drives", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/drives/:DriveID", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/status", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/updates", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/globalsettings", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })

	// build the http server
	server := &http.Server{
		Addr:    ":8080", // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
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

	// run the server
	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("[info] TeslaMateAPI server gracefully shut down")
		} else {
			log.Fatal("[error] TeslaMateAPI server closed unexpectedly")
		}
	}
}

// pqQuote escapes a value for libpq's KV connection-string format. Wraps
// every value in single quotes and backslash-escapes embedded ' and \ —
// safe even for values without specials.
func pqQuote(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(v) + "'"
}

// apiAuthMiddleware enforces API_TOKEN on /api/* with a small public allow-list.
// Anything not in the allow-list goes through validateAuthToken; failures emit
// a real 401 (not the legacy 200+error envelope) so clients and load
// balancers can react correctly.
func apiAuthMiddleware() gin.HandlerFunc {
	publicSuffixes := []string{
		"/api",
		"/api/",
		"/api/ping",
		"/api/healthz",
		"/api/readyz",
		"/api/docs",
		"/api/openapi.yaml",
	}
	return func(c *gin.Context) {
		if getEnvAsBool("API_TOKEN_DISABLE", false) {
			c.Next()
			return
		}
		// Auth is opt-in. Without API_TOKEN we can't enforce anything; warn
		// at startup (initAuthToken) and let traffic through here so
		// existing deployments without a token aren't broken silently.
		if envToken == "" {
			c.Next()
			return
		}
		if slices.Contains(publicSuffixes, c.Request.URL.Path) {
			c.Next()
			return
		}
		ok, msg := validateAuthToken(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}
		c.Next()
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

	// libpq KV format (key='val') with backslash-escaping for ' and \, so a
	// password containing spaces, single quotes, or backslashes doesn't break
	// the connection string. The previous fmt.Sprintf form silently mis-parsed
	// such passwords (and could leak fragments via parser error logs).
	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		pqQuote(dbhost), dbport, pqQuote(dbuser), pqQuote(dbpass), pqQuote(dbname), pqQuote(dbsslmode), dbtimeout,
	)

	// add SSL certificate configuration if provided
	if dbsslrootcert != "" {
		psqlInfo += " sslrootcert=" + pqQuote(dbsslrootcert)
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

// v2HandleErrorResponse emits a real HTTP status code instead of the upstream-compat
// 200+{error} envelope. v1 handlers must keep the old shape; v2 handlers should use this.
func v2HandleErrorResponse(c *gin.Context, handler string, httpCode int, message string, detail string) {
	log.Println("[error] " + handler + " - (" + c.Request.RequestURI + "). " + message + "; " + detail)
	c.JSON(httpCode, gin.H{"error": message})
}

// v2RequirePositiveIntParam parses a path/query integer with strict validation:
// non-numeric or non-positive values respond 400 and return ok=false. Used by
// every v2 handler to avoid silently coercing garbage to 0 (which used to run
// the SQL with an invalid id).
func v2RequirePositiveIntParam(c *gin.Context, handler, name, raw string) (int, bool) {
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		v2HandleErrorResponse(c, handler, http.StatusBadRequest, name+" is required and must be a positive integer.", "got: "+raw)
		return 0, false
	}
	return v, true
}

// v2OptionalIntInRange parses a query integer with a default + min/max clamp.
// Empty string yields the default. Non-numeric or out-of-range values respond 400.
func v2OptionalIntInRange(c *gin.Context, handler, name, raw string, def, min, max int) (int, bool) {
	if raw == "" {
		return def, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < min || v > max {
		v2HandleErrorResponse(c, handler, http.StatusBadRequest,
			fmt.Sprintf("%s must be an integer in [%d, %d].", name, min, max), "got: "+raw)
		return 0, false
	}
	return v, true
}

func TeslaMateAPIHandleOtherResponse(c *gin.Context, httpCode int, s string, j any) {
	// return successful response
	log.Println("[info] " + s + " - (" + c.Request.RequestURI + ") executed successfully.")
	c.JSON(httpCode, j)
}

func TeslaMateAPIHandleSuccessResponse(c *gin.Context, s string, j any) {
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

	// RFC3339 formats first — includes Z or timezone offset
	if t, err := time.Parse(time.RFC3339, datestring); err == nil {
		return t.UTC().Format(dbTimestampFormat), nil
	}

	// DateTime format (2006-01-02 15:04:05) without timezone info, interpret in user's timezone
	normalizedDateString := strings.ReplaceAll(datestring, "T", " ")
	if t, err := time.ParseInLocation(time.DateTime, normalizedDateString, appUsersTimezone); err == nil {
		return t.UTC().Format(dbTimestampFormat), nil
	}

	sanitizedInput := strings.NewReplacer("\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(datestring)
	return "", fmt.Errorf("invalid date format: %s, please use RFC3339 format", sanitizedInput)
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

// healthz is a liveness probe.
//
// @Summary      Liveness probe
// @Description  Returns 200 as long as the process is up. No auth required.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.HealthResponse
// @Router       /api/healthz [get]
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": http.StatusText(http.StatusOK)})
}

// readyz is a readiness probe.
//
// @Summary      Readiness probe
// @Description  Returns 200 when MQTT is connected (or DISABLE_MQTT=true). 503 otherwise. No auth required.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.ReadyResponse
// @Failure      503  {object}  dto.ErrorEnvelope
// @Router       /api/readyz [get]
func readyz(c *gin.Context) {
	if isReady == nil || !isReady.Load().(bool) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": http.StatusText(http.StatusServiceUnavailable)})
		return
	}
	TeslaMateAPIHandleSuccessResponse(c, "webserver", gin.H{"status": http.StatusText(http.StatusOK)})
}

// notFoundHandler renders a JSON 404 envelope for routes that don't match.
//
// @Summary      404 fallback
// @Description  JSON 404 envelope returned for any unmatched route.
// @Tags         system
// @Produce      json
// @Success      404  {object}  dto.NotFoundResponse
func notFoundHandler(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
}

// rootRoot returns the handler for GET / — a tiny liveness banner that
// echoes the configured base path.
//
// @Summary      Root banner
// @Description  Returns a small banner confirming the API process is running.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.MessageEnvelope
// @Router       / [get]
func rootRoot(r *gin.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": r.BasePath()})
	}
}

// apiRoot returns the banner handler for GET /api.
//
// @Summary      /api banner
// @Description  Banner for the /api root.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.MessageEnvelope
// @Router       /api/ [get]
func apiRoot(api *gin.RouterGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": api.BasePath()})
	}
}

// apiV1Root returns the banner handler for GET /api/v1.
//
// @Summary      /api/v1 banner
// @Description  Banner for the /api/v1 root.
// @Tags         system
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dto.MessageEnvelope
// @Router       /api/v1/ [get]
func apiV1Root(v1 *gin.RouterGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi v1 running..", "path": v1.BasePath()})
	}
}

// apiV2Root returns the banner handler for GET /api/v2.
//
// @Summary      /api/v2 banner
// @Description  Banner for the /api/v2 root.
// @Tags         system
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dto.MessageEnvelope
// @Router       /api/v2/ [get]
func apiV2Root(v2 *gin.RouterGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi v2 running..", "path": v2.BasePath()})
	}
}

// apiPing answers the unauthenticated /api/ping liveness check.
//
// @Summary      Ping
// @Description  Returns {"message":"pong"} for simple uptime checks. No auth required.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.PongResponse
// @Router       /api/ping [get]
func apiPing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}
