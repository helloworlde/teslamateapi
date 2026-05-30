package main

import (
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/tobiasehlert/teslamateapi/src/dto"
)

// TeslaMateAPICarsUpdatesV1 returns paginated software-update history.
//
// @Summary      List firmware updates
// @Description  Returns software-update history for the car.
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path   int  true   "TeslaMate cars.id"
// @Param        page   query  int  false  "1-indexed page"  default(1)
// @Param        show   query  int  false  "page size"       default(100)
// @Success      200    {object}  dto.V1UpdatesResponse
// @Failure      401    {object}  dto.ErrorEnvelope
// @Router       /api/v1/cars/{CarID}/updates [get]
func TeslaMateAPICarsUpdatesV1(c *gin.Context) {

	// define error messages
	var CarsUpdatesError1 = "Unable to load updates."

	// getting CarID param from URL
	CarID := convertStringToInteger(c.Param("CarID"))
	// query options to modify query when collecting data
	ResultPage := convertStringToInteger(c.DefaultQuery("page", "1"))
	ResultShow := convertStringToInteger(c.DefaultQuery("show", "100"))

	// creating required vars
	var (
		UpdatesData []dto.V1Update
		CarData     dto.Car
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
			updates.id,
			cars.name,
			start_date,
			end_date,
			version
		FROM updates
		LEFT JOIN cars ON car_id = cars.id
		WHERE car_id = $1 AND end_date IS NOT NULL AND version IS NOT NULL
		ORDER BY start_date DESC
		LIMIT $2 OFFSET $3;`
	rows, err := db.QueryContext(c.Request.Context(), query, CarID, ResultShow, ResultPage)

	// checking for errors in query
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
		return

	}

	// defer closing rows
	defer rows.Close()

	// looping through all results
	for rows.Next() {

		// creating update object based on struct
		update := dto.V1Update{}

		// scanning row and putting values into the update
		err = rows.Scan(
			&update.UpdateID,
			&CarData.CarName,
			&update.StartDate,
			&update.EndDate,
			&update.Version,
		)

		// checking for errors after scanning
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
			return
		}

		// adjusting to timezone differences from UTC to be userspecific
		update.StartDate = getTimeInTimeZone(update.StartDate)
		update.EndDate = getTimeInTimeZone(update.EndDate)

		// appending update to UpdatesData
		UpdatesData = append(UpdatesData, update)
		CarData.CarID = CarID
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
		return
	}

	//
	// build the data-blob
	jsonData := dto.V1UpdatesResponse{
		Data: dto.V1UpdatesData{
			Car:     CarData,
			Updates: UpdatesData,
		},
	}

	// return jsonData
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsUpdatesV1", jsonData)
}
