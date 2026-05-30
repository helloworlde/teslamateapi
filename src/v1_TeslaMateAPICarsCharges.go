package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsChargesV1 returns the per-session charging history.
//
// In addition to the original v1 fields, the response now includes
// charger-shape aggregates derived from the `charges` time-series table:
//
//   - fast_charger_present   bool   — any sample during the session was DC fast
//   - fast_charger_brand     string — first non-null brand (Tesla / etc.)
//   - fast_charger_type      string — first non-null type  (Combo / etc.)
//   - peak_charger_power     int    — max charger_power (kW) over the session
//   - peak_charger_voltage   int    — max charger_voltage (V) over the session
//   - charger_phases         int    — first non-null phase count (1/2/3); null ⇒ DC
//   - conn_charge_cable      string — first non-null cable type (IEC / SAE / GB / GB_AC / GB_DC / "<invalid>" for DC)
//   - pilot_current_max      int    — peak J1772/IEC pilot current advertised by the cable
//   - charge_type            string — derived: "dc" | "ac" | "unknown"
//   - is_tesla_charger       bool   — derived: brand == "Tesla"
//
// charge_type / is_tesla_charger are derived server-side so all clients
// (iOS / web / future TUI) read the same answer for edge cases like
// "<invalid>" cable strings or third-party DC chargers reporting
// brand=`<invalid>`.
//
// All five "describes the charger" columns are session-constant in
// practice (AC session: phases=3 / cable=IEC throughout; DC session:
// phases=NULL / cable='<invalid>' throughout, brand/type set after
// handshake). A single LATERAL pass picking the first non-null value
// ordered by sample time is sufficient.
func TeslaMateAPICarsChargesV1(c *gin.Context) {

	// define error messages
	var CarsChargesError1 = "Unable to load charges."
	var CarsChargesError2 = "Invalid date format."

	// getting CarID param from URL
	CarID := convertStringToInteger(c.Param("CarID"))
	// query options to modify query when collecting data
	ResultPage := convertStringToInteger(c.DefaultQuery("page", "1"))
	ResultShow := convertStringToInteger(c.DefaultQuery("show", "100"))

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

	// creating structs for /cars/<CarID>/charges
	// Car struct - child of Data
	type Car struct {
		CarID   int        `json:"car_id"`   // smallint
		CarName NullString `json:"car_name"` // text (nullable)
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
	// Charges struct - child of Data
	type Charges struct {
		ChargeID           int            `json:"charge_id"`
		StartDate          string         `json:"start_date"`
		EndDate            string         `json:"end_date"`
		Address            string         `json:"address"`
		ChargeEnergyAdded  float64        `json:"charge_energy_added"`
		ChargeEnergyUsed   float64        `json:"charge_energy_used"`
		Cost               float64        `json:"cost"`
		DurationMin        int            `json:"duration_min"`
		DurationStr        string         `json:"duration_str"`
		BatteryDetails     BatteryDetails `json:"battery_details"`
		RangeIdeal         PreferredRange `json:"range_ideal"`
		RangeRated         PreferredRange `json:"range_rated"`
		OutsideTempAvg     float64        `json:"outside_temp_avg"`
		Odometer           float64        `json:"odometer"`
		Latitude           float64        `json:"latitude"`
		Longitude          float64        `json:"longitude"`
		FastChargerPresent bool           `json:"fast_charger_present"`
		FastChargerBrand   NullString     `json:"fast_charger_brand"`
		FastChargerType    NullString     `json:"fast_charger_type"`
		PeakChargerPower   int            `json:"peak_charger_power"`
		PeakChargerVoltage int            `json:"peak_charger_voltage"`
		ChargerPhases      NullInt64      `json:"charger_phases"`
		ConnChargeCable    NullString     `json:"conn_charge_cable"`
		PilotCurrentMax    NullInt64      `json:"pilot_current_max"`
		ChargeType         string         `json:"charge_type"`
		IsTeslaCharger     bool           `json:"is_tesla_charger"`
	}
	// TeslaMateUnits struct - child of Data
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`      // string
		UnitsTemperature string `json:"unit_of_temperature"` // string
	}
	// Data struct - child of JSONData
	type Data struct {
		Car            Car            `json:"car"`
		Charges        []Charges      `json:"charges"`
		TeslaMateUnits TeslaMateUnits `json:"units"`
	}
	// JSONData struct - main
	type JSONData struct {
		Data Data `json:"data"`
	}

	// creating required vars
	var (
		CarName                       NullString
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
	//
	// `agg` LATERAL aggregates the per-sample charges time-series into one
	// row per session. See doc-comment above for column semantics.
	query := `
		SELECT
			charging_processes.id AS charge_id,
			charging_processes.start_date,
			charging_processes.end_date,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, nullif(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS address,
			COALESCE(charging_processes.charge_energy_added, 0) AS charge_energy_added,
			COALESCE(GREATEST(charging_processes.charge_energy_used, charging_processes.charge_energy_added), 0) AS charge_energy_used,
			COALESCE(charging_processes.cost, 0) AS cost,
			charging_processes.start_ideal_range_km AS start_ideal_range,
			charging_processes.end_ideal_range_km AS end_ideal_range,
			charging_processes.start_rated_range_km AS start_rated_range,
			charging_processes.end_rated_range_km AS end_rated_range,
			charging_processes.start_battery_level,
			charging_processes.end_battery_level,
			charging_processes.duration_min,
			TO_CHAR((charging_processes.duration_min * INTERVAL '1 minute'), 'HH24:MI') as duration_str,
			charging_processes.outside_temp_avg,
			COALESCE(position.odometer, 0) as odometer,
			COALESCE(position.latitude, 0) as latitude,
			COALESCE(position.longitude, 0) as longitude,
			COALESCE(agg.fast_charger_present, false) as fast_charger_present,
			agg.fast_charger_brand,
			agg.fast_charger_type,
			COALESCE(agg.peak_charger_power, 0) as peak_charger_power,
			COALESCE(agg.peak_charger_voltage, 0) as peak_charger_voltage,
			agg.charger_phases,
			agg.conn_charge_cable,
			agg.pilot_current_max,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name
		FROM charging_processes
		LEFT JOIN cars ON charging_processes.car_id = cars.id
		LEFT JOIN addresses address ON charging_processes.address_id = address.id
		LEFT JOIN positions position ON charging_processes.position_id = position.id
		LEFT JOIN geofences geofence ON charging_processes.geofence_id = geofence.id
		LEFT JOIN LATERAL (
			SELECT
				bool_or(fast_charger_present) as fast_charger_present,
				MAX(charger_power) as peak_charger_power,
				MAX(charger_voltage) as peak_charger_voltage,
				MAX(charger_pilot_current) as pilot_current_max,
				(array_agg(fast_charger_brand ORDER BY date) FILTER (WHERE fast_charger_present AND fast_charger_brand IS NOT NULL))[1] as fast_charger_brand,
				(array_agg(fast_charger_type  ORDER BY date) FILTER (WHERE fast_charger_present AND fast_charger_type  IS NOT NULL))[1] as fast_charger_type,
				(array_agg(charger_phases    ORDER BY date) FILTER (WHERE charger_phases    IS NOT NULL))[1] as charger_phases,
				(array_agg(conn_charge_cable ORDER BY date) FILTER (WHERE conn_charge_cable IS NOT NULL AND conn_charge_cable <> '<invalid>'))[1] as conn_charge_cable
			FROM charges
			WHERE charges.charging_process_id = charging_processes.id
		) agg ON true
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
		charge := Charges{}

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
			&charge.FastChargerPresent,
			&charge.FastChargerBrand,
			&charge.FastChargerType,
			&charge.PeakChargerPower,
			&charge.PeakChargerVoltage,
			&charge.ChargerPhases,
			&charge.ConnChargeCable,
			&charge.PilotCurrentMax,
			&UnitsLength,
			&UnitsTemperature,
			&CarName,
		)

		// checking for errors after scanning
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
			return
		}

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

		charge.ChargeType = deriveChargeType(charge.FastChargerPresent, charge.ChargerPhases, charge.ConnChargeCable)
		charge.IsTeslaCharger = charge.FastChargerBrand == "Tesla"

		// appending charge to ChargesData
		ChargesData = append(ChargesData, charge)
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesV1", CarsChargesError1, err.Error())
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
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsChargesV1", jsonData)
}

// deriveChargeType returns "dc" / "ac" / "unknown" given what we observed
// during the session. Rules:
//
//   - fast_charger_present=true                              ⇒ dc
//   - charger_phases ∈ {1, 2, 3}                             ⇒ ac
//   - conn_charge_cable IEC / SAE / GB / GB_AC               ⇒ ac
//   - conn_charge_cable GB_DC                                ⇒ dc (last-resort
//                          when fast_charger_present somehow stayed false)
//   - conn_charge_cable == "<invalid>" with fast unobserved  ⇒ unknown
//                          (TeslaMate emits "<invalid>" for both old DC sessions
//                          and pre-handshake samples; without phase data we can't
//                          decide)
//   - everything else                                        ⇒ unknown
func deriveChargeType(fast bool, phases NullInt64, cable NullString) string {
	if fast {
		return "dc"
	}
	if phases.Valid && phases.Int64 >= 1 && phases.Int64 <= 3 {
		return "ac"
	}
	// Cable string vocabulary varies by region: vanilla TeslaMate uses
	// IEC / SAE / GB; CN deployments commonly see GB_AC / GB_DC variants.
	switch string(cable) {
	case "IEC", "SAE", "GB", "GB_AC":
		return "ac"
	case "GB_DC":
		return "dc"
	}
	return "unknown"
}
