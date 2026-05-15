package v1

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
	"github.com/tobiasehlert/teslamateapi/internal/conv"
	"github.com/tobiasehlert/teslamateapi/internal/nullx"
)

// TeslaMateAPICarsChargesV1 godoc
//
// @Summary 车辆充电会话列表
// @Tags v1
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param page query int false "结果页码（默认 1）"
// @Param show query int false "每页大小（默认 100）"
// @Param startDate query string false "筛选起始日期"
// @Param endDate query string false "筛选结束日期"
// @Success 200 {object} V1JSONEnvelope
// @Failure 200 {object} V1ErrorEnvelope
// @Router /v1/cars/{CarID}/charges [get]
func TeslaMateAPICarsChargesV1(c *gin.Context) {

	// define error messages
	var CarsChargesError1 = "Unable to load charges."
	var CarsChargesError2 = "Invalid date format."

	// getting CarID param from URL
	CarID := apicommon.ConvertStringToInteger(c.Param("CarID"))
	// query options to modify query when collecting data
	ResultPage := apicommon.ConvertStringToInteger(c.DefaultQuery("page", "1"))
	ResultShow := apicommon.ConvertStringToInteger(c.DefaultQuery("show", "100"))

	// get startDate and endDate from query parameters
	parsedStartDate, err := apicommon.ParseDateParam(c.Query("startDate"))
	if err != nil {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError2, err.Error())
		return
	}
	parsedEndDate, err := apicommon.ParseDateParam(c.Query("endDate"))
	if err != nil {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError2, err.Error())
		return
	}

	// creating structs for /cars/<CarID>/charges
	// Car struct - child of Data
	type Car struct {
		CarID   int          `json:"car_id"`   // smallint
		CarName nullx.String `json:"car_name"` // text (nullable)
	}
	// BatteryDetails struct - child of Charges
	type BatteryDetails struct {
		StartBatteryLevel int `json:"start_battery_level"` // int
		EndBatteryLevel   int `json:"end_battery_level"`   // int
	}
	// PreferredRange struct - child of Charges
	type PreferredRange struct {
		StartRange float64 `json:"start_range"` // float64
		EndRange   float64 `json:"end_range"`   // float64
	}
	// Geofence struct - child of Charges (added)
	type Geofence struct {
		ID   int    `json:"id"`   // ID
		Name string `json:"name"` // 名称
	}
	// Charges struct - child of Data
	type Charges struct {
		ChargeID               int            `json:"charge_id"`                           // int
		StartDate              string         `json:"start_date"`                          // string
		EndDate                string         `json:"end_date"`                            // string
		Address                string         `json:"address"`                             // string
		ChargeEnergyAdded      float64        `json:"charge_energy_added"`                 // float64
		ChargeEnergyUsed       float64        `json:"charge_energy_used"`                  // float64
		Cost                   float64        `json:"cost"`                                // float64
		DurationMin            int            `json:"duration_min"`                        // int
		DurationStr            string         `json:"duration_str"`                        // string
		BatteryDetails         BatteryDetails `json:"battery_details"`                     // BatteryDetails
		RangeIdeal             PreferredRange `json:"range_ideal"`                         // PreferredRange
		RangeRated             PreferredRange `json:"range_rated"`                         // PreferredRange
		OutsideTempAvg         float64        `json:"outside_temp_avg"`                    // float64
		Odometer               float64        `json:"odometer"`                            // float64
		Latitude               float64        `json:"latitude"`                            // float64
		Longitude              float64        `json:"longitude"`                           // float64
		Geofence               *Geofence      `json:"geofence,omitempty"`                  // struct (added)
		Connection             string         `json:"connection,omitempty"`                // "ac" | "dc" (added)
		FastChargerBrand       nullx.String   `json:"fast_charger_brand,omitempty"`        // text (added)
		FastChargerType        nullx.String   `json:"fast_charger_type,omitempty"`         // text (added)
		ChargerPilotCurrentMax nullx.Int64    `json:"charger_pilot_current_max,omitempty"` // int (added)
		CostPerKwh             nullx.Float64  `json:"cost_per_kwh,omitempty"`              // float (added)
		ChargingEfficiency     nullx.Float64  `json:"charging_efficiency,omitempty"`       // float (added)
		EnergyPerHour          nullx.Float64  `json:"energy_per_hour,omitempty"`           // float (added)
		IsComplete             bool           `json:"is_complete"`                         // bool (added)
	}
	// TeslaMateUnits struct - child of Data
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`      // string
		UnitsTemperature string `json:"unit_of_temperature"` // string
	}
	// Data struct - child of JSONData
	type Data struct {
		Car            Car            `json:"car"` // 车辆
		Charges        []Charges      `json:"charges"`
		TeslaMateUnits TeslaMateUnits `json:"units"` // 单位
	}
	// JSONData struct - main
	type JSONData struct {
		Data Data `json:"data"` // 响应数据
	}

	// creating required vars
	var (
		CarName                       nullx.String
		ChargesData                   []Charges
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
			cars.name,
			geofence.id AS geofence_id,
			geofence.name AS geofence_name,
			(SELECT CASE WHEN MAX(charger_phases) IS NULL THEN 'dc' ELSE 'ac' END FROM charges WHERE charging_process_id = charging_processes.id) AS connection,
			(SELECT MAX(fast_charger_brand) FROM charges WHERE charging_process_id = charging_processes.id AND fast_charger_brand IS NOT NULL AND fast_charger_brand <> '<invalid>') AS fast_charger_brand,
			(SELECT MAX(fast_charger_type) FROM charges WHERE charging_process_id = charging_processes.id AND fast_charger_type IS NOT NULL AND fast_charger_type <> '<invalid>') AS fast_charger_type,
			(SELECT MAX(charger_pilot_current) FROM charges WHERE charging_process_id = charging_processes.id) AS charger_pilot_current_max,
			CASE WHEN COALESCE(charge_energy_added, 0) > 0 AND COALESCE(cost, 0) > 0 THEN cost / charge_energy_added ELSE NULL END AS cost_per_kwh,
			CASE WHEN COALESCE(GREATEST(charge_energy_used, charge_energy_added), 0) > 0 THEN charge_energy_added / GREATEST(charge_energy_used, charge_energy_added) ELSE NULL END AS charging_efficiency,
			CASE WHEN duration_min > 0 THEN charge_energy_added / (duration_min / 60.0) ELSE NULL END AS energy_per_hour,
			(charging_processes.end_date IS NOT NULL) AS is_complete
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
        ORDER BY start_date DESC
        LIMIT $%d OFFSET $%d;`, paramIndex, paramIndex+1)

	queryParams = append(queryParams, ResultShow, ResultPage)

	rows, err := apicommon.DB.Query(query, queryParams...)

	// checking for errors in query
	if err != nil {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
		return
	}

	// defer closing rows
	defer rows.Close()

	// looping through all results
	for rows.Next() {

		// creating charge object based on struct
		charge := Charges{}
		var (
			geofenceID   sql.NullInt64
			geofenceName sql.NullString
			connection   sql.NullString
		)

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
			&geofenceID,
			&geofenceName,
			&connection,
			&charge.FastChargerBrand,
			&charge.FastChargerType,
			&charge.ChargerPilotCurrentMax,
			&charge.CostPerKwh,
			&charge.ChargingEfficiency,
			&charge.EnergyPerHour,
			&charge.IsComplete,
		)
		if geofenceID.Valid && geofenceName.Valid {
			charge.Geofence = &Geofence{ID: int(geofenceID.Int64), Name: geofenceName.String}
		}
		if connection.Valid {
			charge.Connection = connection.String
		}

		// converting values based of settings UnitsLength
		if UnitsLength == "mi" {
			charge.RangeIdeal.StartRange = conv.KmToMi(charge.RangeIdeal.StartRange)
			charge.RangeIdeal.EndRange = conv.KmToMi(charge.RangeIdeal.EndRange)
			charge.RangeRated.StartRange = conv.KmToMi(charge.RangeRated.StartRange)
			charge.RangeRated.EndRange = conv.KmToMi(charge.RangeRated.EndRange)
			charge.Odometer = conv.KmToMi(charge.Odometer)
		}
		// converting values based of settings UnitsTemperature
		if UnitsTemperature == "F" {
			charge.OutsideTempAvg = conv.CelsiusToFahrenheit(charge.OutsideTempAvg)
		}

		// adjusting to timezone differences from UTC to be userspecific
		charge.StartDate = apicommon.GetTimeInTimeZone(charge.StartDate)
		charge.EndDate = apicommon.GetTimeInTimeZone(charge.EndDate)

		// checking for errors after scanning
		if err != nil {
			apicommon.HandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
			return
		}

		// appending charge to ChargesData
		ChargesData = append(ChargesData, charge)
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
		return
	}

	//
	// build the data-blob
	jsonData := JSONData{
		Data{
			Car: Car{
				CarID:   CarID,
				CarName: CarName,
			},
			Charges: ChargesData,
			TeslaMateUnits: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// return jsonData
	apicommon.HandleSuccessResponse(c, "TeslaMateAPICarsChargesV1", jsonData)
}
