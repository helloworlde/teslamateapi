# AGENTS.md

Operational and domain notes for agents working on teslamateapi.

## Energy accounting (v2 stats) — how battery energy is computed

This is the single most error-prone area of the codebase. Read this before
touching any handler that reports kWh.

### The rule: never convert rated range to energy

**Do not compute battery energy as `rated_range_km * cars.efficiency`.**

`cars.efficiency` (TeslaMate, kWh/km) is a rounded/derived constant. On real
data it does **not** reconcile with the metered energy that actually entered the
battery. Empirically, across every charge of one Model Y:

```
charge_energy_added / (rated_range_gained * efficiency) = 0.9466   (constant)
```

i.e. `rated_range_km * efficiency` overstates real battery energy by ~5.6%
(`efficiency` was 0.15; the value implied by metered energy is ~0.142 kWh/km).
The 4-decimal constancy of that ratio proves it is a deterministic calibration
error, not sensor noise.

### Why the overstatement is a real bug, not cosmetics

Endpoints that do **energy accounting** also read the metered
`charge_energy_added` (energy into the battery, Tesla BMS) and `cost`. When the
battery inventory is range-derived (inflated) but the charge input is metered
(true), the books don't balance:

```
post-charge inventory (range)  >  pre-charge inventory (range) + charge_energy_added (metered)
        64.845 kWh             >        17.79 kWh             +      44.54 kWh   = 62.33
```

The 2.5 kWh surplus is **energy from nowhere** — physically impossible. It is
*not* a loss (a loss is `wall_energy - charge_energy_added`, energy that left as
heat *before* the pack). It is purely the range→energy conversion error, and it
surfaced as `balance_status = inventory_reconciliation_mismatch` /
`usage_exceeds_available_energy` and a phantom `inventory_reconciliation_delta`
node in the Sankey.

### The correct basis: SOC × measured capacity-per-percent

Anchor everything to two things the vehicle actually meters:
`charge_energy_added` (kWh into the pack) and `battery_level` (% SOC).

```
kwh_per_pct = charge_energy_added / (end_battery_level - start_battery_level)

energy_at(level)      = level            * kwh_per_pct
driving_energy(drive) = (start_soc - end_soc of the drive's positions) * kwh_per_pct
parking_energy(gap)   = (soc drop over the parked gap)                  * kwh_per_pct
```

Because `post = end_level * kwh_per_pct` and
`start = start_level * kwh_per_pct`, we get `post - start = charge_energy_added`
*exactly*. The reconciliation surplus collapses to ~0 and the Sankey balances
without inventing energy.

- **charge_usage** (`/api/v2/cars/{id}/charges/{id}/usage`): `kwh_per_pct` is
  derived from *this charge* (`selected_charge.kwh_per_pct`), NULL if the charge
  has no usable SOC gain. Drives join `positions` (start/end_position_id) for
  per-drive SOC.
- **energy-flow** (`/api/v2/cars/{id}/stats/energy-flow`): `kwh_per_pct` is the
  lifetime average — `SUM(charge_energy_added) / SUM(SOC gained)` over all
  charges with positive SOC gain (`cap` CTE).

SOC is integer-percent, so it is coarser than rated range. That is an accepted
trade for the two **accounting** endpoints, where consistency with metered
energy matters more than per-drive precision and the SOC deltas are large or
aggregated.

### Endpoints still on `range * efficiency` (intentionally, for now)

`stats_behavior`, `stats_summary`, `stats_lifetime`, `stats_consumption`,
`parkings` still use `range * efficiency`. They are **descriptive** stats
(consumption Wh/km, per-drive energy) that do *not* reconcile against metered
charge energy, so they never produce the impossible-surplus symptom. They are
~5.6% scale-biased, but converting them to integer-SOC would wreck fine-grained
per-drive Wh/km (a 2 km drive is ~0.5% SOC → rounds to 0/1%). The appropriate
fix for *their* bias, if wanted, is recalibrating `cars.efficiency` to the
metered value — not switching to SOC. Flag this when asked to "fix all energy
stats".

### Testing note

`buildChargeUsageData` / `buildEnergyFlowData` take already-computed kWh, so the
Go unit tests exercise the aggregation logic, not the SQL. The SQL energy basis
is only validated against a live database. There is no local Postgres in CI;
verify query changes against a real TeslaMate DB before trusting absolute kWh.

## Query performance — how to not write a full-table scan

Correctness is not enough: this is a time-series DB and the wrong query shape is
invisible on seed data but multi-second in production. The slow-query logger
(`>100ms` warn) is the backstop, not the first line of defence.

### Know the table magnitudes

- **`positions`** — *largest*. One row every few seconds while driving; millions
  of rows after a few months. Anything that scans or sorts all of a car's
  positions is a multi-second query.
- **`charges`** — *second largest*. One row every few seconds while charging;
  tens to hundreds of thousands of rows.
- **`drives` / `charging_processes`** — small (hundreds–thousands), `car_id`
  indexed. Joining these by primary key is cheap.

### Antipatterns that have already bitten us

- **Aggregating a whole table to read its endpoints.** To get the first/last row
  use `ORDER BY date ASC|DESC LIMIT 1`, never `array_agg(x ORDER BY date)[1]` —
  that reads and sorts the entire relation just to take one element.
- **Scanning the same big table twice in one query.** If two CTEs differ only in
  a projected column (e.g. `rated_` vs `ideal_battery_range_km`), compute both in
  one pass and pick each result from the small shared CTE. Merging is only valid
  when their `JOIN`/`WHERE`/`GROUP BY` are byte-identical.
- **Filtering a big table by something other than `(car_id, date)`** when an
  endpoint is per-car — it forces a sort or seq scan. Endpoint tail-lookups
  (`ORDER BY date DESC LIMIT 1`) want a `positions (car_id, date)` index.

### Before committing any SQL that touches `positions` or `charges`

1. Run it through `EXPLAIN` against the dev DB (`dev/docker-compose.yml`). Seed
   data is tiny so *timings* are meaningless, but the **plan shape** (Seq Scan +
   Sort, or an aggregate building an array over a full relation) reveals the
   antipattern.
2. Keep the dev seed schema (`dev/init.sql`) complete enough that the query
   actually *runs* locally — a missing column means it was never executed before
   shipping.
