// Package apicommon hosts shared state, types, and helpers used across the v1
// and v2 API packages: the global *sql.DB handle, the configured user
// timezone, and small response/time helpers.
package apicommon

import (
	"database/sql"
	"time"
)

// DB is the process-wide *sql.DB handle. It must be initialised by the
// server bootstrap (internal/server) before any handler runs.
var DB *sql.DB

// AppUsersTimezone is the *time.Location parsed from the TZ env var.
// Initialised by the server bootstrap.
var AppUsersTimezone *time.Location
