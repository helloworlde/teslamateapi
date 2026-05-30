package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsParkingsV2 derives parking sessions from gaps between
// adjacent drives. The N-th parking covers the window
// (drives[N].end_date, drives[N+1].start_date) on the same car, where drives
// are ordered by start_date ASC.
//
// preceding_drive_id = drives[N].id (the drive that *preceded* the parking
// session). It's stable, monotonic, and uniquely identifies the session even
// after new drives are appended — clients use this id as the path segment in
// /parkings/{preceding_drive_id}.
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
// query params: start_date / end_date / min_duration (minutes) / page / show.
func TeslaMateAPICarsParkingsV2(c *gin.Context) {

	const handler = "TeslaMateAPICarsParkingsV2"
	var ParkingsErr1 = "Unable to load parkings."
	var ParkingsErr2 = "Invalid date format."

	CarID, ok := v2RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}
	ResultPage, ok := v2OptionalIntInRange(c, handler, "page", c.Query("page"), 1, 1, 1<<31-1)
	if !ok {
		return
	}
	ResultShow, ok := v2OptionalIntInRange(c, handler, "show", c.Query("show"), 100, 1, 10000)
	if !ok {
		return
	}
	minDuration, ok := v2OptionalIntInRange(c, handler, "min_duration", c.Query("min_duration"), 0, 0, 1<<31-1)
	if !ok {
		return
	}

	parsedStartDate, err := parseDateParam(c.Query("start_date"))
	if err != nil {
		v2HandleErrorResponse(c, handler, http.StatusBadRequest, ParkingsErr2, err.Error())
		return
	}
	parsedEndDate, err := parseDateParam(c.Query("end_date"))
	if err != nil {
		v2HandleErrorResponse(c, handler, http.StatusBadRequest, ParkingsErr2, err.Error())
		return
	}

	type Car struct {
		CarID   int        `json:"car_id"`
		CarName NullString `json:"car_name"`
	}
	type Parking struct {
		PrecedingDriveID  int         `json:"preceding_drive_id"`
		StartDate         string      `json:"start_date"`
		EndDate           NullString  `json:"end_date"`
		DurationMin       int         `json:"duration_min"`
		DurationStr       string      `json:"duration_str"`
		Address           NullString  `json:"address"`
		GeofenceID        NullInt64   `json:"geofence_id"`
		Latitude          NullFloat64 `json:"latitude"`
		Longitude         NullFloat64 `json:"longitude"`
		StartBatteryLevel NullInt64   `json:"start_battery_level"`
		EndBatteryLevel   NullInt64   `json:"end_battery_level"`
		UsableBatteryDrop int         `json:"usable_battery_drop"`
		EnergyConsumedKWh NullFloat64 `json:"energy_consumed_kwh"`
		HadCharging       bool        `json:"had_charging"`
	}
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`
		UnitsTemperature string `json:"unit_of_temperature"`
	}
	type Pagination struct {
		Page       int  `json:"page"`
		Show       int  `json:"show"`
		TotalCount int  `json:"total_count"`
		HasNext    bool `json:"has_next"`
	}
	type Data struct {
		Car            Car            `json:"car"`
		Parkings       []Parking      `json:"parkings"`
		Pagination     Pagination     `json:"pagination"`
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

	// 1-indexed page → 0-indexed offset. Validation guarantees ResultPage >= 1.
	offset := (ResultPage - 1) * ResultShow

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

	// Build a single filter clause shared by both the page query and the
	// total_count query so the row count exactly matches what would appear if
	// the client paged through every page.
	var filterClauses string
	filterParams := []any{CarID}
	paramIndex := 2

	if parsedStartDate != "" {
		filterClauses += fmt.Sprintf(" AND dp.park_start >= $%d", paramIndex)
		filterParams = append(filterParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		filterClauses += fmt.Sprintf(" AND (dp.park_end IS NULL OR dp.park_end <= $%d)", paramIndex)
		filterParams = append(filterParams, parsedEndDate)
		paramIndex++
	}
	if minDuration > 0 {
		filterClauses += fmt.Sprintf(` AND COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60, EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60)::int >= $%d`, paramIndex)
		filterParams = append(filterParams, minDuration)
		paramIndex++
	}

	query += filterClauses + fmt.Sprintf(`
		ORDER BY dp.park_start DESC
		LIMIT $%d OFFSET $%d;`, paramIndex, paramIndex+1)
	queryParams := append(append([]any{}, filterParams...), ResultShow, offset)

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ParkingsErr1, err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		park := Parking{}
		err = rows.Scan(
			&park.PrecedingDriveID,
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
			v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ParkingsErr1, err.Error())
			return
		}

		park.StartDate = getTimeInTimeZone(park.StartDate)
		if len(park.EndDate) > 0 {
			park.EndDate = NullString(getTimeInTimeZone(string(park.EndDate)))
		}

		ParkingsData = append(ParkingsData, park)
	}

	if err = rows.Err(); err != nil {
		v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ParkingsErr1, err.Error())
		return
	}

	// Total-count query uses the identical filter clauses (without LIMIT/OFFSET)
	// so has_next is exact even when the page is partially full or empty.
	countQuery := `
		WITH drive_pairs AS (
			SELECT
				d.id AS drive_id,
				d.end_date AS park_start,
				LEAD(d.start_date) OVER w AS park_end
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			WINDOW w AS (PARTITION BY d.car_id ORDER BY d.start_date ASC)
		)
		SELECT COUNT(*) FROM drive_pairs dp WHERE 1=1` + filterClauses
	var totalCount int
	if err := db.QueryRow(countQuery, filterParams...).Scan(&totalCount); err != nil {
		v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ParkingsErr1, err.Error())
		return
	}

	jsonData := JSONData{
		Data{
			Car: Car{
				CarID:   CarID,
				CarName: CarName,
			},
			Parkings: ParkingsData,
			Pagination: Pagination{
				Page:       ResultPage,
				Show:       ResultShow,
				TotalCount: totalCount,
				HasNext:    offset+len(ParkingsData) < totalCount,
			},
			TeslaMateUnits: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	TeslaMateAPIHandleSuccessResponse(c, handler, jsonData)
}

// TeslaMateAPICarsParkingsDetailsV2 returns the parking session metadata
// plus an SOC + outside-temp time-series sampled from positions during the
// window. The path segment is `preceding_drive_id` — the drive_id that
// preceded (and ended) the parking session.
func TeslaMateAPICarsParkingsDetailsV2(c *gin.Context) {

	const handler = "TeslaMateAPICarsParkingsDetailsV2"
	var ParkingsErr1 = "Unable to load parking."
	var ParkingsErr2 = "Unable to load parking details."

	CarID, ok := v2RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}
	PrecedingDriveID, ok := v2RequirePositiveIntParam(c, handler, "preceding_drive_id", c.Param("PrecedingDriveID"))
	if !ok {
		return
	}

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
		PrecedingDriveID  int             `json:"preceding_drive_id"`
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

	row := db.QueryRow(headQuery, CarID, PrecedingDriveID)
	err := row.Scan(
		&park.PrecedingDriveID,
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
	if errors.Is(err, sql.ErrNoRows) {
		v2HandleErrorResponse(c, handler, http.StatusNotFound, "Parking not found.", fmt.Sprintf("car_id=%d preceding_drive_id=%d", CarID, PrecedingDriveID))
		return
	}
	if err != nil {
		v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ParkingsErr1, err.Error())
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

	rows, err := db.Query(detailQuery, CarID, PrecedingDriveID)
	if err != nil {
		v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ParkingsErr2, err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		d := ParkingDetail{}
		if err = rows.Scan(&d.Date, &d.BatteryLevel, &d.UsableLevel, &d.OutsideTemp); err != nil {
			v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ParkingsErr2, err.Error())
			return
		}
		if UnitsTemperature == "F" {
			d.OutsideTemp = celsiusToFahrenheitNilSupport(d.OutsideTemp)
		}
		d.Date = getTimeInTimeZone(d.Date)
		details = append(details, d)
	}
	if err = rows.Err(); err != nil {
		v2HandleErrorResponse(c, handler, http.StatusInternalServerError, ParkingsErr2, err.Error())
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
