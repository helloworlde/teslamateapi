package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

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
	writeV1List(c, out, v1Pagination{Limit: limit, Offset: offset, Total: total}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"))
}

func TeslaMateAPICarsMapVisitedUnifiedV2(c *gin.Context) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid visited map range", map[string]any{"reason": err.Error()})
		return
	}
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
	type visitedCell struct {
		Lat, Lng float64
		Count    int
	}
	cells := make(map[string]*visitedCell, len(points))
	for _, p := range points {
		key := fmt.Sprintf("%.4f,%.4f", p.Latitude, p.Longitude)
		if c := cells[key]; c != nil {
			c.Count++
		} else {
			cells[key] = &visitedCell{Lat: p.Latitude, Lng: p.Longitude, Count: 1}
		}
	}
	visitedList := make([]visitedCell, 0, len(cells))
	for _, c := range cells {
		visitedList = append(visitedList, *c)
	}
	sort.SliceStable(visitedList, func(i, j int) bool {
		return visitedList[i].Count > visitedList[j].Count
	})
	visitedPoints := make([]any, 0, len(visitedList))
	for _, v := range visitedList {
		visitedPoints = append(visitedPoints, map[string]any{"latitude": v.Lat, "longitude": v.Lng, "count": v.Count})
	}
	data := map[string]any{
		"car_id":         ctx.CarID,
		"range":          buildRangeDTO(dr),
		"distance_km":    nil,
		"drive_count":    nil,
		"visited_points": visitedPoints,
		"heatmap":        []any{},
		"truncated":      truncated,
	}
	if bounds != nil {
		data["bounds"] = map[string]any{
			"north": bounds.MaxLatitude,
			"south": bounds.MinLatitude,
			"east":  bounds.MaxLongitude,
			"west":  bounds.MinLongitude,
		}
	}
	writeV1Object(c, data, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"))
}

type visitedBounds struct {
	MinLatitude  float64 `json:"min_latitude"`
	MaxLatitude  float64 `json:"max_latitude"`
	MinLongitude float64 `json:"min_longitude"`
	MaxLongitude float64 `json:"max_longitude"`
}

type visitedPoint struct {
	Time      string  `json:"time"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func fetchTimelineEvents(carID int, startUTC, endUTC string, page, show int, order string) ([]ActivityTimelineEvent, int, error) {
	baseQuery, params := buildStateTimelineBaseQuery(carID, startUTC, endUTC)
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	var total int
	if err := db.QueryRowContext(queryCtx, baseQuery+` SELECT COUNT(*)::int FROM timeline;`, params...).Scan(&total); err != nil {
		return nil, 0, err
	}
	items, err := fetchStateTimeline(carID, startUTC, endUTC, page, show)
	if err != nil {
		return nil, 0, err
	}
	events := mapStateTimelineToActivityEvents(items)
	if order == "asc" {
		sort.SliceStable(events, func(i, j int) bool { return events[i].StartDate < events[j].StartDate })
	}
	return events, total, nil
}

func fetchVisitedMap(carID int, startUTC, endUTC string, limit int) ([]visitedPoint, *visitedBounds, bool, error) {
	query := `
		SELECT positions.date, positions.latitude, positions.longitude
		FROM positions
		INNER JOIN drives ON drives.id = positions.drive_id
		WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
			AND positions.latitude IS NOT NULL AND positions.longitude IS NOT NULL
		ORDER BY positions.date DESC
		LIMIT $4`
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, carID, startUTC, endUTC, limit+1)
	if err != nil {
		return nil, nil, false, err
	}
	defer rows.Close()
	points := make([]visitedPoint, 0, limit+1)
	bounds := &visitedBounds{MinLatitude: 999, MaxLatitude: -999, MinLongitude: 999, MaxLongitude: -999}
	for rows.Next() {
		var date string
		var p visitedPoint
		if err := rows.Scan(&date, &p.Latitude, &p.Longitude); err != nil {
			return nil, nil, false, err
		}
		p.Time = getTimeInTimeZone(date)
		points = append(points, p)
		if p.Latitude < bounds.MinLatitude {
			bounds.MinLatitude = p.Latitude
		}
		if p.Latitude > bounds.MaxLatitude {
			bounds.MaxLatitude = p.Latitude
		}
		if p.Longitude < bounds.MinLongitude {
			bounds.MinLongitude = p.Longitude
		}
		if p.Longitude > bounds.MaxLongitude {
			bounds.MaxLongitude = p.Longitude
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, false, err
	}
	truncated := len(points) > limit
	if truncated {
		points = points[:limit]
	}
	if len(points) == 0 {
		return points, nil, false, nil
	}
	return points, bounds, truncated, nil
}
