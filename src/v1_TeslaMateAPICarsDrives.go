package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/tobiasehlert/teslamateapi/src/dto"
)

// TeslaMateAPICarsDrivesV1 returns paginated drives for the car.
//
// @Summary      List drives
// @Description  Returns drives with odometer / battery / range aggregates.
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID        path   int     true   "TeslaMate cars.id"
// @Param        page         query  int     false  "1-indexed page"  default(1)
// @Param        show         query  int     false  "page size"       default(100)
// @Param        startDate    query  string  false  "RFC3339 lower bound"
// @Param        endDate      query  string  false  "RFC3339 upper bound"
// @Param        minDistance  query  number  false  "min distance (user units)"
// @Param        maxDistance  query  number  false  "max distance (user units)"
// @Success      200          {object}  dto.V1DrivesResponse
// @Failure      401          {object}  dto.ErrorEnvelope
// @Router       /api/v1/cars/{CarID}/drives [get]
func TeslaMateAPICarsDrivesV1(c *gin.Context) {

	// define error messages
	var CarsDrivesError1 = "Unable to load drives."
	var CarsDrivesError2 = "Invalid date format."

	// getting CarID param from URL
	CarID := convertStringToInteger(c.Param("CarID"))
	// query options to modify query when collecting data
	ResultPage := convertStringToInteger(c.DefaultQuery("page", "1"))
	ResultShow := convertStringToInteger(c.DefaultQuery("show", "100"))

	// get startDate and endDate from query parameters
	parsedStartDate, err := parseDateParam(c.Query("startDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError2, err.Error())
		return
	}
	parsedEndDate, err := parseDateParam(c.Query("endDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError2, err.Error())
		return
	}
	// get optional minDistance and maxDistance filters from query parameters
	minDistanceParam := c.Query("minDistance")
	minDistance := 0.0
	if minDistanceParam != "" {
		minDistance = convertStringToFloat(minDistanceParam)
		if minDistance < 0 {
			minDistance = 0
		}
	}
	maxDistanceParam := c.Query("maxDistance")
	maxDistance := 0.0
	if maxDistanceParam != "" {
		maxDistance = convertStringToFloat(maxDistanceParam)
		if maxDistance < 0 {
			maxDistance = 0
		}
	}

	// creating required vars
	var (
		CarName                       NullString
		DrivesData                    []dto.V1DriveListItem
		UnitsLength, UnitsTemperature string
	)

	// calculate offset based on page (page 0 is not possible, since first page is minimum 1)
	if ResultPage > 0 {
		ResultPage--
	} else {
		ResultPage = 0
	}
	ResultPage = (ResultPage * ResultShow)

	// getting data from database
	query := `
		SELECT
			drives.id AS drive_id,
			start_date,
			end_date,
			COALESCE(start_geofence.name, CONCAT_WS(', ', COALESCE(start_address.name, nullif(CONCAT_WS(' ', start_address.road, start_address.house_number), '')), start_address.city)) AS start_address,
			COALESCE(end_geofence.name, CONCAT_WS(', ', COALESCE(end_address.name, nullif(CONCAT_WS(' ', end_address.road, end_address.house_number), '')), end_address.city)) AS end_address,
			start_km,
			end_km,
			distance,
			duration_min,
			TO_CHAR((duration_min * INTERVAL '1 minute'), 'HH24:MI') as duration_str,
			speed_max,
			COALESCE(distance / NULLIF(duration_min, 0) * 60, 0) AS speed_avg,
			power_max,
			power_min,
			COALESCE(start_position.usable_battery_level, start_position.battery_level) as start_usable_battery_level,
			start_position.battery_level as start_battery_level,
			COALESCE(end_position.usable_battery_level, end_position.battery_level) as end_usable_battery_level,
			end_position.battery_level as end_battery_level,
			case when ( start_position.battery_level != start_position.usable_battery_level OR end_position.battery_level != end_position.usable_battery_level ) = true then true else false end  as reduced_range,
			duration_min > 1 AND distance > 1 AND ( start_position.usable_battery_level IS NULL OR end_position.usable_battery_level IS NULL OR ( end_position.battery_level - end_position.usable_battery_level ) = 0 ) as is_sufficiently_precise,
			start_ideal_range_km,
			end_ideal_range_km,
			COALESCE( NULLIF ( GREATEST ( start_ideal_range_km - end_ideal_range_km, 0 ), 0 ),0 ) as range_diff_ideal_km,
			start_rated_range_km,
			end_rated_range_km,
			COALESCE( NULLIF ( GREATEST ( start_rated_range_km - end_rated_range_km, 0 ), 0 ),0 ) as range_diff_rated_km,
			outside_temp_avg,
			inside_temp_avg,
			CASE 
				WHEN (start_rated_range_km - end_rated_range_km) > 0 
				THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency 
				ELSE NULL 
			END as energy_consumed_net,
			CASE 
				WHEN (duration_min > 1 AND distance > 1 AND ( start_position.usable_battery_level IS NULL OR end_position.usable_battery_level IS NULL OR ( end_position.battery_level - end_position.usable_battery_level ) = 0 )) AND NULLIF(distance, 0) IS NOT NULL
				THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / NULLIF(distance, 0) * 1000
				ELSE NULL 
			END as consumption_net,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name
		FROM drives
		LEFT JOIN cars ON car_id = cars.id
		LEFT JOIN addresses start_address ON start_address_id = start_address.id
		LEFT JOIN addresses end_address ON end_address_id = end_address.id
		LEFT JOIN positions start_position ON start_position_id = start_position.id
		LEFT JOIN positions end_position ON end_position_id = end_position.id
		LEFT JOIN geofences start_geofence ON start_geofence_id = start_geofence.id
		LEFT JOIN geofences end_geofence ON end_geofence_id = end_geofence.id
		WHERE drives.car_id=$1 AND end_date IS NOT NULL`

	// Parameters to be passed to the query
	var queryParams []any
	queryParams = append(queryParams, CarID)
	paramIndex := 2

	// Add date filtering if provided
	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND drives.start_date >= $%d", paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND drives.end_date <= $%d", paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}

	// Add minimum/maximum distance filtering if provided
	if minDistance > 0 || maxDistance > 0 {
		var unitsLength string
		err = db.QueryRowContext(c.Request.Context(), "SELECT unit_of_length FROM settings LIMIT 1").Scan(&unitsLength)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(
				c,
				"TeslaMateAPICarsDrivesV1",
				CarsDrivesError1,
				fmt.Sprintf("unable to retrieve unit_of_length from settings table: %v", err),
			)
			return
		}
		if unitsLength == "mi" {
			if minDistance > 0 {
				minDistance = milesToKilometers(minDistance)
			}
			if maxDistance > 0 {
				maxDistance = milesToKilometers(maxDistance)
			}
		}

		if minDistance > 0 {
			query += fmt.Sprintf(" AND distance >= $%d", paramIndex)
			queryParams = append(queryParams, minDistance)
			paramIndex++
		}
		if maxDistance > 0 {
			query += fmt.Sprintf(" AND distance <= $%d", paramIndex)
			queryParams = append(queryParams, maxDistance)
			paramIndex++
		}
	}

	query += fmt.Sprintf(`
        ORDER BY start_date DESC
        LIMIT $%d OFFSET $%d;`, paramIndex, paramIndex+1)

	queryParams = append(queryParams, ResultShow, ResultPage)

	rows, err := db.QueryContext(c.Request.Context(), query, queryParams...)

	// checking for errors in query
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError1, err.Error())
		return
	}

	// defer closing rows
	defer rows.Close()

	// looping through all results
	for rows.Next() {

		// creating drive object based on struct
		drive := dto.V1DriveListItem{}

		// scanning row and putting values into the drive
		err = rows.Scan(
			&drive.DriveID,
			&drive.StartDate,
			&drive.EndDate,
			&drive.StartAddress,
			&drive.EndAddress,
			&drive.OdometerDetails.OdometerStart,
			&drive.OdometerDetails.OdometerEnd,
			&drive.OdometerDetails.OdometerDistance,
			&drive.DurationMin,
			&drive.DurationStr,
			&drive.SpeedMax,
			&drive.SpeedAvg,
			&drive.PowerMax,
			&drive.PowerMin,
			&drive.BatteryDetails.StartUsableBatteryLevel,
			&drive.BatteryDetails.StartBatteryLevel,
			&drive.BatteryDetails.EndUsableBatteryLevel,
			&drive.BatteryDetails.EndBatteryLevel,
			&drive.BatteryDetails.ReducedRange,
			&drive.BatteryDetails.IsSufficientlyPrecise,
			&drive.RangeIdeal.StartRange,
			&drive.RangeIdeal.EndRange,
			&drive.RangeIdeal.RangeDiff,
			&drive.RangeRated.StartRange,
			&drive.RangeRated.EndRange,
			&drive.RangeRated.RangeDiff,
			&drive.OutsideTempAvg,
			&drive.InsideTempAvg,
			&drive.EnergyConsumedNet,
			&drive.ConsumptionNet,
			&UnitsLength,
			&UnitsTemperature,
			&CarName,
		)
		// Bail before any unit conversion: a Scan error leaves drive/UnitsLength
		// half-populated, and computing on those values silently produces garbage.
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError1, err.Error())
			return
		}

		// converting values based of settings UnitsLength
		if UnitsLength == "mi" {
			drive.OdometerDetails.OdometerStart = kilometersToMiles(drive.OdometerDetails.OdometerStart)
			drive.OdometerDetails.OdometerEnd = kilometersToMiles(drive.OdometerDetails.OdometerEnd)
			drive.OdometerDetails.OdometerDistance = kilometersToMiles(drive.OdometerDetails.OdometerDistance)
			drive.SpeedMax = int(kilometersToMiles(float64(drive.SpeedMax)))
			drive.SpeedAvg = kilometersToMiles(drive.SpeedAvg)
			drive.RangeIdeal.StartRange = kilometersToMiles(drive.RangeIdeal.StartRange)
			drive.RangeIdeal.EndRange = kilometersToMiles(drive.RangeIdeal.EndRange)
			drive.RangeIdeal.RangeDiff = kilometersToMiles(drive.RangeIdeal.RangeDiff)
			drive.RangeRated.StartRange = kilometersToMiles(drive.RangeRated.StartRange)
			drive.RangeRated.EndRange = kilometersToMiles(drive.RangeRated.EndRange)
			drive.RangeRated.RangeDiff = kilometersToMiles(drive.RangeRated.RangeDiff)
			if drive.ConsumptionNet != nil {
				*drive.ConsumptionNet = kilometersToMiles(*drive.ConsumptionNet)
			}
		}
		// converting values based of settings UnitsTemperature
		if UnitsTemperature == "F" {
			drive.OutsideTempAvg = celsiusToFahrenheit(drive.OutsideTempAvg)
			drive.InsideTempAvg = celsiusToFahrenheit(drive.InsideTempAvg)
		}

		// adjusting to timezone differences from UTC to be userspecific
		drive.StartDate = getTimeInTimeZone(drive.StartDate)
		drive.EndDate = getTimeInTimeZone(drive.EndDate)

		// appending drive to DrivesData
		DrivesData = append(DrivesData, drive)
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsDrivesV1", CarsDrivesError1, err.Error())
		return
	}

	//
	// build the data-blob
	jsonData := dto.V1DrivesResponse{
		Data: dto.V1DrivesData{
			Car: dto.Car{
				CarID:   CarID,
				CarName: CarName,
			},
			Drives: DrivesData,
			TeslaMateUnits: dto.TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// return jsonData
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsDrivesV1", jsonData)
}
