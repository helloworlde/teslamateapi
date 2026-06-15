package httpapi

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/auth"
	"github.com/tobiasehlert/teslamateapi/internal/httpapi/handlers/system"
	v1 "github.com/tobiasehlert/teslamateapi/internal/httpapi/handlers/v1"
	v2 "github.com/tobiasehlert/teslamateapi/internal/httpapi/handlers/v2"
	"github.com/tobiasehlert/teslamateapi/internal/metrics"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
)

// Deps groups everything NewRouter needs to wire the gin engine.
type Deps struct {
	APIVersion      string
	Token           *auth.Token
	System          *system.Handler
	V1              *v1.Handler
	V2              *v2.Handler
	CommandsEnabled bool
}

// NewRouter builds the gin engine with the same routes as the legacy
// src/webserver.go. External HTTP behaviour (paths, methods, redirects,
// 404 shape) is preserved byte-identical.
func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		path := param.Path
		if param.Request != nil {
			path = respond.SafeRequestURIFromURL(param.Request.URL)
		}
		return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %#v\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			path,
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(APIVersionHeader(d.APIVersion))
	r.Use(metrics.Middleware())
	r.NoRoute(system.NotFound)
	_ = r.SetTrustedProxies(nil)

	// root endpoint telling API is running
	r.GET("/", system.Root(r))

	// /metrics is intentionally outside the /api group: it bypasses the
	// bearer-auth gate (matching /api/healthz / /api/readyz) so an in-cluster
	// Prometheus can scrape without plumbing the API_TOKEN through scrape
	// configs. If you need it gated, put it behind your reverse proxy.
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	// /api with auth gate; the gate has its own internal allow-list for
	// public probes (health, readiness, docs, openapi spec, ping) and the
	// /api root.
	api := r.Group("/api", auth.Middleware(d.Token))
	{
		api.GET("/", system.APIRoot(api))

		v1g := api.Group("/v1")
		{
			v1g.GET("/", system.APIV1Root(v1g))

			v1g.GET("/cars", d.V1.Cars)
			v1g.GET("/cars/:CarID", d.V1.Cars)

			v1g.GET("/cars/:CarID/battery-health", d.V1.BatteryHealth)

			v1g.GET("/cars/:CarID/charges", d.V1.Charges)
			v1g.GET("/cars/:CarID/charges/current", d.V1.ChargesCurrent)
			v1g.GET("/cars/:CarID/charges/:ChargeID", d.V1.ChargesDetails)

			// Command + logging routes are only registered when commands
			// are enabled. Skipping the registration (vs. handler-level
			// 403) means scanners get 404 instead of a hint that command
			// machinery is present.
			if d.CommandsEnabled {
				v1g.GET("/cars/:CarID/command", d.V1.CommandList)
				v1g.GET("/cars/:CarID/commands", d.V1.CommandList)
				v1g.POST("/cars/:CarID/command/:Command", d.V1.CommandExec)

				v1g.GET("/cars/:CarID/logging", d.V1.LoggingList)
				v1g.PUT("/cars/:CarID/logging/:Command", d.V1.LoggingExec)

				v1g.POST("/cars/:CarID/wake_up", d.V1.CommandExec)
			}

			v1g.GET("/cars/:CarID/drives", d.V1.Drives)
			v1g.GET("/cars/:CarID/drives/:DriveID", d.V1.DrivesDetails)

			v1g.GET("/cars/:CarID/status", d.V1.Status)

			v1g.GET("/cars/:CarID/updates", d.V1.Updates)

			v1g.GET("/globalsettings", d.V1.Globalsettings)
		}

		v2g := api.Group("/v2")
		{
			v2g.GET("/", system.APIV2Root(v2g))

			v2g.GET("/cars/:CarID/parkings", d.V2.Parkings)
			v2g.GET("/cars/:CarID/parkings/:PrecedingDriveID", d.V2.ParkingsDetails)

			v2g.GET("/cars/:CarID/stats/lifetime", d.V2.StatsLifetime)
			v2g.GET("/cars/:CarID/stats/summary", d.V2.StatsSummary)
			v2g.GET("/cars/:CarID/stats/time-distribution", d.V2.StatsTimeDistribution)
			v2g.GET("/cars/:CarID/stats/by-geofence", d.V2.StatsByGeofence)
			v2g.GET("/cars/:CarID/stats/consumption", d.V2.StatsConsumption)
			v2g.GET("/cars/:CarID/stats/behavior", d.V2.StatsBehavior)
		}

		api.GET("/ping", system.Ping)
		api.GET("/healthz", system.Healthz)
		api.GET("/readyz", d.System.Readyz)

		api.GET("/docs", system.ScalarDocs)
		api.GET("/openapi.yaml", system.OpenAPIYAML)
	}

	// Pre-versioning legacy paths — 301 to /api/v1/...
	basePathV1 := api.BasePath() + "/v1"
	redir := func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, basePathV1+c.Request.RequestURI)
	}
	r.GET("/cars", redir)
	r.GET("/cars/:CarID", redir)
	r.GET("/cars/:CarID/charges", redir)
	r.GET("/cars/:CarID/charges/:ChargeID", redir)
	r.GET("/cars/:CarID/drives", redir)
	r.GET("/cars/:CarID/drives/:DriveID", redir)
	r.GET("/cars/:CarID/status", redir)
	r.GET("/cars/:CarID/updates", redir)
	r.GET("/globalsettings", redir)

	return r
}
