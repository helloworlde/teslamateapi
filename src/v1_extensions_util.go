package main

import (
	"database/sql"
	"strings"
)

func toPercent(v *float64) *float64 {
	if v == nil {
		return nil
	}
	p := *v * 100
	return &p
}

func intOrNil(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return int(v.Int64)
}

func parseCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		v := strings.ToLower(strings.TrimSpace(p))
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func parseMetricSet(raw string) map[string]bool {
	set := map[string]bool{}
	for _, item := range parseCSV(raw) {
		set[item] = true
	}
	return set
}

func asFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case *float64:
		if t == nil {
			return 0, false
		}
		return *t, true
	case int:
		return float64(t), true
	case *int:
		if t == nil {
			return 0, false
		}
		return float64(*t), true
	default:
		return 0, false
	}
}
