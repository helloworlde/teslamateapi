# V2 Field Design Redesign

> Companion to [v2-redesign.md](v2-redesign.md). The route consolidation pass left behind ~250 JSON tags whose names duplicate units already declared in `meta.unit`, plus several legacy decorator fields and inconsistent pointer/value rules. This document is the field-level redesign.
>
> Status tracked in [v2-field-redesign-progress.md](v2-field-redesign-progress.md).

## 1. Principles

1. **Units belong in `meta.unit`, not in field names.** Every JSON tag with a unit suffix (`_km`, `_kwh`, `_kw`, `_kmh`, `_min`, `_percent`, `_c`, `_m`, `_per_km`, `_per_100km`, `_per_kwh`, `_wh_per_km`, `_kwh_per_100km`) is renamed to drop the suffix. `meta.unit` is the single source of truth for how to interpret numeric values.
2. **One physical quantity, one unit.** `consumption` is always `Wh/km`. `cost-per-distance` is always `cost_per_100km`. `duration` is always `seconds`. No more `wh_per_km` vs `kwh_per_100km` co-existing in different domains.
3. **Counts stay dimensionless and named with `_count` suffix.** They are integers, never pointers.
4. **Pointer ⇔ undefined; value ⇔ zero is meaningful.** Aggregates over an empty period return `0`, not `null`. Pointers are reserved for genuinely missing data (no rows scanned, divide-by-zero, COALESCE NULL).
5. **No two summary types per domain.** `V2*AnalyticsSummary` is folded into `V2*Summary`. The `/summary` and `/{domain}` endpoints share the canonical type.
6. **No decorator fields in responses.** Constants masquerading as data (`V2CostDataScope`), implementation telemetry (`SampleCount`, `OdometerCoveragePercent`), and breakdown-level duplications of summary fields are removed.

## 2. New `V2Unit`

Extend [src/v2_models.go](src/v2_models.go):

```go
type V2Unit struct {
    Distance    string `json:"distance" example:"km"`
    Energy      string `json:"energy" example:"kWh"`
    Power       string `json:"power" example:"kW"`
    Speed       string `json:"speed" example:"km/h"`
    Duration    string `json:"duration" example:"seconds"`
    Elevation   string `json:"elevation" example:"m"`
    Consumption string `json:"consumption" example:"Wh/km"`
    Temperature string `json:"temperature" example:"C"`
    Currency    string `json:"currency" example:"CNY"`
}
```

Percentages are dimensionless numerics in `[0, 100]` — documented in API description, no `unit` entry needed.

## 3. Renaming table (selected highlights)

Full sweep is mechanical; this table just calls out the non-obvious decisions.

| Old JSON tag | New JSON tag | Notes |
|---|---|---|
| `distance_km` | `distance` | every domain |
| `duration_min` | `duration` | **value semantics changed**: minutes → seconds |
| `avg_speed_kmh` / `max_speed_kmh` | `avg_speed` / `max_speed` | |
| `peak_drive_power_kw` | `peak_drive_power` | |
| `avg_consumption_wh_per_km` | `avg_consumption` | unit declared in `meta.unit.consumption` |
| `best_efficiency_wh_per_km` / `worst_efficiency_wh_per_km` | `best_consumption` / `worst_consumption` | rename `efficiency` → `consumption` everywhere (same SQL anyway) |
| `avg_consumption_kwh_per_100km` | `avg_consumption` | converted to Wh/km |
| `rated_efficiency_kwh_per_100km` | dropped | duplicates consumption fields |
| `cost_per_km` | dropped | use `cost_per_100km` |
| `cost_per_100km` | `cost_per_distance` | unit pair `(currency, distance)` from `meta.unit` |
| `cost_per_kwh` | `cost_per_energy` | |
| `vampire_drain_range_km` | dropped | redundant with `estimated_vampire_drain` (energy) |
| `state_transition_count` (breakdown) | dropped | already in summary |
| `total_duration_min` (breakdown) | dropped | already in summary |
| `sample_count` (battery) | dropped | implementation detail |
| `odometer_coverage_percent` | dropped | implementation detail |
| `data_scope` | dropped | move to API documentation |
| `estimated_rated_range_at_100_percent_km` | `range_at_full_charge.rated` | nested `RangeAtFullCharge` object |
| `baseline_*` range fields | `baseline_range_at_full_charge.{rated,ideal}` | parallel struct |

## 4. Type consolidation

- `V2DrivingAnalyticsSummary` → merged into `V2DrivingSummary`. The two had ~80% overlap; the merged type carries every field. `/summary` returns the same struct as `/driving`.
- Same merge for `V2ChargingAnalyticsSummary → V2ChargingSummary`, `V2ParkingAnalyticsSummary → V2ParkingSummary`, `V2BatteryAnalyticsSummary → V2BatterySummary`.
- Keep `V2DrivingTimeseriesItem`, `V2ChargingTimeseriesItem`, `V2BatteryTimeseriesItem` distinct from summary — they're per-bucket rows, not aggregates.
- `V2VehicleSummary` keeps only fields *not* derivable from charging/driving summaries: `odometer`, `tracked_distance`, `tracked_drives`, `tracked_charges`. Drop `rated_efficiency_*`, `tracked_consumption_*`, `tracked_wall_*`, `charging_efficiency_*`, `odometer_coverage_*`.

## 5. Restructured types

### V2UpdateVersion → nested

```go
type V2UpdateVersion struct {
    Event   V2UpdateEvent          `json:"event"`
    Window  *V2UpdateWindow        `json:"window,omitempty"`
    Metrics *V2UpdateWindowMetrics `json:"metrics,omitempty"`
}

type V2UpdateEvent struct {
    Version        string  `json:"version"`
    StartedAt      string  `json:"started_at"`
    CompletedAt    *string `json:"completed_at,omitempty"`
    Duration       *float64 `json:"duration,omitempty"`         // seconds
    DaysSincePrior *int64  `json:"days_since_prior,omitempty"`
}

type V2UpdateWindow struct {
    Start    string  `json:"start"`
    End      string  `json:"end"`
    Interval *float64 `json:"interval,omitempty"`               // seconds between updates
}

type V2UpdateWindowMetrics struct {
    Driving  V2UpdateDrivingMetrics  `json:"driving"`
    Charging V2UpdateChargingMetrics `json:"charging"`
    InactiveDuration *float64        `json:"inactive_duration,omitempty"` // seconds
}

type V2UpdateDrivingMetrics struct {
    TripCount       int64    `json:"trip_count"`
    Duration        float64  `json:"duration"`
    Distance        float64  `json:"distance"`
    NetEnergy       *float64 `json:"net_energy,omitempty"`
    AvgConsumption  *float64 `json:"avg_consumption,omitempty"`
}

type V2UpdateChargingMetrics struct {
    SessionCount    int64    `json:"session_count"`
    Duration        float64  `json:"duration"`
    BatteryEnergy   *float64 `json:"battery_energy,omitempty"`
    WallEnergy      *float64 `json:"wall_energy,omitempty"`
    Efficiency      *float64 `json:"efficiency,omitempty"`     // percent
    Cost            *float64 `json:"cost,omitempty"`
}
```

### V2BatterySummary range nesting

```go
type V2BatterySummary struct {
    LatestLevel               *int64                      `json:"latest_level,omitempty"`
    LatestRatedRange          *float64                    `json:"latest_rated_range,omitempty"`
    LatestIdealRange          *float64                    `json:"latest_ideal_range,omitempty"`
    RangeAtFullCharge         *V2BatteryRange             `json:"range_at_full_charge,omitempty"`
    BaselineRangeAtFullCharge *V2BatteryRange             `json:"baseline_range_at_full_charge,omitempty"`
    EstimatedRangeDegradation *float64                    `json:"estimated_range_degradation,omitempty"`
}

type V2BatteryRange struct {
    Rated *float64 `json:"rated,omitempty"`
    Ideal *float64 `json:"ideal,omitempty"`
}
```

## 6. Pointer/value rules (applied uniformly)

- **Counts** (`*_count`): non-pointer `int64`. Empty period = `0`.
- **Total energy/distance/duration**: non-pointer `float64`. Empty period = `0`.
- **Averages, ratios, max/min, "longest"**: pointer. Undefined when no rows.
- **Levels, ranges, latest-snapshot fields**: pointer. Undefined when no positions data.

This kills inconsistencies like `V2DrivingSummary.DistanceKM` (value) vs `AvgTripDistanceKM` (pointer) by making the rule explicit.

## 7. Phase plan

Cross-cutting concerns (merge, unit-strip, drop, pointer-norm) tightly couple models ↔ repositories ↔ services ↔ tests. Phasing by *concern* would leave the build broken between phases; phasing by *domain* keeps each phase atomic.

| # | Phase | Domains touched | Status |
|---|-------|-----------------|--------|
| A | Foundations: extend `V2Unit`, redefine `V2Summary` container; update meta enums | `v2_models.go`, `v2_handler.go` | pending |
| B | Driving | models + repo + service + tests | pending |
| C | Charging | models + repo + service + tests | pending |
| D | Parking | models + repo + service + tests | pending |
| E | Battery + range nesting | models + repo + service + tests | pending |
| F | Cost (drop `data_scope`) | models + repo + service + tests | pending |
| G | Updates (restructure `V2UpdateVersion`) + Lifecycle + Timeline | models + repo + service + tests | pending |
| H | Summary repository + service + tests + handler comparison paths | summary code | pending |
| I | Swagger regen + verify build/test/vet | generated specs | pending |

Each phase ends with `go vet` + `go test` + `go build` green before flipping to done. Phase A unblocks the rest by introducing the new `V2Unit` shape and the canonical `V2*Summary` types; later phases just plug their SQL/services into them.

## 8. Out of scope

- No backwards-compat shims (V2 still pre-stable per redesign §5).
- No client/SDK updates (none in this repo).
- No new endpoints, no new query parameters.
- Scalar/Swagger UI assets unchanged.
