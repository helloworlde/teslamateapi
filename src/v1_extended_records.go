package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	aggregatecache "github.com/tobiasehlert/teslamateapi/src/internal/aggregatecache"
)

// TeslaMateAPICarsRecordsV2 returns all-time personal best records for the car.
// Accepts optional ?year=YYYY to scope records to a specific year.
func TeslaMateAPICarsRecordsV2(c *gin.Context) {
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsRecordsV2")
	if !ok {
		return
	}
	yearFilter := strings.TrimSpace(c.Query("year"))
	records, err := fetchCarRecords(ctx.CarID, ctx.UnitsLength, ctx.UnitsTemperature, yearFilter)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load records", map[string]any{"reason": err.Error()})
		return
	}
	writeV1Object(c, map[string]any{
		"car_id":      ctx.CarID,
		"year_filter": nilIfEmpty(yearFilter),
		"records":     records,
	}, buildV1MetaFromCar(ctx, appUsersTimezone.String()))
}

func fetchCarRecords(carID int, unitsLength, unitsTemperature, yearFilter string) (map[string]any, error) {
	key := aggregatecache.Key("records", carID, unitsLength, unitsTemperature, yearFilter)
	return aggregatecache.Value(key, 6*time.Hour, func() (map[string]any, error) {
		return fetchCarRecordsUncached(carID, unitsLength, unitsTemperature, yearFilter)
	})
}

func fetchCarRecordsUncached(carID int, unitsLength, unitsTemperature, yearFilter string) (map[string]any, error) {
	yearClause := ""
	if yearFilter != "" {
		yearClause = " AND EXTRACT(YEAR FROM start_date AT TIME ZONE $2) = " + yearFilter
	}

	type driveRecord struct {
		LongestDistanceKm  sql.NullFloat64
		LongestDistanceID  sql.NullInt64
		LongestDistDate    sql.NullString
		BestEfficiencyWhKm sql.NullFloat64
		BestEfficiencyID   sql.NullInt64
		BestEfficiencyDate sql.NullString
		HighestSpeedKmh    sql.NullInt64
		HighestSpeedID     sql.NullInt64
		HighestSpeedDate   sql.NullString
		LongestDurationMin sql.NullInt64
		LongestDurationID  sql.NullInt64
		LongestDurationDate sql.NullString
		ColdestTempC       sql.NullFloat64
		ColdestTempDate    sql.NullString
	}

	driveQuery := `
		WITH drive_efficiency AS (
			SELECT id, start_date,
				distance,
				duration_min,
				speed_max,
				outside_temp_avg,
				CASE WHEN distance > 0 AND (start_rated_range_km - end_rated_range_km) > 0
					THEN (start_rated_range_km - end_rated_range_km) * c.efficiency / distance * 1000.0
					ELSE NULL END AS efficiency_wh_km
			FROM drives d
			LEFT JOIN cars c ON c.id = d.car_id
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL` + yearClause + `
		)
		SELECT
			(SELECT distance FROM drive_efficiency ORDER BY distance DESC NULLS LAST LIMIT 1),
			(SELECT id FROM drive_efficiency ORDER BY distance DESC NULLS LAST LIMIT 1),
			(SELECT TO_CHAR(start_date AT TIME ZONE $2, 'YYYY-MM-DD') FROM drive_efficiency ORDER BY distance DESC NULLS LAST LIMIT 1),
			(SELECT efficiency_wh_km FROM drive_efficiency WHERE efficiency_wh_km IS NOT NULL ORDER BY efficiency_wh_km ASC LIMIT 1),
			(SELECT id FROM drive_efficiency WHERE efficiency_wh_km IS NOT NULL ORDER BY efficiency_wh_km ASC LIMIT 1),
			(SELECT TO_CHAR(start_date AT TIME ZONE $2, 'YYYY-MM-DD') FROM drive_efficiency WHERE efficiency_wh_km IS NOT NULL ORDER BY efficiency_wh_km ASC LIMIT 1),
			(SELECT speed_max FROM drive_efficiency ORDER BY speed_max DESC NULLS LAST LIMIT 1),
			(SELECT id FROM drive_efficiency ORDER BY speed_max DESC NULLS LAST LIMIT 1),
			(SELECT TO_CHAR(start_date AT TIME ZONE $2, 'YYYY-MM-DD') FROM drive_efficiency ORDER BY speed_max DESC NULLS LAST LIMIT 1),
			(SELECT duration_min FROM drive_efficiency ORDER BY duration_min DESC NULLS LAST LIMIT 1),
			(SELECT id FROM drive_efficiency ORDER BY duration_min DESC NULLS LAST LIMIT 1),
			(SELECT TO_CHAR(start_date AT TIME ZONE $2, 'YYYY-MM-DD') FROM drive_efficiency ORDER BY duration_min DESC NULLS LAST LIMIT 1),
			(SELECT outside_temp_avg FROM drive_efficiency WHERE outside_temp_avg IS NOT NULL ORDER BY outside_temp_avg ASC LIMIT 1),
			(SELECT TO_CHAR(start_date AT TIME ZONE $2, 'YYYY-MM-DD') FROM drive_efficiency WHERE outside_temp_avg IS NOT NULL ORDER BY outside_temp_avg ASC LIMIT 1)`

	var d driveRecord
	qCtx, cancel := newAggregateQueryContext()
	defer cancel()
	tz := appUsersTimezone.String()
	if err := db.QueryRowContext(qCtx, driveQuery, carID, tz).Scan(
		&d.LongestDistanceKm, &d.LongestDistanceID, &d.LongestDistDate,
		&d.BestEfficiencyWhKm, &d.BestEfficiencyID, &d.BestEfficiencyDate,
		&d.HighestSpeedKmh, &d.HighestSpeedID, &d.HighestSpeedDate,
		&d.LongestDurationMin, &d.LongestDurationID, &d.LongestDurationDate,
		&d.ColdestTempC, &d.ColdestTempDate,
	); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	type chargeRecord struct {
		LargestEnergyKwh   sql.NullFloat64
		LargestEnergyID    sql.NullInt64
		LargestEnergyDate  sql.NullString
		FastestPowerKw     sql.NullFloat64
		FastestPowerID     sql.NullInt64
		FastestPowerDate   sql.NullString
		LongestChargeMin   sql.NullInt64
		LongestChargeID    sql.NullInt64
		LongestChargeDate  sql.NullString
		LowestStartSoc     sql.NullInt64
		LowestStartSocID   sql.NullInt64
		LowestStartSocDate sql.NullString
	}

	chargeYearClause := ""
	if yearFilter != "" {
		chargeYearClause = " AND EXTRACT(YEAR FROM cp.start_date AT TIME ZONE $2) = " + yearFilter
	}
	chargeQuery := `
		WITH charge_power AS (
			SELECT cp.id, cp.start_date, cp.charge_energy_added,
				cp.start_battery_level,
				EXTRACT(EPOCH FROM (cp.end_date - cp.start_date))/60.0 AS duration_min,
				MAX(ch.charger_power) AS max_power_kw
			FROM charging_processes cp
			LEFT JOIN charges ch ON ch.charging_process_id = cp.id
			WHERE cp.car_id = $1 AND cp.end_date IS NOT NULL` + chargeYearClause + `
			GROUP BY cp.id
		)
		SELECT
			(SELECT charge_energy_added FROM charge_power ORDER BY charge_energy_added DESC NULLS LAST LIMIT 1),
			(SELECT id FROM charge_power ORDER BY charge_energy_added DESC NULLS LAST LIMIT 1),
			(SELECT TO_CHAR(start_date AT TIME ZONE $2, 'YYYY-MM-DD') FROM charge_power ORDER BY charge_energy_added DESC NULLS LAST LIMIT 1),
			(SELECT max_power_kw FROM charge_power ORDER BY max_power_kw DESC NULLS LAST LIMIT 1),
			(SELECT id FROM charge_power ORDER BY max_power_kw DESC NULLS LAST LIMIT 1),
			(SELECT TO_CHAR(start_date AT TIME ZONE $2, 'YYYY-MM-DD') FROM charge_power ORDER BY max_power_kw DESC NULLS LAST LIMIT 1),
			(SELECT duration_min FROM charge_power ORDER BY duration_min DESC NULLS LAST LIMIT 1),
			(SELECT id FROM charge_power ORDER BY duration_min DESC NULLS LAST LIMIT 1),
			(SELECT TO_CHAR(start_date AT TIME ZONE $2, 'YYYY-MM-DD') FROM charge_power ORDER BY duration_min DESC NULLS LAST LIMIT 1),
			(SELECT start_battery_level FROM charge_power WHERE start_battery_level IS NOT NULL ORDER BY start_battery_level ASC LIMIT 1),
			(SELECT id FROM charge_power WHERE start_battery_level IS NOT NULL ORDER BY start_battery_level ASC LIMIT 1),
			(SELECT TO_CHAR(start_date AT TIME ZONE $2, 'YYYY-MM-DD') FROM charge_power WHERE start_battery_level IS NOT NULL ORDER BY start_battery_level ASC LIMIT 1)`

	var ch chargeRecord
	qCtx2, cancel2 := newAggregateQueryContext()
	defer cancel2()
	if err := db.QueryRowContext(qCtx2, chargeQuery, carID, tz).Scan(
		&ch.LargestEnergyKwh, &ch.LargestEnergyID, &ch.LargestEnergyDate,
		&ch.FastestPowerKw, &ch.FastestPowerID, &ch.FastestPowerDate,
		&ch.LongestChargeMin, &ch.LongestChargeID, &ch.LongestChargeDate,
		&ch.LowestStartSoc, &ch.LowestStartSocID, &ch.LowestStartSocDate,
	); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Apply unit conversion
	distUnit := "km"
	speedUnit := "km/h"
	consumptionUnit := "Wh/km"
	longestDist := floatOrNil(d.LongestDistanceKm)
	bestEfficiency := floatOrNil(d.BestEfficiencyWhKm)
	highestSpeed := intOrNil(d.HighestSpeedKmh)
	coldestTemp := floatOrNil(d.ColdestTempC)
	if isImperial(unitsLength) {
		distUnit = "mi"
		speedUnit = "mph"
		consumptionUnit = "Wh/mi"
		if v, ok := longestDist.(float64); ok {
			longestDist = kilometersToMiles(v)
		}
		if v, ok := bestEfficiency.(float64); ok {
			bestEfficiency = whPerKmToWhPerMi(v)
		}
		if v, ok := highestSpeed.(int); ok {
			highestSpeed = kilometersToMilesInteger(v)
		}
	}
	if strings.EqualFold(unitsTemperature, "f") {
		if v, ok := coldestTemp.(float64); ok {
			coldestTemp = celsiusToFahrenheit(v)
		}
	}
	tempUnit := "°C"
	if strings.EqualFold(unitsTemperature, "f") {
		tempUnit = "°F"
	}

	makeRecord := func(value any, unit, date string, entityType string, entityID sql.NullInt64) map[string]any {
		rec := map[string]any{
			"value":       value,
			"unit":        unit,
			"date":        nilIfEmpty(date),
			"entity_type": entityType,
		}
		if entityID.Valid {
			rec["entity_id"] = int(entityID.Int64)
		} else {
			rec["entity_id"] = nil
		}
		return rec
	}

	return map[string]any{
		"longest_drive": makeRecord(longestDist, distUnit,
			nullStringVal(d.LongestDistDate), "drive", d.LongestDistanceID),
		"best_efficiency": makeRecord(bestEfficiency, consumptionUnit,
			nullStringVal(d.BestEfficiencyDate), "drive", d.BestEfficiencyID),
		"highest_speed": makeRecord(highestSpeed, speedUnit,
			nullStringVal(d.HighestSpeedDate), "drive", d.HighestSpeedID),
		"longest_drive_duration": makeRecord(intOrNil(d.LongestDurationMin), "min",
			nullStringVal(d.LongestDurationDate), "drive", d.LongestDurationID),
		"coldest_drive": makeRecord(coldestTemp, tempUnit,
			nullStringVal(d.ColdestTempDate), "drive", sql.NullInt64{}),
		"largest_charge": makeRecord(floatOrNil(ch.LargestEnergyKwh), "kWh",
			nullStringVal(ch.LargestEnergyDate), "charge", ch.LargestEnergyID),
		"fastest_charge_power": makeRecord(floatOrNil(ch.FastestPowerKw), "kW",
			nullStringVal(ch.FastestPowerDate), "charge", ch.FastestPowerID),
		"longest_charge_duration": makeRecord(intOrNil(ch.LongestChargeMin), "min",
			nullStringVal(ch.LongestChargeDate), "charge", ch.LongestChargeID),
		"lowest_charge_start_soc": makeRecord(intOrNil(ch.LowestStartSoc), "%",
			nullStringVal(ch.LowestStartSocDate), "charge", ch.LowestStartSocID),
	}, nil
}

func nullStringVal(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
