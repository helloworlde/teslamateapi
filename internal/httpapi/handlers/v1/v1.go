// Package v1 implements every /api/v1/* HTTP handler.
//
// Handlers are methods on Handler so the deps (db, tz, allow-list, status
// cache, narrow per-handler config) flow through a single struct instead
// of package-level globals.
package v1

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/auth"
	"github.com/tobiasehlert/teslamateapi/internal/command"
	"github.com/tobiasehlert/teslamateapi/internal/status"
	"github.com/tobiasehlert/teslamateapi/internal/timefmt"
	"github.com/tobiasehlert/teslamateapi/pkg/nullable"
)

// Type aliases preserve the legacy nullable type names used throughout
// these handler bodies, with the canonical definitions living in
// pkg/nullable.
type (
	// NullInt64 mirrors nullable.Int64.
	NullInt64 = nullable.Int64
	// NullBool mirrors nullable.Bool.
	NullBool = nullable.Bool
	// NullFloat64 mirrors nullable.Float64.
	NullFloat64 = nullable.Float64
	// NullString mirrors nullable.String.
	NullString = nullable.String
)

// Config holds the narrow slice of runtime config that v1 handlers need.
type Config struct {
	APIVersion    string
	EncryptionKey string
	TeslaAPIHost  string
	TeslaMateHost string
	TeslaMatePort string
	TeslaMateSSL  bool

	// CommandsEnabled mirrors ENABLE_COMMANDS. Handler bodies still
	// re-check this so legacy callers see identical 403 behaviour.
	CommandsEnabled bool
}

// Deps is the set of inputs Handler needs to serve every v1 route.
type Deps struct {
	DB          *sql.DB
	TZ          *time.Location
	AllowList   *command.AllowList // nil when commands disabled
	StatusCache *status.Cache      // nil when MQTT disabled
	Token       *auth.Token        // for the per-handler bearer-check
	Cfg         Config
}

// Handler is the v1 HTTP handler registered onto the /api/v1 gin group.
type Handler struct {
	db          *sql.DB
	tz          *time.Location
	allowList   *command.AllowList
	statusCache *status.Cache
	token       *auth.Token
	cfg         Config
}

// New returns a Handler wired with the supplied deps.
func New(d Deps) *Handler {
	return &Handler{
		db:          d.DB,
		tz:          d.TZ,
		allowList:   d.AllowList,
		statusCache: d.StatusCache,
		token:       d.Token,
		cfg:         d.Cfg,
	}
}

// timeInTZ formats a Postgres timestamp string in the user's timezone.
func (h *Handler) timeInTZ(s string) string {
	return timefmt.GetTimeInTimeZone(s, h.tz)
}

// parseDate normalises a user-supplied filter into the DB timestamp
// format. Empty input returns "" with no error.
func (h *Handler) parseDate(s string) (string, error) {
	return timefmt.ParseDateParam(s, h.tz)
}

// validateAuthToken proxies to the configured Token. Handlers wrap this
// to keep the legacy call sites readable.
func (h *Handler) validateAuthToken(c *gin.Context) (bool, string) {
	if h.token == nil {
		return true, ""
	}
	return h.token.Validate(c)
}

// allowListItems returns the resolved /command + /logging allow-list, or
// an empty slice if commands are disabled.
func (h *Handler) allowListItems() []string {
	if h.allowList == nil {
		return nil
	}
	return h.allowList.Items()
}

// allowListContains reports whether cmd is in the allow-list.
func (h *Handler) allowListContains(cmd string) bool {
	if h.allowList == nil {
		return false
	}
	return h.allowList.Contains(cmd)
}
