package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsParkingsV2 derives parking sessions from gaps between
// adjacent drives. The N-th parking covers the window
// (drives[N].end_date, drives[N+1].start_date) on the same car, where drives
// are ordered by start_date ASC.
//
// parking_id = drives[N].id (the drive that *preceded* the parking session).
// This is stable, monotonic, and uniquely identifies the session even after
// new drives are appended.
//
// Energy consumed during parking is approximated by SOC drop * efficiency *
// usable nominal capacity. We expose:
//   - start_battery_level / end_battery_level (% SOC, nullable when missing)
//   - usable_battery_drop (max(0, start - end))
//   - energy_consumed_kwh (computed in km / kWh)
//   - had_charging (was there a charging_process inside the window?)
//
// outside_temp_avg is intentionally NOT in the list — it triggers a
// per-row AVG over the positions table (millions of rows / car) which
// turns the list into an 11s query. The detail endpoint still returns the
// same outside_temp samples (and aggregates) for clients that need it.
//
// query params: startDate / endDate / minDuration (minutes) / page / show.
func TeslaMateAPICarsParkingsV2(c *gin.Context) {

	var ParkingsErr1 = "Unable to load parkings."
	var ParkingsErr2 = "Invalid date format."

	CarID := convertStringToInteger(c.Param("CarID"))
	ResultPage := convertStringToInteger(c.DefaultQuery("page", "1"))
	ResultShow := convertStringToInteger(c.DefaultQuery("show", "100"))

	parsedStartDate, err := parseDateParam(c.Query("startDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsParkingsV2", ParkingsErr2, err.Error())
		return
	}
	parsedEndDate, err := parseDateParam(c.Query("endDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsParkingsV2", ParkingsErr2, err.Error())
		return
	}
	minDurationParam := c.Query("minDuration")
	minDuration := 0
	if minDurationParam != "" {
		minDuration = convertStringToInteger(minDurationParam)
		if minDuration < 0 {
			minDuration = 0
		}
	}

	type Car struct {
		CarID   int        `json:"car_id"`
		CarName NullString `json:"car_name"`
	}
	type Parking struct {
		ParkingID            int         `json:"parking_id"`
		StartDate            string      `json:"start_date"`
		EndDate              NullString  `json:"end_date"`
		DurationMin          int         `json:"duration_min"`
		DurationStr          string      `json:"duration_str"`
		Address              NullString  `json:"address"`
		GeofenceID           NullInt64   `json:"geofence_id"`
		Latitude             NullFloat64 `json:"latitude"`
		Longitude            NullFloat64 `json:"longitude"`
		StartBatteryLevel    NullInt64   `json:"start_battery_level"`
		EndBatteryLevel      NullInt64   `json:"end_battery_level"`
		UsableBatteryDrop    int         `json:"usable_battery_drop"`
		EnergyConsumedKWh    NullFloat64 `json:"energy_consumed_kwh"`
		HadCharging          bool        `json:"had_charging"`
	}
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`
		UnitsTemperature string `json:"unit_of_temperature"`
	}
	type Data struct {
		Car            Car            `json:"car"`
		Parkings       []Parking      `json:"parkings"`
		TeslaMateUnits TeslaMateUnits `json:"units"`
	}
	type JSONData struct {
		Data Data `json:"data"`
	}

	var (
		CarName                       NullString
		ParkingsData                  []Parking
		UnitsLength, UnitsTemperature string
	)

	if ResultPage > 0 {
		ResultPage--
	} else {
		ResultPage = 0
	}
	ResultPage = (ResultPage * ResultShow)

	// Window functions on drives give us each drive's "next start" — that pair
	// (end_date, next_start) is the parking window. The end_position of the
	// preceding drive locates the vehicle. Energy uses the *next* drive's
	// start SOC (the drop during parking). When `next_start` is null
	// (the last drive), parking is in-progress: end_date / end_battery / etc.
	// stay null.
	query := `
		WITH drive_pairs AS (
			SELECT
				d.id AS drive_id,
				d.end_date AS park_start,
				LEAD(d.start_date) OVER w AS park_end,
				d.end_position_id,
				LEAD(d.start_position_id) OVER w AS next_start_position_id,
				d.end_address_id  AS park_address_id,
				d.end_geofence_id AS park_geofence_id,
				d.car_id
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			WINDOW w AS (PARTITION BY d.car_id ORDER BY d.start_date ASC)
		)
		SELECT
			dp.drive_id AS parking_id,
			dp.park_start,
			dp.park_end,
			COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60, EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60)::int AS duration_min,
			TO_CHAR((COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60, EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60)::int * INTERVAL '1 minute'), 'HH24:MI') as duration_str,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(addr.name, nullif(CONCAT_WS(' ', addr.road, addr.house_number), '')), addr.city)) AS address,
			dp.park_geofence_id AS geofence_id,
			start_pos.latitude,
			start_pos.longitude,
			COALESCE(start_pos.usable_battery_level, start_pos.battery_level) AS start_battery_level,
			COALESCE(end_pos.usable_battery_level, end_pos.battery_level) AS end_battery_level,
			GREATEST(
				COALESCE(start_pos.usable_battery_level, start_pos.battery_level)
				- COALESCE(end_pos.usable_battery_level, end_pos.battery_level), 0
			) AS usable_battery_drop,
			CASE
				WHEN start_pos.rated_battery_range_km IS NOT NULL AND end_pos.rated_battery_range_km IS NOT NULL
				THEN GREATEST(start_pos.rated_battery_range_km - end_pos.rated_battery_range_km, 0) * cars.efficiency
				ELSE NULL
			END AS energy_consumed_kwh,
			EXISTS(
				SELECT 1 FROM charging_processes cp
				WHERE cp.car_id = $1 AND cp.start_date >= dp.park_start
				AND (dp.park_end IS NULL OR cp.start_date < dp.park_end)
			) AS had_charging,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name
		FROM drive_pairs dp
		LEFT JOIN cars ON cars.id = dp.car_id
		LEFT JOIN positions start_pos ON start_pos.id = dp.end_position_id
		LEFT JOIN positions end_pos ON end_pos.id = dp.next_start_position_id
		LEFT JOIN addresses addr ON addr.id = dp.park_address_id
		LEFT JOIN geofences geofence ON geofence.id = dp.park_geofence_id
		WHERE 1=1`

	var queryParams []any
	queryParams = append(queryParams, CarID)
	paramIndex := 2

	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND dp.park_start >= $%d", paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND (dp.park_end IS NULL OR dp.park_end <= $%d)", paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}
	if minDuration > 0 {
		query += fmt.Sprintf(` AND COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60, EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60)::int >= $%d`, paramIndex)
		queryParams = append(queryParams, minDuration)
		paramIndex++
	}

	query += fmt.Sprintf(`
		ORDER BY dp.park_start DESC
		LIMIT $%d OFFSET $%d;`, paramIndex, paramIndex+1)
	queryParams = append(queryParams, ResultShow, ResultPage)

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsParkingsV2", ParkingsErr1, err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		park := Parking{}
		err = rows.Scan(
			&park.ParkingID,
			&park.StartDate,
			&park.EndDate,
			&park.DurationMin,
			&park.DurationStr,
			&park.Address,
			&park.GeofenceID,
			&park.Latitude,
			&park.Longitude,
			&park.StartBatteryLevel,
			&park.EndBatteryLevel,
			&park.UsableBatteryDrop,
			&park.EnergyConsumedKWh,
			&park.HadCharging,
			&UnitsLength,
			&UnitsTemperature,
			&CarName,
		)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsParkingsV2", ParkingsErr1, err.Error())
			return
		}

		park.StartDate = getTimeInTimeZone(park.StartDate)
		if len(park.EndDate) > 0 {
			park.EndDate = NullString(getTimeInTimeZone(string(park.EndDate)))
		}

		ParkingsData = append(ParkingsData, park)
	}

	if err = rows.Err(); err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsParkingsV2", ParkingsErr1, err.Error())
		return
	}

	jsonData := JSONData{
		Data{
			Car: Car{
				CarID:   CarID,
				CarName: CarName,
			},
			Parkings: ParkingsData,
			TeslaMateUnits: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsParkingsV2", jsonData)
}

// TeslaMateAPICarsParkingsDetailsV2 returns the parking session metadata
// plus an SOC + outside-temp time-series sampled from positions during the
// window. The `parking_id` is the drive_id that preceded the parking session.
func TeslaMateAPICarsParkingsDetailsV2(c *gin.Context) {

	var ParkingsErr1 = "Unable to load parking."
	var ParkingsErr2 = "Unable to load parking details."

	CarID := convertStringToInteger(c.Param("CarID"))
	ParkingID := convertStringToInteger(c.Param("ParkingID"))

	type Car struct {
		CarID   int        `json:"car_id"`
		CarName NullString `json:"car_name"`
	}
	type ParkingDetail struct {
		Date         string      `json:"date"`
		BatteryLevel NullInt64   `json:"battery_level"`
		UsableLevel  NullInt64   `json:"usable_battery_level"`
		OutsideTemp  NullFloat64 `json:"outside_temp"`
	}
	type Parking struct {
		ParkingID         int             `json:"parking_id"`
		StartDate         string          `json:"start_date"`
		EndDate           NullString      `json:"end_date"`
		DurationMin       int             `json:"duration_min"`
		DurationStr       string          `json:"duration_str"`
		Address           NullString      `json:"address"`
		GeofenceID        NullInt64       `json:"geofence_id"`
		Latitude          NullFloat64     `json:"latitude"`
		Longitude         NullFloat64     `json:"longitude"`
		StartBatteryLevel NullInt64       `json:"start_battery_level"`
		EndBatteryLevel   NullInt64       `json:"end_battery_level"`
		UsableBatteryDrop int             `json:"usable_battery_drop"`
		EnergyConsumedKWh NullFloat64     `json:"energy_consumed_kwh"`
		HadCharging       bool            `json:"had_charging"`
		OutsideTempAvg    NullFloat64     `json:"outside_temp_avg"`
		Details           []ParkingDetail `json:"parking_details"`
	}
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`
		UnitsTemperature string `json:"unit_of_temperature"`
	}
	type Data struct {
		Car            Car            `json:"car"`
		Parking        Parking        `json:"parking"`
		TeslaMateUnits TeslaMateUnits `json:"units"`
	}
	type JSONData struct {
		Data Data `json:"data"`
	}

	var (
		CarName                       NullString
		park                          Parking
		details                       []ParkingDetail
		UnitsLength, UnitsTemperature string
	)

	headQuery := `
		WITH drive_pairs AS (
			SELECT
				d.id AS drive_id,
				d.end_date AS park_start,
				LEAD(d.start_date) OVER w AS park_end,
				d.end_position_id,
				LEAD(d.start_position_id) OVER w AS next_start_position_id,
				d.end_address_id  AS park_address_id,
				d.end_geofence_id AS park_geofence_id,
				d.car_id
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			WINDOW w AS (PARTITION BY d.car_id ORDER BY d.start_date ASC)
		)
		SELECT
			dp.drive_id,
			dp.park_start,
			dp.park_end,
			COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60, EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60)::int AS duration_min,
			TO_CHAR((COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60, EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60)::int * INTERVAL '1 minute'), 'HH24:MI') as duration_str,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(addr.name, nullif(CONCAT_WS(' ', addr.road, addr.house_number), '')), addr.city)) AS address,
			dp.park_geofence_id,
			start_pos.latitude,
			start_pos.longitude,
			COALESCE(start_pos.usable_battery_level, start_pos.battery_level),
			COALESCE(end_pos.usable_battery_level, end_pos.battery_level),
			GREATEST(COALESCE(start_pos.usable_battery_level, start_pos.battery_level) - COALESCE(end_pos.usable_battery_level, end_pos.battery_level), 0) AS usable_battery_drop,
			CASE
				WHEN start_pos.rated_battery_range_km IS NOT NULL AND end_pos.rated_battery_range_km IS NOT NULL
				THEN GREATEST(start_pos.rated_battery_range_km - end_pos.rated_battery_range_km, 0) * cars.efficiency
				ELSE NULL
			END,
			EXISTS(
				SELECT 1 FROM charging_processes cp
				WHERE cp.car_id = $1 AND cp.start_date >= dp.park_start
				AND (dp.park_end IS NULL OR cp.start_date < dp.park_end)
			),
			(
				SELECT AVG(p.outside_temp)
				FROM positions p
				WHERE p.car_id = $1 AND p.date >= dp.park_start
				AND (dp.park_end IS NULL OR p.date < dp.park_end)
			),
			(SELECT unit_of_length FROM settings LIMIT 1),
			(SELECT unit_of_temperature FROM settings LIMIT 1),
			cars.name
		FROM drive_pairs dp
		LEFT JOIN cars ON cars.id = dp.car_id
		LEFT JOIN positions start_pos ON start_pos.id = dp.end_position_id
		LEFT JOIN positions end_pos ON end_pos.id = dp.next_start_position_id
		LEFT JOIN addresses addr ON addr.id = dp.park_address_id
		LEFT JOIN geofences geofence ON geofence.id = dp.park_geofence_id
		WHERE dp.drive_id = $2`

	row := db.QueryRow(headQuery, CarID, ParkingID)
	err := row.Scan(
		&park.ParkingID,
		&park.StartDate,
		&park.EndDate,
		&park.DurationMin,
		&park.DurationStr,
		&park.Address,
		&park.GeofenceID,
		&park.Latitude,
		&park.Longitude,
		&park.StartBatteryLevel,
		&park.EndBatteryLevel,
		&park.UsableBatteryDrop,
		&park.EnergyConsumedKWh,
		&park.HadCharging,
		&park.OutsideTempAvg,
		&UnitsLength,
		&UnitsTemperature,
		&CarName,
	)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsParkingsDetailsV2", ParkingsErr1, err.Error())
		return
	}

	if UnitsTemperature == "F" {
		park.OutsideTempAvg = celsiusToFahrenheitNilSupport(park.OutsideTempAvg)
	}
	park.StartDate = getTimeInTimeZone(park.StartDate)
	if len(park.EndDate) > 0 {
		park.EndDate = NullString(getTimeInTimeZone(string(park.EndDate)))
	}

	// Time-series during the parking window. Cap at 500 rows: long parkings
	// can have thousands of position samples; a sparse sample is enough for
	// SOC drift / temperature trend.
	detailQuery := `
		WITH drive_pairs AS (
			SELECT
				d.id AS drive_id,
				d.end_date AS park_start,
				LEAD(d.start_date) OVER w AS park_end,
				d.car_id
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			WINDOW w AS (PARTITION BY d.car_id ORDER BY d.start_date ASC)
		)
		SELECT
			p.date,
			p.battery_level,
			p.usable_battery_level,
			p.outside_temp
		FROM drive_pairs dp
		JOIN positions p ON p.car_id = dp.car_id
			AND p.date >= dp.park_start
			AND (dp.park_end IS NULL OR p.date < dp.park_end)
		WHERE dp.drive_id = $2
		ORDER BY p.date ASC
		LIMIT 500;`

	rows, err := db.Query(detailQuery, CarID, ParkingID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsParkingsDetailsV2", ParkingsErr2, err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		d := ParkingDetail{}
		if err = rows.Scan(&d.Date, &d.BatteryLevel, &d.UsableLevel, &d.OutsideTemp); err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsParkingsDetailsV2", ParkingsErr2, err.Error())
			return
		}
		if UnitsTemperature == "F" {
			d.OutsideTemp = celsiusToFahrenheitNilSupport(d.OutsideTemp)
		}
		d.Date = getTimeInTimeZone(d.Date)
		details = append(details, d)
	}
	if err = rows.Err(); err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsParkingsDetailsV2", ParkingsErr2, err.Error())
		return
	}
	park.Details = details

	jsonData := JSONData{
		Data{
			Car:     Car{CarID: CarID, CarName: CarName},
			Parking: park,
			TeslaMateUnits: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsParkingsDetailsV2", jsonData)
}
