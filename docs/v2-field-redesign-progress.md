# V2 Field Redesign Implementation Progress

> Tracks the execution of [v2-field-redesign.md](v2-field-redesign.md).
> Each phase finishes only when build + vet + tests are green.

## Baseline

- Date: 2026-05-12
- Branch: `claude/musing-meitner-ea407a` (worktree)
- Pre-change build: `go build` ✅
- Pre-change tests: `go test ./src/... -count=1` ✅
- Pre-change vet: `go vet ./src/...` ✅

## Phase Plan

| # | Phase | Status |
|---|-------|--------|
| A | Foundations: extend `V2Unit`; canonical container types; meta updates | done |
| B | Driving (models + repo + service + tests) | done |
| C | Charging (models + repo + service + tests) | done |
| D | Parking (models + repo + service + tests) | done |
| E | Battery (range nesting + models + repo + service + tests) | done |
| F | Cost (drop `data_scope` + suffix strip + tests) | done |
| G | Updates (restructure version) + Lifecycle + Timeline | done |
| H | Summary repository/service/handler updates | done |
| I | Swagger regen + verify | done |

## Phase A — Foundations

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

- Extended `V2Unit` with `speed`, `duration`, `elevation`, `consumption`. `Duration` is now seconds (was minutes via implicit `_min` suffix on field names).
- `defaultV2Unit()` updated to populate all new fields.
- `V2AnalyticsQuery.Metrics` example switched from `distance_km,duration_min` → `distance,duration`.

## Phase B — Driving

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

- Merged `V2DrivingAnalyticsSummary` into canonical `V2DrivingSummary`; `V2DrivingResponse.Summary` now uses the canonical type.
- Stripped unit suffixes from JSON tags: `distance_km` → `distance`, `duration_min` → `duration`, `max_speed_kmh` → `max_speed`, `avg_consumption_wh_per_km` → `avg_consumption`, `range_loss_km` → `range_loss`, `avg_outside_temp_c` → `avg_outside_temp`, `peak_drive_power_kw` → `peak_drive_power`, etc.
- Switched `Duration` to seconds throughout (SQL: `SUM(duration_min) * 60`); `LongestDriveDuration` likewise in seconds.
- Converted `MaxSpeed` and `BatteryLevelUsed` from value types to `*float64` (undefined when no drives).
- Dropped `total_ascent` / `total_descent` (always zero, no data source).
- Driving timeseries fields renamed to match canonical names.
- `buildV2SummaryComparison` driving keys updated: `driving.distance_km` → `driving.distance`, `driving.duration_min` → `driving.duration`.
- `loadDrivingSummary` in summary repo updated to populate canonical fields and emit duration in seconds.
- Tests updated: `v2_driving_service_test.go`, `v2_summary_service_test.go`, `v2_handler_test.go`.

## Phase C — Charging

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

- Merged `V2ChargingAnalyticsSummary` into canonical `V2ChargingSummary`. Both `/summary` and `/charging` use the same struct now. Added missing aggregate fields (`LongestSessionDuration`, `AvgEnergyAdded`, `LargestSession`, `AvgCost`, `MaxCost`, `AvgCostPerEnergy`, `StartBatteryAvg`, `EndBatteryAvg`, AC/DC counters).
- Stripped unit suffixes: `energy_added_kwh` → `energy_added`, `energy_used_kwh` → `energy_used`, `duration_min` → `duration` (seconds), `avg_power_kw` → `avg_power`, `max_power_kw` → `max_power`, `charging_efficiency_percent` → `charging_efficiency`, `largest_session_kwh` → `largest_session`, `avg_cost_per_kwh` → `avg_cost_per_energy`, `start_battery_avg_percent` → `start_battery_avg`, `end_battery_avg_percent` → `end_battery_avg`, `ac_energy_kwh` → `ac_energy`, `dc_energy_kwh` → `dc_energy`.
- Charging duration now seconds (SQL: `SUM(duration_min) * 60`).
- `Cost` switched from `*float64` to `float64` (zero is meaningful — no charges = no cost). `EnergyUsed` likewise normalized to value.
- `loadChargingSummary` in summary repo updated; vehicle/cost-of-distance derivations point to canonical names.
- `buildV2SummaryComparison` charging keys updated: `charging.energy_added_kwh` → `charging.energy_added`, `charging.energy_used_kwh` → `charging.energy_used`.
- Tests updated: `v2_charging_service_test.go`, `v2_summary_service_test.go`, `v2_handler_test.go`.

## Phase D — Parking

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

- Merged `V2ParkingAnalyticsSummary` into canonical `V2ParkingSummary`. Both `/summary` and `/parking` use the same struct now.
- Stripped unit suffixes: `parked_duration_min` → `parked_duration` (seconds), same for asleep/online/offline/avg_parked. Charging-side parking duration likewise switched to seconds.
- Dropped `vampire_drain_range_km` (redundant with `estimated_vampire_drain` in energy units).
- Renamed `estimated_vampire_drain_kwh` → `estimated_vampire_drain` (unit comes from `meta.unit.energy`).
- `VampireDrainPercent` switched from value to `*float64` (undefined when no drain detected, distinct from zero drain).
- Dropped `total_duration_min` and `state_transition_count` duplicates from `V2ParkingBreakdownResponse` — they live on the summary already.
- `V2ParkingStateItem.duration_min` → `duration` (seconds); SQL switched accordingly.
- Renamed `V2ParkingLocationItem` duration fields and dropped `asleep/online/offline` (location-level state breakdown was always zero with no SQL backing).
- Tests updated: `v2_parking_service_test.go`, `v2_handler_test.go`.

## Phase E — Battery

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

- Replaced `V2BatteryAnalyticsSummary` with canonical `V2BatterySummary` (defined in `src/v2_models.go`); `/summary` and `/battery` share the same struct.
- Introduced `V2BatteryRange{Rated, Ideal}` and nested both current and baseline range under `RangeAtFullCharge` / `BaselineRangeAtFullCharge`.
- Stripped suffixes on snapshots: `latest_battery_level_percent` → `latest_level`, `latest_rated_range_km` → `latest_rated_range`, `latest_ideal_range_km` → `latest_ideal_range`.
- Replaced `estimated_rated_range_at_100_percent_km` / `estimated_ideal_range_at_100_percent_km` with `range_at_full_charge.rated` / `range_at_full_charge.ideal`. Same for baseline.
- Renamed `estimated_range_degradation_percent` → `estimated_range_degradation` (percent unit documented at API level).
- Dropped `sample_count` from both summary and timeseries item (implementation telemetry; retained in internal `V2BatteryStats`).
- `V2BatteryTimeseriesItem` now carries `range_at_full_charge` (nested) and `avg_level` instead of three flat percent/km fields.
- Repository populates the nested types; degradation calc reads from the temporary estimated value rather than the public field.
- Tests updated: `v2_battery_service_test.go`, `v2_handler_test.go`. Summary repo's battery loader now writes to `LatestLevel` / `LatestRatedRange` / `LatestIdealRange`.

## Phase F — Cost

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

- Dropped `V2CostDataScope` from the response (and the `defaultV2CostDataScope()` helper). Documentation about included/excluded scope moves to the API description / OpenAPI text.
- Stripped suffixes on `V2CostSummaryDetails`: `energy_used_kwh` → `energy_used`, `distance_km` → `distance`, `cost_per_kwh` → `cost_per_energy`. Dropped `cost_per_km` (use `cost_per_distance` which is per 100 km).
- Renamed `cost_per_100km` → `cost_per_distance` to align with the unit pair `(currency, distance)` from `meta.unit`.
- `V2CostPeriodItem` and `V2CostLocationItem` likewise: `energy_used_kwh` → `energy_used`.
- `V2CostSummary` (used by `/summary`) trimmed: dropped `cost_per_km`; renamed `cost_per_100km` → `cost_per_distance`.
- `loadVehicleSummary`-adjacent cost derivations updated (only `cost_per_distance` is now populated).
- Service no longer post-fills `DataScope`. Tests updated: `v2_cost_service_test.go`, `v2_handler_test.go`.

## Phase G — Updates / Lifecycle / Timeline

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

- Restructured `V2UpdateVersion` into nested `{Event, Window, Metrics}` per docs §5.
  - `V2UpdateEvent`: `version`, `started_at`, `completed_at`, `duration` (seconds), `days_since_prior`.
  - `V2UpdateWindow`: `start`, `end`, `interval` (seconds, optional).
  - `V2UpdateWindowMetrics`: nested `driving` (`V2UpdateDrivingMetrics`) and `charging` (`V2UpdateChargingMetrics`) plus top-level `inactive_duration` (seconds).
  - Driving metrics: `trip_count`, `duration` (seconds), `distance`, `net_energy`, `avg_consumption`.
  - Charging metrics: `session_count`, `duration` (seconds), `battery_energy`, `wall_energy`, `efficiency`, `cost`.
- Renamed `avg_update_duration_min` → `avg_update_duration` and switched to seconds (SQL: `AVG(EXTRACT(EPOCH FROM (end_date - start_date)))`, no `/60`).
- Per-version SQL aliases now match the Go names (`update_duration`, `interval_seconds`, `driving_duration`, `distance`, `net_drive_energy`, `charging_duration`, `battery_energy`, `wall_energy`, `inactive_duration`); driving/charging durations use `SUM(duration_min) * 60`; inactive duration dropped its `/ 60`.
- Repository emits `Window` only when both bounds valid, and `Metrics` only when at least one of trip/session/inactive is populated (`hasMetrics` gate) so empty objects aren't shipped.
- `V2LifecycleResponse` suffix strip: `distance_km` → `distance`, `energy_added_kwh` → `energy_added`, `energy_used_kwh` → `energy_used`, `avg_daily_distance_km` → `avg_daily_distance`, `avg_monthly_distance_km` → `avg_monthly_distance`, `avg_consumption_wh_per_km` → `avg_consumption`, `cost_per_100km` → `cost_per_distance`. Lifecycle SQL aliases updated to match.
- Timeline already followed the new conventions (no `_km`/`_min` JSON fields), no schema change needed.
- Tests updated: `v2_update_service_test.go`, `v2_lifecycle_service_test.go`.

## Phase H — Summary

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok.

- Trimmed `V2VehicleSummary` per docs §4 — kept only fields not derivable from charging/driving summaries: `odometer`, `tracked_distance`, `tracked_drives`, `tracked_charges`.
- Stripped suffixes: `odometer_km` → `odometer`, `tracked_distance_km` → `tracked_distance`.
- Dropped `rated_efficiency_kwh_per_100km`, `tracked_consumption_kwh_per_100km`, `tracked_wall_kwh_per_100km`, `charging_efficiency_percent`, `odometer_coverage_percent` (all derivable from charging/driving summaries or implementation telemetry).
- `loadVehicleSummary` simplified accordingly: SQL no longer joins `cars` for efficiency, single odometer read; removed all derived-percent computations.

## Phase I — Verify

Done 2026-05-12. `go vet ./src/...` clean, `go test ./src/...` ok, `go build ./src/...` ok.

- Regenerated `src/generated/{swagger.json,swagger.yaml}` via `swag init -g webserver.go -d src -o src/generated`.
- Removed the regenerated `src/generated/docs.go` (per the existing Makefile workflow — only the JSON/YAML are embedded).
- Stripped the `main.` prefix from all definition names / `$ref` paths in both swagger.json and swagger.yaml so the consumer-facing schema names match the Go `@name` annotations.
- Final verification: vet clean, full test suite green, binary builds successfully.
