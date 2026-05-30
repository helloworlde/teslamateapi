package main

// The concrete null-aware scalar wrappers live in the nullable subpackage so
// the DTO package can reuse them without an import cycle through main. The
// aliases below preserve the legacy `main.NullString` / `main.NullInt64` /
// `main.NullBool` / `main.NullFloat64` names so existing handler code keeps
// compiling untouched.

import (
	"github.com/tobiasehlert/teslamateapi/src/nullable"
)

// NullInt64 is an alias for nullable.Int64.
type NullInt64 = nullable.Int64

// NullBool is an alias for nullable.Bool.
type NullBool = nullable.Bool

// NullFloat64 is an alias for nullable.Float64.
type NullFloat64 = nullable.Float64

// NullString is an alias for nullable.String.
type NullString = nullable.String
