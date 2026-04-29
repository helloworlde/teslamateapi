package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsChargesV1 func
func TeslaMateAPICarsChargesV1(c *gin.Context) {

	// define error messages
	var CarsChargesError1 = "Unable to load charges."
	var CarsChargesError2 = "Invalid date format."

	// getting CarID param from URL
	CarID := convertStringToInteger(c.Param("CarID"))
	// query options to modify query when collecting data
	ResultPage := convertStringToInteger(c.DefaultQuery("page", "1"))
	ResultShow := convertStringToInteger(c.DefaultQuery("show", "100"))
	limit := convertStringToInteger(c.DefaultQuery("limit", "0"))
	offset := convertStringToInteger(c.DefaultQuery("offset", "0"))
	sortRaw := strings.TrimSpace(c.DefaultQuery("sort", "-start_date"))
	_ = c.Query("include")

	// get startDate and endDate from query parameters
	parsedStartDate, err := parseDateParam(c.Query("startDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError2, err.Error())
		return
	}
	parsedEndDate, err := parseDateParam(c.Query("endDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError2, err.Error())
		return
	}

	var (
		CarName                       NullString
		ChargesData                   []ChargeListItemV1
		UnitsLength, UnitsTemperature string
	)

	if limit > 0 {
		ResultShow = limit
		ResultPage = 0
		if offset > 0 {
			ResultPage = offset
		}
	} else {
		// calculate offset based on page (page 0 is not possible, since first page is minimum 1)
		if ResultPage > 0 {
			ResultPage--
		} else {
			ResultPage = 0
		}
		ResultPage = (ResultPage * ResultShow)
	}

	orderBy := "start_date DESC"
	switch sortRaw {
	case "start_date":
		orderBy = "start_date ASC"
	case "-start_date":
		orderBy = "start_date DESC"
	case "duration":
		orderBy = "duration_min ASC"
	case "-duration":
		orderBy = "duration_min DESC"
	case "cost":
		orderBy = "cost ASC"
	case "-cost":
		orderBy = "cost DESC"
	case "energy":
		orderBy = "charge_energy_added ASC"
	case "-energy":
		orderBy = "charge_energy_added DESC"
	}

	// getting data from database
	query := `
		SELECT
			charging_processes.id AS charge_id,
			charging_processes.start_date,
			charging_processes.end_date,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, nullif(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS address,
			COALESCE(charge_energy_added, 0) AS charge_energy_added,
			COALESCE(GREATEST(charge_energy_used, charge_energy_added), 0) AS charge_energy_used,
			COALESCE(cost, 0) AS cost,
			start_ideal_range_km AS start_ideal_range,
			end_ideal_range_km AS end_ideal_range,
			start_rated_range_km AS start_rated_range,
			end_rated_range_km AS end_rated_range,
			start_battery_level,
			end_battery_level,
			duration_min,
			TO_CHAR((duration_min * INTERVAL '1 minute'), 'HH24:MI') as duration_str,
			outside_temp_avg,
			position.odometer as odometer,
			position.latitude,
			position.longitude,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name
		FROM charging_processes
		LEFT JOIN cars ON car_id = cars.id
		LEFT JOIN addresses address ON address_id = address.id
		LEFT JOIN positions position ON position_id = position.id
		LEFT JOIN geofences geofence ON geofence_id = geofence.id
		WHERE charging_processes.car_id=$1 AND charging_processes.end_date IS NOT NULL`

	// Parameters to be passed to the query
	var queryParams []any
	queryParams = append(queryParams, CarID)
	paramIndex := 2

	// Add date filtering if provided
	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND charging_processes.start_date >= $%d", paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND charging_processes.end_date <= $%d", paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}

	query += fmt.Sprintf(`
        ORDER BY %s
        LIMIT $%d OFFSET $%d;`, orderBy, paramIndex, paramIndex+1)

	queryParams = append(queryParams, ResultShow, ResultPage)

	rows, err := db.Query(query, queryParams...)

	// checking for errors in query
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
		return
	}

	// defer closing rows
	defer rows.Close()

	// looping through all results
	for rows.Next() {

		// creating charge object based on struct
		charge := ChargeListItemV1{}

		// scanning row and putting values into the charge
		err = rows.Scan(
			&charge.ChargeID,
			&charge.StartDate,
			&charge.EndDate,
			&charge.Address,
			&charge.ChargeEnergyAdded,
			&charge.ChargeEnergyUsed,
			&charge.Cost,
			&charge.RangeIdeal.StartRange,
			&charge.RangeIdeal.EndRange,
			&charge.RangeRated.StartRange,
			&charge.RangeRated.EndRange,
			&charge.BatteryDetails.StartBatteryLevel,
			&charge.BatteryDetails.EndBatteryLevel,
			&charge.DurationMin,
			&charge.DurationStr,
			&charge.OutsideTempAvg,
			&charge.Odometer,
			&charge.Latitude,
			&charge.Longitude,
			&UnitsLength,
			&UnitsTemperature,
			&CarName,
		)

		// converting values based of settings UnitsLength
		if UnitsLength == "mi" {
			charge.RangeIdeal.StartRange = kilometersToMiles(charge.RangeIdeal.StartRange)
			charge.RangeIdeal.EndRange = kilometersToMiles(charge.RangeIdeal.EndRange)
			charge.RangeRated.StartRange = kilometersToMiles(charge.RangeRated.StartRange)
			charge.RangeRated.EndRange = kilometersToMiles(charge.RangeRated.EndRange)
			charge.Odometer = kilometersToMiles(charge.Odometer)
		}
		// converting values based of settings UnitsTemperature
		if UnitsTemperature == "F" {
			charge.OutsideTempAvg = celsiusToFahrenheit(charge.OutsideTempAvg)
		}

		// adjusting to timezone differences from UTC to be userspecific
		charge.StartDate = getTimeInTimeZone(charge.StartDate)
		charge.EndDate = getTimeInTimeZone(charge.EndDate)

		// checking for errors after scanning
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
			return
		}

		// appending charge to ChargesData
		ChargesData = append(ChargesData, charge)
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
		return
	}

	jsonData := ChargesListV1Envelope{
		Data: ChargesListV1Data{
			Car: CarRefV1{
				CarID:   CarID,
				CarName: CarName,
			},
			Charges: ChargesData,
			TeslaMateUnits: UnitsLengthTempV1{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// return jsonData
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsChargesV1", jsonData)
}
