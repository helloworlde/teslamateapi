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
