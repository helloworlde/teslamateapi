package main

// @Summary API 根路径
// @Tags System
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router / [get]
func swaggerAPIRoot() {}

// @Summary API v1 根路径
// @Tags System
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router /v1 [get]
func swaggerAPIV1Root() {}

// @Summary 连通性检查
// @Tags System
// @Produce json
// @Success 200 {object} APISystemMessageResponse
// @Router /ping [get]
func swaggerPing() {}

// @Summary 查询车辆列表
// @Description 兼容接口：返回全部车辆。历史兼容原因下，部分数据库空值仍可能表现为空字符串或 0；部分旧错误仍可能以 HTTP 200 返回只包含 error 字段的 JSON。
// @Tags Compatible API
// @Produce json
// @Success 200 {object} CarsV1Envelope
// @Router /v1/cars [get]
func swaggerCars() {}

// @Summary 查询单车信息
// @Description 兼容接口：返回匹配车辆，结果位于 data.cars 中，通常只有一条。历史兼容原因下，部分旧错误仍可能以 HTTP 200 返回只包含 error 字段的 JSON。
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} CarsV1Envelope
// @Router /v1/cars/{CarID} [get]
func swaggerCar() {}

// @Summary 电池健康
// @Description 兼容接口：保留原 battery-health 响应结构。需要图表友好数据时，建议使用 `/v1/cars/{CarID}/series/battery?metrics=range,soc`。
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} BatteryHealthV1Envelope
// @Router /v1/cars/{CarID}/battery-health [get]
func swaggerBatteryHealth() {}

// @Summary 查询充电记录
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "开始时间，支持 RFC3339、带时区偏移、本地日期时间和纯日期"
// @Param endDate query string false "结束时间，支持 RFC3339、带时区偏移、本地日期时间和纯日期"
// @Param page query int false "页码"
// @Param show query int false "每页数量"
// @Param limit query int false "每页数量别名"
// @Param offset query int false "偏移量别名"
// @Param sort query string false "start_date|-start_date|duration|-duration|cost|-cost|energy|-energy"
// @Param include query string false "summary,location,energy,cost"
// @Success 200 {object} ChargesListV1Envelope
// @Router /v1/cars/{CarID}/charges [get]
func swaggerCharges() {}

// @Summary 当前充电状态
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} CurrentChargeV1Envelope
// @Router /v1/cars/{CarID}/charges/current [get]
func swaggerCurrentCharge() {}

// @Summary 充电详情
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param ChargeID path int true "充电记录 ID"
// @Success 200 {object} ChargeDetailsV1Envelope
// @Router /v1/cars/{CarID}/charges/{ChargeID} [get]
func swaggerChargeDetails() {}

// @Summary 查询可用命令
// @Description 仅在 ENABLE_COMMANDS=true 时注册。接口响应永远不会暴露 Tesla 账号 access/refresh token。
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} EnabledCommandsV1Envelope
// @Router /v1/cars/{CarID}/command [get]
func swaggerCommandCatalog() {}

// @Summary 执行车辆命令
// @Description 仅在 ENABLE_COMMANDS=true 且命令被允许时注册。接口响应永远不会暴露 Tesla 账号 access/refresh token。
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param Command path string true "命令名称"
// @Success 200 {object} TeslaPassthroughJSONBody
// @Router /v1/cars/{CarID}/command/{Command} [post]
func swaggerExecuteCommand() {}

// @Summary 查询行程记录
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "开始时间，支持 RFC3339、带时区偏移、本地日期时间和纯日期"
// @Param endDate query string false "结束时间，支持 RFC3339、带时区偏移、本地日期时间和纯日期"
// @Param minDistance query number false "最小行程距离"
// @Param maxDistance query number false "最大行程距离"
// @Param page query int false "页码"
// @Param show query int false "每页数量"
// @Param limit query int false "每页数量别名"
// @Param offset query int false "偏移量别名"
// @Param sort query string false "start_date|-start_date|distance|-distance|duration|-duration|efficiency|-efficiency"
// @Param include query string false "summary,locations,energy"
// @Success 200 {object} DrivesListV1Envelope
// @Router /v1/cars/{CarID}/drives [get]
func swaggerDrives() {}

// @Summary 行程详情
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param DriveID path int true "行程 ID"
// @Success 200 {object} DriveDetailsV1Envelope
// @Router /v1/cars/{CarID}/drives/{DriveID} [get]
func swaggerDriveDetails() {}

// @Summary 查询日志采集状态
// @Description 仅在 ENABLE_COMMANDS=true 时注册，只返回允许的日志采集命令。
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} EnabledCommandsV1Envelope
// @Router /v1/cars/{CarID}/logging [get]
func swaggerLoggingGet() {}

// @Summary 更新日志采集状态
// @Description 仅在 ENABLE_COMMANDS=true 且日志采集命令被允许时注册。
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param Command path string true "日志采集命令"
// @Success 200 {object} TeslaPassthroughJSONBody
// @Router /v1/cars/{CarID}/logging/{Command} [put]
func swaggerLoggingPut() {}

// @Summary 当前车辆状态
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} CarStatusV1Envelope
// @Router /v1/cars/{CarID}/status [get]
func swaggerStatus() {}

// @Summary 查询车辆更新记录
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param page query int false "页码"
// @Param show query int false "每页数量"
// @Success 200 {object} UpdatesListV1Envelope
// @Router /v1/cars/{CarID}/updates [get]
func swaggerUpdates() {}

// @Summary 唤醒车辆
// @Description 仅在 ENABLE_COMMANDS=true 且 COMMANDS_WAKE、COMMANDS_ALL 或 COMMANDS_ALLOWLIST 允许 /wake_up 时注册。接口响应永远不会暴露 Tesla 账号 access/refresh token。
// @Tags Compatible API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} TeslaPassthroughJSONBody
// @Router /v1/cars/{CarID}/wake_up [post]
func swaggerWakeUp() {}

// @Summary 全局设置
// @Tags Compatible API
// @Produce json
// @Success 200 {object} GlobalsettingsV1Envelope
// @Router /v1/globalsettings [get]
func swaggerGlobalSettings() {}

// @Summary 车辆摘要
// @Description 扩展接口：返回指定时间范围内的规范化摘要，包含 overview、driving、charging、parking、battery、efficiency、cost、quality、state 等稳定分区，不再依赖旧 include 参数拼装稀疏字段。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param period query string false "时间范围类型：year|month|week|custom，默认 month"
// @Param date query string false "period 模式下的参考日期"
// @Param startDate query string false "自定义范围开始时间；出现时必须同时提供 endDate"
// @Param endDate query string false "自定义范围结束时间；出现时必须同时提供 startDate"
// @Success 200 {object} SummaryV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/summary [get]
func swaggerSummaryV2() {}

// @Summary 车辆仪表盘统计
// @Description 扩展接口：仅返回所选时间范围内的整车级统计和概览。实时状态、时序图、分布图、洞察、时间线、行程明细、充电明细分别由独立接口提供。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param period query string false "时间范围类型：year|month|week|custom"
// @Param date query string false "参考日期，支持 YYYY-MM-DD 或 RFC3339"
// @Param startDate query string false "自定义范围开始时间"
// @Param endDate query string false "自定义范围结束时间"
// @Success 200 {object} DashboardV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/dashboard [get]
func swaggerDashboardV2() {}

// @Summary 实时车辆快照
// @Description 扩展接口：从最新位置、最新状态和最新充电过程派生当前车辆快照，用于替代 dashboard 中原本揉杂的实时信息。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} RealtimeV2Envelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/realtime [get]
func swaggerRealtimeV2() {}

// @Summary 日历聚合
// @Description 扩展接口：按 day/week/month 聚合行程和充电数据，返回总体 summary 和倒序 bucket 列表。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Param bucket query string false "聚合粒度：day|week|month"
// @Success 200 {object} CalendarV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/calendar [get]
func swaggerCalendarV2() {}

// @Summary 分区统计
// @Description 扩展接口：返回 overview、drive、charge、battery 等明确分区，覆盖效率、费用、能量、停车等聚合指标。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param period query string false "时间范围类型：year|month|week|custom"
// @Param date query string false "参考日期，支持 YYYY-MM-DD 或 RFC3339"
// @Param startDate query string false "自定义范围开始时间"
// @Param endDate query string false "自定义范围结束时间"
// @Success 200 {object} StatisticsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/statistics [get]
func swaggerStatisticsV2() {}

// @Summary 行程时序数据
// @Description 扩展接口：按 bucket 聚合行程指标，并把同一时间点的多个指标合并到一个 point 对象中，按时间倒序返回。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param metrics query string false "逗号分隔指标：distance,efficiency,speed,max_speed,motor_power,regen_power,elevation,outside_temp,inside_temp,energy,regeneration"
// @Param bucket query string false "聚合粒度：raw|hour|day|week|month|year"
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Success 200 {object} SeriesV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/series/drives [get]
func swaggerDriveSeriesV2() {}

// @Summary 充电时序数据
// @Description 扩展接口：按 bucket 聚合充电指标，默认包含 start_soc 和 end_soc，并把同一时间点的多个指标合并到一个 point 对象中。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param metrics query string false "逗号分隔指标：energy,power,cost,start_soc,end_soc"
// @Param bucket query string false "聚合粒度：raw|hour|day|week|month|year"
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Success 200 {object} SeriesV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/series/charges [get]
func swaggerChargeSeriesV2() {}

// @Summary 电池时序数据
// @Description 扩展接口：返回电量百分比和额定续航等电池时序指标，按时间倒序排列。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param metrics query string false "逗号分隔指标：soc,range"
// @Param bucket query string false "聚合粒度：raw|hour|day|week|month|year"
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Success 200 {object} SeriesV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/series/battery [get]
func swaggerBatterySeriesV2() {}

// @Summary 状态时序数据
// @Description 扩展接口：返回车辆状态持续时间和停车能耗（vampire drain）等状态派生指标。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param metrics query string false "逗号分隔指标：duration,vampire_drain"
// @Param bucket query string false "聚合粒度：raw|hour|day|week|month|year"
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Success 200 {object} SeriesV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/series/states [get]
func swaggerStateSeriesV2() {}

// @Summary 行程分布数据
// @Description 扩展接口：返回行程开始小时、星期、距离、时长、速度、能耗效率等分布桶。桶顺序固定，并补齐 0 值。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param metrics query string false "逗号分隔指标：start_hour,weekday,distance,duration,speed,efficiency"
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Success 200 {object} DistributionsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/distributions/drives [get]
func swaggerDriveDistributionsV2() {}

// @Summary 充电分布数据
// @Description 扩展接口：返回充电开始小时、星期、补能量、时长、功率、费用等分布桶。桶顺序固定，并补齐 0 值。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param metrics query string false "逗号分隔指标：start_hour,weekday,energy,duration,power,cost"
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Success 200 {object} DistributionsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/distributions/charges [get]
func swaggerChargeDistributionsV2() {}

// @Summary 洞察
// @Description 扩展接口：将当前时间范围与上一个等长基线范围对比，生成效率、费用、充电、电池和异常类洞察。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Param types query string false "逗号分隔类型：efficiency,cost,charging,driving,battery,anomaly"
// @Param limit query int false "洞察数量限制，1-100，默认 20"
// @Success 200 {object} InsightsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/insights [get]
func swaggerInsightsV2() {}

// @Summary 时间线
// @Description 扩展接口：统一返回行程、充电、状态活动时间线，支持分页，按时间倒序。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Param limit query int false "分页大小"
// @Param offset query int false "偏移量"
// @Success 200 {object} TimelineV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/timeline [get]
func swaggerTimelineV2() {}

// @Summary 访问地图
// @Description 扩展接口：返回访问过的坐标点、边界和热力图基础数据，用于地图覆盖范围渲染。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Success 200 {object} VisitedMapV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/map/visited [get]
func swaggerVisitedMapV2() {}

// @Summary 地点聚合
// @Description 扩展接口：合并行程起终点和充电地点，返回事件次数、坐标、充电能量、费用和最近出现时间。
// @Tags Extended API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param startDate query string false "范围开始时间"
// @Param endDate query string false "范围结束时间"
// @Param limit query int false "最大地点数量，最多 100"
// @Success 200 {object} LocationsV2Envelope
// @Failure 400 {object} v1ErrorEnvelope
// @Failure 404 {object} v1ErrorEnvelope
// @Failure 500 {object} v1ErrorEnvelope
// @Router /v1/cars/{CarID}/locations [get]
func swaggerLocationsV2() {}
