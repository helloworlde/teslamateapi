# 项目结构 + Swagger 分组 + 注释中文化 实施方案

> 范围：**结构重构 + Swagger Tag 收敛 + 注释中文化** 三件套，一次成型。
> 当前状态：`src/` 下 57 个 `.go` 文件全部 `package main` 平铺；Swagger 有 10 个 Tag；接口/字段注释全部英文。

## 0. 现状速览

```
src/
├── NullSupport.go
├── swagger_query_params_reference.go
├── v1_TeslaMateAPICars*.go         (10 文件, package main)
├── v1_TeslaMateAPIGlobalsettings.go
├── v1_swagger_models.go
├── v2_<domain>_models.go          (battery/charging/cost/driving/lifecycle/parking/summary/update + parking/charging analytics)
├── v2_<domain>_repository.go
├── v2_<domain>_service.go
├── v2_<domain>_service_test.go
├── v2_battery_capacity_by_mileage.go
├── v2_charging_curve.go
├── v2_efficiency.go
├── v2_environmental.go
├── v2_lifecycle_extras.go
├── v2_parking_idle_periods.go
├── v2_common.go / v2_common_test.go
├── v2_handler.go / v2_handler_test.go
├── v2_models.go
├── v2_docs.go                     // 嵌入 docs_assets/ + generated/swagger.json
├── docs_assets/                   // Scalar 静态资源
├── generated/                     // swag 产物：docs.go + swagger.{json,yaml}
└── webserver.go                   // main() + 全部路由

Swagger Tag 现状（10 个）：
  V1
  V2 Battery Analytics
  V2 Charging Analytics
  V2 Cost Analytics
  V2 Driving Analytics
  V2 Environmental Analytics
  V2 Lifecycle
  V2 Parking Analytics
  V2 Summary
  V2 Update Analytics

System 类（/api/ping, /api/healthz, /api/readyz, /api/docs*）目前无 Tag。
```

构建/嵌入相关约束（不能破坏）：
- **Dockerfile** 当前 `COPY src/ .` → `go build -o app ./...`，所有源文件被拍到镜像 `/go/src/` 根目录
- **v2_docs.go** 用 `//go:embed docs_assets/*` 和 `//go:embed generated/swagger.json`，路径相对于该文件
- **Makefile** 中 `swag init -g webserver.go -d src -o src/generated`
- **测试**：`v2_*_service_test.go`、`v2_common_test.go`、`v2_handler_test.go`(8 个)需与各自包同位

---

## 1. 目标

### 1.1 模块化结构

按 **Go 标准布局** 拆分，模块边界清晰、避免循环依赖、保持可测试性。

```
.
├── cmd/
│   └── teslamateapi/
│       └── main.go                          # main()，仅做启动 & 信号处理
├── internal/
│   ├── config/                              # 环境变量、apiVersion、BasePath 常量
│   │   └── config.go
│   ├── nullx/                               # NullSupport.go (NullString / NullFloat64 / NullBool / NullInt64)
│   │   └── nullx.go
│   ├── httpx/                               # 通用 HTTP 工具：错误响应、JSON 包装、CORS、recover
│   │   ├── error.go                         # v2Error → httpx.WriteError
│   │   └── middleware.go
│   ├── server/                              # gin 引擎装配 + 路由注册入口
│   │   ├── server.go                        # New(db) *gin.Engine; Run()
│   │   ├── router.go                        # 注册 v1 / v2 / system / docs 子路由
│   │   ├── system.go                        # /api/ping /api/healthz /api/readyz
│   │   └── redirects.go                     # 旧路径 301 → /api/v1/...
│   ├── docs/                                # Scalar + swag 产物（package docs）
│   │   ├── docs.go                          # 原 v2_docs.go
│   │   ├── docs_assets/                     # embed
│   │   └── generated/                       # swag 输出：docs.go + swagger.{json,yaml}
│   ├── teslamate/                           # 与 TeslaMate Postgres schema 绑定的领域常量、单位换算
│   │   ├── units.go                         # bar→psi、km→mi 等
│   │   └── schema.go                        # 表/字段名常量（可选）
│   ├── api/
│   │   ├── common/                          # v2 envelope、V2Meta、timeBound、V2TimeRange、parseV2AnalyticsQuery、postgresDateTruncUnit、cursor 编解码
│   │   │   ├── envelope.go
│   │   │   ├── timewindow.go
│   │   │   ├── cursor.go
│   │   │   ├── query.go
│   │   │   └── common_test.go
│   │   ├── v1/                              # 全部 v1 handler（package v1）
│   │   │   ├── routes.go                    # Register(rg *gin.RouterGroup, db *sql.DB)
│   │   │   ├── cars.go
│   │   │   ├── cars_battery_health.go
│   │   │   ├── cars_tire_pressure.go
│   │   │   ├── cars_charges.go
│   │   │   ├── cars_charges_current.go
│   │   │   ├── cars_charges_details.go
│   │   │   ├── cars_drives.go
│   │   │   ├── cars_drives_details.go
│   │   │   ├── cars_updates.go
│   │   │   ├── globalsettings.go
│   │   │   ├── swagger_models.go            # 原 v1_swagger_models.go
│   │   │   └── swagger_params.go            # 原 swagger_query_params_reference.go
│   │   └── v2/
│   │       ├── routes.go                    # 总注册入口；按子域 wire 各 service
│   │       ├── handler.go                   # capabilities / by_period 等跨域 handler
│   │       ├── handler_test.go
│   │       ├── models.go                    # 跨域共享 model
│   │       ├── battery/
│   │       │   ├── models.go
│   │       │   ├── repository.go
│   │       │   ├── service.go
│   │       │   ├── service_test.go
│   │       │   ├── capacity_by_mileage.go
│   │       │   └── handlers.go
│   │       ├── charging/
│   │       │   ├── models.go ... curve.go ... handlers.go
│   │       ├── driving/
│   │       │   ├── ... efficiency.go handlers.go
│   │       ├── parking/
│   │       │   ├── ... idle_periods.go handlers.go
│   │       ├── cost/
│   │       ├── environmental/
│   │       ├── update/
│   │       ├── lifecycle/
│   │       │   ├── ... places.go odometer_series.go geofences.go handlers.go
│   │       └── summary/
│   │           └── by_period.go
└── (顶层 docs/ 保持，存放 markdown 计划文档)
```

边界约定：
- `cmd/` 只做启动；不写业务逻辑。
- `internal/api/v1` 与 `internal/api/v2/<domain>` 不能互相导入；如需共享，提到 `internal/api/common`。
- `internal/api/v2/<domain>` 之间也不互相导入；跨域逻辑（如 capabilities、by_period）放 `internal/api/v2`（顶层）。
- `internal/server` 依赖各 `api/*` 的 `Register(...)` 函数，反向不依赖。
- `internal/docs` 嵌入路径就近：`docs_assets/` 和 `generated/` 与 `docs.go` 同级。
- 测试文件随包迁移（`*_test.go`）。

### 1.2 Swagger Tag 收敛为三组

只保留：

| Tag | Name in Spec | 中文显示 | 涵盖路径 |
|---|---|---|---|
| `system` | system | 系统 | `/api/ping`, `/api/healthz`, `/api/readyz`, `/api/docs*`, `/api/v1/`, `/api/v2/` 这类元信息 |
| `v1` | v1 | V1 接口 | `/api/v1/...` 全部 |
| `v2` | v2 | V2 接口 | `/api/v2/...` 全部 |

实现方式：所有 `// @Tags` 改写为 `system` / `v1` / `v2`；在 `webserver.go` 顶部用 swag 的 `@tag.name` / `@tag.description` 注释为三个 tag 设置中文显示名 + 描述：

```go
// @tag.name system
// @tag.description 系统状态、健康检查、API 文档元信息
//
// @tag.name v1
// @tag.description V1 接口 — TeslaMate 原始数据查询（车辆、充电、行驶、OTA、电池、胎压、全局设置）
//
// @tag.name v2
// @tag.description V2 接口 — 聚合分析（充电/行驶/驻车/电池/费用/环境/更新/汇总/生命周期）
```

> **不动 URL 路径**。子域信息保留在 path 段（如 `/v2/cars/{id}/charging/curve`）和 `@Summary` 中文文案中即可。前端按 path 分组依然清晰，但 Swagger 侧栏只有 3 个抽屉，符合用户预期。

### 1.3 注释中文化

涉及三类注解：

#### A. 接口注解（每个 handler）
```go
// @Summary    <中文短标题>
// @Description <中文描述>
// @Tags       v1 | v2 | system
// @Produce    json
// @Param      <name> path/query <type> <required> "<中文说明>"
// @Success    200 {object} XxxResponse "<中文成功描述>"
// @Failure    400 {object} V2ErrorResponse "<中文 400 描述>"
// @Router     /xxx [get]
```

#### B. 模型字段（@name + struct field doc）
保留 `// @name V2BatteryAPIResponse`（swag 生成 schema 名称用），并给关键字段加中文 `description`：

```go
// V2BatteryResponse 电池分析响应。
// @name V2BatteryResponse
type V2BatteryResponse struct {
    LatestRatedRange    *float64 `json:"latest_rated_range,omitempty" example:"412.5"`     // 最新额定续航 (km)
    LatestIdealRange    *float64 `json:"latest_ideal_range,omitempty"`                      // 最新理想续航 (km)
    EstimatedFullRange  *float64 `json:"estimated_full_range,omitempty"`                    // 估算满电额定续航
    ...
}
```

`description` 通过结构体 tag `extensions:"x-display-name=xxx"` 不被 swag 解析；本项目用注释行尾 `//` 格式即可（swag 读取行内 `//` 作为字段描述）。

#### C. 全局 API 描述（webserver.go 顶部）
```go
// @title       TeslaMateApi
// @version     {build-time}
// @description TeslaMate 数据查询与分析 API。提供 V1（原始数据）和 V2（聚合分析）两套接口。
// @BasePath    /api
```

文案语料统一在一个 `docs/v2-i18n-glossary.md` 维护（实施期内创建），保持术语一致：

| 英文 | 中文 |
|---|---|
| rated range | 额定续航 |
| ideal range | 理想续航 |
| trip / drive | 行程 |
| charging session | 充电会话 |
| odometer | 里程表 |
| consumption | 能耗 |
| efficiency | 能效 |
| state of health | 健康度（估算） |
| idle / parking | 驻车 |
| update / OTA | OTA 更新 |
| timeseries | 时序 |
| breakdown | 分项 |
| summary | 汇总 |
| period | 周期 |
| net energy | 净能量 |
| battery energy | 电池端能量 |
| wall energy | 墙端能量 |

---

## 2. 执行阶段（每阶段单独 commit，build/vet/test/swag 全绿后停下）

### Phase R0 — 准备
- 建 `docs/v2-restructure-plan.md`（本文件）+ 占位 `docs/v2-i18n-glossary.md`
- baseline：`make check` 全绿
- **commit**: `docs(plan): restructure + tag consolidation + i18n plan`

### Phase R1 — 抽 `cmd/teslamateapi/main.go` 与 `internal/server` 骨架，保留 `package main` 兼容
> 目的：把 main() 与全局路由从 `src/webserver.go` 抽出，但暂不动 `src/*.go` 其它文件的包。
- 新建 `cmd/teslamateapi/main.go`：仅 `func main()` 调用 `server.Run(db)`
- 新建 `internal/server/server.go`、`internal/server/router.go`、`internal/server/system.go`、`internal/server/redirects.go`
- 新建 `internal/server/v1_register.go` 暂时只是 `func RegisterV1(rg *gin.RouterGroup, db *sql.DB)`，里面调用 `src/...` 已有 `TeslaMateAPI*V1` 函数（通过让 v1 handler 暂存于一个临时桥接 `internal/legacy` 或直接 import 老包）
- 调整 Dockerfile：`COPY . .` + `go build -o app ./cmd/teslamateapi`
- 调整 Makefile：`build: go build -o $(BINARY) ./cmd/teslamateapi`；`swag init -g cmd/teslamateapi/main.go -d cmd,internal,src -o internal/docs/generated`（先指多目录扫描，过渡期）
- 校验：`make check` 全绿
- **commit**: `refactor(structure): introduce cmd/ + internal/server skeleton`

> 风险点：本阶段需要保证 v2_docs.go 的 `//go:embed generated/swagger.json` 在 Dockerfile 改动后仍可加载——R1 不动 v2_docs.go 位置，仅改 Makefile 输出到 `internal/docs/generated/` **是 R3** 的事；R1 swag 仍输出到 `src/generated/`。

### Phase R2 — 把 v1 handler 迁到 `internal/api/v1`（package v1）
- `mkdir internal/api/v1`
- 把所有 `src/v1_*.go`、`src/v1_swagger_models.go`、`src/swagger_query_params_reference.go` 移到 `internal/api/v1/`，重命名（去 v1_ 前缀），改 `package v1`
- handler 函数名简化：`TeslaMateAPICarsV1` → `Cars`，`TeslaMateAPICarsBatteryHealthV1` → `BatteryHealth` …（或保留长名，但去 V1 后缀；二选一，方案选**简短 + Register 内集中导出**）
- 处理 NullString 等依赖：把 `NullSupport.go` 抽到 `internal/nullx`，v1 import 之
- `internal/api/v1/routes.go`：`func Register(rg *gin.RouterGroup, db *sql.DB)`
- `internal/server/router.go` 改为调用 `v1.Register(api.Group("/v1"), db)`
- swag 注释 `@Tags` 全部改为 `v1`
- 校验：`make check` + `make docs`，对比 swagger.json 中 v1 路径不变
- **commit**: `refactor(v1): move handlers to internal/api/v1 package`

### Phase R3 — 迁 v2 共享层 + docs 嵌入
- `internal/api/common/`：吸收 `v2_common.go` / `v2_common_test.go` / `v2_models.go` 中跨域类型（V2Meta、V2TimeRange、V2ErrorResponse、timeBound、parseV2AnalyticsQuery、postgresDateTruncUnit、cursor 编解码、`v2Error`）→ 抽为 `httpx.WriteError`
- `internal/docs/`：移动 `v2_docs.go`、`docs_assets/`，**generated/ 暂留 src/generated**（避免 R3 也动 swag 输出）
- 改 `v2_docs.go` 的 embed 为 `//go:embed ../../../src/generated/swagger.json` ❌ 不允许 (embed 要 file 在子树内)
  → 替代：把 `src/generated/` 物理移到 `internal/docs/generated/`，**同时**改 Makefile/Dockerfile
- 校验：`make docs && make build`，启动跑 `/api/docs/scalar` 渲染正常
- **commit**: `refactor(docs+common): consolidate shared v2 types and docs embed`

### Phase R4 — 拆 v2 各子域到 `internal/api/v2/<domain>` 包
逐子域迁移，每域一个 commit；推荐顺序（依赖少 → 依赖多）：
1. `battery` (含 `capacity_by_mileage`)
2. `charging` (含 `curve`)
3. `driving` (含 `efficiency`)
4. `parking` (含 `idle_periods`)
5. `cost`
6. `environmental`
7. `update`
8. `lifecycle` (places, odometer_series, geofences from `lifecycle_extras`)
9. `summary` (含 `by_period`)

每域操作：
- 移动 `v2_<domain>_models.go` `_repository.go` `_service.go` `_service_test.go` 及对应 handler 文件
- 改 `package <domain>`，类型/函数首字母大写或保持
- 在子包内提供 `Register(rg *gin.RouterGroup, deps Dependencies)` 工厂函数；`internal/api/v2/routes.go` 收口
- 全部 handler 的 `@Tags` 改为 `v2`；`@Summary` `@Description` 中文化
- 跨域 handler（capabilities、by_period）留在 `internal/api/v2/handler.go`
- 校验：每子域 commit 前 `make check`

### Phase R5 — Tag 收敛 + Tag 描述定义
- `cmd/teslamateapi/main.go` 顶部加 `@tag.name` / `@tag.description` 三段
- 全局 grep 确保只剩三种 `@Tags` 值：`system` / `v1` / `v2`
- system handler（healthz/readyz/ping/docs）补上 `@Tags system` + `@Summary` 中文
- 重新生成 swagger，验证 swagger.json `tags` 数组只 3 项
- **commit**: `refactor(swagger): consolidate tags to system/v1/v2 with chinese descriptions`

### Phase R6 — 字段中文化（按子域分批）
- 建 `docs/v2-i18n-glossary.md` 锁词表
- 按子域批次给每个 model struct field 加行内中文 `// 中文说明`
- v1 model 同步处理
- 重新生成 swagger，抽样验证 Scalar 渲染显示中文
- 推荐拆 5–6 个 commit（按子域），不要一锤子提
- 例：
  - `docs(swagger): translate v1 models to chinese`
  - `docs(swagger): translate v2 battery/charging/driving models to chinese`
  - `docs(swagger): translate v2 parking/cost/environmental/update models to chinese`
  - `docs(swagger): translate v2 lifecycle/summary/common models to chinese`
  - `docs(swagger): translate handler @Summary/@Description to chinese`

### Phase R7 — 收尾
- 删除空的 `src/` 残留（`webserver.go` 应迁完后变空可删）
- 更新 `Dockerfile` `COPY . .` 路径与 `go build ./cmd/teslamateapi`
- 更新 `Makefile`：
  - `swag init -g cmd/teslamateapi/main.go -d cmd,internal -o internal/docs/generated`
  - `BINARY := bin/teslamateapi`
  - `build: go build -o $(BINARY) ./cmd/teslamateapi`
- 更新 `README.md` 项目结构段
- **commit**: `chore(structure): finalize layout, remove src/ shell`

---

## 3. 风险与缓解

| 风险 | 影响 | 缓解 |
|---|---|---|
| `//go:embed` 路径在迁包后失效 | docs/scalar 500 | R3 物理移动 generated/ 到 docs 包同级，路径相对 |
| swag 不能跨包扫描 model 引用 | swagger.json 缺 schema | swag init 加 `-d cmd,internal --parseDependency --parseInternal` |
| 子域间隐式依赖（如 lifecycle 引用 charging types） | 编译失败/循环依赖 | 共享类型上提 `internal/api/common`；不允许 v2 子域互引 |
| Dockerfile `COPY src/ .` 假设被打破 | CI 镜像构建失败 | R1 同步改 Dockerfile + Makefile，本地跑 `docker build` 验证 |
| `package main` 内的全局变量（apiVersion 等）跨包后不可见 | 链接 ldflags 失效 | `apiVersion` 移到 `internal/config`，ldflags 改为 `-X github.com/.../internal/config.APIVersion=...` |
| 旧的 `main.` schema 前缀去除脚本 `scripts/normalize_swagger_main_prefix.py` 不存在 | swagger 含 `main.` 前缀 | 改用 swag 自带 `--instanceName` 或直接迁包后 schema 名前缀变成包名（不再是 main.），脚本删除 |
| 测试文件 import 老包路径 | 测试编译失败 | 每子域迁移时同步改测试 import |
| 中文文案与 Scalar UI 字体兼容 | 显示异常 | `docs_assets/` 中字体配置已支持中文（CJK），实测验证 |

---

## 4. 不做的事（明确范围）

- **不改路径**：所有 URL 保持现样（`/api/v1/...`、`/api/v2/...`）。
- **不改 wire format**：JSON 字段名、字段语义、HTTP 状态码、错误码全部保持。
- **不改业务逻辑**：SQL/聚合/单位换算/cursor 协议保持。
- **不删 v2-enhancement-plan/progress 文档**。
- **不动 deferred 的 timeline/`type:"missing"` 与 park `consumption_during_park`**（独立后续 PR）。

---

## 5. 验收清单（每 Phase 与最终）

每 Phase 必须满足：
- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] `go test ./... -count=1` 通过（无 SKIP 增加）
- [ ] `make docs` 重新生成 swagger，diff 仅是 tag 收敛 + 中文文案 + schema 名前缀变化
- [ ] `make build` 二进制产出
- [ ] 启动后 `/api/docs/scalar` 可加载，三个 Tag 抽屉只有 system/v1/v2

最终 R7 完成后还需：
- [ ] `docker build .` 成功（本地验证）
- [ ] 抽 5 个旧 `/v1/...` + 5 个 `/v2/...` URL 跑回归（`curl` 对比 R0 baseline 响应字段不增不减不变）
- [ ] swagger.json `tags` 数组 == 3
- [ ] 全局 grep `@Tags ` 出现的值 == {system, v1, v2}
- [ ] 全局 grep `// @Summary` 行尾文案均为中文
- [ ] README 项目结构段已更新

---

## 6. 一次性回滚策略

每 Phase 单 commit；如果某 Phase 后回归发现问题，`git revert <sha>` 回滚单步。R4 子域分 9 commit 也是为了细粒度回滚。

---

请确认：
1. **结构方案**（cmd + internal/api/{v1,v2/<domain>,common} + internal/server + internal/docs + internal/nullx + internal/httpx + internal/config）是否接受？
2. **Tag 收敛 3 项 + 中文 description 标题**是否接受？是否需要 4 项（增加 `docs` 单独一组）？
3. **中文化粒度**：handler `@Summary/@Description` + model 字段行尾注释。是否要顺便译 `// @Param` 描述？（计划中默认译，确认即可）
4. **Phase 数（R0–R7）+ 子域 9 步迁移**：可接受？想缩短为更粗粒度（如 R4 一锤迁完 v2）？
