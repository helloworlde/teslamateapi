// Package nullable holds the database/sql null-aware scalar wrappers used
// across teslamateapi. They live in their own package (instead of the main
// binary package) so the DTO subpackage can import them without creating
// an import cycle through the main package.
//
// The behavior matches the original NullSupport.go drop-in: every wrapper
// MarshalJSONs to its scalar value or `null` when invalid, and NullString
// supports pq's text / []byte / time.Time scan paths.
package nullable

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// DBTimestampFormat is the layout pq returns timestamp / timestamptz columns
// in. NullString.Scan re-formats time.Time values into this layout so callers
// can re-parse them with time.Parse later.
const DBTimestampFormat = "2006-01-02T15:04:05Z"

// Int64 is an alias for sql.NullInt64. JSON-encodes as the int (or null).
type Int64 struct {
	sql.NullInt64
}

// MarshalJSON encodes Int64 as its underlying int64 or `null` when invalid.
func (ni Int64) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ni.Int64)
}

// Bool is an alias for sql.NullBool. JSON-encodes as the bool (or null).
type Bool struct {
	sql.NullBool
}

// MarshalJSON encodes Bool as its underlying bool or `null` when invalid.
func (nb Bool) MarshalJSON() ([]byte, error) {
	if !nb.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nb.Bool)
}

// Float64 is an alias for sql.NullFloat64. JSON-encodes as the float (or null).
type Float64 struct {
	sql.NullFloat64
}

// MarshalJSON encodes Float64 as its underlying float64 or `null` when invalid.
func (nf Float64) MarshalJSON() ([]byte, error) {
	if !nf.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nf.Float64)
}

// String is a nullable string that scans NULL → "" and time.Time →
// DBTimestampFormat. JSON-encodes as the string (empty strings remain "").
type String string

// Scan implements the sql.Scanner interface.
func (s *String) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = String(v)
	case []byte:
		*s = String(v)
	case time.Time:
		*s = String(v.UTC().Format(DBTimestampFormat))
	default:
		return errors.New("value is not a string")
	}
	return nil
}

// Value implements the driver.Valuer interface.
func (s String) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return string(s), nil
}
