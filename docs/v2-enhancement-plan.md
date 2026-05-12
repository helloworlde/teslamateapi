# V1 字段增强 + V2 接口扩展实施计划

基于 [v2-data-gaps.md](v2-data-gaps.md) 与 v1 端点盘点，重新拆分：
- **v1 已存在的路径**（cars / drives / charges / battery-health / updates / globalsettings）→ 不在 v2 重复造，改为在 v1 上补字段。
- **v2 仅承载聚合 / 分析 / 时序桶**这类 v1 不适合做的东西。

> v1 现有响应字段清单见本文件「附录 A」。下面带 `[v1]` 的为 v1 增强，带 `[v2]` 的为 v2 新端点。

---

## 优先级总览

| Phase | 范围 | 类型 | 工作量 | 风险 |
|---|---|---|---|---|
| **J** | `/v1/cars` + `/v1/globalsettings` 字段补全 | v1 | S | 低 |
| **K** | `/v1/cars/:id/charges*` 字段补全 + `/v2/.../charging/curve` 聚合 | v1 + v2 | M | 中 |
| **L** | `/v2/.../analytics/efficiency`（gross consumption + 温度桶） | v2 | M | 中 |
| **M** | `/v1/cars/:id/drives*` 字段补全 + 路线下采样 | v1 | M | 中 |
| **N** | `/v2/.../parking/idle_periods`（空闲段分析） | v2 | M | 中 |
| **O** | `/v1/cars/:id/battery-health` 字段补全 + `/v2/.../battery/capacity_by_mileage` | v1 + v2 | M | 中 |
| **P** | `/v1/cars/:id/tire-pressure`（新 v1 端点）+ `/v2/.../analytics/environmental` | v1 + v2 | M | 低 |
| **Q** | `/v1/cars/:id/updates` 字段补全 + v2 长尾（odometer 时序 / by_period / lifecycle.places / geofences / timeline.missing） | v1 + v2 | M | 低 |

退出标准：`go build` / `go vet` / `go test ./src/... -count=1` 全绿；swagger 重生成；progress 文档登记。

---

## 设计约束

1. **v1 字段只追加，不修改 / 不删除**。所有新增字段使用 `omitempty`，避免影响现有客户端。
2. **v1 不做单位换算的字段保留 SI**，已有的英制换算（`kilometersToMiles` 等）维持原状。
3. **v2 只做 v1 无法承载的聚合 / 桶 / 时序**；不复制 v1 已有的列表 / 详情 / 实时端点。
4. **跨 v1/v2 共用 SQL helper**：odometer 回退过滤、`date_bin` 桶、slope-adjusted 公式等抽 `src/v2_common_sql.go`。

---

## Phase J — `/v1/cars` + `/v1/globalsettings` 字段补全 `[v1]`

**目标**：lfp_battery / enabled / display_priority / marketing_name / 单位偏好 / theme_mode 六个 schema 字段当前完全没暴露；v1 是它们的天然位置。

### v1 改动

#### `GET /v1/cars` —— `CarDetails` / `CarSettings` / 新增 `CarLifecycle`
```diff
 car_details:
   eid, vid, vin, model, trim_badging, efficiency,
+  marketing_name
+
+car_exterior: 不变

 car_settings:
   suspend_min, suspend_after_idle_min, req_not_unlocked, free_supercharging, use_streaming_api,
+  lfp_battery,
+  sleep_mode_enabled,
+  enabled

+ car_lifecycle:
+   display_priority,
+   first_recorded_at,   // MIN(drives.start_date, charging_processes.start_date, ...)
+   last_recorded_at
```

> `display_priority` 来自 `cars.display_priority`（多车排序）。`first_recorded_at` / `last_recorded_at` 跟 v2 lifecycle 共享一段 SQL，可抽 helper。

#### `GET /v1/globalsettings` —— `TeslaMateUnits` / 新增 `TeslaMateGUI` 字段
```diff
 teslamate_units:
   unit_of_length, unit_of_temperature,
+  unit_of_pressure

 teslamate_webgui:
   preferred_range, language,
+  theme_mode
```

### 涉及表
`cars`, `car_settings`, `settings`, `drives`, `charging_processes`, `updates`, `states`（first/last_recorded_at 取并集 MIN/MAX）。

### 测试
- `/v1/cars` happy + LFP 车 + disabled 车（保留 enabled=false 的行）。
- `/v1/globalsettings` 新字段非空。

---

## Phase K — `/v1/cars/:id/charges*` 字段补全 + `/v2 charging/curve` 聚合 `[v1+v2]`

**目标**：把 `charges` 表里 v1 已读但未暴露的字段（cost_per_kwh / charging_efficiency / interval_sec / position_id 等）放到 v1；DC 充电曲线的*中位线*这种聚合放 v2。

### v1 改动

#### `GET /v1/cars/:id/charges`（列表）—— `Charges` 增强
```diff
 charges[]:
   charge_id, start_date, end_date, address,
   charge_energy_added, charge_energy_used, cost,
   duration_min, duration_str,
   battery_details, range_ideal, range_rated,
   outside_temp_avg, odometer, latitude, longitude,
+  geofence: { id, name } | null,
+  connection: "ac" | "dc",
+  fast_charger_brand, fast_charger_type,
+  charger_pilot_current_max,
+  energy_used_confidence,
+  interval_sec,
+  cost_per_kwh,
+  charging_efficiency,        // energy_added / GREATEST(energy_used, energy_added)
+  energy_per_hour,            // energy_added / duration_h
+  battery_heater_minutes,     // SUM CASE WHEN battery_heater_on
+  is_complete                 // end_date IS NOT NULL
```

#### `GET /v1/cars/:id/charges/:id`（详情）—— `Charge` 同上 + `ChargeDetails` 增强
```diff
 charge_details[]:
   detail_id, date,
   battery_level, usable_battery_level,
   charge_energy_added, not_enough_power_to_heat,
   charger_details, battery_info,
   conn_charge_cable, fast_charger_info,
   outside_temp,
+  t_offset_sec,               // EXTRACT(EPOCH FROM date - cp.start_date)
+  ideal_range,                // 当前缺失（仅 rated 在 BatteryInfo）
+  rated_range
```

#### `GET /v1/cars/:id/charges/current`（活动）—— `Charge` + `Car` 增强
```diff
 charge:
   charge_id, start_date, is_charging, address,
   charge_energy_added, cost, duration_min, duration_str,
   battery_details, rated_range, outside_temp_avg, odometer,
   charge_details[],
+  geofence: { id, name } | null,
+  connection,
+  charger_pilot_current,
+  charger_phases,
+  charger_voltage,
+  estimated_completion,        // start_date + duration_min × (target − cur)/(cur − start)
+  target_battery_level         // 来自 charge_state.charge_limit_soc 缓存？schema 未存，先观察是否能从 charges 推导，否则不加
```

> `target_battery_level` 若 schema 不可得就跳过；不为占位字段。

### v2 改动

#### `GET /v2/cars/{car_id}/analytics/charging/curve` `[v2]`
DC 充电会话的 SoC × 中位 / 分位功率，跨多个 session 聚合，v1 单 session 的 `charge_details` 无法呈现。

参数：`period`, `start`, `end`, `min_sessions`(default 5)。

```json
{
  "data": {
    "samples": [
      {
        "battery_level": 23,
        "session_count": 12,
        "median_power": 142.3,
        "p25_power": 128.0,
        "p75_power": 152.4
      }
    ]
  }
}
```
SQL 提示：
```sql
SELECT c.battery_level,
       count(distinct cp.id),
       percentile_cont(0.50) WITHIN GROUP (ORDER BY c.charger_power),
       percentile_cont(0.25) WITHIN GROUP (ORDER BY c.charger_power),
       percentile_cont(0.75) WITHIN GROUP (ORDER BY c.charger_power)
FROM charges c
JOIN charging_processes cp ON cp.id = c.charging_process_id
WHERE cp.car_id = $1 AND cp.start_date >= $2 AND cp.start_date < $3
  AND (c.charger_phases IS NULL OR c.charger_phases = 0)
GROUP BY c.battery_level
HAVING count(distinct cp.id) >= $4;
```

### 涉及表
`charging_processes`, `charges`, `addresses`, `geofences`, `positions`, `cars`, `car_settings`。

### 公共改动
- `V2ChargingLocationItem` / `V2CostLocationItem` 增加 `latitude`, `longitude`（地图渲染）。
- `V2ChargingSummary` 增加 `network_breakdown`（按 `fast_charger_brand` 聚合）。

---

## Phase L — `/v2/.../analytics/efficiency` `[v2]`

**目标**：v1 的 drives 详情有 `consumption_net`，但跨 drive + idle + parasitic 的 *gross* consumption 与 Consumption Overhead、温度 / 速度桶都是聚合分析，归 v2。

### v2 改动

#### `GET /v2/cars/{car_id}/analytics/efficiency`
参数：`period`, `start`, `end`, `timezone`, `group_by`(temperature_5c | speed_10kmh | none), `include`(summary,timeseries,buckets)。

```json
{
  "data": {
    "summary": {
      "net_consumption": 162.4,
      "gross_consumption": 184.6,
      "consumption_overhead": 0.121,
      "drive_distance": 1240.5,
      "drive_duration": 67200,
      "energy_consumed_drives": 201.4,
      "energy_consumed_idle": 24.5,
      "avg_outside_temp": 12.3
    },
    "timeseries": [{ "period_start": "...", "net_consumption": ..., "gross_consumption": ..., "consumption_overhead": ... }],
    "buckets": {
      "temperature_5c": [
        { "bucket": -10, "drive_count": 12, "distance": 220.4, "range_loss": 35.6, "efficiency": 0.62, "consumption": 246.0, "avg_speed": 41.2 }
      ]
    }
  }
}
```

### 修改 v2 既有
- `V2DrivingSummary`：新增 `gross_consumption`, `consumption_overhead`。
- `V2DrivingTimeseriesItem`：新增可选 `gross_consumption`。

### SQL（移植 Grafana `efficiency.json`）
- 多 CTE：drives + charging_processes + 边界处理 → idle_energy = `prev.end_rated_range − this.start_rated_range` × car.efficiency。
- 温度桶：`ROUND(outside_temp_avg/5)*5`。
- 速度桶：positions `LEAD(date)` 加权。

---

## Phase M — `/v1/cars/:id/drives*` 字段补全 + 路线下采样 `[v1]`

**目标**：drives 详情已暴露大量数据，但缺 `ascent` / `descent` / 起止 lat/lng 与 geofence、slope-adjusted 效率；drive_details 数量大，需要下采样开关。

### v1 改动

#### `GET /v1/cars/:id/drives`（列表）—— `Drives` 增强
```diff
 drives[]:
   drive_id, start_date, end_date, start_address, end_address,
   odometer_details, duration_min, duration_str,
   speed_max, speed_avg, power_max, power_min,
   battery_details, range_ideal, range_rated,
   outside_temp_avg, inside_temp_avg,
   energy_consumed_net, consumption_net,
+  ascent,
+  descent,
+  consumption_slope_adjusted,           // 见公式
+  start_geofence: { id, name } | null,
+  end_geofence: { id, name } | null,
+  start_position: { latitude, longitude },
+  end_position: { latitude, longitude },
+  is_complete                            // end_date IS NOT NULL
```

> `battery_details.reduced_range` 在 v1 里已存在（基于 `usable_battery_level`），保留。

#### `GET /v1/cars/:id/drives/:id`（详情）—— `Drive` 同上；`DriveDetails` 增强
```diff
 drive_details[]:
   detail_id, date, latitude, longitude, speed, power, odometer,
   battery_level, usable_battery_level, elevation,
   climate_info, battery_info,
+  tpms_pressure_fl, tpms_pressure_fr, tpms_pressure_rl, tpms_pressure_rr   // 单位由 settings.unit_of_pressure 控制
```

#### 新增 query 参数（drives 详情）
- `?sample=full|every_5s|every_30s`：下采样选项；默认 `every_5s`（与 Grafana `trip.json` 5s 桶一致）。当前 v1 不限制，长行程返回数千行。
- `?include_route=false`：仅返回 drive 元数据，不返回 drive_details。

### 涉及表
`drives`, `addresses`, `geofences`, `positions`, `cars`, `settings`。

### slope-adjusted 公式
```
consumption_slope_adjusted (Wh/km)
= consumption_net + (ascent − descent) × 9.81 × car_mass / (3600 × distance × 1000) × 1000 / regen_efficiency
其中 car_mass = 2100 (kg), regen_efficiency = 0.85
```
（与 Grafana `drives.json` 完全一致；常量先硬编码，将来按车型扩展。）

### 公共改动
- 抽 `slopeAdjustedConsumption(distance, ascent, descent, baseConsumption float64) float64` 到 helper。

---

## Phase N — `/v2/.../parking/idle_periods` `[v2]`

**目标**：vampire-drain.json 的逐空闲段拆解。v1 没有对应路径；v2 parking 当前只有汇总。

### v2 改动

#### `GET /v2/cars/{car_id}/parking/idle_periods`
参数：`start`, `end`, `min_duration_hours` (default 1), `geofence_id`, `limit`, `cursor`。

```json
{
  "data": {
    "items": [
      {
        "id": "idle:8821-8822",
        "start_date": "...",
        "end_date": "...",
        "duration": 34240,
        "start_battery_level": 67,
        "end_battery_level": 65,
        "soc_diff": -2,
        "range_loss": 12.4,
        "energy_drained": 1.9,
        "avg_power_w": 198.5,
        "range_loss_per_hour": 1.30,
        "standby_ratio": 0.94,
        "has_reduced_range": false,
        "geofence": { "id": 3, "name": "Home" }
      }
    ],
    "has_more": false,
    "next_cursor": null
  }
}
```

### 修改 v2 既有
- `V2ParkingSummary`：新增 `standby_ratio`。

### SQL
直接移植 Grafana `vampire-drain.json` 的 `merge` + `v` CTE：
```sql
WITH events AS (
  SELECT 'drive'::text AS kind, start_date, end_date, end_position_id AS pos
  FROM drives WHERE car_id = $1 AND end_date IS NOT NULL
  UNION ALL
  SELECT 'charge', start_date, end_date, position_id
  FROM charging_processes WHERE car_id = $1 AND end_date IS NOT NULL
), o AS (
  SELECT *,
         lag(end_date)  OVER (ORDER BY start_date) AS prev_end,
         lag(pos)       OVER (ORDER BY start_date) AS prev_pos
  FROM events
), gaps AS (
  SELECT prev_end AS gap_start, start_date AS gap_end, prev_pos, pos
  FROM o WHERE prev_end IS NOT NULL AND start_date - prev_end > INTERVAL '1 hour'
)
SELECT g.*,
       p1.battery_level, p2.battery_level,
       p1.rated_battery_range_km, p2.rated_battery_range_km,
       (SELECT SUM(EXTRACT(EPOCH FROM LEAST(s.end_date, g.gap_end) - GREATEST(s.start_date, g.gap_start)))
        FROM states s
        WHERE s.car_id = $1 AND s.state IN ('asleep','offline')
          AND s.start_date < g.gap_end AND COALESCE(s.end_date, g.gap_end) > g.gap_start) AS standby_seconds
FROM gaps g
JOIN positions p1 ON p1.id = g.prev_pos
JOIN positions p2 ON p2.id = g.pos;
```

---

## Phase O — `/v1/cars/:id/battery-health` 字段补全 + `/v2 capacity_by_mileage` `[v1+v2]`

**目标**：v1 已有 max_capacity / current_capacity / battery_health_percentage，但缺 cycles / data_lost / lfp 标志；半月桶 capacity 散点是分析，归 v2。

### v1 改动

#### `GET /v1/cars/:id/battery-health` —— `BatteryHealth` 增强
```diff
 battery_health:
   max_range, current_range,
   max_capacity, current_capacity,
   rated_efficiency, battery_health_percentage,
+  cycles,                         // floor(SUM(charge_energy_added) / max_capacity_when_new)
+  data_lost,                      // MAX(end_km) - MIN(start_km) - SUM(distance)
+  lfp_battery,                    // car_settings.lfp_battery
+  estimated_capacity_at_full_charge: { rated, ideal },
+  derived_efficiency,             // mode of (charge_energy_added / range_diff)
+  samples_used                    // 计算 capacity 用的 charges 行数
```

### v2 改动

#### `GET /v2/cars/{car_id}/battery/capacity_by_mileage` `[v2]`
半月桶散点。

```json
{
  "data": {
    "items": [
      { "bucket": "2024-04-1", "odometer_avg": 18420, "capacity_kwh_p50": 76.2 },
      { "bucket": "2024-04-2", "odometer_avg": 18910, "capacity_kwh_p50": 76.0 }
    ]
  }
}
```

#### 修改 v2 既有
- `V2BatterySummary`：新增 `lfp_battery`。
- `V2BatteryRange`：新增 `rated_usable`, `ideal_usable`。
- `V2BatteryTimeseriesItem`：新增 `range_at_full_charge_usable`。

### SQL
半月桶：`to_char(date, 'YYYYMM') || CASE WHEN extract(day from date) <= 15 THEN '1' ELSE '2' END`。

---

## Phase P — `/v1/cars/:id/tire-pressure` + `/v2 environmental` `[v1+v2]`

**目标**：胎压 / HVAC / 温度 / 高程当前完全没暴露。v1 加一个胎压端点（车辆健康类）；v2 加聚合时序。

### v1 改动

#### `GET /v1/cars/:id/tire-pressure`（新增端点）
```json
{
  "data": {
    "car": { "car_id": 1, "car_name": "Model 3" },
    "latest": {
      "as_of": "2026-05-12T05:11:30Z",
      "fl": 2.8, "fr": 2.8, "rl": 2.7, "rr": 2.8
    },
    "window": {
      "start": "2026-04-12T00:00:00Z",
      "end": "2026-05-12T00:00:00Z",
      "min": { "fl": 2.5, "fr": 2.6, "rl": 2.4, "rr": 2.5 },
      "max": { "fl": 3.0, "fr": 3.0, "rl": 2.9, "rr": 3.0 }
    },
    "history": [
      { "date": "2026-05-11", "fl_avg": 2.78, "fr_avg": 2.79, "rl_avg": 2.71, "rr_avg": 2.79 }
    ],
    "units": { "unit_of_length": "km", "unit_of_pressure": "bar" }
  }
}
```
- 单位由 `settings.unit_of_pressure` 控制（`bar` 默认，`psi` 用 `barToPsi`）。

### v2 改动

#### `GET /v2/cars/{car_id}/analytics/environmental` `[v2]`
温度 / HVAC / 高程时序聚合。

参数：`period`, `start`, `end`, `metrics`(outside_temp,inside_temp,elevation,climate_on_ratio,battery_heater_minutes,defroster_minutes)。

```json
{
  "data": {
    "summary": {
      "outside_temp_min": 4.2,
      "outside_temp_max": 33.1,
      "outside_temp_avg": 18.4,
      "inside_temp_avg": 22.1,
      "climate_on_minutes": 412,
      "battery_heater_minutes": 38,
      "defroster_minutes": 14,
      "elevation_min": 0.0,
      "elevation_max": 412.5,
      "elevation_gain_total": 1402.0,
      "elevation_loss_total": 1389.0
    },
    "timeseries": [
      {
        "period_start": "2026-05-12T00:00:00Z",
        "outside_temp_avg": 18.5,
        "inside_temp_avg": 22.4,
        "elevation_avg": 12.4,
        "climate_on_ratio": 0.42
      }
    ]
  }
}
```

### 涉及表
`positions`（HVAC / 胎压 / 温度 / 高程）, `drives`（ascent/descent 累计）, `settings`。

### 不暴露
- 单点胎压通过 v1 的 `tire-pressure` 端点提供；v2 不再重复实时单点查询。
- 「实时车辆状态」端点：先不做；v1 的 `/charges/current` 已覆盖充电中场景，其他场景频率低，不值得新建。

---

## Phase Q — `/v1/cars/:id/updates` 字段补全 + v2 长尾 `[v1+v2]`

### v1 改动

#### `GET /v1/cars/:id/updates` —— `Updates` 增强
```diff
 updates[]:
   update_id, start_date, end_date, version,
+  short_version,             // split_part(version, ' ', 1)
+  duration_min,
+  days_since_prior,          // age(start_date, lag(start_date) OVER (ORDER BY start_date))
+  active_duration_days       // EXTRACT(EPOCH FROM next.start_date - this.start_date)/86400 ; 当前是最新版时取 now − start
```

### v2 改动

#### `GET /v2/cars/{car_id}/lifecycle/odometer_series` `[v2]`
odometer 累计时序（drives.start_km / end_km UNION）。

#### `GET /v2/cars/{car_id}/summary/by_period` `[v2]`
打平的 per-period 表（drives + charging + cost + Consumption Overhead + Data Complete 一行）。

#### `GET /v2/cars/{car_id}/lifecycle/places` `[v2]`
addresses 城市 / 州 / 国家计数 + top-N + last-visited。

#### `GET /v2/geofences` `[v2]`
全围栏 + 计费规则（`cost_per_unit`, `billing_type`, `session_fee`, `radius`, lat/lng）。

#### timeline 修改 `[v2]`
- `V2TimelineEvent` 新增 `type: "missing"`：`TP2.odometer - TP1.odometer > 0.5` 且端点 address 不一致。
- park 事件附带 `consumption_during_park = (prev.range − next.range) × cars.efficiency`。

#### updates 汇总 `[v2]`
- `V2UpdateSummary` 新增 `median_days_between_updates`。

---

## 跨 Phase 公共改动

1. **`V2ChargingLocationItem` / `V2ParkingLocationItem` / `V2CostLocationItem`** 增加 `latitude`, `longitude`（Phase K 一次性补全；地图渲染依赖）。
2. **cursor 分页约定**：v2 新增列表（`charging/curve` 不算 / `parking/idle_periods` 算 / `lifecycle/odometer_series` 算）统一 `?limit=`, `?cursor=`，cursor = base64(JSON{last_id, last_date})。v1 现有分页继续沿用 page/per_page。
3. **公共 SQL helper**（建议在 Phase L 之前抽出）：
   - odometer 回退过滤 (`p.odometer - lag(p.odometer) OVER (...) >= 0`)
   - `date_bin` 桶生成（带零填充）
   - slope-adjusted 公式
4. **错误码扩展**：v2 新增 `INVALID_CURSOR`, `INVALID_FILTER`；v1 沿用 `TeslaMateAPIHandleErrorResponse` 字符串形式，不动。
5. **测试基线**：每个新 service 至少 happy path / car_not_found / 边界（空列表 / `is_complete=false` / `lfp_battery=true`）。
6. **Swagger 注解**：新 model 加 `@name`；handler 加 `@Summary` / `@Tags` / `@Param` / `@Success`；保持 `main.` 前缀剥离脚本可用。
7. **v1 字段追加保险**：所有新增字段 `omitempty` + 指针类型，避免未填值时序列化为零值误导客户端。

---

## 待 review 决策点

1. **Phase 顺序**：J → Q 串行，还是放开非依赖 Phase 并行（J 与 P 互不依赖）。
2. **v1 字段追加是否需要灰度开关**？我倾向不加（新字段 `omitempty` 已经向后兼容，加开关反而复杂）。
3. **drive_details 默认下采样到 5s 是否激进**？现在 v1 不限制，长行程响应数千行。可改为新增 `?sample=full` 显式回退，默认值 `every_5s`。
4. **`/v1/cars/:id/tire-pressure` 路径**：是否改成 `/v1/cars/:id/health` 把胎压、电池健康一起放？（注：`/battery-health` 已占用，统一性差，倾向独立路径。）
5. **`charging.live` 的 `target_battery_level`**：schema 不存，要不要忽略（推荐忽略）。
6. **公共 SQL helper 抽取时机**：Phase L 之前 vs 第一个用到的 Phase 内顺手抽。

---

## 实施流程

1. 本文档 review 通过 → 创建 [v2-enhancement-progress.md](v2-enhancement-progress.md)，每个 Phase 登记 pending。
2. 单 Phase 单 commit；本地合到 `feature/v2`；推到远端前等用户确认。
3. 退出标准：`go build / vet / test` 全绿；swagger 重生成；progress 文档登记 done。

---

# 附录 A — V1 现有响应字段清单

来源：`src/v1_TeslaMateAPI*.go`。同名字段下面 v2 增强不会重复实现。

### `GET /v1/cars`, `GET /v1/cars/:id`
- `data.cars[]`：`car_id`, `name`, `car_details`, `car_exterior`, `car_settings`, `teslamate_details`, `teslamate_stats`
- `car_details`：`eid`, `vid`, `vin`, `model`, `trim_badging`, `efficiency`
- `car_exterior`：`exterior_color`, `spoiler_type`, `wheel_type`
- `car_settings`：`suspend_min`, `suspend_after_idle_min`, `req_not_unlocked`, `free_supercharging`, `use_streaming_api`
- `teslamate_details`：`inserted_at`, `updated_at`
- `teslamate_stats`：`total_charges`, `total_drives`, `total_updates`

### `GET /v1/cars/:id/battery-health`
- `data.car`：`car_id`, `car_name`
- `data.battery_health`：`max_range`, `current_range`, `max_capacity`, `current_capacity`, `rated_efficiency`, `battery_health_percentage`
- `data.units`：`unit_of_length`, `unit_of_temperature`

### `GET /v1/cars/:id/charges`
- `data.car`, `data.charges[]`, `data.units`
- `charges[]`：`charge_id`, `start_date`, `end_date`, `address`, `charge_energy_added`, `charge_energy_used`, `cost`, `duration_min`, `duration_str`, `battery_details`, `range_ideal`, `range_rated`, `outside_temp_avg`, `odometer`, `latitude`, `longitude`
- `battery_details`：`start_battery_level`, `end_battery_level`
- `range_ideal` / `range_rated`：`start_range`, `end_range`

### `GET /v1/cars/:id/charges/current`
- `data.charge`：`charge_id`, `start_date`, `is_charging`, `address`, `charge_energy_added`, `cost`, `duration_min`, `duration_str`, `battery_details`, `rated_range`, `outside_temp_avg`, `odometer`, `charge_details[]`
- `battery_details`：`start_battery_level`, `current_battery_level`
- `rated_range`：`start_range`, `current_range`, `added_range`
- `charge_details[]`：`detail_id`, `date`, `battery_level`, `usable_battery_level`, `charge_energy_added`, `not_enough_power_to_heat`, `charger_details`, `battery_info`, `conn_charge_cable`, `fast_charger_info`, `outside_temp`
- `charger_details`：`charger_actual_current`, `charger_phases`, `charger_pilot_current`, `charger_power`, `charger_voltage`
- `fast_charger_info`：`fast_charger_present`, `fast_charger_brand`, `fast_charger_type`
- `battery_info`：`rated_battery_range`, `battery_heater`, `battery_heater_on`, `battery_heater_no_power`

### `GET /v1/cars/:id/charges/:id`
- 同 `current`，差别：`battery_details` 用 `start_battery_level/end_battery_level`；`battery_info` 加 `ideal_battery_range`；`charge` 增加 `end_date`, `latitude`, `longitude`, `range_ideal`。

### `GET /v1/cars/:id/drives`
- `data.car`, `data.drives[]`, `data.units`
- `drives[]`：`drive_id`, `start_date`, `end_date`, `start_address`, `end_address`, `odometer_details`, `duration_min`, `duration_str`, `speed_max`, `speed_avg`, `power_max`, `power_min`, `battery_details`, `range_ideal`, `range_rated`, `outside_temp_avg`, `inside_temp_avg`, `energy_consumed_net`, `consumption_net`
- `odometer_details`：`odometer_start`, `odometer_end`, `odometer_distance`
- `battery_details`：`start_usable_battery_level`, `start_battery_level`, `end_usable_battery_level`, `end_battery_level`, `reduced_range`, `is_sufficiently_precise`
- `range_ideal` / `range_rated`：`start_range`, `end_range`, `range_diff`

### `GET /v1/cars/:id/drives/:id`
- 同 `drives`，外加 `drive_details[]`：`detail_id`, `date`, `latitude`, `longitude`, `speed`, `power`, `odometer`, `battery_level`, `usable_battery_level`, `elevation`, `climate_info`, `battery_info`
- `climate_info`：`inside_temp`, `outside_temp`, `is_climate_on`, `fan_status`, `driver_temp_setting`, `passenger_temp_setting`, `is_rear_defroster_on`, `is_front_defroster_on`
- `battery_info`：`est_battery_range`, `ideal_battery_range`, `rated_battery_range`, `battery_heater`, `battery_heater_on`, `battery_heater_no_power`

### `GET /v1/cars/:id/updates`
- `data.car`, `data.updates[]`
- `updates[]`：`update_id`, `start_date`, `end_date`, `version`

### `GET /v1/globalsettings`
- `data.settings`：`setting_id`, `account_info`, `teslamate_units`, `teslamate_webgui`, `teslamate_urls`
- `account_info`：`inserted_at`, `updated_at`
- `teslamate_units`：`unit_of_length`, `unit_of_temperature`
- `teslamate_webgui`：`preferred_range`, `language`
- `teslamate_urls`：`base_url`, `grafana_url`
