# V2 API 重构方案（破坏性精简版）

> 文档日期：2026-05-11
> 范围：仅 V2，允许破坏性修改；V1 完全不动。
> 目标：**端点收敛、模型去重、数据量友好、性能可控**。
> 输入：[src/generated/swagger.yaml](../src/generated/swagger.yaml)（24 个 v2 端点 + 70+ DTO）+ TeslaMate 实际数据规模。
> 与 [v2-optimization.md](v2-optimization.md) 互补：本文档负责**接口收敛和数据量约束**，那份文档负责**实现层 SQL 与并发**。

---

## 0. 设计原则

1. **能合并就合并**：用查询参数（`breakdown`、`group_by`、`fields`、`include`）取代相似端点。
2. **能并集就并集**：每个领域只保留一份 Summary DTO，不再有"短版"和"长版"两套。
3. **大数据量端点必须强制约束**：`positions` 相关查询不允许无界扫描；列表必须分页；`lifetime` 周期不允许跨表 raw scan。
4. **客户端能自描述**：响应统一带 `kind`，列表统一字段名 `items`，比较块同构于 summary。
5. **删除 > 优化**：能删除的端点和字段直接删，不留半实现。

---

## 1. 数据量与性能基线

| 表 | 体量（5 年活跃用户） | 端点能否 raw scan |
|----|--------------------|-----------------|
| `positions` | 1 GB / 30,000 km，5 GB+ | **❌ 全表扫描禁区** |
| `drives` | 20-60 行/月，约 3,600 行 | ✅ 可全量聚合 |
| `charging_processes` | 10-30 行/月，约 1,800 行 | ✅ 可全量聚合 |
| `states` | 每次状态切换一行，~10K 行 | ✅ 可全量聚合（带 car_id 过滤） |
| `updates` | <100 行 | ✅ 可全量 |

**直接结论**：
- 任何**默认按月**的分析端点，应基于 `drives` / `charging_processes` / `states`，不涉及 positions。
- 仅以下场景才允许触达 `positions`：
  - 取**最新一行**（`ORDER BY date DESC LIMIT 1`）。
  - 用户**显式指定**了 ≤7 天的窗口且需要轨迹。
- `lifetime` 周期默认指向 `cars` 表上的预聚合列（`total_distance` 等）和**已聚合的 drives/charging**，不做 5 年级别的实时聚合。

这条性能基线决定了下面 §2 的端点取舍。

---

## 2. 端点取舍：从 24 收敛到 11

### 2.1 删除清单（直接消失，不替代）

| 删除端点 | 理由 |
|---------|------|
| `/v2/cars/{CarID}/analytics/charging/cost` | 与 `/analytics/cost` 字段 95% 重合，5 个 DTO 重复（见 §3.1） |
| `/v2/cars/{CarID}/analytics/charging/locations` | 合并入 `/analytics/charging?breakdown=location` |
| `/v2/cars/{CarID}/analytics/charging/types` | 合并入 `/analytics/charging?breakdown=charger_type` |
| `/v2/cars/{CarID}/analytics/parking/locations` | 合并入 `/analytics/parking?breakdown=location` |
| `/v2/cars/{CarID}/analytics/parking/states` | 合并入 `/analytics/parking?breakdown=state` |
| `/v2/cars/{CarID}/analytics/efficiency/factors` | 与 distribution 同质，合并入 `/analytics/driving?breakdown=temperature\|speed\|...`（efficiency 整个域并入 driving，见 2.3） |
| `/v2/cars/{CarID}/analytics/efficiency` | 与 driving summary 字段 90% 重合，并入 driving |
| `/v2/cars/{CarID}/analytics/battery/distribution` | 仪表盘用例少；如需要可后续作为 driving 的 breakdown 补回 |
| `/v2/cars/{CarID}/analytics/driving/distribution` | 合并入 `/analytics/driving?breakdown=hour\|weekday\|...` |
| `/v2/cars/{CarID}/analytics/driving/ranking` | 这是查询 drives 表 + sort + limit 的轻量需求，移到 `/v1/cars/{CarID}/drives?sort=...&limit=...` 更合适，V2 不重复造（见 §2.5） |
| `/v2/cars/{CarID}/calendar` | calendar 是 timeseries 的"按天 group_by"特例，去掉，合并入 `/analytics/summary?group_by=day` 或 timeline 的滚动汇总 |
| `/v2/cars/{CarID}/insights` | 启发式规则，输出主观判断，违反"客观事实"原则，删除（见 §2.6） |
| `/v2/cars/{CarID}/reports` | 是 summary + driving + charging + updates 的串联 facade，客户端可自行串联 1 次请求即可（见 §2.7） |

**删除后剩余 11 个端点**（见 §2.8）。

---

### 2.2 端点统一形态

剩余的 analytics 端点**全部遵循一个 schema**：

```
GET /v2/cars/{CarID}/analytics/{domain}
    ?period=day|week|month|quarter|year|custom        # 默认 month；不再有 lifetime
    ?start=...&end=...                                  # period=custom 时必填
    ?timezone=Asia/Shanghai                             # 默认 TZ env
    ?compare=none|previous_period|previous_year         # 仅 summary 输出比较块
    ?include=summary,timeseries,breakdown               # 默认 summary；多选合并一次响应
    ?group_by=day|week|month|quarter|year               # 仅 timeseries
    ?breakdown=location|charger_type|state|hour|weekday # 仅 breakdown（domain 决定可选项）
    ?fields=distance_km,duration_min                    # 字段裁剪
```

`{domain}` ∈ `{driving, charging, parking, battery, cost}`（5 个域，不含 efficiency/updates）。

为什么用 `include` 而不是为每个域开 4 条子路径：
- 客户端常见需求是"summary + timeseries"一次拿到 → 减少一次 RTT。
- 服务端可在同一事务下并发 SQL → 减少 CarExists、时间范围解析的重复开销。
- DTO 数量从"每域 ×4"降到"每域 ×1"。

---

### 2.3 efficiency 并入 driving

`V2EfficiencySummary` 与 `V2DrivingAnalyticsSummary` 字段对比：

| 字段 | driving | efficiency |
|------|---------|-----------|
| `avg_consumption_wh_per_km` | ✅ | ✅ |
| `avg_speed_kmh` | ✅ | ✅ |
| `distance_km` | ✅ | ✅ |
| `drive_count` | ✅ | ✅ |
| `estimated_energy_consumed_kwh` | ✅ | ✅ |
| `best_consumption_wh_per_km` | ❌ | ✅ |
| `worst_consumption_wh_per_km` | ❌ | ✅ |
| `avg_temperature_c` | ❌ | ✅ |

差异只有 3 个字段。**直接合并到 driving summary**，删除 `/analytics/efficiency` 整域。

---

### 2.4 updates 和 lifecycle 简化

**`/analytics/updates`**：

数据量极小（<100 行/车终生），所有"按周期聚合"几乎无意义——一年也就几次更新。
保留为 `/v2/cars/{CarID}/updates`（不在 analytics 下，不接受 period/compare），返回更新历史 + 摘要：

```jsonc
{
  "data": {
    "kind": "update_list",
    "items": [/* V2UpdateVersion */],
    "latest": { "version": "...", "completed_at": "..." }
  }
}
```

删除 `V2UpdateAnalyticsResponse` / `V2UpdateSummary` / `V2UpdateAnalyticsAPIResponse`（被一个 list 端点取代）。

**`/analytics/lifecycle`**：

按 [v2-optimization.md](v2-optimization.md) P0-3 修复语义，但放到 `/v2/cars/{CarID}/lifecycle`（不在 analytics 下）。
不接受 period（语义本就不是周期分析），仅 `?as_of=`。优先读取 `cars` 表预聚合字段，避免对 drives/charging_processes 做生命周期级聚合。

---

### 2.5 ranking 移除（不是 V2 的职责）

`/analytics/driving/ranking` 实际是"按某指标排序的 drives 列表 + 分组"。这是**列表查询**，不是分析聚合，应在 V1 的 `/v1/cars/{CarID}/drives` 上加 `sort` + `limit`，而不是 V2 仿造一份 ranking DTO。

V2 不收。

---

### 2.6 insights 移除（违反客观性原则）

`/insights` 输出 `severity` `category` `description` 等字段——**输出主观判断**与项目"只输出客观事实"原则冲突（见 [docs/v2-optimization.md](v2-optimization.md) §"设计原则"）。
而它的输入又只是 summary 的对比，客户端拿到 `comparison` 块后自行判断更合适。删除整端点，删除 `V2Insight` / `V2InsightResponse` / `V2InsightAPIResponse`。

---

### 2.7 reports 移除（纯 facade）

`/reports` 唯一价值是"一次请求拿多个域"——这正是 §2.2 的 `include` 参数干的事：

```
GET /v2/cars/{CarID}/analytics/summary?include=driving,charging,parking,battery,cost
```

删除 `/reports`、`V2ReportResponse`、`V2ReportSection`、`V2ReportAPIResponse`。

---

### 2.8 重构后的端点全集（11 条）

```
GET /v2/capabilities                                           # 取代 /v2

# 跨域汇总（必须配合 fields 控制返回大小）
GET /v2/cars/{CarID}/analytics/summary
    ?include=driving,charging,parking,battery,cost
    &fields=driving.distance_km,charging.session_count,...
    &compare=previous_period

# 5 个领域分析（统一形态）
GET /v2/cars/{CarID}/analytics/driving        ?include=summary,timeseries,breakdown&...
GET /v2/cars/{CarID}/analytics/charging       ?include=summary,timeseries,breakdown&...
GET /v2/cars/{CarID}/analytics/parking        ?include=summary,timeseries,breakdown&...
GET /v2/cars/{CarID}/analytics/battery        ?include=summary,timeseries&...
GET /v2/cars/{CarID}/analytics/cost           ?include=summary,timeseries,breakdown&...

# 列表型
GET /v2/cars/{CarID}/timeline                 ?type=&before=&after=&limit=    # 游标分页
GET /v2/cars/{CarID}/updates                  # 全量轻量列表

# 终生快照
GET /v2/cars/{CarID}/lifecycle                ?as_of=
```

11 个端点，覆盖原 24 个端点的全部能力。

---

## 3. 数据模型清理

### 3.1 删除清单（共 28 个 DTO）

```
# Cost 重复
V2ChargingCostAPIResponse / V2ChargingCostResponse / V2ChargingCostLocationItem
V2ChargingCostPeriodItem / V2ChargingCostSummary
V2CostSummaryDetails / V2CostLocationItem / V2CostPeriodItem (并入通用 BreakdownItem)

# Summary 双套
V2BatteryAnalyticsSummary / V2ChargingAnalyticsSummary / V2DrivingAnalyticsSummary
V2ParkingAnalyticsSummary

# 域专用 distribution / location / type / state（合并入通用 Breakdown）
V2BatteryDistributionAPIResponse / V2BatteryDistributionResponse / V2BatteryDistributionItem
V2ChargingLocationsAPIResponse / V2ChargingLocationsResponse / V2ChargingLocationItem
V2ChargingTypesAPIResponse / V2ChargingTypesResponse / V2ChargingTypeItem
V2ParkingLocationsAPIResponse / V2ParkingLocationsResponse / V2ParkingLocationItem
V2ParkingStatesAPIResponse / V2ParkingStatesResponse / V2ParkingStateItem
V2DrivingDistributionAPIResponse / V2DrivingDistributionResponse / V2DrivingDistributionItem
V2EfficiencyFactorsAPIResponse / V2EfficiencyFactorsResponse / V2EfficiencyFactorItem

# Efficiency 整域
V2EfficiencyAPIResponse / V2EfficiencyResponse / V2EfficiencySummary

# Ranking
V2DrivingRankingAPIResponse / V2DrivingRankingResponse / V2DrivingRankingItem

# Insights / Reports
V2InsightAPIResponse / V2InsightResponse / V2Insight
V2ReportAPIResponse / V2ReportResponse / V2ReportSection

# Calendar
V2CalendarAPIResponse / V2CalendarResponse / V2CalendarDay / V2ActivityLevel

# Updates 双套
V2UpdateAnalyticsAPIResponse / V2UpdateAnalyticsResponse / V2UpdateSummary
```

### 3.2 保留 + 重命名（核心 DTO）

```
V2APIResponse                # { data, meta } 通用包装
V2Meta
V2Unit
V2ComparisonValue
APIErrorResponse / APIErrorBody

# 5 个领域 Summary（每域一份，取并集字段）
V2DrivingSummary             # 含原 efficiency 的 best/worst/avg_temperature
V2ChargingSummary
V2ParkingSummary
V2BatterySummary
V2CostSummary                # 含 data_scope

# 通用结构
V2TimeseriesPoint            # { period_start, metrics: { [key]: number } }
V2BreakdownEntry             # { key, label, metrics: { [key]: number } }
V2DomainAnalyticsResponse    # { kind, summary?, timeseries?, breakdown?, comparison? }
V2SummaryResponse            # 跨域 summary（已有）

# 列表型
V2TimelineResponse           # { kind, items, next_cursor, has_more }
V2UpdateListResponse         # { kind, items, latest }
V2LifecycleResponse          # { as_of, ...cumulative fields }

# Capabilities
V2CapabilitiesResponse       # { version, domains: [...], breakdown_options: {...} }
```

DTO 总数从 70+ 收敛到 ~20。

### 3.3 通用 timeseries / breakdown 结构

旧的 `V2DrivingTimeseriesItem` 含 `avg_speed_kmh / distance_km / drive_count` 等 ~7 个字段；charging 版本含另一套 ~7 个字段。客户端需要为每个域维护一个 schema。

新结构（自描述，所有域共用）：

```jsonc
{
  "kind": "timeseries",
  "domain": "driving",
  "group_by": "day",
  "metric_keys": ["distance_km", "duration_min", "drive_count"],
  "items": [
    { "period_start": "2026-05-01T00:00:00+08:00",
      "metrics": { "distance_km": 123.4, "duration_min": 45.6, "drive_count": 3 } },
    ...
  ]
}
```

`metric_keys` 提供给客户端用作图例，`metrics` 是 `map<string, number>`。客户端通用渲染逻辑可复用。

同样的 shape 用于 breakdown：

```jsonc
{
  "kind": "breakdown",
  "domain": "charging",
  "by": "location",
  "metric_keys": ["session_count", "energy_added_kwh", "cost"],
  "items": [
    { "key": "geofence:42", "label": "Home",
      "metrics": { "session_count": 14, "energy_added_kwh": 412.3, "cost": 320.5 } },
    ...
  ]
}
```

### 3.4 comparison 与 summary 同构

旧设计：`comparison` 是 `map<string, V2ComparisonValue>`，键名靠文档字符串约定。
新设计：`comparison` 与 summary **路径同构**，客户端可用同一套 selector：

```jsonc
{
  "summary": {
    "driving":  { "distance_km": 1234, "duration_min": 678, ... },
    "charging": { "session_count": 12, ... }
  },
  "comparison": {
    "driving":  { "distance_km":  { "current":1234, "previous":1100, "delta":134, "delta_percent":12.2 } },
    "charging": { "session_count": { ... } }
  }
}
```

---

## 4. 性能护栏（写入到设计层）

### 4.1 路径级约束

| 端点 | 默认 period | 是否允许 `lifetime` | 是否触达 positions |
|------|-------------|---------------------|-------------------|
| `/analytics/summary` | month | ❌ | 否 |
| `/analytics/{domain}` | month | ❌ | 否 |
| `/timeline` | — | — | 否 |
| `/updates` | — | — | 否 |
| `/lifecycle` | — | ✅（这是它的语义） | 仅取最新一行 |

`lifetime` 从 period 枚举中**移除**。终生统计只走 `/lifecycle` 端点（读 `cars` 预聚合 + 边界点查），不在通用 analytics 端点上做 5 年级聚合。

### 4.2 列表强制分页

| 端点 | 分页方式 | 默认 limit | 最大 limit |
|------|---------|-----------|-----------|
| `/timeline` | cursor (`before` / `after`) | 50 | 200 |
| `/updates` | 全量（数据量天然小） | — | — |

### 4.3 字段裁剪默认行为

`/analytics/summary?include=driving,charging,...` 在不传 `fields` 时，每个域只返回**核心字段集**（5 个左右），不返回所有字段。这是与现状破坏性的差异，但避免了 dashboard 一次请求拉回 50+ 字段的浪费。

完整字段需显式 `fields=driving.*` 或 `fields=driving.distance_km,driving.duration_min,...`。

### 4.4 SQL 层强制（实现期）

参考 [v2-optimization.md](v2-optimization.md) P0-1：所有命中 positions 的查询必须带 `car_id = $1 AND date >= $2 AND date < $3`，且时间窗口 ≤ 7 天，否则返回 400。

---

## 5. 兼容性与迁移

V2 还未稳定承诺，本方案**不提供兼容层**。

| 旧路径 | 新路径 |
|--------|--------|
| `/analytics/charging/cost` | 删除（用 `/analytics/cost?breakdown=charger_type`） |
| `/analytics/charging/locations` | `/analytics/charging?include=breakdown&breakdown=location` |
| `/analytics/charging/types` | `/analytics/charging?include=breakdown&breakdown=charger_type` |
| `/analytics/parking/locations` | `/analytics/parking?include=breakdown&breakdown=location` |
| `/analytics/parking/states` | `/analytics/parking?include=breakdown&breakdown=state` |
| `/analytics/driving/distribution` | `/analytics/driving?include=breakdown&breakdown=hour\|weekday\|...` |
| `/analytics/driving/ranking` | 移到 V1 drives 列表查询 |
| `/analytics/driving/timeseries` | `/analytics/driving?include=timeseries&group_by=day` |
| `/analytics/charging/timeseries` | `/analytics/charging?include=timeseries&group_by=day` |
| `/analytics/battery/timeseries` | `/analytics/battery?include=timeseries&group_by=day` |
| `/analytics/battery/distribution` | 删除 |
| `/analytics/efficiency` | 并入 `/analytics/driving` |
| `/analytics/efficiency/factors` | `/analytics/driving?include=breakdown&breakdown=temperature\|...` |
| `/analytics/locations` | 删除（按域用 `breakdown=location`） |
| `/analytics/cost?group_by=...` | `/analytics/cost?include=timeseries&group_by=...` |
| `/calendar` | `/analytics/summary?group_by=day&period=...` |
| `/insights` | 删除 |
| `/reports` | `/analytics/summary?include=driving,charging,...` |
| `/analytics/updates` | `/updates`（轻量列表） |
| `/analytics/lifecycle` | `/lifecycle?as_of=` |

---

## 6. 收益清单

| 维度 | 改动前 | 改动后 |
|------|--------|--------|
| V2 端点数 | 24 | **11**（-54%） |
| V2 DTO 数 | 70+ | **~20**（-71%） |
| 重复 Summary 类型 | 9 个 | **0** |
| 列表字段命名 | `items / events / days / insights / sections / versions / cost_by_*` 7 种 | **统一 items**（带 `kind`） |
| `lifetime` 周期跨表实时聚合风险 | 存在（任何 analytics 端点都接受） | **移除**（仅 `/lifecycle` 走预聚合） |
| 客户端域级 schema 数量 | 每域独立 timeseries/distribution/breakdown DTO | **跨域共用** timeseries/breakdown shape |
| comparison 弱类型 | map + 文档字符串约定 | **与 summary 同构** |

---

## 7. 推进顺序

```
阶段一 — 模型并集与命名统一（不动路由）
  3.1 删 Summary 双套 → 3.3 通用 timeseries/breakdown DTO → 3.4 comparison 同构

阶段二 — 路由收敛（破坏性集中）
  2.1 删除 13 端点 → 2.2 引入 include/breakdown 参数化 → 2.3 efficiency 并入 driving
  → 2.4 updates/lifecycle 移出 analytics

阶段三 — 性能护栏与文档
  4.1 移除 lifetime period → 4.3 字段裁剪默认行为 → 4.4 positions 时间窗口校验
  → 更新 capabilities → CHANGELOG 标注破坏性变更
```

阶段一对外可见结构最小，可以先合代码再合文档；阶段二必须在一个 release 中原子完成，避免客户端踩到中间态。

---

## 8. 不做的事（明确否定）

- ❌ **不引入 GraphQL/JSON-API 风格的稀疏字段集嵌套语法**：`fields=driving.distance_km` 已经够用。
- ❌ **不做 BFF 层的复合端点**（如 `/dashboard/home`）：让客户端用 `include` 自己组合。
- ❌ **不做 server-side cache 层**：HTTP `Cache-Control` 已能解决（见 [v2-optimization.md](v2-optimization.md) P2-2）。
- ❌ **不为对称而对称补端点**：每个 domain 是否支持 timeseries/breakdown 由 `/v2/capabilities` 声明，而非一刀切。
