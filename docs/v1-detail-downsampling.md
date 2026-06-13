# V1 详情接口降采样客户端适配说明

本文说明 V1 重型详情接口的服务端降采样行为。响应 envelope 和字段名不变，客户端主要需要适配查询参数，以及不要再假设明细数组一定是全量、等间隔原始点。

## 影响接口

| 接口 | 明细数组 | 默认行为 |
| --- | --- | --- |
| `GET /api/v1/cars/{CarID}/drives/{DriveID}` | `data.drive.drive_details` | 默认 `sample=auto`，保留路线边界点和关键遥测点。 |
| `GET /api/v1/cars/{CarID}/charges/{ChargeID}` | `data.charge.charge_details` | 默认 `sample=auto`，用于减少大充电会话的响应体积。 |

顶层 summary 字段仍然来自 TeslaMate 聚合/会话数据，不依赖降采样后的明细数组。距离、时长、能耗、费用、起止电量、最大速度、最大/最小功率、充电总量等准确性不受影响。

## 查询参数

### `sample`

| 值 | 含义 | 推荐客户端场景 |
| --- | --- | --- |
| `auto` | 原始点数不超过 `max_points` 时返回全量；超过时保留时间桶首尾、关键极值和状态变化点。行程详情还会保留起终点和经纬度边界点。 | 默认值；推荐图表、移动端和常规详情页使用。 |
| `full` | 返回所有原始明细行。 | 导出、调试，或明确需要原始数据的客户端。 |
| `every_5s` | 固定时间桶采样，每 5 秒桶保留第一行；行程详情会额外保留起终点和经纬度边界点。 | 既有行程详情客户端；兼容模式。 |
| `every_30s` | 固定时间桶采样，每 30 秒桶保留第一行；行程详情会额外保留起终点和经纬度边界点。 | 非常轻量的轨迹/曲线预览。 |

行程详情和充电详情默认 `auto`。显式传 `every_5s` / `every_30s` 的旧调用仍可继续使用固定时间桶模式。

### `max_points`

`max_points` 控制 `sample=auto` 的目标明细密度。

- 默认值：`800`
- 最小接受值：`100`
- 最大接受值：`3000`
- 这是目标值，不是硬上限。状态变化点、关键极值，以及行程路线边界点会强制保留，因此特殊会话的返回数量可能超过 `max_points`。

### `include_route` 和 `include_details`

| 接口 | 参数 | 效果 |
| --- | --- | --- |
| 行程详情 | `include_route=false` 或 `include_route=0` | 省略 `drive_details`，只返回行程元数据和 summary 字段。 |
| 充电详情 | `include_details=false` 或 `include_details=0` | 省略 `charge_details`，只返回充电元数据和 summary 字段。 |

列表行、卡片、摘要页、首页 dashboard 等不需要绘制曲线的场景，建议使用这些参数。

## 调用示例

移动端充电曲线：

```http
GET /api/v1/cars/1/charges/123?sample=auto&max_points=800
```

只取充电元数据：

```http
GET /api/v1/cars/1/charges/123?include_details=false
```

行程详情使用默认 auto 采样：

```http
GET /api/v1/cars/1/drives/456
```

行程详情使用更高目标点数：

```http
GET /api/v1/cars/1/drives/456?sample=auto&max_points=1200
```

导出或调试时取原始全量明细：

```http
GET /api/v1/cars/1/drives/456?sample=full
GET /api/v1/cars/1/charges/123?sample=full
```

## 客户端适配建议

- 除非显式传 `sample=full`，否则把明细数组视为“展示样本”，不要用于精确总量计算。
- 总量和准确性敏感计算使用顶层 summary 字段。
- 图表、轨迹预览、充电曲线优先使用 `sample=auto`。
- 列表、卡片、摘要页优先使用 `include_route=false` 或 `include_details=false`。
- 不要假设明细数组是固定间隔。`auto` 会保留关键点和状态变化，点间隔可能不均匀。

## Swagger

生成的 Swagger/OpenAPI 文件已经包含新参数：

- `internal/docs/generated/swagger.json`
- `internal/docs/generated/swagger.yaml`
