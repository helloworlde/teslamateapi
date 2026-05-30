#!/usr/bin/env bash
# Run teslamateapi against the dev Postgres container started by
# docker-compose.yml. ENCRYPTION_KEY / API_TOKEN are throwaway dev values —
# never reuse them outside this script.
set -euo pipefail

cd "$(dirname "$0")/../src"

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
