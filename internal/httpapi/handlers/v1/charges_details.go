package v1

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// TeslaMateAPICarsChargesDetailsV1 returns one charging session with samples.
//
// @Summary      Charge details
// @Description  Returns one charge session with charge samples. Use sample/include_details query parameters to control detail density.
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID     path      int  true  "TeslaMate cars.id"
// @Param        ChargeID  path      int  true  "charging_processes.id"
// @Param        sample      query     string  false  "充电明细下采样：auto（默认）、full、every_5s、every_30s"  Enums(auto, full, every_5s, every_30s)
// @Param        max_points  query     int     false  "auto 模式目标返回的明细点数，范围 100-3000，默认 800；状态变化点会强制保留"
// @Param        include_details  query  bool  false  "设为 false 可在响应中省略 charge_details"
// @Success      200       {object}  dto.V1ChargeDetailResponse
// @Failure      401       {object}  dto.ErrorEnvelope
// @Router       /api/v1/cars/{CarID}/charges/{ChargeID} [get]
func (h *Handler) ChargesDetails(c *gin.Context) {

	// define error messages
	const handler = "TeslaMateAPICarsChargesDetailsV1"
	var (
		CarsChargesDetailsError1 = "Unable to load charge."
		CarsChargesDetailsError2 = "Unable to load charge details."
	)

	// getting CarID and ChargeID param from URL
	CarID, ok := requirePositiveIntParam(c, handler, "CarID", c.Param("CarID"))
	if !ok {
		return
	}
	ChargeID, ok := requirePositiveIntParam(c, handler, "ChargeID", c.Param("ChargeID"))
	if !ok {
		return
	}

	// creating required vars
	var (
		CarName                       NullString
		charge                        dto.V1ChargeDetail
		ChargeDetailsData             []dto.V1ChargeDetailItem
		UnitsLength, UnitsTemperature string
	)

	// getting data from database
	query := `
		SELECT
			charging_processes.id AS charge_id,
			charging_processes.start_date,
			charging_processes.end_date,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, nullif(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS address,
			COALESCE(charging_processes.charge_energy_added, 0) AS charge_energy_added,
			COALESCE(GREATEST(charge_energy_used, charging_processes.charge_energy_added), 0) AS charge_energy_used,
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
		LEFT JOIN charges ON charging_processes.id = charges.id
		WHERE charging_processes.car_id=$1 AND charging_processes.id=$2 AND charging_processes.end_date IS NOT NULL
		ORDER BY start_date DESC;`
	row := h.db.QueryRowContext(c.Request.Context(), query, CarID, ChargeID)

	// scanning row and putting values into the charge
	err := row.Scan(
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

	switch err {
	case sql.ErrNoRows:
		respond.HandleError(c, "TeslaMateAPICarsChargesDetailsV1", "No rows were returned!", err.Error())
		return
	case nil:
		// nothing wrong.. continuing
		break
	default:
		respond.HandleError(c, "TeslaMateAPICarsChargesDetailsV1", CarsChargesDetailsError1, err.Error())
		return
	}

	// converting values based of settings UnitsLength
	if UnitsLength == "mi" {
		charge.RangeIdeal.StartRange = convert.KilometersToMiles(charge.RangeIdeal.StartRange)
		charge.RangeIdeal.EndRange = convert.KilometersToMiles(charge.RangeIdeal.EndRange)
		charge.RangeRated.StartRange = convert.KilometersToMiles(charge.RangeRated.StartRange)
		charge.RangeRated.EndRange = convert.KilometersToMiles(charge.RangeRated.EndRange)
		charge.Odometer = convert.KilometersToMiles(charge.Odometer)
	}
	// converting values based of settings UnitsTemperature
	if UnitsTemperature == "F" {
		charge.OutsideTempAvg = convert.CelsiusToFahrenheit(charge.OutsideTempAvg)
	}

	// adjusting to timezone differences from UTC to be userspecific
	charge.StartDate = h.timeInTZ(charge.StartDate)
	charge.EndDate = h.timeInTZ(charge.EndDate)

	includeDetails := true
	if v := c.Query("include_details"); v == "false" || v == "0" {
		includeDetails = false
	}
	if !includeDetails {
		jsonData := dto.V1ChargeDetailResponse{
			Data: dto.V1ChargeDetailData{
				Car: dto.Car{
					CarID:   CarID,
					CarName: CarName,
				},
				Charge: charge,
				TeslaMateUnits: dto.TeslaMateUnits{
					UnitsLength:      UnitsLength,
					UnitsTemperature: UnitsTemperature,
				},
			},
		}
		respond.HandleSuccess(c, "TeslaMateAPICarsChargesDetailsV1", jsonData)
		return
	}

	// getting detailed charge data from database
	query, detailArgs := chargeDetailsQuery(ChargeID, c.DefaultQuery("sample", "auto"), detailMaxPoints(c.Query("max_points")))
	rows, err := h.db.QueryContext(c.Request.Context(), query, detailArgs...)

	// checking for errors in query
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsChargesDetailsV1", CarsChargesDetailsError2, err.Error())
		return
	}

	// defer closing rows
	defer rows.Close()

	// looping through all results
	for rows.Next() {

		// creating chargedetails object based on struct
		chargedetails := dto.V1ChargeDetailItem{}

		// scanning row and putting values into the drive
		err = rows.Scan(
			&chargedetails.DetailID,
			&chargedetails.Date,
			&chargedetails.BatteryLevel,
			&chargedetails.UsableBatteryLevel,
			&chargedetails.ChargeEnergyAdded,
			&chargedetails.NotEnoughPowerToHeat,
			&chargedetails.ChargerDetails.ChargerActualCurrent,
			&chargedetails.ChargerDetails.ChargerPhases,
			&chargedetails.ChargerDetails.ChargerPilotCurrent,
			&chargedetails.ChargerDetails.ChargerPower,
			&chargedetails.ChargerDetails.ChargerVoltage,
			&chargedetails.BatteryInfo.IdealBatteryRange,
			&chargedetails.BatteryInfo.RatedBatteryRange,
			&chargedetails.BatteryInfo.BatteryHeater,
			&chargedetails.BatteryInfo.BatteryHeaterOn,
			&chargedetails.BatteryInfo.BatteryHeaterNoPower,
			&chargedetails.ConnChargeCable,
			&chargedetails.FastChargerInfo.FastChargerPresent,
			&chargedetails.FastChargerInfo.FastChargerBrand,
			&chargedetails.FastChargerInfo.FastChargerType,
			&chargedetails.OutsideTemp,
		)

		// converting values based of settings UnitsLength
		if UnitsLength == "mi" {
			chargedetails.BatteryInfo.IdealBatteryRange = convert.KilometersToMiles(chargedetails.BatteryInfo.IdealBatteryRange)
			chargedetails.BatteryInfo.RatedBatteryRange = convert.KilometersToMiles(chargedetails.BatteryInfo.RatedBatteryRange)

		}
		// converting values based of settings UnitsTemperature
		if UnitsTemperature == "F" {
			chargedetails.OutsideTemp = convert.CelsiusToFahrenheit(chargedetails.OutsideTemp)
		}
		// adjusting to timezone differences from UTC to be userspecific
		chargedetails.Date = h.timeInTZ(chargedetails.Date)

		// checking for errors after scanning
		if err != nil {
			respond.HandleError(c, "TeslaMateAPICarsChargesDetailsV1", CarsChargesDetailsError2, err.Error())
			return
		}

		// appending drive to ChargeData
		ChargeDetailsData = append(ChargeDetailsData, chargedetails)
		charge.ChargeDetails = ChargeDetailsData
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsChargesDetailsV1", CarsChargesDetailsError2, err.Error())
		return
	}

	//
	// build the data-blob
	jsonData := dto.V1ChargeDetailResponse{
		Data: dto.V1ChargeDetailData{
			Car: dto.Car{
				CarID:   CarID,
				CarName: CarName,
			},
			Charge: charge,
			TeslaMateUnits: dto.TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// return jsonData
	respond.HandleSuccess(c, "TeslaMateAPICarsChargesDetailsV1", jsonData)
}
