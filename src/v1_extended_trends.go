package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

// TeslaMateAPICarsTrendsV2 returns period-over-period trend indicators for key metrics.
// Always compares the requested period against the immediately preceding equivalent period.
func TeslaMateAPICarsAnalysisTrendsV2(c *gin.Context) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid trends range", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsTrendsV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)

	startT, _ := time.ParseInLocation(dbTimestampFormat, startUTC, time.UTC)
	endT, _ := time.ParseInLocation(dbTimestampFormat, endUTC, time.UTC)
	duration := endT.Sub(startT)
	prevStartUTC := startT.Add(-duration).UTC().Format(dbTimestampFormat)
	prevEndUTC := startUTC

	var (
		curDrive   *DriveHistorySummary
		curCharge  *ChargeHistorySummary
		curPark    *float64
		prevDrive  *DriveHistorySummary
		prevCharge *ChargeHistorySummary
		prevPark   *float64
	)
	g := new(errgroup.Group)
	g.Go(func() error {
		v, err := fetchDriveHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
		if err != nil {
			return err
		}
		curDrive = v
		return nil
	})
	g.Go(func() error {
		v, err := fetchChargeHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
		if err != nil {
			return err
		}
		curCharge = v
		return nil
	})
	g.Go(func() error {
		curPark, _ = fetchParkingEnergyTotal(ctx.CarID, startUTC, endUTC)
		return nil
	})
	g.Go(func() error {
		prevDrive, _ = fetchDriveHistorySummary(ctx.CarID, prevStartUTC, prevEndUTC, ctx.UnitsLength)
		return nil
	})
	g.Go(func() error {
		prevCharge, _ = fetchChargeHistorySummary(ctx.CarID, prevStartUTC, prevEndUTC, ctx.UnitsLength)
		return nil
	})
	g.Go(func() error {
		prevPark, _ = fetchParkingEnergyTotal(ctx.CarID, prevStartUTC, prevEndUTC)
		return nil
	})
	if err := g.Wait(); err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load trends", map[string]any{"reason": err.Error()})
		return
	}

	trends := buildTrends(curDrive, curCharge, curPark, prevDrive, prevCharge, prevPark, ctx.UnitsLength)
	writeV1Object(c, map[string]any{
		"car_id":           ctx.CarID,
		"period":           dr.Period,
		"range":            buildRangeDTO(dr),
		"comparison_range": buildComparisonRangeDTO(prevStartUTC, prevEndUTC, dr.Timezone),
		"trends":           trends,
	}, buildV1MetaFromCar(ctx, dr.Timezone.String()))
}

func buildTrends(
	curDrive *DriveHistorySummary, curCharge *ChargeHistorySummary, curPark *float64,
	prevDrive *DriveHistorySummary, prevCharge *ChargeHistorySummary, prevPark *float64,
	unitsLength string,
) []map[string]any {
	distUnit := "km"
	speedUnit := "km/h"
	consumptionUnit := "Wh/km"
	if isImperial(unitsLength) {
		distUnit = "mi"
		speedUnit = "mph"
		consumptionUnit = "Wh/mi"
	}

	trend := func(metric, name, unit string, cur, prev any, higherIsBetter bool) map[string]any {
		direction := trendDirection(cur, prev, higherIsBetter)
		return map[string]any{
			"metric":          metric,
			"name":            name,
			"unit":            unit,
			"current":         cur,
			"previous":        prev,
			"change_percent":  calcDeltaPercent(cur, prev),
			"direction":       direction,
			"higher_is_better": higherIsBetter,
		}
	}

	result := make([]map[string]any, 0, 10)

	// Drive metrics
	var curDist, prevDist any
	var curEfficiency, prevEfficiency any
	var curSpeed, prevSpeed any
	var curDriveCount, prevDriveCount any
	if curDrive != nil {
		curDist = curDrive.TotalDistance
		curEfficiency = curDrive.AverageConsumption
		curSpeed = curDrive.AverageSpeed
		curDriveCount = curDrive.DriveCount
	}
	if prevDrive != nil {
		prevDist = prevDrive.TotalDistance
		prevEfficiency = prevDrive.AverageConsumption
		prevSpeed = prevDrive.AverageSpeed
		prevDriveCount = prevDrive.DriveCount
	}
	result = append(result,
		trend("distance", "Distance Driven", distUnit, curDist, prevDist, true),
		trend("drive_count", "Drive Count", "count", curDriveCount, prevDriveCount, true),
		trend("efficiency", "Drive Efficiency", consumptionUnit, curEfficiency, prevEfficiency, false),
		trend("avg_speed", "Average Speed", speedUnit, curSpeed, prevSpeed, false),
	)

	// Charge metrics
	var curEnergyAdded, prevEnergyAdded any
	var curChargeCost, prevChargeCost any
	var curChargeCount, prevChargeCount any
	var curChargingEff, prevChargingEff any
	if curCharge != nil {
		curEnergyAdded = curCharge.TotalEnergyAdded
		curChargeCost = curCharge.TotalCost
		curChargeCount = curCharge.ChargeCount
		curChargingEff = curCharge.ChargingEfficiency
	}
	if prevCharge != nil {
		prevEnergyAdded = prevCharge.TotalEnergyAdded
		prevChargeCost = prevCharge.TotalCost
		prevChargeCount = prevCharge.ChargeCount
		prevChargingEff = prevCharge.ChargingEfficiency
	}
	result = append(result,
		trend("energy_added", "Energy Added", "kWh", curEnergyAdded, prevEnergyAdded, false),
		trend("charge_count", "Charge Count", "count", curChargeCount, prevChargeCount, false),
		trend("charge_cost", "Charge Cost", "currency", curChargeCost, prevChargeCost, false),
		trend("charging_efficiency", "Charging Efficiency", "%", curChargingEff, prevChargingEff, true),
	)

	// Parking / battery
	result = append(result,
		trend("vampire_drain", "Vampire Drain", "kWh", curPark, prevPark, false),
	)

	return result
}

func trendDirection(cur, prev any, higherIsBetter bool) string {
	c, ok1 := asFloat64(cur)
	p, ok2 := asFloat64(prev)
	if !ok1 || !ok2 || p == 0 {
		return "neutral"
	}
	delta := (c - p) / p
	if delta > 0.02 {
		if higherIsBetter {
			return "better"
		}
		return "worse"
	}
	if delta < -0.02 {
		if higherIsBetter {
			return "worse"
		}
		return "better"
	}
	return "stable"
}

func isImperial(unitsLength string) bool {
	return unitsLength == "mi"
}

func buildComparisonRangeDTO(startUTC, endUTC string, loc *time.Location) map[string]any {
	s, _ := time.ParseInLocation(dbTimestampFormat, startUTC, time.UTC)
	e, _ := time.ParseInLocation(dbTimestampFormat, endUTC, time.UTC)
	return map[string]any{
		"start":    s.In(loc).Format(time.RFC3339),
		"end":      e.In(loc).Add(-time.Second).Format(time.RFC3339),
		"timezone": loc.String(),
	}
}
