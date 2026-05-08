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
| T06 | Pending |  | efficiency | No | No | No |  |
| T07 | Pending |  | cost | No | No | No |  |
| T08 | Pending |  | locations | No | No | No |  |
| T09 | Pending |  | updates | No | No | No |  |
| T10 | Pending |  | lifecycle, timeline | No | No | No |  |
| T11 | Pending |  | calendar | No | No | No |  |
| T12 | Pending |  | reports | No | No | No |  |
| T13 | Pending |  | insights | No | No | No |  |
