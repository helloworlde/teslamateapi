# 统计/Summary 数据规划

参考第三方 Tesla 数据 App 的「统计 / 能耗分析 / 电池健康」界面，结合 TeslaMate 底层数据，规划新增的 summary 能力。

## 核心原则

1. **客观数据优先**：只承诺 TeslaMate 直接采集、可复现的事实。主观/估算数据明确标注，单独成项，不与客观数据混在同一指标里。
2. **不影响性能**：聚合一律走 `drives` / `charging_processes`（每会话一行）等小表，**禁止为 summary 全表扫描 `positions`**（百万级逐点表）。范围 JOIN 只允许对小表（`updates`、`cars`）。
3. **加法式演进**：新端点不动既有响应结构，沿用现有「单次往返 + CTE」风格。

## 数据分类

### 客观数据（直接采集，可复现）— 优先实现

| 指标 | 来源字段 | 备注 |
|---|---|---|
| 行驶里程 / 行程次数 | `drives.distance`, COUNT | 已有 (lifetime) |
| 能效 Wh/km | rated_range 降幅 × efficiency / distance | 已有，全库统一口径 |
| 充电费用 / 次数 / 电量 / 均价 | `charging_processes.cost/charge_energy_added` | 已有 (lifetime) |
| 平均速度 / 行驶时长 | `drives.speed_avg`, duration | 已有 |
| 各温度区间能耗 | `drives.outside_temp_avg` + 能效 | **新增** (consumption) |
| 各固件版本能耗 | `drives` × `updates` 版本窗口 | **新增** (consumption) |
| 各季节能耗 | `drives.start_date` 月份 → 季节 + 能效 | **新增**，季节按北半球月份约定（半客观） |
| 电池容量/续航/健康度 | `charging_processes` rated_range | 已有 (battery_health) |
| 续航达成率 | `drives.distance` / 额定续航降幅 | **待实现**，口径已定义，见下「计算口径」 |

### 主观/估算数据（依赖假设、阈值或约定）— 单独标注，延后

| 指标 | 为什么主观 | 处置 |
|---|---|---|
| 节省 vs 油车 | 依赖油价、油耗假设 | 走 config 配置项，响应中标注 assumptions |
| 急刹频次 / 百公里 | 阈值定义 + `positions` 采样粗、需扫大表 | **暂不做**（性能 + 主观双重否决） |
| 预估用车费用 | 用全局均摊充电费率套到单次行程（均摊假设） | **待实现**，口径已定义，见下「计算口径」；依赖用户已配置电价 |

### 不可获取（TeslaMate 无此数据源）— 明确放弃

| 指标 | 原因 |
|---|---|
| 电芯电压 / 压差 | 需 BMS/CAN 数据（Scan My Tesla 类硬件），标准 TeslaMate 不采集 |
| 电池模块温度 | 同上。注：参考 App 该页自身标注「演示数据」 |
| 海拔/总爬升 | `positions.elevation` 虽有，但聚合需全表扫描 positions，违反性能原则；延后到专用预聚合方案 |

## 计算口径

### 续航达成率（Range Achievement Rate）— 客观

衡量「实际开出的里程」相对「额定续航掉的里程」的达成程度。

```
续航达成率 = 行程距离 / 续航减少距离 × 100%
```

- **行程距离** = `drives.distance`（km）
- **续航减少距离** = `drives.start_rated_range_km − drives.end_rated_range_km`（额定续航降幅，km）
- 100% 表示开 1km 恰好掉 1km 额定续航；>100% 比额定更省（如再生回收、缓行），<100% 比额定更费（高速、低温、激烈驾驶）。
- 聚合口径用**先求和再相除**，避免单次行程被等权平均放大噪声：
  `SUM(distance) / NULLIF(SUM(GREATEST(start_rated_range_km − end_rated_range_km, 0)), 0) × 100`
- 过滤：`end_date IS NOT NULL`、`distance > 0`、两个 rated_range 均非空且降幅 > 0（排除充电中/数据缺失/异常负降幅）。
- 纯客观、可复现，只扫 `drives`，无 positions 扫描。

### 预估用车费用（Estimated Usage Cost）— 半客观（均摊假设）

把历史总充电费用按总行驶里程均摊成「每公里费率」，再乘以本次行程距离。

```
每公里费率   = 总充电费用 / 总可行驶距离
预估用车费用 = 每公里费率 × 此次行程距离
```

- **总充电费用** = `SUM(charging_processes.cost)`（依赖用户在 TeslaMate 中配置了电价，否则 cost 为空，费率为 0）
- **总可行驶距离** = `SUM(drives.distance)`（lifetime 口径）
- **此次行程距离** = 目标行程的 `drives.distance`
- 半客观：用**全局平均费率**套到单次行程，隐含「每次行程成本结构相同」的均摊假设；响应需标注该 assumption，且与「充电费用」客观字段分开呈现，不混淆。
- 只扫 `drives` / `charging_processes`，无 positions 扫描。

## 实施阶段

### Phase 2（本次执行）：`GET /api/v2/cars/{CarID}/stats/consumption`

能耗分析三个 tab 合一，纯客观数据，只扫 `drives`（+小表 `updates`）。

参数：`group_by = temperature | version | season`（默认 temperature）

响应（`data`）：
- `car`, `group_by`, `overall_consumption`（全局 Wh/km）, `units`
- `groups[]`：每组 `key` / `consumption`（Wh/km）/ `trips_count` / `distance` / `energy_kwh` / `delta_vs_avg_pct`（客观算术）
- temperature 组额外带 `temp_low` / `temp_high`（按用户单位换算）

性能：单 SQL、GROUP BY 在 `drives`；version 用 `updates` 的 `LEAD` 窗口建版本区间再范围 JOIN（updates 仅几十行）。无 positions 扫描。

### Phase 1（已完成）：扩展 battery_health

- ✅ battery_health 增加 `predicted_range`（current_range × 当前 SOC，客观）与 `current_battery_level`
- lifetime 视需要补充客观字段（暂无新增需求）

### Phase 3（已完成）：`GET /api/v2/cars/{CarID}/stats/behavior`

行为习惯画像，三段客观分布合一，单端点返回，只扫 `drives` / `charging_processes`，无 positions 扫描。

- ✅ 时段热力图 `heatmap`（周几×小时，基于 `start_date`，用户时区；drives 计数+里程，charges 计数）
- ✅ 充电电量分布 `charge_levels`（`start/end_battery_level` 10% 直方图，0~90 区间，100 归入 90~100）
- ✅ 行程类型分布 `trip_types`（按 distance 分桶 0/5/20/50/100/200km，trips/距离/能耗，距离按单位换算）

### 已完成（口径见「计算口径」）

- ✅ 续航达成率
  - 单次行程：v1 `GET /cars/{CarID}/drives` 及 `/drives/{DriveID}` 每条 drive 的 `range_achievement_pct`（额定降幅 ≤0 时为 null）
  - 全局聚合：v2 lifetime `drives.range_achievement_pct`（Σ距离 / Σ额定降幅 × 100）
- ✅ 预估用车费用
  - 单次行程：v1 drives 列表/详情每条 drive 的 `estimated_usage_cost`（= 全局每公里费率 × 本次距离）
  - 均摊费率：v2 lifetime `charges.cost_per_distance`（总充电费用 / 总行驶距离，按单位换算 per km/mile）

### 延后/受阻

- 节省 vs 油车（需配置项 + assumptions 标注）
- 海拔/爬升、急刹（需 positions 预聚合或物化视图，避免实时全表扫描）
