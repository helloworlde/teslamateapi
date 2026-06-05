package v1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/convert"
	"github.com/tobiasehlert/teslamateapi/internal/respond"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

// Status returns the live MQTT-derived status of the car.
//
// @Summary      Live car status
// @Description  Returns MQTT-derived status (location, battery, charging, climate, TPMS).
// @Tags         v1
// @Security     BearerAuth
// @Produce      json
// @Param        CarID  path      int  true  "TeslaMate cars.id"
// @Success      200    {object}  dto.V1StatusResponse
// @Failure      401    {object}  dto.ErrorEnvelope
// @Failure      500    {object}  dto.ErrorEnvelope  "MQTT disconnected"
// @Failure      501    {object}  dto.ErrorEnvelope  "MQTT disabled"
// @Router       /api/v1/cars/{CarID}/status [get]
func (h *Handler) Status(c *gin.Context) {
	if h.statusCache == nil || h.statusCache.Disabled() {
		log.Println("[notice] TeslaMateAPICarsStatusV1 DISABLE_MQTT is set to true.. can not return status for car without mqtt!")
		respond.HandleOther(c, http.StatusNotImplemented, "TeslaMateAPICarsStatusV1", gin.H{"error": "mqtt disabled.. status not accessible!"})
		return
	}

	if !h.statusCache.Connected() {
		log.Println("[notice] TeslaMateAPICarsStatusV1 mqtt is disconnected.. can not return status for car without mqtt!")
		respond.HandleOther(c, http.StatusInternalServerError, "TeslaMateAPICarsStatusV1", gin.H{"error": "mqtt disconnected.. status not accessible!"})
		return
	}

	carID, ok := requirePositiveIntParam(c, "TeslaMateAPICarsStatusV1", "CarID", c.Param("CarID"))
	if !ok {
		return
	}
	stat, ok := h.statusCache.Snapshot(carID)
	if !ok {
		respond.HandleError(c, "TeslaMateAPICarsStatusV1", "no info on this car ID", "-")
		return
	}

	var (
		CarData                                      dto.Car
		MQTTInformationData                          dto.V1StatusInformation
		UnitsLength, UnitsPressure, UnitsTemperature string
	)

	query := `
		SELECT
			id,
			name,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_pressure FROM settings LIMIT 1) as unit_of_pressure,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature
		FROM cars
		WHERE id=$1
		LIMIT 1;`
	err := h.db.QueryRowContext(c.Request.Context(), query, carID).Scan(&CarData.CarID,
		&CarData.CarName,
		&UnitsLength,
		&UnitsPressure,
		&UnitsTemperature)
	if err != nil {
		respond.HandleError(c, "TeslaMateAPICarsStatusV1", "Unable to load cars.", err.Error())
		return
	}

	MQTTInformationData.DisplayName = stat.MQTTDataDisplayName
	MQTTInformationData.State = stat.MQTTDataState
	MQTTInformationData.StateSince = stat.MQTTDataStateSince
	MQTTInformationData.CarStatus.Healthy = stat.MQTTDataHealthy
	MQTTInformationData.CarVersions.Version = stat.MQTTDataVersion
	MQTTInformationData.CarVersions.UpdateAvailable = stat.MQTTDataUpdateAvailable
	MQTTInformationData.CarVersions.UpdateVersion = stat.MQTTDataUpdateVersion
	MQTTInformationData.CarDetails.Model = stat.MQTTDataModel
	MQTTInformationData.CarDetails.TrimBadging = stat.MQTTDataTrimBadging
	MQTTInformationData.CarExterior.ExteriorColor = stat.MQTTDataExteriorColor
	MQTTInformationData.CarExterior.WheelType = stat.MQTTDataWheelType
	MQTTInformationData.CarExterior.SpoilerType = stat.MQTTDataSpoilerType
	MQTTInformationData.CarGeodata.Geofence = stat.MQTTDataGeofence
	MQTTInformationData.CarGeodata.Location = dto.V1StatusCarLocation(stat.MQTTDataLocation)
	MQTTInformationData.DrivingDetails.ActiveRoute.Destination = stat.MQTTDataActiveRoute.Destination
	MQTTInformationData.DrivingDetails.ActiveRoute.EnergyAtArrival = stat.MQTTDataActiveRoute.EnergyAtArrival
	MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival = stat.MQTTDataActiveRoute.DistanceToArrival
	MQTTInformationData.DrivingDetails.ActiveRoute.MinutesToArrival = stat.MQTTDataActiveRoute.MinutesToArrival
	MQTTInformationData.DrivingDetails.ActiveRoute.TrafficMinutesDelay = stat.MQTTDataActiveRoute.TrafficMinutesDelay
	MQTTInformationData.DrivingDetails.ActiveRoute.Location = dto.V1StatusCarLocation(stat.MQTTDataActiveRoute.Location)
	MQTTInformationData.DrivingDetails.ShiftState = stat.MQTTDataShiftState
	MQTTInformationData.DrivingDetails.Power = stat.MQTTDataPower
	MQTTInformationData.DrivingDetails.Speed = stat.MQTTDataSpeed
	MQTTInformationData.DrivingDetails.Heading = stat.MQTTDataHeading
	MQTTInformationData.DrivingDetails.Elevation = stat.MQTTDataElevation
	MQTTInformationData.CarStatus.Locked = stat.MQTTDataLocked
	MQTTInformationData.CarStatus.SentryMode = stat.MQTTDataSentryMode
	MQTTInformationData.CarStatus.WindowsOpen = stat.MQTTDataWindowsOpen
	MQTTInformationData.CarStatus.DoorsOpen = stat.MQTTDataDoorsOpen
	MQTTInformationData.CarStatus.DriverFrontDoorOpen = stat.MQTTDataDriverFrontDoorOpen
	MQTTInformationData.CarStatus.DriverRearDoorOpen = stat.MQTTDataDriverRearDoorOpen
	MQTTInformationData.CarStatus.PassengerFrontDoorOpen = stat.MQTTDataPassengerFrontDoorOpen
	MQTTInformationData.CarStatus.PassengerRearDoorOpen = stat.MQTTDataPassengerRearDoorOpen
	MQTTInformationData.CarStatus.TrunkOpen = stat.MQTTDataTrunkOpen
	MQTTInformationData.CarStatus.FrunkOpen = stat.MQTTDataFrunkOpen
	MQTTInformationData.CarStatus.IsUserPresent = stat.MQTTDataIsUserPresent
	MQTTInformationData.CarStatus.CenterDisplayState = stat.MQTTDataCenterDisplayState
	MQTTInformationData.ClimateDetails.IsClimateOn = stat.MQTTDataIsClimateOn
	MQTTInformationData.ClimateDetails.InsideTemp = stat.MQTTDataInsideTemp
	MQTTInformationData.ClimateDetails.OutsideTemp = stat.MQTTDataOutsideTemp
	MQTTInformationData.ClimateDetails.IsPreconditioning = stat.MQTTDataIsPreconditioning
	MQTTInformationData.ClimateDetails.ClimateKeeperMode = stat.MQTTDataClimateKeeperMode
	MQTTInformationData.Odometer = stat.MQTTDataOdometer
	MQTTInformationData.BatteryDetails.EstBatteryRange = stat.MQTTDataEstBatteryRange
	MQTTInformationData.BatteryDetails.RatedBatteryRange = stat.MQTTDataRatedBatteryRange
	MQTTInformationData.BatteryDetails.IdealBatteryRange = stat.MQTTDataIdealBatteryRange
	MQTTInformationData.BatteryDetails.BatteryLevel = stat.MQTTDataBatteryLevel
	MQTTInformationData.BatteryDetails.UsableBatteryLevel = stat.MQTTDataUsableBatteryLevel
	MQTTInformationData.ChargingDetails.PluggedIn = stat.MQTTDataPluggedIn
	MQTTInformationData.ChargingDetails.ChargingState = stat.MQTTDataChargingState
	MQTTInformationData.ChargingDetails.ChargeEnergyAdded = stat.MQTTDataChargeEnergyAdded
	MQTTInformationData.ChargingDetails.ChargeLimitSoc = stat.MQTTDataChargeLimitSoc
	MQTTInformationData.ChargingDetails.ChargePortDoorOpen = stat.MQTTDataChargePortDoorOpen
	MQTTInformationData.ChargingDetails.ChargerActualCurrent = stat.MQTTDataChargerActualCurrent
	MQTTInformationData.ChargingDetails.ChargerPhases = stat.MQTTDataChargerPhases
	MQTTInformationData.ChargingDetails.ChargerPower = stat.MQTTDataChargerPower
	MQTTInformationData.ChargingDetails.ChargerVoltage = stat.MQTTDataChargerVoltage
	MQTTInformationData.ChargingDetails.ChargeCurrentRequest = stat.MQTTDataChargeCurrentRequest
	MQTTInformationData.ChargingDetails.ChargeCurrentRequestMax = stat.MQTTDataChargeCurrentRequestMax
	MQTTInformationData.ChargingDetails.ScheduledChargingStartTime = stat.MQTTDataScheduledChargingStartTime
	MQTTInformationData.ChargingDetails.TimeToFullCharge = stat.MQTTDataTimeToFullCharge
	MQTTInformationData.TpmsDetails.TpmsPressureFL = stat.MQTTDataTpmsPressureFL
	MQTTInformationData.TpmsDetails.TpmsPressureFR = stat.MQTTDataTpmsPressureFR
	MQTTInformationData.TpmsDetails.TpmsPressureRL = stat.MQTTDataTpmsPressureRL
	MQTTInformationData.TpmsDetails.TpmsPressureRR = stat.MQTTDataTpmsPressureRR
	MQTTInformationData.TpmsDetails.TpmsSoftWarningFL = stat.MQTTDataTpmsSoftWarningFL
	MQTTInformationData.TpmsDetails.TpmsSoftWarningFR = stat.MQTTDataTpmsSoftWarningFR
	MQTTInformationData.TpmsDetails.TpmsSoftWarningRL = stat.MQTTDataTpmsSoftWarningRL
	MQTTInformationData.TpmsDetails.TpmsSoftWarningRR = stat.MQTTDataTpmsSoftWarningRR

	// DEPRECATED - setting values for deprecated fields
	MQTTInformationData.CarGeodata.Latitude = stat.MQTTDataLocation.Latitude
	MQTTInformationData.CarGeodata.Longitude = stat.MQTTDataLocation.Longitude
	MQTTInformationData.DrivingDetails.ActiveRouteDestination = stat.MQTTDataActiveRoute.Destination
	MQTTInformationData.DrivingDetails.ActiveRouteLatitude = stat.MQTTDataActiveRoute.Location.Latitude
	MQTTInformationData.DrivingDetails.ActiveRouteLongitude = stat.MQTTDataActiveRoute.Location.Longitude

	if UnitsLength == "mi" {
		MQTTInformationData.Odometer = convert.KilometersToMiles(MQTTInformationData.Odometer)
		MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival = convert.KilometersToMiles(MQTTInformationData.DrivingDetails.ActiveRoute.DistanceToArrival)
		MQTTInformationData.DrivingDetails.Speed = convert.KilometersToMilesInteger(MQTTInformationData.DrivingDetails.Speed)
		MQTTInformationData.BatteryDetails.EstBatteryRange = convert.KilometersToMiles(MQTTInformationData.BatteryDetails.EstBatteryRange)
		MQTTInformationData.BatteryDetails.RatedBatteryRange = convert.KilometersToMiles(MQTTInformationData.BatteryDetails.RatedBatteryRange)
		MQTTInformationData.BatteryDetails.IdealBatteryRange = convert.KilometersToMiles(MQTTInformationData.BatteryDetails.IdealBatteryRange)
	}
	if UnitsPressure == "psi" {
		MQTTInformationData.TpmsDetails.TpmsPressureFL = convert.BarToPsi(MQTTInformationData.TpmsDetails.TpmsPressureFL)
		MQTTInformationData.TpmsDetails.TpmsPressureFR = convert.BarToPsi(MQTTInformationData.TpmsDetails.TpmsPressureFR)
		MQTTInformationData.TpmsDetails.TpmsPressureRL = convert.BarToPsi(MQTTInformationData.TpmsDetails.TpmsPressureRL)
		MQTTInformationData.TpmsDetails.TpmsPressureRR = convert.BarToPsi(MQTTInformationData.TpmsDetails.TpmsPressureRR)
	}
	if UnitsTemperature == "F" {
		MQTTInformationData.ClimateDetails.InsideTemp = convert.CelsiusToFahrenheit(MQTTInformationData.ClimateDetails.InsideTemp)
		MQTTInformationData.ClimateDetails.OutsideTemp = convert.CelsiusToFahrenheit(MQTTInformationData.ClimateDetails.OutsideTemp)
	}

	MQTTInformationData.StateSince = h.timeInTZ(MQTTInformationData.StateSince)
	MQTTInformationData.ChargingDetails.ScheduledChargingStartTime = h.timeInTZ(MQTTInformationData.ChargingDetails.ScheduledChargingStartTime)

	jsonData := dto.V1StatusResponse{
		Data: dto.V1StatusData{
			Car:    CarData,
			Status: MQTTInformationData,
			TeslaMateUnits: dto.TeslaMateUnitsWithPressure{
				UnitsLength:      UnitsLength,
				UnitsPressure:    UnitsPressure,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	respond.HandleSuccess(c, "TeslaMateAPICarsStatusV1", jsonData)
}
