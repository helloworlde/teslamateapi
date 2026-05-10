# TeslaMateApi V2 Analytics API 实现任务、计划与 Codex Prompt

## 0. 总目标

基于当前 `teslamateapi` 项目新增 `/api/v2` API，只实现 V1 不支持或不适合承载的能力：

```text
统计
分析
趋势
分布
排行
生命周期汇总
日历聚合
报表聚合
客观洞察
```

V2 不实现：

```text
车辆命令控制
日志开关
V1 已有的车辆、行程、充电、更新明细基础查询
直接暴露数据库表
主观建议
未经事实数据支撑的结论
```

所有 V2 API 必须满足：

```text
1. 有 Swagger 注释
2. 可被 swag 生成 OpenAPI/Swagger 文档
3. 可通过 github.com/watchakorn-18k/scalar-go 渲染 API Reference
4. 有统一响应结构
5. 有统一错误结构
6. 有测试
7. 每完成一个任务，更新 docs/v2-progress.md
8. 每完成一个任务，必须执行检查、启动服务、curl 验证、单独 Git commit，然后才能继续下一个任务
```

---

## 1. 强制执行流程

每个任务 T00 ~ T14 完成后，必须严格执行以下流程。

### 1.1 代码检查

至少执行：

```bash
gofmt -w .
go test ./...
go build ./...
swag init
```

如果项目不支持 `gofmt -w .` 直接执行，则改为对实际 Go 文件执行：

```bash
find . -name '*.go' -not -path './vendor/*' -print0 | xargs -0 gofmt -w
```

任何一步失败，都必须修复后重新执行。

---

### 1.2 按实际环境启动服务

使用以下命令启动项目：

```bash
TZ=Asia/Shanghai \
DATABASE_USER=teslamate \
DATABASE_PASS=Ihaveapen1! \
DATABASE_NAME=teslamate \
DATABASE_HOST=192.168.2.7 \
DATABASE_PORT=5433 \
MQTT_HOST=saas.host.homelab \
go run ./src
```

如果服务端口来自项目配置，读取项目实际默认端口。后续 curl 验证必须使用该实际端口。

---

### 1.3 curl 验证

每完成一个任务后，必须用 `curl` 验证该任务涉及的 API 能正常返回。

最低要求：

```bash
curl -i http://localhost:<PORT>/api/v2
curl -i http://localhost:<PORT>/api/docs/swagger.json
curl -i http://localhost:<PORT>/api/docs/scalar
```

对于具体任务 API，必须补充对应 curl，例如：

```bash
curl -i 'http://localhost:<PORT>/api/v2/cars/1/analytics/summary?period=month&timezone=Asia/Shanghai'
```

验证要求：

```text
1. HTTP 状态码符合预期
2. JSON 格式正确
3. 错误响应符合统一错误结构
4. Swagger JSON 可访问
5. Scalar 页面可访问
6. 本任务新增 API 出现在 Swagger/Scalar 文档中
7. HTTP 响应内容存在有效数据
```

---

### 1.4 更新进度文档

每个任务完成后，必须更新：

```text
docs/v2-progress.md
```

记录内容包括：

```text
任务编号
任务状态
涉及 API
Swagger 是否完成
Scalar 是否可见
测试是否通过
curl 验证命令
Git commit hash
备注
```

---

### 1.5 单任务 Git 提交

每完成一个任务并完成检查、启动、curl 验证、进度记录后，必须立即为该任务创建单独 Git commit。

提交格式：

```bash
git status
git add .
git commit -m "feat(v2): implement Txx <task name>"
```

示例：

```bash
git commit -m "feat(v2): implement T00 base framework and scalar docs"
git commit -m "feat(v2): implement T01 summary analytics"
```

约束：

```text
1. 不允许多个任务混在同一个 commit
2. 不允许未检查通过就 commit
3. 不允许未 curl 验证就 commit
4. 不允许未更新 docs/v2-progress.md 就 commit
5. 当前任务 commit 完成后，才能继续下一个任务
```

---

## 2. 时间与时区强制要求

所有涉及以下能力的 V2 API 都必须正确处理时区：

```text
时间范围
时间分桶
时间输出
同比计算
环比计算
生命周期统计
日历聚合
报表周期
洞察对比周期
```

### 2.1 默认时区来源

默认时区必须从环境变量 `TZ` 读取：

```bash
TZ=Asia/Shanghai
```

规则：

```text
1. 请求没有显式传入 timezone 时，使用环境变量 TZ
2. 环境变量 TZ 为空时，使用 UTC，并在代码中有明确 fallback
3. 请求显式传入 timezone 时，优先使用请求参数 timezone
4. timezone 必须是合法 IANA timezone，例如 Asia/Shanghai、Asia/Singapore、UTC
5. 非法 timezone 返回 400 INVALID_TIMEZONE
```

---

### 2.2 本地时间边界

所有周期边界必须以业务时区的本地时间计算。

例如：

```http
GET /api/v2/cars/1/analytics/summary?period=day&start=2026-05-01&timezone=Asia/Shanghai
```

应解释为：

```text
Asia/Shanghai 本地时间 2026-05-01 00:00:00
到
Asia/Shanghai 本地时间 2026-05-02 00:00:00，不含结束点
```

再转换为数据库过滤使用的 UTC 时间范围。

---

### 2.3 数据库 UTC 过滤

数据库时间过滤必须遵循：

```text
1. API 输入按请求时区解析
2. 本地时间边界按请求时区计算
3. 查询数据库前转换为 UTC
4. SQL 使用 UTC 范围过滤
5. 输出时间再转换回请求时区
```

推荐半开区间：

```sql
WHERE start_date >= $utc_start
  AND start_date <  $utc_end
```

禁止使用容易重复/漏算的闭区间：

```sql
WHERE start_date >= $start
  AND start_date <= $end
```

除非有明确理由并有测试覆盖边界。

---

### 2.4 时间分桶

按日、周、月、季度、年分桶时，必须基于请求时区计算。

例如：

```text
group_by=day
timezone=Asia/Shanghai
```

结果中的 `period_start` / `period_end` 必须是 Asia/Shanghai 本地时间边界。

---

### 2.5 同比/环比

对比周期必须基于请求时区计算。

规则：

| compare | 说明 |
|---|---|
| `previous_period` | 当前本地周期向前平移一个等长周期 |
| `previous_year` | 当前本地周期向前平移一年 |
| `lifetime_average` | 当前周期与生命周期平均周期比较 |

示例：

```text
当前周期：2026-05-01 00:00:00 ~ 2026-06-01 00:00:00 Asia/Shanghai
previous_period：2026-04-01 00:00:00 ~ 2026-05-01 00:00:00 Asia/Shanghai
previous_year：2025-05-01 00:00:00 ~ 2025-06-01 00:00:00 Asia/Shanghai
```

---

### 2.6 时区测试要求

所有涉及时间的 API 必须至少覆盖：

```text
1. 使用环境变量 TZ 作为默认时区
2. 请求 timezone 覆盖 TZ
3. 非法 timezone 返回 400
4. day 本地边界正确
5. month 本地边界正确
6. previous_period 对比周期正确
7. 输出时间包含正确时区偏移
```

---

## 3. 实现约束

### 3.1 兼容性约束

```text
1. 不破坏任何现有 V1 API
2. 不修改 V1 响应结构
3. 不删除现有路由
4. V2 使用 /api/v2 前缀
5. V2 可以新增代码目录、模型、服务、仓储、测试
```

---

### 3.2 数据原则

V2 所有统计必须是客观事实数据。

允许输出：

```text
本月行驶 1420.5 km，较上月增加 9.17%
本月充电 18 次，充入 420.5 kWh
本月平均能耗 171.0 Wh/km
Office 地点停车耗电 2.1%/day
```

禁止输出：

```text
你的驾驶习惯很好
建议减少快充
电池很健康
这是不好的用车方式
```

如果指标是估算值，必须明确字段命名或返回 `data_quality.warnings`。

例如：

```text
estimated_energy_consumed_kwh
estimated_range_degradation_percent
estimated_vampire_drain_kwh
```

---

### 3.3 文档约束

每个 API 必须包含 Swagger 注释：

```go
// @Summary ...
// @Description ...
// @Tags V2 Analytics
// @Produce json
// @Param CarID path int true "Car ID" default(1)
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param compare query string false "Comparison mode" Enums(none, previous_period, previous_year, lifetime_average)
// @Success 200 {object} model.V2SummaryAPIResponse
// @Failure 400 {object} model.APIErrorResponse
// @Failure 404 {object} model.APIErrorResponse
// @Failure 500 {object} model.APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/summary [get]
```

如果当前 Swagger 泛型支持不好，使用非泛型 wrapper DTO。

---

### 3.4 Scalar 文档约束

必须新增或确认以下文档入口：

```text
/api/docs
/api/docs/swagger.json
/api/docs/scalar
```

要求：

```text
1. /api/docs/swagger.json 返回当前 OpenAPI/Swagger JSON
2. /api/docs/scalar 使用 github.com/watchakorn-18k/scalar-go 渲染文档
3. Scalar 页面能展示 V1 + V2 所有 API
4. V2 API tag 清晰分组
```

推荐 Tags：

```text
V2 Summary
V2 Driving Analytics
V2 Charging Analytics
V2 Parking Analytics
V2 Battery Analytics
V2 Efficiency Analytics
V2 Cost Analytics
V2 Location Analytics
V2 Update Analytics
V2 Lifecycle
V2 Calendar
V2 Reports
V2 Insights
```

---

## 4. 推荐代码结构

根据项目现有结构调整，不要强行重构全项目。优先新增 V2 独立结构。

```text
docs/
  v2-api.md
  v2-progress.md
  v2-openapi-notes.md

internal/ 或 pkg/ 或当前项目既有目录/
  model/
    v2_common.go
    v2_summary.go
    v2_driving.go
    v2_charging.go
    v2_parking.go
    v2_battery.go
    v2_efficiency.go
    v2_cost.go
    v2_location.go
    v2_update.go
    v2_lifecycle.go
    v2_calendar.go
    v2_report.go
    v2_insight.go

  handler/
    v2_summary_handler.go
    v2_driving_handler.go
    v2_charging_handler.go
    v2_parking_handler.go
    v2_battery_handler.go
    v2_efficiency_handler.go
    v2_cost_handler.go
    v2_location_handler.go
    v2_update_handler.go
    v2_lifecycle_handler.go
    v2_calendar_handler.go
    v2_report_handler.go
    v2_insight_handler.go

  service/
    v2_time_range.go
    v2_summary_service.go
    v2_driving_service.go
    v2_charging_service.go
    v2_parking_service.go
    v2_battery_service.go
    v2_efficiency_service.go
    v2_cost_service.go
    v2_location_service.go
    v2_update_service.go
    v2_lifecycle_service.go
    v2_calendar_service.go
    v2_report_service.go
    v2_insight_service.go

  repository/
    v2_summary_repository.go
    v2_driving_repository.go
    v2_charging_repository.go
    v2_parking_repository.go
    v2_battery_repository.go
    v2_efficiency_repository.go
    v2_cost_repository.go
    v2_location_repository.go
    v2_update_repository.go
    v2_lifecycle_repository.go

  router/
    v2.go

  docs/
    swagger.go
    scalar.go
```

如果项目已有对应目录，遵循现有风格，不重复造架构。

---

## 5. 通用模型

### 5.1 通用查询参数

所有 V2 分析接口统一支持：

```go
type V2AnalyticsQuery struct {
    Period   string `form:"period" json:"period" example:"month"`
    Start    string `form:"start" json:"start" example:"2026-05-01T00:00:00+08:00"`
    End      string `form:"end" json:"end" example:"2026-05-31T23:59:59+08:00"`
    Timezone string `form:"timezone" json:"timezone" example:"Asia/Shanghai"`
    Compare  string `form:"compare" json:"compare" example:"previous_period"`
    GroupBy  string `form:"group_by" json:"group_by,omitempty" example:"day"`
    Metrics  string `form:"metrics" json:"metrics,omitempty" example:"distance_km,duration_min"`
    Include  string `form:"include" json:"include,omitempty" example:"summary,comparison,timeseries"`
}
```

---

### 5.2 通用响应结构

```go
type V2Meta struct {
    CarID       int64          `json:"car_id"`
    Period      string         `json:"period"`
    Timezone    string         `json:"timezone"`
    Start       string         `json:"start,omitempty"`
    End         string         `json:"end,omitempty"`
    Compare     string         `json:"compare,omitempty"`
    Unit        V2Unit         `json:"unit"`
    GeneratedAt string         `json:"generated_at"`
    DataQuality *V2DataQuality `json:"data_quality,omitempty"`
}

type V2Unit struct {
    Distance    string `json:"distance" example:"km"`
    Energy      string `json:"energy" example:"kWh"`
    Power       string `json:"power" example:"kW"`
    Temperature string `json:"temperature" example:"C"`
    Currency    string `json:"currency" example:"CNY"`
}

type V2DataQuality struct {
    Complete      bool     `json:"complete"`
    SampleCount   int64    `json:"sample_count,omitempty"`
    MissingFields []string `json:"missing_fields,omitempty"`
    Warnings      []string `json:"warnings,omitempty"`
}

type V2ComparisonValue struct {
    Current      *float64 `json:"current,omitempty"`
    Previous     *float64 `json:"previous,omitempty"`
    Delta        *float64 `json:"delta,omitempty"`
    DeltaPercent *float64 `json:"delta_percent,omitempty"`
}

type APIErrorResponse struct {
    Error APIErrorBody `json:"error"`
}

type APIErrorBody struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
}
```

---

## 6. 进度记录要求

新增文件：

```text
docs/v2-progress.md
```

内容模板：

```markdown
# V2 API Implementation Progress

## Rules

- Each completed task must update this file.
- Each API must have handler, service, repository, response DTO, Swagger comments, and tests.
- Each API must be visible in Scalar documentation.
- Each task must pass gofmt, go test ./..., go build ./..., swag init, service startup, and curl validation before commit.
- Each task must have a dedicated Git commit.

## Progress

| Task | Status | Commit | APIs | Swagger | Scalar | Tests | Build | Curl | Notes |
|---|---|---|---|---|---|---|---|---|---|
| T00 | Pending |  | V2 base framework | No | No | No | No | No |  |
```

---

# 7. 任务拆分

## T00：V2 基础框架、统一响应、Swagger + Scalar

### 目标

建立 V2 基础能力，不实现复杂业务统计。

### 需要完成

```text
1. 新增 /api/v2 health/info 路由
2. 新增 V2 router group
3. 新增通用 query parser
4. 新增统一 response/error helper
5. 新增 period/compare/timezone 校验
6. 接入 Swagger 文档生成
7. 接入 github.com/watchakorn-18k/scalar-go
8. 新增 /api/docs/swagger.json
9. 新增 /api/docs/scalar
10. 初始化 docs/v2-progress.md
```

### API

```http
GET /api/v2
```

### 验收标准

```text
gofmt 通过
go test ./... 通过
go build ./... 通过
swag init 通过
服务可按指定环境变量启动
curl /api/v2 正常返回
curl /api/docs/swagger.json 正常返回
curl /api/docs/scalar 正常返回
docs/v2-progress.md 已创建
T00 已单独 Git commit
```

### Codex Prompt

```text
你现在在 teslamateapi 项目中工作。请实现 T00：V2 基础框架、统一响应、Swagger + Scalar 文档入口。

要求：
1. 不破坏任何现有 V1 API。
2. 新增 /api/v2 路由，返回 V2 能力说明。
3. 新增 V2 通用响应结构、错误结构、时间周期参数结构。
4. 新增 period 校验：day/week/month/quarter/year/custom/lifetime。
5. 新增 compare 校验：none/previous_period/previous_year/lifetime_average。
6. 新增 timezone 解析：默认读取环境变量 TZ；请求传 timezone 时优先使用请求值；非法 timezone 返回 400 INVALID_TIMEZONE。
7. 接入 Swagger 文档生成，保留现有文档能力。
8. 使用 github.com/watchakorn-18k/scalar-go 新增 Scalar API Reference 页面。
9. 新增 /api/docs/swagger.json 和 /api/docs/scalar，确保 Scalar 使用 swagger.json。
10. 所有新增 API 必须有 Swagger 注释。
11. 新增 docs/v2-progress.md，并记录 T00 状态。
12. 添加必要测试。
13. 执行 gofmt、go test ./...、go build ./...、swag init。
14. 使用以下命令启动服务并用 curl 验证 /api/v2、/api/docs/swagger.json、/api/docs/scalar：

TZ=Asia/Shanghai \
ENCRYPTION_KEY=teslamate \
DATABASE_USER=teslamate \
DATABASE_PASS=pass \
DATABASE_NAME=teslamate \
DATABASE_HOST=192.168.2.7 \
DATABASE_PORT=5433 \
MQTT_HOST=saas.host.homelab \
go run ./src

15. 全部通过后更新 docs/v2-progress.md，记录 curl 命令和结果。
16. 为 T00 创建独立 Git commit：feat(v2): implement T00 base framework and scalar docs。
17. commit 完成后再继续后续任务。

不要实现具体 analytics 业务。只做基础框架和文档入口。
```

---

## T01：周期总览 Summary API

### API

```http
GET /api/v2/cars/{CarID}/analytics/summary
```

### 功能

返回指定周期的客观总览统计。

### 包含模块

```text
driving
charging
parking
battery
updates
cost
```

### 指标

```text
driving.drive_count
driving.distance_km
driving.duration_min
driving.max_speed_kmh
driving.avg_consumption_wh_per_km

charging.session_count
charging.energy_added_kwh
charging.energy_used_kwh
charging.duration_min
charging.cost

parking.parked_duration_min
parking.asleep_duration_min
parking.online_duration_min
parking.offline_duration_min
parking.vampire_drain_percent

battery.latest_battery_level_percent
battery.latest_rated_range_km
battery.latest_ideal_range_km

updates.update_count
updates.latest_version

cost.charging_cost
cost.cost_per_km
cost.cost_per_100km
```

### 验收标准

```text
1. 支持 period/start/end/timezone/compare 参数
2. 所有时间边界默认使用环境变量 TZ
3. 请求 timezone 可覆盖 TZ
4. 数据库过滤使用 UTC 时间范围
5. compare=previous_period 可返回 comparison
6. 无数据时返回 0 或 null，不能 500
7. Swagger 可见
8. Scalar 可见
9. curl 验证 summary API
10. T01 单独 Git commit
```

### Codex Prompt

```text
请实现 T01：V2 周期总览 Summary API。

API：
GET /api/v2/cars/{CarID}/analytics/summary

目标：
返回指定周期内车辆使用的客观统计汇总，只基于 TeslaMate PostgreSQL 中已有事实数据，不输出主观建议。

要求：
1. 使用 T00 中的 V2 通用响应、错误结构和查询参数解析。
2. 支持 period/start/end/timezone/compare。
3. period 支持 day/week/month/quarter/year/custom/lifetime。
4. 默认 timezone 必须从环境变量 TZ 读取。
5. 请求显式传入 timezone 时，优先使用请求 timezone。
6. period/start/end 必须先按请求时区计算本地边界，再转换为 UTC 查询数据库。
7. 输出时间必须转换回请求时区。
8. compare=previous_period 必须基于请求时区计算上一周期。
9. compare 支持 none/previous_period/previous_year/lifetime_average，至少实现 none 和 previous_period，其余可返回明确的 not implemented 错误或预留。
10. 返回 driving、charging、parking、battery、updates、cost 六个模块。
11. 所有数值字段使用明确单位后缀，例如 distance_km、duration_min、energy_added_kwh。
12. 估算字段必须使用 estimated 前缀，或者在 data_quality.warnings 中说明。
13. 无数据时返回空统计，不要报错。
14. 添加 Swagger 注释，Tag 使用 V2 Summary。
15. 确保 Scalar 页面能显示该 API。
16. 添加 handler/service/repository 分层，不要把 SQL 写在 handler。
17. 添加测试，覆盖正常查询、无数据、非法 period、非法 car id、TZ 默认时区、请求 timezone 覆盖、previous_period 边界。
18. 执行 gofmt、go test ./...、go build ./...、swag init。
19. 使用指定环境变量启动服务，用 curl 验证该 API 可正常返回。
20. 更新 docs/v2-progress.md 中 T01 的状态、curl 结果和 commit 信息。
21. 为 T01 创建独立 Git commit：feat(v2): implement T01 summary analytics。
```

---

## T02：行驶分析 Driving Analytics

### APIs

```http
GET /api/v2/cars/{CarID}/analytics/driving
GET /api/v2/cars/{CarID}/analytics/driving/timeseries
GET /api/v2/cars/{CarID}/analytics/driving/distribution
GET /api/v2/cars/{CarID}/analytics/driving/ranking
```

### 核心指标

```text
drive_count
distance_km
duration_min
avg_distance_km
avg_duration_min
max_speed_kmh
avg_speed_kmh
estimated_energy_consumed_kwh
avg_consumption_wh_per_km
range_loss_km
battery_level_used_percent
estimated_regenerated_energy_kwh
avg_outside_temp_c
total_ascent_m
total_descent_m
```

### Codex Prompt

```text
请实现 T02：V2 行驶分析 Driving Analytics。

APIs：
1. GET /api/v2/cars/{CarID}/analytics/driving
2. GET /api/v2/cars/{CarID}/analytics/driving/timeseries
3. GET /api/v2/cars/{CarID}/analytics/driving/distribution
4. GET /api/v2/cars/{CarID}/analytics/driving/ranking

目标：
提供 V1 不具备的行驶统计、趋势、分布和排行能力。

要求：
1. 只基于 TeslaMate 已记录的 drives、positions、cars、geofences、addresses 等事实数据。
2. 不返回主观评价，不给驾驶建议。
3. 所有字段必须带明确单位后缀。
4. 能耗、电量、回收能量等非直接字段必须使用 estimated 前缀或 data_quality.warnings 说明。
5. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
6. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
7. /driving 返回周期汇总和 comparison。
8. /timeseries 按 group_by 或 period 返回 day/week/month/year 趋势，分桶边界基于请求时区。
9. /distribution 支持 dimension：hour_of_day、day_of_week、distance_bucket、duration_bucket、speed_bucket、consumption_bucket、temperature_bucket。
10. /ranking 支持 type：longest_distance、longest_duration、highest_speed、lowest_consumption、highest_consumption、highest_distance_day。
11. 返回结构必须适合图表直接使用。
12. 添加完整 Swagger 注释，Tags 使用 V2 Driving Analytics。
13. 确保所有 API 出现在 Scalar 文档中。
14. 添加测试：正常统计、无行程数据、非法 dimension、非法 ranking type、非法 period、TZ 默认时区、timezone 覆盖、分桶边界。
15. 执行 gofmt、go test ./...、go build ./...、swag init。
16. 使用指定环境变量启动服务，用 curl 验证本任务全部 API 可正常返回。
17. 更新 docs/v2-progress.md 中 T02 的状态、curl 结果和 commit 信息。
18. 为 T02 创建独立 Git commit：feat(v2): implement T02 driving analytics。
```

---

## T03：充电分析 Charging Analytics

### APIs

```http
GET /api/v2/cars/{CarID}/analytics/charging
GET /api/v2/cars/{CarID}/analytics/charging/timeseries
GET /api/v2/cars/{CarID}/analytics/charging/locations
GET /api/v2/cars/{CarID}/analytics/charging/types
GET /api/v2/cars/{CarID}/analytics/charging/cost
```

### 指标

```text
session_count
energy_added_kwh
energy_used_kwh
charging_efficiency_percent
duration_min
avg_duration_min
avg_power_kw
max_power_kw
cost
avg_cost_per_kwh
start_battery_avg_percent
end_battery_avg_percent
ac_session_count
dc_session_count
ac_energy_kwh
dc_energy_kwh
```

### Codex Prompt

```text
请实现 T03：V2 充电分析 Charging Analytics。

APIs：
1. GET /api/v2/cars/{CarID}/analytics/charging
2. GET /api/v2/cars/{CarID}/analytics/charging/timeseries
3. GET /api/v2/cars/{CarID}/analytics/charging/locations
4. GET /api/v2/cars/{CarID}/analytics/charging/types
5. GET /api/v2/cars/{CarID}/analytics/charging/cost

目标：
提供充电统计、趋势、地点、类型和成本分析。

要求：
1. 只基于 charging_processes、charges、geofences、addresses 中的事实数据。
2. 不输出主观建议。
3. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
4. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
5. /charging 返回周期汇总和 comparison。
6. /timeseries 返回 day/week/month/year 聚合趋势。
7. /locations 按 geofence/address 聚合充电次数、电量、费用、效率。
8. /types 按 ac/dc/supercharger/unknown 聚合。
9. /cost 返回充电费用、单价、每公里成本、每百公里成本。
10. 如果 cost 数据缺失，返回 null 或 0，并在 data_quality.missing_fields 或 warnings 中说明。
11. 充电效率使用 energy_added_kwh / energy_used_kwh，energy_used_kwh 缺失时不可伪造。
12. 所有字段带单位后缀。
13. 添加完整 Swagger 注释，Tag 使用 V2 Charging Analytics。
14. 确保 Scalar 页面可见。
15. 添加测试：正常统计、无充电数据、缺失 cost、缺失 energy_used、非法 period、TZ 默认时区、timezone 覆盖、分桶边界。
16. 执行 gofmt、go test ./...、go build ./...、swag init。
17. 使用指定环境变量启动服务，用 curl 验证本任务全部 API 可正常返回。
18. 更新 docs/v2-progress.md 中 T03 的状态、curl 结果和 commit 信息。
19. 为 T03 创建独立 Git commit：feat(v2): implement T03 charging analytics。
```

---

## T04：停放与状态分析 Parking Analytics

### APIs

```http
GET /api/v2/cars/{CarID}/analytics/parking
GET /api/v2/cars/{CarID}/analytics/parking/locations
GET /api/v2/cars/{CarID}/analytics/parking/states
```

### Codex Prompt

```text
请实现 T04：V2 停放与状态分析 Parking Analytics。

APIs：
1. GET /api/v2/cars/{CarID}/analytics/parking
2. GET /api/v2/cars/{CarID}/analytics/parking/locations
3. GET /api/v2/cars/{CarID}/analytics/parking/states

目标：
统计车辆非行驶、非充电期间的停放时长、状态占比和停车耗电。

要求：
1. 停放统计只能基于 TeslaMate 已有事实数据推导。
2. 使用 states 统计 online/asleep/offline 时间。
3. 使用 drives 和 charging_processes 的时间窗口推导 parking sessions。
4. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
5. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
6. 如果 vampire drain 依赖估算，字段必须命名为 estimated_vampire_drain_kwh，且 data_quality.warnings 说明。
7. 不输出建议。
8. /parking 返回周期汇总。
9. /locations 按 geofence/address 聚合停放次数、停放时长、状态时长、停车耗电。
10. /states 返回 online/asleep/offline 时长、占比和状态切换次数。
11. 无法精确推导停放 session 时，返回已有 states 统计并在 data_quality.warnings 说明。
12. 添加完整 Swagger 注释，Tag 使用 V2 Parking Analytics。
13. 确保 Scalar 页面可见。
14. 添加测试：正常 states 统计、无 states 数据、停放 session 推导、时间窗口边界、数据缺失 warning、TZ 默认时区、timezone 覆盖。
15. 执行 gofmt、go test ./...、go build ./...、swag init。
16. 使用指定环境变量启动服务，用 curl 验证本任务全部 API 可正常返回。
17. 更新 docs/v2-progress.md 中 T04 的状态、curl 结果和 commit 信息。
18. 为 T04 创建独立 Git commit：feat(v2): implement T04 parking analytics。
```

---

## T05：电池分析 Battery Analytics

### APIs

```http
GET /api/v2/cars/{CarID}/analytics/battery
GET /api/v2/cars/{CarID}/analytics/battery/timeseries
GET /api/v2/cars/{CarID}/analytics/battery/distribution
```

### Codex Prompt

```text
请实现 T05：V2 电池分析 Battery Analytics。

APIs：
1. GET /api/v2/cars/{CarID}/analytics/battery
2. GET /api/v2/cars/{CarID}/analytics/battery/timeseries
3. GET /api/v2/cars/{CarID}/analytics/battery/distribution

目标：
基于 TeslaMate 历史电量和续航记录提供客观电池趋势分析。

要求：
1. 不声称这是官方 SOH。
2. 所有健康、退化、满电续航相关字段必须使用 estimated 前缀。
3. 必须返回 data_quality.warnings，说明该结果受温度、BMS 校准、样本分布影响。
4. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
5. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
6. /battery 返回最近电量、最近续航、估算满电续航、基准续航、估算下降比例。
7. /timeseries 按周期返回估算满电 rated/ideal range 趋势。
8. /distribution 返回电量区间分布，例如 0-10、10-20、...、90-100。
9. 样本不足时，不要输出误导性 degradation，返回 null 并在 warnings 中说明。
10. 添加完整 Swagger 注释，Tag 使用 V2 Battery Analytics。
11. 确保 Scalar 页面可见。
12. 添加测试：正常样本、样本不足、电量为 0 或 null、distribution 分桶、warning 存在、TZ 默认时区、timezone 覆盖。
13. 执行 gofmt、go test ./...、go build ./...、swag init。
14. 使用指定环境变量启动服务，用 curl 验证本任务全部 API 可正常返回。
15. 更新 docs/v2-progress.md 中 T05 的状态、curl 结果和 commit 信息。
16. 为 T05 创建独立 Git commit：feat(v2): implement T05 battery analytics。
```

---

## T06：能耗效率 Efficiency Analytics

### APIs

```http
GET /api/v2/cars/{CarID}/analytics/efficiency
GET /api/v2/cars/{CarID}/analytics/efficiency/factors
```

### Codex Prompt

```text
请实现 T06：V2 能耗效率 Efficiency Analytics。

APIs：
1. GET /api/v2/cars/{CarID}/analytics/efficiency
2. GET /api/v2/cars/{CarID}/analytics/efficiency/factors

目标：
提供能耗统计和按维度分组的能耗事实分析。

要求：
1. 只输出分组事实，不输出因果判断。
2. 不允许写“温度导致能耗升高”这类结论。
3. 可以输出“30-35°C 区间平均能耗为 181.5 Wh/km，20-25°C 区间为 158.2 Wh/km”。
4. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
5. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
6. /efficiency 返回周期总能耗、平均能耗、最佳/最差能耗、平均温度、平均速度。
7. /factors 支持 dimension：temperature、speed、distance、elevation、location、hour_of_day、day_of_week。
8. 估算能耗字段使用 estimated 前缀或 warnings。
9. 添加 Swagger 注释，Tag 使用 V2 Efficiency Analytics。
10. 确保 Scalar 页面可见。
11. 添加测试：正常统计、无行驶数据、非法 dimension、分桶正确、TZ 默认时区、timezone 覆盖。
12. 执行 gofmt、go test ./...、go build ./...、swag init。
13. 使用指定环境变量启动服务，用 curl 验证本任务全部 API 可正常返回。
14. 更新 docs/v2-progress.md 中 T06 的状态、curl 结果和 commit 信息。
15. 为 T06 创建独立 Git commit：feat(v2): implement T06 efficiency analytics。
```

---

## T07：成本分析 Cost Analytics

### API

```http
GET /api/v2/cars/{CarID}/analytics/cost
```

### Codex Prompt

```text
请实现 T07：V2 成本分析 Cost Analytics。

API：
GET /api/v2/cars/{CarID}/analytics/cost

目标：
统计 TeslaMate 可获得的用车成本。当前只统计充电费用，不包含保险、保养、停车、折旧、轮胎、维修等外部成本。

要求：
1. 返回 data_scope，明确 included 和 excluded。
2. included 至少包括 charging_cost。
3. excluded 包括 insurance、maintenance、parking、depreciation、tire、repair。
4. 返回 charging_cost、energy_used_kwh、distance_km、cost_per_kwh、cost_per_km、cost_per_100km。
5. 支持按周期聚合 cost_by_period。
6. 支持按地点聚合 cost_by_location。
7. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
8. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
9. cost 缺失时返回 data_quality.warnings。
10. 不输出“省钱”“昂贵”等主观评价。
11. 添加 Swagger 注释，Tag 使用 V2 Cost Analytics。
12. 确保 Scalar 页面可见。
13. 添加测试：正常 cost、cost 缺失、distance 为 0、energy_used 为 0、TZ 默认时区、timezone 覆盖。
14. 执行 gofmt、go test ./...、go build ./...、swag init。
15. 使用指定环境变量启动服务，用 curl 验证该 API 可正常返回。
16. 更新 docs/v2-progress.md 中 T07 的状态、curl 结果和 commit 信息。
17. 为 T07 创建独立 Git commit：feat(v2): implement T07 cost analytics。
```

---

## T08：地点分析 Location Analytics

### API

```http
GET /api/v2/cars/{CarID}/analytics/locations
```

### Codex Prompt

```text
请实现 T08：V2 地点分析 Location Analytics。

API：
GET /api/v2/cars/{CarID}/analytics/locations

目标：
按 geofence/address 聚合车辆使用行为，包括出发、到达、充电、停放和耗电。

要求：
1. 基于 drives、charging_processes、states、geofences、addresses 的事实数据。
2. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
3. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
4. 返回每个地点：location_name、geofence_id、address_id、drive_start_count、drive_end_count、charging_session_count、parking_session_count、parking_duration_min、energy_added_kwh、charging_cost、vampire_drain_percent。
5. 如果无法归属 geofence，则归入 unknown 或 address。
6. 支持 sort 参数：drive_start_count_desc、drive_end_count_desc、charging_session_count_desc、parking_duration_desc、charging_cost_desc。
7. 不输出主观结论。
8. 添加 Swagger 注释，Tag 使用 V2 Location Analytics。
9. 确保 Scalar 页面可见。
10. 添加测试：geofence 聚合、unknown 地点、sort 参数、无数据、TZ 默认时区、timezone 覆盖。
11. 执行 gofmt、go test ./...、go build ./...、swag init。
12. 使用指定环境变量启动服务，用 curl 验证该 API 可正常返回。
13. 更新 docs/v2-progress.md 中 T08 的状态、curl 结果和 commit 信息。
14. 为 T08 创建独立 Git commit：feat(v2): implement T08 location analytics。
```

---

## T09：更新分析 Update Analytics

### API

```http
GET /api/v2/cars/{CarID}/analytics/updates
```

### Codex Prompt

```text
请实现 T09：V2 更新分析 Update Analytics。

API：
GET /api/v2/cars/{CarID}/analytics/updates

目标：
统计 TeslaMate 已记录的 OTA 更新历史。

要求：
1. 基于 updates 表事实数据。
2. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
3. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
4. 返回 update_count、latest_version、latest_update_at、avg_update_duration_min、versions。
5. versions 按更新时间倒序。
6. 无更新记录时返回 update_count=0，latest_version=null。
7. 不输出版本好坏评价。
8. 添加 Swagger 注释，Tag 使用 V2 Update Analytics。
9. 确保 Scalar 页面可见。
10. 添加测试：正常更新记录、无更新记录、end_date 缺失、TZ 默认时区、timezone 覆盖。
11. 执行 gofmt、go test ./...、go build ./...、swag init。
12. 使用指定环境变量启动服务，用 curl 验证该 API 可正常返回。
13. 更新 docs/v2-progress.md 中 T09 的状态、curl 结果和 commit 信息。
14. 为 T09 创建独立 Git commit：feat(v2): implement T09 update analytics。
```

---

## T10：生命周期分析 Lifecycle Analytics

### APIs

```http
GET /api/v2/cars/{CarID}/analytics/lifecycle
GET /api/v2/cars/{CarID}/timeline
```

### Codex Prompt

```text
请实现 T10：V2 生命周期分析 Lifecycle Analytics。

APIs：
1. GET /api/v2/cars/{CarID}/analytics/lifecycle
2. GET /api/v2/cars/{CarID}/timeline

目标：
提供车辆从 TeslaMate 记录开始以来的累计统计，以及统一生命周期事件流。

要求：
1. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
2. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
3. /analytics/lifecycle 返回累计统计：first_recorded_at、last_recorded_at、recorded_days、odometer_start_km、odometer_latest_km、odometer_delta_km、drive_count、distance_km、charging_session_count、energy_added_kwh、energy_used_kwh、charging_cost、update_count、avg_daily_distance_km、avg_monthly_distance_km、avg_consumption_wh_per_km、cost_per_100km。
4. /timeline 统一返回 drive、charging、parking、update、state 事件。
5. timeline 支持 start/end/type/limit/order 参数。
6. timeline 事件必须包含 type、id、start_time、end_time、title、metrics。
7. 不返回 V1 详情，只返回适合时间线展示的摘要。
8. 不输出主观评价。
9. 添加 Swagger 注释，Tags 使用 V2 Lifecycle。
10. 确保 Scalar 页面可见。
11. 添加测试：lifecycle 累计统计、timeline 多类型事件排序、type 过滤、无数据、TZ 默认时区、timezone 覆盖。
12. 执行 gofmt、go test ./...、go build ./...、swag init。
13. 使用指定环境变量启动服务，用 curl 验证本任务全部 API 可正常返回。
14. 更新 docs/v2-progress.md 中 T10 的状态、curl 结果和 commit 信息。
15. 为 T10 创建独立 Git commit：feat(v2): implement T10 lifecycle analytics。
```

---

## T11：日历聚合 Calendar API

### API

```http
GET /api/v2/cars/{CarID}/calendar
```

### Codex Prompt

```text
请实现 T11：V2 日历聚合 Calendar API。

API：
GET /api/v2/cars/{CarID}/calendar

目标：
返回按天聚合的数据，供日历视图、热力图、日报入口使用。

要求：
1. 返回指定 start/end 范围内每天的数据。
2. 所有日期边界必须基于请求时区；默认从环境变量 TZ 读取。
3. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
4. 输出 date 必须是请求时区下的本地日期。
5. 每天包含 date、drive_count、distance_km、drive_duration_min、charging_session_count、energy_added_kwh、charging_cost、parking_duration_min、vampire_drain_percent、update_count、activity_level。
6. activity_level 是客观分级，用于 UI 显示，不是评价。
7. activity_level 可按固定阈值计算：driving 0/1/2/3/4、charging 0/1/2/3/4、parking_drain 0/1/2/3/4。
8. 如果某天无数据，也应返回该日期，数值为 0。
9. 添加 Swagger 注释，Tag 使用 V2 Calendar。
10. 确保 Scalar 页面可见。
11. 添加测试：日期连续性、无数据日期返回 0、跨月范围、activity_level 分级、TZ 默认时区、timezone 覆盖、本地日期边界。
12. 执行 gofmt、go test ./...、go build ./...、swag init。
13. 使用指定环境变量启动服务，用 curl 验证该 API 可正常返回。
14. 更新 docs/v2-progress.md 中 T11 的状态、curl 结果和 commit 信息。
15. 为 T11 创建独立 Git commit：feat(v2): implement T11 calendar analytics。
```

---

## T12：报表 Reports API

### API

```http
GET /api/v2/cars/{CarID}/reports
```

### Codex Prompt

```text
请实现 T12：V2 报表 Reports API。

API：
GET /api/v2/cars/{CarID}/reports

目标：
返回适合前端直接展示的周期报表聚合数据。报表只组织数据，不输出主观文案。

要求：
1. 支持 period=day/week/month/quarter/year/lifetime/custom。
2. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
3. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
4. 返回 title、period、sections。
5. sections 只包含有数据的模块。
6. 支持 include 参数控制模块：summary、driving、charging、parking、battery、efficiency、cost、locations、updates、insights。
7. 报表复用已实现的 analytics service，不重复写 SQL。
8. 每个 section 包含 type、title、metrics、data_quality。
9. 没有充电数据时，不返回 charging section。
10. 没有更新数据时，不返回 updates section。
11. 添加 Swagger 注释，Tag 使用 V2 Reports。
12. 确保 Scalar 页面可见。
13. 添加测试：月报、年报、include 过滤、空模块隐藏、TZ 默认时区、timezone 覆盖。
14. 执行 gofmt、go test ./...、go build ./...、swag init。
15. 使用指定环境变量启动服务，用 curl 验证该 API 可正常返回。
16. 更新 docs/v2-progress.md 中 T12 的状态、curl 结果和 commit 信息。
17. 为 T12 创建独立 Git commit：feat(v2): implement T12 reports analytics。
```

---

## T13：客观洞察 Insights API

### API

```http
GET /api/v2/cars/{CarID}/insights
```

### Codex Prompt

```text
请实现 T13：V2 客观洞察 Insights API。

API：
GET /api/v2/cars/{CarID}/insights

目标：
基于已实现的 analytics 数据生成客观、可解释、可复算的洞察。

严格要求：
1. 洞察不是建议。
2. 洞察不是主观评价。
3. 每条洞察必须有 metrics 和 evidence。
4. 每条洞察必须说明 current_period 和 baseline_period。
5. 样本不足时不得生成该洞察。
6. 所有时间范围、分桶、输出时间、同比/环比必须使用请求时区；默认从环境变量 TZ 读取。
7. 数据库过滤必须使用由本地时间边界转换得到的 UTC 半开区间。
8. 不允许出现：建议你、最好、应该、健康、不健康、好、差、合理、不合理。
9. 允许出现：增加、减少、高于、低于、占比、新高、新低、样本不足。
10. 支持 period/start/end/timezone/compare。
11. 支持 category 参数：driving、charging、parking、battery、cost、lifecycle。
12. 支持 min_severity 参数：info、warning。
13. 先实现以下洞察：driving_distance_increased、driving_distance_decreased、consumption_increased、consumption_decreased、charging_cost_increased、dc_charging_ratio_increased、vampire_drain_increased、online_duration_high、estimated_range_decreased、monthly_distance_record。
14. 其他类型保留结构，但不要伪实现。
15. 添加最小样本量规则：能耗变化至少 5 次行程且总距离 > 50 km；充电变化至少 3 次充电；停车耗电至少 3 次停放且总停放时长 > 24h；电池趋势至少 10 个有效样本。
16. 添加 Swagger 注释，Tag 使用 V2 Insights。
17. 确保 Scalar 页面可见。
18. 添加测试：正常生成洞察、样本不足不生成、禁止主观词、evidence 存在、category 过滤、TZ 默认时区、timezone 覆盖、baseline_period 正确。
19. 执行 gofmt、go test ./...、go build ./...、swag init。
20. 使用指定环境变量启动服务，用 curl 验证该 API 可正常返回。
21. 更新 docs/v2-progress.md 中 T13 的状态、curl 结果和 commit 信息。
22. 为 T13 创建独立 Git commit：feat(v2): implement T13 objective insights。
```

---

## T14：最终文档、质量检查、OpenAPI 校验

### Codex Prompt

```text
请执行 T14：V2 API 最终文档、质量检查和 OpenAPI 校验。

目标：
确认所有 V2 API 已完成，Swagger + Scalar 文档完整，测试通过，V1 未破坏。

要求：
1. 检查所有 /api/v2 API 是否都有 Swagger 注释。
2. 检查 swagger.json 是否包含所有 V2 API。
3. 检查 /api/docs/scalar 是否能展示所有 V2 API。
4. 整理 docs/v2-api.md，列出所有 V2 API、功能、参数、响应模块。
5. 整理 docs/v2-progress.md，确认所有任务状态为 Done 或注明 Partial 原因。
6. 更新 README，增加 V2 Analytics API 简介和文档入口。
7. 统一检查时区处理：所有涉及时间范围、分桶、时间输出、同比/环比的 API 都必须默认读取环境变量 TZ，请求 timezone 覆盖 TZ，本地边界转 UTC 查询，输出回请求时区。
8. 运行：
   - gofmt
   - go test ./...
   - go build ./...
   - go vet ./...
   - swag init
   - docker build
9. 使用以下命令启动服务：

TZ=Asia/Shanghai \
ENCRYPTION_KEY=teslamate \
DATABASE_USER=teslamate \
DATABASE_PASS=pass \
DATABASE_NAME=teslamate \
DATABASE_HOST=192.168.2.7 \
DATABASE_PORT=5433 \
MQTT_HOST=saas.host.homelab \
go run ./src

10. 用 curl 验证所有 V2 API 至少一次。
11. 用 curl 验证 /api/docs/swagger.json 和 /api/docs/scalar。
12. 修复所有失败。
13. 检查 V1 路由仍然存在且测试通过。
14. 不新增未完成的 API 假实现。
15. 如果某个 API 未完成，必须在 progress 文档中明确标记 Partial 和原因。
16. 全部完成后为 T14 创建独立 Git commit：docs(v2): finalize api docs and validation。
```

---

# 8. 总任务表

| 任务 | API 数 | 内容 | 优先级 | Commit |
|---|---:|---|---|---|
| T00 | 1 | V2 基础框架、Swagger、Scalar | P0 | `feat(v2): implement T00 base framework and scalar docs` |
| T01 | 1 | Summary 总览 | P0 | `feat(v2): implement T01 summary analytics` |
| T02 | 4 | Driving 行驶分析 | P0 | `feat(v2): implement T02 driving analytics` |
| T03 | 5 | Charging 充电分析 | P0 | `feat(v2): implement T03 charging analytics` |
| T11 | 1 | Calendar 日历聚合 | P0 | `feat(v2): implement T11 calendar analytics` |
| T12 | 1 | Reports 报表 | P0 | `feat(v2): implement T12 reports analytics` |
| T04 | 3 | Parking 停放分析 | P1 | `feat(v2): implement T04 parking analytics` |
| T05 | 3 | Battery 电池分析 | P1 | `feat(v2): implement T05 battery analytics` |
| T06 | 2 | Efficiency 能耗效率 | P1 | `feat(v2): implement T06 efficiency analytics` |
| T07 | 1 | Cost 成本分析 | P1 | `feat(v2): implement T07 cost analytics` |
| T08 | 1 | Locations 地点分析 | P1 | `feat(v2): implement T08 location analytics` |
| T09 | 1 | Updates 更新分析 | P1 | `feat(v2): implement T09 update analytics` |
| T10 | 2 | Lifecycle 生命周期 | P1 | `feat(v2): implement T10 lifecycle analytics` |
| T13 | 1 | Insights 客观洞察 | P2 | `feat(v2): implement T13 objective insights` |
| T14 | - | 文档、测试、质量检查 | P0 | `docs(v2): finalize api docs and validation` |

合计：

```text
V2 API 数量：27 个
```

---

# 9. 推荐执行顺序

```text
T00
T01
T02
T03
T11
T12
T04
T05
T06
T07
T08
T09
T10
T13
T14
```

原因：

```text
1. 先完成基础设施和文档入口
2. 先做 summary/driving/charging/calendar/report，立即支撑首页、图表、日报/月报
3. 再补停车、电池、成本、地点、生命周期
4. 最后做 insights，因为 insights 应复用 analytics 结果
```

---

# 10. 一次性交给 Codex 的总 Prompt

```text
你现在在 teslamateapi 项目中工作。请基于现有项目实现 TeslaMateApi V2 Analytics API。

背景：
当前项目已有 V1 API，V1 负责车辆基础信息、当前状态、行程、充电、更新、命令等能力。V2 只补充 V1 不支持的统计、分析、趋势、分布、排行、生命周期、日历、报表和客观洞察能力。

核心原则：
1. 不破坏任何 V1 API。
2. V2 统一使用 /api/v2 前缀。
3. V2 不实现车辆命令控制。
4. V2 不直接暴露数据库表。
5. V2 不重复实现 V1 明细查询。
6. 所有统计必须基于 TeslaMate PostgreSQL 已有事实数据。
7. 禁止输出主观建议或主观评价。
8. 估算指标必须使用 estimated 前缀，或者在 data_quality.warnings 中说明。
9. 所有数值字段必须带明确单位后缀，例如 distance_km、duration_min、energy_added_kwh、cost_per_100km。
10. 所有 API 必须有 Swagger 注释。
11. 所有 API 必须能出现在 github.com/watchakorn-18k/scalar-go 渲染的 Scalar API Reference 页面。
12. 每完成一个任务，必须更新 docs/v2-progress.md。
13. 每个 API 必须有 handler/service/repository/DTO/test。
14. 每个任务完成后必须执行 gofmt、go test ./...、go build ./...、swag init。
15. 每个任务完成后必须按实际运行环境启动服务，并用 curl 验证该任务涉及的 API 可正常返回。
16. 每个任务全部检查和 curl 验证通过后，必须立即为该任务创建独立 Git commit，然后才能继续下一个任务。
17. 不允许多个任务混在同一个 commit。

项目运行命令：
TZ=Asia/Shanghai \
ENCRYPTION_KEY=teslamate \
DATABASE_USER=teslamate \
DATABASE_PASS=pass \
DATABASE_NAME=teslamate \
DATABASE_HOST=192.168.2.7 \
DATABASE_PORT=5433 \
MQTT_HOST=saas.host.homelab \
go run ./src

时区要求：
1. 所有涉及时间范围、时间分桶、时间输出、同比/环比计算的 V2 API，都必须默认从环境变量 TZ 读取 IANA 时区。
2. 请求显式传入 timezone 时，使用请求 timezone。
3. timezone 非法时返回 400 INVALID_TIMEZONE。
4. period/start/end 必须先按请求时区计算本地时间边界。
5. 数据库过滤必须使用本地时间边界转换得到的 UTC 半开区间。
6. 输出时间必须转换回请求时区。
7. previous_period 和 previous_year 必须基于请求时区计算。

需要完成的任务：
T00：V2 基础框架、统一响应、Swagger + Scalar
T01：/api/v2/cars/{CarID}/analytics/summary
T02：Driving Analytics
  - /api/v2/cars/{CarID}/analytics/driving
  - /api/v2/cars/{CarID}/analytics/driving/timeseries
  - /api/v2/cars/{CarID}/analytics/driving/distribution
  - /api/v2/cars/{CarID}/analytics/driving/ranking
T03：Charging Analytics
  - /api/v2/cars/{CarID}/analytics/charging
  - /api/v2/cars/{CarID}/analytics/charging/timeseries
  - /api/v2/cars/{CarID}/analytics/charging/locations
  - /api/v2/cars/{CarID}/analytics/charging/types
  - /api/v2/cars/{CarID}/analytics/charging/cost
T04：Parking Analytics
  - /api/v2/cars/{CarID}/analytics/parking
  - /api/v2/cars/{CarID}/analytics/parking/locations
  - /api/v2/cars/{CarID}/analytics/parking/states
T05：Battery Analytics
  - /api/v2/cars/{CarID}/analytics/battery
  - /api/v2/cars/{CarID}/analytics/battery/timeseries
  - /api/v2/cars/{CarID}/analytics/battery/distribution
T06：Efficiency Analytics
  - /api/v2/cars/{CarID}/analytics/efficiency
  - /api/v2/cars/{CarID}/analytics/efficiency/factors
T07：Cost Analytics
  - /api/v2/cars/{CarID}/analytics/cost
T08：Location Analytics
  - /api/v2/cars/{CarID}/analytics/locations
T09：Update Analytics
  - /api/v2/cars/{CarID}/analytics/updates
T10：Lifecycle Analytics
  - /api/v2/cars/{CarID}/analytics/lifecycle
  - /api/v2/cars/{CarID}/timeline
T11：Calendar API
  - /api/v2/cars/{CarID}/calendar
T12：Reports API
  - /api/v2/cars/{CarID}/reports
T13：Insights API
  - /api/v2/cars/{CarID}/insights
T14：最终文档、测试、OpenAPI、Scalar 校验

通用查询参数：
period=day|week|month|quarter|year|custom|lifetime
start=RFC3339 datetime
end=RFC3339 datetime
timezone=IANA timezone
compare=none|previous_period|previous_year|lifetime_average
group_by=day|week|month|quarter|year|hour|weekday|location|charger_type
metrics=comma separated metric list
include=comma separated module list

通用响应结构：
{
  "data": {},
  "meta": {
    "car_id": 1,
    "period": "month",
    "timezone": "Asia/Shanghai",
    "start": "...",
    "end": "...",
    "compare": "previous_period",
    "unit": {
      "distance": "km",
      "energy": "kWh",
      "power": "kW",
      "temperature": "C",
      "currency": "CNY"
    },
    "generated_at": "...",
    "data_quality": {
      "complete": true,
      "sample_count": 0,
      "missing_fields": [],
      "warnings": []
    }
  }
}

统一错误结构：
{
  "error": {
    "code": "INVALID_PERIOD",
    "message": "invalid period",
    "details": {}
  }
}

文档要求：
1. 新增 /api/docs/swagger.json
2. 新增 /api/docs/scalar
3. 使用 github.com/watchakorn-18k/scalar-go 渲染 Swagger/OpenAPI 文档
4. 所有 V2 API 必须有 Swagger 注释
5. Tags 必须清晰：
   - V2 Summary
   - V2 Driving Analytics
   - V2 Charging Analytics
   - V2 Parking Analytics
   - V2 Battery Analytics
   - V2 Efficiency Analytics
   - V2 Cost Analytics
   - V2 Location Analytics
   - V2 Update Analytics
   - V2 Lifecycle
   - V2 Calendar
   - V2 Reports
   - V2 Insights

进度要求：
1. 新增 docs/v2-progress.md
2. 每完成一个任务更新状态
3. 每个任务记录：API、Swagger 是否完成、Scalar 是否可见、测试是否完成、gofmt/go test/go build/swag init 是否通过、curl 验证命令和结果、Git commit hash、备注。

测试要求：
1. 每个 API 至少覆盖：正常请求、无数据、非法 period、非法 car id。
2. 涉及时间的 API 必须覆盖：TZ 默认时区、请求 timezone 覆盖、非法 timezone、本地时间边界、previous_period 边界、输出时区。
3. 特定 API 增加维度、分桶、排行、样本不足、warning 等测试。
4. 最终 go test ./... 必须通过。

执行方式：
请按 T00 到 T14 的顺序逐个实现。每完成一个任务，先执行检查、启动服务、curl 验证并更新进度文档，再创建独立 Git commit，然后继续下一个任务。不要跳过文档、测试、curl 验证和 commit。不要破坏 V1。
```

---

# 11. 建议分批给 Codex 的短命令

不要一次性让 Codex 实现 27 个 API。建议逐步执行。

```text
先只实现 T00。完成后必须执行 gofmt、go test ./...、go build ./...、swag init，按指定环境变量启动服务，用 curl 验证 /api/v2、/api/docs/swagger.json、/api/docs/scalar，更新 docs/v2-progress.md，并创建独立 Git commit。
```

然后逐个继续：

```text
继续实现 T01。完成后执行检查、启动服务、curl 验证本任务 API、更新 docs/v2-progress.md，并创建独立 Git commit。
```

依次执行：

```text
T02
T03
T11
T12
T04
T05
T06
T07
T08
T09
T10
T13
T14
```

---

# 12. API 清单

## Summary

```http
GET /api/v2/cars/{CarID}/analytics/summary
```

## Driving Analytics

```http
GET /api/v2/cars/{CarID}/analytics/driving
GET /api/v2/cars/{CarID}/analytics/driving/timeseries
GET /api/v2/cars/{CarID}/analytics/driving/distribution
GET /api/v2/cars/{CarID}/analytics/driving/ranking
```

## Charging Analytics

```http
GET /api/v2/cars/{CarID}/analytics/charging
GET /api/v2/cars/{CarID}/analytics/charging/timeseries
GET /api/v2/cars/{CarID}/analytics/charging/locations
GET /api/v2/cars/{CarID}/analytics/charging/types
GET /api/v2/cars/{CarID}/analytics/charging/cost
```

## Parking Analytics

```http
GET /api/v2/cars/{CarID}/analytics/parking
GET /api/v2/cars/{CarID}/analytics/parking/locations
GET /api/v2/cars/{CarID}/analytics/parking/states
```

## Battery Analytics

```http
GET /api/v2/cars/{CarID}/analytics/battery
GET /api/v2/cars/{CarID}/analytics/battery/timeseries
GET /api/v2/cars/{CarID}/analytics/battery/distribution
```

## Efficiency Analytics

```http
GET /api/v2/cars/{CarID}/analytics/efficiency
GET /api/v2/cars/{CarID}/analytics/efficiency/factors
```

## Cost Analytics

```http
GET /api/v2/cars/{CarID}/analytics/cost
```

## Location Analytics

```http
GET /api/v2/cars/{CarID}/analytics/locations
```

## Update Analytics

```http
GET /api/v2/cars/{CarID}/analytics/updates
```

## Lifecycle

```http
GET /api/v2/cars/{CarID}/analytics/lifecycle
GET /api/v2/cars/{CarID}/timeline
```

## Calendar

```http
GET /api/v2/cars/{CarID}/calendar
```

## Reports

```http
GET /api/v2/cars/{CarID}/reports
```

## Insights

```http
GET /api/v2/cars/{CarID}/insights
```
