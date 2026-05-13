#!/usr/bin/env python3
"""Translate swagger annotation phrases to Chinese."""
import os, glob

SUMMARY_MAP = {
    'Get battery health for one car': '获取车辆电池健康度',
    'List TeslaMate cars or get one car': '车辆列表或单车详情',
    'List charging sessions for one car': '车辆充电会话列表',
    'Get active charging session if any': '当前进行中的充电会话',
    'Get tire pressure for one car': '车辆胎压（最近读数 + 窗口）',
    'List drives for one car': '车辆行程列表',
    'Get one drive by ID': '单条行程详情',
    'Get one charging session by ID': '单条充电会话详情',
    'Get TeslaMate global settings': 'TeslaMate 全局设置',
    'List firmware updates for one car': '车辆 OTA 更新历史',
    'V2 battery capacity by mileage': 'V2 电池容量随里程趋势',
    'V2 DC charging curve (aggregate)': 'V2 直流充电曲线（聚合）',
    'V2 environmental analytics': 'V2 环境分析（温度/HVAC/海拔）',
    'V2 API capabilities': 'V2 API 能力声明',
    'V2 period summary analytics': 'V2 周期汇总分析',
    'V2 driving analytics summary': 'V2 行驶分析汇总',
    'V2 driving analytics timeseries': 'V2 行驶分析时序',
    'V2 charging analytics summary': 'V2 充电分析汇总',
    'V2 parking analytics summary': 'V2 驻车分析汇总',
    'V2 battery analytics summary': 'V2 电池分析汇总',
    'V2 battery analytics timeseries': 'V2 电池分析时序',
    'V2 cost analytics': 'V2 费用分析',
    'V2 update analytics': 'V2 OTA 更新分析',
    'V2 lifetime cumulative analytics': 'V2 累计生命周期统计',
    'V2 unified event timeline': 'V2 统一事件时间线',
    'V2 efficiency analytics': 'V2 能效分析',
    'V2 cumulative odometer series': 'V2 累计里程序列',
    'V2 flat per-period summary': 'V2 周期扁平汇总',
    'V2 lifecycle places (city / state / country breakdown)': 'V2 生命周期地点分布（城市/州/国家）',
    'V2 list of geofences with billing rules': 'V2 围栏列表（含计费规则）',
    'V2 parking idle periods': 'V2 驻车闲置区间',
}

DESC_MAP = {
    'Returns cars registered in TeslaMate. Omit CarID to list all cars; include CarID for a single car.':
        '返回 TeslaMate 中登记的车辆。省略 CarID 返回全部车辆，传入 CarID 返回指定车辆。',
    'Latest TPMS reading + window min/max + per-day history. Pressure unit follows settings.unit_of_pressure (bar default; psi via barToPsi).':
        '最近一次 TPMS 读数 + 时间窗内极值 + 按天历史。气压单位遵循 settings.unit_of_pressure（默认 bar；psi 由 barToPsi 换算）。',
    'Returns half-month bucketed median capacity (kWh) vs odometer for one car.':
        '按半月分桶返回该车的中位电池容量 (kWh) 与里程关系。',
    'Returns aggregated DC charging curve samples (per battery level: session count, median / p25 / p75 power) for one car.':
        '返回该车直流充电曲线的聚合样本（按电量分桶：会话数、功率中位数 / P25 / P75）。',
    'Returns aggregated outside/inside temperature, HVAC active minutes, and elevation statistics for one car.':
        '返回该车的车外/车内温度聚合、HVAC 活动时长、海拔统计。',
    'Returns the V2 analytics API version, the per-domain feature flags (compare/timeseries/breakdown), and the allowed breakdown values per domain. Clients use this to decide which query parameters to send instead of probing each endpoint.':
        '返回 V2 分析接口版本、各子域能力开关（compare/timeseries/breakdown）、各子域允许的 breakdown 取值。客户端据此决定查询参数，免去逐个接口探测。',
    'Returns objective driving, charging, parking, battery, update, and charging-cost summary metrics for one car in a selected period.':
        '返回该车在所选周期内的行驶、充电、驻车、电池、OTA、充电费用汇总指标。',
    'Returns objective driving statistics for one car.':
        '返回该车的客观行驶统计。',
    'Returns driving metrics grouped by day, week, month, or year for charting.':
        '返回按天/周/月/年分组的行驶指标，用于图表。',
    'Returns objective charging statistics for one car.':
        '返回该车的客观充电统计。',
    'Returns objective parked duration, state duration, inferred parking sessions, and estimated parking drain for one car.':
        '返回该车的驻车时长、状态时长、推断驻车会话与估算驻车电量损失。',
    'Returns objective latest battery range samples, estimated full-range values, baseline range, and estimated range degradation. These estimates are not official state of health.':
        '返回该车最近的额定/理想续航采样、估算满电续航、基线续航、估算续航衰减。注意：此为估算，非官方 SOH。',
    'Returns estimated full rated and ideal range grouped by day, week, month, or year for trend charts.':
        '返回按天/周/月/年分组的估算满电额定/理想续航，用于趋势图。',
    'Returns objective charging-cost analytics. Current data scope includes charging_cost only and excludes insurance, maintenance, parking, depreciation, tire, and repair costs.':
        '返回客观的充电费用分析。当前数据范围仅含 charging_cost，不含保险、保养、停车、折旧、轮胎、维修费用。',
    'Returns OTA update history statistics for a car in the selected period.':
        '返回所选周期内该车的 OTA 更新历史统计。',
    'Returns cumulative lifetime statistics for a car from the first recorded event up to as_of (defaults to now). Use as_of for historical snapshots.':
        '返回该车从首次事件至 as_of（默认当前时间）的累计生命周期统计；可通过 as_of 取历史快照。',
    'Returns a cursor-paginated unified chronological timeline of drive, charging, and update events.':
        '返回行驶/充电/OTA 事件的统一时间线，支持游标分页。',
    'Returns net + gross consumption, consumption overhead, plus optional temperature / speed buckets for one car.':
        '返回该车的净能耗、毛能耗、能耗开销，以及可选的温度/速度分桶。',
    'Returns one odometer reading per day in the requested window for one car.':
        '返回该车在所选窗口内每日一条里程读数。',
    'Returns drive + charging + cost metrics flattened per period (one row per period).':
        '返回按周期扁平化的行驶 + 充电 + 费用指标（每周期一行）。',
    'Returns the top-N most visited cities, states and countries for one car, with last-visited timestamps.':
        '返回该车访问最多的前 N 个城市/州/国家，含最后访问时间。',
    'Returns all geofences with their location, radius, billing type, cost per unit, and per-session fee.':
        '返回全部围栏：位置、半径、计费类型、单位费用、每次会话固定费用。',
}

PARAM_MAP = {
    '"Car ID"': '"车辆 ID"',
    '"Car ID (omit for list)"': '"车辆 ID（不传则返回全部车辆）"',
    '"Charge ID"': '"充电会话 ID"',
    '"Drive ID"': '"行程 ID"',
    '"Result page (default 1)"': '"结果页码（默认 1）"',
    '"Page size (default 100)"': '"每页大小（默认 100）"',
    '"Filter start date"': '"筛选起始日期"',
    '"Filter end date"': '"筛选结束日期"',
    '"History window length in days (default 30, max 365)"': '"历史窗口天数（默认 30，最大 365）"',
    '"Result page"': '"结果页码"',
    '"Page size"': '"每页大小"',
    '"Aggregation period"': '"聚合周期"',
    '"Start datetime in RFC3339 format"': '"起始时间（RFC3339）"',
    '"End datetime in RFC3339 format"': '"结束时间（RFC3339）"',
    '"IANA timezone"': '"IANA 时区"',
    '"Drive details downsampling: full, every_5s (default), every_30s"': '"行程明细下采样：full、every_5s（默认）、every_30s"',
    '"Set to false to skip drive_details (route) in the response"': '"设为 false 可在响应中省略 drive_details（行程轨迹）"',
    '"Minimum sessions per battery level (default 5)"': '"每个电量分桶的最少会话数（默认 5）"',
    '"Comma-separated extras: timeseries"': '"逗号分隔的扩展项，如 timeseries"',
    '"Comma-separated breakdowns: location, type"': '"逗号分隔的分项：location、type"',
    '"Comma-separated breakdowns: location, state, geofence"': '"逗号分隔的分项：location、state、geofence"',
    '"Comma-separated breakdowns: location, level"': '"逗号分隔的分项：location、level"',
    '"Group by interval"': '"分组粒度"',
    '"Snapshot timestamp (RFC3339); defaults to now"': '"快照时间（RFC3339），默认当前时间"',
    '"Cursor for pagination"': '"分页游标"',
    '"Cursor"': '"分页游标"',
    '"Limit"': '"返回条数上限"',
    '"Limit (default 500, max 1000)"': '"返回条数上限（默认 500，最大 1000）"',
    '"Limit (default 20, max 100)"': '"返回条数上限（默认 20，最大 100）"',
    '"Comma-separated event types"': '"逗号分隔的事件类型"',
    '"Filter by event type"': '"按事件类型过滤"',
    '"Filter by event types (comma-separated): drive, charge, update"': '"按事件类型过滤（逗号分隔）：drive、charge、update"',
    '"Filter by domain"': '"按子域过滤"',
    '"Cursor token"': '"分页游标 token"',
    '"Bucket size in days (default 14)"': '"分桶大小（天，默认 14）"',
    '"Minimum samples per bucket (default 3)"': '"每个分桶的最少样本数（默认 3）"',
    '"Filter by minimum power (kW)"': '"按最低功率过滤 (kW)"',
}

FAILURE_MAP = {
    '"Bad Request"': '"请求参数错误"',
    '"Not Found"': '"资源不存在"',
    '"Internal Server Error"': '"服务器内部错误"',
    '"OK"': '"成功"',
    '"Success"': '"成功"',
    '"Successful response"': '"成功"',
    '"Service Unavailable"': '"服务不可用"',
}

def translate_file(path):
    with open(path, 'r', encoding='utf-8') as f:
        s = f.read()
    orig = s
    for en, zh in SUMMARY_MAP.items():
        s = s.replace(f'@Summary {en}', f'@Summary {zh}')
        s = s.replace(f'@Summary  {en}', f'@Summary  {zh}')
    for en, zh in DESC_MAP.items():
        s = s.replace(f'@Description {en}', f'@Description {zh}')
        s = s.replace(f'@Description  {en}', f'@Description  {zh}')
    for en, zh in PARAM_MAP.items():
        s = s.replace(en, zh)
    for en, zh in FAILURE_MAP.items():
        s = s.replace(en, zh)
    if s != orig:
        with open(path, 'w', encoding='utf-8') as f:
            f.write(s)
        return True
    return False

if __name__ == '__main__':
    files = sorted(glob.glob('src/v1_*.go') + glob.glob('src/v2_*.go') + glob.glob('src/swagger*.go'))
    n = 0
    for fp in files:
        if translate_file(fp):
            n += 1
            print('translated:', fp)
    print(f'{n} files updated')
