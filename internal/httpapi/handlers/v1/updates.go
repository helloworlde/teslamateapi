package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
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
func (h *Handler) Updates(c *gin.Context) {

	// define error messages
	const handler = "TeslaMateAPICarsUpdatesV1"
	var CarsUpdatesError1 = "Unable to load updates."

	// getting CarID param from URL
	CarID, ok := requirePositiveIntParam(c, handler, "CarID", c.Param("CarID"))
	if !ok {
		return
	}
	// query options to modify query when collecting data
	ResultPage, ok := optionalIntInRange(c, handler, "page", c.Query("page"), 1, 1, 2147483647)
	if !ok {
		return
	}
	ResultShow, ok := optionalIntInRange(c, handler, "show", c.Query("show"), 100, 1, maxV1PageSize)
	if !ok {
		return
	}

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
	rows, err := h.db.QueryContext(c.Request.Context(), query, CarID, ResultShow, ResultPage)

	// checking for errors in query
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
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
			respond.HandleError(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
			return
		}

		// adjusting to timezone differences from UTC to be userspecific
		update.StartDate = h.timeInTZ(update.StartDate)
		update.EndDate = h.timeInTZ(update.EndDate)

		// appending update to UpdatesData
		UpdatesData = append(UpdatesData, update)
		CarData.CarID = CarID
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
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
	respond.HandleSuccess(c, "TeslaMateAPICarsUpdatesV1", jsonData)
}
