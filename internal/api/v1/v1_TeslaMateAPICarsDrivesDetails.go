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

// TeslaMateAPICarsDrivesDetailsV1 godoc
//
// @Summary 单条行程详情
// @Tags v1
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param DriveID path int true "行程 ID"
// @Param sample query string false "行程明细下采样：full、every_5s（默认）、every_30s" Enums(full, every_5s, every_30s)
// @Param include_route query bool false "设为 false 可在响应中省略 drive_details（行程轨迹）"
// @Success 200 {object} V1JSONEnvelope
// @Failure 200 {object} V1ErrorEnvelope
// @Router /v1/cars/{CarID}/drives/{DriveID} [get]
func TeslaMateAPICarsDrivesDetailsV1(c *gin.Context) {

	// define error messages
	var (
		CarsDrivesDetailsError1 = "Unable to load drive."
		CarsDrivesDetailsError2 = "Unable to load drive details."
	)

	// getting CarID and DriveID param from URL
	CarID := apicommon.ConvertStringToInteger(c.Param("CarID"))
	DriveID := apicommon.ConvertStringToInteger(c.Param("DriveID"))

	// creating structs for /cars/<CarID>/drives/<DriveID>
	// Car struct - child of Data
	type Car struct {
		CarID   int          `json:"car_id"`   // smallint
		CarName nullx.String `json:"car_name"` // text (nullable)
	}
	// OdometerDetails struct - child of Drives
	type OdometerDetails struct {
		OdometerStart    float64 `json:"odometer_start"`    // float64
		OdometerEnd      float64 `json:"odometer_end"`      // float64
		OdometerDistance float64 `json:"odometer_distance"` // float64
	}
	// BatteryDetails struct - child of Drives
	type BatteryDetails struct {
		StartUsableBatteryLevel int  `json:"start_usable_battery_level"` // int
		StartBatteryLevel       int  `json:"start_battery_level"`        // int
		EndUsableBatteryLevel   int  `json:"end_usable_battery_level"`   // int
		EndBatteryLevel         int  `json:"end_battery_level"`          // int
		ReducedRange            bool `json:"reduced_range"`              // bool
		IsSufficientlyPrecise   bool `json:"is_sufficiently_precise"`    // bool
	}
	// PreferredRange struct - child of Drives
	type PreferredRange struct {
		StartRange float64 `json:"start_range"` // float64
		EndRange   float64 `json:"end_range"`   // float64
		RangeDiff  float64 `json:"range_diff"`  // float64
	}
	// ClimateInfo struct - child of DriveDetails
	type ClimateInfo struct {
		InsideTemp           nullx.Float64 `json:"inside_temp"`            // numeric(4,1)
		OutsideTemp          nullx.Float64 `json:"outside_temp"`           // numeric(4,1)
		IsClimateOn          nullx.Bool    `json:"is_climate_on"`          // boolean
		FanStatus            nullx.Int64   `json:"fan_status"`             // integer
		DriverTempSetting    nullx.Float64 `json:"driver_temp_setting"`    // numeric(4,1)
		PassengerTempSetting nullx.Float64 `json:"passenger_temp_setting"` // numeric(4,1)
		IsRearDefrosterOn    nullx.Bool    `json:"is_rear_defroster_on"`   // boolean
		IsFrontDefrosterOn   nullx.Bool    `json:"is_front_defroster_on"`  // boolean
	}
	// BatteryInfo struct - child of DriveDetails
	type BatteryInfo struct {
		EstBatteryRange      nullx.Float64 `json:"est_battery_range"`       // numeric(6,2)
		IdealBatteryRange    nullx.Float64 `json:"ideal_battery_range"`     // numeric(6,2)
		RatedBatteryRange    nullx.Float64 `json:"rated_battery_range"`     // numeric(6,2)
		BatteryHeater        nullx.Bool    `json:"battery_heater"`          // boolean
		BatteryHeaterOn      nullx.Bool    `json:"battery_heater_on"`       // boolean
		BatteryHeaterNoPower nullx.Bool    `json:"battery_heater_no_power"` // boolean
	}
	// DriveDetails struct - child of Drive
	type DriveDetails struct {
		DetailID           int         `json:"detail_id"`                  // integer
		Date               string      `json:"date"`                       // timestamp without time zone
		Latitude           float64     `json:"latitude"`                   // numeric(8,6)
		Longitude          float64     `json:"longitude"`                  // numeric(9,6)
		Speed              int         `json:"speed"`                      // smallint
		Power              int         `json:"power"`                      // smallint
		Odometer           float64     `json:"odometer"`                   // double precision
		BatteryLevel       int         `json:"battery_level"`              // smallint
		UsableBatteryLevel nullx.Int64 `json:"usable_battery_level"`       // smallint
		Elevation          nullx.Int64 `json:"elevation"`                  // smallint
		ClimateInfo        ClimateInfo `json:"climate_info"`               // struct
		BatteryInfo        BatteryInfo `json:"battery_info"`               // struct
		TpmsPressureFL     *float64    `json:"tpms_pressure_fl,omitempty"` // (added)
		TpmsPressureFR     *float64    `json:"tpms_pressure_fr,omitempty"` // (added)
		TpmsPressureRL     *float64    `json:"tpms_pressure_rl,omitempty"` // (added)
		TpmsPressureRR     *float64    `json:"tpms_pressure_rr,omitempty"` // (added)
	}
	// Geofence struct - child of Drive (added)
	type Geofence struct {
		ID   int    `json:"id"`   // ID
		Name string `json:"name"` // 名称
	}
	// Position struct - child of Drive (added)
	type Position struct {
		Latitude  float64 `json:"latitude"`  // 纬度
		Longitude float64 `json:"longitude"` // 经度
	}
	// Drive struct - child of Data
	type Drive struct {
		DriveID                  int             `json:"drive_id"`                             // int
		StartDate                string          `json:"start_date"`                           // string
		EndDate                  string          `json:"end_date"`                             // string
		StartAddress             string          `json:"start_address"`                        // string
		EndAddress               string          `json:"end_address"`                          // string
		OdometerDetails          OdometerDetails `json:"odometer_details"`                     // OdometerDetails
		DurationMin              int             `json:"duration_min"`                         // int
		DurationStr              string          `json:"duration_str"`                         // string
		SpeedMax                 int             `json:"speed_max"`                            // int
		SpeedAvg                 float64         `json:"speed_avg"`                            // float64
		PowerMax                 int             `json:"power_max"`                            // int
		PowerMin                 int             `json:"power_min"`                            // int
		BatteryDetails           BatteryDetails  `json:"battery_details"`                      // BatteryDetails
		RangeIdeal               PreferredRange  `json:"range_ideal"`                          // PreferredRange
		RangeRated               PreferredRange  `json:"range_rated"`                          // PreferredRange
		OutsideTempAvg           float64         `json:"outside_temp_avg"`                     // float64
		InsideTempAvg            float64         `json:"inside_temp_avg"`                      // float64
		EnergyConsumedNet        *float64        `json:"energy_consumed_net"`                  // Energy consumed (net) in kWh
		ConsumptionNet           *float64        `json:"consumption_net"`                      // Ø Consumption (net) per distance unit
		Ascent                   *float64        `json:"ascent,omitempty"`                     // (added)
		Descent                  *float64        `json:"descent,omitempty"`                    // (added)
		ConsumptionSlopeAdjusted *float64        `json:"consumption_slope_adjusted,omitempty"` // (added)
		StartGeofence            *Geofence       `json:"start_geofence,omitempty"`             // (added)
		EndGeofence              *Geofence       `json:"end_geofence,omitempty"`               // (added)
		StartPosition            *Position       `json:"start_position,omitempty"`             // (added)
		EndPosition              *Position       `json:"end_position,omitempty"`               // (added)
		IsComplete               bool            `json:"is_complete"`                          // (added)
		DriveDetails             []DriveDetails  `json:"drive_details"`                        // struct
	}
	// TeslaMateUnits struct - child of Data
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`      // string
		UnitsTemperature string `json:"unit_of_temperature"` // string
	}
	// Data struct - child of JSONData
	type Data struct {
		Car            Car            `json:"car"` // 车辆
		Drive          Drive          `json:"drive"`
		TeslaMateUnits TeslaMateUnits `json:"units"` // 单位
	}
	// JSONData struct - main
	type JSONData struct {
		Data Data `json:"data"` // 响应数据
	}

	// creating required vars
	var (
		CarName                                      nullx.String
		drive                                        Drive
		DriveDetailsData                             []DriveDetails
		UnitsLength, UnitsTemperature, UnitsPressure string
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
			drives.ascent,
			drives.descent,
			start_geofence.id AS start_geofence_id,
			start_geofence.name AS start_geofence_name,
			end_geofence.id AS end_geofence_id,
			end_geofence.name AS end_geofence_name,
			start_position.latitude AS start_lat,
			start_position.longitude AS start_lng,
			end_position.latitude AS end_lat,
			end_position.longitude AS end_lng,
			(end_date IS NOT NULL) AS is_complete,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			(SELECT COALESCE(unit_of_pressure, 'bar') FROM settings LIMIT 1) as unit_of_pressure,
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
	row := apicommon.DB.QueryRow(query, CarID, DriveID)

	var (
		ascent, descent                    sql.NullFloat64
		startGeofenceID, endGeofenceID     sql.NullInt64
		startGeofenceName, endGeofenceName sql.NullString
		startLat, startLng, endLat, endLng sql.NullFloat64
		isComplete                         sql.NullBool
	)

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
		&ascent,
		&descent,
		&startGeofenceID,
		&startGeofenceName,
		&endGeofenceID,
		&endGeofenceName,
		&startLat,
		&startLng,
		&endLat,
		&endLng,
		&isComplete,
		&UnitsLength,
		&UnitsTemperature,
		&UnitsPressure,
		&CarName,
	)
	if ascent.Valid {
		v := ascent.Float64
		drive.Ascent = &v
	}
	if descent.Valid {
		v := descent.Float64
		drive.Descent = &v
	}
	if startGeofenceID.Valid && startGeofenceName.Valid {
		drive.StartGeofence = &Geofence{ID: int(startGeofenceID.Int64), Name: startGeofenceName.String}
	}
	if endGeofenceID.Valid && endGeofenceName.Valid {
		drive.EndGeofence = &Geofence{ID: int(endGeofenceID.Int64), Name: endGeofenceName.String}
	}
	if startLat.Valid && startLng.Valid {
		drive.StartPosition = &Position{Latitude: startLat.Float64, Longitude: startLng.Float64}
	}
	if endLat.Valid && endLng.Valid {
		drive.EndPosition = &Position{Latitude: endLat.Float64, Longitude: endLng.Float64}
	}
	if isComplete.Valid {
		drive.IsComplete = isComplete.Bool
	}
	if drive.ConsumptionNet != nil && drive.OdometerDetails.OdometerDistance > 0 && drive.Ascent != nil && drive.Descent != nil {
		adj := conv.SlopeAdjustedConsumption(drive.OdometerDetails.OdometerDistance, *drive.Ascent, *drive.Descent, *drive.ConsumptionNet)
		drive.ConsumptionSlopeAdjusted = &adj
	}

	switch err {
	case sql.ErrNoRows:
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", "No rows were returned!", err.Error())
		return
	case nil:
		// nothing wrong.. continuing
		break
	default:
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError1, err.Error())
		return
	}

	// converting values based of settings UnitsLength
	if UnitsLength == "mi" {
		drive.OdometerDetails.OdometerStart = conv.KmToMi(drive.OdometerDetails.OdometerStart)
		drive.OdometerDetails.OdometerEnd = conv.KmToMi(drive.OdometerDetails.OdometerEnd)
		drive.OdometerDetails.OdometerDistance = conv.KmToMi(drive.OdometerDetails.OdometerDistance)
		drive.SpeedMax = int(conv.KmToMi(float64(drive.SpeedMax)))
		drive.SpeedAvg = conv.KmToMi(drive.SpeedAvg)
		drive.RangeIdeal.StartRange = conv.KmToMi(drive.RangeIdeal.StartRange)
		drive.RangeIdeal.EndRange = conv.KmToMi(drive.RangeIdeal.EndRange)
		drive.RangeIdeal.RangeDiff = conv.KmToMi(drive.RangeIdeal.RangeDiff)
		drive.RangeRated.StartRange = conv.KmToMi(drive.RangeRated.StartRange)
		drive.RangeRated.EndRange = conv.KmToMi(drive.RangeRated.EndRange)
		drive.RangeRated.RangeDiff = conv.KmToMi(drive.RangeRated.RangeDiff)
		if drive.ConsumptionNet != nil {
			*drive.ConsumptionNet = conv.KmToMi(*drive.ConsumptionNet)
		}
		if drive.ConsumptionSlopeAdjusted != nil {
			*drive.ConsumptionSlopeAdjusted = conv.KmToMi(*drive.ConsumptionSlopeAdjusted)
		}
	}
	// converting values based of settings UnitsTemperature
	if UnitsTemperature == "F" {
		drive.OutsideTempAvg = conv.CelsiusToFahrenheit(drive.OutsideTempAvg)
		drive.InsideTempAvg = conv.CelsiusToFahrenheit(drive.InsideTempAvg)
	}
	// adjusting to timezone differences from UTC to be userspecific
	drive.StartDate = apicommon.GetTimeInTimeZone(drive.StartDate)
	drive.EndDate = apicommon.GetTimeInTimeZone(drive.EndDate)

	// optional ?include_route=false skips the heavy positions query.
	includeRoute := true
	if v := c.Query("include_route"); v == "false" || v == "0" {
		includeRoute = false
	}
	if !includeRoute {
		// build response with empty drive_details and return early
		jsonData := JSONData{
			Data{
				Car: Car{
					CarID:   CarID,
					CarName: CarName,
				},
				Drive: drive,
				TeslaMateUnits: TeslaMateUnits{
					UnitsLength:      UnitsLength,
					UnitsTemperature: UnitsTemperature,
				},
			},
		}
		apicommon.HandleSuccessResponse(c, "TeslaMateAPICarsDrivesDetailsV1", jsonData)
		return
	}

	// optional ?sample=full|every_5s|every_30s — default every_5s downsamples to one row per 5s bucket.
	sampleMode := c.DefaultQuery("sample", "every_5s")
	bucketSeconds := 5
	switch sampleMode {
	case "full":
		bucketSeconds = 0
	case "every_30s":
		bucketSeconds = 30
	case "every_5s", "":
		bucketSeconds = 5
	default:
		bucketSeconds = 5
	}

	// getting detailed drive data from database
	var detailQuery string
	if bucketSeconds > 0 {
		// Use DISTINCT ON to keep the earliest position per N-second bucket.
		detailQuery = fmt.Sprintf(`
		 			SELECT * FROM (
		 				SELECT DISTINCT ON (date_trunc('minute', date) + (FLOOR(EXTRACT(SECOND FROM date) / %d) * %d) * INTERVAL '1 second')
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
		 					battery_heater_no_power,
		 					tpms_pressure_fl,
		 					tpms_pressure_fr,
		 					tpms_pressure_rl,
		 					tpms_pressure_rr
		 				FROM positions
		 				WHERE drive_id = $1
		 				ORDER BY date_trunc('minute', date) + (FLOOR(EXTRACT(SECOND FROM date) / %d) * %d) * INTERVAL '1 second', id ASC
		 			) bucketed
		 			ORDER BY detail_id ASC;`, bucketSeconds, bucketSeconds, bucketSeconds, bucketSeconds)
	} else {
		detailQuery = `
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
		 				battery_heater_no_power,
		 				tpms_pressure_fl,
		 				tpms_pressure_fr,
		 				tpms_pressure_rl,
		 				tpms_pressure_rr
		 			FROM positions
		 			WHERE drive_id = $1
		 			ORDER BY id ASC;`
	}
	rows, err := apicommon.DB.Query(detailQuery, DriveID)

	// checking for errors in query
	if err != nil {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
		return
	}

	// defer closing rows
	defer rows.Close()

	// looping through all results
	for rows.Next() {

		// creating drivedetails object based on struct
		drivedetails := DriveDetails{}

		var tpmsFL, tpmsFR, tpmsRL, tpmsRR sql.NullFloat64

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
			&tpmsFL,
			&tpmsFR,
			&tpmsRL,
			&tpmsRR,
		)
		// TPMS values stored in bar; convert to psi if settings.unit_of_pressure = 'psi'.
		convertPressure := func(v sql.NullFloat64) *float64 {
			if !v.Valid {
				return nil
			}
			val := v.Float64
			if UnitsPressure == "psi" {
				val = conv.BarToPsi(val)
			}
			return &val
		}
		drivedetails.TpmsPressureFL = convertPressure(tpmsFL)
		drivedetails.TpmsPressureFR = convertPressure(tpmsFR)
		drivedetails.TpmsPressureRL = convertPressure(tpmsRL)
		drivedetails.TpmsPressureRR = convertPressure(tpmsRR)

		// converting values based of settings UnitsLength
		if UnitsLength == "mi" {
			drivedetails.Odometer = conv.KmToMi(drivedetails.Odometer)
			drivedetails.Speed = int(conv.KmToMi(float64(drivedetails.Speed)))
			drivedetails.BatteryInfo.EstBatteryRange = conv.KmToMiNullable(drivedetails.BatteryInfo.EstBatteryRange)
			drivedetails.BatteryInfo.IdealBatteryRange = conv.KmToMiNullable(drivedetails.BatteryInfo.IdealBatteryRange)
			drivedetails.BatteryInfo.RatedBatteryRange = conv.KmToMiNullable(drivedetails.BatteryInfo.RatedBatteryRange)
		}
		// converting values based of settings UnitsTemperature
		if UnitsTemperature == "F" {
			drivedetails.ClimateInfo.InsideTemp = conv.CelsiusToFahrenheitNullable(drivedetails.ClimateInfo.InsideTemp)
			drivedetails.ClimateInfo.OutsideTemp = conv.CelsiusToFahrenheitNullable(drivedetails.ClimateInfo.OutsideTemp)
			drivedetails.ClimateInfo.DriverTempSetting = conv.CelsiusToFahrenheitNullable(drivedetails.ClimateInfo.DriverTempSetting)
			drivedetails.ClimateInfo.PassengerTempSetting = conv.CelsiusToFahrenheitNullable(drivedetails.ClimateInfo.PassengerTempSetting)
		}
		// adjusting to timezone differences from UTC to be userspecific
		drivedetails.Date = apicommon.GetTimeInTimeZone(drivedetails.Date)

		// checking for errors after scanning
		if err != nil {
			apicommon.HandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
			return
		}

		// appending drive to drive
		DriveDetailsData = append(DriveDetailsData, drivedetails)
		drive.DriveDetails = DriveDetailsData
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsDrivesDetailsV1", CarsDrivesDetailsError2, err.Error())
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
			Drive: drive,
			TeslaMateUnits: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// return jsonData
	apicommon.HandleSuccessResponse(c, "TeslaMateAPICarsDrivesDetailsV1", jsonData)
}
