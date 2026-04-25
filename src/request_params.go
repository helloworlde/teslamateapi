package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	dateTimeSpaceOffsetRe    = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\s+([+-]?\d{2}:\d{2})$`)
	dateTimeSpaceOffsetNoTRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\s+([+-]?\d{2}:\d{2})$`)
)

func repairDateQueryParam(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if m := dateTimeSpaceOffsetRe.FindStringSubmatch(raw); len(m) == 3 {
		off := strings.TrimSpace(m[2])
		if !strings.HasPrefix(off, "-") && !strings.HasPrefix(off, "+") {
			off = "+" + off
		}
		return m[1] + off
	}
	if m := dateTimeSpaceOffsetNoTRe.FindStringSubmatch(raw); len(m) == 3 {
		off := strings.TrimSpace(m[2])
		if !strings.HasPrefix(off, "-") && !strings.HasPrefix(off, "+") {
			off = "+" + off
		}
		return strings.Replace(m[1], " ", "T", 1) + off
	}
	return raw
}

func parseCarID(c *gin.Context) (int, error) {
	raw := strings.TrimSpace(c.Param("CarID"))
	if raw == "" {
		return 0, fmt.Errorf("CarID is required")
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("CarID must be a positive integer")
	}
	return id, nil
}

func parseChargeID(c *gin.Context) (int, error) {
	raw := strings.TrimSpace(c.Param("ChargeID"))
	if raw == "" {
		return 0, fmt.Errorf("ChargeID is required")
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("ChargeID must be a positive integer")
	}
	return id, nil
}

func parsePaginationParams(c *gin.Context, defaultPage, defaultShow, maxShow int) (page int, show int, err error) {
	page, err = parsePositiveIntQuery(c.DefaultQuery("page", strconv.Itoa(defaultPage)), defaultPage, 1, 100000)
	if err != nil {
		return 0, 0, fmt.Errorf("page: %w", err)
	}
	show, err = parsePositiveIntQuery(c.DefaultQuery("show", strconv.Itoa(defaultShow)), defaultShow, 1, maxShow)
	if err != nil {
		return 0, 0, fmt.Errorf("show: %w", err)
	}
	return page, show, nil
}

func parsePositiveIntQuery(raw string, defaultValue, minValue, maxValue int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("must be an integer")
	}
	if v < minValue {
		v = minValue
	}
	if v > maxValue {
		v = maxValue
	}
	return v, nil
}

func parseSummaryDateRange(c *gin.Context) (string, string, error) {
	return parseDateRangeValues(c.Query("startDate"), c.Query("endDate"), appUsersTimezone)
}

func parseDateRangeValues(startRaw string, endRaw string, loc *time.Location) (string, string, error) {
	startUTC, err := parseDateParam(startRaw)
	if err != nil {
		return "", "", err
	}
	endUTC, err := parseDateParam(endRaw)
	if err != nil {
		return "", "", err
	}
	if isDateOnlyValue(endRaw) && endUTC != "" {
		endTime, err := parseAPITime(endRaw, loc)
		if err != nil {
			return "", "", err
		}
		endUTC = endTime.Add(24*time.Hour - time.Second).UTC().Format(dbTimestampFormat)
	}
	return startUTC, endUTC, nil
}

func parseSortOrder(c *gin.Context, allowed map[string]string, defaultSort string) (string, string, error) {
	sortValue := strings.TrimSpace(c.DefaultQuery("sort", defaultSort))
	orderValue := strings.ToLower(strings.TrimSpace(c.DefaultQuery("order", "desc")))
	if orderValue == "" {
		orderValue = "desc"
	}
	if orderValue != "asc" && orderValue != "desc" {
		return "", "", fmt.Errorf("order must be asc or desc")
	}
	column, ok := allowed[sortValue]
	if !ok {
		return "", "", fmt.Errorf("unsupported sort %q", sortValue)
	}
	return column, orderValue, nil
}

func parseBucketParam(c *gin.Context, defaultBucket string) (string, error) {
	bucket := strings.ToLower(strings.TrimSpace(c.DefaultQuery("bucket", defaultBucket)))
	switch bucket {
	case "day", "week", "month", "year":
		return bucket, nil
	default:
		return "", fmt.Errorf("bucket must be one of day, week, month, year")
	}
}

func parseSeriesLimitParam(c *gin.Context, key string, defaultValue, maxValue int) (int, error) {
	return parsePositiveIntQuery(c.DefaultQuery(key, strconv.Itoa(defaultValue)), defaultValue, 1, maxValue)
}
