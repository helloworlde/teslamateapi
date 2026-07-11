// Package v1 implements every /api/v1/* HTTP handler.
//
// Handlers are methods on Handler so the deps (db, tz, allow-list, status
// cache, narrow per-handler config) flow through a single struct instead
// of package-level globals.
package v1

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/auth"
	"github.com/tobiasehlert/teslamateapi/internal/command"
	"github.com/tobiasehlert/teslamateapi/internal/database"
	"github.com/tobiasehlert/teslamateapi/internal/httpparams"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
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
	Language      string

	// CommandsEnabled mirrors ENABLE_COMMANDS. Handler bodies still
	// re-check this so legacy callers see identical 403 behaviour.
	CommandsEnabled bool
}

// Deps is the set of inputs Handler needs to serve every v1 route.
type Deps struct {
	DB          *database.DB
	TZ          *time.Location
	AllowList   *command.AllowList // nil when commands disabled
	StatusCache *status.Cache      // nil when MQTT disabled
	Token       *auth.Token        // for the per-handler bearer-check
	Cfg         Config
}

// Handler is the v1 HTTP handler registered onto the /api/v1 gin group.
type Handler struct {
	db          *database.DB
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

const maxV1PageSize = 10000

const (
	maxProxyRequestBodyBytes  int64 = 1 << 20
	maxProxyResponseBodyBytes int64 = 10 << 20
)

var errBodyTooLarge = errors.New("body exceeds configured limit")

func requirePositiveIntParam(c *gin.Context, handler, name, raw string) (int, bool) {
	v, err := httpparams.PositiveInt(raw)
	if err != nil {
		respond.HandleError(c, handler, name+" must be a positive integer.", "got: "+raw)
		return 0, false
	}
	return v, true
}

func requirePositiveIntParamStatus(c *gin.Context, name, raw string) (int, bool) {
	v, err := httpparams.PositiveInt(raw)
	if err != nil {
		respond.HandleOther(c, http.StatusBadRequest, gin.H{"error": name + " invalid"})
		return 0, false
	}
	return v, true
}

func optionalIntInRange(c *gin.Context, handler, name, raw string, def, min, max int) (int, bool) {
	v, err := httpparams.OptionalIntInRange(raw, def, min, max)
	if err != nil {
		respond.HandleError(c, handler, fmt.Sprintf("%s must be an integer in [%d, %d].", name, min, max), "got: "+raw)
		return 0, false
	}
	return v, true
}

func optionalFloatMin(c *gin.Context, handler, name, raw string, min float64) (float64, bool) {
	v, err := httpparams.OptionalFloatMin(raw, min)
	if err != nil {
		respond.HandleError(c, handler, fmt.Sprintf("%s must be a number >= %.0f.", name, min), "got: "+raw)
		return 0, false
	}
	return v, true
}

func readAllLimited(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("%w: %d bytes", errBodyTooLarge, limit)
	}
	return b, nil
}

func bodyTooLarge(err error) bool {
	return errors.Is(err, errBodyTooLarge)
}
