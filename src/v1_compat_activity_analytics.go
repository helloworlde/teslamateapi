package main

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type OverviewSummary struct {
	Coverage                         HistorySummaryCoverage `json:"coverage"`
	DriveCount                       int                    `json:"drive_count"`
	ChargeCount                      int                    `json:"charge_count"`
	ParkingSessionCount              int                    `json:"parking_session_count"`
	TotalDriveDurationMin            int                    `json:"total_drive_duration_min"`
	TotalChargeDurationMin           int                    `json:"total_charge_duration_min"`
	TotalParkingDurationMin          int                    `json:"total_parking_duration_min"`
	TotalDistance                    float64                `json:"total_distance"`
	TotalEnergyAdded                 float64                `json:"total_energy_added"`
	TotalEnergyUsed                  *float64               `json:"total_energy_used,omitempty"`
	TotalEnergyConsumed              *float64               `json:"total_energy_consumed"`
	TotalChargeCost                  *float64               `json:"total_charge_cost,omitempty"`
	ParkingDurationMin               int                    `json:"parking_duration_min"`
	AverageConsumption               *float64               `json:"average_consumption,omitempty"`
	AverageDriveDistance             *float64               `json:"average_drive_distance"`
	AverageChargeEnergyAdded         *float64               `json:"average_charge_energy_added"`
	ChargingEfficiency               *float64               `json:"charging_efficiency"`
	BatteryConsumptionPer100Distance *float64               `json:"battery_consumption_per_100_distance"`
	WallConsumptionPer100Distance    *float64               `json:"wall_consumption_per_100_distance"`
	LastDriveDate                    *string                `json:"last_drive_date"`
	LastChargeDate                   *string                `json:"last_charge_date"`
	LastParkingDate                  *string                `json:"last_parking_date"`
	AverageSpeed                     *float64               `json:"average_speed,omitempty"`
	QueryStartDate                   *string                `json:"query_start_date,omitempty"`
	QueryEndDate                     *string                `json:"query_end_date,omitempty"`
}

type ActivityShareSummary struct {
	DrivingDurationMin  int      `json:"driving_duration_min"`
	ChargingDurationMin int      `json:"charging_duration_min"`
	ParkingDurationMin  int      `json:"parking_duration_min"`
	DrivingShare        *float64 `json:"driving_share"`
	ChargingShare       *float64 `json:"charging_share"`
	ParkingShare        *float64 `json:"parking_share"`
}

type AnalysisSummary struct {
	ActivityShare          *ActivityShareSummary  `json:"activity_share"`
	DriveWeekdayDistance   []SummaryCategoryValue `json:"drive_weekday_distance"`
	ChargeWeekdayEnergy    []SummaryCategoryValue `json:"charge_weekday_energy"`
	DriveHourlyStartCount  []SummaryCategoryValue `json:"drive_hourly_start_count"`
	ChargeHourlyStartCount []SummaryCategoryValue `json:"charge_hourly_start_count"`
	ParkingStateDuration   []SummaryCategoryValue `json:"parking_state_duration"`
	StateDurationShare     []SummaryCategoryValue `json:"state_duration_share"`
	SleepRatio             *float64               `json:"sleep_ratio,omitempty"`
	OnlineAwakeRatio       *float64               `json:"online_awake_ratio,omitempty"`
}

func TeslaMateAPICarsOverviewV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsOverviewV1"

	CarID, err := parseCarID(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusBadRequest, actionName, "Invalid CarID parameter.", err.Error())
		return
	}
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusBadRequest, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if respondSummaryMetadataError(c, actionName, err, "Unable to load overview.") {
		return
	}

	driveSummary, err := fetchDriveHistorySummary(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load overview.", err.Error())
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load overview.", err.Error())
		return
	}
	parkingSummary, err := fetchParkingHistorySummary(CarID, parsedStartDate, parsedEndDate, nil)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load overview.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.Overview = makeOverviewSummary(driveSummary, chargeSummary, parkingSummary, summaryFilterDate(parsedStartDate), summaryFilterDate(parsedEndDate))

	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"overview": data.Overview,
	}))
}

func TeslaMateAPICarsAnalyticsV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsAnalyticsV1"

	CarID, err := parseCarID(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusBadRequest, actionName, "Invalid CarID parameter.", err.Error())
		return
	}
	parsedStartDate, parsedEndDate, err := parseSummaryDateRange(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusBadRequest, actionName, "Invalid date format.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if respondSummaryMetadataError(c, actionName, err, "Unable to load analytics.") {
		return
	}

	driveSummary, err := fetchDriveHistorySummary(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load analytics.", err.Error())
		return
	}
	chargeSummary, err := fetchChargeHistorySummary(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load analytics.", err.Error())
		return
	}
	parkingSummary, err := fetchParkingHistorySummary(CarID, parsedStartDate, parsedEndDate, nil)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load analytics.", err.Error())
		return
	}

	analysisSummary, err := fetchAnalysisSummary(CarID, parsedStartDate, parsedEndDate, unitsLength, driveSummary, chargeSummary, parkingSummary)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load analytics.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, parsedStartDate, parsedEndDate, unitsLength, unitsTemperature)
	data.AnalysisSummary = analysisSummary

	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"analysis_summary": data.AnalysisSummary,
	}))
}

func TeslaMateAPICarsChartDriveWeekdayV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsChartDriveWeekdayV1"
	TeslaMateAPIHandleChartCategoryResponse(c, actionName, "Unable to load drive weekday chart.", func(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) ([]SummaryCategoryValue, error) {
		return fetchDriveWeekdayDistance(CarID, parsedStartDate, parsedEndDate, unitsLength)
	}, "drive_weekday_distance")
}

func TeslaMateAPICarsChartDriveHourlyV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsChartDriveHourlyV1"
	TeslaMateAPIHandleChartCategoryResponse(c, actionName, "Unable to load drive hourly chart.", func(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) ([]SummaryCategoryValue, error) {
		return fetchDriveHourlyStartCount(CarID, parsedStartDate, parsedEndDate)
	}, "drive_hourly_start_count")
}

func TeslaMateAPICarsChartChargeWeekdayV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsChartChargeWeekdayV1"
	TeslaMateAPIHandleChartCategoryResponse(c, actionName, "Unable to load charge weekday chart.", func(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) ([]SummaryCategoryValue, error) {
		return fetchChargeWeekdayEnergy(CarID, parsedStartDate, parsedEndDate)
	}, "charge_weekday_energy")
}

func TeslaMateAPICarsChartChargeHourlyV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsChartChargeHourlyV1"
	TeslaMateAPIHandleChartCategoryResponse(c, actionName, "Unable to load charge hourly chart.", func(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) ([]SummaryCategoryValue, error) {
		return fetchChargeHourlyStartCount(CarID, parsedStartDate, parsedEndDate)
	}, "charge_hourly_start_count")
}

func fetchAnalysisSummary(
	CarID int,
	parsedStartDate string,
	parsedEndDate string,
	unitsLength string,
	driveSummary *DriveHistorySummary,
	chargeSummary *ChargeHistorySummary,
	parkingSummary *ParkingHistorySummary,
) (*AnalysisSummary, error) {
	driveWeekdayDistance, err := fetchDriveWeekdayDistance(CarID, parsedStartDate, parsedEndDate, unitsLength)
	if err != nil {
		return nil, err
	}
	chargeWeekdayEnergy, err := fetchChargeWeekdayEnergy(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		return nil, err
	}
	driveHourlyStartCount, err := fetchDriveHourlyStartCount(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		return nil, err
	}
	chargeHourlyStartCount, err := fetchChargeHourlyStartCount(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		return nil, err
	}
	stateShare, sleepR, onlineR, err := fetchStateAnalyticsRatios(CarID, parsedStartDate, parsedEndDate)
	if err != nil {
		return nil, err
	}
	parkingStateDuration := make([]SummaryCategoryValue, 0)
	if parkingSummary != nil {
		for index, item := range parkingSummary.StateBreakdown {
			parkingStateDuration = append(parkingStateDuration, SummaryCategoryValue{
				ID:     fmt.Sprintf("parking-state-%d", index),
				Label:  item.State,
				Period: "parking_state",
				Unit:   "min",
				Value:  float64(item.DurationMin),
			})
		}
	}

	return &AnalysisSummary{
		ActivityShare:          makeActivityShareSummary(driveSummary, chargeSummary, parkingSummary),
		DriveWeekdayDistance:   driveWeekdayDistance,
		ChargeWeekdayEnergy:    chargeWeekdayEnergy,
		DriveHourlyStartCount:  driveHourlyStartCount,
		ChargeHourlyStartCount: chargeHourlyStartCount,
		ParkingStateDuration:   parkingStateDuration,
		StateDurationShare:     stateShare,
		SleepRatio:             sleepR,
		OnlineAwakeRatio:       onlineR,
	}, nil
}

func fetchStateAnalyticsRatios(CarID int, parsedStartDate, parsedEndDate string) ([]SummaryCategoryValue, *float64, *float64, error) {
	q := `
		SELECT
			states.state::text AS state,
			COALESCE(SUM(GREATEST(EXTRACT(EPOCH FROM (COALESCE(states.end_date, NOW() AT TIME ZONE 'UTC') - states.start_date)) / 60.0, 0)), 0)::float8 AS duration_min
		FROM states
		WHERE states.car_id = $1`
	params := []any{CarID}
	idx := 2
	q, params, idx = appendStateTimelineDateFilters(q, params, idx, "states.start_date", "states.end_date", parsedStartDate, parsedEndDate)
	q += ` GROUP BY states.state`

	rows, err := db.Query(q, params...)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	byState := make(map[string]float64)
	for rows.Next() {
		var st string
		var dm float64
		if err := rows.Scan(&st, &dm); err != nil {
			return nil, nil, nil, err
		}
		byState[st] += dm
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	total := 0.0
	for _, v := range byState {
		total += v
	}
	if total <= 0 {
		return []SummaryCategoryValue{}, nil, nil, nil
	}

	type kv struct {
		k string
		v float64
	}
	list := make([]kv, 0, len(byState))
	for k, v := range byState {
		list = append(list, kv{k: k, v: v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].k < list[j].k })

	out := make([]SummaryCategoryValue, 0, len(list))
	for i, e := range list {
		share := e.v / total
		out = append(out, SummaryCategoryValue{
			ID:     fmt.Sprintf("state-share-%d", i+1),
			Label:  e.k,
			Period: "state",
			Unit:   "ratio",
			Value:  share,
			Extra:  map[string]any{"duration_min": e.v},
		})
	}

	var sleepRatio, onlineRatio *float64
	if v, ok := byState["asleep"]; ok {
		s := v / total
		sleepRatio = &s
	}
	if v, ok := byState["online"]; ok {
		s := v / total
		onlineRatio = &s
	}
	return out, sleepRatio, onlineRatio, nil
}

func fetchDriveWeekdayDistance(CarID int, parsedStartDate string, parsedEndDate string, unitsLength string) ([]SummaryCategoryValue, error) {
	query := `
		SELECT
			EXTRACT(ISODOW FROM drives.start_date)::int AS bucket,
			COALESCE(SUM(GREATEST(COALESCE(drives.distance, 0), 0)), 0) AS value
		FROM drives
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, _ = appendSummaryDateFilters(query, queryParams, paramIndex, "drives", parsedStartDate, parsedEndDate)
	query += `
		GROUP BY bucket
		ORDER BY bucket ASC;`

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := makeWeekdayBuckets("drive-weekday", "weekday", chartDistanceUnit(unitsLength))
	for rows.Next() {
		var (
			bucket int
			value  float64
		)
		if err := rows.Scan(&bucket, &value); err != nil {
			return nil, err
		}
		if bucket < 1 || bucket > len(result) {
			continue
		}
		if unitsLength == "mi" {
			value = kilometersToMiles(value)
		}
		result[bucket-1].Value = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func fetchChargeWeekdayEnergy(CarID int, parsedStartDate string, parsedEndDate string) ([]SummaryCategoryValue, error) {
	query := `
		SELECT
			EXTRACT(ISODOW FROM charging_processes.start_date)::int AS bucket,
			COALESCE(SUM(GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0)), 0) AS value
		FROM charging_processes
		WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, _ = appendSummaryDateFilters(query, queryParams, paramIndex, "charging_processes", parsedStartDate, parsedEndDate)
	query += `
		GROUP BY bucket
		ORDER BY bucket ASC;`

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := makeWeekdayBuckets("charge-weekday", "weekday", "kWh")
	for rows.Next() {
		var (
			bucket int
			value  float64
		)
		if err := rows.Scan(&bucket, &value); err != nil {
			return nil, err
		}
		if bucket < 1 || bucket > len(result) {
			continue
		}
		result[bucket-1].Value = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func fetchDriveHourlyStartCount(CarID int, parsedStartDate string, parsedEndDate string) ([]SummaryCategoryValue, error) {
	query := `
		SELECT
			EXTRACT(HOUR FROM drives.start_date)::int AS bucket,
			COUNT(*) AS value
		FROM drives
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, _ = appendSummaryDateFilters(query, queryParams, paramIndex, "drives", parsedStartDate, parsedEndDate)
	query += `
		GROUP BY bucket
		ORDER BY bucket ASC;`

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := makeHourBuckets("drive-hour", "hour", "sessions")
	for rows.Next() {
		var (
			bucket int
			value  int
		)
		if err := rows.Scan(&bucket, &value); err != nil {
			return nil, err
		}
		if bucket < 0 || bucket >= len(result) {
			continue
		}
		result[bucket].Value = float64(value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func fetchChargeHourlyStartCount(CarID int, parsedStartDate string, parsedEndDate string) ([]SummaryCategoryValue, error) {
	query := `
		SELECT
			EXTRACT(HOUR FROM charging_processes.start_date)::int AS bucket,
			COUNT(*) AS value
		FROM charging_processes
		WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL`

	queryParams := []any{CarID}
	paramIndex := 2
	query, queryParams, _ = appendSummaryDateFilters(query, queryParams, paramIndex, "charging_processes", parsedStartDate, parsedEndDate)
	query += `
		GROUP BY bucket
		ORDER BY bucket ASC;`

	rows, err := db.Query(query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := makeHourBuckets("charge-hour", "hour", "sessions")
	for rows.Next() {
		var (
			bucket int
			value  int
		)
		if err := rows.Scan(&bucket, &value); err != nil {
			return nil, err
		}
		if bucket < 0 || bucket >= len(result) {
			continue
		}
		result[bucket].Value = float64(value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func makeOverviewSummary(driveSummary *DriveHistorySummary, chargeSummary *ChargeHistorySummary, parkingSummary *ParkingHistorySummary, queryStartDate *string, queryEndDate *string) *OverviewSummary {
	if driveSummary == nil && chargeSummary == nil && parkingSummary == nil {
		return nil
	}

	lifetimeSummary := makeLifetimeConsumptionSummary(driveSummary, chargeSummary)
	overview := &OverviewSummary{
		Coverage: HistorySummaryCoverage{
			StartDate: minSummaryDate(
				coverageStartDate(driveSummary),
				coverageStartDate(chargeSummary),
				coverageStartDate(parkingSummary),
			),
			EndDate: maxSummaryDate(
				coverageEndDate(driveSummary),
				coverageEndDate(chargeSummary),
				coverageEndDate(parkingSummary),
			),
		},
		LastDriveDate:   coverageEndDate(driveSummary),
		LastChargeDate:  coverageEndDate(chargeSummary),
		LastParkingDate: coverageEndDate(parkingSummary),
		QueryStartDate:  queryStartDate,
		QueryEndDate:    queryEndDate,
	}

	if driveSummary != nil {
		overview.DriveCount = driveSummary.DriveCount
		overview.TotalDriveDurationMin = driveSummary.TotalDurationMin
		overview.TotalDistance = driveSummary.TotalDistance
		overview.AverageDriveDistance = driveSummary.AverageDistance
		overview.TotalEnergyConsumed = driveSummary.TotalEnergyConsumed
		overview.AverageConsumption = driveSummary.AverageConsumption
		overview.AverageSpeed = driveSummary.AverageSpeed
	}
	if chargeSummary != nil {
		overview.ChargeCount = chargeSummary.ChargeCount
		overview.TotalChargeDurationMin = chargeSummary.TotalDurationMin
		overview.TotalEnergyAdded = chargeSummary.TotalEnergyAdded
		overview.TotalEnergyUsed = chargeSummary.TotalEnergyUsed
		overview.TotalChargeCost = chargeSummary.TotalCost
		overview.AverageChargeEnergyAdded = chargeSummary.AverageEnergyAdded
		overview.ChargingEfficiency = chargeSummary.ChargingEfficiency
	}
	if parkingSummary != nil {
		overview.ParkingSessionCount = parkingSummary.SessionCount
		overview.TotalParkingDurationMin = parkingSummary.TotalDurationMin
		overview.ParkingDurationMin = parkingSummary.TotalDurationMin
	}
	if lifetimeSummary != nil {
		overview.BatteryConsumptionPer100Distance = lifetimeSummary.BatteryConsumptionPer100Distance
		overview.WallConsumptionPer100Distance = lifetimeSummary.WallConsumptionPer100Distance
		if overview.ChargingEfficiency == nil {
			overview.ChargingEfficiency = lifetimeSummary.ChargingEfficiency
		}
	}

	return overview
}

func makeActivityShareSummary(driveSummary *DriveHistorySummary, chargeSummary *ChargeHistorySummary, parkingSummary *ParkingHistorySummary) *ActivityShareSummary {
	drivingDuration := 0
	if driveSummary != nil {
		drivingDuration = driveSummary.TotalDurationMin
	}
	chargingDuration := 0
	if chargeSummary != nil {
		chargingDuration = chargeSummary.TotalDurationMin
	}
	parkingDuration := 0
	if parkingSummary != nil {
		parkingDuration = parkingSummary.TotalDurationMin
	}

	totalDuration := drivingDuration + chargingDuration + parkingDuration
	if totalDuration == 0 {
		return nil
	}

	drivingShare := float64(drivingDuration) / float64(totalDuration)
	chargingShare := float64(chargingDuration) / float64(totalDuration)
	parkingShare := float64(parkingDuration) / float64(totalDuration)

	return &ActivityShareSummary{
		DrivingDurationMin:  drivingDuration,
		ChargingDurationMin: chargingDuration,
		ParkingDurationMin:  parkingDuration,
		DrivingShare:        &drivingShare,
		ChargingShare:       &chargingShare,
		ParkingShare:        &parkingShare,
	}
}

func chartDistanceUnit(unitsLength string) string {
	if unitsLength == "mi" {
		return "mi"
	}
	return "km"
}

func makeWeekdayBuckets(prefix, period, unit string) []SummaryCategoryValue {
	labels := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	result := make([]SummaryCategoryValue, 0, len(labels))
	for index, label := range labels {
		result = append(result, SummaryCategoryValue{
			ID:     fmt.Sprintf("%s-%d", prefix, index+1),
			Label:  label,
			Period: period,
			Unit:   unit,
			Value:  0,
		})
	}
	return result
}

func makeHourBuckets(prefix, period, unit string) []SummaryCategoryValue {
	result := make([]SummaryCategoryValue, 0, 24)
	for hour := 0; hour < 24; hour++ {
		result = append(result, SummaryCategoryValue{
			ID:     fmt.Sprintf("%s-%02d", prefix, hour),
			Label:  fmt.Sprintf("%02d:00", hour),
			Period: period,
			Unit:   unit,
			Value:  0,
		})
	}
	return result
}

func formatDurationMinutes(durationMin int) string {
	if durationMin < 0 {
		durationMin = 0
	}

	days := durationMin / (24 * 60)
	hours := (durationMin % (24 * 60)) / 60
	minutes := durationMin % 60

	if days > 0 {
		return fmt.Sprintf("%dd %02d:%02d", days, hours, minutes)
	}
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

func coverageStartDate[T interface{ getCoverageStart() *string }](item T) *string {
	return item.getCoverageStart()
}

func coverageEndDate[T interface{ getCoverageEnd() *string }](item T) *string {
	return item.getCoverageEnd()
}

func (d *DriveHistorySummary) getCoverageStart() *string {
	if d == nil {
		return nil
	}
	return d.Coverage.StartDate
}

func (d *DriveHistorySummary) getCoverageEnd() *string {
	if d == nil {
		return nil
	}
	return d.Coverage.EndDate
}

func (c *ChargeHistorySummary) getCoverageStart() *string {
	if c == nil {
		return nil
	}
	return c.Coverage.StartDate
}

func (c *ChargeHistorySummary) getCoverageEnd() *string {
	if c == nil {
		return nil
	}
	return c.Coverage.EndDate
}

func (p *ParkingHistorySummary) getCoverageStart() *string {
	if p == nil {
		return nil
	}
	return p.Coverage.StartDate
}

func (p *ParkingHistorySummary) getCoverageEnd() *string {
	if p == nil {
		return nil
	}
	return p.Coverage.EndDate
}

func minSummaryDate(values ...*string) *string {
	return compareSummaryDates(true, values...)
}

func maxSummaryDate(values ...*string) *string {
	return compareSummaryDates(false, values...)
}

func compareSummaryDates(useMin bool, values ...*string) *string {
	var selected *time.Time
	for _, value := range values {
		if value == nil || *value == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, *value)
		if err != nil {
			continue
		}
		if selected == nil ||
			(useMin && parsed.Before(*selected)) ||
			(!useMin && parsed.After(*selected)) {
			selected = &parsed
		}
	}
	if selected == nil {
		return nil
	}
	formatted := selected.Format(time.RFC3339)
	return &formatted
}
