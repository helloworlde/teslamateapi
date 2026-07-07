# Drive Cost Accounting Consistency

This document is the working checklist for the drive-cost accounting repair.
Progress is tracked here so the implementation can be audited against the
requested scope.

## Required Accounting Basis

Drive cost must be computed from drive energy, not from distance-amortised
charging cost.

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
both on the vehicle-side accounting basis.

## Explicit Non-Goals

- Do not compute drive cost as total charging cost divided by total distance.
- Do not convert rated range to energy for cost accounting.
- Do not scan all `positions` rows to compute drive energy. Use each drive's
  `start_position_id` and `end_position_id`.
- Do not collapse charging cost fields (`total_cost`, `charges_cost`) into drive
  cost fields. They remain charging-bill totals.

## Progress

| Area | Requirement | Status |
| --- | --- | --- |
| Shared SQL helpers | Provide reusable SOC energy and charge-price SQL fragments | Complete |
| v1 drive list/detail | `estimated_usage_cost = drive_energy_kwh * charge_price_per_kwh` | Complete |
| v2 lifetime | `cost_per_distance = total_drive_cost / total_distance` | Complete |
| v2 summary buckets | Add period drive cost from bucket drive energy and bucket charge price | Complete |
| v2 consumption groups | Add drive cost using each group's drive energy and global charge price | Complete |
| v2 energy-flow | Use `charging_cost_per_kwh` for `driving_cost` | Complete |
| DTO comments | Replace distance-amortised cost descriptions | Complete |
| README | Document the new drive-cost basis | Complete |
| Swagger | Regenerate generated OpenAPI docs | Complete |
| Tests | Update available unit tests for cost formula changes | Complete for unit tests; DB SQL tests blocked below |
| Verification | `gofmt`, `go test`, `go vet`, doc diff review | Complete |
| DB-backed SQL formula tests | Validate v1/lifetime/summary/consumption SQL outputs against Postgres fixtures | Blocked: no local Postgres/psql and Docker daemon unavailable |
| Dev DB EXPLAIN | Validate changed SQL plan shape against local dev Postgres | Blocked: no local Postgres/psql and Docker daemon unavailable |

## Completion Notes

Implementation notes and verification evidence should be appended here as the
work progresses.

- v1 drive list and drive detail now compute `estimated_usage_cost` from raw
  SOC drop, lifetime `kwh_per_pct`, and battery-side average charging price.
  Drive-list `startDate` / `endDate` filters also scope the charging price and
  SOC calibration CTEs.
- v2 lifetime now exposes drive-side `estimated_usage_cost`; `cost_per_distance`
  is derived from drive usage cost divided by total drive distance. These fields
  are `null` when the SOC energy basis or charge price cannot be calculated.
- v2 summary now exposes `drives_estimated_usage_cost` and
  `drives_cost_per_distance` per bucket. Buckets with driving but no
  battery-side charge energy return `null` for drive cost instead of `0`.
- v2 consumption groups now expose `estimated_usage_cost` and
  `cost_per_distance`; the underlying Wh/distance ranking remains descriptive
  rated-range consumption, while cost uses SOC-derived drive energy and does not
  require rated-range readings.
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
- DB-backed SQL formula tests and Dev DB EXPLAIN were not run in this
  environment: no local `postgres` process was present, `psql` was not
  installed, and Docker was unavailable (`/var/run/docker.sock` missing). The
  changed query shapes join positions only by each drive's `start_position_id` /
  `end_position_id`; they do not scan sampled positions by car/date.
