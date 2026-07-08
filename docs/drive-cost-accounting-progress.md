# Drive Cost Accounting Consistency

This document is the working checklist for the drive-cost accounting repair.
Progress is tracked here so the implementation can be audited against the
requested scope.

## Required Accounting Basis

Drive cost must be computed from drive energy, not from distance-amortised
charging cost.

For v1 single-drive list/detail endpoints, the drive cost must use the same
energy basis as the row's `energy_consumed_net` field so adjacent short trips do
not collapse to the same integer-SOC cost:

```
energy_consumed_net = (start_rated_range_km - end_rated_range_km)
                    * cars.efficiency

charge_price_per_kwh = SUM(charging_processes.cost)
                     / SUM(charging_processes.charge_energy_added)

estimated_usage_cost = energy_consumed_net * charge_price_per_kwh
```

The v1 field is `null` when `energy_consumed_net` is unavailable or when there
is no positive `charge_energy_added` denominator. If positive charge energy
exists but costs are missing or zero, the average charge price is `0` and the
estimated usage cost is `0`.

Other drive-cost stats still use the SOC-derived battery-side accounting basis:

```
charge_price_per_kwh = SUM(charging_processes.cost)
                     / SUM(charging_processes.charge_energy_added)

kwh_per_pct = SUM(charging_processes.charge_energy_added)
            / SUM(end_battery_level - start_battery_level)
              over completed charges with positive SOC gain

drive_energy_kwh = max(start_position.battery_level - end_position.battery_level, 0)
                 * kwh_per_pct

drive_cost = drive_energy_kwh * charge_price_per_kwh
```

`charge_energy_added` is battery-side energy that entered the vehicle. Drive
energy uses the same SOC-derived battery-side basis, so cost and energy are
both on the vehicle-side accounting basis for those aggregate endpoints.
Aggregate endpoints return `null` for drive-cost fields when any
positive-distance drive in scope lacks start/end SOC data or when the
kWh-per-SOC calibration cannot be computed. They should not silently treat
missing SOC energy as zero and understate cost.

## Explicit Non-Goals

- Do not compute drive cost as total charging cost divided by total distance.
- Do not use integer SOC drop for v1 single-drive list/detail costs; use the
  row's `energy_consumed_net` basis instead.
- Do not convert rated range to energy for aggregate cost accounting unless the
  endpoint is explicitly matching a row-level rated-range energy field.
- Do not scan all `positions` rows to compute drive energy. Use each drive's
  `start_position_id` and `end_position_id`.
- Do not collapse charging cost fields (`total_cost`, `charges_cost`) into drive
  cost fields. They remain charging-bill totals.

## Progress

| Area | Requirement | Status |
| --- | --- | --- |
| Shared SQL helpers | Provide reusable SOC energy and charge-price SQL fragments | Complete |
| v1 drive list/detail | `estimated_usage_cost = energy_consumed_net * charge_price_per_kwh` | Complete |
| v2 lifetime | `cost_per_distance = total_drive_cost / total_distance` | Complete |
| v2 summary buckets | Add period drive cost from bucket drive energy and bucket charge price | Complete |
| v2 consumption groups | Add drive cost using each group's drive energy and global charge price | Complete |
| v2 energy-flow | Use `charging_cost_per_kwh` for `driving_cost` | Complete |
| v2 partial-SOC cost guard | Return null instead of underestimating aggregate drive costs | Complete |
| DTO comments | Replace distance-amortised cost descriptions | Complete |
| README | Document the new drive-cost basis | Complete |
| Swagger | Regenerate generated OpenAPI docs | Complete |
| Tests | Update available unit tests for cost formula changes | Complete for unit tests and v1 Docker-Postgres integration |
| Verification | `gofmt`, `go test`, `go vet`, doc diff review | Complete |
| DB-backed SQL formula tests | Validate v1 list/detail SQL outputs against Postgres fixtures | Complete for v1 single-drive list/detail |
| Dev DB EXPLAIN | Validate changed v1 SQL plan shape against local dev Postgres | Complete for changed v1 list/detail query shape |

## Completion Notes

Implementation notes and verification evidence should be appended here as the
work progresses.

- v1 drive list and drive detail now compute `estimated_usage_cost` from
  `energy_consumed_net` and battery-side average charging price. Drive-list
  `startDate` / `endDate` filters also scope the charging price CTE.
- v2 lifetime now exposes drive-side `estimated_usage_cost`; `cost_per_distance`
  is derived from drive usage cost divided by total drive distance. These fields
  are `null` when the SOC energy basis or charge price cannot be calculated, or
  when any positive-distance drive in scope lacks start/end SOC data.
- v2 summary now exposes `drives_estimated_usage_cost` and
  `drives_cost_per_distance` per bucket. Buckets with driving but no
  battery-side charge energy or partial drive SOC data return `null` for drive
  cost instead of `0`.
- v2 consumption groups now expose `estimated_usage_cost` and
  `cost_per_distance`; the underlying Wh/distance ranking remains descriptive
  rated-range consumption, while cost uses SOC-derived drive energy and returns
  `null` for groups with partial drive SOC data.
- v2 energy-flow metrics-level `driving_cost` now uses
  `charging_cost_per_kwh`; node/link costs stay on the vehicle accounting pool
  rate so the graph remains cost-conserved.
- Swagger docs were regenerated with `swag init --dir cmd/teslamateapi,internal,pkg/dto -g main.go -o docs --outputTypes go,yaml,json --parseDependency --parseInternal`.
- Verification passed:
  - `go test ./internal/httpapi/handlers/v2 -count=1`
  - `go test ./internal/httpapi/handlers/v1 -count=1`
  - `go test ./...`
  - `go vet ./...`
  - `git diff --check`
- Docker-backed v1 verification was added as
  `TestV1DriveCostIntegrationWithPostgres`. Run it against the dev Postgres
  container with:

  ```
  docker compose -f dev/docker-compose.yml up -d --wait
  TESLAMATEAPI_DB_INTEGRATION=1 \
  DATABASE_HOST=127.0.0.1 DATABASE_PORT=55432 \
  DATABASE_USER=teslamate DATABASE_PASS=secret \
  DATABASE_NAME=teslamate DATABASE_SSL=disable \
  go test ./internal/httpapi/handlers/v1 \
    -run TestV1DriveCostIntegrationWithPostgres -count=1 -v
  ```

  The fixture inserts two drives that both drop 1% SOC but have different
  `energy_consumed_net`, one drive with non-positive rated-range drop, and one
  car with no `charge_energy_added`; v1 list/detail outputs matched the new
  null and cost semantics.
- Dev DB `EXPLAIN (COSTS OFF)` was run against equivalent changed v1 list and
  detail SQL. The changed shape keeps `positions` joined only through each
  drive's start/end position IDs (`Index Scan using positions_pkey` for both
  start and end rows) and reduces charge-price work to one
  `charging_processes` aggregate CTE. On the tiny seed database PostgreSQL may
  still choose seq scans/sorts for small `drives`/`charging_processes` tables;
  that is not a regression from this change, which removes the old second
  charging-process aggregate.
