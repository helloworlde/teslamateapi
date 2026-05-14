# TeslaMateAPI 接口设计规范

本规范是后续所有 v1/v2 接口改动必须遵守的底线。审计文档（`docs/v2-api-audit-2026-05.md`）暴露的问题之所以反复发生，根因是过往迭代只在 "翻译/补字段" 的执行层打转，没有先确立一份关于 "什么字段应当存在、什么聚合有意义、参数与默认值如何协同" 的判断标准。本文先立这套标准。

> 适用范围：`internal/api/v1`、`internal/api/v2`、`internal/server`，以及生成的 swagger 文档。
> 后续每条 PR 都必须能在 PR 描述里指出 "我满足了规范的哪些条" 或 "我违反了哪条、为什么"。

---

## 0. 核心原则

每个字段、每个聚合、每个默认值都要能回答一个问题：**读这个数的人会用它做什么决策？** 答不上来的字段，删；答得上但需要另一种聚合的，换。

---

## 1. 字段语义规范（最关键）

### 1.1 禁止无意义的均值

下列情形 **必须** 用 min/max/中位数/分位数/分布，而不是简单平均：

| 数据类型 | 禁止 | 允许 |
|---|---|---|
| 受异常值主导（一次超充压均功率） | 算术平均 | 中位数、P25/P75；或者按场景拆分（AC/DC、城市/高速） |
| 用于发现异常的传感器（胎压、电压） | 时间窗内的算术平均 | min/max（关注异常方向）；可选 P50 提供基线 |
| 单位非线性的量（速度 = 距离/时间） | per-row 算术平均 | 总量加权（`SUM(distance)/SUM(duration)`） |
| 跨期会失真的 ratio（成本/距离） | 跨周期混合 | 同周期内对齐；或显式标注 `same_window=true` 并在 swagger 中说明 |
| 时序展示极值（温度、海拔、功率） | 仅 avg | min/max 必备，avg 为补充 |

执行时机：见审计文档 §1.1、§1.5、§2.1。

### 1.2 时间序列必须保留状态变化语义

**判定准则**：当一个 timeseries 端点在做 "把 N 行原始记录折叠成 M 行均值"，且 M < N/3 时，要重新审视它的存在价值。

- 行驶/充电这种 **本身就是离散事件** 的数据，时序展示应优先返回 per-event 列表（最多附带轻量分桶聚合），而不是抹平成日均/月均。
- 状态变化（SoC 升降、充电会话、停车段）应附带：起止时间、起止值、活动类型、关联 ID（drive_id / charging_process_id）。
- 真正需要 "趋势线" 的场景（如成本月度走势）才使用 bucketed timeseries，且 bucket 内必须同时返回 sum/min/max/avg。

执行时机：见审计文档 §1.2、§2.3。

### 1.3 禁止与 v1 重复的 v2 端点

新增 v2 分析端点必须先回答：**"这个端点提供了 v1 没有的什么信号？"** 答案有三种合法情形：

1. **聚合**：v1 是单条记录，v2 提供时间窗或维度聚合（如 `/v2/.../analytics/charging` 聚合 `v1/charges`）。
2. **跨表组合**：v2 联接多张表得到 v1 单端点拿不到的复合字段（如 `efficiency` = drives + charges）。
3. **派生指标**：v2 在原始数据上做模型计算（如 `battery/capacity_by_mileage`）。

不满足以上任一项的 v2 端点应当合并到 v1。审计中 `/v2/analytics/battery` 就是反例 —— 字段集是 v1 `battery-health` 的真子集。

### 1.4 字段必须自描述其单位与口径

- 数值字段名前缀/后缀显式带单位：`*_km`、`*_kwh`、`*_kw`、`*_pct`、`*_seconds`、`*_minutes`。
- 比例字段：`_pct`（0–100，整数或小数皆可，需 swagger 注明）或 `_ratio`（0.0–1.0）。
- 时间字段：RFC3339 字符串；命名以 `_at` / `_start` / `_end` 结尾。
- 货币字段必须在响应 meta 中标注 `currency`，且 `currency` 不得硬编码（详见 §3.2）。

---

## 2. 参数与默认值规范

### 2.1 参数必须真正生效；无效组合必须显式报错

- **任何被读取的查询参数都必须影响响应**，否则就是 contract bug。
- 参数之间存在依赖关系（如 `group_by` 依赖 `include=timeseries`）时，必须在 handler 层校验：
  - 缺少前置条件时返回 **400** + `code: PARAM_DEPENDENCY_VIOLATION`，错误体包含具体哪个参数依赖哪个。
  - 不允许 "静默丢弃" 或 "无声补默认"。
- swagger 注解必须用一句话写清依赖：例如 `@Description ...; group_by 仅在 include=timeseries 时生效，否则返回 400`。

执行时机：见审计文档 §1.4、§2.2。

### 2.2 默认值必须 "裸调可用"

- 用户不传任何参数直接 GET 一次，应当能拿到 **有意义** 的响应。空响应不算有意义。
- 具体规则：
  - 时间窗：分析端点的默认窗口要根据数据语义独立设定。`charging/curve` 这种本质上是 "全生命周期统计" 的端点，默认就是 lifetime；`/analytics/charging` 这种月度审视的端点，默认 month。**不要无差别套用同一套默认值。**
  - `min_*` 这类阈值参数，要保证默认值下大多数数据集都不会被滤光。给阈值时要在 swagger 注释里给出 "本数据集典型值"。
  - 分页：默认 `page=1, show=100`，最大 `show ≤ 1000`，超出报 400。

执行时机：见审计文档 §1.5。

### 2.3 时区与货币必须可配且不可硬编码

- 时区（`Timezone`）来源优先级：`?timezone=` 查询参数 → `apicommon.AppUsersTimezone`（来自 `TZ` 环境变量） → UTC（兜底，仅当前两者不可用且必须告警日志）。
- 货币（`Currency`）来源优先级：`?currency=` 查询参数（仅展示用，不做汇率换算） → TeslaMate `settings.currency` → 环境变量 `CURRENCY` → `"USD"`（兜底）。
- **任何 `Timezone: "UTC"` 或 `Currency: "CNY"` 的硬编码都视为 bug**。
- 时间字段在响应中统一为 RFC3339（带偏移），而不是裸字符串。

执行时机：见审计文档 §2.4。

---

## 3. 端点演进与兼容性

### 3.1 破坏性变更走两步

1. 第一个 minor 版本：新字段加入；旧字段保留并在 swagger 上标记 `deprecated: true`，响应头加 `Deprecation: true` 与 `Sunset: <RFC3339 date>`。
2. 第二个 minor 版本：删除旧字段。

破坏性变更必须在 `CHANGELOG.md` 单独章节列出，并在 PR 描述中显式提到 "本变更属于破坏性变更，遵守 §3.1 两步流程"。

### 3.2 端点合并/删除走 deprecate 流程

- 删除端点前必须先在响应头加 `Deprecation: true` 至少一个 minor。
- 合并到其它端点时，被合并端点的所有字段必须能在新端点上找到等价物，文档里要给出 "字段映射表"。

### 3.3 capabilities 必须随路由表自动同步

`/v2/capabilities` 不能再手动维护静态映射。要么用反射/注册中心，要么在每次 `RegisterV2Routes` 时同步追加。手动维护会再次漂移，禁止。

---

## 4. 测试规范

### 4.1 必须覆盖的测试维度

每个新增/修改的 handler/service 必须同时包含以下用例：

1. **Happy path**：典型参数组合返回 200 + 字段完备。
2. **默认值裸调**：不传任何查询参数 → 200 + 非空响应（除非数据本身为空）。
3. **参数依赖违反**：`group_by` 没带 `include` → 400 + 指定 error code（见 §2.1）。
4. **边界**：空数据集、单条数据、跨期数据、时区分界点（00:00 前后 1 分钟各一条）。
5. **聚合语义**：用已知输入断言聚合输出（不能只断言字段存在）。例如：
   - `avg_speed` 测试要包含 "1 km × 1 分钟 + 200 km × 120 分钟" 的数据，断言 `avg_speed ≈ 100.5 km/h`（距离加权），而不是 `~60 km/h`（行加权）。

### 4.2 决策驱动的字段断言

测试不仅要验证字段存在，还要验证字段值能支撑业务决策：

- "此电池组本月的最大充电功率？" → 测试 curve 的 `max_power`。
- "我家附近这周的胎压是不是在掉？" → 测试 tire-pressure 历史的 `*_min` 是否捕捉到下行尖峰。

如果一个字段写不出决策驱动的测试，说明它可能不该存在。

---

## 5. 命名与响应格式

### 5.1 端点路径

- v1：保持 TeslaMate 原始数据贴合（`/v1/cars/:CarID/charges`）。
- v2：`/v2/cars/:CarID/<domain>[/<sub>]`，`<domain>` ∈ `analytics | lifecycle | summary | parking | battery | timeline | updates`。
- 不允许 `/analytics/x` 与 `/lifecycle/x` 同时存在功能重叠的端点（如已暴露的 odometer_series 与 by_period 重叠）；新增前必须看是否能并入既有 domain。

### 5.2 响应包络

所有响应必须走 `apicommon.HandleSuccessResponse` / `HandleErrorResponse`，统一形如：

```jsonc
{
  "data": { ... },                 // 业务数据
  "meta": {                        // 元数据
    "timezone": "Asia/Shanghai",
    "currency": "USD",
    "applied_filters": { ... },    // 实际生效的查询参数（含默认值回填）
    "deprecations": [              // 端点本身或字段的弃用提示
      { "field": "fl_avg", "sunset": "2026-08-01" }
    ]
  }
}
```

`meta.applied_filters` 是新规范要求：让客户端能看见自己 "实际拿到" 的查询条件，避免再次出现 group_by 静默丢弃这种 silent failure。

### 5.3 错误格式

```jsonc
{
  "error": {
    "code": "PARAM_DEPENDENCY_VIOLATION",
    "message": "group_by requires include=timeseries",
    "details": { "param": "group_by", "depends_on": "include=timeseries" }
  }
}
```

`code` 必须是稳定枚举，文档化在 `docs/api-error-codes.md`（待建）。

---

## 6. swagger 注解规范

每个端点的 swagger 注解必须包含：

- `@Summary` 一句话，主语是 "这个端点回答的问题"，而不是 "这个端点查的表"。
  - ✗ "查询充电会话表"
  - ✓ "按时间窗口聚合的充电统计（次数/能量/功率分布）"
- `@Description` 必须写明：
  1. 默认时间窗（lifetime / current month / last 7 days）；
  2. 参数依赖（如 `group_by` 的依赖关系）；
  3. 聚合口径（"avg_power = AC/DC 各自的会话内最大功率的中位数"）。
- `@Param` 描述必须写出该参数缺省时的行为。
- `@Failure` 必须列出所有可能返回的非 200 状态码及其 error code。

---

## 7. 实施前的自检清单（PR 自审用）

提交 PR 之前对照本清单逐条打钩：

- [ ] 新增/修改的字段都能回答 "用这个数做什么决策"
- [ ] 没有引入新的算术平均（除非数据特性允许，并在 swagger 中说明理由）
- [ ] 任何接收的查询参数都能影响响应；存在依赖关系的已加 400 校验
- [ ] 默认值裸调能拿到非空响应
- [ ] `Currency` 与 `Timezone` 不出现硬编码
- [ ] 破坏性变更走了 §3.1 两步流程
- [ ] 测试覆盖了 §4.1 的全部维度
- [ ] swagger 按 §6 写了 Summary / Description / Param 缺省 / Failure
- [ ] `meta.applied_filters` 已填充
- [ ] 若新增端点：已对照 §1.3 的三条标准说明 "v2 提供了 v1 没有的什么"
- [ ] `/v2/capabilities` 同步更新（或确认走自动同步）

---

## 8. 与审计文档的对照

| 审计文档章节 | 本规范对应 § | 实施阶段 |
|---|---|---|
| §1.1 tire-pressure avg | §1.1、§2.3 | A（时区）/ B（min/max） |
| §1.2 battery timeseries | §1.2 | C |
| §1.3 v2 battery 重复 | §1.3、§3.2 | C |
| §1.4 charging group_by | §2.1 | A |
| §1.5 charging curve | §2.2 | A |
| §2.1 误导均值 | §1.1 | B |
| §2.2 v1↔v2 重复 | §1.3 | C |
| §2.4 时区/货币 | §2.3 | A |
| §2.5 timeline/places 校验 | §2.1 | D |
| §2.6 capabilities 漂移 | §3.3 | A |

后续 PR 按这个对照表分阶段提交，每条 PR 描述里写明 "本 PR 落地审计文档 §X.Y，对应规范 §A.B"。
