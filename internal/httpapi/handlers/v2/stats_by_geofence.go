package v2

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
)

// TeslaMateAPICarsStatsByGeofenceV2 aggregates drives / charges / parkings
// by geofence. Drives contribute as either an arrival (end_geofence) or a
// departure (start_geofence). Parkings are bound to start_position.geofence_id
// of the preceding drive's end_position. Records without a geofence are
// dropped — those typically represent a driveby or in-transit position.
//
// @Summary      Stats by geofence
// @Description  Aggregates drives, charges, and parking sessions grouped by geofence.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID       path   int     true   "TeslaMate cars.id"
// @Param        start_date  query  string  false  "RFC3339 lower bound"
// @Param        end_date    query  string  false  "RFC3339 upper bound"
// @Success      200  {object}  dto.V2ByGeofenceResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/stats/by-geofence [get]
func (h *Handler) StatsByGeofence(c *gin.Context) {

	const handler = "TeslaMateAPICarsStatsByGeofenceV2"
	var ErrMsg = "Unable to load by-geofence stats."
	var ErrDate = "Invalid date format."

	CarID, ok := respond.RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}

	parsedStartDate, err := h.parseDate(c.Query("start_date"))
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusBadRequest, ErrDate, err.Error())
		return
	}
	parsedEndDate, err := h.parseDate(c.Query("end_date"))
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusBadRequest, ErrDate, err.Error())
		return
	}

	type GeofenceRow struct {
		GeofenceID               int64   `json:"geofence_id"`
		GeofenceName             string  `json:"geofence_name"`
		DrivesArrived            int     `json:"drives_arrived"`
		DrivesDeparted           int     `json:"drives_departed"`
		ChargesCount             int     `json:"charges_count"`
		ChargesEnergyAddedKWh    float64 `json:"charges_energy_added_kwh"`
		ChargesCost              float64 `json:"charges_cost"`
		ParkingsCount            int     `json:"parkings_count"`
		ParkingsTotalDurationMin int     `json:"parkings_total_duration_min"`
	}
	type Car struct {
		CarID   int        `json:"car_id"`
		CarName NullString `json:"car_name"`
	}
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`
		UnitsTemperature string `json:"unit_of_temperature"`
	}
	type Data struct {
		Car       Car            `json:"car"`
		Geofences []GeofenceRow  `json:"geofences"`
		Units     TeslaMateUnits `json:"units"`
	}
	type JSONData struct {
		Data Data `json:"data"`
	}

	var (
		args      = []any{CarID}
		paramIdx  = 2
		drvFilter = ""
		chFilter  = ""
		pkFilter  = ""
	)
	if parsedStartDate != "" {
		drvFilter += fmt.Sprintf(" AND d.start_date >= $%d", paramIdx)
		chFilter += fmt.Sprintf(" AND cp.start_date >= $%d", paramIdx)
		pkFilter += fmt.Sprintf(" AND dp.park_start >= $%d", paramIdx)
		args = append(args, parsedStartDate)
		paramIdx++
	}
	if parsedEndDate != "" {
		drvFilter += fmt.Sprintf(" AND d.start_date <= $%d", paramIdx)
		chFilter += fmt.Sprintf(" AND cp.start_date <= $%d", paramIdx)
		pkFilter += fmt.Sprintf(" AND dp.park_start <= $%d", paramIdx)
		args = append(args, parsedEndDate)
		paramIdx++
	}

	query := fmt.Sprintf(`
		WITH dep AS (
			SELECT d.start_geofence_id AS gid, COUNT(*) AS cnt
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL AND d.start_geofence_id IS NOT NULL %s
			GROUP BY 1
		),
		arr AS (
			SELECT d.end_geofence_id AS gid, COUNT(*) AS cnt
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL AND d.end_geofence_id IS NOT NULL %s
			GROUP BY 1
		),
		ch AS (
			SELECT cp.geofence_id AS gid,
				COUNT(*) AS cnt,
				COALESCE(SUM(charge_energy_added), 0) AS added,
				COALESCE(SUM(cost), 0) AS cost
			FROM charging_processes cp
			WHERE cp.car_id = $1 AND cp.end_date IS NOT NULL AND cp.geofence_id IS NOT NULL %s
			GROUP BY 1
		),
		dp AS (
			SELECT
				d.id AS drive_id,
				d.end_date AS park_start,
				LEAD(d.start_date) OVER w AS park_end,
				d.end_geofence_id AS park_geofence_id
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			WINDOW w AS (PARTITION BY d.car_id ORDER BY d.start_date ASC)
		),
		pk AS (
			SELECT dp.park_geofence_id AS gid,
				COUNT(*) AS cnt,
				COALESCE(SUM(
					COALESCE(EXTRACT(EPOCH FROM (dp.park_end - dp.park_start))/60, EXTRACT(EPOCH FROM (NOW() - dp.park_start))/60)
				)::int, 0) AS dur
			FROM dp
			WHERE dp.park_geofence_id IS NOT NULL %s
			GROUP BY 1
		),
		keys AS (
			SELECT gid FROM dep UNION
			SELECT gid FROM arr UNION
			SELECT gid FROM ch  UNION
			SELECT gid FROM pk
		)
		SELECT
			k.gid,
			COALESCE(g.name, '') AS geofence_name,
			COALESCE(arr.cnt, 0),
			COALESCE(dep.cnt, 0),
			COALESCE(ch.cnt, 0),
			COALESCE(ch.added, 0),
			COALESCE(ch.cost, 0),
			COALESCE(pk.cnt, 0),
			COALESCE(pk.dur, 0),
			(SELECT unit_of_length FROM settings LIMIT 1),
			(SELECT unit_of_temperature FROM settings LIMIT 1),
			(SELECT name FROM cars WHERE id = $1)
		FROM keys k
		LEFT JOIN dep ON dep.gid = k.gid
		LEFT JOIN arr ON arr.gid = k.gid
		LEFT JOIN ch  ON ch.gid  = k.gid
		LEFT JOIN pk  ON pk.gid  = k.gid
		LEFT JOIN geofences g ON g.id = k.gid
		WHERE k.gid IS NOT NULL
		ORDER BY (COALESCE(arr.cnt, 0) + COALESCE(dep.cnt, 0) + COALESCE(ch.cnt, 0) + COALESCE(pk.cnt, 0)) DESC;`,
		drvFilter, drvFilter, chFilter, pkFilter,
	)

	rows, err := h.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}
	defer rows.Close()

	var (
		out                           []GeofenceRow
		UnitsLength, UnitsTemperature string
		CarName                       NullString
	)

	for rows.Next() {
		row := GeofenceRow{}
		if err = rows.Scan(
			&row.GeofenceID,
			&row.GeofenceName,
			&row.DrivesArrived,
			&row.DrivesDeparted,
			&row.ChargesCount,
			&row.ChargesEnergyAddedKWh,
			&row.ChargesCost,
			&row.ParkingsCount,
			&row.ParkingsTotalDurationMin,
			&UnitsLength,
			&UnitsTemperature,
			&CarName,
		); err != nil {
			respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
			return
		}
		out = append(out, row)
	}
	if err = rows.Err(); err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	respond.HandleSuccess(c, handler, JSONData{
		Data: Data{
			Car:       Car{CarID: CarID, CarName: CarName},
			Geofences: out,
			Units: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	})
}
