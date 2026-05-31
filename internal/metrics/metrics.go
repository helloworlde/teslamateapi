// Package metrics exposes Prometheus instrumentation for TeslaMateApi.
//
// The /metrics endpoint is served unauthenticated (alongside /healthz and
// /readyz) so a typical in-cluster Prometheus can scrape it without
// having to plumb the API_TOKEN through scrape configs. Every metric is
// prefixed with `teslamateapi_` to avoid colliding with Go runtime / process
// collectors that ship under their own namespaces.
package metrics

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "teslamateapi"

// Registry is the registry every TeslaMateApi metric is registered against.
// It's intentionally a private registry (instead of prometheus.DefaultRegisterer)
// so we control which Go/process collectors are exposed and so unit tests can
// build a fresh registry without globals leaking between runs.
var Registry = prometheus.NewRegistry()

// HTTPRequestsTotal counts every HTTP request that hits the gin engine,
// labelled by method, normalised route, and response status code. The route
// label is the gin route pattern (e.g. /api/v1/cars/:CarID) — never the raw
// URL — to keep label cardinality bounded.
var HTTPRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total HTTP requests received, by method, route pattern, and status code.",
	},
	[]string{"method", "route", "status"},
)

// HTTPRequestDuration is the request latency histogram. Buckets cover the
// full latency range we expect: snappy MQTT-cache reads (<10ms), routine
// Postgres queries (10-200ms), and the slowest Tesla owner-api round-trips
// (>1s when the car is asleep).
var HTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request latency in seconds, by method and route pattern.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	},
	[]string{"method", "route"},
)

// AuditEventsTotal counts privileged-command audit events, mirroring the
// records emitted by internal/audit. Useful for alerting on spikes of
// `outcome="denied"` (auth failures) or `outcome="upstream_error"` (Tesla
// owner-api degradation). Reason is left empty for successes.
var AuditEventsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "audit",
		Name:      "events_total",
		Help:      "Privileged command audit events, by action, outcome, and reason.",
	},
	[]string{"action", "outcome", "reason"},
)

// MQTTConnected reports the MQTT broker connection state: 1 = connected,
// 0 = disconnected, -1 = MQTT disabled by configuration. Using -1 instead of
// "no metric at all" lets dashboards distinguish "broker is down" from
// "this deployment was configured without MQTT".
var MQTTConnected = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "mqtt",
		Name:      "connected",
		Help:      "MQTT broker connection state (1 connected, 0 disconnected, -1 disabled).",
	},
)

// dbStatsCollector is replaced by RegisterDB; we keep the slot so re-registers
// during tests don't panic.
func init() {
	Registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		HTTPRequestsTotal,
		HTTPRequestDuration,
		AuditEventsTotal,
		MQTTConnected,
	)
}

// RegisterDB attaches a sql.DBStats collector to Registry so connection-pool
// stats (open/idle/in-use, wait counts, lifetime closures) show up under
// `go_sql_*` metrics. Safe to call once at startup; calling again will panic
// on duplicate registration, which is the desired loud failure.
func RegisterDB(db *sql.DB) {
	Registry.MustRegister(collectors.NewDBStatsCollector(db, "teslamate"))
}

// SetMQTTConnected updates the MQTT gauge. Use Disabled() / Connected() from
// status.Cache to derive the right value at startup; thereafter the MQTT
// callbacks should call this on every state transition.
func SetMQTTConnected(state MQTTState) {
	MQTTConnected.Set(float64(state))
}

// MQTTState is the discriminated value stored in MQTTConnected.
type MQTTState int

const (
	// MQTTDisabled means the deployment opted out of MQTT entirely
	// (DISABLE_MQTT=true). Distinct from a transient disconnection.
	MQTTDisabled MQTTState = -1
	// MQTTDisconnected means MQTT is configured on but the broker is
	// unreachable right now.
	MQTTDisconnected MQTTState = 0
	// MQTTConnectedState means the broker is currently connected.
	MQTTConnectedState MQTTState = 1
)

// Handler returns the http.Handler that serves /metrics. It writes the
// Prometheus exposition format from Registry only — no implicit fall-through
// to the global default registry.
func Handler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{
		Registry:          Registry,
		EnableOpenMetrics: true,
	})
}

// Middleware returns a gin middleware that records request count and latency.
// Routes that didn't match any handler are bucketed under route="<unmatched>"
// so we don't blow up label cardinality with the raw URLs of scanner traffic.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "<unmatched>"
		}
		status := strconv.Itoa(c.Writer.Status())
		HTTPRequestsTotal.WithLabelValues(c.Request.Method, route, status).Inc()
		HTTPRequestDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}

// ObserveAudit records one audit event into AuditEventsTotal. Lives here
// (instead of internal/audit) so the audit package stays free of the
// Prometheus dependency — audit.Log calls into this via a small registered
// hook so the two packages don't form an import cycle.
func ObserveAudit(action, outcome, reason string) {
	AuditEventsTotal.WithLabelValues(action, outcome, reason).Inc()
}
