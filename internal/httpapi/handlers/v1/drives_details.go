package v1

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// TeslaMateAPICarsDrivesDetailsV1 returns one drive with per-sample positions.
//
// @Summary      Drive details
// @Description  Returns one drive along with every position sample recorded for it.
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID    path      int  true  "TeslaMate cars.id"
// @Param        DriveID  path      int  true  "drives.id"
// @Success      200      {object}  dto.V1DriveDetailResponse
// @Failure      401      {object}  dto.ErrorEnvelope
// @Router       /api/v1/cars/{CarID}/drives/{DriveID} [get]
func (h *Handler) DrivesDetails(c *gin.Context) {

	// define error messages
	var (
		CarsDrivesDetailsError1 = "Unable to load drive."
		CarsDrivesDetailsError2 = "Unable to load drive details."
	)

	// getting CarID and DriveID param from URL
	CarID := convert.StrToInt(c.Param("CarID"))
	DriveID := convert.StrToInt(c.Param("DriveID"))

	// creating required vars
	var (
		CarName                       NullString
		drive                         dto.V1DriveDetail
		DriveDetailsData              []dto.V1DriveDetailPoint
		UnitsLength, UnitsTemperature string
	)

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
		WHERE drives.car_id=$1 AND end_date IS NOT NULL AND drives.id = $2;`
	row := h.db.QueryRowContext(c.Request.Context(), query, CarID, DriveID)

	// scanning row and putting values into the drive
	err := row.Scan(
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

	switch err {
	case sql.ErrNoRows:
		respond.HandleError(c, "TeslaMateAPICarsDrivesDetailsV1", "No rows were returned!", err.Error())
		return
	case nil:
		// nothing wrong.. continuing
		break
	default:
		respond.HandleError(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError1, err.Error())
		return
	}

	// converting values based of settings UnitsLength
	if UnitsLength == "mi" {
		drive.OdometerDetails.OdometerStart = convert.KilometersToMiles(drive.OdometerDetails.OdometerStart)
		drive.OdometerDetails.OdometerEnd = convert.KilometersToMiles(drive.OdometerDetails.OdometerEnd)
		drive.OdometerDetails.OdometerDistance = convert.KilometersToMiles(drive.OdometerDetails.OdometerDistance)
		drive.SpeedMax = int(convert.KilometersToMiles(float64(drive.SpeedMax)))
		drive.SpeedAvg = convert.KilometersToMiles(drive.SpeedAvg)
		drive.RangeIdeal.StartRange = convert.KilometersToMiles(drive.RangeIdeal.StartRange)
		drive.RangeIdeal.EndRange = convert.KilometersToMiles(drive.RangeIdeal.EndRange)
		drive.RangeIdeal.RangeDiff = convert.KilometersToMiles(drive.RangeIdeal.RangeDiff)
		drive.RangeRated.StartRange = convert.KilometersToMiles(drive.RangeRated.StartRange)
		drive.RangeRated.EndRange = convert.KilometersToMiles(drive.RangeRated.EndRange)
		drive.RangeRated.RangeDiff = convert.KilometersToMiles(drive.RangeRated.RangeDiff)
		if drive.ConsumptionNet != nil {
			*drive.ConsumptionNet = convert.KilometersToMiles(*drive.ConsumptionNet)
		}
	}
	// converting values based of settings UnitsTemperature
	if UnitsTemperature == "F" {
		drive.OutsideTempAvg = convert.CelsiusToFahrenheit(drive.OutsideTempAvg)
		drive.InsideTempAvg = convert.CelsiusToFahrenheit(drive.InsideTempAvg)
	}
	// adjusting to timezone differences from UTC to be userspecific
	drive.StartDate = h.timeInTZ(drive.StartDate)
	drive.EndDate = h.timeInTZ(drive.EndDate)

	// getting detailed drive data from database
	query = `
		 			SELECT
						id AS detail_id,
						date,
						latitude,
						longitude,
						COALESCE(speed, 0) AS speed,
						power,
						odometer,
						battery_level,
						usable_battery_level,
						elevation,
						inside_temp,
						outside_temp,
						is_climate_on,
						fan_status,
						driver_temp_setting,
						passenger_temp_setting,
						is_rear_defroster_on,
						is_front_defroster_on,
						est_battery_range_km,
						ideal_battery_range_km,
						rated_battery_range_km,
						battery_heater,
						battery_heater_on,
						battery_heater_no_power
		 			FROM positions
		 			WHERE drive_id = $1
		 			ORDER BY id ASC;`
	rows, err := h.db.QueryContext(c.Request.Context(), query, DriveID)

	// checking for errors in query
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
		return
	}

	// defer closing rows
	defer rows.Close()

	// looping through all results
	for rows.Next() {

		// creating drivedetails object based on struct
		drivedetails := dto.V1DriveDetailPoint{}

		// scanning row and putting values into the drive
		err = rows.Scan(
			&drivedetails.DetailID,
			&drivedetails.Date,
			&drivedetails.Latitude,
			&drivedetails.Longitude,
			&drivedetails.Speed,
			&drivedetails.Power,
			&drivedetails.Odometer,
			&drivedetails.BatteryLevel,
			&drivedetails.UsableBatteryLevel,
			&drivedetails.Elevation,
			&drivedetails.ClimateInfo.InsideTemp,
			&drivedetails.ClimateInfo.OutsideTemp,
			&drivedetails.ClimateInfo.IsClimateOn,
			&drivedetails.ClimateInfo.FanStatus,
			&drivedetails.ClimateInfo.DriverTempSetting,
			&drivedetails.ClimateInfo.PassengerTempSetting,
			&drivedetails.ClimateInfo.IsRearDefrosterOn,
			&drivedetails.ClimateInfo.IsFrontDefrosterOn,
			&drivedetails.BatteryInfo.EstBatteryRange,
			&drivedetails.BatteryInfo.IdealBatteryRange,
			&drivedetails.BatteryInfo.RatedBatteryRange,
			&drivedetails.BatteryInfo.BatteryHeater,
			&drivedetails.BatteryInfo.BatteryHeaterOn,
			&drivedetails.BatteryInfo.BatteryHeaterNoPower,
		)

		// converting values based of settings UnitsLength
		if UnitsLength == "mi" {
			drivedetails.Odometer = convert.KilometersToMiles(drivedetails.Odometer)
			drivedetails.Speed = int(convert.KilometersToMiles(float64(drivedetails.Speed)))
			drivedetails.BatteryInfo.EstBatteryRange = convert.KilometersToMilesNullable(drivedetails.BatteryInfo.EstBatteryRange)
			drivedetails.BatteryInfo.IdealBatteryRange = convert.KilometersToMilesNullable(drivedetails.BatteryInfo.IdealBatteryRange)
			drivedetails.BatteryInfo.RatedBatteryRange = convert.KilometersToMilesNullable(drivedetails.BatteryInfo.RatedBatteryRange)
		}
		// converting values based of settings UnitsTemperature
		if UnitsTemperature == "F" {
			drivedetails.ClimateInfo.InsideTemp = convert.CelsiusToFahrenheitNullable(drivedetails.ClimateInfo.InsideTemp)
			drivedetails.ClimateInfo.OutsideTemp = convert.CelsiusToFahrenheitNullable(drivedetails.ClimateInfo.OutsideTemp)
			drivedetails.ClimateInfo.DriverTempSetting = convert.CelsiusToFahrenheitNullable(drivedetails.ClimateInfo.DriverTempSetting)
			drivedetails.ClimateInfo.PassengerTempSetting = convert.CelsiusToFahrenheitNullable(drivedetails.ClimateInfo.PassengerTempSetting)
		}
		// adjusting to timezone differences from UTC to be userspecific
		drivedetails.Date = h.timeInTZ(drivedetails.Date)

		// checking for errors after scanning
		if err != nil {
			respond.HandleError(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
			return
		}

		// appending drive to drive
		DriveDetailsData = append(DriveDetailsData, drivedetails)
		drive.DriveDetails = DriveDetailsData
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
		return
	}

	//
	// build the data-blob
	jsonData := dto.V1DriveDetailResponse{
		Data: dto.V1DriveDetailData{
			Car: dto.Car{
				CarID:   CarID,
				CarName: CarName,
			},
			Drive: drive,
			TeslaMateUnits: dto.TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// return jsonData
	respond.HandleSuccess(c, "TeslaMateAPICarsDrivesDetailsV1", jsonData)
}
