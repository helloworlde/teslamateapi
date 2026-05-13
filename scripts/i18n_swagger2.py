#!/usr/bin/env python3
"""Second pass param description translator."""
import glob

EXTRA_PARAM = {
    '"Breakdown dimension (used when include=breakdown)"': '"分项维度（仅在 include=breakdown 时生效）"',
    '"Bucket grouping (used when include=buckets)"': '"分桶维度（仅在 include=buckets 时生效）"',
    '"Comma-separated event types: drive,charging,update"': '"逗号分隔的事件类型：drive、charging、update"',
    '"Comma-separated extras: breakdown"': '"逗号分隔的扩展项：breakdown"',
    '"Comma-separated extras: buckets"': '"逗号分隔的扩展项：buckets"',
    '"Comma-separated extras: timeseries,breakdown"': '"逗号分隔的扩展项：timeseries、breakdown"',
    '"Comparison mode"': '"对比模式"',
    '"Cost grouping"': '"费用聚合粒度"',
    '"Cutoff datetime in RFC3339 format. Defaults to now."': '"截止时间（RFC3339），默认当前时间。"',
    '"Filter by resolved geofence id"': '"按已解析的围栏 ID 过滤"',
    '"Max results per page"': '"每页最大结果数"',
    '"Minimum gap length in hours (default 1)"': '"最小间隔时长（小时，默认 1）"',
    '"Page size (default 100, max 500)"': '"每页大小（默认 100，最大 500）"',
    '"Page size (default 500, max 1000)"': '"每页大小（默认 500，最大 1000）"',
    '"Pagination cursor returned by a previous response"': '"由上一次响应返回的分页游标"',
    '"Period grouping"': '"周期聚合粒度"',
    '"Return events after this RFC3339 timestamp (cursor, ASC order)"': '"返回此 RFC3339 时间之后的事件（升序游标）"',
    '"Return events before this RFC3339 timestamp (cursor, DESC order)"': '"返回此 RFC3339 时间之前的事件（降序游标）"',
    '"Timeseries grouping (used when include=timeseries)"': '"时序聚合粒度（仅在 include=timeseries 时生效）"',
    '"Timeseries grouping"': '"时序聚合粒度"',
    '"Top N per dimension (default 20, max 100)"': '"每个维度的 Top N（默认 20，最大 100）"',
}

def translate(path):
    with open(path) as f: s = f.read()
    orig = s
    for en, zh in EXTRA_PARAM.items():
        s = s.replace(en, zh)
    if s != orig:
        with open(path, 'w') as f: f.write(s)
        return True
    return False

if __name__ == '__main__':
    n = 0
    for fp in sorted(glob.glob('src/v1_*.go') + glob.glob('src/v2_*.go') + glob.glob('src/swagger*.go')):
        if translate(fp):
            n += 1
            print('translated:', fp)
    print(f'{n} files')
