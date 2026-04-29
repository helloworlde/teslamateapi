package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func TeslaMateAPICarsUnifiedInsightsV2(c *gin.Context) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid insight range", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsUnifiedInsightsV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	types := parseCSV(c.Query("types"))
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	insights := buildSimpleInsights(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, types, limit)
	levels := map[string]int{"positive": 0, "warning": 0, "info": 0}
	for _, item := range insights {
		level, _ := item["level"].(string)
		levels[level]++
	}
	writeV1Object(c, map[string]any{
		"car_id": ctx.CarID,
		"range":  buildRangeDTO(dr),
		"summary": map[string]any{
			"positive_count": levels["positive"],
			"warning_count":  levels["warning"],
			"info_count":     levels["info"],
			"total_count":    len(insights),
		},
		"insights": insights,
	}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"))
}

func buildSimpleInsights(carID int, startUTC, endUTC, unitsLength string, types []string, limit int) []map[string]any {
	items := make([]map[string]any, 0)
	typeSet := map[string]bool{}
	for _, t := range types {
		typeSet[t] = true
	}
	accept := func(tp string) bool {
		if len(typeSet) == 0 {
			return true
		}
		return typeSet[tp]
	}
	appendInsight := func(id, tp, level, title, message, metric string, current any, baseline any, related map[string]any) {
		if len(items) >= limit || !accept(tp) {
			return
		}
		delta := calcDeltaPercent(current, baseline)
		items = append(items, map[string]any{
			"id":            id,
			"type":          tp,
			"level":         level,
			"title":         title,
			"message":       message,
			"metric":        metric,
			"current":       current,
			"baseline":      baseline,
			"delta_percent": delta,
			"related":       related,
		})
	}

	currentDrive, err := fetchDriveHistorySummary(carID, startUTC, endUTC, unitsLength)
	if err != nil {
		return items
	}
	currentCharge, err := fetchChargeHistorySummary(carID, startUTC, endUTC, unitsLength)
	if err != nil {
		return items
	}
	currentRegen, regenErr := fetchRegenerationSummary(carID, startUTC, endUTC, currentDrive, unitsLength)
	_ = regenErr
	currentPark, parkErr := fetchParkingEnergyTotal(carID, startUTC, endUTC)
	_ = parkErr

	startT, startErr := time.ParseInLocation(dbTimestampFormat, startUTC, time.UTC)
	endT, endErr := time.ParseInLocation(dbTimestampFormat, endUTC, time.UTC)
	if startErr != nil || endErr != nil || !endT.After(startT) {
		return items
	}
	duration := endT.Sub(startT)
	baseStart := startT.Add(-duration)
	baseEnd := startT
	baseStartUTC := baseStart.UTC().Format(dbTimestampFormat)
	baseEndUTC := baseEnd.UTC().Format(dbTimestampFormat)

	baseDrive, driveBaseErr := fetchDriveHistorySummary(carID, baseStartUTC, baseEndUTC, unitsLength)
	baseCharge, chargeBaseErr := fetchChargeHistorySummary(carID, baseStartUTC, baseEndUTC, unitsLength)
	if chargeBaseErr != nil {
		baseCharge = nil
	}
	var baseRegen *RegenerationSummary
	if driveBaseErr == nil {
		baseRegen, _ = fetchRegenerationSummary(carID, baseStartUTC, baseEndUTC, baseDrive, unitsLength)
	}
	basePark, _ := fetchParkingEnergyTotal(carID, baseStartUTC, baseEndUTC)

	if currentDrive.AverageConsumption != nil && baseDrive != nil && baseDrive.AverageConsumption != nil {
		cur := *currentDrive.AverageConsumption
		base := *baseDrive.AverageConsumption
		if base > 0 && cur >= base*1.1 {
			appendInsight("drive_efficiency_worse", "efficiency", "warning", "Driving efficiency worsened", "Average consumption is at least 10% higher than the previous equivalent period.", "avg_efficiency", cur, base, map[string]any{"entity_type": "drive"})
		}
		if base > 0 && cur <= base*0.9 {
			appendInsight("drive_efficiency_better", "efficiency", "positive", "Driving efficiency improved", "Average consumption is at least 10% lower than the previous equivalent period.", "avg_efficiency", cur, base, map[string]any{"entity_type": "drive"})
		}
	}
	if currentCharge.AverageCostPerKwh != nil && baseCharge != nil && baseCharge.AverageCostPerKwh != nil {
		cur := *currentCharge.AverageCostPerKwh
		base := *baseCharge.AverageCostPerKwh
		if base > 0 && cur >= base*1.15 {
			appendInsight("charge_cost_higher", "cost", "warning", "Charging cost increased", "Average charging cost per kWh is at least 15% higher than the previous equivalent period.", "cost_per_kwh", cur, base, map[string]any{"entity_type": "charge"})
		}
		if base > 0 && cur <= base*0.85 {
			appendInsight("charge_cost_lower", "cost", "positive", "Charging cost decreased", "Average charging cost per kWh is at least 15% lower than the previous equivalent period.", "cost_per_kwh", cur, base, map[string]any{"entity_type": "charge"})
		}
	}
	if currentCharge.ChargingEfficiency != nil && baseCharge != nil && baseCharge.ChargingEfficiency != nil {
		cur := *currentCharge.ChargingEfficiency * 100.0
		base := *baseCharge.ChargingEfficiency * 100.0
		if cur < base-5 {
			appendInsight("charge_efficiency_drop", "charging", "warning", "Charging efficiency dropped", "Charging efficiency is more than 5 percentage points below the previous equivalent period.", "charging_efficiency_percent", cur, base, map[string]any{"entity_type": "charge"})
		}
	}
	if currentRegen != nil && currentRegen.RecoveryShare != nil && baseRegen != nil && baseRegen.RecoveryShare != nil {
		cur := *currentRegen.RecoveryShare
		base := *baseRegen.RecoveryShare
		if base > 0 && cur >= base*1.1 {
			appendInsight("regen_share_higher", "driving", "positive", "Regeneration share improved", "Estimated recovered energy share is at least 10% higher than the previous equivalent period.", "regeneration_share", cur, base, map[string]any{"entity_type": "drive"})
		}
	}
	if currentPark != nil && basePark != nil {
		cur := *currentPark
		base := *basePark
		if base > 0 && cur >= base*1.2 {
			appendInsight("vampire_drain_higher", "battery", "warning", "Parking energy loss increased", "Estimated parking energy loss is at least 20% higher than the previous equivalent period.", "park_energy_kwh", cur, base, map[string]any{"entity_type": "state"})
		}
	}
	if currentDrive.DriveCount > 0 {
		ratio := float64(currentDrive.LowSpeedTripCount) / float64(currentDrive.DriveCount)
		if ratio >= 0.45 {
			appendInsight("traffic_ratio_high", "anomaly", "info", "Frequent low-speed driving", "At least 45% of trips were low-speed trips, which usually indicates congestion or short urban driving.", "low_speed_trip_ratio", ratio, nil, map[string]any{"entity_type": "drive"})
		}
	}
	if currentCharge.ChargeCount > 0 {
		days := duration.Hours() / 24.0
		if days > 0 {
			freq := float64(currentCharge.ChargeCount) / days
			if freq >= 1.2 {
				appendInsight("charge_frequency_high", "charging", "info", "High charging frequency", "Charging frequency is above 1.2 sessions per day for this period.", "charges_per_day", freq, nil, map[string]any{"entity_type": "charge"})
			}
		}
	}
	return items
}

func calcDeltaPercent(current any, baseline any) any {
	cur, ok1 := asFloat64(current)
	base, ok2 := asFloat64(baseline)
	if !ok1 || !ok2 || base == 0 {
		return nil
	}
	return (cur - base) / base * 100.0
}
