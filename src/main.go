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
	dbTimestampFormat = "2006-01-02T15:04:05Z" // PostgreSQL 时间字段使用的格式。
)

var (
	// Kubernetes readyz 探针使用的应用就绪状态。
	isReady *atomic.Value

	// TeslaMateApi 运行参数。
	apiVersion = "unspecified"

	// 全局数据库连接池。
	db *sql.DB

	// 应用使用的本地时区。
	appUsersTimezone *time.Location
)

// main 初始化运行环境、路由和 HTTP 服务。
func main() {
	// 初始化就绪探针状态。
	isReady = &atomic.Value{}
	isReady.Store(false)

	// 设置日志格式。
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	// DEBUG_MODE 未开启时使用 Gin 发布模式。
	if !getEnvAsBool("DEBUG_MODE", false) {
		// 将 GIN_MODE 设置为发布模式。
		gin.SetMode(gin.ReleaseMode)
		log.Printf("[info] TeslaMateApi running in release mode.")
	} else {
		// 将 GIN_MODE 设置为调试模式。
		gin.SetMode(gin.DebugMode)
		log.Printf("[info] TeslaMateApi running in debug mode.")
	}

	// 从环境变量读取应用配置。
	appUsersTimezone, _ = time.LoadLocation(getEnv("TZ", "Europe/Berlin"))
	if gin.IsDebugging() {
		log.Println("[debug] TeslaMateApi appUsersTimezone:", appUsersTimezone)
	}

	// 初始化数据库连接。
	initDBconnection()
	defer db.Close()

	// 连接 MQTT，用于兼容状态接口的实时数据缓存。
	statusCache, err := startMQTT()
	mqttStatusCache = statusCache
	if getEnvAsBool("DISABLE_MQTT", false) {
		log.Printf("[info] TeslaMateApi MQTT connection not established.")
	} else {
		if err != nil {
			log.Fatalf("[error] TeslaMateApi MQTT connection failed: %s", err)
		}
	}

	// 初始化 Gin 路由。
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/api"

	// 启用 GZIP 响应压缩。
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	r.Use(func(c *gin.Context) {
		c.Header(headerAPIVersion, apiVersion)
		c.Next()
	})

	// 设置 404 响应。
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	// 禁用 Gin 的代理信任配置。
	_ = r.SetTrustedProxies(nil)

	// 根路径返回 API 运行状态。
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": r.BasePath()})
	})

	// 注册 TeslaMateApi /api 路由。
	api := r.Group("/api")
	BasePathV1 := api.BasePath() + "/v1"
	BasePathV2 := api.BasePath() + "/v2"
	{
		// 注册 TeslaMateApi /api 根路由。
		api.GET("/", apiRoot)

		// 注册 TeslaMateApi /api/v1 路由。
		v1 := api.Group("/v1")
		{
			// 注册 TeslaMateApi /api/v1 根路由。
			v1.GET("/", apiV1Root)
			docsui.RegisterRoutes(v1, BasePathV1)
			registerCompatibleV1Routes(v1)
		}

		// 注册 TeslaMateApi /api/v2 路由。
		v2 := api.Group("/v2")
		{
			// 注册 TeslaMateApi /api/v2 根路由。
			v2.GET("/", apiV2Root)
			docsui.RegisterRoutes(v2, BasePathV2)
			registerExtendedV2Routes(v2)
		}

		// 注册 /api/ping 存活检查。
		api.GET("/ping", apiPing)

		// 注册 Kubernetes 健康检查路由。
		api.GET("/healthz", healthz)
		api.GET("/readyz", readyz)
	}

	// 创建 HTTP 服务。
	listenAddr := getEnv("TESLAMATEAPI_LISTEN_ADDR", ":8080")
	server := &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}

	// 禁用 MQTT 时，数据库初始化完成即可认为服务就绪。
	if getEnvAsBool("DISABLE_MQTT", false) {
		isReady.Store(true)
	}

	// 处理优雅关闭。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// 后台等待关闭信号。
	go func() {
		<-quit
		log.Println("[info] TeslaMateAPI received shutdown input")
		if err := server.Close(); err != nil {
			log.Fatal("[error] TeslaMateAPI server close error:", err)
		}
	}()

	log.Printf("[info] TeslaMateAPI listening on %s", listenAddr)
	// 启动 HTTP 服务。
	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("[info] TeslaMateAPI server gracefully shut down")
		} else {
			log.Fatal("[error] TeslaMateAPI server closed unexpectedly")
		}
	}
}

// apiRoot 返回 /api 根路径状态。
// @Summary API 根路径
// @Tags 系统
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router / [get]
func apiRoot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi container running..", "path": "/api"})
}

// apiV1Root 返回 /api/v1 根路径状态。
// @Summary API v1 根路径
// @Tags 系统
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router /v1 [get]
func apiV1Root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi v1 running..", "path": "/api/v1"})
}

// apiV2Root 返回 /api/v2 根路径状态。
// @Summary API v2 根路径
// @Tags 系统
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router /v2 [get]
func apiV2Root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "TeslaMateApi v2 running..", "path": "/api/v2"})
}

// apiPing 返回简单的 API 存活响应。
// @Summary 连通性检查
// @Tags 系统
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router /ping [get]
func apiPing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

// initDBconnection 初始化数据库连接池。
func initDBconnection() {
	var err error

	// 读取数据库连接环境变量及默认值。
	dbhost := getEnv("DATABASE_HOST", "database")
	dbport := getEnvAsInt("DATABASE_PORT", 5432)
	dbuser := getEnv("DATABASE_USER", "teslamate")
	dbpass := getEnv("DATABASE_PASS", "secret")
	dbname := getEnv("DATABASE_NAME", "teslamate")
	dbtimeout := (getEnvAsInt("DATABASE_TIMEOUT", 60000) / 1000)
	dbsslmode := getEnv("DATABASE_SSL", "disable")
	dbsslrootcert := getEnv("DATABASE_SSL_CA_CERT_FILE", "")

	// 兼容历史布尔写法的 SSL 模式配置。
	switch dbsslmode {
	case "true", "noverify":
		dbsslmode = "require"
	case "false":
		dbsslmode = "disable"
	}

	// 构建 PostgreSQL 连接字符串。
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d", dbhost, dbport, dbuser, dbpass, dbname, dbsslmode, dbtimeout)

	// 如配置了 SSL 根证书，则加入连接参数。
	if dbsslrootcert != "" {
		psqlInfo += " sslrootcert=" + dbsslrootcert
	}

	// 打开数据库连接池。
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("[error] initDBconnection - database connection error: %v", err)
	}

	// 测试数据库连通性。
	if err = db.Ping(); err != nil {
		log.Fatalf("[error] initDBconnection - database ping error: %v", err)
	}

	// 调试模式下记录数据库连接成功信息。
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

	// 优先解析 RFC3339 或数据库时间格式，失败后回退到 API 时间解析器。
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

	// 按用户配置时区输出 RFC3339 时间。
	ReturnDate := t.In(appUsersTimezone).Format(time.RFC3339)

	// 记录时区转换结果，便于排查响应时间字段。
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

// getEnv 读取环境变量；变量不存在或为空时返回默认值。
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultVal
}

// getEnvAsBool 读取布尔环境变量；解析失败时返回默认值。
func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

// getEnvAsInt 读取整数环境变量；解析失败时返回默认值。
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

// convertStringToBool 将字符串转换为布尔值；解析失败时返回 false。
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

// convertStringToFloat 将字符串转换为 float64；解析失败时返回 0。
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

// convertStringToInteger 将字符串转换为 int；解析失败时返回 0。
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

// kilometersToMiles 将公里转换为英里。
func kilometersToMiles(km float64) float64 {
	return (km * 0.62137119223733)
}

// kilometersToMilesNilSupport 将可空公里数转换为英里。
func kilometersToMilesNilSupport(km NullFloat64) NullFloat64 {
	km.Float64 = (km.Float64 * 0.62137119223733)
	return (km)
}

// milesToKilometers 将英里转换为公里。
func milesToKilometers(mi float64) float64 {
	return (mi * 1.609344)
}

// kilometersToMilesInteger 将整数公里数转换为整数英里数。
func kilometersToMilesInteger(km int) int {
	return int(float64(km) * 0.62137119223733)
}

// barToPsi 将 bar 转换为 psi。
func barToPsi(bar float64) float64 {
	return (bar * 14.503773800722)
}

// celsiusToFahrenheit 将摄氏度转换为华氏度。
func celsiusToFahrenheit(c float64) float64 {
	return (c*9/5 + 32)
}

// celsiusToFahrenheitNilSupport 将可空摄氏度转换为华氏度。
func celsiusToFahrenheitNilSupport(c NullFloat64) NullFloat64 {
	c.Float64 = (c.Float64*9/5 + 32)
	return (c)
}

// checkArrayContainsString 判断字符串是否存在于字符串数组中。
func checkArrayContainsString(s []string, e string) bool {
	return slices.Contains(s, e)
}

// healthz 返回服务存活状态。
// @Summary 健康检查
// @Tags 系统
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router /healthz [get]
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": http.StatusText(http.StatusOK)})
}

// readyz 返回服务就绪状态。
// @Summary 就绪检查
// @Tags 系统
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
