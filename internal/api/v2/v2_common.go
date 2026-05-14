package v2

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
	"github.com/tobiasehlert/teslamateapi/internal/config"
)

func parseV2AnalyticsQuery(c *gin.Context, now time.Time) (V2AnalyticsQuery, V2TimeRange, error) {
	query := V2AnalyticsQuery{
		Period:   strings.ToLower(v2QueryDefault(c, "period", v2DefaultPeriod)),
		Start:    strings.TrimSpace(c.Query("start")),
		End:      strings.TrimSpace(c.Query("end")),
		Timezone: v2QueryDefault(c, "timezone", defaultV2Timezone()),
		Compare:  strings.ToLower(v2QueryDefault(c, "compare", v2DefaultCompare)),
		GroupBy:  strings.TrimSpace(c.Query("group_by")),
		Metrics:  strings.TrimSpace(c.Query("metrics")),
		Include:  strings.TrimSpace(c.Query("include")),
	}

	if !v2AllowedPeriods[query.Period] {
		return query, V2TimeRange{}, fmt.Errorf("invalid period %q", query.Period)
	}
	if !v2AllowedCompares[query.Compare] {
		return query, V2TimeRange{}, fmt.Errorf("invalid compare %q", query.Compare)
	}

	location, err := time.LoadLocation(query.Timezone)
	if err != nil {
		return query, V2TimeRange{}, fmt.Errorf("invalid timezone %q", query.Timezone)
	}

	start, end, err := resolveV2Range(query, location, now)
	if err != nil {
		return query, V2TimeRange{}, err
	}

	result := V2TimeRange{
		Period:         query.Period,
		Timezone:       query.Timezone,
		Compare:        query.Compare,
		Start:          start.UTC(),
		End:            end.UTC(),
		AppliedFilters: appliedAnalyticsFilters(query),
	}

	if query.Compare == "previous_period" {
		duration := result.End.Sub(result.Start)
		previousEnd := result.Start
		previousStart := previousEnd.Add(-duration)
		result.PreviousStart = &previousStart
		result.PreviousEnd = &previousEnd
	}

	return query, result, nil
}

func v2QueryDefault(c *gin.Context, key string, fallback string) string {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return fallback
	}
	return value
}

func defaultV2Timezone() string {
	return defaultV2Location().String()
}

func defaultV2Location() *time.Location {
	if apicommon.AppUsersTimezone != nil {
		return apicommon.AppUsersTimezone
	}
	location, err := time.LoadLocation(config.Env("TZ", "Europe/Berlin"))
	if err != nil {
		return time.UTC
	}
	return location
}

func resolveV2Range(query V2AnalyticsQuery, location *time.Location, now time.Time) (time.Time, time.Time, error) {
	now = now.In(location)
	if !now.After(time.Time{}) {
		now = time.Now().In(location)
	}

	var start time.Time
	var end time.Time
	var err error

	switch query.Period {
	case "day":
		start = beginningOfDay(now)
		end = start.AddDate(0, 0, 1)
	case "week":
		weekdayOffset := (int(now.Weekday()) + 6) % 7
		start = beginningOfDay(now).AddDate(0, 0, -weekdayOffset)
		end = start.AddDate(0, 0, 7)
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
		end = start.AddDate(0, 1, 0)
	case "quarter":
		quarterMonth := time.Month(((int(now.Month())-1)/3)*3 + 1)
		start = time.Date(now.Year(), quarterMonth, 1, 0, 0, 0, 0, location)
		end = start.AddDate(0, 3, 0)
	case "year":
		start = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, location)
		end = start.AddDate(1, 0, 0)
	case "custom":
		if query.Start == "" || query.End == "" {
			return time.Time{}, time.Time{}, fmt.Errorf("custom period requires both start and end")
		}
	}

	if query.Start != "" {
		start, err = parseV2ClientTime(query.Start, location)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start: %w", err)
		}
	}
	if query.End != "" {
		end, err = parseV2ClientTime(query.End, location)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end: %w", err)
		}
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end must be after start")
	}

	return start, end, nil
}

func beginningOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func parseV2ClientTime(value string, location *time.Location) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", value, location); err == nil {
		return t, nil
	}
	normalized := strings.ReplaceAll(value, "T", " ")
	return time.ParseInLocation(time.DateTime, normalized, location)
}

func newV2Meta(carID int64, timeRange V2TimeRange) V2Meta {
	location := timeRangeLocation(timeRange)
	return V2Meta{
		CarID:          carID,
		Period:         timeRange.Period,
		Timezone:       timeRange.Timezone,
		Start:          timeRange.Start.In(location).Format(time.RFC3339),
		End:            timeRange.End.In(location).Format(time.RFC3339),
		Compare:        timeRange.Compare,
		Unit:           defaultV2Unit(),
		GeneratedAt:    time.Now().In(location).Format(time.RFC3339),
		AppliedFilters: timeRange.AppliedFilters,
	}
}

// appliedAnalyticsFilters projects the resolved analytics query into a
// normalized map for meta.applied_filters (spec §5.2). Empty keys are dropped
// so clients see exactly the filters the server honored — including any
// values it defaulted on their behalf.
func appliedAnalyticsFilters(query V2AnalyticsQuery) map[string]string {
	out := map[string]string{}
	addIf := func(k, v string) {
		if v = strings.TrimSpace(v); v != "" {
			out[k] = v
		}
	}
	addIf("period", query.Period)
	addIf("timezone", query.Timezone)
	addIf("compare", query.Compare)
	addIf("group_by", query.GroupBy)
	addIf("metrics", query.Metrics)
	addIf("include", query.Include)
	addIf("start", query.Start)
	addIf("end", query.End)
	if len(out) == 0 {
		return nil
	}
	return out
}

func timeRangeLocation(timeRange V2TimeRange) *time.Location {
	if timeRange.Timezone == "" {
		return defaultV2Location()
	}
	location, err := time.LoadLocation(timeRange.Timezone)
	if err != nil {
		return defaultV2Location()
	}
	return location
}

func defaultV2Unit() V2Unit {
	return V2Unit{
		Distance:    "km",
		Energy:      "kWh",
		Power:       "kW",
		Speed:       "km/h",
		Duration:    "seconds",
		Elevation:   "m",
		Consumption: "Wh/km",
		Temperature: "C",
		Currency:    defaultV2Currency(),
	}
}

// defaultV2Currency resolves the response currency from configuration. Order:
//  1. CURRENCY env var (operator override)
//  2. TeslaMate `settings.currency` (read once when the V2 routes register)
//  3. "USD" as a neutral fallback
//
// We do NOT hardcode CNY: every deployment lies otherwise.
func defaultV2Currency() string {
	if v := strings.TrimSpace(config.Env("CURRENCY", "")); v != "" {
		return strings.ToUpper(v)
	}
	if v := strings.TrimSpace(teslamateSettingsCurrency); v != "" {
		return strings.ToUpper(v)
	}
	return "USD"
}

// teslamateSettingsCurrency is populated at startup from the TeslaMate `settings`
// table (best-effort; remains empty if read fails or table is absent). Updated
// via SetTeslaMateSettingsCurrency from the routes registration.
var teslamateSettingsCurrency string

// SetTeslaMateSettingsCurrency lets the bootstrap code seed the cached currency.
func SetTeslaMateSettingsCurrency(value string) {
	teslamateSettingsCurrency = strings.TrimSpace(value)
}

// seedTeslaMateSettingsCurrency reads TeslaMate's settings.currency once at
// startup. Errors are intentionally swallowed: the table may not exist on
// minimal/embedded TeslaMate variants, and the v2 layer must still respond
// (defaulting to env / "USD"). This is the only place we touch that table.
func seedTeslaMateSettingsCurrency() {
	if apicommon.DB == nil {
		return
	}
	var currency string
	// LIMIT 1: the table is single-row in practice but we don't want to error
	// if a deployment has duplicates.
	err := apicommon.DB.QueryRow(`SELECT COALESCE(NULLIF(currency, ''), '') FROM settings LIMIT 1`).Scan(&currency)
	if err != nil {
		return
	}
	SetTeslaMateSettingsCurrency(currency)
}

func v2JSON(c *gin.Context, status int, data interface{}, meta V2Meta) {
	c.JSON(status, V2APIResponse{Data: data, Meta: meta})
}

// setV2DeprecationHeaders attaches IETF-style deprecation headers to the
// response. Callers pass the successor path (relative to the API root); we
// emit:
//   - `Deprecation: true`        — RFC 8594 marker
//   - `Link: <successor>; rel="successor-version"` — points clients at the
//     replacement so they don't have to read swagger to find it
//
// Used by Stage C (audit §1.3 / §1.2): /v2/analytics/battery → v1
// battery-health, /v2/analytics/battery/timeseries → /v2/timeline.
func setV2DeprecationHeaders(c *gin.Context, successorPath string) {
	c.Header("Deprecation", "true")
	if successorPath != "" {
		c.Header("Link", `<`+successorPath+`>; rel="successor-version"`)
	}
}

// parseV2IncludeSet splits the include query parameter into a lower-cased set.
// Empty input returns an empty set; callers decide which keys are required.
func parseV2IncludeSet(raw string) map[string]bool {
	out := map[string]bool{}
	if raw == "" {
		return out
	}
	for _, part := range strings.Split(raw, ",") {
		key := strings.ToLower(strings.TrimSpace(part))
		if key != "" {
			out[key] = true
		}
	}
	return out
}

func float64Ptr(value float64) *float64 {
	return &value
}

func compareFloat(current float64, previous float64) V2ComparisonValue {
	comparison := V2ComparisonValue{
		Current:  float64Ptr(current),
		Previous: float64Ptr(previous),
	}
	delta := current - previous
	comparison.Delta = &delta
	if previous != 0 {
		deltaPercent := delta / previous * 100
		comparison.DeltaPercent = &deltaPercent
	}
	return comparison
}

func compareFloatPtr(current *float64, previous *float64) V2ComparisonValue {
	c, p := 0.0, 0.0
	if current != nil {
		c = *current
	}
	if previous != nil {
		p = *previous
	}
	return compareFloat(c, p)
}
