# TeslaMateApi

[![GitHub CI](https://img.shields.io/github/actions/workflow/status/tobiasehlert/teslamateapi/build.yml?branch=main&logo=github)](https://github.com/tobiasehlert/teslamateapi/actions/workflows/build.yml)
[![GitHub go.mod version](https://img.shields.io/github/go-mod/go-version/tobiasehlert/teslamateapi?logo=go)](https://github.com/tobiasehlert/teslamateapi/blob/main/go.mod)
[![GitHub release](https://img.shields.io/github/v/release/tobiasehlert/teslamateapi?sort=semver&logo=github)](https://github.com/tobiasehlert/teslamateapi/releases)
[![Docker image size (tag)](https://img.shields.io/docker/image-size/tobiasehlert/teslamateapi/latest?logo=docker)](https://hub.docker.com/r/tobiasehlert/teslamateapi)
[![GitHub license](https://img.shields.io/github/license/tobiasehlert/teslamateapi)](https://github.com/tobiasehlert/teslamateapi/blob/main/LICENSE)
[![Docker pulls](https://img.shields.io/docker/pulls/tobiasehlert/teslamateapi)](https://hub.docker.com/r/tobiasehlert/teslamateapi)

TeslaMateApi is a RESTful API to get data collected by self-hosted data logger **[TeslaMate](https://github.com/teslamate-org/teslamate)** in JSON.

- Written in **[Golang](https://golang.org/)**
- Data is collected from TeslaMate **Postgres** database and local **MQTT** Broker
- Endpoints return data in JSON format
- Send commands to your Tesla through the TeslaMateApi

### Table of Contents

- [How to use](#how-to-use)
  - [Docker-compose](#docker-compose)
  - [Environment variables](#environment-variables)
- [API documentation](#api-documentation)
  - [Available endpoints](#available-endpoints)
  - [Authentication](#authentication)
  - [Commands](#commands)
- [Security information](#security-information)
- [Credits](#credits)

## How to use

You can either use it in a Docker container or go download the code and deploy it yourself on any server.

### Docker-compose

If you run the simple Docker deployment of TeslaMate, then adding this will do the trick. You'll have TeslaMateApi exposed at port 8080 locally then.

```yaml
services:
  teslamateapi:
    image: tobiasehlert/teslamateapi:latest
    restart: always
    depends_on:
      - database
    environment:
      - ENCRYPTION_KEY=MySuperSecretEncryptionKey
      - DATABASE_USER=teslamate
      - DATABASE_PASS=secret
      - DATABASE_NAME=teslamate
      - DATABASE_HOST=database
      - MQTT_HOST=mosquitto
      - TZ=Europe/Berlin
    ports:
      - 8080:8080
```

If you are using TeslaMate Traefik setup in Docker with environment variables file (.env), then you can simply add this section to the `services:` section of the `docker-compose.yml` file:

```yaml
services:
  teslamateapi:
    image: tobiasehlert/teslamateapi:latest
    restart: always
    depends_on:
      - database
    environment:
      - ENCRYPTION_KEY=${TM_ENCRYPTION_KEY}
      - DATABASE_USER=${TM_DB_USER}
      - DATABASE_PASS=${TM_DB_PASS}
      - DATABASE_NAME=${TM_DB_NAME}
      - DATABASE_HOST=database
      - MQTT_HOST=mosquitto
      - TZ=${TM_TZ}
    labels:
      - "traefik.enable=true"
      - "traefik.port=8080"
      - "traefik.http.middlewares.redirect.redirectscheme.scheme=https"
      - "traefik.http.middlewares.teslamateapi-auth.basicauth.realm=teslamateapi"
      - "traefik.http.middlewares.teslamateapi-auth.basicauth.usersfile=/auth/.htpasswd"
      - "traefik.http.routers.teslamateapi-insecure.rule=Host(`${FQDN_TM}`)"
      - "traefik.http.routers.teslamateapi-insecure.middlewares=redirect"
      - "traefik.http.routers.teslamateapi.rule=Host(`${FQDN_TM}`) && (Path(`/api`) || PathPrefix(`/api/`))"
      - "traefik.http.routers.teslamateapi.entrypoints=websecure"
      - "traefik.http.routers.teslamateapi.middlewares=teslamateapi-auth"
      - "traefik.http.routers.teslamateapi.tls.certresolver=tmhttpchallenge"
```

In this case, the TeslaMateApi would be accessible at teslamate.example.com/api/

### Environment variables

Basically the same environment variables for the database, mqqt and timezone need to be set for TeslaMateApi as you have for TeslaMate.

**Required** environment variables (even if there are some default values available)

| Variable           | Type   | Default         |
| ------------------ | ------ | --------------- |
| **DATABASE_USER**  | string | _teslamate_     |
| **DATABASE_PASS**  | string | _secret_        |
| **DATABASE_NAME**  | string | _teslamate_     |
| **DATABASE_HOST**  | string | _database_      |
| **ENCRYPTION_KEY** | string |                 |
| **MQTT_HOST**      | string | _mosquitto_     |
| **TZ**             | string | _Europe/Berlin_ |

**Optional** environment variables

| Variable                      | Type    | Default                       |
| ----------------------------- | ------- | ----------------------------- |
| **TESLAMATE_SSL**             | boolean | _false_                       |
| **TESLAMATE_HOST**            | string  | _teslamate_                   |
| **TESLAMATE_PORT**            | string  | _4000_                        |
| **API_TOKEN**                 | string  |                               |
| **API_TOKEN_DISABLE**         | string  | _false_                       |
| **DATABASE_PORT**             | integer | _5432_                        |
| **DATABASE_TIMEOUT**          | integer | _60000_                       |
| **DATABASE_SSL**              | string  | _disable_                     |
| **DATABASE_SSL_CA_CERT_FILE** | string  |                               |
| **DEBUG_MODE**                | boolean | _false_                       |
| **TESLAMATEAPI_LISTEN_ADDR**  | string  | _:8080_                       |
| **DISABLE_MQTT**              | boolean | _false_                       |
| **MQTT_TLS**                  | boolean | _false_                       |
| **MQTT_PORT**                 | integer | _1883 (if TLS is true: 8883)_ |
| **MQTT_USERNAME**             | string  |                               |
| **MQTT_PASSWORD**             | string  |                               |
| **MQTT_NAMESPACE**            | string  |                               |
| **MQTT_CLIENTID**             | string  | _4 char random string_        |
| **TESLA_API_HOST**            | string  | _retrieved by access token_   |

**Commands** environment variables

Command routes are not registered unless `ENABLE_COMMANDS=true` is set explicitly. With the default configuration, `/command`, `/logging`, and `/wake_up` command endpoints return 404 because they are not mounted.

| Variable                    | Type    | Default           |
| --------------------------- | ------- | ----------------- |
| **ENABLE_COMMANDS**         | boolean | _false_           |
| **COMMANDS_ALL**            | boolean | _false_           |
| **COMMANDS_ALLOWLIST**      | string  | _allow_list.json_ |
| **COMMANDS_LOGGING**        | boolean | _false_           |
| **COMMANDS_WAKE**           | boolean | _false_           |
| **COMMANDS_ALERT**          | boolean | _false_           |
| **COMMANDS_REMOTESTART**    | boolean | _false_           |
| **COMMANDS_HOMELINK**       | boolean | _false_           |
| **COMMANDS_SPEEDLIMIT**     | boolean | _false_           |
| **COMMANDS_VALET**          | boolean | _false_           |
| **COMMANDS_SENTRYMODE**     | boolean | _false_           |
| **COMMANDS_DOORS**          | boolean | _false_           |
| **COMMANDS_TRUNK**          | boolean | _false_           |
| **COMMANDS_WINDOWS**        | boolean | _false_           |
| **COMMANDS_SUNROOF**        | boolean | _false_           |
| **COMMANDS_CHARGING**       | boolean | _false_           |
| **COMMANDS_CLIMATE**        | boolean | _false_           |
| **COMMANDS_MEDIA**          | boolean | _false_           |
| **COMMANDS_SHARING**        | boolean | _false_           |
| **COMMANDS_SOFTWAREUPDATE** | boolean | _false_           |
| **COMMANDS_UNKNOWN**        | boolean | _false_           |

## API documentation

Interactive API docs are auto-generated from Go annotation comments; HTML is produced with [scalar-go](https://github.com/watchakorn-18k/scalar-go) and [Scalar](https://github.com/scalar/scalar).

- Scalar API Reference: `/api/v1/docs` or `/api/v1/docs/swagger/index.html`
- OpenAPI JSON: `/api/v1/docs/openapi.json`
- Legacy JSON alias: `/api/v1/docs/swagger/doc.json`
- `/api/v1/docs/swagger` redirects to `/api/v1/docs/swagger/index.html`
- Redesigned extension endpoints use concrete response models such as `SummaryV2Envelope`, `DashboardV2Envelope`, `SeriesV2Envelope`, and `LocationsV2Envelope`.

Regenerate OpenAPI JSON after route or annotation changes:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g swagger_info.go -d src -o src/docs
```

Local run on another port (optional): set **`TESLAMATEAPI_LISTEN_ADDR=:18088`** (default **`:8080`**).

### Compatibility policy

- All original TeslaMateApi endpoints listed below remain compatible.
- All redesigned extension endpoints are now unified under `/api/v1/cars/:CarID/*`.
- Removed extension aliases include legacy `summaries/*`, `analytics/*`, `calendar/*`, `charts/*`, `insights/events`, and other fragmented chart endpoints.

### Date parameters

All redesigned extension endpoints accept:

- RFC3339: `2026-04-24T06:57:02Z`
- RFC3339 with offset: `2026-04-02T10:55:30+08:00`
- URL-decoded space offset: `2026-04-02T10:55:30 08:00`
- Local datetime: `2026-04-02 10:55:30`
- Date only: `2026-04-02`

When sending `+08:00` in query strings, prefer `%2B08:00`. The API also repairs the common case where `+` is decoded into a space.

### Data correctness check

Integration test against a real Postgres DB: compares `fetchDriveHistorySummary` / `fetchChargeHistorySummary` to independent SQL with the same filters.

```bash
export TESLAMATEAPI_DATACHECK=1
go test ./src -run TestDatacheckSummaryVsReferenceSQL -count=1 -v
```

Optional:

- `TESLAMATEAPI_DATACHECK_CAR_ID` (default `1`)
- `TESLAMATEAPI_DATACHECK_START_DATE`
- `TESLAMATEAPI_DATACHECK_END_DATE`
- `TESLAMATEAPI_ENDPOINT_CHECK=1` for redesigned endpoint integration tests

### Available endpoints

**System**

- GET `/api`
- GET `/api/v1`
- GET `/api/ping`
- GET `/api/healthz`
- GET `/api/readyz`
- GET `/api/v1/docs`
- GET `/api/v1/docs/openapi.json`
- GET `/api/v1/docs/swagger`
- GET `/api/v1/docs/swagger/index.html`
- GET `/api/v1/docs/swagger/doc.json`

**Compatible API**

- GET `/api/v1/cars`
- GET `/api/v1/cars/:CarID`
- GET `/api/v1/cars/:CarID/battery-health`
- GET `/api/v1/cars/:CarID/charges`
- GET `/api/v1/cars/:CarID/charges/current`
- GET `/api/v1/cars/:CarID/charges/:ChargeID`
- GET `/api/v1/cars/:CarID/drives`
- GET `/api/v1/cars/:CarID/drives/:DriveID`
- GET `/api/v1/cars/:CarID/status`
- GET `/api/v1/cars/:CarID/updates`
- GET `/api/v1/globalsettings`

**Compatible command API, registered only when `ENABLE_COMMANDS=true`**

- GET `/api/v1/cars/:CarID/command`
- POST `/api/v1/cars/:CarID/command/:Command`
- GET `/api/v1/cars/:CarID/logging`
- PUT `/api/v1/cars/:CarID/logging/:Command`
- POST `/api/v1/cars/:CarID/wake_up`

**Unified extension API**

- GET `/api/v1/cars/:CarID/summary`
- GET `/api/v1/cars/:CarID/dashboard`
- GET `/api/v1/cars/:CarID/calendar`
- GET `/api/v1/cars/:CarID/statistics`
- GET `/api/v1/cars/:CarID/series/drives`
- GET `/api/v1/cars/:CarID/series/charges`
- GET `/api/v1/cars/:CarID/series/battery`
- GET `/api/v1/cars/:CarID/series/states`
- GET `/api/v1/cars/:CarID/distributions/drives`
- GET `/api/v1/cars/:CarID/distributions/charges`
- GET `/api/v1/cars/:CarID/insights`
- GET `/api/v1/cars/:CarID/timeline`
- GET `/api/v1/cars/:CarID/map/visited`
- GET `/api/v1/cars/:CarID/locations`

### Extension API purpose

- `summary`: 规范化范围摘要（overview/driving/charging/parking/battery/efficiency/cost/quality/state）。
- `dashboard`: app 首页聚合数据，减少客户端请求次数。
- `calendar`: 日历维度聚合（day/week/month）。
- `statistics`: 年/月/周/自定义范围统计。
- `series/*`: 按领域拆分的时序曲线（drives/charges/battery/states）。
- `distributions/*`: 按领域拆分的分布图数据，bucket 固定排序并补齐 0 值。
- `insights`: 可扩展洞察事件与摘要。
- `timeline`: 时间线事件流（drive/charge/state）。
- `locations`: 地点聚合（驾驶起终点、充电地点、次数、能量、费用、坐标）。
- `map/visited`: 访问点、边界和热力图基础数据。

### Units and warnings

- 默认单位为 metric（km, km/h, kWh, Wh/km）。
- 所有时间字段使用 RFC3339。
- 无法可靠计算的字段返回 `null`，并通过 `warnings` 说明原因。
- 禁止返回伪造值或用 `0` 冒充未知值。

### Query examples

```bash
# 查询本月日历
curl "http://localhost:8080/api/v1/cars/1/calendar?startDate=2026-04-01&endDate=2026-04-30&bucket=day&timezone=Asia/Shanghai"

# 查询本月统计
curl "http://localhost:8080/api/v1/cars/1/statistics?period=month&date=2026-04-01&timezone=Asia/Shanghai"

# 查询本月里程和速度曲线
curl "http://localhost:8080/api/v1/cars/1/series/drives?metrics=distance,speed&bucket=day&startDate=2026-04-01&endDate=2026-04-30"

# 查询充电时间分布
curl "http://localhost:8080/api/v1/cars/1/distributions/charges?metrics=start_hour&startDate=2026-04-01&endDate=2026-04-30"

# 查询洞察
curl "http://localhost:8080/api/v1/cars/1/insights?startDate=2026-04-01&endDate=2026-04-30"
```

### Verification

- Static and unit validation: `go test ./...` and `go vet ./...`
- Redesign notes: `docs/api-redesign.md`

## Security information

There is **no** possibility to get access to your Tesla account tokens by this API and we'll keep it this way!

The data that is accessible is data like the cars, charges, drives, current status, updates and global settings.

Also, apply some authentication on your webserver in front of the container, so your data is not unprotected and too exposed. In the example above, we use the same .htpasswd file as used by TeslaMate.

If you have applied a level of authentication in front of the container `API_TOKEN_DISABLE=true` will allow commands without requiring the header or uri token value. But even then it's always rekommended to use an apikey.

## Credits

- Authors: Tobias Lindberg – [List of contributors](https://github.com/tobiasehlert/teslamateapi/graphs/contributors)
- Distributed under MIT License
