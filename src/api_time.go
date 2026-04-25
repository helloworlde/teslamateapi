package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	apiLocalDateTimeLayout = "2006-01-02 15:04:05"
	apiDateLayout          = "2006-01-02"
)

func parseAPITime(value string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}
	value = repairDateQueryParam(strings.TrimSpace(value))
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time value")
	}

	layouts := []string{time.RFC3339Nano, time.RFC3339}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}

	localCandidate := strings.Replace(value, "T", " ", 1)
	if t, err := time.ParseInLocation(apiLocalDateTimeLayout, localCandidate, loc); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation(apiDateLayout, value, loc); err == nil {
		return t, nil
	}

	sanitizedInput := strings.NewReplacer("\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(value)
	return time.Time{}, fmt.Errorf("invalid date format: %q (expected RFC3339, RFC3339 with timezone offset, local datetime YYYY-MM-DD HH:mm:ss, or date YYYY-MM-DD; encode + as %%2B in query strings)", sanitizedInput)
}

func parseOptionalAPITime(value string, loc *time.Location) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	t, err := parseAPITime(value, loc)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func isDateOnlyValue(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	value = repairDateQueryParam(value)
	return len(value) == len(apiDateLayout) && strings.Count(value, "-") == 2 && !strings.ContainsAny(value, "T :")
}
