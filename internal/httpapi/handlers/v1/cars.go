package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// TeslaMateAPICarsV1 returns all cars (or one when CarID is set).
//
// @Summary      List cars
// @Description  Returns every car TeslaMate tracks (with details / exterior / settings / stats).
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path      int  false  "TeslaMate cars.id; omit for all cars"
// @Success      200    {object}  dto.V1CarsResponse
// @Failure      401    {object}  dto.ErrorEnvelope  "missing/invalid bearer token"
// @Router       /api/v1/cars      [get]
// @Router       /api/v1/cars/{CarID} [get]
func (h *Handler) Cars(c *gin.Context) {

	// define error messages
	var CarsError1 = "Unable to load cars."

	// getting CarID param from URL
	ParamCarID := c.Param("CarID")
	var CarID int
	if ParamCarID != "" {
		var ok bool
		CarID, ok = requirePositiveIntParam(c, "TeslaMateAPICarsV1", "CarID", ParamCarID)
		if !ok {
			return
		}
	}

	// creating required vars
	var CarsData []dto.V1Car

	// totals are aggregated in a single GROUP BY pass per table and joined
	// once per car, instead of three correlated COUNT(*) subqueries that
	// each forced a per-car index lookup. Same result, but linear in
	// (cars + charging_processes + drives + updates) rows total instead of
	// quadratic on large installs. Note: the upstream query mis-aliased
	// total_updates as `total_charges`; the alias is cosmetic (Scan order is
	// what matters), so leaving it untouched here would also work, but we
	// give it the right name for readers.
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
			COALESCE(cp.cnt, 0) as total_charges,
			COALESCE(d.cnt, 0)  as total_drives,
			COALESCE(u.cnt, 0)  as total_updates
		FROM cars
		LEFT JOIN car_settings ON cars.id = car_settings.id
		LEFT JOIN (SELECT car_id, COUNT(*) AS cnt FROM charging_processes GROUP BY car_id) cp ON cp.car_id = cars.id
		LEFT JOIN (SELECT car_id, COUNT(*) AS cnt FROM drives             GROUP BY car_id) d  ON d.car_id  = cars.id
		LEFT JOIN (SELECT car_id, COUNT(*) AS cnt FROM updates            GROUP BY car_id) u  ON u.car_id  = cars.id
		ORDER BY cars.id;`
	rows, err := h.db.QueryContext(c.Request.Context(), query)

	// checking for errors in query
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsV1", CarsError1, err.Error())
		return
	}

	// defer closing rows
	defer rows.Close()

	// looping through all results
	for rows.Next() {

		// creating car object based on struct
		car := dto.V1Car{}

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
			respond.HandleError(c, "TeslaMateAPICarsV1", CarsError1, err.Error())
			return
		}

		// appending car to CarsData if CarID is 0 or is CarID matches car.CarID
		if CarID == 0 && len(ParamCarID) == 0 || CarID != 0 && CarID == car.CarID {

			// adjusting to timezone differences from UTC to be userspecific
			car.TeslaMateDetails.InsertedAt = h.timeInTZ(car.TeslaMateDetails.InsertedAt)
			car.TeslaMateDetails.UpdatedAt = h.timeInTZ(car.TeslaMateDetails.UpdatedAt)
			car.CarDetails.VINDetails = decodeTeslaModelYVIN(car.CarDetails.Vin)

			CarsData = append(CarsData, car)
		}
	}

	// checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsV1", CarsError1, err.Error())
		return
	}

	//
	// build the data-blob
	jsonData := dto.V1CarsResponse{
		Data: dto.V1CarsData{
			Cars: CarsData,
		},
	}

	// return jsonData
	respond.HandleSuccess(c, "TeslaMateAPICarsV1", jsonData)

}
