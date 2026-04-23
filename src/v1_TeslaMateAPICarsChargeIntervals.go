package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type ChargeIntervalSummary struct {
	ChargeID         int                            `json:"charge_id"`
	PreviousChargeID *int                           `json:"previous_charge_id"`
	Interval         *ChargeIntervalWindow          `json:"interval"`
	EnergyBreakdown  *ChargeIntervalEnergyBreakdown `json:"energy_breakdown"`
}

type ChargeIntervalWindow struct {
	StartDate                *string  `json:"start_date"`
	EndDate                  *string  `json:"end_date"`
	Distance                 *float64 `json:"distance"`
	RatedRangeBudgetDistance *float64 `json:"rated_range_budget_distance"`
	RangeCompletion          *float64 `json:"range_completion"`
	ConsumedSOCPercent       *int     `json:"consumed_soc_percent"`
}

type ChargeIntervalEnergyBreakdown struct {
	RangeCompletion *float64 `json:"range_completion"`
	DrivingShare    float64  `json:"driving_share"`
	ParkedShare     float64  `json:"parked_share"`
	OtherShare      float64  `json:"other_share"`
	DrivingKWh      *float64 `json:"driving_kwh"`
	ParkedKWh       *float64 `json:"parked_kwh"`
	OtherKWh        *float64 `json:"other_kwh"`
	TotalKWh        *float64 `json:"total_kwh"`
}

type chargeIntervalRow struct {
	ID              int
	StartDate       sql.NullString
	EndDate         sql.NullString
	StartBattery    sql.NullInt64
	EndBattery      sql.NullInt64
	StartRatedRange sql.NullFloat64
	EndRatedRange   sql.NullFloat64
	StartIdealRange sql.NullFloat64
	EndIdealRange   sql.NullFloat64
	Odometer        sql.NullFloat64
}

func TeslaMateAPICarsChargeIntervalV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsChargeIntervalV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	ChargeID := convertStringToInteger(c.Param("ChargeID"))

	current, unitsLength, unitsTemperature, carName, carEfficiency, err := fetchChargeIntervalCurrentCharge(CarID, ChargeID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load charge interval.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, "", "", unitsLength, unitsTemperature)
	summary := ChargeIntervalSummary{ChargeID: ChargeID}

	if !current.StartDate.Valid {
		TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
			"charge_interval": summary,
		}))
		return
	}

	previous, err := fetchPreviousChargeIntervalCharge(CarID, current.StartDate.String)
	if err == sql.ErrNoRows {
		TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
			"charge_interval": summary,
		}))
		return
	}
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load charge interval.", err.Error())
		return
	}

	previousID := previous.ID
	summary.PreviousChargeID = &previousID
	summary.Interval, summary.EnergyBreakdown, err = makeChargeIntervalSummary(current, previous, CarID, carEfficiency, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load charge interval.", err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"charge_interval": summary,
	}))
}

func fetchChargeIntervalCurrentCharge(CarID int, ChargeID int) (chargeIntervalRow, string, string, NullString, float64, error) {
	query := `
		SELECT
			charging_processes.id,
			charging_processes.start_date,
			charging_processes.end_date,
			charging_processes.start_battery_level,
			charging_processes.end_battery_level,
			charging_processes.start_rated_range_km,
			charging_processes.end_rated_range_km,
			charging_processes.start_ideal_range_km,
			charging_processes.end_ideal_range_km,
			position.odometer,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name,
			cars.efficiency
		FROM charging_processes
		LEFT JOIN cars ON charging_processes.car_id = cars.id
		LEFT JOIN positions position ON charging_processes.position_id = position.id
		WHERE charging_processes.car_id=$1
			AND charging_processes.id=$2
			AND charging_processes.end_date IS NOT NULL;`

	var (
		charge                        chargeIntervalRow
		unitsLength, unitsTemperature string
		carName                       NullString
		carEfficiency                 sql.NullFloat64
	)

	err := db.QueryRow(query, CarID, ChargeID).Scan(
		&charge.ID,
		&charge.StartDate,
		&charge.EndDate,
		&charge.StartBattery,
		&charge.EndBattery,
		&charge.StartRatedRange,
		&charge.EndRatedRange,
		&charge.StartIdealRange,
		&charge.EndIdealRange,
		&charge.Odometer,
		&unitsLength,
		&unitsTemperature,
		&carName,
		&carEfficiency,
	)
	if err != nil {
		return chargeIntervalRow{}, "", "", "", 0, err
	}
	if !carEfficiency.Valid {
		carEfficiency.Float64 = 0
	}
	return charge, unitsLength, unitsTemperature, carName, carEfficiency.Float64, nil
}

func fetchPreviousChargeIntervalCharge(CarID int, currentStartDate string) (chargeIntervalRow, error) {
	query := `
		SELECT
			charging_processes.id,
			charging_processes.start_date,
			charging_processes.end_date,
			charging_processes.start_battery_level,
			charging_processes.end_battery_level,
			charging_processes.start_rated_range_km,
			charging_processes.end_rated_range_km,
			charging_processes.start_ideal_range_km,
			charging_processes.end_ideal_range_km,
			position.odometer
		FROM charging_processes
		LEFT JOIN positions position ON charging_processes.position_id = position.id
		WHERE charging_processes.car_id=$1
			AND charging_processes.end_date IS NOT NULL
			AND charging_processes.end_date < $2
		ORDER BY charging_processes.end_date DESC
		LIMIT 1;`

	var charge chargeIntervalRow
	err := db.QueryRow(query, CarID, currentStartDate).Scan(
		&charge.ID,
		&charge.StartDate,
		&charge.EndDate,
		&charge.StartBattery,
		&charge.EndBattery,
		&charge.StartRatedRange,
		&charge.EndRatedRange,
		&charge.StartIdealRange,
		&charge.EndIdealRange,
		&charge.Odometer,
	)
	return charge, err
}

func makeChargeIntervalSummary(
	current chargeIntervalRow,
	previous chargeIntervalRow,
	CarID int,
	carEfficiency float64,
	unitsLength string,
) (*ChargeIntervalWindow, *ChargeIntervalEnergyBreakdown, error) {
	interval := &ChargeIntervalWindow{
		StartDate: timeZoneStringPointer(previous.EndDate),
		EndDate:   timeZoneStringPointer(current.StartDate),
	}

	distance := chargeIntervalDistance(previous, current)
	rangeBudgetKm := ratedRangeBudgetDistanceBetweenChargeRows(current, previous)
	rangeCompletion := chargeIntervalRatio(distance, rangeBudgetKm)
	consumedSOC := chargeIntervalConsumedSOC(previous, current)
	interval.RangeCompletion = rangeCompletion
	interval.ConsumedSOCPercent = consumedSOC

	if distance != nil {
		value := *distance
		if unitsLength == "mi" {
			value = kilometersToMiles(value)
		}
		interval.Distance = &value
	}
	if rangeBudgetKm != nil {
		value := *rangeBudgetKm
		if unitsLength == "mi" {
			value = kilometersToMiles(value)
		}
		interval.RatedRangeBudgetDistance = &value
	}

	breakdown, err := makeChargeIntervalEnergyBreakdown(current, previous, CarID, carEfficiency, rangeBudgetKm, rangeCompletion)
	return interval, breakdown, err
}

func makeChargeIntervalEnergyBreakdown(
	current chargeIntervalRow,
	previous chargeIntervalRow,
	CarID int,
	carEfficiency float64,
	rangeBudgetKm *float64,
	rangeCompletion *float64,
) (*ChargeIntervalEnergyBreakdown, error) {
	if rangeCompletion == nil {
		return nil, nil
	}

	if rangeBudgetKm == nil || carEfficiency <= 0 {
		drivingShare := *rangeCompletion
		if drivingShare > 1 {
			drivingShare = 1
		}
		if drivingShare < 0 {
			drivingShare = 0
		}
		return &ChargeIntervalEnergyBreakdown{
			RangeCompletion: rangeCompletion,
			DrivingShare:    drivingShare,
			ParkedShare:     1 - drivingShare,
			OtherShare:      0,
		}, nil
	}

	totalKWh := *rangeBudgetKm * carEfficiency
	if totalKWh <= 0 {
		return nil, nil
	}

	drivingKWh, err := fetchDrivingEnergyKWhBetweenChargeRows(CarID, previous, current)
	if err != nil {
		return nil, err
	}
	rangeDrivingFraction := *rangeCompletion
	if rangeDrivingFraction < 0 {
		rangeDrivingFraction = 0
	}
	if rangeDrivingFraction > 1 {
		rangeDrivingFraction = 1
	}
	if drivingKWh <= 0.0001 {
		drivingKWh = totalKWh * rangeDrivingFraction
	}

	parkedKWh := totalKWh - drivingKWh
	if parkedKWh < 0 {
		parkedKWh = 0
	}
	otherKWh := drivingKWh - totalKWh
	if otherKWh < 0 {
		otherKWh = 0
	}
	denom := drivingKWh + parkedKWh + otherKWh
	if denom <= 0 {
		return nil, nil
	}

	return &ChargeIntervalEnergyBreakdown{
		RangeCompletion: rangeCompletion,
		DrivingShare:    drivingKWh / denom,
		ParkedShare:     parkedKWh / denom,
		OtherShare:      otherKWh / denom,
		DrivingKWh:      &drivingKWh,
		ParkedKWh:       &parkedKWh,
		OtherKWh:        &otherKWh,
		TotalKWh:        &totalKWh,
	}, nil
}

func fetchDrivingEnergyKWhBetweenChargeRows(CarID int, previous chargeIntervalRow, current chargeIntervalRow) (float64, error) {
	if !previous.EndDate.Valid || !current.StartDate.Valid {
		return 0, nil
	}

	query := `
		WITH interval AS (
			SELECT $2::timestamp AS interval_start, $3::timestamp AS interval_end
		),
		filtered_drives AS (
			SELECT
				drives.start_date,
				drives.end_date,
				CASE
					WHEN (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency
					ELSE NULL
				END AS energy_consumed_net
			FROM drives
			LEFT JOIN cars ON drives.car_id = cars.id
			CROSS JOIN interval
			WHERE drives.car_id=$1
				AND drives.end_date IS NOT NULL
				AND drives.start_date < interval.interval_end
				AND drives.end_date > interval.interval_start
		)
		SELECT COALESCE(SUM(
			CASE
				WHEN energy_consumed_net > 0
					AND end_date > start_date
					AND LEAST(end_date, interval.interval_end) > GREATEST(start_date, interval.interval_start)
				THEN energy_consumed_net * (
					EXTRACT(EPOCH FROM (LEAST(end_date, interval.interval_end) - GREATEST(start_date, interval.interval_start)))
					/ NULLIF(EXTRACT(EPOCH FROM (end_date - start_date)), 0)
				)
				ELSE 0
			END
		), 0)
		FROM filtered_drives, interval;`

	var result sql.NullFloat64
	err := db.QueryRow(query, CarID, previous.EndDate.String, current.StartDate.String).Scan(&result)
	if err != nil {
		return 0, err
	}
	if !result.Valid {
		return 0, nil
	}
	return result.Float64, nil
}

func chargeIntervalDistance(previous chargeIntervalRow, current chargeIntervalRow) *float64 {
	if !previous.Odometer.Valid || !current.Odometer.Valid {
		return nil
	}
	value := current.Odometer.Float64 - previous.Odometer.Float64
	if value <= 0 {
		return nil
	}
	return &value
}

func chargeIntervalConsumedSOC(previous chargeIntervalRow, current chargeIntervalRow) *int {
	if !previous.EndBattery.Valid || !current.StartBattery.Valid {
		return nil
	}
	value := int(previous.EndBattery.Int64 - current.StartBattery.Int64)
	if value <= 0 {
		return nil
	}
	return &value
}

func chargeIntervalRatio(distance *float64, rangeBudget *float64) *float64 {
	if distance == nil || rangeBudget == nil || *rangeBudget <= 0 {
		return nil
	}
	value := *distance / *rangeBudget
	return &value
}

func ratedRangeBudgetDistanceBetweenChargeRows(current chargeIntervalRow, previous chargeIntervalRow) *float64 {
	if !previous.EndBattery.Valid || !current.StartBattery.Valid || previous.EndBattery.Int64 <= current.StartBattery.Int64 {
		return nil
	}

	prevEndRange := firstValidFloat(previous.EndRatedRange, previous.EndIdealRange)
	curStartRange := firstValidFloat(current.StartRatedRange, current.StartIdealRange)
	if prevEndRange != nil && curStartRange != nil && *prevEndRange > *curStartRange {
		value := *prevEndRange - *curStartRange
		if value > 0 {
			return &value
		}
	}

	if prevEndRange == nil || *prevEndRange <= 0 || previous.EndBattery.Int64 <= 0 {
		return nil
	}
	maxRange := *prevEndRange / (float64(previous.EndBattery.Int64) / 100.0)
	socConsumed := float64(previous.EndBattery.Int64 - current.StartBattery.Int64)
	value := maxRange * (socConsumed / 100.0)
	if value <= 0 {
		return nil
	}
	return &value
}

func firstValidFloat(values ...sql.NullFloat64) *float64 {
	for _, value := range values {
		if value.Valid {
			result := value.Float64
			return &result
		}
	}
	return nil
}
