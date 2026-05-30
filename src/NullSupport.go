package main

// based on Gist:
//   https://gist.github.com/rsudip90/022c4ef5d98130a224c9239e0a1ab397

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// NullInt64 is an alias for sql.NullInt64 data type
type NullInt64 struct {
	sql.NullInt64
}

// MarshalJSON for NullInt64. Value receiver so the JSON encoder can invoke
// it on non-addressable struct fields (e.g., when the containing struct is
// passed by value through several layers of response wrappers).
func (ni NullInt64) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ni.Int64)
}

// NullBool is an alias for sql.NullBool data type
type NullBool struct {
	sql.NullBool
}

// MarshalJSON for NullBool. Value receiver — see [NullInt64.MarshalJSON].
func (nb NullBool) MarshalJSON() ([]byte, error) {
	if !nb.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nb.Bool)
}

// NullFloat64 is an alias for sql.NullFloat64 data type
type NullFloat64 struct {
	sql.NullFloat64
}

// MarshalJSON for NullFloat64. Value receiver — see [NullInt64.MarshalJSON].
func (nf NullFloat64) MarshalJSON() ([]byte, error) {
	if !nf.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nf.Float64)
}

type NullString string

func (s *NullString) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = NullString(v)
	case []byte:
		*s = NullString(v)
	case time.Time:
		// pq returns time.Time for timestamp / timestamptz columns; format
		// matches `dbTimestampFormat` so getTimeInTimeZone can re-parse it.
		*s = NullString(v.UTC().Format(dbTimestampFormat))
	default:
		return errors.New("value is not a string")
	}
	return nil
}

func (s NullString) Value() (driver.Value, error) {
	if len(s) == 0 { // if nil or empty string
		return nil, nil
	}
	return string(s), nil
}
