#!/usr/bin/env bash
# Run teslamateapi against the dev Postgres container started by
# docker-compose.yml. ENCRYPTION_KEY / API_TOKEN are throwaway dev values —
# never reuse them outside this script.
set -euo pipefail

cd "$(dirname "$0")/../src"

# Regenerate the embedded OpenAPI spec from in-source annotations before
# compiling. openapi_handler.go //go:embed docs/swagger.yaml — without this
# step, edits to handler annotations won't show up at /api/docs.
if ! command -v swag >/dev/null 2>&1; then
  echo "[run-api] installing swag CLI (one-time)..."
  go install github.com/swaggo/swag/cmd/swag@v1.16.4
fi
"$(go env GOPATH)/bin/swag" init -g webserver.go -o docs --outputTypes go,yaml,json --parseDependency --parseInternal >/dev/null

export DATABASE_HOST=127.0.0.1
export DATABASE_PORT=55432
export DATABASE_USER=teslamate
export DATABASE_PASS=secret
export DATABASE_NAME=teslamate
export DATABASE_SSL=disable
export ENCRYPTION_KEY=devkey-not-for-prod-XXXXXXXXXXXX
export DISABLE_MQTT=true
export API_TOKEN_DISABLE=true
export TZ=Asia/Shanghai
export DEBUG_MODE=true
export GIN_MODE=release

exec go run ./...
