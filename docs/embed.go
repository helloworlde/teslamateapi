// Package docs is the swag-generated OpenAPI bundle. The Spec variable
// below embeds the YAML so external packages can serve it without
// re-reading from disk; see internal/httpapi/handlers/system/docs.go.
package docs

import _ "embed"

// Spec is the swag-generated OpenAPI 3.x YAML embedded at build time.
// `swag init` writes swagger.yaml here; this file embeds it for runtime
// serving by the /api/openapi.yaml and /api/docs handlers.
//
//go:embed swagger.yaml
var Spec []byte
