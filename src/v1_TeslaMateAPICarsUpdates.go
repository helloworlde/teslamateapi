package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsUpdatesV1 godoc
//
// @Summary List firmware updates for one car
// @Tags V1
// @Produce json
// @Param CarID path int true "Car ID"
// @Success 200 {object} V1JSONEnvelope
// @Failure 200 {object} V1ErrorEnvelope
// @Router /v1/cars/{CarID}/updates [get]
func TeslaMateAPICarsUpdatesV1(c *gin.Context) {

	// define error messages
	var CarsUpdatesError1 = "Unable to load updates."

	// getting CarID param from URL
	CarID := convertStringToInteger(c.Param("CarID"))
	// query options to modify query when collecting data
	ResultPage := convertStringToInteger(c.DefaultQuery("page", "1"))
	ResultShow := convertStringToInteger(c.DefaultQuery("show", "100"))

	// creating structs for /cars/<CarID>/updates
	// Car struct - child of Data
	type Car struct {
		CarID   int        `json:"car_id"`   // smallint
		CarName NullString `json:"car_name"` // text (nullable)
	}
	// Updates struct - child of Data
	type Updates struct {
		UpdateID           int      `json:"update_id"`                    // smallint
		StartDate          string   `json:"start_date"`                   // string
		EndDate            string   `json:"end_date"`                     // string
		Version            string   `json:"version"`                      // string
		ShortVersion       *string  `json:"short_version,omitempty"`      // (added) split_part(version, ' ', 1)
		DurationMin        *float64 `json:"duration_min,omitempty"`       // (added) EXTRACT(EPOCH FROM end-start)/60
		DaysSincePrior     *float64 `json:"days_since_prior,omitempty"`   // (added) days between this and previous start_date
		ActiveDurationDays *float64 `json:"active_duration_days,omitempty"` // (added) days this version was active until next/now
	}
	// Data struct - child of JSONData
	type Data struct {
		Car     Car       `json:"car"`
		Updates []Updates `json:"updates"`
	}
	// JSONData struct - main
	type JSONData struct {
		Data Data `json:"data"`
	}

	// creating required vars
	var (
		UpdatesData []Updates
		CarData     Car
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
		WITH ordered AS (
			SELECT
				updates.id,
				cars.name,
				start_date,
				end_date,
				version,
				LAG(start_date)  OVER (ORDER BY start_date ASC) AS prev_start_date,
				LEAD(start_date) OVER (ORDER BY start_date ASC) AS next_start_date
			FROM updates
			LEFT JOIN cars ON car_id = cars.id
			WHERE car_id = $1 AND end_date IS NOT NULL AND version IS NOT NULL
		)
		SELECT
			id,
			name,
			start_date,
			end_date,
			version,
			split_part(version, ' ', 1) AS short_version,
			EXTRACT(EPOCH FROM (end_date - start_date)) / 60.0 AS duration_min,
			CASE WHEN prev_start_date IS NOT NULL
				THEN EXTRACT(EPOCH FROM (start_date - prev_start_date)) / 86400.0
				ELSE NULL END AS days_since_prior,
			CASE WHEN next_start_date IS NOT NULL
				THEN EXTRACT(EPOCH FROM (next_start_date - start_date)) / 86400.0
				ELSE EXTRACT(EPOCH FROM (NOW() - start_date)) / 86400.0
			END AS active_duration_days
		FROM ordered
		ORDER BY start_date DESC
		LIMIT $2 OFFSET $3;`
	rows, err := db.Query(query, CarID, ResultShow, ResultPage)

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
		update := Updates{}
		var (
			shortVersion       sql.NullString
			durationMin        sql.NullFloat64
			daysSincePrior     sql.NullFloat64
			activeDurationDays sql.NullFloat64
		)

		// scanning row and putting values into the update
		err = rows.Scan(
			&update.UpdateID,
			&CarData.CarName,
			&update.StartDate,
			&update.EndDate,
			&update.Version,
			&shortVersion,
			&durationMin,
			&daysSincePrior,
			&activeDurationDays,
		)

		// checking for errors after scanning
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsUpdatesV1", CarsUpdatesError1, err.Error())
			return
		}

		if shortVersion.Valid && shortVersion.String != "" {
			s := shortVersion.String
			update.ShortVersion = &s
		}
		if durationMin.Valid {
			v := durationMin.Float64
			update.DurationMin = &v
		}
		if daysSincePrior.Valid {
			v := daysSincePrior.Float64
			update.DaysSincePrior = &v
		}
		if activeDurationDays.Valid {
			v := activeDurationDays.Float64
			update.ActiveDurationDays = &v
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
	jsonData := JSONData{
		Data{
			Car:     CarData,
			Updates: UpdatesData,
		},
	}

	// return jsonData
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsUpdatesV1", jsonData)
}
