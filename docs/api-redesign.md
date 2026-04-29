# API 重构审计与设计说明

## 范围

本文档记录扩展 API 的重构背景、保留/删除的路由、新的接口组织方式，以及当前实现中的 SQL、时区、缓存和响应结构约定。

## 路由清单

### 原兼容 API

以下路由属于兼容面，继续保留原路径和响应结构：

- `GET /api`
- `GET /api/v1`
- `GET /api/v1/cars`
- `GET /api/v1/cars/:CarID`
- `GET /api/v1/cars/:CarID/battery-health`
- `GET /api/v1/cars/:CarID/charges`
- `GET /api/v1/cars/:CarID/charges/current`
- `GET /api/v1/cars/:CarID/charges/:ChargeID`
- `GET /api/v1/cars/:CarID/drives`
- `GET /api/v1/cars/:CarID/drives/:DriveID`
- `GET /api/v1/cars/:CarID/status`
- `GET /api/v1/cars/:CarID/updates`
- `GET /api/v1/globalsettings`
- `GET /api/healthz`
- `GET /api/ping`
- `GET /api/readyz`

以下命令路由只有在通过 `ENABLE_COMMANDS=true` 显式启用时才注册；未启用时不会挂载到路由表：

- `GET /api/v1/cars/:CarID/command`
- `POST /api/v1/cars/:CarID/command/:Command`
- `GET /api/v1/cars/:CarID/logging`
- `PUT /api/v1/cars/:CarID/logging/:Command`
- `POST /api/v1/cars/:CarID/wake_up`

### 文档路由

- `GET /api/v1/docs`
- `GET /api/v1/docs/openapi.json`
- `GET /api/v1/docs/swagger`
- `GET /api/v1/docs/swagger/index.html`
- `GET /api/v1/docs/swagger/doc.json`

### 已删除的旧扩展路由

以下路由在重构前存在，但语义重复、组织分散或已被新接口替代，当前不再注册：

- `GET /api/v1/summaries/options`
- `GET /api/v1/cars/:CarID/parking-sessions`
- `GET /api/v1/cars/:CarID/summaries`
- `GET /api/v1/cars/:CarID/summaries/overview`
- `GET /api/v1/cars/:CarID/summaries/lifetime`
- `GET /api/v1/cars/:CarID/summaries/drives`
- `GET /api/v1/cars/:CarID/summaries/charges`
- `GET /api/v1/cars/:CarID/summaries/parking`
- `GET /api/v1/cars/:CarID/summaries/statistics`
- `GET /api/v1/cars/:CarID/summaries/state-activity`
- `GET /api/v1/cars/:CarID/activity-timeline`
- `GET /api/v1/cars/:CarID/dashboards/drives`
- `GET /api/v1/cars/:CarID/dashboards/charges`
- `GET /api/v1/cars/:CarID/calendars/drives`
- `GET /api/v1/cars/:CarID/charts/efficiency`
- `GET /api/v1/cars/:CarID/charts/drives/monthly-distance`
- `GET /api/v1/cars/:CarID/charts/drives/weekday-distance`
- `GET /api/v1/cars/:CarID/charts/drives/hourly-starts`
- `GET /api/v1/cars/:CarID/charts/charges/monthly-energy`
- `GET /api/v1/cars/:CarID/charts/charges/location-energy`
- `GET /api/v1/cars/:CarID/charts/charges/weekday-energy`
- `GET /api/v1/cars/:CarID/charts/charges/hourly-starts`
- `GET /api/v1/cars/:CarID/charts/activity/duration`
- `GET /api/v1/cars/:CarID/charges/:ChargeID/interval`

### 重构后的扩展路由

- `GET /api/v1/cars/:CarID/summary`
- `GET /api/v1/cars/:CarID/dashboard`
- `GET /api/v1/cars/:CarID/realtime`
- `GET /api/v1/cars/:CarID/calendar`
- `GET /api/v1/cars/:CarID/statistics`
- `GET /api/v1/cars/:CarID/series/drives`
- `GET /api/v1/cars/:CarID/series/charges`
- `GET /api/v1/cars/:CarID/series/battery`
- `GET /api/v1/cars/:CarID/series/states`
- `GET /api/v1/cars/:CarID/distributions/drives`
- `GET /api/v1/cars/:CarID/distributions/charges`
- `GET /api/v1/cars/:CarID/insights`
- `GET /api/v1/cars/:CarID/timeline`
- `GET /api/v1/cars/:CarID/map/visited`
- `GET /api/v1/cars/:CarID/locations`

### OpenAPI 响应模型

重构后的扩展路由使用具备业务含义的 Swagger 模型，避免用泛化 object 描述真实结构：

- `summary` -> `SummaryV2Envelope`
- `dashboard` -> `DashboardV2Envelope`
- `realtime` -> `RealtimeV2Envelope`
- `calendar` -> `CalendarV2Envelope`
- `statistics` -> `StatisticsV2Envelope`
- `series/*` -> `SeriesV2Envelope`
- `distributions/*` -> `DistributionsV2Envelope`
- `insights` -> `InsightsV2Envelope`
- `timeline` -> `TimelineV2Envelope`
- `map/visited` -> `VisitedMapV2Envelope`

## 审计结论

### REST 与命名

- `summaries/*` 与 `dashboards/*` 语义高度重叠，客户端难以判断数据边界。
- `activity-timeline` 与 `timeline` 表达同一资源，已合并为统一时间线。
- 日历数据统一为 `calendar` 资源，通过查询参数控制 bucket 和可选指标，不再拆成 drives/charges 多个别名。
- 图表接口改为稳定的 `series` / `distributions` 资源和明确 metric 名称，不再使用 `monthly-distance` 这类实现细节命名。

### 第三方客户端适配性

- 旧 summary/dashboard 路由要求客户端理解多个别名背后的字段子集，维护成本高。
- 新设计拆分 summary、statistics、series、distributions、timeline 等职责，图表数据更适合直接渲染。
- 新列表类响应统一暴露 `limit`、`offset`、`total` 分页字段。

### SQL 与过滤规则

- 日期解析支持 RFC3339、时区偏移、本地日期时间和纯日期；未携带时区的日期使用环境变量时区。
- 图表 bucket 不再编码到路径中，而由 `bucket` 查询参数显式选择。
- 重构路由的 SQL 均带 `car_id` 过滤，并使用 UTC 半开区间 `[start, end)` 避免边界重复统计。
- 数据库侧分桶统一使用 `timezone($4, timestamp)` 转换到环境时区后再 `date_trunc`。
- 停车能耗、vampire drain、动能回收等较重派生计算设置查询超时；不可用的可选值返回 `null`。
- 历史聚合读使用 TTL 缓存，缓存 key 包含车辆、范围、时区、单位、scope、metric 和 bucket，减少图表客户端重复扫描。

### 代码组织

- `v1_compat_*.go`: 原 TeslaMateApi 兼容接口，保持历史路由和响应结构。
- `v1_extended_summary.go`: 扩展摘要接口。
- `v1_extended_dashboard.go`: 车辆级 dashboard 和实时快照接口。
- `v1_extended_calendar.go`: 日历聚合。
- `v1_extended_series.go`: 按领域拆分的时序数据。
- `v1_extended_distributions.go`: 按领域拆分的分布桶。
- `v1_extended_statistics.go`: 统计接口及电池、停车、动能回收聚合 helper。
- `v1_extended_insights.go`: 基线对比洞察。
- `v1_extended_locations.go`: 地点聚合。
- `v1_extended_timeline_map.go`: 时间线和访问地图接口。
- `v1_extended_models_*.go`: 扩展接口 Swagger 响应模型，按业务域拆分。
- `v1_aggregate_cache.go`: 历史聚合数据的进程内 TTL 缓存。
- `v1_legacy_extended.go`: 旧扩展重构过程遗留的未注册 handler 与图表 helper，保留给内部 helper 和迁移参考。

### 单位与空数据

- 重构后的对象响应包含稳定的 `unit` 元数据。
- 图表响应没有数据时返回空数组，不返回 `null`。
- 列表响应没有数据时返回空 `data` 数组，并保留分页信息。

## 保留、合并、删除与重命名

### 保留

- 上文列出的所有原兼容路由。
- 文档和健康检查路由。
- 原始行程/充电详情路由，用于兼容历史客户端。

### 合并

- `summaries/overview`, `summaries/drives`, `summaries/charges`, `summaries/parking`, and `summaries/state-activity` into `summary` and `statistics`.
- `dashboards/drives` and `dashboards/charges` into `summary`, `statistics`, `series`, and `distributions`.
- `activity-timeline` into `timeline`.

### 删除

- 删除“已删除的旧扩展路由”中列出的低价值别名。

### 重命名

- `calendars/*` -> unified `calendar`
- bucket-specific chart aliases -> unified `series` / `distributions`
- fragmented analytics aliases -> unified `summary`, `statistics`, `insights`

### 新增

- `summary`
- `dashboard`
- `realtime`
- `statistics`
- `series`
- `distributions`
- `insights`
- `timeline`
- `calendar`
- `map/visited`
- `locations`

## 兼容策略

- 原兼容路由继续注册，并保持路径和响应形态。
- 扩展路由视为可重构接口，可在消除重复语义时进行破坏性调整。
- OpenAPI 明确区分 Compatible API 与 Extended API。

## 数据定义与限制

- `startDate` / `endDate` 支持 RFC3339、时区偏移、本地日期时间和纯日期；本地日期使用环境变量时区。
- 纯日期格式的 `endDate` 会扩展到本地当天最后一秒，再转换成 SQL 半开区间。
- series 和 distributions 默认有时间范围约束，避免无边界扫描。
- `map/visited` 限制返回点数量，并通过 `truncated` 表示是否截断。
- `vampire_drain` 基于状态窗口和电池/续航采样估算；计算不可用或超时时返回 `null`。
- 距离单位来自 TeslaMate 设置的 `km` 或 `mi`；速度为 `km/h` 或 `mi/h`；能耗为 `Wh/km` 或 `Wh/mi`；能量为 `kWh`。
