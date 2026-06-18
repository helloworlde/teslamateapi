package database

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"
)

// SlowQueryThreshold is the latency above which queries are logged at warn
// level. Everything faster stays silent — the gin access log already records
// per-request latency, so only genuinely slow DB work is worth surfacing.
const SlowQueryThreshold = 100 * time.Millisecond

// DB wraps *sql.DB and logs any query slower than SlowQueryThreshold at warn
// level. The embedded *sql.DB keeps the full connection surface (Ping, Close,
// Begin, stats, …) available to callers that need it.
type DB struct {
	*sql.DB
}

// Wrap returns a DB that times QueryContext/QueryRowContext and warns on slow
// queries. Pass the raw *sql.DB elsewhere (e.g. metrics) unchanged.
func Wrap(db *sql.DB) *DB { return &DB{DB: db} }

// QueryContext runs the query and logs it if it exceeds SlowQueryThreshold.
func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := d.DB.QueryContext(ctx, query, args...)
	logSlow(start, query)
	return rows, err
}

// QueryRowContext runs the single-row query and logs it if it exceeds
// SlowQueryThreshold. database/sql executes the query eagerly here, so the
// measured duration reflects real work even though Scan happens later.
func (d *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := d.DB.QueryRowContext(ctx, query, args...)
	logSlow(start, query)
	return row
}

// logSlow emits a single warn line for queries past the threshold.
func logSlow(start time.Time, query string) {
	elapsed := time.Since(start)
	if elapsed < SlowQueryThreshold {
		return
	}
	log.Printf("[warn] slow query (%s): %s", elapsed.Round(time.Millisecond), summarizeQuery(query))
}

// summarizeQuery collapses a multi-line SQL statement onto one line and
// truncates it so a slow-query line stays grep-friendly.
func summarizeQuery(query string) string {
	s := strings.Join(strings.Fields(query), " ")
	const max = 300
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}
