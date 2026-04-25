package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DriveCalendarDay struct {
	Date               string   `json:"date"`
	DriveCount         int      `json:"drive_count"`
	DurationMin        int      `json:"duration_min"`
	Distance           float64  `json:"distance"`
	EnergyConsumed     *float64 `json:"energy_consumed,omitempty"`
	AverageConsumption *float64 `json:"average_consumption,omitempty"`
	FirstDriveStart    *string  `json:"first_drive_start,omitempty"`
	LastDriveEnd       *string  `json:"last_drive_end,omitempty"`
	Day                int      `json:"day,omitempty"`
	Weekday            int      `json:"weekday,omitempty"`
	IsCurrentMonth     bool     `json:"is_current_month,omitempty"`
	IsToday            bool     `json:"is_today,omitempty"`
}

type DriveCalendarMonth struct {
	Year      int                `json:"year"`
	Month     int                `json:"month"`
	MonthName string             `json:"month_name"`
	StartDate string             `json:"start_date"`
	EndDate   string             `json:"end_date"`
	Days      []DriveCalendarDay `json:"days"`
}

type DriveCalendarFilters struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

type TeslaMateDriveCalendarData struct {
	Car            TeslaMateSummaryCar   `json:"car"`
	Filters        DriveCalendarFilters  `json:"filters"`
	Calendar       DriveCalendarMonth    `json:"calendar"`
	TeslaMateUnits TeslaMateSummaryUnits `json:"units"`
}

type TeslaMateDriveCalendarJSONData struct {
	Data TeslaMateDriveCalendarData `json:"data"`
}

func TeslaMateAPICarsDriveCalendarV1(c *gin.Context) {
	const actionName = "TeslaMateAPICarsDriveCalendarV1"

	CarID, err := parseCarID(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusBadRequest, actionName, "Invalid CarID parameter.", err.Error())
		return
	}
	year, err := parseSummaryPositiveIntParam(c.Query("year"), time.Now().In(appUsersTimezone).Year(), 2012, 2100)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusBadRequest, actionName, "Invalid calendar parameter.", err.Error())
		return
	}
	month, err := parseSummaryPositiveIntParam(c.Query("month"), int(time.Now().In(appUsersTimezone).Month()), 1, 12)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusBadRequest, actionName, "Invalid calendar parameter.", err.Error())
		return
	}

	unitsLength, unitsTemperature, carName, err := fetchSummaryMetadata(CarID)
	if respondSummaryMetadataError(c, actionName, err, "Unable to load drive calendar.") {
		return
	}

	calendar, err := fetchDriveCalendarMonth(CarID, year, month, unitsLength)
	if err != nil {
		TeslaMateAPIHandleErrorResponseWithStatus(c, http.StatusInternalServerError, actionName, "Unable to load drive calendar.", err.Error())
		return
	}

	jsonData := TeslaMateDriveCalendarJSONData{
		Data: TeslaMateDriveCalendarData{
			Car: TeslaMateSummaryCar{
				CarID:   CarID,
				CarName: carName,
			},
			Filters: DriveCalendarFilters{
				Year:  year,
				Month: month,
			},
			Calendar:       calendar,
			TeslaMateUnits: buildSummaryUnits(unitsLength, unitsTemperature),
		},
	}

	TeslaMateAPIHandleSuccessResponse(c, actionName, jsonData)
}

func fetchDriveCalendarMonth(CarID int, year int, month int, unitsLength string) (DriveCalendarMonth, error) {
	startLocal := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, appUsersTimezone)
	endLocal := startLocal.AddDate(0, 1, 0)
	startUTC := startLocal.UTC().Format(dbTimestampFormat)
	endUTC := endLocal.UTC().Format(dbTimestampFormat)

	query := `
		SELECT
			timezone($4, drives.start_date)::date AS local_date,
			COUNT(*)::int AS drive_count,
			COALESCE(SUM(GREATEST(COALESCE(drives.duration_min, 0), 0)), 0)::int AS total_duration_min,
			COALESCE(SUM(GREATEST(COALESCE(drives.distance, 0), 0)), 0)::float8 AS total_distance,
			NULLIF(SUM(
				CASE
					WHEN (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN (drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency
					ELSE 0
				END
			), 0) AS total_energy_consumed,
			MIN(drives.start_date) AS first_drive_at,
			MAX(drives.end_date) AS last_drive_at
		FROM drives
		LEFT JOIN cars ON cars.id = drives.car_id
		WHERE drives.car_id = $1
			AND drives.end_date IS NOT NULL
			AND drives.start_date >= $2
			AND drives.end_date < $3
		GROUP BY local_date
		ORDER BY local_date ASC;`

	rows, err := db.Query(query, CarID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return DriveCalendarMonth{}, err
	}
	defer rows.Close()

	type dayAggregate struct {
		DriveCount          int
		TotalDurationMin    int
		TotalDistance       float64
		TotalEnergyConsumed *float64
		FirstDriveAt        *string
		LastDriveAt         *string
		AverageConsumption  *float64
	}

	aggregates := map[string]dayAggregate{}
	for rows.Next() {
		var (
			localDate           string
			aggregate           dayAggregate
			totalEnergyConsumed sql.NullFloat64
			firstDriveAt        sql.NullString
			lastDriveAt         sql.NullString
		)
		if err := rows.Scan(
			&localDate,
			&aggregate.DriveCount,
			&aggregate.TotalDurationMin,
			&aggregate.TotalDistance,
			&totalEnergyConsumed,
			&firstDriveAt,
			&lastDriveAt,
		); err != nil {
			return DriveCalendarMonth{}, err
		}

		energyPtr := floatPointer(totalEnergyConsumed)
		distKm := aggregate.TotalDistance
		if unitsLength == "mi" {
			aggregate.TotalDistance = kilometersToMiles(aggregate.TotalDistance)
		}
		aggregate.TotalEnergyConsumed = energyPtr
		aggregate.FirstDriveAt = timeZoneStringPointer(firstDriveAt)
		aggregate.LastDriveAt = timeZoneStringPointer(lastDriveAt)
		if energyPtr != nil && distKm > 0 && *energyPtr > 0 {
			v := *energyPtr / distKm * 1000.0
			if unitsLength == "mi" {
				v = whPerKmToWhPerMi(v)
			}
			aggregate.AverageConsumption = &v
		}
		aggregates[localDate] = aggregate
	}
	if err := rows.Err(); err != nil {
		return DriveCalendarMonth{}, err
	}

	nowLocal := time.Now().In(appUsersTimezone)
	days := make([]DriveCalendarDay, 0, endLocal.Day()-1)
	for current := startLocal; current.Before(endLocal); current = current.AddDate(0, 0, 1) {
		dateKey := current.Format("2006-01-02")
		aggregate, exists := aggregates[dateKey]
		day := DriveCalendarDay{
			Date:           current.Format("2006-01-02"),
			Day:            current.Day(),
			Weekday:        int(current.Weekday()),
			IsCurrentMonth: true,
			IsToday:        current.Year() == nowLocal.Year() && current.YearDay() == nowLocal.YearDay(),
		}
		if exists {
			day.DriveCount = aggregate.DriveCount
			day.DurationMin = aggregate.TotalDurationMin
			day.Distance = aggregate.TotalDistance
			day.EnergyConsumed = aggregate.TotalEnergyConsumed
			day.AverageConsumption = aggregate.AverageConsumption
			day.FirstDriveStart = aggregate.FirstDriveAt
			day.LastDriveEnd = aggregate.LastDriveAt
		}
		days = append(days, day)
	}

	return DriveCalendarMonth{
		Year:      year,
		Month:     month,
		MonthName: startLocal.Month().String(),
		StartDate: startLocal.Format(time.RFC3339),
		EndDate:   endLocal.Add(-time.Second).Format(time.RFC3339),
		Days:      days,
	}, nil
}
