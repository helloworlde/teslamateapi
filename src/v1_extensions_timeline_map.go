package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func TeslaMateAPICarsUnifiedTimelineV2(c *gin.Context) {
	offset, limit, err := parseOffsetLimit(c, 50, 200)
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_pagination", err.Error(), nil)
		return
	}
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid timeline range", map[string]any{"reason": err.Error()})
		return
	}
	warnings := []any{}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsUnifiedTimelineV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	page := offset/limit + 1
	items, total, err := fetchTimelineEvents(ctx.CarID, startUTC, endUTC, page, limit, "desc")
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load timeline", map[string]any{"reason": err.Error()})
		return
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		entityID := 0
		if id, err := strconv.Atoi(item.SourceID); err == nil {
			entityID = id
		}
		out = append(out, map[string]any{
			"id":          item.ID,
			"type":        item.Type,
			"start_date":  item.StartDate,
			"end_date":    item.EndDate,
			"title":       item.Title,
			"summary":     item.Metrics,
			"entity_type": item.Type,
			"entity_id":   entityID,
		})
	}
	writeV1List(c, out, v1Pagination{Limit: limit, Offset: offset, Total: total}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func TeslaMateAPICarsMapVisitedUnifiedV2(c *gin.Context) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid visited map range", map[string]any{"reason": err.Error()})
		return
	}
	warnings := []any{}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsMapVisitedUnifiedV2")
	if !ok {
		return
	}
	startUTC, endUTC := dbTimeRange(dr)
	points, bounds, truncated, err := fetchVisitedMap(ctx.CarID, startUTC, endUTC, 10000)
	if err != nil {
		writeV1Error(c, http.StatusInternalServerError, "query_error", "unable to load visited map", map[string]any{"reason": err.Error()})
		return
	}
	visited := map[string]int{}
	for _, p := range points {
		key := fmt.Sprintf("%.4f,%.4f", p.Latitude, p.Longitude)
		visited[key]++
	}
	visitedPoints := make([]any, 0, len(visited))
	for key, count := range visited {
		parts := strings.Split(key, ",")
		lat, _ := strconv.ParseFloat(parts[0], 64)
		lng, _ := strconv.ParseFloat(parts[1], 64)
		visitedPoints = append(visitedPoints, map[string]any{"latitude": lat, "longitude": lng, "count": count})
	}
	sort.SliceStable(visitedPoints, func(i, j int) bool {
		return visitedPoints[i].(map[string]any)["count"].(int) > visitedPoints[j].(map[string]any)["count"].(int)
	})
	data := map[string]any{
		"car_id":         ctx.CarID,
		"range":          buildRangeDTO(dr),
		"distance_km":    nil,
		"drive_count":    nil,
		"visited_points": visitedPoints,
		"heatmap":        []any{},
	}
	if bounds != nil {
		data["bounds"] = map[string]any{
			"north": bounds.MaxLatitude,
			"south": bounds.MinLatitude,
			"east":  bounds.MaxLongitude,
			"west":  bounds.MinLongitude,
		}
	}
	if truncated {
		warnings = append(warnings, map[string]any{"code": "data_truncated", "message": "visited points were truncated to limit 10000"})
	}
	writeV1Object(c, data, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}
