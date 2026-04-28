package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type APIUnits struct {
	Distance    string `json:"distance,omitempty"`
	Speed       string `json:"speed,omitempty"`
	Energy      string `json:"energy,omitempty"`
	Consumption string `json:"consumption,omitempty"`
	Temperature string `json:"temperature,omitempty"`
	Duration    string `json:"duration,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Percentage  string `json:"percentage,omitempty"`
}

type APIRange struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type APIPagination struct {
	Page  int `json:"page"`
	Show  int `json:"show"`
	Total int `json:"total"`
}

type APIErrorDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type APIErrorResponse struct {
	Error APIErrorDetail `json:"error"`
}

type APIObjectResponse struct {
	CarID    int            `json:"car_id"`
	Timezone string         `json:"timezone"`
	Unit     APIUnits       `json:"unit,omitempty"`
	Range    APIRange       `json:"range,omitempty"`
	Data     map[string]any `json:"data"`
	Meta     map[string]any `json:"meta,omitempty"`
}

type APIListResponse struct {
	CarID      int            `json:"car_id"`
	Timezone   string         `json:"timezone"`
	Range      APIRange       `json:"range,omitempty"`
	Data       []any          `json:"data"`
	Pagination APIPagination  `json:"pagination"`
	Meta       map[string]any `json:"meta,omitempty"`
}

type APIChartPoint struct {
	Time  string   `json:"time,omitempty"`
	Label string   `json:"label,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type APIChartSeries struct {
	Name   string          `json:"name"`
	Unit   string          `json:"unit,omitempty"`
	Points []APIChartPoint `json:"points"`
}

type APIChartResponse struct {
	CarID    int              `json:"car_id"`
	Timezone string           `json:"timezone"`
	Range    APIRange         `json:"range,omitempty"`
	Bucket   string           `json:"bucket,omitempty"`
	Series   []APIChartSeries `json:"series"`
	Meta     map[string]any   `json:"meta,omitempty"`
}

func writeAPIError(c *gin.Context, status int, code, message string, details map[string]any) {
	details = responseErrorDetails(c, status, code, message, details)
	c.JSON(status, APIErrorResponse{Error: APIErrorDetail{Code: code, Message: message, Details: details}})
}

func buildAPIUnits(unitsLength, unitsTemperature string) APIUnits {
	distanceUnit := "km"
	speedUnit := "km/h"
	consumptionUnit := "Wh/km"
	if strings.EqualFold(unitsLength, "mi") {
		distanceUnit = "mi"
		speedUnit = "mi/h"
		consumptionUnit = "Wh/mi"
	}
	temperatureUnit := "C"
	if strings.EqualFold(unitsTemperature, "f") {
		temperatureUnit = "F"
	}
	return APIUnits{
		Distance:    distanceUnit,
		Speed:       speedUnit,
		Energy:      "kWh",
		Consumption: consumptionUnit,
		Temperature: temperatureUnit,
		Duration:    "minutes",
		Currency:    "currency",
		Percentage:  "%",
	}
}

func buildAPIRange(startUTC, endUTC string) APIRange {
	result := APIRange{}
	if startUTC != "" {
		result.Start = getTimeInTimeZone(startUTC)
	}
	if endUTC != "" {
		result.End = getTimeInTimeZone(endUTC)
	}
	return result
}

func newAPIObjectResponse(carID int, units APIUnits, startUTC, endUTC string, data map[string]any, meta map[string]any) APIObjectResponse {
	if data == nil {
		data = map[string]any{}
	}
	if meta == nil {
		meta = map[string]any{}
	}
	return APIObjectResponse{
		CarID:    carID,
		Timezone: appUsersTimezone.String(),
		Unit:     units,
		Range:    buildAPIRange(startUTC, endUTC),
		Data:     data,
		Meta:     meta,
	}
}

func newAPIChartResponse(carID int, startUTC, endUTC, bucket string, series []APIChartSeries, meta map[string]any) APIChartResponse {
	if series == nil {
		series = []APIChartSeries{}
	}
	if meta == nil {
		meta = map[string]any{}
	}
	return APIChartResponse{
		CarID:    carID,
		Timezone: appUsersTimezone.String(),
		Range:    buildAPIRange(startUTC, endUTC),
		Bucket:   bucket,
		Series:   series,
		Meta:     meta,
	}
}

func newAPIListResponse(carID int, startUTC, endUTC string, items []any, page, show, total int, meta map[string]any) APIListResponse {
	if items == nil {
		items = []any{}
	}
	if meta == nil {
		meta = map[string]any{}
	}
	return APIListResponse{
		CarID:    carID,
		Timezone: appUsersTimezone.String(),
		Range:    buildAPIRange(startUTC, endUTC),
		Data:     items,
		Pagination: APIPagination{
			Page:  page,
			Show:  show,
			Total: total,
		},
		Meta: meta,
	}
}

func writeAPISuccess(c *gin.Context, payload any) {
	c.JSON(http.StatusOK, payload)
}
