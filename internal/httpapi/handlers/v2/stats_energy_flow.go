package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

type energyFlowInputs struct {
	carID             int
	carName           NullString
	totalDistance     float64
	wallEnergyKWh     float64
	vehicleAddedKWh   float64
	totalChargingCost float64
	drivingEnergyKWh  float64
	parkingEnergyKWh  float64
	startBatteryKWh   float64
	endBatteryKWh     float64
	unitsLength       string
	unitsTemperature  string
}

// TeslaMateAPICarsStatsEnergyFlowV2 returns a canonical energy/cost flow for
// client Sankey visualisations. It starts from wall-side charging input, splits
// that into vehicle-added energy and charging loss, then splits vehicle energy
// into vehicle-side availability. Vehicle-side availability is the recorded
// vehicle-added energy and is split into driving, parking, ending battery
// inventory, and unmetered residual loss. Starting battery inventory is exposed
// as context but does not reduce recorded charging cost allocation.
//
// Cost allocation uses the wall-side average price
// (charging_processes.cost / wall energy used), so flow costs reconcile to the
// recorded charging bill when wall-side energy is available. The response also
// exposes charging_cost_per_kwh based on vehicle-added energy for users who
// prefer the TeslaMate charge-energy-added denominator.
//
// @Summary      Energy and cost flow stats
// @Description  Lifetime energy and cost accounting model for Sankey charts.
// @Tags         v2
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path  int  true  "TeslaMate cars.id"
// @Success      200  {object}  dto.V2EnergyFlowResponse
// @Failure      400  {object}  dto.ErrorEnvelope
// @Failure      500  {object}  dto.ErrorEnvelope
// @Router       /api/v2/cars/{CarID}/stats/energy-flow [get]
func (h *Handler) StatsEnergyFlow(c *gin.Context) {
	const handler = "TeslaMateAPICarsStatsEnergyFlowV2"
	var ErrMsg = "Unable to load energy flow stats."

	CarID, ok := respond.RequirePositiveIntParam(c, handler, "car_id", c.Param("CarID"))
	if !ok {
		return
	}

	var in energyFlowInputs
	in.carID = CarID

	query := `
		WITH cap AS (
			-- Measured kWh per 1% of state-of-charge across this car's charging
			-- history: metered charge_energy_added divided by the SOC actually
			-- gained. Replaces rated_range_km * cars.efficiency, which used a
			-- rounded efficiency constant and systematically overstated battery
			-- energy. NULL when there is no calibratable charge.
			SELECT CASE
				WHEN SUM(end_battery_level - start_battery_level)
					FILTER (WHERE end_battery_level > start_battery_level) > 0
				THEN SUM(charge_energy_added)
						FILTER (WHERE end_battery_level > start_battery_level)
					/ SUM(end_battery_level - start_battery_level)
						FILTER (WHERE end_battery_level > start_battery_level)
				ELSE NULL
			END AS kwh_per_pct
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL
		),
		d AS (
			SELECT
				COALESCE(SUM(distance), 0) AS total_km,
				COALESCE(SUM(
					CASE WHEN sp.id IS NOT NULL AND ep.id IS NOT NULL
					THEN GREATEST(
						COALESCE(sp.usable_battery_level, sp.battery_level)
						- COALESCE(ep.usable_battery_level, ep.battery_level), 0)
					ELSE 0 END
				), 0) AS total_pct
			FROM drives
			LEFT JOIN positions sp ON sp.id = drives.start_position_id
			LEFT JOIN positions ep ON ep.id = drives.end_position_id
			WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL
		),
		ch AS (
			SELECT
				COALESCE(SUM(charge_energy_added), 0) AS total_added,
				COALESCE(SUM(GREATEST(charge_energy_used, charge_energy_added)), 0) AS total_used,
				COALESCE(SUM(cost), 0) AS total_cost
			FROM charging_processes
			WHERE car_id = $1 AND end_date IS NOT NULL
		),
		dp AS (
			SELECT
				d.end_date AS park_start,
				LEAD(d.start_date) OVER w AS park_end,
				d.end_position_id,
				LEAD(d.start_position_id) OVER w AS next_start_position_id
			FROM drives d
			WHERE d.car_id = $1 AND d.end_date IS NOT NULL
			WINDOW w AS (PARTITION BY d.car_id ORDER BY d.start_date ASC)
		),
		pk AS (
			SELECT
				COALESCE(SUM(
					CASE
						WHEN sp.id IS NOT NULL AND ep.id IS NOT NULL
						AND COALESCE(sp.usable_battery_level, sp.battery_level)
							> COALESCE(ep.usable_battery_level, ep.battery_level)
						AND NOT EXISTS(SELECT 1 FROM charging_processes cp
							WHERE cp.car_id = $1 AND cp.start_date >= dp.park_start
							AND (dp.park_end IS NULL OR cp.start_date < dp.park_end))
						THEN COALESCE(sp.usable_battery_level, sp.battery_level)
							- COALESCE(ep.usable_battery_level, ep.battery_level)
						ELSE 0
					END
				), 0) AS total_pct
			FROM dp
			LEFT JOIN positions sp ON sp.id = dp.end_position_id
			LEFT JOIN positions ep ON ep.id = dp.next_start_position_id
		),
		battery_samples AS (
			SELECT
				p.date,
				COALESCE(p.usable_battery_level, p.battery_level) AS soc
			FROM positions p
			WHERE p.car_id = $1
				AND p.date IS NOT NULL
				AND COALESCE(p.usable_battery_level, p.battery_level) IS NOT NULL
		),
		bat AS (
			SELECT
				COALESCE((array_agg(soc ORDER BY date ASC))[1], 0) AS start_soc,
				COALESCE((array_agg(soc ORDER BY date DESC))[1], 0) AS end_soc
			FROM battery_samples
		)
		SELECT
			(SELECT name FROM cars WHERE id = $1),
			COALESCE(d.total_km, 0),
			COALESCE(ch.total_used, 0),
			COALESCE(ch.total_added, 0),
			COALESCE(ch.total_cost, 0),
			COALESCE(d.total_pct * cap.kwh_per_pct, 0),
			COALESCE(pk.total_pct * cap.kwh_per_pct, 0),
			COALESCE(bat.start_soc * cap.kwh_per_pct, 0),
			COALESCE(bat.end_soc * cap.kwh_per_pct, 0),
			(SELECT unit_of_length FROM settings LIMIT 1),
			(SELECT unit_of_temperature FROM settings LIMIT 1)
		FROM (VALUES (1)) anchor(_)
		LEFT JOIN cap ON true
		LEFT JOIN d ON true
		LEFT JOIN ch ON true
		LEFT JOIN pk ON true
		LEFT JOIN bat ON true;`

	err := h.db.QueryRowContext(c.Request.Context(), query, CarID).Scan(
		&in.carName,
		&in.totalDistance,
		&in.wallEnergyKWh,
		&in.vehicleAddedKWh,
		&in.totalChargingCost,
		&in.drivingEnergyKWh,
		&in.parkingEnergyKWh,
		&in.startBatteryKWh,
		&in.endBatteryKWh,
		&in.unitsLength,
		&in.unitsTemperature,
	)
	if err != nil {
		respond.HandleErrorV2(c, handler, http.StatusInternalServerError, ErrMsg, err.Error())
		return
	}

	if in.unitsLength == "mi" {
		in.totalDistance = convert.KilometersToMiles(in.totalDistance)
	}

	respond.HandleSuccess(c, handler, dto.V2EnergyFlowResponse{
		Data: buildEnergyFlowData(in),
	})
}

func buildEnergyFlowData(in energyFlowInputs) dto.V2EnergyFlowData {
	wallEnergy := nonNegative(in.wallEnergyKWh)
	vehicleEnergy := nonNegative(in.vehicleAddedKWh)
	chargingCost := nonNegative(in.totalChargingCost)
	drivingEnergy := nonNegative(in.drivingEnergyKWh)
	parkingEnergy := nonNegative(in.parkingEnergyKWh)
	startBatteryEnergy := nonNegative(in.startBatteryKWh)
	endBatteryEnergy := nonNegative(in.endBatteryKWh)
	chargingLossEnergy := nonNegative(wallEnergy - vehicleEnergy)
	vehicleSupplyEnergy := vehicleEnergy
	vehicleAccountedEnergy := drivingEnergy + parkingEnergy + endBatteryEnergy
	unattributedEnergy := nonNegative(vehicleSupplyEnergy - vehicleAccountedEnergy)
	unmatchedUsageEnergy := nonNegative(vehicleAccountedEnergy - vehicleSupplyEnergy)
	wallCostPerKWh := ratio(chargingCost, wallEnergy)
	chargingCostPerKWh := ratio(chargingCost, vehicleEnergy)
	vehicleAddedCost := vehicleEnergy * wallCostPerKWh
	vehicleAccountingCostPerKWh := wallCostPerKWh
	drivingCost := drivingEnergy * vehicleAccountingCostPerKWh
	parkingCost := parkingEnergy * vehicleAccountingCostPerKWh
	lossCost := chargingLossEnergy * wallCostPerKWh
	endBatteryCost := endBatteryEnergy * vehicleAccountingCostPerKWh
	unattributedCost := unattributedEnergy * vehicleAccountingCostPerKWh
	vehicleDrivingEnergy := drivingEnergy
	vehicleParkingEnergy := parkingEnergy
	vehicleEndBatteryEnergy := endBatteryEnergy
	unmatchedDrivingEnergy := 0.0
	unmatchedParkingEnergy := 0.0
	unmatchedEndBatteryEnergy := 0.0
	if unmatchedUsageEnergy > 0 && vehicleAccountedEnergy > 0 {
		vehicleDrivingEnergy = drivingEnergy * ratio(vehicleSupplyEnergy, vehicleAccountedEnergy)
		vehicleParkingEnergy = parkingEnergy * ratio(vehicleSupplyEnergy, vehicleAccountedEnergy)
		vehicleEndBatteryEnergy = endBatteryEnergy * ratio(vehicleSupplyEnergy, vehicleAccountedEnergy)
		unmatchedDrivingEnergy = drivingEnergy - vehicleDrivingEnergy
		unmatchedParkingEnergy = parkingEnergy - vehicleParkingEnergy
		unmatchedEndBatteryEnergy = endBatteryEnergy - vehicleEndBatteryEnergy
	}

	nodes := []dto.V2EnergyFlowNode{
		{ID: "wall_input", Label: "Wall input", EnergyKWh: wallEnergy, Cost: chargingCost},
		{ID: "vehicle_added", Label: "Added to vehicle", EnergyKWh: vehicleEnergy, Cost: vehicleAddedCost},
		{ID: "charging_loss", Label: "Charging loss", EnergyKWh: chargingLossEnergy, Cost: lossCost},
		{ID: "vehicle_available", Label: "Vehicle-side available energy", EnergyKWh: vehicleSupplyEnergy, Cost: vehicleAddedCost},
		{ID: "driving_usage", Label: "Driving use", EnergyKWh: drivingEnergy, Cost: drivingCost},
		{ID: "parking_usage", Label: "Parking use", EnergyKWh: parkingEnergy, Cost: parkingCost},
		{ID: "end_battery_inventory", Label: "Ending battery inventory", EnergyKWh: endBatteryEnergy, Cost: endBatteryCost},
	}
	if unattributedEnergy > 0 {
		nodes = append(nodes, dto.V2EnergyFlowNode{
			ID:        "unattributed_vehicle_energy",
			Label:     "Unmetered vehicle loss",
			EnergyKWh: unattributedEnergy,
			Cost:      unattributedCost,
		})
	}
	if unmatchedUsageEnergy > 0 {
		nodes = append(nodes, dto.V2EnergyFlowNode{
			ID:        "unmatched_vehicle_usage",
			Label:     "Usage gap",
			EnergyKWh: unmatchedUsageEnergy,
			Cost:      unmatchedUsageEnergy * wallCostPerKWh,
		})
	}

	links := []dto.V2EnergyFlowLink{
		makeEnergyFlowLink("wall_input", "vehicle_added", vehicleEnergy, vehicleAddedCost, wallEnergy),
		makeEnergyFlowLink("wall_input", "charging_loss", chargingLossEnergy, lossCost, wallEnergy),
		makeEnergyFlowLink("vehicle_added", "vehicle_available", vehicleEnergy, vehicleAddedCost, vehicleEnergy),
		makeEnergyFlowLink("vehicle_available", "driving_usage", vehicleDrivingEnergy, vehicleDrivingEnergy*vehicleAccountingCostPerKWh, vehicleSupplyEnergy),
		makeEnergyFlowLink("vehicle_available", "parking_usage", vehicleParkingEnergy, vehicleParkingEnergy*vehicleAccountingCostPerKWh, vehicleSupplyEnergy),
		makeEnergyFlowLink("vehicle_available", "end_battery_inventory", vehicleEndBatteryEnergy, vehicleEndBatteryEnergy*vehicleAccountingCostPerKWh, vehicleSupplyEnergy),
	}
	if unattributedEnergy > 0 {
		links = append(links, makeEnergyFlowLink(
			"vehicle_available",
			"unattributed_vehicle_energy",
			unattributedEnergy,
			unattributedCost,
			vehicleSupplyEnergy,
		))
	}
	if unmatchedDrivingEnergy > 0 {
		links = append(links, makeEnergyFlowLink(
			"unmatched_vehicle_usage",
			"driving_usage",
			unmatchedDrivingEnergy,
			unmatchedDrivingEnergy*wallCostPerKWh,
			unmatchedUsageEnergy,
		))
	}
	if unmatchedParkingEnergy > 0 {
		links = append(links, makeEnergyFlowLink(
			"unmatched_vehicle_usage",
			"parking_usage",
			unmatchedParkingEnergy,
			unmatchedParkingEnergy*wallCostPerKWh,
			unmatchedUsageEnergy,
		))
	}
	if unmatchedEndBatteryEnergy > 0 {
		links = append(links, makeEnergyFlowLink(
			"unmatched_vehicle_usage",
			"end_battery_inventory",
			unmatchedEndBatteryEnergy,
			unmatchedEndBatteryEnergy*vehicleAccountingCostPerKWh,
			unmatchedUsageEnergy,
		))
	}

	balanceStatus := "balanced"
	if unmatchedUsageEnergy > 0 {
		balanceStatus = "usage_exceeds_vehicle_added"
	} else if unattributedEnergy > 0 {
		balanceStatus = "has_unattributed_vehicle_energy"
	}

	return dto.V2EnergyFlowData{
		Car: dto.Car{
			CarID:   in.carID,
			CarName: in.carName,
		},
		Nodes: nodes,
		Links: links,
		Metrics: dto.V2EnergyFlowMetrics{
			WallEnergyKWh:                wallEnergy,
			VehicleEnergyAddedKWh:        vehicleEnergy,
			VehicleAvailableEnergyKWh:    vehicleSupplyEnergy,
			StartBatteryEnergyKWh:        startBatteryEnergy,
			EndBatteryEnergyKWh:          endBatteryEnergy,
			BatteryInventoryDeltaKWh:     endBatteryEnergy - startBatteryEnergy,
			ChargingLossEnergyKWh:        chargingLossEnergy,
			DrivingEnergyKWh:             drivingEnergy,
			ParkingEnergyKWh:             parkingEnergy,
			UnattributedVehicleEnergyKWh: unattributedEnergy,
			UnmatchedVehicleUsageKWh:     unmatchedUsageEnergy,
			TotalChargingCost:            chargingCost,
			WallCostPerKWh:               wallCostPerKWh,
			ChargingCostPerKWh:           chargingCostPerKWh,
			VehicleAccountingCostPerKWh:  vehicleAccountingCostPerKWh,
			DrivingCost:                  drivingCost,
			DrivingCostPerDistance:       ratio(drivingCost, nonNegative(in.totalDistance)),
			ParkingCost:                  parkingCost,
			ChargingLossCost:             lossCost,
			EndBatteryCost:               endBatteryCost,
			ActualDrivingUsageRatePct:    pct(drivingEnergy, wallEnergy),
			ActualLossRatePct:            pct(chargingLossEnergy, wallEnergy),
			ChargingEfficiencyPct:        pct(vehicleEnergy, wallEnergy),
			VehicleDrivingSharePct:       pct(drivingEnergy, vehicleSupplyEnergy),
		},
		Units: dto.TeslaMateUnits{
			UnitsLength:      in.unitsLength,
			UnitsTemperature: in.unitsTemperature,
		},
		BalanceStatus: balanceStatus,
	}
}

func makeEnergyFlowLink(source, target string, energy, cost, sourceTotal float64) dto.V2EnergyFlowLink {
	return dto.V2EnergyFlowLink{
		Source:          source,
		Target:          target,
		EnergyKWh:       nonNegative(energy),
		Cost:            nonNegative(cost),
		PercentOfSource: pct(energy, sourceTotal),
	}
}

func pct(numerator, denominator float64) float64 {
	return ratio(numerator, denominator) * 100
}

func ratio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

func nonNegative(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}
