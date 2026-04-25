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
- `GET /api/v1/cars/:CarID/command`
- `POST /api/v1/cars/:CarID/command/:Command`
- `GET /api/v1/cars/:CarID/drives`
- `GET /api/v1/cars/:CarID/drives/:DriveID`
- `PUT /api/v1/cars/:CarID/logging/:Command`
- `GET /api/v1/cars/:CarID/logging`
- `GET /api/v1/cars/:CarID/status`
- `GET /api/v1/cars/:CarID/updates`
- `POST /api/v1/cars/:CarID/wake_up`
- `GET /api/v1/globalsettings`
- `GET /api/healthz`
- `GET /api/ping`
- `GET /api/readyz`

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
- `GET /api/v1/cars/:CarID/statistics`
- `GET /api/v1/cars/:CarID/charts/overview`
- `GET /api/v1/cars/:CarID/charts/drives/distance`
- `GET /api/v1/cars/:CarID/charts/drives/energy`
- `GET /api/v1/cars/:CarID/charts/drives/efficiency`
- `GET /api/v1/cars/:CarID/charts/drives/speed`
- `GET /api/v1/cars/:CarID/charts/drives/temperature`
- `GET /api/v1/cars/:CarID/charts/charges/energy`
- `GET /api/v1/cars/:CarID/charts/charges/cost`
- `GET /api/v1/cars/:CarID/charts/charges/efficiency`
- `GET /api/v1/cars/:CarID/charts/charges/power`
- `GET /api/v1/cars/:CarID/charts/charges/location`
- `GET /api/v1/cars/:CarID/charts/charges/soc`
- `GET /api/v1/cars/:CarID/charts/battery/range`
- `GET /api/v1/cars/:CarID/charts/battery/health`
- `GET /api/v1/cars/:CarID/charts/states/duration`
- `GET /api/v1/cars/:CarID/charts/vampire-drain`
- `GET /api/v1/cars/:CarID/charts/mileage`
- `GET /api/v1/cars/:CarID/drives/:DriveID/details`
- `GET /api/v1/cars/:CarID/charges/:ChargeID/details`
- `GET /api/v1/cars/:CarID/timeline`
- `GET /api/v1/cars/:CarID/calendar/drives`
- `GET /api/v1/cars/:CarID/calendar/charges`
- `GET /api/v1/cars/:CarID/map/visited`
- `GET /api/v1/cars/:CarID/insights`
- `GET /api/v1/cars/:CarID/insights/events`
- `GET /api/v1/cars/:CarID/analytics/activity`
- `GET /api/v1/cars/:CarID/analytics/regeneration`

## Audit findings

### REST and naming

- `summaries/*` and `dashboards/*` overlapped heavily in meaning.
- `activity-timeline` and `timeline` represented the same conceptual resource; the dashboard-specific name was removed.
- Calendar naming was normalized from `calendars/drives` to `calendar/drives` and extended with `calendar/charges`.
- Chart endpoints were renamed to stable nouns and metric names instead of implementation-specific aliases like `monthly-distance`.

### Third-party app suitability

- Old summary and dashboard routes forced clients to know which alias carried which subset of metrics.
- The redesign separates summary/statistics/chart/timeline/detail concerns and provides render-friendly chart series.
- New list endpoints expose consistent pagination fields: `page`, `show`, and `total`.

### SQL and filtering findings

- Date parsing previously rejected offset values after URL decoding converted `+` to a space.
- Several chart aliases encoded bucket semantics in the path instead of query parameters; bucket selection is now explicit.
- Query helpers now consistently apply `car_id` and date filters in the database layer used by the redesigned routes.
- Open-ended or unsupported calculations such as vampire drain are returned as explicit empty structures with limitations instead of guessed values.

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
- `dashboards/drives` and `dashboards/charges` into `summary`, `statistics`, and `charts/*`.
- `activity-timeline` into `timeline`.

### Delete

- Low-value aliases listed in “Legacy extension routes removed”.

### Rename

- `calendars/drives` -> `calendar/drives`
- `charts/charges/location-energy` -> `charts/charges/location`
- `charts/activity/duration` -> `charts/states/duration`
- bucket-specific chart aliases -> `charts/...` with `bucket` query parameter

### Add

- `summary`
- `statistics`
- chart families under `charts/*`
- `drives/:DriveID/details`
- `charges/:ChargeID/details`
- `timeline`
- `calendar/charges`
- `map/visited`

## Compatibility strategy

- Original compatible routes are still registered and keep their paths and response shapes.
- Extension routes are treated as redesignable and may change when consolidating duplicate semantics.
- OpenAPI clearly marks the extension surface as redesigned and potentially breaking.

## Data definitions and limits

- `startDate` / `endDate` accept RFC3339, timezone offset, decoded-space offset, local datetime, and date-only formats.
- Date-only `endDate` values are expanded to local end-of-day for redesigned range parsing.
- Chart endpoints default to bounded time windows to avoid unbounded scans.
- `map/visited` limits points and reports truncation in metadata.
- `vampire-drain` returns an empty structure with limitations until a reliable calculation path is available.
- Distances use TeslaMate settings-derived `km` or `mi`; speeds use `km/h` or `mi/h`; consumption uses `Wh/km` or `Wh/mi`; energy uses `kWh`.
