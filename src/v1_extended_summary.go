package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func parseSummaryRangeStrict(c *gin.Context) (v1DateRange, error) {
	loc, _, err := parseTimezoneParam(c)
	if err != nil {
		return v1DateRange{}, err
	}
	startRaw := c.Query("startDate")
	endRaw := c.Query("endDate")
	if startRaw != "" || endRaw != "" {
		start, err := parseDateOnlyOrTime(startRaw, loc, false)
		if err != nil {
			return v1DateRange{}, err
		}
		end, err := parseDateOnlyOrTime(endRaw, loc, true)
		if err != nil {
			return v1DateRange{}, err
		}
		if start.IsZero() || end.IsZero() {
			return v1DateRange{}, fmt.Errorf("startDate and endDate are required together")
		}
		if end.Before(start) {
			return v1DateRange{}, fmt.Errorf("startDate must be before endDate")
		}
		return v1DateRange{Period: "custom", Timezone: loc, Start: start, End: end}, nil
	}
	return parseDateRangeFromQuery(c, "month")
}

func buildUnifiedSummary(ctx *apiCarContext, dr v1DateRange) (map[string]any, error) {
	startUTC, endUTC := dbTimeRange(dr)

	driveSummary, err := fetchDriveHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		return nil, fmt.Errorf("drive summary: %w", err)
	}
	chargeSummary, err := fetchChargeHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		return nil, fmt.Errorf("charge summary: %w", err)
	}
	parkingSummary, err := fetchParkingHistorySummary(ctx.CarID, startUTC, endUTC, nil)
	if err != nil {
		return nil, fmt.Errorf("parking summary: %w", err)
	}
	stateSummary, err := fetchStateSummary(ctx.CarID, startUTC, endUTC)
	if err != nil {
		return nil, fmt.Errorf("state summary: %w", err)
	}
	statistics, err := fetchStatisticsSummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, ctx.UnitsTemperature, driveSummary, chargeSummary)
	if err != nil {
		return nil, fmt.Errorf("statistics: %w", err)
	}

	regeneration, err := fetchRegenerationSummary(ctx.CarID, startUTC, endUTC, driveSummary, ctx.UnitsLength)
	batterySnapshot, err := fetchBatterySnapshot(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		batterySnapshot = map[string]any{}
	}
	parkEnergy, err := fetchParkingEnergyTotal(ctx.CarID, startUTC, endUTC)
	_ = err
	latestOdometer, err := fetchLatestOdometer(ctx.CarID, ctx.UnitsLength)
	_ = err

	regenEnergy := any(nil)
	regenShare := any(nil)
	regenEstimated := false
	if regeneration != nil {
		regenEnergy = regeneration.EstimatedRecoveredEnergyKwh
		regenShare = regeneration.RecoveryShare
		regenEstimated = regeneration.MetricsEstimated
	}

	dataComplete := any(nil)
	if statistics.DataComplete != nil {
		dataComplete = *statistics.DataComplete
	}

	return map[string]any{
		"schema_version": "summary.v1",
		"car": map[string]any{
			"car_id":   ctx.CarID,
			"car_name": string(ctx.CarName),
		},
		"range": buildRangeDTO(dr),
		"units": buildSummaryUnits(ctx.UnitsLength, ctx.UnitsTemperature),
		"overview": map[string]any{
			"drive_count":          driveSummary.DriveCount,
			"charge_count":         chargeSummary.ChargeCount,
			"parking_count":        parkingSummary.SessionCount,
			"distance":             driveSummary.TotalDistance,
			"drive_duration_min":   driveSummary.TotalDurationMin,
			"charge_duration_min":  chargeSummary.TotalDurationMin,
			"parking_duration_min": parkingSummary.TotalDurationMin,
			"energy_used":          driveSummary.TotalEnergyConsumed,
			"energy_added":         chargeSummary.TotalEnergyAdded,
			"cost":                 chargeSummary.TotalCost,
			"latest_odometer":      latestOdometer,
		},
		"driving": map[string]any{
			"count":    driveSummary.DriveCount,
			"distance": driveSummary.TotalDistance,
			"duration": map[string]any{
				"total_min":   driveSummary.TotalDurationMin,
				"average_min": driveSummary.AverageDurationMin,
				"longest_min": driveSummary.LongestDurationMin,
			},
			"speed": map[string]any{
				"average": driveSummary.AverageSpeed,
				"max":     driveSummary.MaxSpeed,
			},
			"energy": map[string]any{
				"used":               driveSummary.TotalEnergyConsumed,
				"regenerated":        regenEnergy,
				"regeneration_ratio": regenShare,
			},
			"consumption": map[string]any{
				"average": driveSummary.AverageConsumption,
				"best":    driveSummary.BestConsumption,
				"worst":   driveSummary.WorstConsumption,
			},
			"power": map[string]any{
				"peak_drive": driveSummary.PeakDrivePower,
				"peak_regen": driveSummary.PeakRegenPower,
			},
		},
		"charging": map[string]any{
			"count": chargeSummary.ChargeCount,
			"duration": map[string]any{
				"total_min":   chargeSummary.TotalDurationMin,
				"average_min": chargeSummary.AverageDurationMin,
				"longest_min": chargeSummary.LongestDurationMin,
			},
			"energy": map[string]any{
				"added":         chargeSummary.TotalEnergyAdded,
				"charger_used":  chargeSummary.TotalEnergyUsed,
				"largest_added": chargeSummary.LargestEnergyAdded,
				"average_added": chargeSummary.AverageEnergyAdded,
			},
			"power": map[string]any{
				"average": chargeSummary.AveragePower,
				"max":     chargeSummary.MaxPower,
			},
			"efficiency": chargeSummary.ChargingEfficiency,
		},
		"parking": map[string]any{
			"count":           parkingSummary.SessionCount,
			"duration_min":    parkingSummary.TotalDurationMin,
			"average_min":     parkingSummary.AverageDurationMin,
			"longest_min":     parkingSummary.LongestDurationMin,
			"dominant_state":  parkingSummary.DominantState,
			"parked_share":    stateSummary.ParkedShare,
			"state_breakdown": parkingSummary.StateBreakdown,
		},
		"battery": map[string]any{
			"soc": map[string]any{
				"start": batterySnapshot["soc_start_percent"],
				"end":   batterySnapshot["soc_end_percent"],
			},
			"rated_range": map[string]any{
				"start": batterySnapshot["range_start_km"],
				"end":   batterySnapshot["range_end_km"],
			},
			"vampire_drain_energy": parkEnergy,
		},
		"efficiency": map[string]any{
			"drive_average_consumption": driveSummary.AverageConsumption,
			"drive_best_consumption":    driveSummary.BestConsumption,
			"drive_worst_consumption":   driveSummary.WorstConsumption,
			"gross_consumption":         statistics.AverageConsumptionGross,
			"charging_efficiency":       chargeSummary.ChargingEfficiency,
			"consumption_overhead":      statistics.ConsumptionOverhead,
			"regeneration_ratio":        regenShare,
		},
		"cost": map[string]any{
			"total":            chargeSummary.TotalCost,
			"average":          chargeSummary.AverageCost,
			"highest":          chargeSummary.HighestCost,
			"average_per_kwh":  chargeSummary.AverageCostPerKwh,
			"per_100_distance": chargeSummary.CostPer100Distance,
			"currency":         "currency",
		},
		"quality": map[string]any{
			"data_complete":                dataComplete,
			"regeneration_estimated":       regenEstimated,
			"low_speed_drive_count":        driveSummary.LowSpeedTripCount,
			"congestion_like_drive_count":  driveSummary.CongestionLikeTripCount,
			"high_consumption_drive_count": driveSummary.HighConsumptionTripCount,
			"low_efficiency_charge_count":  chargeSummary.LowEfficiencyChargeCount,
			"abnormal_charge_count":        chargeSummary.AbnormalChargeCount,
			"cost_available":               chargeSummary.TotalCost != nil,
			"gross_consumption_available":  statistics.AverageConsumptionGross != nil,
			"battery_snapshot_available":   len(batterySnapshot) > 0,
		},
		"state": map[string]any{
			"current":           stateSummary.CurrentState,
			"last_state_change": stateSummary.LastStateChange,
			"breakdown":         stateSummary.StateBreakdown,
		},
		"generated_at": time.Now().In(dr.Timezone).Format(time.RFC3339),
	}, nil
}
