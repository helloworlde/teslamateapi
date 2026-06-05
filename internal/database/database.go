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
