# API redesign audit and plan

## Scope

This document audits the extension API surface that existed before the redesign and records the new route layout now implemented in this repository.

## Route inventory

### Original compatible API

These routes are preserved and remain registered exactly as compatibility guarantees require:

- `GET /api`
- `GET /api/v1`
- `GET /api/v1/cars`
- `GET /api/v1/cars/:CarID`
- `GET /api/v1/cars/:CarID/battery-health`
- `GET /api/v1/cars/:CarID/charges`
- `GET /api/v1/cars/:CarID/charges/current`
- `GET /api/v1/cars/:CarID/charges/:ChargeID`
- `GET /api/v1/cars/:CarID/drives`
- `GET /api/v1/cars/:CarID/drives/:DriveID`
- `GET /api/v1/cars/:CarID/status`
- `GET /api/v1/cars/:CarID/updates`
- `GET /api/v1/globalsettings`
- `GET /api/healthz`
- `GET /api/ping`
- `GET /api/readyz`

The compatible command routes are preserved only when command execution is explicitly enabled with `ENABLE_COMMANDS=true`; otherwise they are not mounted at all:

- `GET /api/v1/cars/:CarID/command`
- `POST /api/v1/cars/:CarID/command/:Command`
- `GET /api/v1/cars/:CarID/logging`
- `PUT /api/v1/cars/:CarID/logging/:Command`
- `POST /api/v1/cars/:CarID/wake_up`

### Documentation routes

- `GET /api/v1/docs`
- `GET /api/v1/docs/openapi.json`
- `GET /api/v1/docs/swagger`
- `GET /api/v1/docs/swagger/index.html`
- `GET /api/v1/docs/swagger/doc.json`

### Legacy extension routes removed

These routes were present before the redesign and are no longer registered:

- `GET /api/v1/summaries/options`
- `GET /api/v1/cars/:CarID/parking-sessions`
- `GET /api/v1/cars/:CarID/summaries`
- `GET /api/v1/cars/:CarID/summaries/overview`
- `GET /api/v1/cars/:CarID/summaries/lifetime`
- `GET /api/v1/cars/:CarID/summaries/drives`
- `GET /api/v1/cars/:CarID/summaries/charges`
- `GET /api/v1/cars/:CarID/summaries/parking`
- `GET /api/v1/cars/:CarID/summaries/statistics`
- `GET /api/v1/cars/:CarID/summaries/state-activity`
- `GET /api/v1/cars/:CarID/activity-timeline`
- `GET /api/v1/cars/:CarID/dashboards/drives`
- `GET /api/v1/cars/:CarID/dashboards/charges`
- `GET /api/v1/cars/:CarID/calendars/drives`
- `GET /api/v1/cars/:CarID/charts/efficiency`
- `GET /api/v1/cars/:CarID/charts/drives/monthly-distance`
- `GET /api/v1/cars/:CarID/charts/drives/weekday-distance`
- `GET /api/v1/cars/:CarID/charts/drives/hourly-starts`
- `GET /api/v1/cars/:CarID/charts/charges/monthly-energy`
- `GET /api/v1/cars/:CarID/charts/charges/location-energy`
- `GET /api/v1/cars/:CarID/charts/charges/weekday-energy`
- `GET /api/v1/cars/:CarID/charts/charges/hourly-starts`
- `GET /api/v1/cars/:CarID/charts/activity/duration`
- `GET /api/v1/cars/:CarID/charges/:ChargeID/interval`

### Redesigned extension routes

- `GET /api/v1/cars/:CarID/summary`
- `GET /api/v1/cars/:CarID/dashboard`
- `GET /api/v1/cars/:CarID/calendar`
- `GET /api/v1/cars/:CarID/statistics`
- `GET /api/v1/cars/:CarID/series/drives`
- `GET /api/v1/cars/:CarID/series/charges`
- `GET /api/v1/cars/:CarID/series/battery`
- `GET /api/v1/cars/:CarID/series/states`
- `GET /api/v1/cars/:CarID/distributions/drives`
- `GET /api/v1/cars/:CarID/distributions/charges`
- `GET /api/v1/cars/:CarID/insights`
- `GET /api/v1/cars/:CarID/timeline`
- `GET /api/v1/cars/:CarID/map/visited`
- `GET /api/v1/cars/:CarID/locations`

### OpenAPI response models

The redesigned extension routes use concrete, business-named Swagger models instead of a generic object envelope:

- `summary` -> `SummaryV2Envelope`
- `dashboard` -> `DashboardV2Envelope`
- `calendar` -> `CalendarV2Envelope`
- `statistics` -> `StatisticsV2Envelope`
- `series/*` -> `SeriesV2Envelope`
- `distributions/*` -> `DistributionsV2Envelope`
- `insights` -> `InsightsV2Envelope`
- `timeline` -> `TimelineV2Envelope`
- `map/visited` -> `VisitedMapV2Envelope`

## Audit findings

### REST and naming

- `summaries/*` and `dashboards/*` overlapped heavily in meaning.
- `activity-timeline` and `timeline` represented the same conceptual resource; the dashboard-specific name was removed.
- Calendar data was normalized into one `calendar` resource with query-driven buckets and optional metrics, instead of separate drive/charge calendar aliases.
- Chart endpoints were renamed to stable nouns and metric names instead of implementation-specific aliases like `monthly-distance`.

### Third-party app suitability

- Old summary and dashboard routes forced clients to know which alias carried which subset of metrics.
- The redesign separates summary/statistics/domain-specific series/domain-specific distributions/timeline concerns and provides render-friendly chart series.
- New list endpoints expose consistent `limit`, `offset`, and `total` pagination fields.

### SQL and filtering findings

- Date parsing previously rejected offset values after URL decoding converted `+` to a space.
- Several chart aliases encoded bucket semantics in the path instead of query parameters; bucket selection is now explicit.
- Query helpers now consistently apply `car_id` and timezone-aware half-open date filters in the database layer used by the redesigned routes.
- Expensive derived calculations such as parking energy / vampire drain and regeneration are bounded with query timeouts and return warnings instead of blocking aggregate responses indefinitely.

### Units and empty data

- Redesigned object responses include stable `unit` metadata.
- Redesigned chart responses return empty `series` arrays instead of `null`.
- Redesigned list responses return empty `data` arrays and total-aware pagination.

## Keep / merge / delete / rename plan

### Keep

- All original compatible routes listed above.
- Documentation and health-check routes.
- Existing raw drive/charge detail routes for backward compatibility.

### Merge

- `summaries/overview`, `summaries/drives`, `summaries/charges`, `summaries/parking`, and `summaries/state-activity` into `summary` and `statistics`.
- `dashboards/drives` and `dashboards/charges` into `summary`, `statistics`, `series`, and `distributions`.
- `activity-timeline` into `timeline`.

### Delete

- Low-value aliases listed in “Legacy extension routes removed”.

### Rename

- `calendars/*` -> unified `calendar`
- bucket-specific chart aliases -> unified `series` / `distributions`
- fragmented analytics aliases -> unified `summary`, `statistics`, `insights`

### Add

- `summary`
- `dashboard`
- `statistics`
- `series`
- `distributions`
- `insights`
- `timeline`
- `calendar`
- `map/visited`
- `locations`

## Compatibility strategy

- Original compatible routes are still registered and keep their paths and response shapes.
- Extension routes are treated as redesignable and may change when consolidating duplicate semantics.
- OpenAPI clearly marks the extension surface as redesigned and potentially breaking.

## Data definitions and limits

- `startDate` / `endDate` accept RFC3339, timezone offset, decoded-space offset, local datetime, and date-only formats.
- Date-only `endDate` values are expanded to local end-of-day for redesigned range parsing, then translated into half-open SQL filters.
- Series and distribution endpoints default to bounded time windows to avoid unbounded scans.
- `map/visited` limits points and reports truncation in metadata.
- `vampire_drain` is derived from state windows and battery/range samples where available; expensive queries are time-boxed and return `null` plus warnings when the data is unavailable or too slow.
- Distances use TeslaMate settings-derived `km` or `mi`; speeds use `km/h` or `mi/h`; consumption uses `Wh/km` or `Wh/mi`; energy uses `kWh`.
