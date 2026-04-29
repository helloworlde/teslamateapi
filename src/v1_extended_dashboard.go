package main

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func TeslaMateAPICarsDashboardV2(c *gin.Context) {
	// dashboard 只聚合车辆级统计，避免把实时状态、图表和明细列表揉进首页接口。
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid dashboard range", map[string]any{"reason": err.Error()})
		return
	}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsDashboardV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	driveSummary, err := fetchDriveHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load dashboard drive summary", map[string]any{"reason": err.Error()})
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load dashboard charge summary", map[string]any{"reason": err.Error()})
		return
	}
	statistics, err := fetchStatisticsSummary(ctx.CarID, startUTC, endUTC, ctx.UnitsLength, ctx.UnitsTemperature, driveSummary, chargeSummary)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load dashboard statistics", map[string]any{"reason": err.Error()})
		return
	}
	data := map[string]any{
		"car_id": ctx.CarID,
		"range":  buildRangeDTO(dr),
		"overview": map[string]any{
			"drive_count":         driveSummary.DriveCount,
			"charge_count":        chargeSummary.ChargeCount,
			"distance":            driveSummary.TotalDistance,
			"drive_duration_min":  driveSummary.TotalDurationMin,
			"charge_duration_min": chargeSummary.TotalDurationMin,
			"energy_used":         driveSummary.TotalEnergyConsumed,
			"energy_added":        chargeSummary.TotalEnergyAdded,
			"cost":                chargeSummary.TotalCost,
			"average_consumption": driveSummary.AverageConsumption,
			"charging_efficiency": chargeSummary.ChargingEfficiency,
		},
		"statistics": statistics,
	}
	writeV1Object(c, data, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"))
}

func TeslaMateAPICarsRealtimeV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsRealtimeV2")
	if !ok {
		return
	}
	current, err := fetchDashboardCurrentSnapshot(ctx.CarID, ctx.UnitsLength, ctx.UnitsTemperature)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load realtime vehicle snapshot", map[string]any{"reason": err.Error()})
		return
	}
	writeV1Object(c, map[string]any{
		"car_id":  ctx.CarID,
		"current": current,
	}, buildV1Meta(ctx.CarID, appUsersTimezone.String(), "metric"))
}

func fetchDashboardCurrentSnapshot(carID int, unitsLength, unitsTemperature string) (map[string]any, error) {
	// 实时快照只取每类最新记录：最新位置、最新状态、最新充电过程。
	query := `
		WITH latest_position AS (
			-- 最新位置提供电量、里程、速度、温度、海拔等当前展示字段
			SELECT date, latitude, longitude, speed, power, odometer, battery_level, usable_battery_level,
				rated_battery_range_km, ideal_battery_range_km, outside_temp, inside_temp, elevation
			FROM positions
			WHERE car_id = $1
			ORDER BY date DESC
			LIMIT 1
		),
		latest_state AS (
			-- 最新状态用于判断车辆当前状态和状态持续时间
			SELECT state::text, start_date, end_date
			FROM states
			WHERE car_id = $1
			ORDER BY start_date DESC
			LIMIT 1
		),
		latest_charge AS (
			-- 最新充电过程用于展示是否正在充电和最近一次充电信息
			SELECT id, start_date, end_date, charge_energy_added, charge_energy_used, cost,
				start_battery_level, end_battery_level
			FROM charging_processes
			WHERE car_id = $1
			ORDER BY end_date IS NULL DESC, start_date DESC
			LIMIT 1
		)
		SELECT
			(SELECT date FROM latest_position),
			(SELECT latitude FROM latest_position),
			(SELECT longitude FROM latest_position),
			(SELECT speed FROM latest_position),
			(SELECT power FROM latest_position),
			(SELECT odometer FROM latest_position),
			(SELECT battery_level FROM latest_position),
			(SELECT usable_battery_level FROM latest_position),
			(SELECT rated_battery_range_km FROM latest_position),
			(SELECT ideal_battery_range_km FROM latest_position),
			(SELECT outside_temp FROM latest_position),
			(SELECT inside_temp FROM latest_position),
			(SELECT elevation FROM latest_position),
			(SELECT state FROM latest_state),
			(SELECT start_date FROM latest_state),
			(SELECT end_date FROM latest_state),
			(SELECT id FROM latest_charge),
			(SELECT start_date FROM latest_charge),
			(SELECT end_date FROM latest_charge),
			(SELECT charge_energy_added FROM latest_charge),
			(SELECT charge_energy_used FROM latest_charge),
			(SELECT cost FROM latest_charge),
			(SELECT start_battery_level FROM latest_charge),
			(SELECT end_battery_level FROM latest_charge)`
	var (
		positionDate                                          sql.NullString
		latitude, longitude, odometer, ratedRange, idealRange sql.NullFloat64
		outsideTemp, insideTemp, elevation                    sql.NullFloat64
		speed, power, batteryLevel, usableBatteryLevel        sql.NullInt64
		state, stateStart, stateEnd                           sql.NullString
		chargeID, chargeStartBattery, chargeEndBattery        sql.NullInt64
		chargeStart, chargeEnd                                sql.NullString
		chargeEnergyAdded, chargeEnergyUsed, chargeCost       sql.NullFloat64
	)
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(queryCtx, query, carID).Scan(
		&positionDate,
		&latitude,
		&longitude,
		&speed,
		&power,
		&odometer,
		&batteryLevel,
		&usableBatteryLevel,
		&ratedRange,
		&idealRange,
		&outsideTemp,
		&insideTemp,
		&elevation,
		&state,
		&stateStart,
		&stateEnd,
		&chargeID,
		&chargeStart,
		&chargeEnd,
		&chargeEnergyAdded,
		&chargeEnergyUsed,
		&chargeCost,
		&chargeStartBattery,
		&chargeEndBattery,
	); err != nil {
		return nil, err
	}
	if strings.EqualFold(unitsLength, "mi") {
		odometer = kilometersToMilesSqlNullFloat64(odometer)
		ratedRange = kilometersToMilesSqlNullFloat64(ratedRange)
		idealRange = kilometersToMilesSqlNullFloat64(idealRange)
		speed = kilometersToMilesSqlNullInt64(speed)
	}
	if strings.EqualFold(unitsTemperature, "f") {
		if outsideTemp.Valid {
			outsideTemp.Float64 = celsiusToFahrenheit(outsideTemp.Float64)
		}
		if insideTemp.Valid {
			insideTemp.Float64 = celsiusToFahrenheit(insideTemp.Float64)
		}
	}
	isCharging := chargeID.Valid && !chargeEnd.Valid
	return map[string]any{
		"position": map[string]any{
			"time":                 timeZoneStringPointer(positionDate),
			"latitude":             floatPointer(latitude),
			"longitude":            floatPointer(longitude),
			"speed":                intPointer(speed),
			"power":                intPointer(power),
			"odometer":             floatPointer(odometer),
			"battery_level":        intPointer(batteryLevel),
			"usable_battery_level": intPointer(usableBatteryLevel),
			"rated_range":          floatPointer(ratedRange),
			"ideal_range":          floatPointer(idealRange),
			"outside_temperature":  floatPointer(outsideTemp),
			"inside_temperature":   floatPointer(insideTemp),
			"elevation":            floatPointer(elevation),
		},
		"state": map[string]any{
			"name":       stringPointer(state),
			"start_time": timeZoneStringPointer(stateStart),
			"end_time":   timeZoneStringPointer(stateEnd),
		},
		"charge": map[string]any{
			"charge_id":           intPointer(chargeID),
			"is_charging":         isCharging,
			"start_time":          timeZoneStringPointer(chargeStart),
			"end_time":            timeZoneStringPointer(chargeEnd),
			"energy_added":        floatPointer(chargeEnergyAdded),
			"energy_used":         floatPointer(chargeEnergyUsed),
			"cost":                floatPointer(chargeCost),
			"start_battery_level": intPointer(chargeStartBattery),
			"end_battery_level":   intPointer(chargeEndBattery),
		},
	}, nil
}
