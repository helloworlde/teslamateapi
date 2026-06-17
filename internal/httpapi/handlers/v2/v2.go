// Package v2 implements every /api/v2/* HTTP handler.
//
// v2 is additive over v1; routes here use real HTTP error codes (not the
// legacy 200+envelope shape) and stricter param validation.
package v2

import (
	"database/sql"
	"time"

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

// Deps is the set of inputs Handler needs to serve every v2 route.
type Deps struct {
	DB *sql.DB
	TZ *time.Location
}

// Handler is the v2 HTTP handler registered onto the /api/v2 gin group.
type Handler struct {
	db *sql.DB
	tz *time.Location
}

// New returns a Handler wired with the supplied deps.
func New(d Deps) *Handler {
	return &Handler{db: d.DB, tz: d.TZ}
}

// timeInTZ formats a Postgres timestamp string in the user's timezone.
func (h *Handler) timeInTZ(s string) string {
	return timefmt.GetTimeInTimeZone(s, h.tz)
}

// parseDate normalises a user-supplied filter into the DB timestamp format.
func (h *Handler) parseDate(s string) (string, error) {
	return timefmt.ParseDateParam(s, h.tz)
}
