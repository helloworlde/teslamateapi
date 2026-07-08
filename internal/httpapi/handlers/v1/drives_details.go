package v1

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// TeslaMateAPICarsDrivesDetailsV1 returns one drive with per-sample positions.
//
// @Summary      Drive details
// @Description  Returns one drive with position samples. Use sample/include_route query parameters to control detail density.
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID    path      int  true  "TeslaMate cars.id"
// @Param        DriveID  path      int  true  "drives.id"
// @Param        sample      query     string  false  "行程明细下采样：auto（默认）、full、every_5s、every_30s"  Enums(auto, full, every_5s, every_30s)
// @Param        max_points  query     int     false  "auto 模式目标返回的明细点数，范围 100-3000，默认 800；状态变化点和路线边界点会强制保留"
// @Param        include_route  query  bool    false  "设为 false 可在响应中省略 drive_details（行程轨迹）"
// @Success      200      {object}  dto.V1DriveDetailResponse
// @Failure      401      {object}  dto.ErrorEnvelope
// @Router       /api/v1/cars/{CarID}/drives/{DriveID} [get]
func (h *Handler) DrivesDetails(c *gin.Context) {

	// define error messages
	const handler = "TeslaMateAPICarsDrivesDetailsV1"
	var (
		CarsDrivesDetailsError1 = "Unable to load drive."
		CarsDrivesDetailsError2 = "Unable to load drive details."
	)

	// getting CarID and DriveID param from URL
	CarID, ok := requirePositiveIntParam(c, handler, "CarID", c.Param("CarID"))
	if !ok {
		return
	}
	DriveID, ok := requirePositiveIntParam(c, handler, "DriveID", c.Param("DriveID"))
	if !ok {
		return
	}

	// creating required vars
	var (
		CarName                       NullString
		drive                         dto.V1DriveDetail
		DriveDetailsData              []dto.V1DriveDetailPoint
		UnitsLength, UnitsTemperature string
	)

	// getting data from database
	query := fmt.Sprintf(`
		WITH charge_price AS (
			SELECT CASE
				WHEN SUM(charge_energy_added) > 0
				THEN COALESCE(SUM(cost), 0) / NULLIF(SUM(charge_energy_added), 0)
				ELSE NULL
			END AS cost_per_kwh
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL
		)
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
			%[1]s as energy_consumed_net,
			CASE
				WHEN (start_rated_range_km - end_rated_range_km) > 0 AND NULLIF(distance, 0) IS NOT NULL
				THEN (start_rated_range_km - end_rated_range_km) * cars.efficiency / NULLIF(distance, 0) * 1000
				ELSE NULL
			END as consumption_net,
			CASE
				WHEN (start_rated_range_km - end_rated_range_km) > 0 AND distance > 0
				THEN distance / (start_rated_range_km - end_rated_range_km) * 100
				ELSE NULL
			END as range_achievement_pct,
			%[2]s as estimated_usage_cost,
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
		CROSS JOIN charge_price
		WHERE drives.car_id=$1 AND end_date IS NOT NULL AND drives.id = $2;`, v1DriveEnergyConsumedNetSQL, v1EstimatedUsageCostSQL)
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
		&drive.RangeAchievementPct,
		&drive.EstimatedUsageCost,
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
			*drive.ConsumptionNet = convert.WhPerKmToWhPerMile(*drive.ConsumptionNet)
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

	includeRoute := true
	if v := c.Query("include_route"); v == "false" || v == "0" {
		includeRoute = false
	}
	if !includeRoute {
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
		respond.HandleSuccess(c, "TeslaMateAPICarsDrivesDetailsV1", jsonData)
		return
	}

	// getting detailed drive data from database
	query, detailArgs := driveDetailsQuery(DriveID, c.DefaultQuery("sample", "auto"), detailMaxPoints(c.Query("max_points")))
	rows, err := h.db.QueryContext(c.Request.Context(), query, detailArgs...)

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
		// Bail before any unit conversion: a Scan error leaves drivedetails
		// half-populated, and computing on those values silently produces garbage.
		if err != nil {
			respond.HandleError(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
			return
		}

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
