# V2 Redesign Implementation Progress

> Tracks the execution of [v2-redesign.md](v2-redesign.md).
> Each phase finishes only when build + vet + tests are green.

## Baseline

- Date: 2026-05-11
- Branch: `claude/musing-meitner-ea407a` (worktree)
- Pre-change build: `make build` ✅
- Pre-change tests: `go test ./src/... -count=1` ✅ (`ok github.com/tobiasehlert/teslamateapi/src`)

## Phase Plan

| # | Phase | Endpoints/DTOs touched | Status |
|---|-------|-----------------------|--------|
| A | Delete pure-removal endpoints (`/insights`, `/reports`, `/calendar`, `/driving/ranking`, `/driving/distribution`, `/battery/distribution`, `/efficiency/factors`, `/analytics/locations`) | -8 endpoints | done |
| B | Delete `/analytics/efficiency` domain; merge essential fields into driving summary | -2 endpoints, fold efficiency models | done |
| C | Consolidate `/analytics/charging/*` into `/analytics/charging?include=&breakdown=` | -3 routes (timeseries/locations/types collapse) | done |
| D | Consolidate `/analytics/parking/*` into `/analytics/parking?include=&breakdown=` | -2 routes | done |
| E | Delete `/analytics/charging/cost` (use `/analytics/cost` only) | -1 route, -5 DTOs | done |
| F | Move out of `/analytics`: `updates → /updates`, `lifecycle → /lifecycle` | path move | done |
| G | Remove `lifetime` from period enum; remove `compare` from non-summary endpoints | param tightening | done |
| H | Replace `/v2` info with `/v2/capabilities` (structured feature map) | shape change | done |
| I | Regenerate swagger; verify build/tests/vet | verification | done |

## Notes / Decisions

- **DTO consolidation**: keep domain-specific DTOs (`V2DrivingSummary`, `V2ChargingTimeseriesItem`, …) — collapsing them into a generic `metrics: map[string]float64` would touch every repository SQL row scan and add no runtime value. Routes are consolidated; DTOs stay typed.
- **Breakdown dispatch**: `/analytics/{domain}?include=breakdown&breakdown=<key>` calls the existing per-key repository functions (locations, charger_type, state); the response wraps them under a `breakdown` field. No SQL changes.
- **Efficiency merge**: `avg_temperature_c`, `best_consumption_wh_per_km`, `worst_consumption_wh_per_km`, `estimated_regenerated_energy_kwh` are appended to `V2DrivingSummary`. Repository union-selects pull from existing efficiency repo SQL.
- **No backwards compatibility shims**: V2 not yet stable, breaking changes ship in one release per redesign doc §5.

## Phase A — Pure deletions

Done 2026-05-11. `go vet ./src/...` clean, `go test ./src/...` ok.

Removed routes:
- `/v2/cars/:CarID/analytics/insights`
- `/v2/cars/:CarID/analytics/reports`
- `/v2/cars/:CarID/analytics/reports/timeseries`
- `/v2/cars/:CarID/analytics/calendar`
- `/v2/cars/:CarID/analytics/locations`
- `/v2/cars/:CarID/analytics/driving/distribution`
- `/v2/cars/:CarID/analytics/driving/ranking`
- `/v2/cars/:CarID/analytics/battery/distribution`
- `/v2/cars/:CarID/analytics/efficiency/factors`

Removed types: V2InsightAPIResponse/Response, V2ReportAPIResponse/Response/TimeseriesResponse, V2CalendarAPIResponse/Response/Item, V2LocationAnalyticsAPIResponse/Response/Item, V2DrivingDistributionAPIResponse/Response/Item, V2DrivingRankingAPIResponse/Response/Item, V2BatteryDistributionAPIResponse/Response/Item, V2EfficiencyFactorsAPIResponse/Response/Item.

Removed services/builders: V2InsightService, V2ReportService, V2CalendarService, V2LocationService.

Removed repository methods: PostgresV2DrivingRepository.{Distribution,Ranking,rankingHighestDistanceDay} + bucket/ranking helpers; PostgresV2BatteryRepository.Distribution + helper; PostgresV2EfficiencyRepository.Factors + helper.

Removed errors: errV2InvalidDrivingDimension, errV2InvalidDrivingRanking, errV2InvalidEfficiencyDimension, errV2InvalidLocationSort.

## Phase B — Efficiency merge

Done 2026-05-11. `go vet ./src/...` clean, `go test ./src/...` ok.

Removed routes:
- `/v2/cars/:CarID/analytics/efficiency`

Merged fields into `V2DrivingAnalyticsSummary` (already present): `BestConsumptionWhPerKM`, `WorstConsumptionWhPerKM`, `EstimatedRegeneratedEnergyKWh`, `AvgOutsideTempC`. Driving summary SQL rewritten as a CTE so per-row consumption can feed AVG/MIN/MAX in one pass.

Deleted files:
- `src/v2_efficiency_models.go`
- `src/v2_efficiency_repository.go`
- `src/v2_efficiency_service.go`
- `src/v2_efficiency_service_test.go`

Removed from `src/v2_handler.go`: `V2EfficiencyBuilder` interface, `efficiencyBuilder` field, `efficiencyRepository` wiring, `Efficiency` handler, `handleV2EfficiencyError`, `/analytics/efficiency` route, `"efficiency"` from feature list.

Removed from `src/v2_handler_test.go`: `fakeV2EfficiencyBuilder`, `TestV2EfficiencyHandlerSuccess`, `/v2/cars/{CarID}/analytics/efficiency` from swagger/scalar path assertions.

## Phase C — Charging consolidation

Done 2026-05-11. `go vet ./src/...` clean, `go test ./src/...` ok.

Removed routes (folded into `/analytics/charging?include=timeseries,breakdown&group_by=...&breakdown=...`):
- `/v2/cars/:CarID/analytics/charging/timeseries`
- `/v2/cars/:CarID/analytics/charging/locations`
- `/v2/cars/:CarID/analytics/charging/types`

Removed types: `V2ChargingTimeseriesAPIResponse`, `V2ChargingLocationsAPIResponse`, `V2ChargingLocationsResponse`, `V2ChargingTypesAPIResponse`, `V2ChargingTypesResponse`. `V2ChargingTimeseriesResponse` and the per-row item types (`V2ChargingTimeseriesItem`, `V2ChargingLocationItem`, `V2ChargingTypeItem`) are now nested under `V2ChargingResponse.timeseries` / `V2ChargingResponse.breakdown`.

`V2ChargingBuilder` interface collapsed to two methods: `BuildCharging(..., V2ChargingBuildOptions)` and `BuildChargingCost(...)`. `V2ChargingService.BuildChargingTimeseries/Locations/Types` deleted.

New: `V2ChargingBuildOptions{IncludeTimeseries, IncludeBreakdown, GroupBy, BreakdownBy}`; `V2ChargingBreakdownResponse{by, locations?, types?}`; error `errV2InvalidChargingBreakdown` (400 with valid breakdown values listed).

## Phase D — Parking consolidation

Done 2026-05-11. `go vet ./src/...` clean, `go test ./src/...` ok.

Removed routes (folded into `/analytics/parking?include=breakdown&breakdown=...`):
- `/v2/cars/:CarID/analytics/parking/locations`
- `/v2/cars/:CarID/analytics/parking/states`

Removed types: `V2ParkingLocationsAPIResponse`, `V2ParkingLocationsResponse`, `V2ParkingStatesAPIResponse`. `V2ParkingStatesResponse` kept as plain (non-`@name`) internal type returned by the parking repository's `StateBreakdown` query; its `total_duration_min` and `state_transition_count` are folded into `V2ParkingBreakdownResponse` when `breakdown=state`.

`V2ParkingBuilder` interface collapsed to a single `BuildParking(ctx, carIDParam, timeRange, V2ParkingBuildOptions) (V2ParkingResponse, int64, error)`. `V2ParkingService.BuildParkingLocations/BuildParkingStates` deleted.

New: `V2ParkingBuildOptions{IncludeBreakdown, BreakdownBy}`; `V2ParkingBreakdownResponse{by, locations?, states?, total_duration_min?, state_transition_count?}`; error `errV2InvalidParkingBreakdown` (400 with valid breakdown values listed).

## Phase E — Cost de-duplication

Done 2026-05-11. `go vet ./src/...` clean, `go test ./src/...` ok.

Removed route: `/v2/cars/:CarID/analytics/charging/cost` (clients should use `/v2/cars/:CarID/analytics/cost`).

Removed types: `V2ChargingCostAPIResponse`, `V2ChargingCostResponse`, `V2ChargingCostSummary`, `V2ChargingCostPeriodItem`, `V2ChargingCostLocationItem`.

`V2ChargingBuilder` interface narrowed further to a single `BuildCharging(...)` method. `V2ChargingService.BuildChargingCost` and `PostgresV2ChargingRepository.{Cost,costByPeriod,costByLocation}` deleted; `V2ChargingRepository.Cost` removed from the interface.

## Phase F — Path restructure

Done 2026-05-11. `go vet ./src/...` clean, `go test ./src/...` ok.

Route renames (gin registration + swagger `@Router` annotations):
- `/v2/cars/:CarID/analytics/updates` → `/v2/cars/:CarID/updates`
- `/v2/cars/:CarID/analytics/lifecycle` → `/v2/cars/:CarID/lifecycle`

Updates and lifecycle are not period-windowed analytics — they live alongside `/timeline` outside the `/analytics` namespace per redesign §4. Handlers, builders, and DTOs are otherwise unchanged.

## Phase G — Param tightening

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

Allowed enums tightened:
- `period`: removed `lifetime` (lifetime data is exposed via `/lifecycle`, not via period-windowed analytics).
- `compare`: removed `previous_year` and `lifetime_average`. Only `none` and `previous_period` are accepted now.

Comparison support narrowed to `/analytics/summary` only. Removed `compare` query parameter, comparison block on response, comparison helper functions, and `errV2CompareUnsupported` error case from `driving`, `charging`, `parking`, and `battery` services and handlers. The `Updates` handler also drops its meta-only `compare` annotation.

Removed types/fields:
- `V2DrivingResponse.Comparison`, `V2ChargingResponse.Comparison`, `V2ParkingResponse.Comparison`, `V2BatteryResponse.Comparison`.

Removed functions:
- `buildV2DrivingComparison`, `buildV2ChargingComparison`, `buildV2ParkingComparison`, `buildV2BatteryComparison`.

Removed errors:
- `errV2CompareUnsupported` (parser now rejects invalid compare values up front; the dedicated error is no longer needed).

Tests for non-summary services rewritten to drop comparison-block assertions; summary tests keep comparison support.

## Phase H — Capabilities endpoint

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

Replaced `GET /v2` (and `GET /v2/`) with `GET /v2/capabilities`. The new payload describes per-domain feature flags rather than a flat string list, so clients can decide which query parameters to send without probing each endpoint.

Removed types: `V2InfoResponse`, `V2InfoAPIResponse`.

New types:
- `V2CapabilitiesResponse{version, domains[], breakdown_options}`
- `V2CapabilitiesDomain{name, path, supports_compare, supports_timeseries, supports_breakdown}`
- `V2CapabilitiesAPIResponse{data, meta}`

Domains advertised: summary (compare), driving (timeseries), charging (timeseries+breakdown), parking (breakdown), battery (timeseries), cost (timeseries), updates, lifecycle, timeline. Breakdown options exposed per domain: `charging: [location, charger_type]`, `parking: [location, state]`.

Replaced `TestV2InfoHandler` with `TestV2CapabilitiesHandler` that asserts version, non-empty domains list, summary→supports_compare flag, and charging breakdown options.

## Phase I — Verify

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok, `go build` ok.

Regenerated `src/generated/swagger.json` and `src/generated/swagger.yaml` via `swag init -g webserver.go -d src -o src/generated`, then stripped the `main.` namespace prefix from definitions and `$ref`s (190 occurrences in each file → 0) to match the committed style. Removed `src/generated/docs.go` per Makefile pattern.

Spec verification:
- `/v2/capabilities` is present.
- 11 V2 routes total: `/v2/capabilities`, `/v2/cars/{CarID}/{analytics/summary,analytics/driving,analytics/driving/timeseries,analytics/charging,analytics/parking,analytics/battery,analytics/battery/timeseries,analytics/cost,updates,lifecycle,timeline}`.
- Removed routes are absent: no `/analytics/insights`, `/analytics/reports*`, `/analytics/calendar`, `/analytics/locations`, `/analytics/driving/{distribution,ranking}`, `/analytics/battery/distribution`, `/analytics/efficiency*`, `/analytics/charging/{timeseries,locations,types,cost}`, `/analytics/parking/{locations,states}`, `GET /v2`.

All 9 phases of the V2 redesign are complete. Endpoint count: 24 → 11. Capability discovery moved from a flat string list to per-domain feature flags. Comparison support narrowed to `/analytics/summary`. `lifetime` removed from period enum. No backwards-compat shims.
