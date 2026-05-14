# V1 / V2 接口审计与优化方案（2026-05）

本文档梳理用户反馈的五个具体问题，扩展审计了所有 v1/v2 端点，并给出一份可执行的优化方案。每条问题都附带 `file:line` 证据，便于后续按章节落地实施。

涉及的路由注册点：
- `internal/server/server.go`（v1 + 顶层）
- `internal/api/v2/v2_handler.go::RegisterV2Routes`（v2 全部路由）

---

## 1. 用户反馈的 5 个问题

### 1.1 `/v1/cars/{CarID}/tire-pressure` 的 `avg` 没有意义

**结论：部分确认。**

- 顶层 `window` 块本来就返回每个轮胎的 `min`/`max`（结构体在 `internal/api/v1/v1_TeslaMateAPICarsTirePressure.go:51-63`，对应 SQL 在 `134-148`）。
- 用户真正反对的 `avg` 字段在 **每日历史数组** 中：`fl_avg/fr_avg/rl_avg/rr_avg`（结构体 `internal/api/v1/v1_TeslaMateAPICarsTirePressure.go:65-71`，SQL `156-168`）。
- 胎压是按车辆轮询节奏离散采样的，慢漏气在 "日均" 视图里几乎被抹平；真正能体现异常的是当日的最低值与最高值之差。

**建议修复**

- 把每日历史的 `*_avg` 替换为 `*_min` / `*_max`（或同时返回三者，破坏性较小）。
- SQL 中的 `AVG(tpms_pressure_*)` → `MIN(...)` / `MAX(...)`。
- 顺手修复时区问题：`to_char(date_trunc('day', date), 'YYYY-MM-DD')`（`v1_TeslaMateAPICarsTirePressure.go:156`）按 DB 会话时区分桶（默认 UTC），对非 UTC 用户在凌晨前后会出现 off-by-one；需用 `AT TIME ZONE` 或 Go 侧基于 `apicommon.AppUsersTimezone` 重新分桶。

---

### 1.2 `/v2/cars/{CarID}/analytics/battery/timeseries` 抽象错误

**结论：完全确认。**

- 当前实现（`internal/api/v2/v2_battery_repository.go:112-171`）是把 `drives.start_rated_range_km / sp.battery_level * 100` 按日/周/月平均，作为 "假设 100% SoC 时的续航估计"。
- 返回结构（`internal/api/v2/v2_battery_models.go:27-31`）只有 `period_start / range_at_full_charge / avg_level`。
- 对 "电量随时间变化" 的诊断诉求来说，这个抽象是错的：
  - `avg_level` 在月聚合下毫无意义。
  - 完全没有 "电量从多少变到多少、什么活动导致" 的语义。

**建议修复（推荐方案：替换为 SoC 事件流）**

把这条端点重写为 "SoC 状态变化事件流"，事件来源 = `drives ∪ charging_processes`：

```jsonc
{
  "event_type": "drive" | "charge" | "idle",  // idle 由相邻事件之间的间隔推导
  "id": 12345,                                 // drive_id 或 charging_process_id
  "start_date": "2026-05-12T08:00:00+08:00",
  "end_date":   "2026-05-12T08:42:00+08:00",
  "start_battery_level": 78,
  "end_battery_level":   62,
  "soc_delta": -16,                            // 负数=放电，正数=充电
  "energy_kwh": 11.4,                          // 充电时的 charge_energy_added；行驶用 efficiency*距离 估算
  "distance_km": 32.5,                         // 仅 drive
  "location": { ... },                         // 起点/终点 / 充电站 / geofence
  "tags": ["highway", "supercharger"]          // 可选
}
```

复用已存在的 `v2_lifecycle_extras.go::buildTimeline` 思路（已经在做 drive/charge/update 的并表），新端点本质是 timeline 的 "只看电量" 视图。如果 `/timeline` 已经能覆盖，此端点可直接删除并把字段 `start/end_battery_level` 补到 timeline 的 drive/charge 元素上。

---

### 1.3 `/v2/cars/{CarID}/analytics/battery` 与 v1 重复

**结论：部分确认。**

| 字段 | v1 `battery-health` | v2 `analytics/battery` |
|---|---|---|
| 当前 SoC | — | `latest_level` |
| 最新续航（rated/ideal） | `current_range` | `latest_rated_range`/`latest_ideal_range` |
| 满电估算续航（rated/ideal） | `estimated_capacity_at_full_charge.{rated,ideal}` | `range_at_full_charge.{rated,ideal}` |
| 最大续航 | `max_range` | — |
| 当前 / 最大容量 | `current_capacity` / `max_capacity` | — |
| 健康度百分比 | `battery_health_percentage` | — |
| 循环数 | `cycles` | — |
| 是否 LFP / 数据丢失 | `lfp_battery` / `data_lost` | — |
| 推导效率 / 样本量 | `derived_efficiency` / `samples_used` | — |
| 基线满电续航 | — | `baseline_range_at_full_charge` |
| 续航衰减估算 | — | `estimated_range_degradation` |

v1 是严格更丰富的集合，v2 仅多了 `baseline_*` 与 `estimated_range_degradation` 两个字段，且都能在 v1 的现有 SQL 上廉价补出（基线 = 最早 N 次满充的中位数估算）。

**建议修复**

1. 把 `baseline_range_at_full_charge` 与 `estimated_range_degradation` 两个字段加进 v1 `battery-health`。
2. 删除 v2 `/analytics/battery`（`v2_handler.go:487-503`）和对应的 service / repository / model。
3. v2 `/analytics/battery/timeseries` 按 §1.2 单独处理（删或改）。

---

### 1.4 `/v2/cars/{CarID}/analytics/charging` 的 `group_by` 没生效

**结论：完全确认。**

- handler 在 `v2_handler.go:391` 无条件读 `c.Query("group_by")` 并写入 `opts.GroupBy`。
- service 仅在 `IncludeTimeseries == true` 时才校验/应用它（`v2_charging_service.go:51-58`）。
- 因此 `?group_by=month` 在不带 `include=timeseries` 时被静默丢弃，响应不会带任何按月的数据，也不会报错。
- **同样问题** 出现在 `/v2/cars/:CarID/analytics/efficiency`（`v2_efficiency.go:381-385`）：`group_by` 仅在 `include=buckets` 时生效。

**建议修复（二选一）**

- **A（推荐）：严格化** —— 不带 `include=timeseries` 时收到 `group_by` 直接 400 + 明确报错；同时在 swagger 注解里把这个依赖写清楚。
- **B：宽容化** —— 只要 `group_by` 出现就隐式打开 `include=timeseries`，但要在响应中显式告知客户端发生了隐式启用（`meta.applied_include` 字段）。

不论选哪条，`v2_efficiency.go::Timeseries` 这个声明了但永远不被赋值的死字段（`v2_efficiency.go:24`）应当一起清理或填上。

---

### 1.5 `/v2/cars/{CarID}/analytics/charging/curve` 的平均没有意义

**结论：基本反驳，但有一个真问题。**

- 曲线已经按电量桶（`battery_level` bucket）返回 **median / P25 / P75**（`v2_charging_curve.go:78-88`），并不是单点平均；只有 `avg_voltage`（`84` 行）是 AVG，但电压方差很小，影响有限。
- **真正的 bug**：曲线响应了 `period`/`start`/`end`（`v2_charging_curve.go:161, 89`），默认与其它 v2 端点一样落到当前周期（如 "本月"）。结合默认 `min_sessions=5`，大多数用户调用时拿到的是空 `samples` —— 这才是用户实际感知到的 "没有意义"。

**建议修复**

1. 让 curve 默认覆盖 **整个生命周期**（除非显式传入 `start`/`end`），避免和其它分析端点共用 "本期" 默认值。
2. 在每个 SoC 桶里追加 `min_power` / `max_power`（沿用现有 SQL，廉价）。这样既能看分布上下沿（用户的诉求），也保留中位数/分位数（异常值过滤）。
3. 可选：`avg_voltage` → `median_voltage`，与功率字段保持风格一致。

---

## 2. 审计中发现的其它问题

### 2.1 误导性的均值

- `/v2/.../analytics/charging` summary 的 `avg_power` 是 `AVG(MAX(charger_power) per session)`（`v2_charging_repository.go:61`）。一次超充会把整月慢充的均值拉飞；要么按 AC/DC 分组，要么用中位数。
- `/v2/.../analytics/cost` summary 仅有 `avg_cost`/`max_cost`，无 `min`/`p50`/每 kWh 分布；且 `cost_per_distance` 把 "本期充电的钱" 除以 "本期开过的距离"（`v2_summary_repository.go:65-67`、`v2_cost_repository.go:59-62`），但电是这个月充、油是下个月用，跨期会失真。
- `/v2/.../analytics/environmental` timeseries 仅 `*_avg`（`v2_environmental.go:193-199`）；驾驶效率受温度极值与海拔总爬升影响最大，应同时返回 min/max 与 elevation gain/loss。
- `/v2/.../analytics/driving` 的 `avg_speed` 是 `AVG(distance/duration*60)` per drive（`v2_driving_repository.go:87`），不是按距离加权；1 km 的挪车与 200 km 的高速被赋予相同权重。应改为 `SUM(distance)/SUM(duration)`。

### 2.2 v1 ↔ v2 重复

- `/v2/analytics/battery` ↔ v1 `battery-health` —— 见 §1.3。
- `/v2/summary/by_period`（`v2_lifecycle_extras.go:240-289`）与 `/v2/analytics/summary?period=...`（多次调用）重叠；前者一次返回多行，后者单行。
- `/v2/lifecycle/odometer_series`（`v2_lifecycle_extras.go:87-98`）≈ `/v2/summary/by_period` 的 distance 列；可以从 by_period 推导，独立端点价值有限。

### 2.3 时间序列丢失了状态变化语义

- `/v2/analytics/battery/timeseries` —— 见 §1.2。
- `/v2/analytics/driving/timeseries` 按日/周分桶（`v2_driving_repository.go:156-202`）；对 ≤90 天窗口直接返回 per-drive 数组（或重定向到 `/timeline?type=drive`）更有用。

### 2.4 单位 / 货币 / 时区

- **`Currency` 硬编码 `CNY`**（`v2_common.go:187` `defaultV2Unit`）。任何非 CN 部署的响应 meta 都在说谎。应读 TeslaMate `settings.currency`，或新增 `CURRENCY` 环境变量覆盖。
- `/v2/lifecycle/places` 的 `last_visited` 直接 `MAX(d.end_date)::text`（`v2_lifecycle_extras.go:438`），未走 RFC3339 + 时区归一化，与其它 v2 端点不一致。
- v1 `tire-pressure` 历史日期用 DB 会话时区（默认 UTC）分桶（`v1_TeslaMateAPICarsTirePressure.go:156`），非 UTC 用户跨午夜会 off-by-one。
- v2 `lifecycle / timeline / geofences / places` meta 的 `Timezone` 硬编码 `"UTC"`（`v2_handler.go:667, 723`，`v2_lifecycle_extras.go:524, 642`），忽略 `?timezone=`，与其它 v2 端点不一致。

### 2.5 分页 / 过滤 / 时间范围

- `/v2/timeline` 同时接收 `before` 与 `after` 时未做互斥校验（`v2_handler.go:697-714`），两者会一起送进 `BuildTimeline`。
- `/v2/lifecycle/places` 只支持 `top_n`（`v2_lifecycle_extras.go:434-447`），无法问 "2025 年去过的地方"；缺日期范围过滤。
- v1 `charges`/`drives`/`updates` 的 `page`/`show` 参数已正确实现（`v1_TeslaMateAPICarsCharges.go:36-37, 190-194`），无问题。

### 2.6 capability map 漂移

`/v2/capabilities`（`v2_handler.go:208`）声明的能力集（`217-225`）与实际注册的路由不一致，至少缺少：`efficiency`、`parking/idle_periods`、`battery/capacity_by_mileage`、`lifecycle/odometer_series`、`lifecycle/places`、`summary/by_period`、`geofences`、`analytics/environmental`、`analytics/charging/curve`。客户端按 capabilities 探测能力时会漏掉这些。

---

## 3. 端点状态总览

| 方法 | 路径 | 处理函数 file:line | 状态 |
|---|---|---|---|
| GET | `/v1/cars` (含 `:CarID`) | `v1_TeslaMateAPICars.go` | clean |
| GET | `/v1/cars/:CarID/battery-health` | `v1_TeslaMateAPICarsBatteryHealth.go` | clean（拟扩展并取代 v2） |
| GET | `/v1/cars/:CarID/tire-pressure` | `v1_TeslaMateAPICarsTirePressure.go:25` | **problematic**（§1.1 + 时区） |
| GET | `/v1/cars/:CarID/charges{,/current,/:ChargeID}` | `v1_TeslaMateAPICarsCharges*.go` | clean |
| GET | `/v1/cars/:CarID/drives{,/:DriveID}` | `v1_TeslaMateAPICarsDrives*.go` | clean |
| GET | `/v1/cars/:CarID/status` | `v1_TeslaMateAPICarsStatus.go` | clean |
| GET | `/v1/cars/:CarID/updates` | `v1_TeslaMateAPICarsUpdates.go` | clean |
| GET | `/v1/globalsettings` | `v1_TeslaMateAPIGlobalsettings.go` | clean |
| GET | `/v2/capabilities` | `v2_handler.go:208` | **problematic**（§2.6） |
| GET | `/v2/cars/:CarID/analytics/summary` | `v2_handler.go:251` | clean（与 by_period 有重叠） |
| GET | `/v2/cars/:CarID/analytics/driving` | `v2_handler.go:296` | **problematic**（§2.1 avg_speed） |
| GET | `/v2/cars/:CarID/analytics/driving/timeseries` | `v2_handler.go:331` | clean |
| GET | `/v2/cars/:CarID/analytics/charging` | `v2_handler.go:377` | **problematic**（§1.4 + §2.1 avg_power） |
| GET | `/v2/cars/:CarID/analytics/charging/curve` | `v2_charging_curve.go:160` | **problematic**（§1.5） |
| GET | `/v2/cars/:CarID/analytics/efficiency` | `v2_efficiency.go:371` | **problematic**（§1.4 同款 + 死字段） |
| GET | `/v2/cars/:CarID/analytics/parking` | `v2_handler.go:435` | clean |
| GET | `/v2/cars/:CarID/parking/idle_periods` | `v2_parking_idle_periods.go` | clean |
| GET | `/v2/cars/:CarID/analytics/battery` | `v2_handler.go:487` | **removable**（§1.3） |
| GET | `/v2/cars/:CarID/analytics/battery/timeseries` | `v2_handler.go:522` | **removable / 重写**（§1.2） |
| GET | `/v2/cars/:CarID/battery/capacity_by_mileage` | `v2_battery_capacity_by_mileage.go:165` | clean |
| GET | `/v2/cars/:CarID/analytics/environmental` | `v2_environmental.go` | **problematic**（§2.1） |
| GET | `/v2/cars/:CarID/lifecycle/odometer_series` | `v2_lifecycle_extras.go:149` | clean（与 by_period 有重叠） |
| GET | `/v2/cars/:CarID/lifecycle/places` | `v2_lifecycle_extras.go:499` | **problematic**（§2.4 + §2.5） |
| GET | `/v2/cars/:CarID/summary/by_period` | `v2_lifecycle_extras.go:362` | clean |
| GET | `/v2/geofences` | `v2_lifecycle_extras.go:631` | clean |
| GET | `/v2/cars/:CarID/analytics/cost` | `v2_handler.go:566` | **problematic**（§2.1） |
| GET | `/v2/cars/:CarID/updates` | `v2_handler.go:613` | clean |
| GET | `/v2/cars/:CarID/lifecycle` | `v2_handler.go:644` | clean（meta TZ 硬编码） |
| GET | `/v2/cars/:CarID/timeline` | `v2_handler.go:689` | clean（同上 + before/after 校验） |

---

## 4. 优化方案与执行计划

按破坏性 / 收益分四批落地，每一批可独立合入并发版本号。

### 阶段 A — 兼容性修复（无破坏性，先做）

1. **修 `group_by` 静默忽略（§1.4）**
   - `v2_handler.go::handleChargingAnalytics`（`377` 起）与 `handleEfficiencyAnalytics`（`v2_efficiency.go:371` 起）：
     - 收到 `group_by` 但未 `include=timeseries`/`include=buckets` 时返回 400，错误体含 `code: GROUP_BY_REQUIRES_INCLUDE`。
     - 在 swagger 注解里写清楚依赖。
   - 清理 `V2EfficiencyResponse.Timeseries` 死字段（`v2_efficiency.go:24`）—— 要么填上，要么删字段。

2. **修 charging curve 默认时间窗（§1.5）**
   - `v2_charging_curve.go:160` 起：当客户端未传 `start`/`end`/`period` 时，默认全生命周期；保留显式传参时按窗过滤。
   - 每桶补 `min_power` / `max_power`。

3. **修硬编码时区与货币（§2.4）**
   - `v2_common.go:187` `Currency` 默认值改为读 `settings.currency`；新增 `CURRENCY` 环境变量覆盖。
   - `v2_handler.go:667, 723`、`v2_lifecycle_extras.go:524, 642` 的 `Timezone: "UTC"` 改为读 `timeRange.Timezone`，缺省回退到 `apicommon.AppUsersTimezone`。
   - `v2_lifecycle_extras.go:438` `last_visited` 走 RFC3339 + 时区。

4. **修 capabilities 漂移（§2.6）**
   - `v2_handler.go:208` 改造为根据实际路由表生成（或在每个 service 注册 `Capability()`，handler 聚合）。

5. **修 v1 tire-pressure 时区（§1.1 子项）**
   - SQL 中的日期分桶用 `date AT TIME ZONE $tz`；`tz` 从 `apicommon.AppUsersTimezone.String()` 注入。

**风险**：响应里多/少字段属于增量；`group_by` 严格化是软破坏，但旧客户端如果本来就被静默丢弃，不会回归。

### 阶段 B — 字段语义修正（小幅破坏）

1. **v1 tire-pressure 历史 `*_avg` → `*_min`/`*_max`（§1.1）**
   - 同时返回 `*_min`、`*_max`，保留 `*_avg` 一个版本作过渡，并在 swagger 上标记 deprecated。
   - 下个 minor 版本删 `*_avg`。

2. **`avg_speed` 距离加权（§2.1）**
   - `v2_driving_repository.go:87`：`AVG(60.0 * distance / NULLIF(duration_min, 0))` → `60.0 * SUM(distance) / NULLIF(SUM(duration_min), 0)`。
   - `summary.avg_speed` 字段语义不变，但现在确实是平均速度。

3. **`avg_power` 区分 AC / DC（§2.1）**
   - `v2_charging_repository.go:61` 拆为 `avg_power_ac` / `avg_power_dc` + 各自的 `median_power_*`。
   - 旧 `avg_power` 字段保留一个版本。

4. **environmental timeseries 增加 min/max + 海拔变化（§2.1）**
   - `v2_environmental.go:193-199` 在 `*_avg` 旁增加 `*_min` / `*_max`；驾驶聚合补 `elevation_gain` / `elevation_loss`。

5. **cost：补 `min`/`p50`，并对 `cost_per_distance` 做窗口对齐说明**（§2.1）
   - 在 swagger 注解里明确 `cost_per_distance = 同窗内充电费用 / 同窗内行驶距离`，并说明跨期失真的风险；推荐配合 `period=year` 使用。

### 阶段 C — 端点合并 / 删除（破坏性，新版本）

1. **删 `/v2/analytics/battery`，把字段并入 v1（§1.3）**
   - v1 `battery-health` 增加 `baseline_range_at_full_charge` 和 `estimated_range_degradation`。
   - v2 端点保留一个版本作 deprecation，响应头加 `Deprecation: true`。

2. **重写 `/v2/analytics/battery/timeseries`（§1.2）**
   - 候选 A：直接删除，改用 `/v2/timeline?include_battery_levels=true`。
   - 候选 B：保留路径，响应改为 SoC 事件流（结构见 §1.2）。
   - 推荐 A，避免一个接口两套语义。

3. **`/v2/lifecycle/odometer_series` 标记 deprecated**（§2.2）
   - 提示客户端切到 `/v2/summary/by_period`。

### 阶段 D — 体验性补强（独立可做）

1. `/v2/lifecycle/places` 增加 `start` / `end` 过滤（§2.5）。
2. `/v2/timeline` 校验 `before` 与 `after` 互斥或要求 `before > after`（§2.5）。
3. `/v2/analytics/driving/timeseries` 在 `granularity=auto` 且窗口 ≤ 90 天时直接返回 per-drive，避免聚合丢信息（§2.3）。

---

## 5. 文档与 swagger 配套

每条改动都需要同步：
1. 修改对应 handler 的 swagger 注解（`@Summary`、`@Description`、`@Param`、新增/弃用字段）。
2. 在 `docs/CHANGELOG.md`（若存在；否则新增）中按 minor 版本归档破坏性变更。
3. `make docs` 重新生成 `internal/docs/generated/swagger.{json,yaml}`，并 commit。
4. 阶段 C 之前发一个 deprecation notice 的 minor 版本，下个 minor 再真正删除。

---

## 6. 建议的合入顺序

```
v2.1.x  阶段 A（兼容性修复 + 时区/货币 + capabilities 修正）
v2.2.0  阶段 B（字段语义修正，破坏性向后兼容）
v2.3.0  阶段 C（合并/删除端点；先 deprecate）
v2.4.0  正式删除阶段 C 的旧端点；阶段 D 体验补强可穿插
```

每个阶段配套 PR 和测试用例（`v2_*_service_test.go` 已经覆盖了主要路径，按字段维度补 case 即可）。
