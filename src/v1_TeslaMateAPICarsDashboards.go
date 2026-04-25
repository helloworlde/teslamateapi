package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DashboardWindow struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type DriveDashboardPeriod struct {
	Window  DashboardWindow      `json:"window"`
	Summary *DriveHistorySummary `json:"summary"`
}

type ChargeDashboardPeriod struct {
	Window  DashboardWindow       `json:"window"`
	Summary *ChargeHistorySummary `json:"summary"`
}

type DriveDashboardSet struct {
	Today DriveDashboardPeriod `json:"today"`
	Week  DriveDashboardPeriod `json:"week"`
	Month DriveDashboardPeriod `json:"month"`
	Year  DriveDashboardPeriod `json:"year"`
}

type ChargeDashboardSet struct {
	Today ChargeDashboardPeriod `json:"today"`
	Week  ChargeDashboardPeriod `json:"week"`
	Month ChargeDashboardPeriod `json:"month"`
	Year  ChargeDashboardPeriod `json:"year"`
}

type dashboardWindowInternal struct {
	DashboardWindow
	StartUTC string
	EndUTC   string
}

func TeslaMateAPICarsDriveDashboardsV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsDriveDashboardsV1"

	CarID, err := parseCarID(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusBadRequest, actionName, "Invalid CarID parameter.", err.Error())
		return
	}
	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if respondSummaryMetadataError(c, actionName, err, "Unable to load drive dashboards.") {
		return
	}

	dashboards, err := fetchDriveDashboards(CarID, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load drive dashboards.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, "", "", unitsLength, unitsTemperature)
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"dashboards": dashboards,
	}))
}

func TeslaMateAPICarsChargeDashboardsV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsChargeDashboardsV1"

	CarID, err := parseCarID(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusBadRequest, actionName, "Invalid CarID parameter.", err.Error())
		return
	}
	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if respondSummaryMetadataError(c, actionName, err, "Unable to load charge dashboards.") {
		return
	}

	dashboards, err := fetchChargeDashboards(CarID, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load charge dashboards.", err.Error())
		return
	}

	data := makeSummaryResponseData(CarID, carName, "", "", unitsLength, unitsTemperature)
	TeslaMateAPIHandleSuccessResponse(c, actionName, focusedSummaryResponse(data, gin.H{
		"dashboards": dashboards,
	}))
}

func fetchDriveDashboards(CarID int, unitsLength string) (*DriveDashboardSet, error) {
	windows := makeDashboardWindows()
	result := &DriveDashboardSet{}

	for _, window := range windows {
		summary, err := fetchDriveHistorySummary(CarID, window.StartUTC, window.EndUTC, unitsLength)
		if err != nil {
			return nil, err
		}
		period := DriveDashboardPeriod{
			Window:  window.DashboardWindow,
			Summary: summary,
		}
		switch window.Key {
		case "today":
			result.Today = period
		case "week":
			result.Week = period
		case "month":
			result.Month = period
		case "year":
			result.Year = period
		}
	}

	return result, nil
}

func fetchChargeDashboards(CarID int, unitsLength string) (*ChargeDashboardSet, error) {
	windows := makeDashboardWindows()
	result := &ChargeDashboardSet{}

	for _, window := range windows {
		summary, err := fetchChargeHistorySummary(CarID, window.StartUTC, window.EndUTC, unitsLength)
		if err != nil {
			return nil, err
		}
		period := ChargeDashboardPeriod{
			Window:  window.DashboardWindow,
			Summary: summary,
		}
		switch window.Key {
		case "today":
			result.Today = period
		case "week":
			result.Week = period
		case "month":
			result.Month = period
		case "year":
			result.Year = period
		}
	}

	return result, nil
}

func makeDashboardWindows() []dashboardWindowInternal {
	now := time.Now().In(appUsersTimezone)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, appUsersTimezone)
	weekStart := todayStart.AddDate(0, 0, -((int(todayStart.Weekday()) + 6) % 7))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, appUsersTimezone)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, appUsersTimezone)

	makeWindow := func(key string, label string, start time.Time, end time.Time) dashboardWindowInternal {
		return dashboardWindowInternal{
			DashboardWindow: DashboardWindow{
				Key:       key,
				Label:     label,
				StartDate: start.Format(time.RFC3339),
				EndDate:   end.Add(-time.Second).Format(time.RFC3339),
			},
			StartUTC: start.UTC().Format(dbTimestampFormat),
			EndUTC:   end.UTC().Format(dbTimestampFormat),
		}
	}

	return []dashboardWindowInternal{
		makeWindow("today", "Today", todayStart, todayStart.AddDate(0, 0, 1)),
		makeWindow("week", "This Week", weekStart, weekStart.AddDate(0, 0, 7)),
		makeWindow("month", "This Month", monthStart, monthStart.AddDate(0, 1, 0)),
		makeWindow("year", "This Year", yearStart, yearStart.AddDate(1, 0, 0)),
	}
}
