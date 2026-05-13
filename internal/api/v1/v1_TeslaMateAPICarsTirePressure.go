package v1

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
	"github.com/tobiasehlert/teslamateapi/internal/conv"
	"github.com/tobiasehlert/teslamateapi/internal/nullx"
)

// TeslaMateAPICarsTirePressureV1 godoc
//
// @Summary 车辆胎压（最近读数 + 窗口）
// @Description 最近一次 TPMS 读数 + 时间窗内极值 + 按天历史。气压单位遵循 settings.unit_of_pressure（默认 bar；psi 由 conv.BarToPsi 换算）。
// @Tags v1
// @Produce json
// @Param CarID path int true "车辆 ID"
// @Param window_days query int false "历史窗口天数（默认 30，最大 365）"
// @Success 200 {object} V1JSONEnvelope
// @Failure 200 {object} V1ErrorEnvelope
// @Router /v1/cars/{CarID}/tire-pressure [get]
func TeslaMateAPICarsTirePressureV1(c *gin.Context) {
	const errMsg = "Unable to load tire pressure data."
	CarID := apicommon.ConvertStringToInteger(c.Param("CarID"))

	// resolve window length (default 30 days)
	windowDays := 30
	if v := c.Query("window_days"); v != "" {
		if parsed := apicommon.ConvertStringToInteger(v); parsed > 0 && parsed <= 365 {
			windowDays = parsed
		}
	}

	// Car struct - child of Data
	type Car struct {
		CarID   int          `json:"car_id"` // 车辆 ID
		CarName nullx.String `json:"car_name"`
	}
	// Latest tire pressure snapshot
	type Latest struct {
		AsOf string   `json:"as_of"`        // 快照时间
		FL   *float64 `json:"fl,omitempty"` // 左前胎压
		FR   *float64 `json:"fr,omitempty"` // 右前胎压
		RL   *float64 `json:"rl,omitempty"` // 左后胎压
		RR   *float64 `json:"rr,omitempty"` // 右后胎压
	}
	// PressureSet for min/max corner values
	type PressureSet struct {
		FL *float64 `json:"fl,omitempty"` // 左前胎压
		FR *float64 `json:"fr,omitempty"` // 右前胎压
		RL *float64 `json:"rl,omitempty"` // 左后胎压
		RR *float64 `json:"rr,omitempty"` // 右后胎压
	}
	// Window aggregation
	type Window struct {
		Start string      `json:"start"` // 起始时间
		End   string      `json:"end"`   // 结束时间
		Min   PressureSet `json:"min"`   // 最小值
		Max   PressureSet `json:"max"`   // 最大值
	}
	// History entry (one per day in window)
	type HistoryEntry struct {
		Date  string   `json:"date"`
		FLAvg *float64 `json:"fl_avg,omitempty"`
		FRAvg *float64 `json:"fr_avg,omitempty"`
		RLAvg *float64 `json:"rl_avg,omitempty"`
		RRAvg *float64 `json:"rr_avg,omitempty"`
	}
	// Units struct
	type Units struct {
		UnitsLength   string `json:"unit_of_length"`   // 长度单位
		UnitsPressure string `json:"unit_of_pressure"` // 气压单位
	}
	// Data struct - child of JSONData
	type Data struct {
		Car     Car            `json:"car"`     // 车辆
		Latest  Latest         `json:"latest"`  // 最近值
		Window  Window         `json:"window"`  // 观察窗口
		History []HistoryEntry `json:"history"` // 历史记录
		Units   Units          `json:"units"`   // 单位
	}
	// JSONData wrapper
	type JSONData struct {
		Data Data `json:"data"` // 响应数据
	}

	// car name + units
	var (
		CarName       nullx.String
		UnitsLength   string
		UnitsPressure string
	)
	err := apicommon.DB.QueryRow(`
		SELECT
			cars.name,
			(SELECT unit_of_length FROM settings LIMIT 1) AS unit_of_length,
			(SELECT COALESCE(unit_of_pressure, 'bar') FROM settings LIMIT 1) AS unit_of_pressure
		FROM cars WHERE id = $1
	`, CarID).Scan(&CarName, &UnitsLength, &UnitsPressure)
	if err != nil {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, err.Error())
		return
	}

	// latest sample (one row, the most recent reading with at least one TPMS value)
	var (
		latestDate         sql.NullTime
		latestFL, latestFR sql.NullFloat64
		latestRL, latestRR sql.NullFloat64
	)
	err = apicommon.DB.QueryRow(`
		SELECT date, tpms_pressure_fl, tpms_pressure_fr, tpms_pressure_rl, tpms_pressure_rr
		FROM positions
		WHERE car_id = $1
		  AND (tpms_pressure_fl IS NOT NULL OR tpms_pressure_fr IS NOT NULL
		    OR tpms_pressure_rl IS NOT NULL OR tpms_pressure_rr IS NOT NULL)
		ORDER BY date DESC
		LIMIT 1
	`, CarID).Scan(&latestDate, &latestFL, &latestFR, &latestRL, &latestRR)
	if err != nil && err != sql.ErrNoRows {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, err.Error())
		return
	}

	// window aggregation (min/max per corner over last N days)
	var (
		winStart, winEnd           sql.NullTime
		minFL, minFR, minRL, minRR sql.NullFloat64
		maxFL, maxFR, maxRL, maxRR sql.NullFloat64
	)
	winErr := apicommon.DB.QueryRow(`
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
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, winErr.Error())
		return
	}

	// per-day history
	rows, hErr := apicommon.DB.Query(`
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
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, hErr.Error())
		return
	}
	defer rows.Close()

	history := []HistoryEntry{}
	for rows.Next() {
		var (
			day                        string
			flAvg, frAvg, rlAvg, rrAvg sql.NullFloat64
		)
		if err := rows.Scan(&day, &flAvg, &frAvg, &rlAvg, &rrAvg); err != nil {
			apicommon.HandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, err.Error())
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
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, err.Error())
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
			v = conv.BarToPsi(v)
		}
		return &v
	}
	if psi {
		for i := range history {
			if history[i].FLAvg != nil {
				v := conv.BarToPsi(*history[i].FLAvg)
				history[i].FLAvg = &v
			}
			if history[i].FRAvg != nil {
				v := conv.BarToPsi(*history[i].FRAvg)
				history[i].FRAvg = &v
			}
			if history[i].RLAvg != nil {
				v := conv.BarToPsi(*history[i].RLAvg)
				history[i].RLAvg = &v
			}
			if history[i].RRAvg != nil {
				v := conv.BarToPsi(*history[i].RRAvg)
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
	apicommon.HandleSuccessResponse(c, "TeslaMateAPICarsTirePressureV1", jsonData)
}
