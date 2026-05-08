# V2 API 优化方案

> 文档日期：2026-05-08
> 基于对 TeslaMate 数据模型、真实使用场景和当前 V2 实现的综合分析。

---

## 背景与约束

### 真实使用方

| 客户端 | 使用方式 | 关键需求 |
|--------|----------|----------|
| Home Assistant / 自动化 | MQTT 实时状态 + 偶尔拉历史聚合 | 低延迟、稳定 |
| MateDroid 等移动 App | 列表 + 周期汇总 | 延迟敏感、数据准确 |
| 自定义 Dashboard | 一次渲染多维度数据 | 减少请求数、可组合 |
| Grafana | 直接查 PostgreSQL，不走 API | 不受影响 |

### 数据量特征

| 表 | 特征 | 影响 |
|----|------|------|
| `positions` | 绝对瓶颈：~1 GB / 30,000 km；5 年活跃用户可超 5 GB | 任何扫描该表的查询都需严格限制范围 |
| `drives` | 20-60 条/月，已聚合 | 轻量，适合作为统计基础 |
| `charging_processes` | 10-30 条/月 | 轻量 |
| `states` | 每次状态切换一行，数量中等 | UNION 查询时注意索引 |
| `updates` | 极少（数条/年） | 可忽略 |

### 设计原则（不变）

1. 不破坏任何 V1 API
2. 所有统计只输出客观事实，不含主观建议
3. 时区默认读取 `TZ` 环境变量，请求 `timezone` 参数优先
4. 数据库过滤使用 UTC 半开区间
5. 估算字段使用 `estimated_` 前缀或 `data_quality.warnings` 说明

---

## 优化清单

### P0 — 必须修复（影响正确性或生产稳定性）

---

#### P0-1：`positions` 表扫描风险

**问题**

`/analytics/battery` 和 `/analytics/efficiency` 的底层查询若全表扫描 `positions`，在大数据量下（5 GB+）会导致超时，影响所有 API 用户。

**根因**

Battery 分布和效率因素分析倾向于扫描 positions 的 `battery_level`、`outside_temp` 等字段，而不是使用已聚合的 `drives` 表。

**改进方案**

- Battery 分析的**历史趋势和分布**改为基于 `drives.start_battery_level / end_battery_level`（已聚合，行数少）
- 仅"最新电量"使用 positions 点查：`SELECT ... FROM positions WHERE car_id = $1 AND date < $2 ORDER BY date DESC LIMIT 1`，依赖 `(car_id, date DESC)` 索引
- 效率因素（温度、速度分段）同样改为基于 `drives` 的聚合字段，不扫描 positions
- 所有涉及 positions 的查询必须携带 `car_id = $1 AND date >= $2 AND date < $3` 的精确时间范围条件

**必要索引（须在文档中声明）**

```sql
-- positions 最新值点查
CREATE INDEX IF NOT EXISTS idx_positions_car_date ON positions (car_id, date DESC);

-- drives 时间范围过滤
CREATE INDEX IF NOT EXISTS idx_drives_car_start ON drives (car_id, start_date);

-- charging_processes 时间范围过滤
CREATE INDEX IF NOT EXISTS idx_charging_car_start ON charging_processes (car_id, start_date);

-- states 时间范围过滤（timeline UNION 使用）
CREATE INDEX IF NOT EXISTS idx_states_car_start ON states (car_id, start_date);
```

---

#### P0-2：`/timeline` 缺少游标分页

**问题**

`/timeline` 当前只有 `limit`（最大 200），无法翻页。有多年数据的用户第 201 条事件永远取不到。UNION 查询在大数据集上也是重查询，需要分页控制扫描范围。

**改进方案**

改为游标分页（cursor-based pagination），避免 `OFFSET` 的全表扫描问题：

```
GET /cars/{CarID}/timeline?limit=50&before=2026-04-01T00:00:00+08:00
GET /cars/{CarID}/timeline?limit=50&after=2026-01-01T00:00:00+08:00
```

响应中增加游标字段：

```json
{
  "data": {
    "events": [...],
    "total": 1240,
    "next_cursor": "2026-03-15T08:23:00+08:00",
    "has_more": true
  }
}
```

SQL 层改为基于时间戳的范围过滤（利用索引），而非 `OFFSET N`：

```sql
-- 向前翻页（before 游标）
WHERE event_time < $cursor ORDER BY event_time DESC LIMIT $limit

-- 向后翻页（after 游标）  
WHERE event_time > $cursor ORDER BY event_time ASC LIMIT $limit
```

---

#### P0-3：`/analytics/lifecycle` 忽略时间参数

**问题**

当前实现完全忽略传入的 `period / start / end` 参数，总是返回全量累计数据。但 API 仍然接受这些参数，对调用方造成误导（传了参数不生效，没有任何提示）。

**改进方案（选一）**

**方案 A（推荐）**：明确语义，去掉 period/start/end，增加可选的 `as_of` 参数：

```
GET /cars/{CarID}/analytics/lifecycle
  → 返回从第一条记录至今的全量累计统计

GET /cars/{CarID}/analytics/lifecycle?as_of=2026-04-30T23:59:59+08:00
  → 返回截至该时刻的累计统计（用于历史对比）
```

**方案 B**：保留时间参数，但语义改为"该时间段内的增量统计"（而非生命周期累计），需要更新 Swagger 说明和响应字段命名。

**方案 A 的 Swagger 更新**

```go
// @Param as_of query string false "Cutoff datetime (RFC3339). Defaults to now. Returns cumulative stats up to this point."
// @Description Returns cumulative lifetime statistics from the first recorded event. Use as_of to get a historical snapshot.
```

---

#### P0-4：`/insights` 劫持 `compare` 参数

**问题**

当用户传 `compare=none`（默认值）时，Insights 服务内部强制将 `timeRange.Compare` 改为 `previous_period`，违背参数承诺，且修改了调用方传入的值（副作用）。

**改进方案**

Insights 内部维护独立的对比周期计算，不依赖也不修改 `timeRange.Compare`：

```go
func (s V2InsightService) buildBaselinePeriod(timeRange V2TimeRange) V2TimeRange {
    // 内部计算，永远不修改传入的 timeRange
    duration := timeRange.End.Sub(timeRange.Start)
    prevEnd := timeRange.Start
    prevStart := prevEnd.Add(-duration)
    return V2TimeRange{
        Start:    prevStart,
        End:      prevEnd,
        Timezone: timeRange.Timezone,
        Compare:  "none",
    }
}
```

调用方的 `compare=none` 仍然生效（不影响 Insights 自身的内部基线计算）。

---

### P1 — 应该修复（影响性能和功能完整性）

---

#### P1-1：`/reports` 串行 N 次数据库调用

**问题**

Reports 服务顺序调用 summary → driving → charging → updates 四个 service，每个再各自查库，共约 8-12 次 SQL 往返。在高延迟链路下累积效应明显。

**改进方案**

改为 goroutine 并发调用：

```go
type reportResult struct {
    section V2ReportSection
    err     error
}

results := make(chan reportResult, len(modules))

for _, mod := range modules {
    go func(mod string) {
        section, err := s.buildSection(ctx, mod, carIDParam, timeRange)
        results <- reportResult{section, err}
    }(mod)
}

// 收集结果
for range modules {
    r := <-results
    if r.err != nil { ... }
    if r.section != nil { sections = append(sections, *r.section) }
}
```

查询时间从串行总和降为最慢单次耗时（通常 driving 查询最慢，其他并发执行）。

---

#### P1-2：`/reports` 的 `include` 模块不完整

**问题**

TASKS.md 要求 `include` 支持 10 个模块：`summary / driving / charging / parking / battery / efficiency / cost / locations / updates / insights`。当前只实现了 4 个：`summary / driving / charging / updates`。其余 6 个模块传入后被静默忽略，没有报错也没有说明。

**改进方案**

补全 6 个缺失模块的实现，或在当前版本中明确返回未实现提示：

```go
case "parking", "battery", "efficiency", "cost", "locations", "insights":
    // 返回占位 section，data_quality.warnings 说明暂未实现
    sections = append(sections, V2ReportSection{
        Type:  mod,
        Title: moduleTitle(mod),
        DataQuality: V2DataQuality{
            Complete: false,
            Warnings: []string{"This section is not yet implemented in the current version."},
        },
    })
```

同时更新 Swagger 文档，明确标注哪些模块已实现、哪些计划中。

---

#### P1-3：`CarExists` 重复查询

**问题**

每个 Service 都独立调用 `CarExists`，Reports 这类聚合 API 一次请求会执行 4+ 次 `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`。

**改进方案**

在路由层加中间件，统一处理 CarID 校验：

```go
// middleware
func V2CarValidationMiddleware(db *sql.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        carIDStr := c.Param("CarID")
        carID, err := strconv.ParseInt(carIDStr, 10, 64)
        if err != nil || carID <= 0 {
            v2Error(c, 400, "INVALID_CAR_ID", "invalid car id", nil)
            c.Abort()
            return
        }
        var exists bool
        db.QueryRowContext(c.Request.Context(),
            `SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)`, carID,
        ).Scan(&exists)
        if !exists {
            v2Error(c, 404, "CAR_NOT_FOUND", "car not found", nil)
            c.Abort()
            return
        }
        c.Set("carID", carID)
        c.Next()
    }
}

// 路由注册
v2Cars := v2.Group("/cars/:CarID", V2CarValidationMiddleware(db))
v2Cars.GET("/analytics/summary", handlers.Summary)
// ...
```

各 Service 从 Context 直接读取已验证的 carID，无需再次查库。

---

### P2 — 建议改进（提升可用性和扩展性）

---

#### P2-1：行驶/充电分析增加 `sections` 参数

**背景**

Dashboard 渲染行驶分析页面通常同时需要 summary + timeseries + distribution，当前需要 3 次独立请求。

**方案**

在保留所有现有独立端点的基础上，为 `/analytics/driving` 增加可选的 `sections` 参数：

```
GET /analytics/driving
  → 只返回 summary（默认，最快）

GET /analytics/driving?sections=summary,timeseries&group_by=day
  → 返回 summary + timeseries，一次请求

GET /analytics/driving?sections=summary,distribution&dimension=hour_of_day
  → 返回 summary + distribution
```

这是业界标准的"稀疏字段集"模式（参考 GitHub API 的 `expand[]`、Stripe 的 `expand[]`），不是粗暴合并：
- 独立端点保持不变，已有调用不受影响
- `sections` 是增量功能，内部仍然是独立的 SQL 查询
- 减少 Dashboard 的 HTTP 往返次数

同样适用于 `/analytics/charging`。

---

#### P2-2：增加 HTTP 缓存控制头

**背景**

Home Assistant 等客户端定期轮询，历史周期数据（上个月的统计）是不变的，但当前所有响应都没有 `Cache-Control` 头，每次都重新查库。

**方案**

```go
func v2CacheControl(c *gin.Context, timeRange V2TimeRange) {
    now := time.Now().UTC()
    if timeRange.End.Before(now.Add(-24 * time.Hour)) {
        // 历史期间（结束时间超过 24 小时前）：可缓存 1 小时
        c.Header("Cache-Control", "public, max-age=3600")
        c.Header("Vary", "Accept-Encoding")
    } else {
        // 含当前时刻的期间：不缓存
        c.Header("Cache-Control", "no-cache, no-store")
    }
}
```

对于月报、年报等完全历史数据，`max-age=3600` 可显著降低数据库负载。

---

#### P2-3：补全 `/analytics/lifecycle` 的 `as_of` 参数

见 P0-3，此处只列出需要补充的响应字段：

```json
{
  "data": {
    "first_recorded_at": "2021-03-15T08:00:00+08:00",
    "as_of": "2026-04-30T23:59:59+08:00",
    "recorded_days": 1872,
    "odometer_start_km": 0,
    "odometer_latest_km": 98432.5,
    "odometer_delta_km": 98432.5,
    "drive_count": 2841,
    "distance_km": 98432.5,
    "charging_session_count": 683,
    "energy_added_kwh": 14820.3,
    "cost": 9234.5,
    "update_count": 47,
    "avg_daily_distance_km": 52.6,
    "avg_monthly_distance_km": 1600.8,
    "avg_consumption_wh_per_km": 168.2,
    "cost_per_100km": 9.38
  }
}
```

---

### P3 — 低优先级（文档和语义澄清）

---

#### P3-1：`/charging/cost` 与 `/cost` 语义澄清

**现状**：两个端点在当前实现中统计内容高度重叠（都只统计充电费用）。

**建议**：
- `/analytics/cost` 在 `data_scope` 中明确说明当前只覆盖充电成本，`excluded` 字段列出保险、停车、维保等；未来加入其他成本类型时不需要改 URL，只扩展 `included` 列表
- `/analytics/charging/cost` 专注充电维度的费用细分（按地点、按时段、按充电桩类型），与 `/analytics/cost` 的定位区分开

文档中加一句说明：`/analytics/cost` 是成本汇总入口（未来可扩展），`/analytics/charging/cost` 是充电费用的深度分析。

---

#### P3-2：`/reports` 的 `title` 字段改为结构化

**现状**：`"title": "Report for car 1"` — 硬编码字符串，前端无法直接使用。

**建议**：

```json
{
  "data": {
    "period_label": "2026年5月",
    "period": "month",
    "start": "2026-05-01T00:00:00+08:00",
    "end": "2026-06-01T00:00:00+08:00",
    "car_id": 1,
    "sections": [...]
  }
}
```

`period_label` 由服务端根据 `period` 和 `timezone` 生成本地化格式字符串，前端也可根据 `start/end` 自行渲染。

---

## 优先级汇总

| ID | 问题 | 影响范围 | 改动成本 | 优先级 |
|----|------|----------|----------|--------|
| P0-1 | positions 表扫描性能风险 | 生产超时 | 中（改 SQL + 索引文档） | **P0** |
| P0-2 | timeline 无游标分页 | 数据截断，功能缺失 | 中（加游标逻辑） | **P0** |
| P0-3 | lifecycle 忽略时间参数 | API 语义错误 | 低（文档 + 加 as_of） | **P0** |
| P0-4 | insights 劫持 compare 参数 | 参数语义破坏 | 低（改内部逻辑） | **P0** |
| P1-1 | reports 串行 N 次查询 | 高延迟 | 低（改并发） | **P1** |
| P1-2 | reports include 模块不完整 | 功能缺失 | 高（补 6 个模块） | **P1** |
| P1-3 | CarExists 重复查询 | 轻微性能浪费 | 低（加中间件） | **P1** |
| P2-1 | driving/charging 缺少 sections 参数 | Dashboard 多次往返 | 中（增量功能） | **P2** |
| P2-2 | 缺少 HTTP Cache-Control | 轮询负载浪费 | 低（加 header） | **P2** |
| P2-3 | lifecycle as_of 参数 | 历史快照能力缺失 | 低（改参数定义） | **P2** |
| P3-1 | cost 端点语义重叠 | 文档混淆 | 低（只改文档） | **P3** |
| P3-2 | reports title 不结构化 | 前端使用不便 | 低（改响应字段） | **P3** |

---

## 建议执行顺序

```
阶段一（P0，正确性修复）：
  P0-3 lifecycle 语义 → P0-4 insights 参数 → P0-1 positions SQL → P0-2 timeline 分页

阶段二（P1，性能与完整性）：
  P1-3 CarExists 中间件 → P1-1 reports 并发 → P1-2 reports 模块补全

阶段三（P2，可用性提升）：
  P2-2 HTTP 缓存头 → P2-3 lifecycle as_of → P2-1 sections 参数

阶段四（P3，文档与细节）：
  P3-1 P3-2 文档和字段调整
```

P0-3 和 P0-4 改动成本最低，建议优先完成，快速消除 API 语义错误，再处理 SQL 性能问题。
