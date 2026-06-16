// Package database opens the Postgres connection used by every handler.
package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/tobiasehlert/teslamateapi/internal/config"
)

// pqQuote escapes a value for libpq's KV connection-string format. Wraps
// every value in single quotes and backslash-escapes embedded ' and \ —
// safe even for values without specials.
func pqQuote(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(v) + "'"
}

// New opens and pings a Postgres connection using settings from cfg.
//
// Startup policy lives in main; this package only returns wrapped errors so
// tests and future callers can decide whether to retry, fail readiness, or exit.
func New(cfg config.Config) (*sql.DB, error) {
	dbsslmode := cfg.DBSSLMode
	switch dbsslmode {
	case "true", "noverify":
		dbsslmode = "require"
	case "false":
		dbsslmode = "disable"
	}

	dbtimeout := cfg.DBTimeoutMS / 1000

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		pqQuote(cfg.DBHost), cfg.DBPort, pqQuote(cfg.DBUser), pqQuote(cfg.DBPass),
		pqQuote(cfg.DBName), pqQuote(dbsslmode), dbtimeout,
	)

	// Disable JIT on every connection from this API (TM_DB_DISABLE_JIT, default on).
	//
	// The v2 stats endpoints (summary, energy-flow, time-distribution, …) build
	// large multi-CTE queries whose *estimated* cost easily clears jit_above_cost
	// (and often jit_optimize/inline_above_cost). Postgres then JIT-compiles the
	// many CASE / timezone expressions on every call. Compilation cost depends on
	// query complexity, not row count, so it is a roughly fixed multi-second tax
	// that dwarfs the actual work — e.g. the period-summary query measures ~85ms
	// of execution behind ~550ms+ of JIT compilation (multiple seconds on slower
	// hosts that also run the LLVM optimize/inline passes). None of these queries
	// run long enough for JIT to pay off, so it is off by default; set
	// TM_DB_DISABLE_JIT=false to restore Postgres' own jit setting.
	if cfg.DBDisableJIT {
		psqlInfo += " options=" + pqQuote("-c jit=off")
	}

	if cfg.DBSSLRootCert != "" {
		psqlInfo += " sslrootcert=" + pqQuote(cfg.DBSSLRootCert)
	}

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("database open: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database ping: %w", err)
	}
	if gin.IsDebugging() {
		log.Println("[debug] initDBconnection - database connection established successfully.")
	}
	return db, nil
}
