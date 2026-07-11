// Package main is the TeslaMateApi HTTP server entry point.
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
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/audit"
	"github.com/tobiasehlert/teslamateapi/internal/auth"
	"github.com/tobiasehlert/teslamateapi/internal/command"
	"github.com/tobiasehlert/teslamateapi/internal/config"
	"github.com/tobiasehlert/teslamateapi/internal/database"
	"github.com/tobiasehlert/teslamateapi/internal/httpapi"
	"github.com/tobiasehlert/teslamateapi/internal/httpapi/handlers/system"
	v1 "github.com/tobiasehlert/teslamateapi/internal/httpapi/handlers/v1"
	v2 "github.com/tobiasehlert/teslamateapi/internal/httpapi/handlers/v2"
	"github.com/tobiasehlert/teslamateapi/internal/metrics"
	"github.com/tobiasehlert/teslamateapi/internal/status"
)

// apiVersion is injected at link time (-ldflags "-X main.apiVersion=...").
// It's the only package-level var here and is read-only after init.
var apiVersion = "unspecified"

func main() {
	// readiness flag for k8s probes
	ready := &atomic.Value{}
	ready.Store(false)

	log.SetFlags(log.Ldate | log.Lmicroseconds)

	cfg := config.Load()

	if !cfg.DebugMode {
		gin.SetMode(gin.ReleaseMode)
		log.Printf("[info] TeslaMateApi running in release mode.")
	} else {
		gin.SetMode(gin.DebugMode)
		log.Printf("[info] TeslaMateApi running in debug mode.")
	}

	tz := config.LoadTZ(cfg.TZName)
	if gin.IsDebugging() {
		log.Println("[debug] TeslaMateApi appUsersTimezone:", tz)
	}

	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("[error] database init: %v", err)
	}
	defer db.Close()
	metrics.RegisterDB(db)

	// Hand the slow-query-logging wrapper to the handlers; metrics keeps the
	// raw *sql.DB so connection-pool stats are unaffected.
	loggingDB := database.Wrap(db)

	// Audit hook → Prometheus counter. Wired here (rather than in the audit
	// package's init) so internal/audit stays free of the prometheus
	// dependency for environments that compile it out.
	audit.ObserveHook = metrics.ObserveAudit

	tok := auth.New(cfg)
	allowList := command.NewAllowList(cfg)

	// Commands hit Tesla's owner-api / TeslaMate's logging endpoint with the
	// car's stored access token. Exposing them without auth turns the API
	// into an open relay. Refuse to start in that configuration.
	if cfg.CommandsEnabled {
		if cfg.APITokenDisable {
			log.Fatal("[error] ENABLE_COMMANDS=true requires authentication; refusing to start with API_TOKEN_DISABLE=true.")
		}
		if cfg.APIToken == "" {
			log.Fatal("[error] ENABLE_COMMANDS=true requires API_TOKEN to be set; refusing to start without it.")
		}
		if len(cfg.APIToken) < 32 {
			log.Fatal("[error] ENABLE_COMMANDS=true requires API_TOKEN of at least 32 characters; refusing to start.")
		}
	}

	// Connect to the MQTT broker
	statusCache, err := status.New(cfg, ready)
	if err != nil {
		log.Fatalf("[error] TeslaMateApi MQTT connection failed: %s", err)
	}
	if cfg.MQTTDisabled {
		log.Printf("[info] TeslaMateApi MQTT connection not established.")
		metrics.SetMQTTConnected(metrics.MQTTDisabled)
	} else if statusCache.Connected() {
		metrics.SetMQTTConnected(metrics.MQTTConnectedState)
	} else {
		metrics.SetMQTTConnected(metrics.MQTTDisconnected)
	}

	if cfg.APITokenDisable {
		log.Println("[warning] validateAuthToken - header authorization bearer token disabled. Authorization: Bearer token will not be required for commands.")
	}

	if cfg.TeslaAPIHost != "" {
		log.Printf("[info] TESLA_API_HOST is set: %s", cfg.TeslaAPIHost)
	}

	v1Handler := v1.New(v1.Deps{
		DB:          loggingDB,
		TZ:          tz,
		AllowList:   allowList,
		StatusCache: statusCache,
		Token:       tok,
		Cfg: v1.Config{
			APIVersion:      apiVersion,
			EncryptionKey:   cfg.EncryptionKey,
			TeslaAPIHost:    cfg.TeslaAPIHost,
			TeslaMateHost:   cfg.TeslaMateHost,
			TeslaMatePort:   cfg.TeslaMatePort,
			TeslaMateSSL:    cfg.TeslaMateSSL,
			Language:        cfg.Language,
			CommandsEnabled: cfg.CommandsEnabled,
		},
	})

	v2Handler := v2.New(v2.Deps{
		DB: loggingDB,
		TZ: tz,
	})

	systemHandler := system.New(ready)

	router := httpapi.NewRouter(httpapi.Deps{
		APIVersion:      apiVersion,
		Token:           tok,
		System:          systemHandler,
		V1:              v1Handler,
		V2:              v2Handler,
		CommandsEnabled: cfg.CommandsEnabled,
	})

	listenAddr := cfg.ListenAddr
	server := &http.Server{
		Addr:              listenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if cfg.MQTTDisabled {
		ready.Store(true)
	}

	// graceful shutdown
	shutdownSignal, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	go func() {
		<-shutdownSignal.Done()
		log.Println("[info] TeslaMateAPI received shutdown input")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("[error] TeslaMateAPI graceful shutdown error: %v", err)
			if closeErr := server.Close(); closeErr != nil {
				log.Printf("[error] TeslaMateAPI server close error: %v", closeErr)
			}
		}
	}()

	log.Printf("[info] TeslaMateAPI listening on %s (version=%s)", listenAddr, apiVersion)
	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("[info] TeslaMateAPI server gracefully shut down")
		} else {
			// Most common cause: port already bound by another instance — surface
			// the underlying os/syscall error so the user can see "address already
			// in use" instead of just "closed unexpectedly".
			log.Fatalf("[error] TeslaMateAPI server failed to start on %s: %v", listenAddr, err)
		}
	}
}
