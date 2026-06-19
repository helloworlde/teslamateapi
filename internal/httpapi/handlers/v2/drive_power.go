package v2

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

const maxValidPowerIntervalSeconds = 1.5

type drivePowerInputs struct {
	carID                  int
	carName                NullString
	driveID                int
	startDate              string
	endDate                string
	distance               float64
	durationMin            int
	durationSeconds        float64
	batteryOutputEnergyKWh float64
	regenRecoveredKWh      float64
	outputDurationSeconds  float64
	regenDurationSeconds   float64
	peakOutputPowerKW      int
	peakRegenPowerKW       int
	powerSampleCount       int
	validIntervalCount     int
	ignoredGapCount        int
	validSampleSeconds     float64
	unitsLength            string
	unitsTemperature       string
}

// TeslaMateAPICarsDrivePowerV2 returns observed power / regenerative braking
// statistics for one completed drive. Energy is estimated by integrating the
// full-resolution positions.power time series for this drive: positive power is
// battery output and negative power is regenerative braking. Only intervals
// shorter than 1.5 seconds are integrated so Streaming API gaps do not create
// fake energy. The result is useful for trip analysis, but is not a BMS-metered
// battery-accounting source.
//
// @Summary      Drive power stats
// @Description  Observed battery output and regenerative braking statistics for one drive. Energy is estimated from full positions.power samples and includes data-quality fields so clients can decide whether to show kWh values.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID    path  int  true  "TeslaMate cars.id"
// @Param        DriveID  path  int  true  "drives.id"
// @Success      200  {object}  dto.V2DrivePowerResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      404  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/drives/{DriveID}/power [get]
func (h *Handler) DrivePower(c *gin.Context) {
	const handler = "TeslaMateAPICarsDrivePowerV2"
	var ErrMsg = "Unable to load drive power stats."

	CarID, ok := respond.RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}
	DriveID, ok := respond.RequirePositiveIntParam(c, handler, "drive_id", c.Param("DriveID"))
	if !ok {
		return
	}

	query := `
		WITH selected_drive AS (
			SELECT
				d.id,
				d.car_id,
				d.start_date,
				d.end_date,
				COALESCE(d.distance, 0) AS distance,
				COALESCE(d.duration_min, 0) AS duration_min,
				COALESCE(EXTRACT(EPOCH FROM (d.end_date - d.start_date)), COALESCE(d.duration_min, 0) * 60, 0) AS duration_seconds,
				COALESCE(d.power_max, 0) AS drive_power_max,
				COALESCE(-d.power_min, 0) AS drive_regen_max,
				cars.name AS car_name
			FROM drives d
			JOIN cars ON cars.id = d.car_id
			WHERE d.car_id = $1 AND d.id = $2 AND d.end_date IS NOT NULL
		),
		samples AS (
			SELECT
				p.id,
				p.date,
				p.power,
				LAG(p.date) OVER (ORDER BY p.date ASC, p.id ASC) AS prev_date
			FROM positions p
			JOIN selected_drive sd ON sd.id = p.drive_id AND sd.car_id = p.car_id
			WHERE p.date IS NOT NULL
				AND p.date >= sd.start_date
				AND p.date <= sd.end_date
				AND p.power IS NOT NULL
		),
		intervals AS (
			SELECT
				power,
				EXTRACT(EPOCH FROM (date - prev_date)) AS seconds
			FROM samples
		),
		agg AS (
			SELECT
				COUNT(*)::int AS sample_count,
				COUNT(*) FILTER (WHERE seconds > 0 AND seconds < 1.5)::int AS valid_interval_count,
				COUNT(*) FILTER (WHERE seconds IS NOT NULL AND NOT (seconds > 0 AND seconds < 1.5))::int AS ignored_gap_count,
				COALESCE(SUM(CASE WHEN power > 0 AND seconds > 0 AND seconds < 1.5 THEN power * seconds / 3600 ELSE 0 END), 0) AS output_kwh,
				COALESCE(SUM(CASE WHEN power < 0 AND seconds > 0 AND seconds < 1.5 THEN -power * seconds / 3600 ELSE 0 END), 0) AS regen_kwh,
				COALESCE(SUM(CASE WHEN power > 0 AND seconds > 0 AND seconds < 1.5 THEN seconds ELSE 0 END), 0) AS output_seconds,
				COALESCE(SUM(CASE WHEN power < 0 AND seconds > 0 AND seconds < 1.5 THEN seconds ELSE 0 END), 0) AS regen_seconds,
				COALESCE(SUM(CASE WHEN seconds > 0 AND seconds < 1.5 THEN seconds ELSE 0 END), 0) AS valid_sample_seconds,
				COALESCE(MAX(power) FILTER (WHERE power > 0), 0) AS peak_output_power,
				COALESCE(-MIN(power) FILTER (WHERE power < 0), 0) AS peak_regen_power
			FROM intervals
		)
		SELECT
			sd.id,
			sd.car_id,
			sd.car_name,
			sd.start_date,
			sd.end_date,
			sd.distance,
			sd.duration_min,
			sd.duration_seconds,
			COALESCE(agg.output_kwh, 0),
			COALESCE(agg.regen_kwh, 0),
			COALESCE(agg.output_seconds, 0),
			COALESCE(agg.regen_seconds, 0),
			GREATEST(COALESCE(agg.peak_output_power, 0), sd.drive_power_max)::int,
			GREATEST(COALESCE(agg.peak_regen_power, 0), sd.drive_regen_max)::int,
			COALESCE(agg.sample_count, 0)::int,
			COALESCE(agg.valid_interval_count, 0)::int,
			COALESCE(agg.ignored_gap_count, 0)::int,
			COALESCE(agg.valid_sample_seconds, 0),
			(SELECT unit_of_length FROM settings LIMIT 1),
			(SELECT unit_of_temperature FROM settings LIMIT 1)
		FROM selected_drive sd
		LEFT JOIN agg ON true;`

	var in drivePowerInputs
	err := h.db.QueryRowContext(c.Request.Context(), query, CarID, DriveID).Scan(
		&in.driveID,
		&in.carID,
		&in.carName,
		&in.startDate,
		&in.endDate,
		&in.distance,
		&in.durationMin,
		&in.durationSeconds,
		&in.batteryOutputEnergyKWh,
		&in.regenRecoveredKWh,
		&in.outputDurationSeconds,
		&in.regenDurationSeconds,
		&in.peakOutputPowerKW,
		&in.peakRegenPowerKW,
		&in.powerSampleCount,
		&in.validIntervalCount,
		&in.ignoredGapCount,
		&in.validSampleSeconds,
		&in.unitsLength,
		&in.unitsTemperature,
	)
	if errors.Is(err, sql.ErrNoRows) {
		respond.HandleErrorV2(c, handler, http.StatusNotFound, "Drive not found.", fmt.Sprintf("car_id=%d drive_id=%d", CarID, DriveID))
		return
	}
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	if in.unitsLength == "mi" {
		in.distance = convert.KilometersToMiles(in.distance)
	}
	in.startDate = h.timeInTZ(in.startDate)
	in.endDate = h.timeInTZ(in.endDate)

	respond.HandleSuccess(c, handler, dto.V2DrivePowerResponse{
		Data: buildDrivePowerData(in),
	})
}

func buildDrivePowerData(in drivePowerInputs) dto.V2DrivePowerData {
	outputEnergy := nonNegative(in.batteryOutputEnergyKWh)
	regenEnergy := nonNegative(in.regenRecoveredKWh)
	outputSeconds := nonNegative(in.outputDurationSeconds)
	regenSeconds := nonNegative(in.regenDurationSeconds)
	durationSeconds := nonNegative(in.durationSeconds)
	validSampleSeconds := nonNegative(in.validSampleSeconds)
	coveragePct := 0.0
	if durationSeconds > 0 {
		coveragePct = pct(validSampleSeconds, durationSeconds)
		if coveragePct > 100 {
			coveragePct = 100
		}
	}
	hasUsablePowerSamples := in.powerSampleCount >= 2 && in.validIntervalCount > 0

	return dto.V2DrivePowerData{
		Car: dto.Car{
			CarID:   in.carID,
			CarName: in.carName,
		},
		Drive: dto.V2DrivePowerDrive{
			DriveID:         in.driveID,
			StartDate:       in.startDate,
			EndDate:         in.endDate,
			Distance:        nonNegative(in.distance),
			DurationMin:     in.durationMin,
			DurationSeconds: durationSeconds,
		},
		Metrics: dto.V2DrivePowerMetrics{
			ObservedBatteryOutputEnergyKWh: outputEnergy,
			ObservedRegenRecoveredKWh:      regenEnergy,
			ObservedNetBatteryEnergyKWh:    outputEnergy - regenEnergy,
			RegenShareOfOutputPct:          nullableFloat64(pct(regenEnergy, outputEnergy), outputEnergy > 0),
			RegenShareOfPowerActivityPct:   nullableFloat64(pct(regenEnergy, outputEnergy+regenEnergy), outputEnergy+regenEnergy > 0),
			PeakOutputPowerKW:              in.peakOutputPowerKW,
			PeakRegenPowerKW:               in.peakRegenPowerKW,
			AvgOutputPowerKW:               nullableFloat64(ratio(outputEnergy, outputSeconds/3600), outputSeconds > 0),
			AvgRegenPowerKW:                nullableFloat64(ratio(regenEnergy, regenSeconds/3600), regenSeconds > 0),
			OutputDurationSeconds:          outputSeconds,
			RegenDurationSeconds:           regenSeconds,
		},
		DataQuality: dto.V2DrivePowerDataQuality{
			Confidence:              drivePowerConfidence(coveragePct, hasUsablePowerSamples),
			SampleCoveragePct:       coveragePct,
			PowerSampleCount:        in.powerSampleCount,
			ValidIntervalCount:      in.validIntervalCount,
			IgnoredGapCount:         in.ignoredGapCount,
			ValidSampleSeconds:      validSampleSeconds,
			DriveDurationSeconds:    durationSeconds,
			MaxValidIntervalSeconds: maxValidPowerIntervalSeconds,
			HasPowerSamples:         hasUsablePowerSamples,
		},
		Units: dto.TeslaMateUnits{
			UnitsLength:      in.unitsLength,
			UnitsTemperature: in.unitsTemperature,
		},
	}
}

func drivePowerConfidence(coveragePct float64, hasUsablePowerSamples bool) string {
	if !hasUsablePowerSamples {
		return "unavailable"
	}
	if coveragePct >= 95 {
		return "high"
	}
	if coveragePct >= 80 {
		return "medium"
	}
	return "low"
}
