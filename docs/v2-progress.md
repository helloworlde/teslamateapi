# V2 API Implementation Progress

## Rules

- Each completed task must update this file.
- Each API must have handler, service, repository, response DTO, Swagger comments, and tests.
- Each API must be visible in Scalar documentation.
- After each task is complete, run checks, start the API with the real project runtime configuration, verify the task's API endpoints with curl, commit that task separately, then continue to the next task.
- All V2 API time ranges, time buckets, time outputs, and comparisons must default to the `TZ` environment variable as the IANA timezone; an explicit `timezone` query parameter overrides it, and database filtering must still use correctly converted UTC instants.

## Progress

| Task | Status | PR/Commit | APIs | Swagger | Scalar | Tests | Notes |
|---|---|---|---|---|---|---|---|
| T00 | Done |  | V2 base framework, `/api/v2`, `/api/docs`, `/api/docs/swagger.json`, `/api/docs/scalar` | Yes | Yes | Yes | scalar-go wired to the generated OpenAPI JSON handler. |
| T01 | Done |  | `/api/v2/cars/{CarID}/analytics/summary` | Yes | Yes | Yes | Supports `none` and `previous_period`; `previous_year` and `lifetime_average` return explicit not implemented errors. |
| T02 | Done |  | `/api/v2/cars/{CarID}/analytics/driving`, `/timeseries`, `/distribution`, `/ranking` | Yes | Yes | Yes | Supports summary, previous-period comparison, chart timeseries, dimension distributions, and objective rankings. Ranking SQL is qualified; time buckets and outputs use the selected/default timezone. |
| T03 | Done |  | `/api/v2/cars/{CarID}/analytics/charging`, `/timeseries`, `/locations`, `/types`, `/cost` | Yes | Yes | Yes | Supports summary, previous-period comparison, chart timeseries, location aggregation, charger type aggregation, and charging cost analytics. Time buckets and outputs use the selected/default timezone. |
| T04 | Done |  | `/api/v2/cars/{CarID}/analytics/parking`, `/locations`, `/states` | Yes | Yes | Yes | Supports state-duration summary, previous-period comparison, inferred parking sessions, inferred location aggregation, state breakdown, and estimated parking drain data quality warnings. |
| T05 | Done |  | `/api/v2/cars/{CarID}/analytics/battery`, `/timeseries`, `/distribution` | Yes | Yes | Yes | Supports latest battery/range samples, estimated full-range trend, battery-level distribution buckets, previous-period comparison, and data-quality warnings that estimates are not official SOH. |
| T06 | Done |  | `/api/v2/cars/{CarID}/analytics/efficiency`, `/factors` | Yes | Yes | Yes | Supports estimated energy/consumption summary and factual factor groupings by temperature, speed, distance, elevation, location, hour, and weekday without causal claims. |
| T07 | Done |  | `/api/v2/cars/{CarID}/analytics/cost` | Yes | Yes | Yes | Supports charging-cost scope, explicit excluded external cost categories, cost by period/location, and data-quality warnings for missing cost or energy data. Runtime curl verification intentionally skipped per temporary user instruction. |
| T08 | Done |  | `/api/v2/cars/{CarID}/analytics/locations` | Yes | Yes | Yes | Aggregates drive starts/ends, charging, inferred parking, charging cost, and estimated vampire drain by geofence/address with sort options. Runtime curl verification intentionally skipped per temporary user instruction. |
| T09 | Done |  | `/api/v2/cars/{CarID}/analytics/updates` | Yes | Yes | Yes | OTA update history with version list, update_count, latest_version, avg_duration_min. UTC half-open interval filtering. curl verification skipped per user instruction. |
| T10 | Done |  | `/api/v2/cars/{CarID}/analytics/lifecycle`, `/api/v2/cars/{CarID}/timeline` | Yes | Yes | Yes | Lifecycle cumulative stats from first recorded event; timeline returns unified drive/charge/park/update events with type/start/end/title/metrics, supports type filter and order. curl verification skipped per user instruction. |
| T11 | Done |  | `/api/v2/cars/{CarID}/calendar` | Yes | Yes | Yes | Daily aggregation over requested date range; fills gaps with zero rows; activity_level computed from fixed thresholds for driving/charging/parking_drain. UTC half-open interval; local date output. curl verification skipped per user instruction. |
| T12 | Done |  | `/api/v2/cars/{CarID}/reports` | Yes | Yes | Yes | Composite period report composing existing analytics services; sections hidden when no data; include param controls modules; no SQL duplication. curl verification skipped per user instruction. |
| T13 | Done |  | `/api/v2/cars/{CarID}/insights` | Yes | Yes | Yes | Objective insights with evidence and baseline_period; minimum sample rules enforced; no subjective language; supports category/min_severity filters; 10 insight types implemented. curl verification skipped per user instruction. |
