package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type HistorySummaryCoverage struct {
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

type DriveHistorySummary struct {
	Coverage            HistorySummaryCoverage `json:"coverage"`
	DriveCount          int                    `json:"drive_count"`
	TotalDurationMin    int                    `json:"total_duration_min"`
	TotalDistance       float64                `json:"total_distance"`
	LongestDistance     *float64               `json:"longest_distance"`
	LongestDurationMin  *int                   `json:"longest_duration_min"`
	AverageDistance     *float64               `json:"average_distance"`
	AverageDurationMin  *float64               `json:"average_duration_min"`
	AverageSpeed        *float64               `json:"average_speed"`
	TotalEnergyConsumed *float64               `json:"total_energy_consumed"`
	AverageConsumption  *float64               `json:"average_consumption"`
	BestConsumption     *float64               `json:"best_consumption"`
	WorstConsumption    *float64               `json:"worst_consumption"`
	MaxSpeed            *int                   `json:"max_speed"`
	PeakDrivePower      *int                   `json:"peak_drive_power"`
	PeakRegenPower      *int                   `json:"peak_regen_power"`
}

type ChargeHistorySummary struct {
	Coverage           HistorySummaryCoverage `json:"coverage"`
	ChargeCount        int                    `json:"charge_count"`
	TotalDurationMin   int                    `json:"total_duration_min"`
	TotalEnergyAdded   float64                `json:"total_energy_added"`
	TotalEnergyUsed    *float64               `json:"total_energy_used"`
	LongestDurationMin *int                   `json:"longest_duration_min"`
	LargestEnergyAdded *float64               `json:"largest_energy_added"`
	AverageEnergyAdded *float64               `json:"average_energy_added"`
	AverageDurationMin *float64               `json:"average_duration_min"`
	AveragePower       *float64               `json:"average_power"`
	MaxPower           *int                   `json:"max_power"`
	ChargingEfficiency *float64               `json:"charging_efficiency"`
	TotalCost          *float64               `json:"total_cost"`
	AverageCost        *float64               `json:"average_cost"`
	HighestCost        *float64               `json:"highest_cost"`
}

type LifetimeConsumptionSummary struct {
	BatteryConsumptionPer100Distance *float64 `json:"battery_consumption_per_100_distance"`
	WallConsumptionPer100Distance    *float64 `json:"wall_consumption_per_100_distance"`
	ChargingEfficiency               *float64 `json:"charging_efficiency"`
	TotalDistance                    float64  `json:"total_distance"`
	TotalEnergyAdded                 float64  `json:"total_energy_added"`
	TotalEnergyUsed                  float64  `json:"total_energy_used"`
	DriveCount                       int      `json:"drive_count"`
	ChargeCount                      int      `json:"charge_count"`
}

type DashboardSeriesSummary struct {
	EfficiencySeries    []SummaryTimeSeriesPoint `json:"efficiency_series"`
	MonthlyDistance     []SummaryCategoryValue   `json:"monthly_distance"`
	MonthlyChargeEnergy []SummaryCategoryValue   `json:"monthly_charge_energy"`
	ChargeLocations     []SummaryCategoryValue   `json:"charge_locations"`
}

type SummaryTimeSeriesPoint struct {
	ID    string  `json:"id"`
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type SummaryCategoryValue struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Period string  `json:"period,omitempty"`
	Value  float64 `json:"value"`
}

type TeslaMateSummaryCar struct {
	CarID   int        `json:"car_id"`
	CarName NullString `json:"car_name"`
}

type TeslaMateSummaryUnits struct {
	UnitsLength      string `json:"unit_of_length"`
	UnitsTemperature string `json:"unit_of_temperature"`
}

type TeslaMateSummaryFilters struct {
	StartDate *string  `json:"start_date"`
	EndDate   *string  `json:"end_date"`
	Include   []string `json:"include,omitempty"`
}

type TeslaMateSummaryData struct {
	Car                 TeslaMateSummaryCar         `json:"car"`
	Filters             TeslaMateSummaryFilters     `json:"filters"`
	Overview            *OverviewSummary            `json:"overview"`
	LifetimeSummary     *LifetimeConsumptionSummary `json:"lifetime_summary"`
	DriveSummary        *DriveHistorySummary        `json:"drive_summary"`
	ChargeSummary       *ChargeHistorySummary       `json:"charge_summary"`
	ParkingSummary      *ParkingHistorySummary      `json:"parking_summary"`
	AnalysisSummary     *AnalysisSummary            `json:"analysis_summary"`
	StatisticsSummary   *StatisticsSummary          `json:"statistics_summary"`
	StateSummary        *StateSummary               `json:"state_summary"`
	DashboardSeries     *DashboardSeriesSummary     `json:"dashboard_series"`
	EfficiencySeries    []SummaryTimeSeriesPoint    `json:"efficiency_series"`
	MonthlyDistance     []SummaryCategoryValue      `json:"monthly_distance"`
	MonthlyChargeEnergy []SummaryCategoryValue      `json:"monthly_charge_energy"`
	ChargeLocations     []SummaryCategoryValue      `json:"charge_locations"`
	TeslaMateUnits      TeslaMateSummaryUnits       `json:"units"`
}

type TeslaMateSummaryJSONData struct {
	Data TeslaMateSummaryData `json:"data"`
}

// TeslaMateAPICarsSummaryV1 func
func TeslaMateAPICarsSummaryV1(c *gin.Context) {
	var (
		CarsSummaryError1 = "Unable to load summary."
		CarsSummaryError2 = "Invalid date format."
		CarsSummaryError3 = "Invalid summary parameter."
	)

	CarID := convertStringToInteger(c.Param("CarID"))

	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError2, err.Error())
		return
	}

	include := parseSummaryIncludes(c.DefaultQuery("include", "all"))

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError1, err.Error())
		return
	}

	var driveSummary *DriveHistorySummary
	if include["drives"] || include["lifetime"] || include["overview"] || include["analysis"] || include["statistics"] {
		driveSummary, err = fetchDriveHistorySummary(CarID, parsedStartDate, parsedEndDate, unitsLength)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError1, err.Error())
			return
		}
	}

	var chargeSummary *ChargeHistorySummary
	if include["charges"] || include["lifetime"] || include["overview"] || include["analysis"] || include["statistics"] {
		chargeSummary, err = fetchChargeHistorySummary(CarID, parsedStartDate, parsedEndDate)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError1, err.Error())
			return
		}
	}

	var parkingSummary *ParkingHistorySummary
	if include["parking"] || include["overview"] || include["analysis"] {
		parkingSummary, err = fetchParkingHistorySummary(CarID, parsedStartDate, parsedEndDate, nil)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError1, err.Error())
			return
		}
	}

	var lifetimeSummary *LifetimeConsumptionSummary
	if include["lifetime"] {
		lifetimeSummary = makeLifetimeConsumptionSummary(driveSummary, chargeSummary)
	}

	var overviewSummary *OverviewSummary
	if include["overview"] {
		overviewSummary = makeOverviewSummary(driveSummary, chargeSummary, parkingSummary)
	}

	var analysisSummary *AnalysisSummary
	if include["analysis"] {
		analysisSummary, err = fetchAnalysisSummary(CarID, parsedStartDate, parsedEndDate, unitsLength, driveSummary, chargeSummary, parkingSummary)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError1, err.Error())
			return
		}
	}

	var statisticsSummary *StatisticsSummary
	if include["statistics"] {
		statisticsSummary, err = fetchStatisticsSummary(CarID, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature, driveSummary, chargeSummary)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError1, err.Error())
			return
		}
	}

	var stateSummary *StateSummary
	if include["states"] {
		stateSummary, err = fetchStateSummary(CarID, parsedStartDate, parsedEndDate)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError1, err.Error())
			return
		}
	}

	var dashboardSeries *DashboardSeriesSummary
	if include["series"] {
		seriesLimit, err := parseSummaryPositiveIntParam(c.Query("seriesLimit"), 12, 1, 100)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError3, err.Error())
			return
		}
		seriesMonths, err := parseSummaryPositiveIntParam(c.Query("seriesMonths"), 6, 1, 24)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError3, err.Error())
			return
		}
		locationLimit, err := parseSummaryPositiveIntParam(c.Query("locationLimit"), 4, 1, 20)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError3, err.Error())
			return
		}

		dashboardSeries, err = fetchDashboardSeriesSummary(CarID, parsedStartDate, parsedEndDate, unitsLength, seriesLimit, seriesMonths, locationLimit)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsSummaryV1", CarsSummaryError1, err.Error())
			return
		}
	}

	jsonData := TeslaMateSummaryJSONData{
		Data: TeslaMateSummaryData{
			Car: TeslaMateSummaryCar{
				CarID:   CarID,
				CarName: carName,
			},
			Filters: TeslaMateSummaryFilters{
				StartDate: summaryFilterDate(parsedStartDate),
				EndDate:   summaryFilterDate(parsedEndDate),
				Include:   summaryIncludeList(include),
			},
			Overview:          includedSummary(include, "overview", overviewSummary),
			LifetimeSummary:   lifetimeSummary,
			DriveSummary:      includedSummary(include, "drives", driveSummary),
			ChargeSummary:     includedSummary(include, "charges", chargeSummary),
			ParkingSummary:    includedSummary(include, "parking", parkingSummary),
			AnalysisSummary:   includedSummary(include, "analysis", analysisSummary),
			StatisticsSummary: includedSummary(include, "statistics", statisticsSummary),
			StateSummary:      includedSummary(include, "states", stateSummary),
			DashboardSeries:   dashboardSeries,
			TeslaMateUnits: TeslaMateSummaryUnits{
				UnitsLength:      unitsLength,
				UnitsTemperature: unitsTemperature,
			},
		},
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsSummaryV1", jsonData)
}

func TeslaMateAPICarsLifetimeSummaryV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsLifetimeSummaryV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load lifetime summary.", err.Error())
		return
	}

	driveSummary, err := fetchDriveHistorySummary(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load lifetime summary.", err.Error())
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load lifetime summary.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.LifetimeSummary = makeLifetimeConsumptionSummary(driveSummary, chargeSummary)
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"lifetime_summary": data.LifetimeSummary,
	}))
}

func TeslaMateAPICarsDriveSummaryV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsDriveSummaryV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load drive summary.", err.Error())
		return
	}
	driveSummary, err := fetchDriveHistorySummary(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load drive summary.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.DriveSummary = driveSummary
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"drive_summary": data.DriveSummary,
	}))
}

func TeslaMateAPICarsChargeSummaryV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsChargeSummaryV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load charge summary.", err.Error())
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load charge summary.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.ChargeSummary = chargeSummary
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"charge_summary": data.ChargeSummary,
	}))
}

func TeslaMateAPICarsDashboardEfficiencySeriesV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsDashboardEfficiencySeriesV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}
	limit, err := parseSummaryPositiveIntParam(c.Query("limit"), 12, 1, 100)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid dashboard parameter.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load dashboard efficiency series.", err.Error())
		return
	}
	series, err := fetchDashboardEfficiencySeries(CarID, parsedStartDate, parsedEndDate, unitsLength, limit)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load dashboard efficiency series.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.EfficiencySeries = series
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"efficiency_series": data.EfficiencySeries,
	}))
}

func TeslaMateAPICarsDashboardMonthlyDistanceV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsDashboardMonthlyDistanceV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}
	months, err := parseSummaryPositiveIntParam(c.Query("months"), 6, 1, 24)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid dashboard parameter.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load dashboard monthly distance.", err.Error())
		return
	}
	items, err := fetchDashboardMonthlyDistance(CarID, parsedStartDate, parsedEndDate, unitsLength, months)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load dashboard monthly distance.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.MonthlyDistance = items
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"monthly_distance": data.MonthlyDistance,
	}))
}

func TeslaMateAPICarsDashboardMonthlyChargeEnergyV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsDashboardMonthlyChargeEnergyV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}
	months, err := parseSummaryPositiveIntParam(c.Query("months"), 6, 1, 24)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid dashboard parameter.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load dashboard monthly charge energy.", err.Error())
		return
	}
	items, err := fetchDashboardMonthlyChargeEnergy(CarID, parsedStartDate, parsedEndDate, months)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load dashboard monthly charge energy.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.MonthlyChargeEnergy = items
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"monthly_charge_energy": data.MonthlyChargeEnergy,
	}))
}

func TeslaMateAPICarsDashboardChargeLocationsV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsDashboardChargeLocationsV1"

	CarID := convertStringToInteger(c.Param("CarID"))
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid date format.", err.Error())
		return
	}
	limit, err := parseSummaryPositiveIntParam(c.Query("limit"), 4, 1, 20)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Invalid dashboard parameter.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load dashboard charge locations.", err.Error())
		return
	}
	items, err := fetchDashboardChargeLocations(CarID, parsedStartDate, parsedEndDate, limit)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, actionName, "Unable to load dashboard charge locations.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.ChargeLocations = items
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"charge_locations": data.ChargeLocations,
	}))
}

func parseSummaryIncludes(raw string) map[string]bool {
	include := map[string]bool{
		"overview":   false,
		"lifetime":   false,
		"drives":     false,
		"charges":    false,
		"parking":    false,
		"analysis":   false,
		"statistics": false,
		"states":     false,
		"series":     false,
	}

	for _, part := range strings.Split(raw, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "", "all":
			include["overview"] = true
			include["lifetime"] = true
			include["drives"] = true
			include["charges"] = true
			include["parking"] = true
			include["analysis"] = true
			include["statistics"] = true
			include["states"] = true
		case "overview", "summary":
			include["overview"] = true
		case "lifetime", "lifecycle":
			include["lifetime"] = true
		case "drive", "drives", "trips":
			include["drives"] = true
		case "charge", "charges", "charging":
			include["charges"] = true
		case "parking", "parked":
			include["parking"] = true
		case "analysis", "analytics", "insights":
			include["analysis"] = true
		case "statistics", "stats":
			include["statistics"] = true
		case "states", "state":
			include["states"] = true
		case "series", "dashboard", "charts":
			include["series"] = true
		}
	}

	if !include["overview"] && !include["lifetime"] && !include["drives"] && !include["charges"] && !include["parking"] && !include["analysis"] && !include["statistics"] && !include["states"] && !include["series"] {
		include["overview"] = true
		include["lifetime"] = true
		include["drives"] = true
		include["charges"] = true
		include["parking"] = true
		include["analysis"] = true
		include["statistics"] = true
		include["states"] = true
	}

	return include
}

func summaryIncludeList(include map[string]bool) []string {
	result := make([]string, 0, 9)
	for _, item := range []string{"overview", "lifetime", "drives", "charges", "parking", "analysis", "statistics", "states", "series"} {
		if include[item] {
			result = append(result, item)
		}
	}
	return result
}

func includedSummary[T any](include map[string]bool, key string, summary *T) *T {
	if include[key] {
		return summary
	}
	return nil
}

func parseSummaryDateRange(c *gin.Context) (string, string, error) {
	parsedStartDate, err := parseDateParam(c.Query("startDate"))
	if err != nil {
		return "", "", err
	}
	parsedEndDate, err := parseDateParam(c.Query("endDate"))
	if err != nil {
		return "", "", err
	}
	return parsedStartDate, parsedEndDate, nil
}

func makeSummaryResponseData(
	CarID int,
	carName NullString,
	parsedStartDate string,
	parsedEndDate string,
	unitsLength string,
	unitsTemperature string,
) TeslaMateSummaryData {
	return TeslaMateSummaryData{
		Car: TeslaMateSummaryCar{
			CarID:   CarID,
			CarName: carName,
		},
		Filters: TeslaMateSummaryFilters{
			StartDate: summaryFilterDate(parsedStartDate),
			EndDate:   summaryFilterDate(parsedEndDate),
		},
		TeslaMateUnits: TeslaMateSummaryUnits{
			UnitsLength:      unitsLength,
			UnitsTemperature: unitsTemperature,
		},
	}
}

func focusedSummaryResponse(base TeslaMateSummaryData, fields gin.H) gin.H {
	data := gin.H{
		"car":     base.Car,
		"filters": base.Filters,
		"units":   base.TeslaMateUnits,
	}
	for key, value := range fields {
		data[key] = value
	}
	return gin.H{"data": data}
}

func parseSummaryPositiveIntParam(raw string, defaultValue int, minValue int, maxValue int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", raw)
	}
	if value < minValue {
		return minValue, nil
	}
	if value > maxValue {
		return maxValue, nil
	}
	return value, nil
}

func summaryFilterDate(raw string) *string {
	if raw == "" {
		return nil
	}
	value := getTimeInTimeZone(raw)
	return &value
}

func fetchSummaryMetadata(CarID int) (string, string, NullString, error) {
	var unitsLength, unitsTemperature string
	var carName NullString

	err := db.QueryRow(`
		SELECT
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name
		FROM cars
		WHERE cars.id = $1;`, CarID).Scan(&unitsLength, &unitsTemperature, &carName)
	if err != nil {
		return "", "", "", err
	}

	return unitsLength, unitsTemperature, carName, nil
}

func fetchDriveHistorySummary(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) (*DriveHistorySummary, error) {
	query := `
		WITH filtered_drives AS (
			SELECT
				drives.start_date,
				drives.end_date,
				GREATEST(COALESCE(drives.duration_min, 0), 0) AS duration_min,
				GREATEST(COALESCE(drives.distance, 0), 0) AS distance,
				drives.speed_max,
				drives.power_max,
				drives.power_min,
				CASE
					WHEN (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency
					ELSE NULL
				END AS energy_consumed_net,
				CASE
					WHEN (
						drives.duration_min > 1
						AND drives.distance > 1
						AND (
							start_position.usable_battery_level IS NULL
							OR end_position.usable_battery_level IS NULL
							OR (end_position.battery_level - end_position.usable_battery_level) = 0
						)
					)
					AND NULLIF(drives.distance, 0) IS NOT NULL
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / NULLIF(drives.distance, 0) * 1000
					ELSE NULL
				END AS consumption_net
			FROM drives
			LEFT JOIN cars ON drives.car_id = cars.id
			LEFT JOIN positions start_position ON drives.start_position_id = start_position.id
			LEFT JOIN positions end_position ON drives.end_position_id = end_position.id
			WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, paramIndex = appendSummaryDateFilters(query, queryParams, paramIndex, "drives", parsedStartDate, parsedEndDate)
	query += `
		),
		aggregated_drives AS (
			SELECT
				COUNT(*) AS drive_count,
				COALESCE(SUM(duration_min), 0) AS total_duration_min,
				COALESCE(SUM(distance), 0) AS total_distance,
				MAX(NULLIF(distance, 0)) AS longest_distance,
				MAX(NULLIF(duration_min, 0)) AS longest_duration_min,
				CASE WHEN COUNT(*) > 0 AND SUM(distance) > 0 THEN SUM(distance) / COUNT(*) ELSE NULL END AS average_distance,
				CASE WHEN COUNT(*) > 0 AND SUM(duration_min) > 0 THEN SUM(duration_min)::float8 / COUNT(*) ELSE NULL END AS average_duration_min,
				CASE WHEN SUM(duration_min) > 0 AND SUM(distance) > 0 THEN SUM(distance) / (SUM(duration_min)::float8 / 60.0) ELSE NULL END AS average_speed,
				NULLIF(SUM(GREATEST(COALESCE(energy_consumed_net, 0), 0)), 0) AS total_energy_consumed,
				CASE
					WHEN SUM(distance) > 0 AND SUM(GREATEST(COALESCE(energy_consumed_net, 0), 0)) > 0
					THEN SUM(GREATEST(COALESCE(energy_consumed_net, 0), 0)) / SUM(distance) * 1000.0
					ELSE NULL
				END AS average_consumption,
				MIN(CASE WHEN consumption_net > 0 THEN consumption_net ELSE NULL END) AS best_consumption,
				MAX(CASE WHEN consumption_net > 0 THEN consumption_net ELSE NULL END) AS worst_consumption,
				MAX(speed_max) AS max_speed,
				MAX(power_max) AS peak_drive_power,
				MAX(CASE WHEN power_min < 0 THEN ABS(power_min) ELSE NULL END) AS peak_regen_power,
				MIN(start_date) AS coverage_start,
				MAX(end_date) AS coverage_end
			FROM filtered_drives
		)
		SELECT
			drive_count,
			total_duration_min,
			total_distance,
			longest_distance,
			longest_duration_min,
			average_distance,
			average_duration_min,
			average_speed,
			total_energy_consumed,
			average_consumption,
			best_consumption,
			worst_consumption,
			max_speed,
			peak_drive_power,
			peak_regen_power,
			coverage_start,
			coverage_end
		FROM aggregated_drives;`

	var (
		driveCount          int
		totalDurationMin    int
		totalDistance       float64
		longestDistance     sql.NullFloat64
		longestDurationMin  sql.NullInt64
		averageDistance     sql.NullFloat64
		averageDurationMin  sql.NullFloat64
		averageSpeed        sql.NullFloat64
		totalEnergyConsumed sql.NullFloat64
		averageConsumption  sql.NullFloat64
		bestConsumption     sql.NullFloat64
		worstConsumption    sql.NullFloat64
		maxSpeed            sql.NullInt64
		peakDrivePower      sql.NullInt64
		peakRegenPower      sql.NullInt64
		coverageStart       sql.NullString
		coverageEnd         sql.NullString
	)

	err := db.QueryRow(query, queryParams...).Scan(
		&driveCount,
		&totalDurationMin,
		&totalDistance,
		&longestDistance,
		&longestDurationMin,
		&averageDistance,
		&averageDurationMin,
		&averageSpeed,
		&totalEnergyConsumed,
		&averageConsumption,
		&bestConsumption,
		&worstConsumption,
		&maxSpeed,
		&peakDrivePower,
		&peakRegenPower,
		&coverageStart,
		&coverageEnd,
	)
	if err != nil {
		return nil, err
	}
	if driveCount == 0 {
		return nil, nil
	}

	if unitsLength == "mi" {
		totalDistance = kilometersToMiles(totalDistance)
		longestDistance = kilometersToMilesSqlNullFloat64(longestDistance)
		averageDistance = kilometersToMilesSqlNullFloat64(averageDistance)
		averageSpeed = kilometersToMilesSqlNullFloat64(averageSpeed)
		bestConsumption = kilometersToMilesSqlNullFloat64(bestConsumption)
		worstConsumption = kilometersToMilesSqlNullFloat64(worstConsumption)
		maxSpeed = kilometersToMilesSqlNullInt64(maxSpeed)
	}

	return &DriveHistorySummary{
		Coverage: HistorySummaryCoverage{
			StartDate: timeZoneStringPointer(coverageStart),
			EndDate:   timeZoneStringPointer(coverageEnd),
		},
		DriveCount:          driveCount,
		TotalDurationMin:    totalDurationMin,
		TotalDistance:       totalDistance,
		LongestDistance:     floatPointer(longestDistance),
		LongestDurationMin:  intPointer(longestDurationMin),
		AverageDistance:     floatPointer(averageDistance),
		AverageDurationMin:  floatPointer(averageDurationMin),
		AverageSpeed:        floatPointer(averageSpeed),
		TotalEnergyConsumed: floatPointer(totalEnergyConsumed),
		AverageConsumption:  floatPointer(averageConsumption),
		BestConsumption:     floatPointer(bestConsumption),
		WorstConsumption:    floatPointer(worstConsumption),
		MaxSpeed:            intPointer(maxSpeed),
		PeakDrivePower:      intPointer(peakDrivePower),
		PeakRegenPower:      intPointer(peakRegenPower),
	}, nil
}

func fetchChargeHistorySummary(CarID int, parsedStartDate string, parsedEndDate string) (*ChargeHistorySummary, error) {
	query := `
		WITH filtered_charges AS (
			SELECT
				charging_processes.id,
				charging_processes.start_date,
				charging_processes.end_date,
				GREATEST(COALESCE(charging_processes.duration_min, 0), 0) AS duration_min,
				GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0) AS charge_energy_added,
				GREATEST(
					COALESCE(charging_processes.charge_energy_used, 0),
					COALESCE(charging_processes.charge_energy_added, 0),
					0
				) AS charge_energy_used,
				COALESCE(charging_processes.cost, 0) AS cost
			FROM charging_processes
			WHERE charging_processes.car_id=$1 AND charging_processes.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, paramIndex = appendSummaryDateFilters(query, queryParams, paramIndex, "charging_processes", parsedStartDate, parsedEndDate)
	query += `
		),
		aggregated_charges AS (
			SELECT
				COUNT(*) AS charge_count,
				COALESCE(SUM(duration_min), 0) AS total_duration_min,
				COALESCE(SUM(charge_energy_added), 0) AS total_energy_added,
				NULLIF(SUM(charge_energy_used), 0) AS total_energy_used,
				MAX(NULLIF(duration_min, 0)) AS longest_duration_min,
				MAX(NULLIF(charge_energy_added, 0)) AS largest_energy_added,
				CASE WHEN COUNT(*) > 0 AND SUM(charge_energy_added) > 0 THEN SUM(charge_energy_added) / COUNT(*) ELSE NULL END AS average_energy_added,
				CASE WHEN COUNT(*) > 0 AND SUM(duration_min) > 0 THEN SUM(duration_min)::float8 / COUNT(*) ELSE NULL END AS average_duration_min,
				CASE
					WHEN SUM(duration_min) > 0 AND GREATEST(SUM(charge_energy_used), SUM(charge_energy_added)) > 0
					THEN GREATEST(SUM(charge_energy_used), SUM(charge_energy_added)) / (SUM(duration_min)::float8 / 60.0)
					ELSE NULL
				END AS average_power,
				CASE
					WHEN SUM(charge_energy_used) > 0 AND SUM(charge_energy_added) > 0
					THEN SUM(charge_energy_added) / SUM(charge_energy_used)
					ELSE NULL
				END AS charging_efficiency,
				NULLIF(SUM(CASE WHEN cost > 0 THEN cost ELSE 0 END), 0) AS total_cost,
				CASE
					WHEN COUNT(CASE WHEN cost > 0 THEN 1 ELSE NULL END) > 0
					THEN SUM(CASE WHEN cost > 0 THEN cost ELSE 0 END) / COUNT(CASE WHEN cost > 0 THEN 1 ELSE NULL END)
					ELSE NULL
				END AS average_cost,
				MAX(CASE WHEN cost > 0 THEN cost ELSE NULL END) AS highest_cost,
				MIN(start_date) AS coverage_start,
				MAX(end_date) AS coverage_end
			FROM filtered_charges
		),
		peak_power AS (
			SELECT MAX(charges.charger_power) AS max_power
			FROM filtered_charges
			LEFT JOIN charges ON charges.charging_process_id = filtered_charges.id
		)
		SELECT
			charge_count,
			total_duration_min,
			total_energy_added,
			total_energy_used,
			longest_duration_min,
			largest_energy_added,
			average_energy_added,
			average_duration_min,
			average_power,
			max_power,
			charging_efficiency,
			total_cost,
			average_cost,
			highest_cost,
			coverage_start,
			coverage_end
		FROM aggregated_charges, peak_power;`

	var (
		chargeCount        int
		totalDurationMin   int
		totalEnergyAdded   float64
		totalEnergyUsed    sql.NullFloat64
		longestDurationMin sql.NullInt64
		largestEnergyAdded sql.NullFloat64
		averageEnergyAdded sql.NullFloat64
		averageDurationMin sql.NullFloat64
		averagePower       sql.NullFloat64
		maxPower           sql.NullInt64
		chargingEfficiency sql.NullFloat64
		totalCost          sql.NullFloat64
		averageCost        sql.NullFloat64
		highestCost        sql.NullFloat64
		coverageStart      sql.NullString
		coverageEnd        sql.NullString
	)

	err := db.QueryRow(query, queryParams...).Scan(
		&chargeCount,
		&totalDurationMin,
		&totalEnergyAdded,
		&totalEnergyUsed,
		&longestDurationMin,
		&largestEnergyAdded,
		&averageEnergyAdded,
		&averageDurationMin,
		&averagePower,
		&maxPower,
		&chargingEfficiency,
		&totalCost,
		&averageCost,
		&highestCost,
		&coverageStart,
		&coverageEnd,
	)
	if err != nil {
		return nil, err
	}
	if chargeCount == 0 {
		return nil, nil
	}

	return &ChargeHistorySummary{
		Coverage: HistorySummaryCoverage{
			StartDate: timeZoneStringPointer(coverageStart),
			EndDate:   timeZoneStringPointer(coverageEnd),
		},
		ChargeCount:        chargeCount,
		TotalDurationMin:   totalDurationMin,
		TotalEnergyAdded:   totalEnergyAdded,
		TotalEnergyUsed:    floatPointer(totalEnergyUsed),
		LongestDurationMin: intPointer(longestDurationMin),
		LargestEnergyAdded: floatPointer(largestEnergyAdded),
		AverageEnergyAdded: floatPointer(averageEnergyAdded),
		AverageDurationMin: floatPointer(averageDurationMin),
		AveragePower:       floatPointer(averagePower),
		MaxPower:           intPointer(maxPower),
		ChargingEfficiency: floatPointer(chargingEfficiency),
		TotalCost:          floatPointer(totalCost),
		AverageCost:        floatPointer(averageCost),
		HighestCost:        floatPointer(highestCost),
	}, nil
}

func fetchDashboardSeriesSummary(
	CarID int,
	parsedStartDate string,
	parsedEndDate string,
	unitsLength string,
	seriesLimit int,
	seriesMonths int,
	locationLimit int,
) (*DashboardSeriesSummary, error) {
	efficiencySeries, err := fetchDashboardEfficiencySeries(CarID, parsedStartDate, parsedEndDate, unitsLength, seriesLimit)
	if err != nil {
		return nil, err
	}
	monthlyDistance, err := fetchDashboardMonthlyDistance(CarID, parsedStartDate, parsedEndDate, unitsLength, seriesMonths)
	if err != nil {
		return nil, err
	}
	monthlyChargeEnergy, err := fetchDashboardMonthlyChargeEnergy(CarID, parsedStartDate, parsedEndDate, seriesMonths)
	if err != nil {
		return nil, err
	}
	chargeLocations, err := fetchDashboardChargeLocations(CarID, parsedStartDate, parsedEndDate, locationLimit)
	if err != nil {
		return nil, err
	}

	return &DashboardSeriesSummary{
		EfficiencySeries:    efficiencySeries,
		MonthlyDistance:     monthlyDistance,
		MonthlyChargeEnergy: monthlyChargeEnergy,
		ChargeLocations:     chargeLocations,
	}, nil
}

func fetchDashboardEfficiencySeries(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string, limit int) ([]SummaryTimeSeriesPoint, error) {
	query := `
		WITH filtered_drives AS (
			SELECT
				drives.id,
				drives.start_date,
				CASE
					WHEN (
						drives.duration_min > 1
						AND drives.distance > 1
						AND (
							start_position.usable_battery_level IS NULL
							OR end_position.usable_battery_level IS NULL
							OR (end_position.battery_level - end_position.usable_battery_level) = 0
						)
					)
					AND NULLIF(drives.distance, 0) IS NOT NULL
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / NULLIF(drives.distance, 0) * 1000
					ELSE NULL
				END AS value
			FROM drives
			LEFT JOIN cars ON drives.car_id = cars.id
			LEFT JOIN positions start_position ON drives.start_position_id = start_position.id
			LEFT JOIN positions end_position ON drives.end_position_id = end_position.id
			WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, paramIndex = appendSummaryDateFilters(query, queryParams, paramIndex, "drives", parsedStartDate, parsedEndDate)
	query += fmt.Sprintf(`
		),
		latest_points AS (
			SELECT id, start_date, value
			FROM filtered_drives
			WHERE value > 0
			ORDER BY start_date DESC
			LIMIT $%d
		)
		SELECT id, start_date, value
		FROM latest_points
		ORDER BY start_date ASC;`, paramIndex)
	queryParams = append(queryParams, limit)

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]SummaryTimeSeriesPoint, 0)
	for rows.Next() {
		var (
			id    int
			date  string
			value float64
		)
		if err := rows.Scan(&id, &date, &value); err != nil {
			return nil, err
		}
		if unitsLength == "mi" {
			value = kilometersToMiles(value)
		}
		points = append(points, SummaryTimeSeriesPoint{
			ID:    strconv.Itoa(id),
			Date:  getTimeInTimeZone(date),
			Value: value,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

func fetchDashboardMonthlyDistance(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string, limit int) ([]SummaryCategoryValue, error) {
	query := `
		WITH monthly AS (
			SELECT
				DATE_TRUNC('month', drives.start_date) AS month,
				COALESCE(SUM(GREATEST(COALESCE(drives.distance, 0), 0)), 0) AS value
			FROM drives
			WHERE drives.car_id=$1 AND drives.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, paramIndex = appendSummaryDateFilters(query, queryParams, paramIndex, "drives", parsedStartDate, parsedEndDate)
	query += fmt.Sprintf(`
			GROUP BY DATE_TRUNC('month', drives.start_date)
		),
		latest_months AS (
			SELECT month, value
			FROM monthly
			WHERE value > 0
			ORDER BY month DESC
			LIMIT $%d
		)
		SELECT
			TO_CHAR(month, 'YYYY-MM') AS period,
			TO_CHAR(month, 'FMMon') AS label,
			value
		FROM latest_months
		ORDER BY month ASC;`, paramIndex)
	queryParams = append(queryParams, limit)

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SummaryCategoryValue, 0)
	for rows.Next() {
		var (
			period string
			label  string
			value  float64
		)
		if err := rows.Scan(&period, &label, &value); err != nil {
			return nil, err
		}
		if unitsLength == "mi" {
			value = kilometersToMiles(value)
		}
		items = append(items, SummaryCategoryValue{
			ID:     "distance-" + period,
			Label:  label,
			Period: period,
			Value:  value,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func fetchDashboardMonthlyChargeEnergy(CarID int, parsedStartDate string, parsedEndDate string, limit int) ([]SummaryCategoryValue, error) {
	query := `
		WITH monthly AS (
			SELECT
				DATE_TRUNC('month', charging_processes.start_date) AS month,
				COALESCE(SUM(GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0)), 0) AS value
			FROM charging_processes
			WHERE charging_processes.car_id=$1 AND charging_processes.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, paramIndex = appendSummaryDateFilters(query, queryParams, paramIndex, "charging_processes", parsedStartDate, parsedEndDate)
	query += fmt.Sprintf(`
			GROUP BY DATE_TRUNC('month', charging_processes.start_date)
		),
		latest_months AS (
			SELECT month, value
			FROM monthly
			WHERE value > 0
			ORDER BY month DESC
			LIMIT $%d
		)
		SELECT
			TO_CHAR(month, 'YYYY-MM') AS period,
			TO_CHAR(month, 'FMMon') AS label,
			value
		FROM latest_months
		ORDER BY month ASC;`, paramIndex)
	queryParams = append(queryParams, limit)

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SummaryCategoryValue, 0)
	for rows.Next() {
		var (
			period string
			label  string
			value  float64
		)
		if err := rows.Scan(&period, &label, &value); err != nil {
			return nil, err
		}
		items = append(items, SummaryCategoryValue{
			ID:     "charge-energy-" + period,
			Label:  label,
			Period: period,
			Value:  value,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func fetchDashboardChargeLocations(CarID int, parsedStartDate string, parsedEndDate string, limit int) ([]SummaryCategoryValue, error) {
	query := `
		WITH filtered_charges AS (
			SELECT
				COALESCE(
					NULLIF(
						TRIM(COALESCE(
							geofence.name,
							CONCAT_WS(', ', COALESCE(address.name, NULLIF(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)
						)),
						''
					),
					'Unknown'
				) AS location,
				GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0) AS value
			FROM charging_processes
			LEFT JOIN addresses address ON address_id = address.id
			LEFT JOIN geofences geofence ON geofence_id = geofence.id
			WHERE charging_processes.car_id=$1 AND charging_processes.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, paramIndex = appendSummaryDateFilters(query, queryParams, paramIndex, "charging_processes", parsedStartDate, parsedEndDate)
	query += fmt.Sprintf(`
		)
		SELECT location, SUM(value) AS total
		FROM filtered_charges
		WHERE value > 0
		GROUP BY location
		ORDER BY total DESC
		LIMIT $%d;`, paramIndex)
	queryParams = append(queryParams, limit)

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SummaryCategoryValue, 0)
	index := 0
	for rows.Next() {
		var (
			label string
			value float64
		)
		if err := rows.Scan(&label, &value); err != nil {
			return nil, err
		}
		items = append(items, SummaryCategoryValue{
			ID:    fmt.Sprintf("charge-location-%d", index),
			Label: label,
			Value: value,
		})
		index++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func appendSummaryDateFilters(query string, queryParams []any, paramIndex int, table string, parsedStartDate string, parsedEndDate string) (string, []any, int) {
	if parsedStartDate != "" {
		query += fmt.Sprintf(" AND %s.start_date >= $%d", table, paramIndex)
		queryParams = append(queryParams, parsedStartDate)
		paramIndex++
	}
	if parsedEndDate != "" {
		query += fmt.Sprintf(" AND %s.end_date <= $%d", table, paramIndex)
		queryParams = append(queryParams, parsedEndDate)
		paramIndex++
	}
	return query, queryParams, paramIndex
}

func makeLifetimeConsumptionSummary(driveSummary *DriveHistorySummary, chargeSummary *ChargeHistorySummary) *LifetimeConsumptionSummary {
	if driveSummary == nil || chargeSummary == nil {
		return nil
	}

	totalDistance := driveSummary.TotalDistance
	totalEnergyAdded := chargeSummary.TotalEnergyAdded
	totalEnergyUsed := 0.0
	if chargeSummary.TotalEnergyUsed != nil {
		totalEnergyUsed = *chargeSummary.TotalEnergyUsed
	}
	if totalDistance <= 0 || (totalEnergyAdded <= 0 && totalEnergyUsed <= 0) {
		return nil
	}

	var batteryConsumptionPer100Distance *float64
	if totalEnergyAdded > 0 {
		value := totalEnergyAdded / totalDistance * 100.0
		batteryConsumptionPer100Distance = &value
	}

	var wallConsumptionPer100Distance *float64
	if totalEnergyUsed > 0 {
		value := totalEnergyUsed / totalDistance * 100.0
		wallConsumptionPer100Distance = &value
	}

	var chargingEfficiency *float64
	if totalEnergyUsed > 0 && totalEnergyAdded > 0 {
		value := totalEnergyAdded / totalEnergyUsed
		chargingEfficiency = &value
	}

	return &LifetimeConsumptionSummary{
		BatteryConsumptionPer100Distance: batteryConsumptionPer100Distance,
		WallConsumptionPer100Distance:    wallConsumptionPer100Distance,
		ChargingEfficiency:               chargingEfficiency,
		TotalDistance:                    totalDistance,
		TotalEnergyAdded:                 totalEnergyAdded,
		TotalEnergyUsed:                  totalEnergyUsed,
		DriveCount:                       driveSummary.DriveCount,
		ChargeCount:                      chargeSummary.ChargeCount,
	}
}

func floatPointer(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func intPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func timeZoneStringPointer(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	result := getTimeInTimeZone(value.String)
	return &result
}

func kilometersToMilesSqlNullFloat64(value sql.NullFloat64) sql.NullFloat64 {
	if value.Valid {
		value.Float64 = kilometersToMiles(value.Float64)
	}
	return value
}

func kilometersToMilesSqlNullInt64(value sql.NullInt64) sql.NullInt64 {
	if value.Valid {
		value.Int64 = int64(kilometersToMilesInteger(int(value.Int64)))
	}
	return value
}

func TeslaMateAPISummaryOptionsV1(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"parameters": gin.H{
			"startDate":     "optional RFC3339 date; filters records starting at or after this time",
			"endDate":       "optional RFC3339 date; filters records ending at or before this time",
			"include":       "combined /summaries only; optional comma-separated list: all, overview, lifetime, drives, charges, parking, analysis, statistics, states, series",
			"limit":         "chart series/location endpoints; optional integer limit",
			"months":        "chart monthly endpoints; optional integer month bucket count",
			"seriesLimit":   "combined /summaries only; optional integer; default 12, max 100",
			"seriesMonths":  "combined /summaries only; optional integer; default 6, max 24",
			"locationLimit": "combined /summaries only; optional integer; default 4, max 20",
			"types":         "insight events only; optional comma-separated list: harsh_brake, charge_power_drop, sleep_interruption",
			"page":          "parking, state timeline, insight events; optional integer page number",
			"show":          "parking, state timeline, insight events; optional integer page size",
			"year":          "drive calendar only; optional integer year",
			"month":         "drive calendar only; optional integer month (1-12)",
		},
		"endpoints": []string{
			"/api/v1/cars/:CarID/summaries",
			"/api/v1/cars/:CarID/summaries/overview",
			"/api/v1/cars/:CarID/parking-sessions",
			"/api/v1/cars/:CarID/summaries/lifetime",
			"/api/v1/cars/:CarID/summaries/drives",
			"/api/v1/cars/:CarID/summaries/charges",
			"/api/v1/cars/:CarID/summaries/parking",
			"/api/v1/cars/:CarID/summaries/statistics",
			"/api/v1/cars/:CarID/summaries/state-activity",
			"/api/v1/cars/:CarID/analytics/activity",
			"/api/v1/cars/:CarID/analytics/regeneration",
			"/api/v1/cars/:CarID/activity-timeline",
			"/api/v1/cars/:CarID/dashboards/drives",
			"/api/v1/cars/:CarID/dashboards/charges",
			"/api/v1/cars/:CarID/insights",
			"/api/v1/cars/:CarID/insights/events",
			"/api/v1/cars/:CarID/calendars/drives",
			"/api/v1/cars/:CarID/charts/efficiency",
			"/api/v1/cars/:CarID/charts/drives/monthly-distance",
			"/api/v1/cars/:CarID/charts/drives/weekday-distance",
			"/api/v1/cars/:CarID/charts/drives/hourly-starts",
			"/api/v1/cars/:CarID/charts/charges/monthly-energy",
			"/api/v1/cars/:CarID/charts/charges/location-energy",
			"/api/v1/cars/:CarID/charts/charges/weekday-energy",
			"/api/v1/cars/:CarID/charts/charges/hourly-starts",
			"/api/v1/cars/:CarID/charts/activity/duration",
			"/api/v1/docs",
			"/api/v1/docs/swagger",
		},
	})
}
