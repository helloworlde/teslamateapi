package v2

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

const temperatureConsumptionBucketCelsius = 5

func temperatureConsumptionBucketExpr(column string) string {
	width := strconv.Itoa(temperatureConsumptionBucketCelsius)
	return "(floor(" + column + " / " + width + ") * " + width + ")"
}

// TeslaMateAPICarsStatsConsumptionV2 returns energy-consumption broken down by
// one objective dimension — temperature band, firmware version, or season.
//
// Everything here is measured, not estimated: consumption is derived from the
// rated-range drop * car efficiency over distance, the same energy model used
// by lifetime / summary. There are no thresholds, gas-price assumptions, or
// per-point heuristics.
//
// Performance: a single round-trip that aggregates over `drives` only (one row
// per drive). The `version` grouping range-joins against `updates` (a few dozen
// rows) via a LEAD window. It never scans the multi-million-row `positions`
// table.
//
// Query params:
//
//	group_by   temperature | version | season | month   (default: temperature)
//
// @Summary      Consumption analysis
// @Description  Wh/distance broken down by temperature band, firmware version, season, or month. Objective data only.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID     path   int     true   "TeslaMate cars.id"
// @Param        group_by  query  string  false  "grouping dimension"  Enums(temperature,version,season,month)  default(temperature)
// @Success      200  {object}  dto.V2ConsumptionResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/stats/consumption [get]
func (h *Handler) StatsConsumption(c *gin.Context) {

	const handler = "TeslaMateAPICarsStatsConsumptionV2"
	var ErrMsg = "Unable to load consumption stats."

	CarID, ok := respond.RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}

	groupBy := c.DefaultQuery("group_by", "temperature")

	// Each branch defines: the group key expression, an ordering expression,
	// any extra FROM/JOIN, and an extra WHERE predicate. The energy model and
	// aggregation are shared. tz is only referenced by `season`.
	var (
		keyExpr, ordExpr, joinClause, whereExtra string
		needTZ                                   bool
	)
	switch groupBy {
	case "temperature":
		// 5°C-wide bands keyed by their Celsius lower bound; drives without a
		// recorded outside temperature can't be placed in a band.
		keyExpr = temperatureConsumptionBucketExpr("d.outside_temp_avg")
		ordExpr = temperatureConsumptionBucketExpr("d.outside_temp_avg")
		whereExtra = "AND d.outside_temp_avg IS NOT NULL"
	case "version":
		// Each drive is attributed to the firmware version that was live when it
		// started — derived from the update timeline, an objective record.
		keyExpr = "v.version"
		ordExpr = "MIN(v.start_date)"
		joinClause = "LEFT JOIN ver v ON d.start_date >= v.start_date AND (v.next_start IS NULL OR d.start_date < v.next_start)"
		whereExtra = "AND v.version IS NOT NULL"
	case "season":
		// Meteorological seasons by calendar month in the user's timezone.
		// Semi-objective: assumes the northern-hemisphere convention.
		localDriveStart := localTimestampSQL("d.start_date", "$2")
		needTZ = true
		keyExpr = `CASE EXTRACT(MONTH FROM ` + localDriveStart + `)::int
			WHEN 12 THEN 'winter' WHEN 1 THEN 'winter' WHEN 2 THEN 'winter'
			WHEN 3 THEN 'spring' WHEN 4 THEN 'spring' WHEN 5 THEN 'spring'
			WHEN 6 THEN 'summer' WHEN 7 THEN 'summer' WHEN 8 THEN 'summer'
			ELSE 'autumn' END`
		ordExpr = `MIN(CASE EXTRACT(MONTH FROM ` + localDriveStart + `)::int
			WHEN 3 THEN 1 WHEN 4 THEN 1 WHEN 5 THEN 1
			WHEN 6 THEN 2 WHEN 7 THEN 2 WHEN 8 THEN 2
			WHEN 9 THEN 3 WHEN 10 THEN 3 WHEN 11 THEN 3
			ELSE 4 END)`
	case "month":
		// Calendar months in the user's timezone, keyed "YYYY-MM" (chronological
		// order == lexical order). Objective: just bucketing drives by start date.
		localDriveStart := localTimestampSQL("d.start_date", "$2")
		needTZ = true
		keyExpr = `to_char(` + localDriveStart + `, 'YYYY-MM')`
		ordExpr = `MIN(d.start_date)`
	default:
		respond.HandleErrorV2(c, handler, http.StatusBadRequest, "Invalid group_by.",
			"group_by must be one of: temperature, version, season, month")
		return
	}

	args := []any{CarID}
	if needTZ {
		args = append(args, h.tz.String())
	}

	// `ver` builds firmware-version windows from the (tiny) updates table; it is
	// only referenced when group_by=version. One row per qualifying drive is
	// attributed to a group key, then aggregated into the energy model.
	consumptionFilter := `d.distance > 0
			AND d.start_rated_range_km IS NOT NULL
			AND d.end_rated_range_km IS NOT NULL`
	query := `
		WITH ` + accountingKWhPerPctCTE + `,
		` + accountingChargePriceCTE + `,
		ver AS (
			SELECT version, start_date,
				LEAD(start_date) OVER (ORDER BY start_date) AS next_start
			FROM updates
			WHERE car_id = $1 AND version IS NOT NULL AND end_date IS NOT NULL
		)
		SELECT
			` + keyExpr + `::text AS gkey,
			COUNT(*) FILTER (WHERE ` + consumptionFilter + `) AS trips,
			COALESCE(SUM(d.distance) FILTER (WHERE ` + consumptionFilter + `), 0) AS dist,
			COALESCE(SUM(GREATEST(d.start_rated_range_km - d.end_rated_range_km, 0) * cars.efficiency) FILTER (WHERE ` + consumptionFilter + `), 0) AS kwh,
			COALESCE(SUM(d.distance) FILTER (WHERE d.distance > 0), 0) AS cost_dist,
			CASE WHEN COALESCE(bool_and(
					sp.battery_level IS NOT NULL
					AND ep.battery_level IS NOT NULL
					AND cap.kwh_per_pct IS NOT NULL
				) FILTER (WHERE d.distance > 0), false)
				AND charge_price.cost_per_kwh IS NOT NULL
				THEN SUM(` + driveSOCEnergyKWh + `) FILTER (WHERE d.distance > 0) * charge_price.cost_per_kwh
				ELSE NULL
			END AS estimated_usage_cost
		FROM drives d
		` + joinClause + `
		LEFT JOIN cars ON cars.id = d.car_id
		LEFT JOIN positions sp ON sp.id = d.start_position_id
		LEFT JOIN positions ep ON ep.id = d.end_position_id
		CROSS JOIN cap
		CROSS JOIN charge_price
		WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			` + whereExtra + `
		GROUP BY ` + keyExpr + `, charge_price.cost_per_kwh
		ORDER BY ` + ordExpr + ` ASC`

	rows, err := h.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}
	defer rows.Close()

	type rawGroup struct {
		key      string
		trips    int
		dist     float64
		kwh      float64
		costDist float64
		cost     NullFloat64
	}
	var (
		raw                                []rawGroup
		totalDist, totalKWh, totalCostDist float64
		totalCost                          NullFloat64
		totalCostComplete                  = true
	)
	for rows.Next() {
		var g rawGroup
		if err = rows.Scan(&g.key, &g.trips, &g.dist, &g.kwh, &g.costDist, &g.cost); err != nil {
			respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
			return
		}
		raw = append(raw, g)
		totalDist += g.dist
		totalKWh += g.kwh
		totalCostDist += g.costDist
		if g.costDist > 0 && !g.cost.Valid {
			totalCostComplete = false
		}
		if g.cost.Valid {
			totalCost.Float64 += g.cost.Float64
			totalCost.Valid = true
		}
	}
	if err = rows.Err(); err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	var UnitsLength, UnitsTemperature string
	var CarName NullString
	if err = h.db.QueryRowContext(c.Request.Context(),
		`SELECT (SELECT unit_of_length FROM settings LIMIT 1),
		        (SELECT unit_of_temperature FROM settings LIMIT 1),
		        (SELECT name FROM cars WHERE id = $1)`, CarID).
		Scan(&UnitsLength, &UnitsTemperature, &CarName); err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	// Overall consumption (Wh/km) before unit conversion, used both as a headline
	// figure and as the baseline for each group's delta.
	overall := 0.0
	if totalDist > 0 {
		overall = totalKWh / totalDist * 1000
	}
	if !totalCostComplete {
		totalCost = NullFloat64{}
	}
	overallCostPerDistance := NullFloat64{}
	if totalCost.Valid && totalCostDist > 0 {
		overallCostPerDistance.Float64 = totalCost.Float64 / totalCostDist
		overallCostPerDistance.Valid = true
	}

	groups := make([]dto.V2ConsumptionGroup, 0, len(raw))
	for _, g := range raw {
		consumption := 0.0
		if g.dist > 0 {
			consumption = g.kwh / g.dist * 1000
		}
		costPerDistance := NullFloat64{}
		if g.cost.Valid && g.costDist > 0 {
			costPerDistance.Float64 = g.cost.Float64 / g.costDist
			costPerDistance.Valid = true
		}
		delta := 0.0
		if overall > 0 {
			delta = (consumption - overall) / overall * 100
		}
		grp := dto.V2ConsumptionGroup{
			Key:                g.key,
			Consumption:        consumption,
			TripsCount:         g.trips,
			Distance:           g.dist,
			EnergyKWh:          g.kwh,
			EstimatedUsageCost: g.cost,
			CostPerDistance:    costPerDistance,
			DeltaVsAvgPct:      delta,
		}
		// Temperature bands carry their numeric edges so clients can render
		// "20~25°C" without re-parsing the key.
		if groupBy == "temperature" {
			if g.key != "" {
				low := convert.StrToFloat(g.key)
				grp.TempLow = NullFloat64{NullFloat64: sql.NullFloat64{Float64: low, Valid: true}}
				grp.TempHigh = NullFloat64{NullFloat64: sql.NullFloat64{Float64: low + temperatureConsumptionBucketCelsius, Valid: true}}
			}
		}
		groups = append(groups, grp)
	}

	if UnitsLength == "mi" {
		overall = convert.WhPerKmToWhPerMile(overall)
		if overallCostPerDistance.Valid {
			overallCostPerDistance.Float64 = overallCostPerDistance.Float64 * 1.609344
		}
		for i := range groups {
			groups[i].Consumption = convert.WhPerKmToWhPerMile(groups[i].Consumption)
			groups[i].Distance = convert.KilometersToMiles(groups[i].Distance)
			if groups[i].CostPerDistance.Valid {
				groups[i].CostPerDistance.Float64 = groups[i].CostPerDistance.Float64 * 1.609344
			}
		}
	}
	if UnitsTemperature == "F" {
		for i := range groups {
			if groups[i].TempLow.Valid {
				groups[i].TempLow.Float64 = convert.CelsiusToFahrenheit(groups[i].TempLow.Float64)
			}
			if groups[i].TempHigh.Valid {
				groups[i].TempHigh.Float64 = convert.CelsiusToFahrenheit(groups[i].TempHigh.Float64)
			}
		}
	}

	respond.HandleSuccess(c, handler, dto.V2ConsumptionResponse{
		Data: dto.V2ConsumptionData{
			Car:                       dto.Car{CarID: CarID, CarName: CarName},
			GroupBy:                   groupBy,
			OverallConsumption:        overall,
			OverallEstimatedUsageCost: totalCost,
			OverallCostPerDistance:    overallCostPerDistance,
			Groups:                    groups,
			Units: dto.TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	})
}
