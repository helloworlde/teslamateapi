package main

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsTirePressureV1 godoc
//
// @Summary Get tire pressure for one car
// @Description Latest TPMS reading + window min/max + per-day history. Pressure unit follows settings.unit_of_pressure (bar default; psi via barToPsi).
// @Tags v1
// @Produce json
// @Param CarID path int true "Car ID"
// @Param window_days query int false "History window length in days (default 30, max 365)"
// @Success 200 {object} V1JSONEnvelope
// @Failure 200 {object} V1ErrorEnvelope
// @Router /v1/cars/{CarID}/tire-pressure [get]
func TeslaMateAPICarsTirePressureV1(c *gin.Context) {
	const errMsg = "Unable to load tire pressure data."
	CarID := convertStringToInteger(c.Param("CarID"))

	// resolve window length (default 30 days)
	windowDays := 30
	if v := c.Query("window_days"); v != "" {
		if parsed := convertStringToInteger(v); parsed > 0 && parsed <= 365 {
			windowDays = parsed
		}
	}

	// Car struct - child of Data
	type Car struct {
		CarID   int        `json:"car_id"`
		CarName NullString `json:"car_name"`
	}
	// Latest tire pressure snapshot
	type Latest struct {
		AsOf string   `json:"as_of"`
		FL   *float64 `json:"fl,omitempty"`
		FR   *float64 `json:"fr,omitempty"`
		RL   *float64 `json:"rl,omitempty"`
		RR   *float64 `json:"rr,omitempty"`
	}
	// PressureSet for min/max corner values
	type PressureSet struct {
		FL *float64 `json:"fl,omitempty"`
		FR *float64 `json:"fr,omitempty"`
		RL *float64 `json:"rl,omitempty"`
		RR *float64 `json:"rr,omitempty"`
	}
	// Window aggregation
	type Window struct {
		Start string      `json:"start"`
		End   string      `json:"end"`
		Min   PressureSet `json:"min"`
		Max   PressureSet `json:"max"`
	}
	// History entry (one per day in window)
	type HistoryEntry struct {
		Date   string   `json:"date"`
		FLAvg  *float64 `json:"fl_avg,omitempty"`
		FRAvg  *float64 `json:"fr_avg,omitempty"`
		RLAvg  *float64 `json:"rl_avg,omitempty"`
		RRAvg  *float64 `json:"rr_avg,omitempty"`
	}
	// Units struct
	type Units struct {
		UnitsLength   string `json:"unit_of_length"`
		UnitsPressure string `json:"unit_of_pressure"`
	}
	// Data struct - child of JSONData
	type Data struct {
		Car     Car            `json:"car"`
		Latest  Latest         `json:"latest"`
		Window  Window         `json:"window"`
		History []HistoryEntry `json:"history"`
		Units   Units          `json:"units"`
	}
	// JSONData wrapper
	type JSONData struct {
		Data Data `json:"data"`
	}

	// car name + units
	var (
		CarName       NullString
		UnitsLength   string
		UnitsPressure string
	)
	err := db.QueryRow(`
		SELECT
			cars.name,
			(SELECT unit_of_length FROM settings LIMIT 1) AS unit_of_length,
			(SELECT COALESCE(unit_of_pressure, '') FROM settings LIMIT 1) AS unit_of_pressure
		FROM cars WHERE id = $1
	`, CarID).Scan(&CarName, &UnitsLength, &UnitsPressure)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, err.Error())
		return
	}

	// latest sample (one row, the most recent reading with at least one TPMS value)
	var (
		latestDate                 sql.NullTime
		latestFL, latestFR         sql.NullFloat64
		latestRL, latestRR         sql.NullFloat64
	)
	err = db.QueryRow(`
		SELECT date, tpms_pressure_fl, tpms_pressure_fr, tpms_pressure_rl, tpms_pressure_rr
		FROM positions
		WHERE car_id = $1
		  AND (tpms_pressure_fl IS NOT NULL OR tpms_pressure_fr IS NOT NULL
		    OR tpms_pressure_rl IS NOT NULL OR tpms_pressure_rr IS NOT NULL)
		ORDER BY date DESC
		LIMIT 1
	`, CarID).Scan(&latestDate, &latestFL, &latestFR, &latestRL, &latestRR)
	if err != nil && err != sql.ErrNoRows {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, err.Error())
		return
	}

	// window aggregation (min/max per corner over last N days)
	var (
		winStart, winEnd                       sql.NullTime
		minFL, minFR, minRL, minRR             sql.NullFloat64
		maxFL, maxFR, maxRL, maxRR             sql.NullFloat64
	)
	winErr := db.QueryRow(`
		SELECT
			MIN(date), MAX(date),
			MIN(tpms_pressure_fl), MIN(tpms_pressure_fr), MIN(tpms_pressure_rl), MIN(tpms_pressure_rr),
			MAX(tpms_pressure_fl), MAX(tpms_pressure_fr), MAX(tpms_pressure_rl), MAX(tpms_pressure_rr)
		FROM positions
		WHERE car_id = $1
		  AND date >= NOW() - make_interval(days => $2::int)
		  AND (tpms_pressure_fl IS NOT NULL OR tpms_pressure_fr IS NOT NULL
		    OR tpms_pressure_rl IS NOT NULL OR tpms_pressure_rr IS NOT NULL)
	`, CarID, windowDays).Scan(
		&winStart, &winEnd,
		&minFL, &minFR, &minRL, &minRR,
		&maxFL, &maxFR, &maxRL, &maxRR,
	)
	if winErr != nil && winErr != sql.ErrNoRows {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, winErr.Error())
		return
	}

	// per-day history
	rows, hErr := db.Query(`
		SELECT to_char(date_trunc('day', date), 'YYYY-MM-DD') AS day,
		       AVG(tpms_pressure_fl)::float8,
		       AVG(tpms_pressure_fr)::float8,
		       AVG(tpms_pressure_rl)::float8,
		       AVG(tpms_pressure_rr)::float8
		FROM positions
		WHERE car_id = $1
		  AND date >= NOW() - make_interval(days => $2::int)
		  AND (tpms_pressure_fl IS NOT NULL OR tpms_pressure_fr IS NOT NULL
		    OR tpms_pressure_rl IS NOT NULL OR tpms_pressure_rr IS NOT NULL)
		GROUP BY 1
		ORDER BY 1 ASC
	`, CarID, windowDays)
	if hErr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, hErr.Error())
		return
	}
	defer rows.Close()

	history := []HistoryEntry{}
	for rows.Next() {
		var (
			day                    string
			flAvg, frAvg, rlAvg, rrAvg sql.NullFloat64
		)
		if err := rows.Scan(&day, &flAvg, &frAvg, &rlAvg, &rrAvg); err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, err.Error())
			return
		}
		entry := HistoryEntry{Date: day}
		if flAvg.Valid {
			v := flAvg.Float64
			entry.FLAvg = &v
		}
		if frAvg.Valid {
			v := frAvg.Float64
			entry.FRAvg = &v
		}
		if rlAvg.Valid {
			v := rlAvg.Float64
			entry.RLAvg = &v
		}
		if rrAvg.Valid {
			v := rrAvg.Float64
			entry.RRAvg = &v
		}
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, err.Error())
		return
	}

	// pressure unit conversion (bar default; convert to psi when configured)
	psi := UnitsPressure == "psi"
	convPtr := func(n sql.NullFloat64) *float64 {
		if !n.Valid {
			return nil
		}
		v := n.Float64
		if psi {
			v = barToPsi(v)
		}
		return &v
	}
	if psi {
		for i := range history {
			if history[i].FLAvg != nil {
				v := barToPsi(*history[i].FLAvg)
				history[i].FLAvg = &v
			}
			if history[i].FRAvg != nil {
				v := barToPsi(*history[i].FRAvg)
				history[i].FRAvg = &v
			}
			if history[i].RLAvg != nil {
				v := barToPsi(*history[i].RLAvg)
				history[i].RLAvg = &v
			}
			if history[i].RRAvg != nil {
				v := barToPsi(*history[i].RRAvg)
				history[i].RRAvg = &v
			}
		}
	}

	latest := Latest{
		FL: convPtr(latestFL),
		FR: convPtr(latestFR),
		RL: convPtr(latestRL),
		RR: convPtr(latestRR),
	}
	if latestDate.Valid {
		latest.AsOf = latestDate.Time.UTC().Format(time.RFC3339)
	}

	window := Window{
		Min: PressureSet{
			FL: convPtr(minFL), FR: convPtr(minFR),
			RL: convPtr(minRL), RR: convPtr(minRR),
		},
		Max: PressureSet{
			FL: convPtr(maxFL), FR: convPtr(maxFR),
			RL: convPtr(maxRL), RR: convPtr(maxRR),
		},
	}
	if winStart.Valid {
		window.Start = winStart.Time.UTC().Format(time.RFC3339)
	}
	if winEnd.Valid {
		window.End = winEnd.Time.UTC().Format(time.RFC3339)
	}

	displayUnit := "bar"
	if psi {
		displayUnit = "psi"
	}

	jsonData := JSONData{
		Data{
			Car: Car{
				CarID:   CarID,
				CarName: CarName,
			},
			Latest:  latest,
			Window:  window,
			History: history,
			Units: Units{
				UnitsLength:   UnitsLength,
				UnitsPressure: displayUnit,
			},
		},
	}
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsTirePressureV1", jsonData)
}
