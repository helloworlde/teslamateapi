// Package server assembles the gin engine: middleware, system endpoints,
// V1/V2 routes, docs, and database initialisation.
//
// @title TeslaMateApi
// @version 2.0
// @description TeslaMate 数据查询与分析 API。提供 V1（原始数据）和 V2（聚合分析）两套接口。
// @BasePath /api
//
// @tag.name system
// @tag.description 系统状态、健康检查、API 文档元信息
//
// @tag.name v1
// @tag.description V1 接口 — TeslaMate 原始数据查询（车辆、充电、行驶、OTA、电池、胎压、全局设置）
//
// @tag.name v2
// @tag.description V2 接口 — 聚合分析（充电/行驶/驻车/电池/费用/环境/更新/汇总/生命周期）
package server

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/api/v1"
	v2 "github.com/tobiasehlert/teslamateapi/internal/api/v2"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
	"github.com/tobiasehlert/teslamateapi/internal/config"
	"github.com/tobiasehlert/teslamateapi/internal/docs"
)

// Run boots the HTTP server and blocks until SIGINT.
func Run() {
	apicommon.IsReady.Store(false)

	log.SetFlags(log.Ldate | log.Lmicroseconds)

	if !config.EnvAsBool("DEBUG_MODE", false) {
		gin.SetMode(gin.ReleaseMode)
		log.Printf("[info] TeslaMateApi running in release mode.")
	} else {
		gin.SetMode(gin.DebugMode)
		log.Printf("[info] TeslaMateApi running in debug mode.")
	}

	apicommon.AppUsersTimezone, _ = time.LoadLocation(config.Env("TZ", "Europe/Berlin"))
	if gin.IsDebugging() {
		log.Println("[debug] TeslaMateApi appUsersTimezone:", apicommon.AppUsersTimezone)
	}

	initDB()
	defer apicommon.DB.Close()

	// Connect to the MQTT broker for the v1 status endpoint. When
	// DISABLE_MQTT=true the cache is nil and /v1/cars/:CarID/status
	// responds with 501 NotImplemented.
	statusCache, err := v1.StartMQTT()
	if config.EnvAsBool("DISABLE_MQTT", false) {
		log.Printf("[info] TeslaMateApi MQTT connection not established.")
	} else if err != nil {
		log.Fatalf("[error] TeslaMateApi MQTT connection failed: %s", err)
	}

	r := gin.Default()
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(func(c *gin.Context) {
		c.Header(config.HeaderAPIVersion, config.APIVersion)
		c.Next()
	})
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})
	_ = r.SetTrustedProxies(nil)

	r.GET("/", systemRoot(r))

	api := r.Group("/api")
	{
		api.GET("/", systemAPIRoot(api))

		v1g := api.Group("/v1")
		{
			v1g.GET("/", systemV1Root(v1g))

			v1g.GET("/cars", v1.TeslaMateAPICarsV1)
			v1g.GET("/cars/:CarID", v1.TeslaMateAPICarsV1)
			v1g.GET("/cars/:CarID/battery-health", v1.TeslaMateAPICarsBatteryHealthV1)
			v1g.GET("/cars/:CarID/tire-pressure", v1.TeslaMateAPICarsTirePressureV1)
			v1g.GET("/cars/:CarID/charges", v1.TeslaMateAPICarsChargesV1)
			v1g.GET("/cars/:CarID/charges/current", v1.TeslaMateAPICarsChargesCurrentV1)
			v1g.GET("/cars/:CarID/charges/:ChargeID", v1.TeslaMateAPICarsChargesDetailsV1)
			v1g.GET("/cars/:CarID/drives", v1.TeslaMateAPICarsDrivesV1)
			v1g.GET("/cars/:CarID/drives/:DriveID", v1.TeslaMateAPICarsDrivesDetailsV1)
			if statusCache != nil {
				v1g.GET("/cars/:CarID/status", statusCache.TeslaMateAPICarsStatusV1)
			}
			v1g.GET("/cars/:CarID/updates", v1.TeslaMateAPICarsUpdatesV1)
			v1g.GET("/globalsettings", v1.TeslaMateAPIGlobalsettingsV1)
		}

		v2.RegisterV2Routes(api, nil)
		docs.RegisterRoutes(api)

		api.GET("/ping", systemPing)
		api.GET("/healthz", healthz)
		api.GET("/readyz", readyz)
	}

	BasePathV1 := api.BasePath() + "/v1"
	r.GET("/cars", redirectTo(BasePathV1))
	r.GET("/cars/:CarID", redirectTo(BasePathV1))
	r.GET("/cars/:CarID/charges", redirectTo(BasePathV1))
	r.GET("/cars/:CarID/charges/:ChargeID", redirectTo(BasePathV1))
	r.GET("/cars/:CarID/drives", redirectTo(BasePathV1))
	r.GET("/cars/:CarID/drives/:DriveID", redirectTo(BasePathV1))
	r.GET("/cars/:CarID/status", redirectTo(BasePathV1))
	r.GET("/cars/:CarID/updates", redirectTo(BasePathV1))
	r.GET("/globalsettings", redirectTo(BasePathV1))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// readyz: ready as soon as the DB connection is up — unless MQTT is
	// enabled, in which case the MQTT connect handler will flip readiness
	// once the broker subscription succeeds.
	if config.EnvAsBool("DISABLE_MQTT", false) {
		apicommon.IsReady.Store(true)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		log.Println("[info] TeslaMateAPI received shutdown input")
		if err := srv.Close(); err != nil {
			log.Fatal("[error] TeslaMateAPI server close error:", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("[info] TeslaMateAPI server gracefully shut down")
		} else {
			log.Fatal("[error] TeslaMateAPI server closed unexpectedly")
		}
	}
}

func redirectTo(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, prefix+c.Request.RequestURI)
	}
}
