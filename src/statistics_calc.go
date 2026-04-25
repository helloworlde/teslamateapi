package main

func calcDurationSeconds(startUnix, endUnix int64) *float64 {
	if endUnix <= startUnix {
		return nil
	}
	v := float64(endUnix - startUnix)
	return &v
}

func calcAvgSpeed(distanceKm, durationSec *float64) *float64 {
	if distanceKm == nil || durationSec == nil || *durationSec <= 0 {
		return nil
	}
	v := (*distanceKm) / (*durationSec / 3600.0)
	return &v
}

func calcEfficiencyWhPerKm(usedEnergyKWh, distanceKm *float64) *float64 {
	if usedEnergyKWh == nil || distanceKm == nil || *distanceKm <= 0 {
		return nil
	}
	v := (*usedEnergyKWh) * 1000.0 / (*distanceKm)
	return &v
}

func calcChargeEfficiencyPercent(energyAddedKWh, chargerEnergyKWh *float64) *float64 {
	if energyAddedKWh == nil || chargerEnergyKWh == nil || *chargerEnergyKWh <= 0 {
		return nil
	}
	v := (*energyAddedKWh) / (*chargerEnergyKWh) * 100.0
	return &v
}
