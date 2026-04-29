package main

import (
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsV1 func
func TeslaMateAPICarsV1(c *gin.Context) {

	// define error messages
	var CarsError1 = "Unable to load cars."

	// getting CarID param from URL
	ParamCarID := c.Param("CarID")
	var CarID int
	if ParamCarID != "" {
		CarID = convertStringToInteger(ParamCarID)
	}

	// creating required vars
	var CarsData []CarsV1Car

	// getting data from database
	query := `
		SELECT
			cars.id,
			eid,
			vid,
			model,
			efficiency,
			inserted_at,
			updated_at,
			vin,
			name,
			trim_badging,
			exterior_color,
			spoiler_type,
			wheel_type,
			suspend_min,
			suspend_after_idle_min,
			req_not_unlocked,
			free_supercharging,
			use_streaming_api,
			(SELECT COUNT(*) FROM charging_processes WHERE car_id=cars.id) as total_charges,
			(SELECT COUNT(*) FROM drives WHERE car_id=cars.id) as total_drives,
			(SELECT COUNT(*) FROM updates WHERE car_id=cars.id) as total_charges
		FROM cars
		LEFT JOIN car_settings ON cars.id = car_settings.id
		ORDER BY id;`
	rows, err := db.Query(query)

	// checking for errors in query
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsV1", CarsError1, err.Error())
		return
	}

	// defer closing rows
	defer rows.Close()

	// looping through all results
	for rows.Next() {

		car := CarsV1Car{}

		// scanning row and putting values into the car
		err = rows.Scan(
			&car.CarID,
			&car.CarDetails.EID,
			&car.CarDetails.VID,
			&car.CarDetails.Model,
			&car.CarDetails.Efficiency,
			&car.TeslaMateDetails.InsertedAt,
			&car.TeslaMateDetails.UpdatedAt,
			&car.CarDetails.Vin,
			&car.Name,
			&car.CarDetails.TrimBadging,
			&car.CarExterior.ExteriorColor,
			&car.CarExterior.SpoilerType,
			&car.CarExterior.WheelType,
			&car.CarSettings.SuspendMin,
			&car.CarSettings.SuspendAfterIdleMin,
			&car.CarSettings.ReqNotUnlocked,
			&car.CarSettings.FreeSupercharging,
			&car.CarSettings.UseStreamingAPI,
			&car.TeslaMateStats.TotalCharges,
			&car.TeslaMateStats.TotalDrives,
			&car.TeslaMateStats.TotalUpdates,
		)

		// checking for errors after scanning
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsV1", CarsError1, err.Error())
			return
		}

		// appending car to CarsData if CarID is 0 or is CarID matches car.CarID
		if CarID == 0 && len(ParamCarID) == 0 || CarID != 0 && CarID == car.CarID {

			// adjusting to timezone differences from UTC to be userspecific
			car.TeslaMateDetails.InsertedAt = getTimeInTimeZone(car.TeslaMateDetails.InsertedAt)
			car.TeslaMateDetails.UpdatedAt = getTimeInTimeZone(car.TeslaMateDetails.UpdatedAt)

			CarsData = append(CarsData, car)
		}
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsV1", CarsError1, err.Error())
		return
	}

	jsonData := CarsV1Envelope{
		Data: CarsV1Data{Cars: CarsData},
	}

	// return jsonData
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsV1", jsonData)

}
