#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
CAR_ID="${CAR_ID:-1}"
EMPTY_CAR_ID="${EMPTY_CAR_ID:-999}"
START_DATE="${START_DATE:-2026-04-02T10:55:30%2B08:00}"
END_DATE="${END_DATE:-2026-04-24T06:57:02Z}"
EMPTY_START_DATE="${EMPTY_START_DATE:-2035-01-01T00:00:00Z}"
EMPTY_END_DATE="${EMPTY_END_DATE:-2035-01-02T00:00:00Z}"
INVALID_DATE="${INVALID_DATE:-not-a-date}"

RED='\033[31m'
GREEN='\033[32m'
YELLOW='\033[33m'
BLUE='\033[34m'
RESET='\033[0m'

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

failures=0

log() {
  printf "%b[%s]%b %s\n" "$BLUE" "verify" "$RESET" "$1"
}

pass() {
  printf "%b[PASS]%b %s\n" "$GREEN" "$RESET" "$1"
}

warn() {
  printf "%b[WARN]%b %s\n" "$YELLOW" "$RESET" "$1"
}

fail() {
  printf "%b[FAIL]%b %s\n" "$RED" "$RESET" "$1"
  failures=$((failures + 1))
}

request() {
  local label="$1"
  local path="$2"
  local expect_class="${3:-2}"
  local tmp_body
  tmp_body="$(mktemp)"
  local code
  code="$(curl -sS -o "$tmp_body" -w "%{http_code}" "${BASE_URL}${path}")" || code="000"
  printf "%s -> HTTP %s\n" "$label" "$code"
  if [[ "$code" != ${expect_class}* ]]; then
    fail "$label expected ${expect_class}xx got ${code}"
    cat "$tmp_body"
    rm -f "$tmp_body"
    return
  fi
  if ! jq . < "$tmp_body" >/dev/null 2>&1; then
    fail "$label returned invalid JSON"
    cat "$tmp_body"
    rm -f "$tmp_body"
    return
  fi
  pass "$label"
  rm -f "$tmp_body"
}

resolve_first_id() {
  local path="$1"
  local jq_filter="$2"
  local fallback="$3"
  local value
  value="$(curl -sS "${BASE_URL}${path}" | jq -r "$jq_filter // empty" 2>/dev/null || true)"
  if [[ -z "$value" || "$value" == "null" ]]; then
    echo "$fallback"
  else
    echo "$value"
  fi
}

log "checking health and docs"
request "ping" "/api/ping"
request "healthz" "/api/healthz"
request "readyz" "/api/readyz" 2
request "openapi" "/api/v1/docs/openapi.json"

DRIVE_ID="$(resolve_first_id "/api/v1/cars/${CAR_ID}/drives?show=1" '.data.drives[0].drive_id' 1)"
CHARGE_ID="$(resolve_first_id "/api/v1/cars/${CAR_ID}/charges?show=1" '.data.charges[0].charge_id' 1)"

ENDPOINTS=(
  "/api/v1/cars/${CAR_ID}/summary"
  "/api/v1/cars/${CAR_ID}/statistics"
  "/api/v1/cars/${CAR_ID}/charts/overview"
  "/api/v1/cars/${CAR_ID}/charts/drives/distance?bucket=month"
  "/api/v1/cars/${CAR_ID}/charts/drives/energy?bucket=month"
  "/api/v1/cars/${CAR_ID}/charts/drives/efficiency?bucket=month"
  "/api/v1/cars/${CAR_ID}/charts/drives/speed?bucket=month"
  "/api/v1/cars/${CAR_ID}/charts/drives/temperature?bucket=month"
  "/api/v1/cars/${CAR_ID}/charts/charges/energy?bucket=month"
  "/api/v1/cars/${CAR_ID}/charts/charges/cost?bucket=month"
  "/api/v1/cars/${CAR_ID}/charts/charges/efficiency?bucket=month"
  "/api/v1/cars/${CAR_ID}/charts/charges/power?bucket=month"
  "/api/v1/cars/${CAR_ID}/charts/charges/location"
  "/api/v1/cars/${CAR_ID}/charts/charges/soc"
  "/api/v1/cars/${CAR_ID}/charts/battery/range"
  "/api/v1/cars/${CAR_ID}/charts/battery/health"
  "/api/v1/cars/${CAR_ID}/charts/states/duration"
  "/api/v1/cars/${CAR_ID}/charts/vampire-drain"
  "/api/v1/cars/${CAR_ID}/charts/mileage"
  "/api/v1/cars/${CAR_ID}/drives/${DRIVE_ID}/details"
  "/api/v1/cars/${CAR_ID}/charges/${CHARGE_ID}/details"
  "/api/v1/cars/${CAR_ID}/timeline"
  "/api/v1/cars/${CAR_ID}/calendar/drives"
  "/api/v1/cars/${CAR_ID}/calendar/charges"
  "/api/v1/cars/${CAR_ID}/map/visited"
  "/api/v1/cars/${CAR_ID}/insights"
  "/api/v1/cars/${CAR_ID}/insights/events"
  "/api/v1/cars/${CAR_ID}/analytics/activity"
  "/api/v1/cars/${CAR_ID}/analytics/regeneration"
)

for path in "${ENDPOINTS[@]}"; do
  request "default ${path}" "$path"
  separator='?'
  if [[ "$path" == *\?* ]]; then
    separator='&'
  fi
  request "range ${path}" "${path}${separator}startDate=${START_DATE}&endDate=${END_DATE}"
  request "empty ${path}" "${path}${separator}startDate=${EMPTY_START_DATE}&endDate=${EMPTY_END_DATE}"
  request "invalid-date ${path}" "${path}${separator}startDate=${INVALID_DATE}" 4
  request "missing-car ${path}" "$(echo "$path" | sed "s#/cars/${CAR_ID}/#/cars/${EMPTY_CAR_ID}/#")" 4
  echo "---"
done

if [[ "$failures" -gt 0 ]]; then
  fail "verification finished with ${failures} failure(s)"
  exit 1
fi

pass "all verification checks completed"
