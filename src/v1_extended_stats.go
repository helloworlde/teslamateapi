package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

// TeslaMateAPICarsStatsV2 返回结构清晰且无重复的周期核心指标。
// @Summary 周期统计
// @Description 扩展接口 v2：返回行程、充电、电池和停车的周期核心指标。
// @Tags 扩展 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Param period query string false "week|month|year|custom"
// @Param date query string false "参考日期"
// @Param startDate query string false "自定义范围开始时间"
// @Param endDate query string false "自定义范围结束时间"
// @Success 200 {object} StatsV2Envelope
// @Failure 400,404,500 {object} v1ErrorEnvelope
// @Router /v2/cars/{CarID}/stats [get]
func TeslaMateAPICarsStatsV2(c *gin.Context) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid stats range", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsStatsV2")
	if !ok {
		return
	}
	data, err := buildStats(ctx, dr)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load stats", map[string]any{"reason": err.Error()})
		return
	}
	writeV1Object(c, data, buildV1MetaFromCar(ctx, dr.Timezone.String()))
}

func buildStats(ctx *apiCarContext, dr v1DateRange) (map[string]any, error) {
	startUTC, endUTC := dbTimeRange(dr)

	// 阶段 A：所有独立查询并发执行。
	var (
		drive        *DriveHistorySummary
		charge       *ChargeHistorySummary
		parking      *ParkingHistorySummary
		state        *StateSummary
		battery      = map[string]any{}
		vampireDrain *float64
		odometer     *float64
	)
	gA := new(errgroup.Group)
	gA.Go(func() error {
		v, err := fetchDriveHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
		if err != nil {
			return fmt.Errorf("drives: %w", err)
		}
		drive = v
		return nil
	})
	gA.Go(func() error {
		v, err := fetchChargeHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
		if err != nil {
			return fmt.Errorf("charges: %w", err)
		}
		charge = v
		return nil
	})
	gA.Go(func() error {
		v, err := fetchParkingHistorySummary(ctx.CarID, startUTC, endUTC, nil)
		if err != nil {
			return fmt.Errorf("parking: %w", err)
		}
		parking = v
		return nil
	})
	gA.Go(func() error {
		v, err := fetchStateSummary(ctx.CarID, startUTC, endUTC)
		if err != nil {
			return fmt.Errorf("state: %w", err)
		}
		state = v
		return nil
	})
	gA.Go(func() error {
		v, err := fetchBatterySnapshot(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
		if err == nil {
			battery = v
		}
		return nil
	})
	gA.Go(func() error {
		vampireDrain, _ = fetchParkingEnergyTotal(ctx.CarID, startUTC, endUTC)
		return nil
	})
	gA.Go(func() error {
		odometer, _ = fetchLatestOdometer(ctx.CarID, ctx.UnitsLength)
		return nil
	})
	if err := gA.Wait(); err != nil {
		return nil, err
	}

	// 阶段 B：动能回收依赖阶段 A 的行程汇总。
	var regen *RegenerationSummary
	regen, _ = fetchRegenerationSummary(ctx.CarID, startUTC, endUTC, drive, ctx.UnitsLength)

	return buildStatsResponse(ctx, dr, drive, charge, parking, state, battery, vampireDrain, odometer, regen), nil
}

func buildStatsResponse(
	ctx *apiCarContext,
	dr v1DateRange,
	drive *DriveHistorySummary,
	charge *ChargeHistorySummary,
	parking *ParkingHistorySummary,
	state *StateSummary,
	battery map[string]any,
	vampireDrain *float64,
	odometer *float64,
	regen *RegenerationSummary,
) map[string]any {
	// 行程分区。
	var regenEnergyKwh any
	var regenRatio any
	var regenEstimated bool
	if regen != nil {
		regenEnergyKwh = regen.EstimatedRecoveredEnergyKwh
		regenRatio = regen.RecoveryShare
		regenEstimated = regen.MetricsEstimated
	}
	drivesSection := map[string]any{
		"count":            drive.DriveCount,
		"distance":         drive.TotalDistance,
		"duration_min":     drive.TotalDurationMin,
		"energy_used_kwh":  drive.TotalEnergyConsumed,
		"avg_speed":        drive.AverageSpeed,
		"max_speed":        drive.MaxSpeed,
		"avg_efficiency":   drive.AverageConsumption,
		"best_efficiency":  drive.BestConsumption,
		"worst_efficiency": drive.WorstConsumption,
		"peak_power_kw":    drive.PeakDrivePower,
		"regen_energy_kwh": regenEnergyKwh,
		"regen_ratio":      regenRatio,
		"regen_estimated":  regenEstimated,
	}

	// 充电分区。
	chargesSection := map[string]any{
		"count":                 charge.ChargeCount,
		"duration_min":          charge.TotalDurationMin,
		"energy_added_kwh":      charge.TotalEnergyAdded,
		"energy_input_kwh":      charge.TotalEnergyUsed,
		"efficiency":            charge.ChargingEfficiency,
		"cost":                  charge.TotalCost,
		"cost_per_kwh":          charge.AverageCostPerKwh,
		"cost_per_100_distance": charge.CostPer100Distance,
		"avg_power_kw":          charge.AveragePower,
		"max_power_kw":          charge.MaxPower,
	}

	// 电池分区。
	batterySection := map[string]any{
		"soc_start":         battery["soc_start_percent"],
		"soc_end":           battery["soc_end_percent"],
		"range_start":       battery["range_start_km"],
		"range_end":         battery["range_end_km"],
		"vampire_drain_kwh": vampireDrain,
	}

	// 停车分区。
	var dominantState any
	stateBreakdown := any(nil)
	if parking != nil {
		dominantState = parking.DominantState
		if len(parking.StateBreakdown) > 0 {
			stateBreakdown = parking.StateBreakdown
		}
	}
	var parkCount, parkDurationMin int
	if parking != nil {
		parkCount = parking.SessionCount
		parkDurationMin = parking.TotalDurationMin
	}
	// 附加当前状态上下文。
	var currentState, lastStateChange any
	if state != nil {
		currentState = state.CurrentState
		lastStateChange = state.LastStateChange
	}
	parkingSection := map[string]any{
		"count":             parkCount,
		"duration_min":      parkDurationMin,
		"dominant_state":    dominantState,
		"state_breakdown":   stateBreakdown,
		"current_state":     currentState,
		"last_state_change": lastStateChange,
	}

	return map[string]any{
		"period":       dr.Period,
		"range":        buildRangeDTO(dr),
		"drives":       drivesSection,
		"charges":      chargesSection,
		"battery":      batterySection,
		"parking":      parkingSection,
		"odometer":     odometer,
		"generated_at": time.Now().In(dr.Timezone).Format(time.RFC3339),
	}
}
