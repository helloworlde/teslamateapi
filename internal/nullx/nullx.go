// Package nullx provides null-safe SQL types that JSON-marshal to null when invalid.
//
// Based on https://gist.github.com/rsudip90/022c4ef5d98130a224c9239e0a1ab397
package nullx

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Int64 wraps sql.NullInt64 with null-safe JSON marshalling.
//
// @name NullInt64
type Int64 struct {
	sql.NullInt64
}

// MarshalJSON for Int64.
func (ni *Int64) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ni.Int64)
}

// Bool wraps sql.NullBool with null-safe JSON marshalling.
//
// @name NullBool
type Bool struct {
	sql.NullBool
}

// MarshalJSON for Bool.
func (nb *Bool) MarshalJSON() ([]byte, error) {
	if !nb.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nb.Bool)
}

// Float64 wraps sql.NullFloat64 with null-safe JSON marshalling.
//
// @name NullFloat64
type Float64 struct {
	sql.NullFloat64
}

// MarshalJSON for Float64.
func (nf *Float64) MarshalJSON() ([]byte, error) {
	if !nf.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nf.Float64)
}

// String is a sql.Scanner / driver.Valuer compatible string type that
// scans NULLs into the empty string and writes empty strings as NULLs.
type String string

// Scan implements sql.Scanner.
//
// In addition to plain strings the scanner accepts byte slices, time.Time
// values (formatted as RFC3339) and falls back to fmt.Sprintf for any other
// type so that timestamp columns can be safely scanned without losing data.
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
		*s = String(v.Format(time.RFC3339))
	default:
		*s = String(fmt.Sprintf("%v", v))
	}
	return nil
}

// Value implements driver.Valuer.
func (s String) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return string(s), nil
}
