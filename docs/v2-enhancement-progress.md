# V1 增强 + V2 扩展 进度跟踪

> 跟踪 [v2-enhancement-plan.md](v2-enhancement-plan.md) 的执行。
> 每个 Phase 完成条件：build + vet + test 全绿；swagger 重生成；v1 字段为追加且 `omitempty`，不影响既有响应。

## Baseline

- Date: 2026-05-12
- Branch: `claude/musing-meitner-ea407a` (worktree)
- Pre-change build: `go build` ✅
- Pre-change tests: `go test ./src/... -count=1` ✅
- Pre-change vet: `go vet ./src/...` ✅

## Phase Plan

| # | Phase | 类型 | Status |
|---|-------|------|--------|
| J | /v1/cars + /v1/globalsettings 字段补全 | v1 | ✅ done |
| K | /v1/charges* 字段补全 + /v2 charging/curve | v1+v2 | ✅ done |
| L | /v2 analytics/efficiency | v2 | ✅ done |
| M | /v1/drives* 字段补全 + 下采样 | v1 | ✅ done |
| N | /v2 parking/idle_periods | v2 | ✅ done |
| O | /v1/battery-health 补全 + /v2 capacity_by_mileage | v1+v2 | ✅ done |
| P | /v1/tire-pressure + /v2 analytics/environmental | v1+v2 | ✅ done |
| Q | /v1/updates 补全 + v2 长尾 | v1+v2 | ✅ done (timeline `type:"missing"` 合成事件 / park `consumption_during_park` 推迟到后续 PR) |
