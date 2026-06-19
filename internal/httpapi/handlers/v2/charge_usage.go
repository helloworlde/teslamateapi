package v2

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

type chargeUsageInputs struct {
	carID                   int
	carName                 NullString
	chargeID                int
	nextChargeID            NullInt64
	chargeStartDate         string
	chargeEndDate           string
	analysisEndDate         string
	isComplete              bool
	durationMin             int
	durationStr             string
	startBatteryLevel       NullInt64
	postChargeBatteryLevel  NullInt64
	endBatteryLevel         NullInt64
	startBatteryKWh         NullFloat64
	postChargeBatteryKWh    NullFloat64
	endBatteryKWh           NullFloat64
	wallEnergyKWh           float64
	chargeEnergyAddedKWh    float64
	chargeCost              float64
	drivingEnergyKWh        float64
	parkingEnergyKWh        float64
	totalDistance           float64
	driveCount              int
	driveDurationMin        int
	hasPreChargeRangeData   bool
	hasPostChargeRangeData  bool
	hasEndRangeData         bool
	drivesRangeDataComplete bool
	unitsLength             string
	unitsTemperature        string
}

// TeslaMateAPICarsChargeUsageV2 returns battery accounting for one charge
// session and the following usage cycle. The cycle starts when the selected
// charge ends and stops at the next charge start. If there is no next charge
// yet, it stops at the latest known position sample and marks is_complete=false.
//
// The accounting model is:
//
//	analysis-start battery inventory
//	  -> driving use + parking use + ending battery inventory + untracked gap
//
// All battery energy is expressed as state-of-charge (battery_level %) times the
// measured battery capacity per percent for this charge
// (charge_energy_added / SOC gained). It deliberately avoids the
// rated_range_km * cars.efficiency conversion, which used a rounded efficiency
// constant and made post-charge inventory exceed pre-charge inventory plus the
// metered charge_energy_added. Driving energy comes from completed drives in the
// cycle (start/end SOC of each drive's positions). Parking energy is estimated
// from SOC drops between charge-end/drive-start and drive-end/next-drive-start
// (or window-end) boundaries, so it avoids scanning the full positions table for
// closed cycles.
//
// @Summary      Charge usage stats
// @Description  Battery accounting for one charge and usage until the next charge. `metrics.untracked_energy_kwh` is the aggregate residual bucket for all cycle energy that cannot be assigned cleanly to driving, parking, or ending battery inventory. `metrics.unmatched_usage_kwh` and `metrics.unmatched_usage_share_of_available_pct` are diagnostic only; unmatched usage is already included in `metrics.untracked_energy_kwh` when present.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID     path  int  true  "TeslaMate cars.id"
// @Param        ChargeID  path  int  true  "charging_processes.id"
// @Success      200  {object}  dto.V2ChargeUsageResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      404  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/charges/{ChargeID}/usage [get]
func (h *Handler) ChargeUsage(c *gin.Context) {
	const handler = "TeslaMateAPICarsChargeUsageV2"
	var ErrMsg = "Unable to load charge usage stats."

	CarID, ok := respond.RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}
	ChargeID, ok := respond.RequirePositiveIntParam(c, handler, "charge_id", c.Param("ChargeID"))
	if !ok {
		return
	}

	var in chargeUsageInputs
	in.carID = CarID
	in.chargeID = ChargeID

	query := `
		WITH selected_charge AS (
			SELECT
				cp.id,
				cp.car_id,
				cp.start_date,
				cp.end_date,
				cp.start_battery_level,
				cp.end_battery_level,
				COALESCE(cp.charge_energy_added, 0) AS charge_energy_added,
				COALESCE(GREATEST(cp.charge_energy_used, cp.charge_energy_added), 0) AS wall_energy,
				COALESCE(cp.cost, 0) AS cost,
				cars.name AS car_name,
				-- Measured battery capacity per 1% of state-of-charge for THIS charge,
				-- derived purely from metered energy and SOC change. This replaces the
				-- old rated_range_km * cars.efficiency conversion: efficiency is a
				-- rounded constant, so range * efficiency systematically overstated
				-- battery energy and made post-charge inventory exceed pre-charge
				-- inventory plus the metered charge_energy_added (energy from nowhere).
				-- Anchoring every inventory figure to charge_energy_added / SOC gain
				-- keeps the accounting consistent with what was physically metered into
				-- the pack. NULL when the charge has no usable SOC gain to calibrate on.
				CASE
					WHEN cp.charge_energy_added > 0
						AND cp.end_battery_level > cp.start_battery_level
					THEN cp.charge_energy_added::float / (cp.end_battery_level - cp.start_battery_level)
					ELSE NULL
				END AS kwh_per_pct
			FROM charging_processes cp
			JOIN cars ON cars.id = cp.car_id
			WHERE cp.car_id = $1 AND cp.id = $2 AND cp.end_date IS NOT NULL
		),
		next_charge AS (
			SELECT cp.*
			FROM charging_processes cp
			JOIN selected_charge sc ON sc.car_id = cp.car_id
			WHERE cp.start_date >= sc.end_date AND cp.id <> sc.id
			ORDER BY cp.start_date ASC
			LIMIT 1
		),
		latest_position AS (
			SELECT
				p.date,
				COALESCE(p.usable_battery_level, p.battery_level) AS battery_level
			FROM positions p
			JOIN selected_charge sc ON sc.car_id = p.car_id
			WHERE NOT EXISTS (SELECT 1 FROM next_charge)
				AND p.date >= sc.end_date
			ORDER BY p.date DESC
			LIMIT 1
		),
		end_snapshot AS (
			SELECT
				nc.start_date AS snapshot_date,
				nc.start_battery_level::bigint AS battery_level,
				(nc.start_battery_level IS NOT NULL) AS has_soc
			FROM next_charge nc
			UNION ALL
			SELECT
				lp.date AS snapshot_date,
				lp.battery_level::bigint AS battery_level,
				(lp.battery_level IS NOT NULL) AS has_soc
			FROM latest_position lp
			WHERE NOT EXISTS (SELECT 1 FROM next_charge)
			UNION ALL
			SELECT
				sc.end_date AS snapshot_date,
				sc.end_battery_level::bigint AS battery_level,
				(sc.end_battery_level IS NOT NULL) AS has_soc
			FROM selected_charge sc
			WHERE NOT EXISTS (SELECT 1 FROM next_charge)
				AND NOT EXISTS (SELECT 1 FROM latest_position)
		),
		drives_in_window AS (
			SELECT
				d.start_date,
				d.end_date,
				d.distance,
				d.duration_min,
				COALESCE(sp.usable_battery_level, sp.battery_level) AS start_soc,
				COALESCE(ep.usable_battery_level, ep.battery_level) AS end_soc
			FROM drives d
			JOIN selected_charge sc ON sc.car_id = d.car_id
			LEFT JOIN next_charge nc ON true
			LEFT JOIN positions sp ON sp.id = d.start_position_id
			LEFT JOIN positions ep ON ep.id = d.end_position_id
			WHERE d.end_date IS NOT NULL
				AND d.start_date >= sc.end_date
				AND (nc.start_date IS NULL OR d.start_date < nc.start_date)
		),
		drive_agg AS (
			SELECT
				COUNT(*)::int AS drive_count,
				COALESCE(SUM(distance), 0) AS total_distance,
				COALESCE(SUM(duration_min), 0)::int AS drive_duration_min,
				COALESCE(SUM(
					CASE WHEN sc.kwh_per_pct IS NOT NULL
						AND d.start_soc IS NOT NULL
						AND d.end_soc IS NOT NULL
					THEN GREATEST(d.start_soc - d.end_soc, 0) * sc.kwh_per_pct
					ELSE 0 END
				), 0) AS driving_energy,
				COALESCE(bool_and(
					sc.kwh_per_pct IS NOT NULL
					AND d.start_soc IS NOT NULL
					AND d.end_soc IS NOT NULL
				), (SELECT kwh_per_pct IS NOT NULL FROM selected_charge)) AS range_data_complete
			FROM drives_in_window d
			CROSS JOIN selected_charge sc
		),
		events AS (
			SELECT
				sc.end_date AS event_date,
				'charge_end' AS event_kind,
				sc.end_battery_level * sc.kwh_per_pct AS energy_kwh,
				0 AS event_order
			FROM selected_charge sc
			UNION ALL
			SELECT
				d.start_date,
				'drive_start',
				d.start_soc * sc.kwh_per_pct,
				1
			FROM drives_in_window d
			CROSS JOIN selected_charge sc
			UNION ALL
			SELECT
				d.end_date,
				'drive_end',
				d.end_soc * sc.kwh_per_pct,
				2
			FROM drives_in_window d
			CROSS JOIN selected_charge sc
			UNION ALL
			SELECT
				es.snapshot_date,
				'window_end',
				es.battery_level * sc.kwh_per_pct,
				3
			FROM end_snapshot es
			CROSS JOIN selected_charge sc
		),
		ordered_events AS (
			SELECT
				event_kind,
				energy_kwh,
				LAG(event_kind) OVER (ORDER BY event_date ASC, event_order ASC) AS prev_kind,
				LAG(energy_kwh) OVER (ORDER BY event_date ASC, event_order ASC) AS prev_energy_kwh
			FROM events
			WHERE event_date IS NOT NULL AND energy_kwh IS NOT NULL
		),
		parking AS (
			SELECT COALESCE(SUM(
				CASE
					WHEN prev_kind IN ('charge_end', 'drive_end')
						AND event_kind IN ('drive_start', 'window_end')
					THEN GREATEST(prev_energy_kwh - energy_kwh, 0)
					ELSE 0
				END
			), 0) AS parking_energy
			FROM ordered_events
		)
		SELECT
			sc.car_name,
			sc.start_date,
			sc.end_date,
			nc.id,
			es.snapshot_date,
			(nc.id IS NOT NULL) AS is_complete,
			span.duration_min,
			(span.duration_min / 60)::text || ':' || LPAD((span.duration_min % 60)::text, 2, '0') AS duration_str,
			sc.start_battery_level::bigint,
			sc.end_battery_level::bigint,
			es.battery_level,
			sc.start_battery_level * sc.kwh_per_pct AS start_battery_kwh,
			sc.end_battery_level * sc.kwh_per_pct AS post_charge_battery_kwh,
			es.battery_level * sc.kwh_per_pct AS end_battery_kwh,
			sc.wall_energy,
			sc.charge_energy_added,
			sc.cost,
			COALESCE(da.driving_energy, 0),
			COALESCE(pk.parking_energy, 0),
			COALESCE(da.total_distance, 0),
			COALESCE(da.drive_count, 0),
			COALESCE(da.drive_duration_min, 0),
			(sc.start_battery_level IS NOT NULL AND sc.kwh_per_pct IS NOT NULL),
			(sc.end_battery_level IS NOT NULL AND sc.kwh_per_pct IS NOT NULL),
			(es.has_soc AND sc.kwh_per_pct IS NOT NULL),
			COALESCE(da.range_data_complete, true),
			(SELECT unit_of_length FROM settings LIMIT 1),
			(SELECT unit_of_temperature FROM settings LIMIT 1)
		FROM selected_charge sc
		LEFT JOIN next_charge nc ON true
		CROSS JOIN end_snapshot es
		CROSS JOIN LATERAL (
			SELECT COALESCE(EXTRACT(EPOCH FROM (es.snapshot_date - sc.end_date))/60, 0)::int AS duration_min
		) span
		LEFT JOIN drive_agg da ON true
		LEFT JOIN parking pk ON true;`

	err := h.db.QueryRowContext(c.Request.Context(), query, CarID, ChargeID).Scan(
		&in.carName,
		&in.chargeStartDate,
		&in.chargeEndDate,
		&in.nextChargeID,
		&in.analysisEndDate,
		&in.isComplete,
		&in.durationMin,
		&in.durationStr,
		&in.startBatteryLevel,
		&in.postChargeBatteryLevel,
		&in.endBatteryLevel,
		&in.startBatteryKWh,
		&in.postChargeBatteryKWh,
		&in.endBatteryKWh,
		&in.wallEnergyKWh,
		&in.chargeEnergyAddedKWh,
		&in.chargeCost,
		&in.drivingEnergyKWh,
		&in.parkingEnergyKWh,
		&in.totalDistance,
		&in.driveCount,
		&in.driveDurationMin,
		&in.hasPreChargeRangeData,
		&in.hasPostChargeRangeData,
		&in.hasEndRangeData,
		&in.drivesRangeDataComplete,
		&in.unitsLength,
		&in.unitsTemperature,
	)
	if errors.Is(err, sql.ErrNoRows) {
		respond.HandleErrorV2(c, handler, http.StatusNotFound, "Charge not found.", fmt.Sprintf("car_id=%d charge_id=%d", CarID, ChargeID))
		return
	}
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	if in.unitsLength == "mi" {
		in.totalDistance = convert.KilometersToMiles(in.totalDistance)
	}
	in.chargeStartDate = h.timeInTZ(in.chargeStartDate)
	in.chargeEndDate = h.timeInTZ(in.chargeEndDate)
	in.analysisEndDate = h.timeInTZ(in.analysisEndDate)

	respond.HandleSuccess(c, handler, dto.V2ChargeUsageResponse{
		Data: buildChargeUsageData(in),
	})
}

func buildChargeUsageData(in chargeUsageInputs) dto.V2ChargeUsageData {
	startBatteryEnergy := nonNegativeNullable(in.startBatteryKWh)
	postChargeEnergy := nonNegativeNullable(in.postChargeBatteryKWh)
	endBatteryEnergy := nonNegativeNullable(in.endBatteryKWh)
	wallEnergy := nonNegative(in.wallEnergyKWh)
	chargeAddedEnergy := nonNegative(in.chargeEnergyAddedKWh)
	chargeCost := nonNegative(in.chargeCost)
	drivingEnergy := nonNegative(in.drivingEnergyKWh)
	parkingEnergy := nonNegative(in.parkingEnergyKWh)
	reconciliationDataComplete := in.hasPreChargeRangeData && in.hasPostChargeRangeData
	expectedPostChargeEnergy := 0.0
	if reconciliationDataComplete {
		expectedPostChargeEnergy = startBatteryEnergy + chargeAddedEnergy
	}
	availableEnergy := postChargeEnergy
	inventoryReconciliationDelta := postChargeEnergy - expectedPostChargeEnergy
	inventoryReconciliationGain := 0.0
	inventoryReconciliationLoss := 0.0
	if reconciliationDataComplete {
		inventoryReconciliationGain = nonNegative(inventoryReconciliationDelta)
		inventoryReconciliationLoss = nonNegative(-inventoryReconciliationDelta)
	}
	trackedUsageEnergy := drivingEnergy + parkingEnergy
	batteryUsedEnergy := nonNegative(availableEnergy - endBatteryEnergy)
	untrackedEnergy := nonNegative(availableEnergy - trackedUsageEnergy - endBatteryEnergy)
	unmatchedUsageEnergy := nonNegative(trackedUsageEnergy + endBatteryEnergy - availableEnergy)
	residualEnergy := untrackedEnergy + unmatchedUsageEnergy
	chargingLossEnergy := nonNegative(wallEnergy - chargeAddedEnergy)
	accountedEnergy := trackedUsageEnergy + endBatteryEnergy
	hasAvailableEnergy := in.hasPostChargeRangeData
	hasEndEnergy := in.hasEndRangeData
	hasDriveEnergy := in.drivesRangeDataComplete
	hasBatteryUsedEnergy := hasAvailableEnergy && hasEndEnergy
	hasCycleAccounting := hasAvailableEnergy && hasEndEnergy && hasDriveEnergy
	cycleDrivingEnergy := drivingEnergy
	cycleParkingEnergy := parkingEnergy
	cycleEndBatteryEnergy := endBatteryEnergy
	unmatchedDrivingEnergy := 0.0
	unmatchedParkingEnergy := 0.0
	unmatchedEndBatteryEnergy := 0.0
	if unmatchedUsageEnergy > 0 && accountedEnergy > 0 {
		cycleDrivingEnergy = drivingEnergy * ratio(availableEnergy, accountedEnergy)
		cycleParkingEnergy = parkingEnergy * ratio(availableEnergy, accountedEnergy)
		cycleEndBatteryEnergy = endBatteryEnergy * ratio(availableEnergy, accountedEnergy)
		unmatchedDrivingEnergy = drivingEnergy - cycleDrivingEnergy
		unmatchedParkingEnergy = parkingEnergy - cycleParkingEnergy
		unmatchedEndBatteryEnergy = endBatteryEnergy - cycleEndBatteryEnergy
	}

	nodes := []dto.V2ChargeUsageNode{
		makeChargeUsageNode("wall_input", "Wall input", wallEnergy, availableEnergy),
		makeChargeUsageNode("charge_added", "Charge added", chargeAddedEnergy, availableEnergy),
	}
	if in.hasPreChargeRangeData {
		nodes = append(nodes, makeChargeUsageNode("pre_charge_battery_inventory", "Pre-charge battery inventory", startBatteryEnergy, availableEnergy))
	}
	if hasAvailableEnergy {
		nodes = append(nodes,
			makeChargeUsageNode("analysis_start_battery_inventory", "Analysis start battery inventory", postChargeEnergy, availableEnergy),
			makeChargeUsageNode("cycle_available", "Available for cycle", availableEnergy, availableEnergy),
		)
	}
	if hasAvailableEnergy && hasDriveEnergy {
		nodes = append(nodes, makeChargeUsageNode("driving_usage", "Driving use", drivingEnergy, availableEnergy))
	}
	if hasCycleAccounting {
		nodes = append(nodes,
			makeChargeUsageNode("parking_usage", "Parking use", parkingEnergy, availableEnergy),
			makeChargeUsageNode("end_battery_inventory", "Ending battery inventory", endBatteryEnergy, availableEnergy),
		)
	}
	if reconciliationDataComplete {
		nodes = append(nodes, makeChargeUsageNode("charge_accounting_pool", "Pre-charge plus added energy", expectedPostChargeEnergy, availableEnergy))
	}
	if chargingLossEnergy > 0 {
		nodes = append(nodes, makeChargeUsageNode("charging_loss", "Charging loss", chargingLossEnergy, wallEnergy))
	}
	if inventoryReconciliationGain > 0 || inventoryReconciliationLoss > 0 {
		nodes = append(nodes, makeChargeUsageNode(
			"inventory_reconciliation_delta",
			"Inventory reconciliation delta",
			inventoryReconciliationGain+inventoryReconciliationLoss,
			availableEnergy,
		))
	}
	if hasCycleAccounting && untrackedEnergy > 0 {
		nodes = append(nodes, makeChargeUsageNode("untracked_energy", "Untracked energy", untrackedEnergy, availableEnergy))
	}
	if hasCycleAccounting && unmatchedUsageEnergy > 0 {
		nodes = append(nodes, makeChargeUsageNode("unmatched_usage", "Usage gap", unmatchedUsageEnergy, availableEnergy))
	}

	links := []dto.V2ChargeUsageLink{
		makeChargeUsageLink("wall_input", "charge_added", chargeAddedEnergy, wallEnergy),
	}
	if hasAvailableEnergy && hasDriveEnergy {
		links = append(links, makeChargeUsageLink("cycle_available", "driving_usage", cycleDrivingEnergy, availableEnergy))
	}
	if hasCycleAccounting {
		links = append(links,
			makeChargeUsageLink("cycle_available", "parking_usage", cycleParkingEnergy, availableEnergy),
			makeChargeUsageLink("cycle_available", "end_battery_inventory", cycleEndBatteryEnergy, availableEnergy),
		)
	}
	if reconciliationDataComplete {
		links = append(links,
			makeChargeUsageLink("pre_charge_battery_inventory", "charge_accounting_pool", startBatteryEnergy, startBatteryEnergy),
			makeChargeUsageLink("charge_added", "charge_accounting_pool", chargeAddedEnergy, chargeAddedEnergy),
		)
		if inventoryReconciliationGain > 0 {
			links = append(links,
				makeChargeUsageLink("charge_accounting_pool", "analysis_start_battery_inventory", expectedPostChargeEnergy, expectedPostChargeEnergy),
				makeChargeUsageLink("inventory_reconciliation_delta", "analysis_start_battery_inventory", inventoryReconciliationGain, inventoryReconciliationGain),
			)
		} else {
			links = append(links, makeChargeUsageLink("charge_accounting_pool", "analysis_start_battery_inventory", postChargeEnergy, expectedPostChargeEnergy))
			if inventoryReconciliationLoss > 0 {
				links = append(links, makeChargeUsageLink("charge_accounting_pool", "inventory_reconciliation_delta", inventoryReconciliationLoss, expectedPostChargeEnergy))
			}
		}
	} else if hasAvailableEnergy {
		links = append(links, makeChargeUsageLink("analysis_start_battery_inventory", "cycle_available", postChargeEnergy, postChargeEnergy))
	}
	if reconciliationDataComplete && hasAvailableEnergy {
		links = append(links, makeChargeUsageLink("analysis_start_battery_inventory", "cycle_available", postChargeEnergy, postChargeEnergy))
	}
	if chargingLossEnergy > 0 {
		links = append(links, makeChargeUsageLink("wall_input", "charging_loss", chargingLossEnergy, wallEnergy))
	}
	if hasCycleAccounting && untrackedEnergy > 0 {
		links = append(links, makeChargeUsageLink("cycle_available", "untracked_energy", untrackedEnergy, availableEnergy))
	}
	if hasCycleAccounting && unmatchedDrivingEnergy > 0 {
		links = append(links, makeChargeUsageLink("unmatched_usage", "driving_usage", unmatchedDrivingEnergy, unmatchedUsageEnergy))
	}
	if hasCycleAccounting && unmatchedParkingEnergy > 0 {
		links = append(links, makeChargeUsageLink("unmatched_usage", "parking_usage", unmatchedParkingEnergy, unmatchedUsageEnergy))
	}
	if hasCycleAccounting && unmatchedEndBatteryEnergy > 0 {
		links = append(links, makeChargeUsageLink("unmatched_usage", "end_battery_inventory", unmatchedEndBatteryEnergy, unmatchedUsageEnergy))
	}

	balanceStatus := "partial"
	if hasCycleAccounting {
		balanceStatus = "balanced"
		if chargeUsageHasInventoryMismatch(inventoryReconciliationDelta, reconciliationDataComplete) {
			balanceStatus = "inventory_reconciliation_mismatch"
		} else if unmatchedUsageEnergy > 0 {
			balanceStatus = "usage_exceeds_available_energy"
		} else if untrackedEnergy > 0 {
			balanceStatus = "has_untracked_energy"
		}
	}

	return dto.V2ChargeUsageData{
		Car: dto.Car{
			CarID:   in.carID,
			CarName: in.carName,
		},
		Cycle: dto.V2ChargeUsageCycle{
			ChargeID:          in.chargeID,
			NextChargeID:      in.nextChargeID,
			ChargeStartDate:   in.chargeStartDate,
			ChargeEndDate:     in.chargeEndDate,
			AnalysisStartDate: in.chargeEndDate,
			AnalysisEndDate:   in.analysisEndDate,
			IsComplete:        in.isComplete,
			DurationMin:       in.durationMin,
			DurationStr:       in.durationStr,
		},
		Battery: dto.V2ChargeUsageBattery{
			StartBatteryLevel:      in.startBatteryLevel,
			PostChargeBatteryLevel: in.postChargeBatteryLevel,
			EndBatteryLevel:        in.endBatteryLevel,
			AddedBatteryLevel:      nullableInt64Delta(in.postChargeBatteryLevel, in.startBatteryLevel),
			UsedBatteryLevel:       nullableInt64Delta(in.postChargeBatteryLevel, in.endBatteryLevel),
			StartBatteryEnergyKWh:  in.startBatteryKWh,
			PostChargeEnergyKWh:    in.postChargeBatteryKWh,
			EndBatteryEnergyKWh:    in.endBatteryKWh,
		},
		Metrics: dto.V2ChargeUsageMetrics{
			WallEnergyKWh:                     wallEnergy,
			ChargeEnergyAddedKWh:              chargeAddedEnergy,
			ChargingLossEnergyKWh:             chargingLossEnergy,
			ChargeCost:                        chargeCost,
			WallCostPerKWh:                    ratio(chargeCost, wallEnergy),
			ChargeCostPerKWh:                  ratio(chargeCost, chargeAddedEnergy),
			InventoryExpectedEnergyKWh:        nullableFloat64(expectedPostChargeEnergy, reconciliationDataComplete),
			InventoryReconciliationKWh:        nullableFloat64(inventoryReconciliationDelta, reconciliationDataComplete),
			VehicleAvailableEnergyKWh:         nullableFloat64(availableEnergy, hasAvailableEnergy),
			DrivingEnergyKWh:                  nullableFloat64(drivingEnergy, hasDriveEnergy),
			ParkingEnergyKWh:                  nullableFloat64(parkingEnergy, hasCycleAccounting),
			EndBatteryEnergyKWh:               nullableFloat64(endBatteryEnergy, hasEndEnergy),
			UntrackedEnergyKWh:                nullableFloat64(residualEnergy, hasCycleAccounting),
			UnmatchedUsageKWh:                 nullableFloat64(unmatchedUsageEnergy, hasCycleAccounting),
			UnmatchedUsageShareOfAvailablePct: nullableFloat64(pct(unmatchedUsageEnergy, availableEnergy), hasCycleAccounting),
			BatteryUsedEnergyKWh:              nullableFloat64(batteryUsedEnergy, hasBatteryUsedEnergy),
			BatteryUsedRatePct:                nullableFloat64(pct(batteryUsedEnergy, availableEnergy), hasBatteryUsedEnergy),
			TrackedUsageRatePct:               nullableFloat64(pct(trackedUsageEnergy, availableEnergy), hasCycleAccounting),
			DrivingShareOfAvailablePct:        nullableFloat64(pct(drivingEnergy, availableEnergy), hasAvailableEnergy && hasDriveEnergy),
			ParkingShareOfAvailablePct:        nullableFloat64(pct(parkingEnergy, availableEnergy), hasCycleAccounting),
			RemainingShareOfAvailablePct:      nullableFloat64(pct(endBatteryEnergy, availableEnergy), hasBatteryUsedEnergy),
			UntrackedShareOfAvailablePct:      nullableFloat64(pct(residualEnergy, availableEnergy), hasCycleAccounting),
			TotalDistance:                     nonNegative(in.totalDistance),
			DriveCount:                        in.driveCount,
			DriveDurationMin:                  in.driveDurationMin,
			DrivingEnergyPerDistanceWh:        nullableFloat64(ratio(drivingEnergy*1000, nonNegative(in.totalDistance)), hasDriveEnergy),
		},
		Nodes: nodes,
		Links: links,
		Units: dto.TeslaMateUnits{
			UnitsLength:      in.unitsLength,
			UnitsTemperature: in.unitsTemperature,
		},
		DataQuality: dto.V2ChargeUsageDataQuality{
			AccountingStatus:           chargeUsageAccountingStatus(in),
			RangeDataComplete:          in.hasPostChargeRangeData && in.hasEndRangeData && in.drivesRangeDataComplete,
			ReconciliationDataComplete: reconciliationDataComplete,
			HasPreChargeRangeData:      in.hasPreChargeRangeData,
			HasPostChargeRangeData:     in.hasPostChargeRangeData,
			HasEndRangeData:            in.hasEndRangeData,
			DrivesRangeDataComplete:    in.drivesRangeDataComplete,
			PartialFields:              chargeUsagePartialFields(in),
		},
		BalanceStatus: balanceStatus,
	}
}

func chargeUsagePartialFields(in chargeUsageInputs) []string {
	var fields []string
	seen := map[string]bool{}
	appendFields := func(names ...string) {
		for _, name := range names {
			if !seen[name] {
				seen[name] = true
				fields = append(fields, name)
			}
		}
	}
	if !in.hasPreChargeRangeData {
		appendFields("start_battery_energy_kwh", "inventory_expected_energy_kwh", "inventory_reconciliation_kwh")
	}
	if !in.hasPostChargeRangeData {
		appendFields(
			"post_charge_battery_energy_kwh",
			"vehicle_available_energy_kwh",
			"battery_used_energy_kwh",
			"battery_used_rate_pct",
			"tracked_usage_rate_pct",
			"driving_share_of_available_pct",
			"parking_share_of_available_pct",
			"remaining_share_of_available_pct",
			"untracked_share_of_available_pct",
		)
	}
	if !in.hasEndRangeData {
		appendFields("end_battery_energy_kwh", "battery_used_energy_kwh", "battery_used_rate_pct", "remaining_share_of_available_pct", "parking_energy_kwh", "untracked_energy_kwh", "unmatched_usage_kwh", "unmatched_usage_share_of_available_pct")
	}
	if !in.drivesRangeDataComplete {
		appendFields("driving_energy_kwh", "parking_energy_kwh", "untracked_energy_kwh", "unmatched_usage_kwh", "unmatched_usage_share_of_available_pct", "tracked_usage_rate_pct", "driving_share_of_available_pct", "driving_energy_per_distance_wh")
	}
	return fields
}

func chargeUsageAccountingStatus(in chargeUsageInputs) string {
	if in.hasPostChargeRangeData && in.hasEndRangeData && in.drivesRangeDataComplete {
		return "complete"
	}
	return "partial"
}

func makeChargeUsageNode(id, label string, energy, availableEnergy float64) dto.V2ChargeUsageNode {
	return dto.V2ChargeUsageNode{
		ID:                 id,
		Label:              label,
		EnergyKWh:          nonNegative(energy),
		PercentOfAvailable: pct(energy, availableEnergy),
	}
}

func makeChargeUsageLink(source, target string, energy, sourceTotal float64) dto.V2ChargeUsageLink {
	return dto.V2ChargeUsageLink{
		Source:          source,
		Target:          target,
		EnergyKWh:       nonNegative(energy),
		PercentOfSource: pct(energy, sourceTotal),
	}
}

func nonNegativeNullable(value NullFloat64) float64 {
	if !value.Valid || value.Float64 < 0 {
		return 0
	}
	return value.Float64
}

func chargeUsageHasInventoryMismatch(delta float64, valid bool) bool {
	const toleranceKWh = 0.5
	return valid && math.Abs(delta) > toleranceKWh
}

func nullableFloat64(value float64, valid bool) NullFloat64 {
	if !valid {
		return NullFloat64{}
	}
	return NullFloat64{NullFloat64: sql.NullFloat64{Float64: value, Valid: true}}
}

func nullableInt64Delta(a, b NullInt64) NullInt64 {
	if !a.Valid || !b.Valid {
		return NullInt64{}
	}
	return NullInt64{NullInt64: sql.NullInt64{Int64: a.Int64 - b.Int64, Valid: true}}
}
