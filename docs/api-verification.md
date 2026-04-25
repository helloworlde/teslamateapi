# API verification

## Static verification

Commands executed successfully in this environment:

- `go test ./...`
- `go vet ./...`
- `swag init -g swagger_info.go -d src -o src/docs`

## Runtime verification

Service start command used:

```bash
TZ=Asia/Shanghai \
ENCRYPTION_KEY=<redacted> \
DATABASE_USER=<redacted> \
DATABASE_PASS=<redacted> \
DATABASE_NAME=<redacted> \
DATABASE_HOST=<redacted> \
DATABASE_PORT=<redacted> \
MQTT_HOST=<redacted> \
go run ./src
```

Observed startup result:

- Service started successfully on `:8080`
- Postgres connection succeeded
- MQTT connection succeeded and subscribed to `teslamate/cars/#`

## HTTP verification script

Command executed:

```bash
BASE_URL=http://localhost:8080 CAR_ID=1 ./scripts/verify-api.sh
```

Observed result:

- Health endpoints passed: `/api/ping`, `/api/healthz`, `/api/readyz`
- Docs endpoint passed: `/api/v1/docs/openapi.json`
- All redesigned endpoints passed these checks:
  - default request
  - request with `startDate` / `endDate`
  - empty-result time range
  - invalid date request
  - non-existent `CarID`
- Script finished with: `all verification checks completed`

## Additional notes

- Date parsing now accepts RFC3339, timezone-offset strings, decoded-space offsets, local datetime, and date-only values.
- Non-existent `CarID` verification uses an in-range missing ID to avoid Postgres smallint overflow and correctly exercise 404 handling.
- `vampire-drain` deliberately returns an explicit empty structure with limitations metadata instead of guessed values.
