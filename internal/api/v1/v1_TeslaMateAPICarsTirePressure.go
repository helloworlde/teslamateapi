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
// @Description 最近一次 TPMS 读数 + 时间窗内极值 + 按天历史。每日历史返回 *_min / *_max（用于发现慢漏气）和 *_avg（已弃用，下个 minor 删除）。日历桶按 apicommon.AppUsersTimezone (TZ 环境变量) 切分，避免非 UTC 用户跨午夜 off-by-one。气压单位遵循 settings.unit_of_pressure（默认 bar；psi 由 conv.BarToPsi 换算）。
// @Tags v1
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
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
	// History entry (one per day in window).
	//
	// per docs/v2-api-audit-2026-05.md §1.1: a daily AVG over TPMS readings
	// hides slow leaks (the value the user actually wants to see). We surface
	// MIN / MAX alongside AVG and mark *_avg as deprecated in swagger; the next
	// minor will drop the avg fields. Until then both shapes are returned to
	// keep existing dashboards working.
	type HistoryEntry struct {
		Date  string   `json:"date"`
		FLMin *float64 `json:"fl_min,omitempty"`
		FRMin *float64 `json:"fr_min,omitempty"`
		RLMin *float64 `json:"rl_min,omitempty"`
		RRMin *float64 `json:"rr_min,omitempty"`
		FLMax *float64 `json:"fl_max,omitempty"`
		FRMax *float64 `json:"fr_max,omitempty"`
		RLMax *float64 `json:"rl_max,omitempty"`
		RRMax *float64 `json:"rr_max,omitempty"`
		// Deprecated: prefer fl_min/fl_max — averaging masks slow leaks.
		FLAvg *float64 `json:"fl_avg,omitempty"`
		// Deprecated: prefer fr_min/fr_max — averaging masks slow leaks.
		FRAvg *float64 `json:"fr_avg,omitempty"`
		// Deprecated: prefer rl_min/rl_max — averaging masks slow leaks.
		RLAvg *float64 `json:"rl_avg,omitempty"`
		// Deprecated: prefer rr_min/rr_max — averaging masks slow leaks.
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

	// per-day history.
	//
	// IMPORTANT: bucket by user-visible local day, not by DB session timezone.
	// `date_trunc('day', date)` follows the DB session TZ (typically UTC), so
	// for non-UTC users a sample at 00:30 local time falls into the previous
	// day. We push the timezone in as a parameter and compute the day in that
	// zone (audit §1.1 sub-item).
	tz := tirePressureTimezone()
	rows, hErr := apicommon.DB.Query(`
		SELECT to_char(date_trunc('day', date AT TIME ZONE $3), 'YYYY-MM-DD') AS day,
		       MIN(tpms_pressure_fl)::float8,
		       MIN(tpms_pressure_fr)::float8,
		       MIN(tpms_pressure_rl)::float8,
		       MIN(tpms_pressure_rr)::float8,
		       MAX(tpms_pressure_fl)::float8,
		       MAX(tpms_pressure_fr)::float8,
		       MAX(tpms_pressure_rl)::float8,
		       MAX(tpms_pressure_rr)::float8,
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
	`, CarID, windowDays, tz)
	if hErr != nil {
		apicommon.HandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, hErr.Error())
		return
	}
	defer rows.Close()

	history := []HistoryEntry{}
	for rows.Next() {
		var (
			day                        string
			flMin, frMin, rlMin, rrMin sql.NullFloat64
			flMax, frMax, rlMax, rrMax sql.NullFloat64
			flAvg, frAvg, rlAvg, rrAvg sql.NullFloat64
		)
		if err := rows.Scan(&day,
			&flMin, &frMin, &rlMin, &rrMin,
			&flMax, &frMax, &rlMax, &rrMax,
			&flAvg, &frAvg, &rlAvg, &rrAvg,
		); err != nil {
			apicommon.HandleErrorResponse(c, "TeslaMateAPICarsTirePressureV1", errMsg, err.Error())
			return
		}
		entry := HistoryEntry{Date: day}
		assign := func(dst **float64, src sql.NullFloat64) {
			if src.Valid {
				v := src.Float64
				*dst = &v
			}
		}
		assign(&entry.FLMin, flMin)
		assign(&entry.FRMin, frMin)
		assign(&entry.RLMin, rlMin)
		assign(&entry.RRMin, rrMin)
		assign(&entry.FLMax, flMax)
		assign(&entry.FRMax, frMax)
		assign(&entry.RLMax, rlMax)
		assign(&entry.RRMax, rrMax)
		assign(&entry.FLAvg, flAvg)
		assign(&entry.FRAvg, frAvg)
		assign(&entry.RLAvg, rlAvg)
		assign(&entry.RRAvg, rrAvg)
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
		convertField := func(p **float64) {
			if *p == nil {
				return
			}
			v := conv.BarToPsi(**p)
			*p = &v
		}
		for i := range history {
			convertField(&history[i].FLMin)
			convertField(&history[i].FRMin)
			convertField(&history[i].RLMin)
			convertField(&history[i].RRMin)
			convertField(&history[i].FLMax)
			convertField(&history[i].FRMax)
			convertField(&history[i].RLMax)
			convertField(&history[i].RRMax)
			convertField(&history[i].FLAvg)
			convertField(&history[i].FRAvg)
			convertField(&history[i].RLAvg)
			convertField(&history[i].RRAvg)
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

// tirePressureTimezone returns the IANA timezone the day buckets should align
// with. Order: apicommon.AppUsersTimezone (TZ env-derived) → "UTC". Postgres
// AT TIME ZONE accepts IANA names, so this is a direct passthrough.
func tirePressureTimezone() string {
	if apicommon.AppUsersTimezone != nil {
		return apicommon.AppUsersTimezone.String()
	}
	return "UTC"
}
