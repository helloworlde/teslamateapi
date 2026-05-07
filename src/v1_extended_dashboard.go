package main

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TeslaMateAPICarsStatusV2 返回实时车辆状态快照。
// @Summary 车辆状态快照
// @Description 扩展接口 v2：返回最新已知车辆状态。
// @Tags 扩展 API
// @Produce json
// @Param CarID path int true "车辆 ID" default(1)
// @Success 200 {object} StatusV2Envelope
// @Failure 400,404,500 {object} v1ErrorEnvelope
// @Router /v2/cars/{CarID}/status [get]
func TeslaMateAPICarsStatusV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsStatusV2")
	if !ok {
		return
	}
	snapshot, err := fetchVehicleStatus(ctx.CarID, ctx.UnitsLength, ctx.UnitsTemperature)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load vehicle status", map[string]any{"reason": err.Error()})
		return
	}
	writeV1Object(c, snapshot, buildV1MetaFromCar(ctx, appUsersTimezone.String()))
}

func fetchVehicleStatus(carID int, unitsLength, unitsTemperature string) (map[string]any, error) {
	query := `
		WITH latest_position AS (
			SELECT date, latitude, longitude, speed, power, odometer, battery_level,
				usable_battery_level, rated_battery_range_km, ideal_battery_range_km,
				outside_temp, inside_temp, elevation
			FROM positions
			WHERE car_id = $1
			ORDER BY date DESC
			LIMIT 1
		),
		latest_state AS (
			SELECT state::text, start_date, end_date
			FROM states
			WHERE car_id = $1
			ORDER BY start_date DESC
			LIMIT 1
		),
		active_charge AS (
			SELECT id, start_date, charge_energy_added, start_battery_level
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NULL
			ORDER BY start_date DESC
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
			(SELECT id FROM active_charge),
			(SELECT start_date FROM active_charge),
			(SELECT charge_energy_added FROM active_charge),
			(SELECT start_battery_level FROM active_charge)`

	var (
		posDate                                  sql.NullString
		lat, lng, odo, ratedRange, idealRange    sql.NullFloat64
		outsideTemp, insideTemp, elevation       sql.NullFloat64
		speed, power, battLevel, usableBattLevel sql.NullInt64
		state, stateSince                        sql.NullString
		chargeID, chargeStartSOC                 sql.NullInt64
		chargeStart                              sql.NullString
		chargeEnergyAdded                        sql.NullFloat64
	)
	qCtx, cancel := newAggregateQueryContext()
	defer cancel()
	if err := db.QueryRowContext(qCtx, query, carID).Scan(
		&posDate, &lat, &lng, &speed, &power, &odo,
		&battLevel, &usableBattLevel, &ratedRange, &idealRange,
		&outsideTemp, &insideTemp, &elevation,
		&state, &stateSince,
		&chargeID, &chargeStart, &chargeEnergyAdded, &chargeStartSOC,
	); err != nil {
		return nil, err
	}

	if strings.EqualFold(unitsLength, "mi") {
		odo = kilometersToMilesSqlNullFloat64(odo)
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

	// 当前充电会话，仅在充电进行中返回。
	var activeCharge any
	if chargeID.Valid {
		activeCharge = map[string]any{
			"id":               int(chargeID.Int64),
			"started_at":       timeZoneStringPointer(chargeStart),
			"energy_added_kwh": floatPointer(chargeEnergyAdded),
			"start_soc":        intPointer(chargeStartSOC),
		}
	}

	return map[string]any{
		"state": stringPointer(state),
		"since": timeZoneStringPointer(stateSince),
		"battery": map[string]any{
			"level":        intPointer(battLevel),
			"usable_level": intPointer(usableBattLevel),
			"rated_range":  floatPointer(ratedRange),
			"ideal_range":  floatPointer(idealRange),
		},
		"position": map[string]any{
			"time":      timeZoneStringPointer(posDate),
			"lat":       floatPointer(lat),
			"lng":       floatPointer(lng),
			"speed":     intPointer(speed),
			"power":     intPointer(power),
			"elevation": floatPointer(elevation),
		},
		"environment": map[string]any{
			"outside_temp": floatPointer(outsideTemp),
			"inside_temp":  floatPointer(insideTemp),
		},
		"odometer":      floatPointer(odo),
		"active_charge": activeCharge,
	}, nil
}
