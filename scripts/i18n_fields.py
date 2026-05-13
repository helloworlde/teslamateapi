#!/usr/bin/env python3
"""Append chinese trailing comments to struct fields based on json key.

Rules:
  - Only modify lines that match a json:"key"... pattern.
  - Skip lines that already have a `//` trailing comment.
  - Only modify struct fields inside files under src/ that participate in swagger generation.
"""
import os, re, glob

# json_key -> chinese description
# All keys observed in src/v2_* and src/v1_swagger_models.go.
DICT = {
    # envelope / common
    'data': '响应数据',
    'meta': '响应元信息',
    'items': '条目列表',
    'has_more': '是否还有更多',
    'next_cursor': '下一页游标',
    'cursor': '分页游标',
    'limit': '返回条数上限',
    'offset': '偏移量',
    'page': '当前页',
    'count': '总数',
    'page_size': '每页大小',
    'request_id': '请求 ID',
    'time_range': '时间范围',
    'timezone': '时区',
    'period': '聚合周期',
    'group_by': '聚合粒度',
    'compare': '对比模式',
    'breakdown': '分项维度',
    'previous_period': '对比前一周期',
    'as_of': '快照时间',
    'start': '起始时间',
    'end': '结束时间',
    'interval': '区间秒数',
    'unit': '单位',
    'units': '单位',
    'unit_of_length': '长度单位',
    'unit_of_pressure': '气压单位',
    'unit_of_energy': '能量单位',
    'unit_of_temperature': '温度单位',

    # error
    'code': '错误码',
    'message': '错误描述',
    'error': '错误体',
    'details': '错误详情',

    # car / cars
    'car': '车辆',
    'cars': '车辆列表',
    'car_id': '车辆 ID',
    'name': '名称',
    'short_version': '短版本号',
    'version': '版本',
    'versions': '版本列表',
    'vehicle': '车辆信息',

    # drive
    'drive_id': '行程 ID',
    'drive_count': '行程数',
    'distance': '距离 (km)',
    'duration': '时长 (秒)',
    'duration_min': '时长 (分)',
    'avg_speed': '平均速度 (km/h)',
    'avg_consumption': '平均能耗 (Wh/km)',
    'best_consumption': '最佳能耗 (Wh/km)',
    'worst_consumption': '最差能耗 (Wh/km)',
    'net_energy': '净能量 (kWh)',
    'gross_energy': '毛能量 (kWh)',
    'gross_consumption': '毛能耗 (Wh/km)',
    'consumption_overhead': '能耗开销 (Wh/km)',
    'driving': '行驶指标',
    'driving_duration': '行驶时长 (秒)',
    'avg_distance': '平均距离 (km)',
    'avg_duration': '平均时长 (秒)',
    'trip_count': '行程数',
    'route': '行程轨迹',

    # charging
    'charge_id': '充电会话 ID',
    'session_count': '充电会话数',
    'charging_sessions': '充电会话数',
    'charging_session_count': '充电会话数',
    'charging': '充电指标',
    'energy_added': '充入电池能量 (kWh)',
    'energy_used': '墙端用电 (kWh)',
    'wall_energy': '墙端用电 (kWh)',
    'battery_energy': '电池端能量 (kWh)',
    'charging_efficiency': '充电效率',
    'efficiency': '效率',
    'charging_duration': '充电时长 (秒)',
    'charge_cost': '充电费用',
    'charging_cost': '充电费用',
    'cost': '费用',
    'cost_per_distance': '单位里程费用',
    'avg_power': '平均功率 (kW)',
    'peak_power': '峰值功率 (kW)',
    'samples': '样本',
    'session': '会话',
    'sessions': '会话列表',
    'min_power_kw': '最低功率 (kW)',
    'median_power_kw': '功率中位数 (kW)',
    'p25_power_kw': '功率 P25 (kW)',
    'p75_power_kw': '功率 P75 (kW)',
    'battery_level': '电量百分比',
    'usable_battery_level': '可用电量百分比',
    'starting_battery_level': '起始电量',
    'ending_battery_level': '结束电量',
    'start_battery_level': '起始电量',
    'end_battery_level': '结束电量',

    # battery
    'battery': '电池指标',
    'rated_range': '额定续航 (km)',
    'ideal_range': '理想续航 (km)',
    'latest_rated_range': '最近额定续航 (km)',
    'latest_ideal_range': '最近理想续航 (km)',
    'estimated_full_range': '估算满电额定续航 (km)',
    'estimated_full_ideal_range': '估算满电理想续航 (km)',
    'baseline_range': '基线续航 (km)',
    'range_degradation': '续航衰减比例',
    'range_at_full_charge': '满电续航 (km)',
    'range_loss': '续航损失 (km)',
    'range_loss_per_hour': '每小时续航损失 (km/h)',
    'capacity_kwh': '电池容量 (kWh)',
    'samples_count': '样本数',
    'odometer_km': '里程 (km)',
    'bucket_start_km': '分桶起始里程 (km)',
    'bucket_end_km': '分桶结束里程 (km)',
    'soc_diff': 'SoC 差值',
    'has_reduced_range': '是否处于受限续航状态',

    # state / parking
    'state': '状态',
    'state_transition_count': '状态切换次数',
    'state_durations': '状态时长',
    'parked_duration': '驻车时长 (秒)',
    'parking_session_count': '驻车会话数',
    'avg_parked_duration': '平均驻车时长 (秒)',
    'inactive_duration': '不活跃时长 (秒)',
    'standby_seconds': '待机时长 (秒)',
    'standby_ratio': '待机占比',
    'energy_drained': '消耗能量 (kWh)',
    'avg_power_w': '平均功率 (W)',
    'vampire_drain_percent': '吸血式漏电比例 (%)',

    # location / address / geofence
    'address': '地址',
    'address_id': '地址 ID',
    'location': '地点',
    'location_name': '地点名称',
    'locations': '地点列表',
    'city': '城市',
    'state_name': '州/省',
    'country': '国家',
    'cities': '城市分布',
    'states': '州/省分布',
    'countries': '国家分布',
    'visit_count': '访问次数',
    'last_visited': '最后访问时间',
    'top_n': 'Top N 配置',
    'geofence': '围栏',
    'geofence_id': '围栏 ID',
    'geofences': '围栏列表',
    'latitude': '纬度',
    'longitude': '经度',
    'radius': '半径 (米)',
    'cost_per_unit': '单位费用',
    'billing_type': '计费类型',
    'session_fee': '每次会话固定费用',

    # update / OTA
    'update_count': 'OTA 更新次数',
    'latest_version': '最近版本',
    'latest_updated_at': '最近更新时间',
    'avg_update_duration': '平均更新时长 (秒)',
    'median_days_between_updates': '更新间隔中位数 (天)',
    'days_since_prior': '距上次更新天数',
    'started_at': '开始时间',
    'completed_at': '完成时间',
    'window': '观察窗口',
    'event': '事件',
    'metrics': '指标',

    # timeline / events
    'type': '类型',
    'id': 'ID',
    'occurred_at': '发生时间',
    'kind': '类别',
    'before': '游标（早于）',
    'after': '游标（晚于）',
    'event_id': '事件 ID',
    'events': '事件列表',
    'cumulative': '累计',
    'lifetime': '生命周期累计',

    # environmental
    'outside_temp_min': '车外最低温 (°C)',
    'outside_temp_max': '车外最高温 (°C)',
    'outside_temp_avg': '车外平均温 (°C)',
    'inside_temp_avg': '车内平均温 (°C)',
    'climate_on_minutes': '空调开启时长 (分)',
    'battery_heater_minutes': '电池加热器时长 (分)',
    'defroster_minutes': '除霜时长 (分)',
    'elevation_min': '最低海拔 (米)',
    'elevation_max': '最高海拔 (米)',
    'elevation_gain_total': '累计爬升 (米)',
    'elevation_loss_total': '累计下降 (米)',
    'timeseries': '时序',
    'timezones': '时区',

    # tire pressure
    'fl': '左前胎压',
    'fr': '右前胎压',
    'rl': '左后胎压',
    'rr': '右后胎压',
    'min': '最小值',
    'max': '最大值',
    'history': '历史记录',
    'latest': '最近值',

    # period summary
    'period_start': '周期起始',
    'period_end': '周期结束',
    'data_complete': '数据是否完整',
    'flat': '扁平指标',

    # active charging
    'active_session': '当前会话',
    'is_charging': '是否正在充电',

    # capabilities
    'capabilities': '能力',
    'features': '特性',
    'allowed_breakdown': '允许的分项取值',

    # ranges / tags
    'tags': '标签',
    'available': '是否可用',
    'flag': '开关',
    'enabled': '是否启用',
}

FIELD_RE = re.compile(r'^(\s*)([A-Z]\w*)(\s+[\*\[\]\w\.]+)\s+`([^`]*json:"([^",]+)[^"]*"[^`]*)`(\s*)(.*)$')

def patch_file(path):
    with open(path, 'r', encoding='utf-8') as f:
        lines = f.readlines()
    changed = False
    for i, line in enumerate(lines):
        m = FIELD_RE.match(line.rstrip('\n'))
        if not m:
            continue
        indent, field_name, type_part, tag, json_key, after_tag, trailing = m.groups()
        # Skip if already has trailing //
        if '//' in trailing:
            continue
        if json_key not in DICT:
            continue
        zh = DICT[json_key]
        new_line = f'{indent}{field_name}{type_part} `{tag}` // {zh}\n'
        lines[i] = new_line
        changed = True
    if changed:
        with open(path, 'w', encoding='utf-8') as f:
            f.writelines(lines)
        return True
    return False

if __name__ == '__main__':
    files = sorted(set(glob.glob('src/v1_*.go') + glob.glob('src/v2_*.go')))
    n = 0
    for fp in files:
        if patch_file(fp):
            n += 1
            print('annotated:', fp)
    print(f'{n} files updated')
