# V2 数据缺口与优化建议

研究 TeslaMate Postgres 模式（`priv/repo/migrations`）和官方 Grafana 仪表板（`grafana/dashboards`）后整理出的数据缺口清单。每条 `[GAP]` 给出列名 / SQL 提示，可直接落到 v2 接口。

## 范围说明
- 数据源：`teslamate-org/teslamate` master 分支。
- 现有 v2 已覆盖：driving / charging / parking / battery / updates / lifecycle / timeline / cost / summary。
- 文档目标：列出 schema 中尚未利用的字段与 Grafana 中已实现但 v2 缺失的派生指标。

---

## 1. Schema 层缺口

### 1.1 `cars` / `car_settings`
| 字段 | 现状 | 建议 |
|---|---|---|
| `marketing_name`, `model`, `vin`, `exterior_color`, `wheel_type`, `spoiler_type`, `efficiency` | 未暴露 | `[GAP]` 新增 `/v2/vehicles/{id}` 车辆画像（含 EPA 效率、配置、显示名） |
| `display_priority` | 未暴露 | `[GAP]` 多车 UI 排序需要 |
| `car_settings.lfp_battery` | 未暴露 | `[GAP]` LFP 与 NCA/NCM 在 100% / 80% 阈值上行为不同；电池健康/range 计算需暴露 |
| `car_settings.free_supercharging` | 未暴露 | `[GAP]` 成本统计应可排除 Tesla SC |
| `car_settings.enabled` | 未暴露 | `[GAP]` 多车列表过滤 |
| `settings.unit_of_length / temperature / pressure / language / base_url / theme_mode` | 未暴露 | `[GAP]` `/v2/settings` 让客户端复用 TeslaMate 用户偏好 |

### 1.2 `drives`
| 字段 | 现状 | 建议 |
|---|---|---|
| `inside_temp_avg` | 未用 | `[GAP]` driving summary 与 timeseries 增加车内平均温度 |
| `ascent`, `descent` | 未用 | `[GAP]` 驾驶能耗的关键变量；纳入 summary、per-drive 列表 |
| `start_position_id`, `end_position_id` | 未用 | `[GAP]` 让 timeline / per-drive 端点直接返回起止经纬度，无需二次查询 |
| `start_geofence_id` vs `end_geofence_id` | 仅作为标签 | `[GAP]` 支持「从家出发 / 回到家」过滤 |
| `power_min` | 仅 summary 暴露 peak_regen | `[GAP]` per-drive 列表保留 min vs max power |

### 1.3 `charging_processes` / `charges`
| 字段 | 现状 | 建议 |
|---|---|---|
| `charges.battery_level` × `charger_power` 时序 | 未暴露 | `[GAP]` `/v2/charging/sessions/{id}/curve` 返回 (soc, power, voltage, current, t_offset)，对应 Grafana DC Charging Curve |
| `charge_energy_used_confidence`, `interval_sec` | 未暴露 | `[GAP]` 数据质量字段，客户端可隐藏低置信值 |
| `usable_battery_level` | 未暴露 | `[GAP]` 冷天减容信号；驱动 Grafana ❄️ reduced range 标记 |
| `charger_pilot_current` | 未暴露 | `[GAP]` 家用回路诊断「为什么没有跑满」 |
| `fast_charger_brand`, `fast_charger_type` | 未暴露 | `[GAP]` 仅按 AC/DC 拆分粒度太粗；增加按运营商 / 接口拆分 |
| `battery_heater_on`, `not_enough_power_to_heat` | 未暴露 | `[GAP]` 寒冷地区核心 KPI（预热分钟数、桩功率不足次数） |
| `charging_processes.position_id` | 未暴露 | `[GAP]` 充电会话起点经纬度，渲染地图所需 |

### 1.4 `positions`
| 字段 | 现状 | 建议 |
|---|---|---|
| `inside_temp` | 未暴露 | `[GAP]` 车内温度时序 |
| `outside_temp` 时序 | summary 仅 avg | `[GAP]` 时间桶化温度趋势 |
| `elevation` | 未暴露 | `[GAP]` 配合 drives.ascent/descent 的高程时序 |
| HVAC 簇：`is_climate_on`, `driver_temp_setting`, `passenger_temp_setting`, `fan_status`, `is_rear_defroster_on`, `is_front_defroster_on` | 未暴露 | `[GAP]` 「驾驶时空调开启占比」「设定温度分布」「除霜分钟」 |
| `tpms_pressure_fl/fr/rl/rr` | 未暴露 | `[GAP]` 车辆健康端点，最近一次 + 窗口 min/max |
| `usable_battery_level` (positions) | 未暴露 | `[GAP]` 冷天减容信号 |
| `battery_heater_on` (positions) | 未暴露 | `[GAP]` 预热分钟统计 |
| `power` per-position | 仅 summary | `[GAP]` 真实功率/加速分布、速度直方图 |
| 原始 lat/lng | 未暴露 | `[GAP]` 客户端地图、热力图、行程回放都依赖 |

### 1.5 `addresses` / `geofences`
| 字段 | 现状 | 建议 |
|---|---|---|
| `addresses.city/state/country` | 仅作为单地址标签 | `[GAP]` 新增 city/state/country count + top-N 聚合 |
| `geofences` 全集 + 费率 (`cost_per_unit`, `billing_type`, `session_fee`) | 未暴露 | `[GAP]` `/v2/geofences` 列表，包含计费规则 |
| 地理围栏停留时长 | 未暴露 | `[GAP]` Grafana 都没做；可作为差异化指标 |

### 1.6 `updates`
| 字段 | 现状 | 建议 |
|---|---|---|
| 单版本停留时长 | per-event 已支持 days_since_prior | `[GAP]` 加 `version_active_duration` + 该窗口内 charges/distance |
| 版本短号 (`split_part(version,' ',1)`) | 未暴露 | `[GAP]` 客户端常见诉求 |

---

## 2. Grafana 派生指标缺口

### 2.1 driving
- `[GAP]` **gross consumption**（drive + idle + parasitic）：用 `lag(end_*_range_km) OVER (ORDER BY start_date)` 串联 charges 与 drives。Grafana `overview.json` / `efficiency.json` 头号面板。
- `[GAP]` **slope-adjusted efficiency**：drives.json 公式 `(distance + (ascent − descent)·9.81·2100 / (efficiency·3600·1000)) / range_diff`，假设 85% regen、2100 kg。
- `[GAP]` **median / p90 / p99** of distance / duration / consumption。`percentile_cont` window。
- `[GAP]` **speed histogram by time spent**：`LEAD(date) OVER (PARTITION BY drive_id)` 求 dt，按 10 km/h 桶累加。
- `[GAP]` **efficiency × outside_temp 桶**：`ROUND(outside_temp_avg / 5) * 5` 分组，输出 `(temp_bucket, drive_count, distance, range_loss, efficiency, consumption, avg_speed)`。
- `[GAP]` **efficiency × speed 桶**：position 级派生，Grafana 没有但呼声高。
- `[GAP]` **top destinations**：`drives.end_address_id` group + 排除列表 (`NOT ILIKE ALL ('%home%', ...)`).
- `[GAP]` **per-drive 列表端点**：geofence 名优先覆盖、has_reduced_range、incomplete、起止 lat/lng、最小距离/速度过滤。
- `[GAP]` **incomplete drives**：`WHERE end_date IS NULL`。
- `[GAP]` **effective speed including DC stops**：分母 = 行驶 + DC 充电时间。
- `[GAP]` **drives 时序零填充**：`generate_series` LEFT JOIN，避免 sparse buckets。

### 2.2 charging
- `[GAP]` **per-session 列表**：含 `cost_per_kwh`、`charging_efficiency = energy_added / GREATEST(energy_used, energy_added)`、`charge_energy_added_per_hour`、AC/DC 标签、odometer 回退过滤。
- `[GAP]` **incomplete sessions** (`end_date IS NULL`)。
- `[GAP]` **charge curve**：`SELECT battery_level, charger_power, percentile_cont(0.5) WITHIN GROUP (ORDER BY charger_power) OVER (PARTITION BY battery_level)` 限定 DC 会话。
- `[GAP]` **start↔end SoC heatmap**（2D 分布）。
- `[GAP]` **charge / discharge SoC 桶 + 连续会话去重**：`lead(activity)` / `lag(activity)` 排除连续同向会话。
- `[GAP]` **Supercharger-only 成本**：`fast_charger_brand = 'Tesla'` AND `charger_phases IS NULL`，结合 `free_supercharging` 排除。
- `[GAP]` **每地点能量 + lat/lng**：现有 location breakdown 没有坐标，无法画地图标注。
- `[GAP]` **live charging state**：`end_date IS NULL` 的最新会话 + 实时 voltage/power/SoC。
- `[GAP]` **odometer regression filter**：`p.odometer - lag(p.odometer) OVER (ORDER BY start_date) >= 0` 过滤数据回退会话。
- `[GAP]` **charging cycles** = `floor(SUM(charge_energy_added) / battery_capacity_when_new)`。

### 2.3 battery
- `[GAP]` **rated × usable 双投影 range**：`V2BatteryRange` 增加 `RatedUsable` / `IdealUsable`。
- `[GAP]` **degradation rate (slope)**：`regr_slope(range_at_full, extract(epoch FROM period_start) / 86400)`。
- `[GAP]` **capacity-based health**：current_capacity_kwh、max_capacity_kwh、battery_health_percent，公式 `AVG(rated_battery_range_km * efficiency / NULLIF(usable_battery_level, 0))`，最近 100 个有效充电样本。
- `[GAP]` **derived efficiency**：`charging_processes` 中 `ROUND(charge_energy_added / (end_rated_range - start_rated_range), 2)` 取 mode；`duration_min > 10 AND end_battery_level <= 95 AND charge_energy_added > 0`。
- `[GAP]` **capacity by mileage**：半月桶 `YYYYMM` + `'1'/'2'`，`PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY capacity)`。
- `[GAP]` **SoC distribution timeseries**：median + p7.5/p92.5。需要 `generate_series` 填空 + LOCF 填充 + percentile 滑窗。
- `[GAP]` **data lost** = `MAX(end_km) - MIN(start_km) - SUM(distance)`。
- `[GAP]` 暴露 `lfp_battery` 标志。

### 2.4 parking
- `[GAP]` **per-idle-gap 列表**：UNION drives + charging_processes 后用 `LAG()` 推空闲段；返回 `(start, end, duration, range_loss, soc_diff, energy_kwh, avg_power_w, range_loss_per_hour, standby_ratio, has_reduced_range)`。
- `[GAP]` **standby_ratio 汇总 KPI**：`LATERAL JOIN states` 求空闲段中 `state IN ('asleep','offline')` 时间占比。
- `[GAP]` **geofence 停留时长**。
- `[GAP]` **状态转移按小时/星期分布**。
- `[GAP]` parking event 的「停车期间消耗」= `(prev.range - next.range) × car.efficiency`。

### 2.5 updates
- `[GAP]` **median 升级间隔**：`percentile_disc(0.5) WITHIN GROUP (ORDER BY epoch_diff)`。
- `[GAP]` **version-active window**：`lag(start_date) OVER (ORDER BY start_date DESC)` 派生 next_start_date，聚合该区间 charges/drives。
- `[GAP]` 短版本号 + release-notes 链接路径。

### 2.6 vehicle / environmental
- `[GAP]` `/v2/vehicles/{id}` 车辆画像。
- `[GAP]` `/v2/settings` 单位偏好。
- `[GAP]` 胎压：最新 + 窗口 min/max + 历史。
- `[GAP]` HVAC 行为：climate_on 占比、setpoint 分布、除霜分钟、fan_status 直方图。
- `[GAP]` 车内/外温度时序、电池预热分钟。
- `[GAP]` `/v2/state/current`：最近一帧位置 + 温度 + 充电状态（live）。

### 2.7 lifecycle / cost
- `[GAP]` city/state/country 数量 + top-N。
- `[GAP]` last-visited per geofence/address。
- `[GAP]` odometer 时序。
- `[GAP]` `/v2/summary/by_period`：drives + charging + cost + Consumption Overhead + Data Complete 一行打平。
- `[GAP]` Consumption Overhead = `(gross - net) / gross`。
- `[GAP]` Data Complete 标志：`bool_or(end_date IS NULL)` over drives+charges。
- `[GAP]` Supercharger-only cost（含 free_supercharging 排除）。

### 2.8 timeline
- `[GAP]` `missing` 事件类型：`TP2.odometer - TP1.odometer > 0.5` 且起止 address 不一致。
- `[GAP]` parking event 携带停车期间消耗。

---

## 3. SQL 工具与索引

迁移已注册的辅助函数 / 扩展可在新端点中复用：
- `convert_km(numeric, text)`、`convert_celsius(numeric, text)`、`convert_m(numeric, text)`、`convert_tire_pressure(numeric, text)`
- `cube` + `earthdistance`（`earth_distance(ll_to_earth(lat1,lon1), ll_to_earth(lat2,lon2))`）— Grafana 未使用，可作为聚合 / 最近站差异点。
- `pg_stat_statements`（可选）。

强相关复合索引（写新查询前看一眼避免 seq scan）：
- `(car_id, date)` predicates with `WHERE ideal_battery_range_km IS NOT NULL`（多次迁移定义）。

---

## 4. 优先级建议（个人判断，可调）

1. **vehicle profile + settings**（`/v2/vehicles/{id}`, `/v2/settings`）：成本低，多个客户端立刻受益。
2. **charge curve + per-session listing**：用户可见度最高的功能性缺口。
3. **gross consumption / consumption overhead / efficiency × temperature**：现 v2 已暴露 net consumption，补这层后取代 Grafana `efficiency.json` 主面板。
4. **per-drive listing + slope-adjusted efficiency + has_reduced_range**：driving 端打平 Grafana drives.json。
5. **per-idle-gap listing + standby_ratio**：parking 端打平 vampire-drain.json。
6. **battery health（capacity-based）+ capacity by mileage + cycles**：电池长期价值指标。
7. **HVAC / 胎压 / 高程 / 温度时序**：完善 environmental & vehicle health 维度。
8. **timeline missing + odometer 时序 + by_period summary + city/state 聚合**：长尾完善。

每项实现前建议在 progress 文档登记一条 phase（J / K / L …），保持单次 PR 范围在 1-2 域内，避免重蹈 v2 大改的回归风险。

---

## 5. 参考资料

- Migrations: https://github.com/teslamate-org/teslamate/tree/master/priv/repo/migrations
- Dashboards: https://github.com/teslamate-org/teslamate/tree/master/grafana/dashboards
- 关键面板：`overview.json`, `charges.json`, `charge-level.json`, `charging-stats.json`, `drive-stats.json`, `drives.json`, `efficiency.json`, `projected-range.json`, `battery-health.json`, `vampire-drain.json`, `locations.json`, `statistics.json`, `timeline.json`, `trip.json`, `updates.json`。
