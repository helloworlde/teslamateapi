下面内容可以直接作为 TASKS.md / IMPLEMENTATION_PLAN.md / Codex Prompt 使用。

已确认：teslamateapi 是 Go 项目，定位为从 TeslaMate PostgreSQL 和 Mosquitto 读取数据并通过 REST API 暴露；scalar-go 用于基于 OpenAPI/Swagger 文件渲染 API Reference HTML；swaggo 的常规流程是通过 Go 注释生成 docs 目录和 OpenAPI/Swagger 文件。 ￼

⸻

TeslaMateApi V2 Analytics API 实现任务与 Codex Prompt

0. 总目标

基于当前 teslamateapi 项目新增 /api/v2 API，只实现 V1 不支持的能力：

统计
分析
趋势
分布
排行
生命周期汇总
日历聚合
报表聚合
客观洞察

V2 不实现：

车辆命令控制
日志开关
V1 已有的车辆、行程、充电、更新明细基础查询
直接暴露数据库表
主观建议
未经事实数据支撑的结论

所有 V2 API 必须满足：

1. 有 Swagger 注释
2. 可被 swag 生成 OpenAPI/Swagger 文档
3. 可通过 github.com/watchakorn-18k/scalar-go 渲染 API Reference
4. 有统一响应结构
5. 有统一错误结构
6. 有单元测试或集成测试
7. 每完成一个任务，更新 docs/v2-progress.md
8. 每完成一个任务后必须先执行检查（至少 gofmt、go test ./...、go build、swag init），并按本项目实际运行环境启动服务、用 curl 验证该任务涉及的 API 可正常返回；全部通过后立即为该任务单独创建 Git 提交，再继续下一个任务。
9. 所有涉及时间范围、时间分桶、时间输出、同比/环比计算的 V2 API 都必须从环境变量 TZ 读取默认 IANA 时区；请求显式传入 timezone 时使用请求时区，并正确处理本地时间边界与数据库 UTC 时间过滤。

⸻

1. 实现约束

1.1 兼容性约束

1. 不破坏任何现有 V1 API
2. 不修改 V1 响应结构
3. 不删除现有路由
4. V2 使用 /api/v2 前缀
5. V2 可以新增代码目录、模型、服务、仓储、测试

⸻

1.2 数据原则

V2 所有统计必须是客观事实数据。

允许输出：

本月行驶 1420.5 km，较上月增加 9.17%
本月充电 18 次，充入 420.5 kWh
本月平均能耗 171.0 Wh/km
Office 地点停车耗电 2.1%/day

禁止输出：

你的驾驶习惯很好
建议减少快充
电池很健康
这是不好的用车方式

如果指标是估算值，必须明确字段命名或返回 data_quality.warnings。

例如：

estimated_energy_consumed_kwh
estimated_range_degradation_percent
estimated_vampire_drain_kwh

⸻

1.3 文档约束

每个 API 必须包含 Swagger 注释：

// @Summary ...
// @Description ...
// @Tags V2 Analytics
// @Produce json
// @Param CarID path int true "Car ID"
// @Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom, lifetime)
// @Param start query string false "Start datetime in RFC3339 format"
// @Param end query string false "End datetime in RFC3339 format"
// @Param timezone query string false "IANA timezone"
// @Param compare query string false "Comparison mode" Enums(none, previous_period, previous_year, lifetime_average)
// @Success 200 {object} model.APIResponse[model.V2SummaryResponse]
// @Failure 400 {object} model.APIErrorResponse
// @Failure 404 {object} model.APIErrorResponse
// @Failure 500 {object} model.APIErrorResponse
// @Router /v2/cars/{CarID}/analytics/summary [get]

如果当前 Swagger 泛型支持不好，使用非泛型 wrapper DTO。

⸻

1.4 Scalar 文档约束

必须新增或确认以下文档入口：

/api/docs
/api/docs/swagger.json
/api/docs/scalar

要求：

1. /api/docs/swagger.json 返回当前 OpenAPI/Swagger JSON
2. /api/docs/scalar 使用 github.com/watchakorn-18k/scalar-go 渲染文档
3. Scalar 页面能展示 V1 + V2 所有 API
4. V2 API tag 清晰分组

推荐 Tags：

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

⸻

2. 推荐代码结构

根据项目现有结构调整，不要强行重构全项目。优先新增 V2 独立结构。

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

如果项目已有对应目录，遵循现有风格，不重复造架构。

⸻

3. 通用模型

3.1 通用查询参数

所有 V2 分析接口统一支持：

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

⸻

3.2 通用响应结构

type V2Meta struct {
CarID       int64             `json:"car_id"`
Period      string            `json:"period"`
Timezone    string            `json:"timezone"`
Start       string            `json:"start,omitempty"`
End         string            `json:"end,omitempty"`
Compare     string            `json:"compare,omitempty"`
Unit        V2Unit            `json:"unit"`
GeneratedAt string            `json:"generated_at"`
DataQuality *V2DataQuality    `json:"data_quality,omitempty"`
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

⸻

4. 进度记录要求

新增文件：

docs/v2-progress.md

内容模板：

# V2 API Implementation Progress
## Rules
- Each completed task must update this file.
- Each API must have handler, service, repository, response DTO, Swagger comments, and tests.
- Each API must be visible in Scalar documentation.
## Progress
| Task | Status | PR/Commit | APIs | Swagger | Scalar | Tests | Notes |
|---|---|---|---|---|---|---|---|
| T00 | Done |  | V2 base framework | Yes | Yes | Yes |  |
| T01 | Pending |  | summary | No | No | No |  |

每完成一个任务，把 Pending 改成 Done 或 Partial。

⸻

5. 任务拆分

T00：V2 基础框架、统一响应、Swagger + Scalar

目标

建立 V2 基础能力，不实现复杂业务统计。

需要完成

1. 新增 /api/v2 health/info 路由
2. 新增 V2 router group
3. 新增通用 query parser
4. 新增统一 response/error helper
5. 新增 period/compare 校验
6. 接入 Swagger 文档生成
7. 接入 github.com/watchakorn-18k/scalar-go
8. 新增 /api/docs/swagger.json
9. 新增 /api/docs/scalar
10. 初始化 docs/v2-progress.md

API

GET /api/v2

响应示例

{
"data": {
"version": "v2",
"scope": "analytics",
"features": [
"summary",
"driving",
"charging",
"parking",
"battery",
"efficiency",
"cost",
"locations",
"updates",
"lifecycle",
"calendar",
"reports",
"insights"
]
},
"meta": {
"generated_at": "2026-05-07T20:00:00+08:00"
}
}

验收标准

go test ./...
go build ./...
swag init 可以成功
/api/docs/swagger.json 可访问
/api/docs/scalar 可访问
/api/v2 可访问
docs/v2-progress.md 已创建

Codex Prompt

你现在在 teslamateapi 项目中工作。请实现 T00：V2 基础框架、统一响应、Swagger + Scalar 文档入口。
要求：
1. 不破坏任何现有 V1 API。
2. 新增 /api/v2 路由，返回 V2 能力说明。
3. 新增 V2 通用响应结构、错误结构、时间周期参数结构。
4. 新增 period 校验：day/week/month/quarter/year/custom/lifetime。
5. 新增 compare 校验：none/previous_period/previous_year/lifetime_average。
6. 接入 Swagger 文档生成，保留现有文档能力。
7. 使用 github.com/watchakorn-18k/scalar-go 新增 Scalar API Reference 页面。
8. 新增 /api/docs/swagger.json 和 /api/docs/scalar，确保 Scalar 使用 swagger.json。
9. 所有新增 API 必须有 Swagger 注释。
10. 新增 docs/v2-progress.md，并记录 T00 状态。
11. 添加必要测试。
12. 运行 gofmt、go test ./...，修复所有失败。
    不要实现具体 analytics 业务。只做基础框架和文档入口。

⸻

T01：周期总览 Summary API

API

GET /api/v2/cars/{CarID}/analytics/summary

功能

返回指定周期的客观总览统计。

包含模块

driving
charging
parking
battery
updates
cost

响应 DTO

type V2SummaryResponse struct {
Summary V2Summary `json:"summary"`
Comparison map[string]V2ComparisonValue `json:"comparison,omitempty"`
}
type V2Summary struct {
Driving  V2DrivingSummary  `json:"driving"`
Charging V2ChargingSummary `json:"charging"`
Parking  V2ParkingSummary  `json:"parking"`
Battery  V2BatterySummary  `json:"battery"`
Updates  V2UpdateSummary   `json:"updates"`
Cost     V2CostSummary     `json:"cost"`
}

指标

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

验收标准

1. 支持 period/start/end/timezone/compare 参数
2. compare=previous_period 可返回 comparison
3. 无数据时返回 0 或 null，不能 500
4. Swagger 可见
5. Scalar 可见
6. docs/v2-progress.md 更新 T01

Codex Prompt

请实现 T01：V2 周期总览 Summary API。
API：
GET /api/v2/cars/{CarID}/analytics/summary
目标：
返回指定周期内车辆使用的客观统计汇总，只基于 TeslaMate PostgreSQL 中已有事实数据，不输出主观建议。
要求：
1. 使用 T00 中的 V2 通用响应、错误结构和查询参数解析。
2. 支持 period/start/end/timezone/compare。
3. period 支持 day/week/month/quarter/year/custom/lifetime。
4. compare 支持 none/previous_period/previous_year/lifetime_average，至少实现 none 和 previous_period，其余可返回明确的 not implemented 错误或预留。
5. 返回 driving、charging、parking、battery、updates、cost 六个模块。
6. 所有数值字段使用明确单位后缀，例如 distance_km、duration_min、energy_added_kwh。
7. 估算字段必须使用 estimated 前缀，或者在 data_quality.warnings 中说明。
8. 无数据时返回空统计，不要报错。
9. 添加 Swagger 注释，Tag 使用 V2 Summary。
10. 确保 Scalar 页面能显示该 API。
11. 添加 handler/service/repository 分层，不要把 SQL 写在 handler。
12. 添加测试，覆盖正常查询、无数据、非法 period、非法 car id。
13. 更新 docs/v2-progress.md 中 T01 的状态。
14. 运行 gofmt、go test ./...，修复所有失败。

⸻

T02：行驶分析 Driving Analytics

APIs

GET /api/v2/cars/{CarID}/analytics/driving
GET /api/v2/cars/{CarID}/analytics/driving/timeseries
GET /api/v2/cars/{CarID}/analytics/driving/distribution
GET /api/v2/cars/{CarID}/analytics/driving/ranking

功能

行驶统计
周期趋势
分布分析
极值排行

核心指标

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

distribution 支持

hour_of_day
day_of_week
distance_bucket
duration_bucket
speed_bucket
consumption_bucket
temperature_bucket

ranking 支持

longest_distance
longest_duration
highest_speed
lowest_consumption
highest_consumption
highest_distance_day

Codex Prompt

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
5. /driving 返回周期汇总和 comparison。
6. /timeseries 按 group_by 或 period 返回 day/week/month/year 趋势。
7. /distribution 支持 dimension 参数：
    - hour_of_day
    - day_of_week
    - distance_bucket
    - duration_bucket
    - speed_bucket
    - consumption_bucket
    - temperature_bucket
8. /ranking 支持 type 参数：
    - longest_distance
    - longest_duration
    - highest_speed
    - lowest_consumption
    - highest_consumption
    - highest_distance_day
9. 返回结构必须适合图表直接使用。
10. 添加完整 Swagger 注释，Tags：
    - V2 Driving Analytics
11. 确保所有 API 出现在 Scalar 文档中。
12. 添加测试：
    - 正常统计
    - 无行程数据
    - 非法 dimension
    - 非法 ranking type
    - 非法 period
13. 更新 docs/v2-progress.md 中 T02 的状态。
14. 运行 gofmt、go test ./...，修复所有失败。

⸻

T03：充电分析 Charging Analytics

APIs

GET /api/v2/cars/{CarID}/analytics/charging
GET /api/v2/cars/{CarID}/analytics/charging/timeseries
GET /api/v2/cars/{CarID}/analytics/charging/locations
GET /api/v2/cars/{CarID}/analytics/charging/types
GET /api/v2/cars/{CarID}/analytics/charging/cost

指标

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

Codex Prompt

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
3. /charging 返回周期汇总和 comparison。
4. /timeseries 返回 day/week/month/year 聚合趋势。
5. /locations 按 geofence/address 聚合充电次数、电量、费用、效率。
6. /types 按 ac/dc/supercharger/unknown 聚合。
7. /cost 返回充电费用、单价、每公里成本、每百公里成本。
8. 如果 cost 数据缺失，返回 null 或 0，并在 data_quality.missing_fields 或 warnings 中说明。
9. 充电效率使用 energy_added_kwh / energy_used_kwh，energy_used_kwh 缺失时不可伪造。
10. 所有字段带单位后缀。
11. 添加完整 Swagger 注释，Tag 使用 V2 Charging Analytics。
12. 确保 Scalar 页面可见。
13. 添加测试：
    - 正常统计
    - 无充电数据
    - 缺失 cost
    - 缺失 energy_used
    - 非法 period
14. 更新 docs/v2-progress.md 中 T03 的状态。
15. 运行 gofmt、go test ./...，修复所有失败。

⸻

T04：停放与状态分析 Parking Analytics

APIs

GET /api/v2/cars/{CarID}/analytics/parking
GET /api/v2/cars/{CarID}/analytics/parking/locations
GET /api/v2/cars/{CarID}/analytics/parking/states

指标

parking_session_count
parked_duration_min
avg_parked_duration_min
asleep_duration_min
online_duration_min
offline_duration_min
vampire_drain_percent
vampire_drain_range_km
estimated_vampire_drain_kwh
avg_drain_percent_per_day
state_transition_count

Codex Prompt

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
4. 如果 vampire drain 依赖估算，字段必须命名为 estimated_vampire_drain_kwh，且 data_quality.warnings 说明。
5. 不输出建议。
6. /parking 返回周期汇总。
7. /locations 按 geofence/address 聚合停放次数、停放时长、状态时长、停车耗电。
8. /states 返回 online/asleep/offline 时长、占比和状态切换次数。
9. 无法精确推导停放 session 时，返回已有 states 统计并在 data_quality.warnings 说明。
10. 添加完整 Swagger 注释，Tag 使用 V2 Parking Analytics。
11. 确保 Scalar 页面可见。
12. 添加测试：
    - 正常 states 统计
    - 无 states 数据
    - 停放 session 推导
    - 时间窗口边界
    - 数据缺失 warning
13. 更新 docs/v2-progress.md 中 T04 的状态。
14. 运行 gofmt、go test ./...，修复所有失败。

⸻

T05：电池分析 Battery Analytics

APIs

GET /api/v2/cars/{CarID}/analytics/battery
GET /api/v2/cars/{CarID}/analytics/battery/timeseries
GET /api/v2/cars/{CarID}/analytics/battery/distribution

指标

latest_battery_level_percent
latest_rated_range_km
latest_ideal_range_km
estimated_rated_range_at_100_percent_km
estimated_ideal_range_at_100_percent_km
baseline_rated_range_at_100_percent_km
estimated_range_degradation_percent
sample_count

Codex Prompt

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
4. /battery 返回最近电量、最近续航、估算满电续航、基准续航、估算下降比例。
5. /timeseries 按周期返回估算满电 rated/ideal range 趋势。
6. /distribution 返回电量区间分布，例如 0-10、10-20、...、90-100。
7. 样本不足时，不要输出误导性 degradation，返回 null 并在 warnings 中说明。
8. 添加完整 Swagger 注释，Tag 使用 V2 Battery Analytics。
9. 确保 Scalar 页面可见。
10. 添加测试：
    - 正常样本
    - 样本不足
    - 电量为 0 或 null
    - distribution 分桶
    - warning 存在
11. 更新 docs/v2-progress.md 中 T05 的状态。
12. 运行 gofmt、go test ./...，修复所有失败。

⸻

T06：能耗效率 Efficiency Analytics

APIs

GET /api/v2/cars/{CarID}/analytics/efficiency
GET /api/v2/cars/{CarID}/analytics/efficiency/factors

指标

distance_km
estimated_energy_consumed_kwh
avg_consumption_wh_per_km
best_consumption_wh_per_km
worst_consumption_wh_per_km
avg_temperature_c
avg_speed_kmh
estimated_regenerated_energy_kwh

factors 支持维度

temperature
speed
distance
elevation
location
hour_of_day
day_of_week

Codex Prompt

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
4. /efficiency 返回周期总能耗、平均能耗、最佳/最差能耗、平均温度、平均速度。
5. /factors 支持 dimension：
    - temperature
    - speed
    - distance
    - elevation
    - location
    - hour_of_day
    - day_of_week
6. 估算能耗字段使用 estimated 前缀或 warnings。
7. 添加 Swagger 注释，Tag 使用 V2 Efficiency Analytics。
8. 确保 Scalar 页面可见。
9. 添加测试：
    - 正常统计
    - 无行驶数据
    - 非法 dimension
    - 分桶正确
10. 更新 docs/v2-progress.md 中 T06 的状态。
11. 运行 gofmt、go test ./...，修复所有失败。

⸻

T07：成本分析 Cost Analytics

API

GET /api/v2/cars/{CarID}/analytics/cost

指标

charging_cost
energy_used_kwh
distance_km
cost_per_kwh
cost_per_km
cost_per_100km
cost_by_location
cost_by_period

Codex Prompt

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
7. cost 缺失时返回 data_quality.warnings。
8. 不输出“省钱”“昂贵”等主观评价。
9. 添加 Swagger 注释，Tag 使用 V2 Cost Analytics。
10. 确保 Scalar 页面可见。
11. 添加测试：
    - 正常 cost
    - cost 缺失
    - distance 为 0
    - energy_used 为 0
12. 更新 docs/v2-progress.md 中 T07 的状态。
13. 运行 gofmt、go test ./...，修复所有失败。

⸻

T08：地点分析 Location Analytics

API

GET /api/v2/cars/{CarID}/analytics/locations

指标

location_name
drive_start_count
drive_end_count
charging_session_count
parking_session_count
parking_duration_min
energy_added_kwh
charging_cost
vampire_drain_percent

Codex Prompt

请实现 T08：V2 地点分析 Location Analytics。
API：
GET /api/v2/cars/{CarID}/analytics/locations
目标：
按 geofence/address 聚合车辆使用行为，包括出发、到达、充电、停放和耗电。
要求：
1. 基于 drives、charging_processes、states、geofences、addresses 的事实数据。
2. 返回每个地点：
    - location_name
    - geofence_id
    - address_id
    - drive_start_count
    - drive_end_count
    - charging_session_count
    - parking_session_count
    - parking_duration_min
    - energy_added_kwh
    - charging_cost
    - vampire_drain_percent
3. 如果无法归属 geofence，则归入 unknown 或 address。
4. 支持 sort 参数：
    - drive_start_count_desc
    - drive_end_count_desc
    - charging_session_count_desc
    - parking_duration_desc
    - charging_cost_desc
5. 不输出主观结论。
6. 添加 Swagger 注释，Tag 使用 V2 Location Analytics。
7. 确保 Scalar 页面可见。
8. 添加测试：
    - geofence 聚合
    - unknown 地点
    - sort 参数
    - 无数据
9. 更新 docs/v2-progress.md 中 T08 的状态。
10. 运行 gofmt、go test ./...，修复所有失败。

⸻

T09：更新分析 Update Analytics

API

GET /api/v2/cars/{CarID}/analytics/updates

指标

update_count
latest_version
latest_update_at
avg_update_duration_min
versions

Codex Prompt

请实现 T09：V2 更新分析 Update Analytics。
API：
GET /api/v2/cars/{CarID}/analytics/updates
目标：
统计 TeslaMate 已记录的 OTA 更新历史。
要求：
1. 基于 updates 表事实数据。
2. 返回 update_count、latest_version、latest_update_at、avg_update_duration_min、versions。
3. versions 按更新时间倒序。
4. 无更新记录时返回 update_count=0，latest_version=null。
5. 不输出版本好坏评价。
6. 添加 Swagger 注释，Tag 使用 V2 Update Analytics。
7. 确保 Scalar 页面可见。
8. 添加测试：
    - 正常更新记录
    - 无更新记录
    - end_date 缺失
9. 更新 docs/v2-progress.md 中 T09 的状态。
10. 运行 gofmt、go test ./...，修复所有失败。

⸻

T10：生命周期分析 Lifecycle Analytics

APIs

GET /api/v2/cars/{CarID}/analytics/lifecycle
GET /api/v2/cars/{CarID}/timeline

lifecycle 指标

first_recorded_at
last_recorded_at
recorded_days
odometer_start_km
odometer_latest_km
odometer_delta_km
drive_count
distance_km
charging_session_count
energy_added_kwh
energy_used_kwh
charging_cost
update_count
avg_daily_distance_km
avg_monthly_distance_km
avg_consumption_wh_per_km
cost_per_100km

timeline 事件类型

drive
charging
parking
update
state

Codex Prompt

请实现 T10：V2 生命周期分析 Lifecycle Analytics。
APIs：
1. GET /api/v2/cars/{CarID}/analytics/lifecycle
2. GET /api/v2/cars/{CarID}/timeline
   目标：
   提供车辆从 TeslaMate 记录开始以来的累计统计，以及统一生命周期事件流。
   要求：
1. /analytics/lifecycle 返回累计统计：
    - first_recorded_at
    - last_recorded_at
    - recorded_days
    - odometer_start_km
    - odometer_latest_km
    - odometer_delta_km
    - drive_count
    - distance_km
    - charging_session_count
    - energy_added_kwh
    - energy_used_kwh
    - charging_cost
    - update_count
    - avg_daily_distance_km
    - avg_monthly_distance_km
    - avg_consumption_wh_per_km
    - cost_per_100km
2. /timeline 统一返回 drive、charging、parking、update、state 事件。
3. timeline 支持 start/end/type/limit/order 参数。
4. timeline 事件必须包含 type、id、start_time、end_time、title、metrics。
5. 不返回 V1 详情，只返回适合时间线展示的摘要。
6. 不输出主观评价。
7. 添加 Swagger 注释，Tags：
    - V2 Lifecycle
8. 确保 Scalar 页面可见。
9. 添加测试：
    - lifecycle 累计统计
    - timeline 多类型事件排序
    - type 过滤
    - 无数据
10. 更新 docs/v2-progress.md 中 T10 的状态。
11. 运行 gofmt、go test ./...，修复所有失败。

⸻

T11：日历聚合 Calendar API

API

GET /api/v2/cars/{CarID}/calendar

指标

date
drive_count
distance_km
drive_duration_min
charging_session_count
energy_added_kwh
charging_cost
parking_duration_min
vampire_drain_percent
update_count
activity_level.driving
activity_level.charging
activity_level.parking_drain

Codex Prompt

请实现 T11：V2 日历聚合 Calendar API。
API：
GET /api/v2/cars/{CarID}/calendar
目标：
返回按天聚合的数据，供日历视图、热力图、日报入口使用。
要求：
1. 返回指定 start/end 范围内每天的数据。
2. 每天包含：
    - date
    - drive_count
    - distance_km
    - drive_duration_min
    - charging_session_count
    - energy_added_kwh
    - charging_cost
    - parking_duration_min
    - vampire_drain_percent
    - update_count
    - activity_level
3. activity_level 是客观分级，用于 UI 显示，不是评价。
4. activity_level 可按固定阈值计算：
    - driving: 0/1/2/3/4
    - charging: 0/1/2/3/4
    - parking_drain: 0/1/2/3/4
5. 如果某天无数据，也应返回该日期，数值为 0。
6. 支持 timezone。
7. 添加 Swagger 注释，Tag 使用 V2 Calendar。
8. 确保 Scalar 页面可见。
9. 添加测试：
    - 日期连续性
    - 无数据日期返回 0
    - 跨月范围
    - activity_level 分级
10. 更新 docs/v2-progress.md 中 T11 的状态。
11. 运行 gofmt、go test ./...，修复所有失败。

⸻

T12：报表 Reports API

API

GET /api/v2/cars/{CarID}/reports

报表模块

summary
driving
charging
parking
battery
efficiency
cost
locations
updates
insights

Codex Prompt

请实现 T12：V2 报表 Reports API。
API：
GET /api/v2/cars/{CarID}/reports
目标：
返回适合前端直接展示的周期报表聚合数据。报表只组织数据，不输出主观文案。
要求：
1. 支持 period=day/week/month/quarter/year/lifetime/custom。
2. 返回 title、period、sections。
3. sections 只包含有数据的模块。
4. 支持 include 参数控制模块：
    - summary
    - driving
    - charging
    - parking
    - battery
    - efficiency
    - cost
    - locations
    - updates
    - insights
5. 报表复用已实现的 analytics service，不重复写 SQL。
6. 每个 section 包含 type、title、metrics、data_quality。
7. 没有充电数据时，不返回 charging section。
8. 没有更新数据时，不返回 updates section。
9. 添加 Swagger 注释，Tag 使用 V2 Reports。
10. 确保 Scalar 页面可见。
11. 添加测试：
    - 月报
    - 年报
    - include 过滤
    - 空模块隐藏
12. 更新 docs/v2-progress.md 中 T12 的状态。
13. 运行 gofmt、go test ./...，修复所有失败。

⸻

T13：客观洞察 Insights API

API

GET /api/v2/cars/{CarID}/insights

洞察原则

每条洞察必须包含：

type
severity
title
description
metrics
evidence
time_range
data_quality

支持类型

driving_distance_increased
driving_distance_decreased
driving_frequency_changed
consumption_increased
consumption_decreased
short_trip_ratio_high
long_drive_record
high_speed_record
charging_energy_increased
charging_cost_increased
charging_cost_per_kwh_increased
dc_charging_ratio_increased
home_charging_ratio_changed
charging_efficiency_decreased
parking_duration_increased
online_duration_high
vampire_drain_increased
location_drain_high
estimated_range_decreased
estimated_range_increased
low_battery_events
high_soc_duration_high
monthly_distance_record
monthly_cost_record
efficiency_record
charging_location_changed

Codex Prompt

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
6. 不允许出现：
    - 建议你
    - 最好
    - 应该
    - 健康
    - 不健康
    - 好
    - 差
    - 合理
    - 不合理
7. 允许出现：
    - 增加
    - 减少
    - 高于
    - 低于
    - 占比
    - 新高
    - 新低
    - 样本不足
      实现规则：
1. 支持 period/start/end/timezone/compare。
2. 支持 category 参数：
    - driving
    - charging
    - parking
    - battery
    - cost
    - lifecycle
3. 支持 min_severity 参数：
    - info
    - warning
4. 先实现以下洞察：
    - driving_distance_increased
    - driving_distance_decreased
    - consumption_increased
    - consumption_decreased
    - charging_cost_increased
    - dc_charging_ratio_increased
    - vampire_drain_increased
    - online_duration_high
    - estimated_range_decreased
    - monthly_distance_record
5. 其他类型保留结构，但不要伪实现。
6. 添加最小样本量规则：
    - 能耗变化：至少 5 次行程且总距离 > 50 km
    - 充电变化：至少 3 次充电
    - 停车耗电：至少 3 次停放且总停放时长 > 24h
    - 电池趋势：至少 10 个有效样本
7. 添加 Swagger 注释，Tag 使用 V2 Insights。
8. 确保 Scalar 页面可见。
9. 添加测试：
    - 正常生成洞察
    - 样本不足不生成
    - 禁止主观词
    - evidence 存在
    - category 过滤
10. 更新 docs/v2-progress.md 中 T13 的状态。
11. 运行 gofmt、go test ./...，修复所有失败。

⸻

T14：最终文档、质量检查、OpenAPI 校验

目标

完成所有 API 后做统一收尾。

需要完成

1. 整理 docs/v2-api.md
2. 整理 docs/v2-progress.md
3. 确认所有 V2 API 都有 Swagger 注释
4. 确认 Scalar 页面展示完整
5. 确认所有 tags 分组合理
6. 确认 go test ./... 通过
7. 确认 go vet ./... 通过
8. 确认 README 增加 V2 API 说明
9. 确认 docker build 通过
10. 确认没有破坏 V1

Codex Prompt

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
7. 运行：
    - gofmt
    - go test ./...
    - go vet ./...
    - swag init
    - docker build
8. 修复所有失败。
9. 检查 V1 路由仍然存在且测试通过。
10. 不新增未完成的 API 假实现。
11. 如果某个 API 未完成，必须在 progress 文档中明确标记 Partial 和原因。

⸻

6. 总任务表

任务	API 数	内容	优先级
T00	1	V2 基础框架、Swagger、Scalar	P0
T01	1	Summary 总览	P0
T02	4	Driving 行驶分析	P0
T03	5	Charging 充电分析	P0
T11	1	Calendar 日历聚合	P0
T12	1	Reports 报表	P0
T04	3	Parking 停放分析	P1
T05	3	Battery 电池分析	P1
T06	2	Efficiency 能耗效率	P1
T07	1	Cost 成本分析	P1
T08	1	Locations 地点分析	P1
T09	1	Updates 更新分析	P1
T10	2	Lifecycle 生命周期	P1
T13	1	Insights 客观洞察	P2
T14	-	文档、测试、质量检查	P0

合计：

V2 API 数量：27 个

⸻

7. 推荐执行顺序

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

原因：

1. 先完成基础设施和文档入口
2. 先做 summary/driving/charging/calendar/report，立即支撑首页、图表、日报/月报
3. 再补停车、电池、成本、地点、生命周期
4. 最后做 insights，因为 insights 应复用 analytics 结果

⸻

8. 一次性交给 Codex 的总 Prompt

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
14. 每个阶段都运行 gofmt、go test ./...，修复所有失败。
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
3. 每个任务记录：
    - API
    - Swagger 是否完成
    - Scalar 是否可见
    - 测试是否完成
    - 备注
      测试要求：
1. 每个 API 至少覆盖：
    - 正常请求
    - 无数据
    - 非法 period
    - 非法 car id
2. 特定 API 增加维度、分桶、排行、样本不足、warning 等测试。
3. 最终 go test ./... 必须通过。
   执行方式：
   请按 T00 到 T14 的顺序逐个实现。每完成一个任务，先运行测试并更新进度文档，再进入下一个任务。不要跳过文档和测试。不要破坏 V1。

⸻
