# V2 API 优化执行进度

> 开始日期：2026-05-08
> 基于 docs/v2-optimization.md 优化方案

## 执行规则

- 每完成一项更新本文件
- 每项完成后执行 `go build ./...` 和 `go test ./...`
- P0 全部完成后做一次 curl 验证
- 最终创建独立 Git commit

## P0 — 必须修复

| ID | 问题 | 状态 | 文件 | Commit |
|----|------|------|------|--------|
| P0-1 | positions 表扫描性能风险 | Pending | v2_battery_repository.go, v2_efficiency_repository.go | |
| P0-2 | /timeline 缺少游标分页 | Pending | v2_handler.go, v2_lifecycle_repository.go | |
| P0-3 | /lifecycle 忽略时间参数 | Pending | v2_handler.go, v2_lifecycle_repository.go, v2_lifecycle_service.go | |
| P0-4 | /insights 劫持 compare 参数 | Pending | v2_insight_service.go | |

## P1 — 应该修复

| ID | 问题 | 状态 | 文件 | Commit |
|----|------|------|------|--------|
| P1-1 | /reports 串行 N 次查询 | Pending | v2_report_service.go | |
| P1-2 | /reports include 模块不完整 | Pending | v2_report_service.go | |
| P1-3 | CarExists 重复查询 | Pending | v2_handler.go | |

## 详细记录

### P0-3：lifecycle 忽略时间参数

**改动**：
- 去掉 period/start/end 参数处理
- 新增 `as_of` 可选参数（RFC3339），默认为当前时间
- lifecycle 统计从 first_recorded_at 累计到 as_of 时刻

**接口变更**：
```
GET /cars/{CarID}/analytics/lifecycle           → 全量累计至今
GET /cars/{CarID}/analytics/lifecycle?as_of=... → 累计至指定时刻
```

**状态**：Pending

---

### P0-4：insights 劫持 compare 参数

**改动**：
- 内部新增 `buildBaselinePeriod(timeRange)` 方法
- 不再修改传入的 `timeRange.Compare`
- 调用方 compare=none 不受 insights 内部逻辑影响

**状态**：Pending

---

### P0-1：positions 表扫描

**改动**：
- battery distribution 改为基于 `drives.start_battery_level / end_battery_level`
- battery timeseries 改为基于 drives 聚合字段
- 仅 battery summary 的"最新电量"使用 positions 点查，带 LIMIT 1
- efficiency summary/factors 已基于 drives 表（已OK）
- 必要索引声明到文档

**状态**：Pending

---

### P0-2：timeline 游标分页

**改动**：
- 新增 `before` / `after` 游标参数（RFC3339 时间戳）
- 响应增加 `next_cursor`、`has_more`、`total` 字段
- SQL 改为基于时间戳范围过滤，不用 OFFSET

**状态**：Pending

---

### P1-3：CarExists 中间件

**改动**：
- 新增 `V2CarValidationMiddleware(db)` gin 中间件
- 路由组 `/cars/:CarID` 统一挂载
- 各 service 不再调用 `CarExists`

**状态**：Pending

---

### P1-1：reports 并发化

**改动**：
- 用 goroutine + channel 并发调用各 section builder
- 查询时间从串行总和降为最慢单次耗时

**状态**：Pending

---

### P1-2：reports include 模块补全

**改动**：
- 补全 parking / battery / efficiency / cost / locations / insights 6 个模块
- 每个 section 复用现有 analytics service，不重复 SQL
- 无数据时隐藏该 section（与现有行为一致）

**状态**：Pending

---

## curl 验证命令

```bash
# lifecycle as_of
curl -i 'http://localhost:8080/api/v2/cars/1/analytics/lifecycle'
curl -i 'http://localhost:8080/api/v2/cars/1/analytics/lifecycle?as_of=2026-04-30T23:59:59%2B08:00'

# timeline 游标
curl -i 'http://localhost:8080/api/v2/cars/1/timeline?limit=5'
curl -i 'http://localhost:8080/api/v2/cars/1/timeline?limit=5&before=2026-05-01T00:00:00%2B08:00'

# insights（compare 参数不被劫持）
curl -i 'http://localhost:8080/api/v2/cars/1/insights?period=month&compare=none&timezone=Asia/Shanghai'

# reports 完整 include
curl -i 'http://localhost:8080/api/v2/cars/1/reports?period=month&include=summary,driving,charging,parking,battery,efficiency,cost,locations,updates&timezone=Asia/Shanghai'
```
