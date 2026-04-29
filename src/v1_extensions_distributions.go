package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func TeslaMateAPICarsDriveDistributionsV2(c *gin.Context) {
	writeScopedDistributions(c, "drives", []string{"start_hour", "weekday", "distance", "duration", "speed", "efficiency"})
}

func TeslaMateAPICarsChargeDistributionsV2(c *gin.Context) {
	writeScopedDistributions(c, "charges", []string{"start_hour", "weekday", "energy", "duration", "power", "cost"})
}

func writeScopedDistributions(c *gin.Context, scope string, defaultMetrics []string) {
	dr, err := parseDateRangeStrictOrDefault(c, "month")
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_date_range", "invalid distribution range", map[string]any{"reason": err.Error()})
		return
	}
	warnings := []any{}
	ctx, ok := loadAPICarContext(c, "TeslaMateAPICarsDistributionsV2")
	if !ok {
		return
	}
	metrics := parseCSV(c.Query("metrics"))
	if len(metrics) == 0 {
		metrics = defaultMetrics
	}
	startUTC, endUTC := dbTimeRange(dr)
	distributions := make([]any, 0, len(metrics))
	for _, metric := range metrics {
		item, err := fetchDistribution(ctx.CarID, scope, metric, startUTC, endUTC)
		if err != nil {
			warnings = append(warnings, nonFatalWarning("distribution_query_failed", "failed to load distribution", map[string]any{"metric": metric}, err))
			continue
		}
		distributions = append(distributions, item)
	}
	writeV1Object(c, map[string]any{
		"car_id":        ctx.CarID,
		"scope":         scope,
		"range":         buildRangeDTO(dr),
		"distributions": distributions,
	}, buildV1Meta(ctx.CarID, dr.Timezone.String(), "metric"), warnings)
}

func fetchDashboardDistributions(carID int, startUTC, endUTC string) ([]any, []any) {
	specs := []struct {
		Scope  string
		Metric string
	}{
		{Scope: "drives", Metric: "start_hour"},
		{Scope: "drives", Metric: "distance"},
		{Scope: "charges", Metric: "start_hour"},
		{Scope: "charges", Metric: "energy"},
	}
	distributions := make([]any, 0, len(specs))
	warnings := make([]any, 0)
	for _, spec := range specs {
		item, err := fetchDistribution(carID, spec.Scope, spec.Metric, startUTC, endUTC)
		if err != nil {
			warnings = append(warnings, nonFatalWarning("dashboard_distribution_unavailable", "failed to load dashboard distribution", map[string]any{"scope": spec.Scope, "metric": spec.Metric}, err))
			continue
		}
		distributions = append(distributions, item)
	}
	return distributions, warnings
}

func fetchDistribution(carID int, scope, metric, startUTC, endUTC string) (map[string]any, error) {
	key := aggregateCacheKey("distribution", carID, scope, metric, startUTC, endUTC, appUsersTimezone.String())
	return cachedValue(key, aggregateCacheTTL(endUTC), func() (map[string]any, error) {
		return fetchDistributionUncached(carID, scope, metric, startUTC, endUTC)
	})
}

func fetchDistributionUncached(carID int, scope, metric, startUTC, endUTC string) (map[string]any, error) {
	switch scope + ":" + metric {
	case "drives:start_hour":
		return fetchHourDistribution("drives", "drives.start_date", carID, startUTC, endUTC, "drive_start_hour")
	case "drives:weekday":
		return fetchWeekdayDistribution("drives", "drives.start_date", carID, startUTC, endUTC, "drive_weekday")
	case "drives:distance":
		return fetchNumericDistribution(`
			WITH buckets(ord, label, min_value, max_value) AS (
				VALUES (1, '0-5', 0.0, 5.0), (2, '5-10', 5.0, 10.0), (3, '10-20', 10.0, 20.0),
					(4, '20-50', 20.0, 50.0), (5, '50+', 50.0, NULL)
			),
			filtered AS (
				SELECT GREATEST(COALESCE(distance, 0), 0)::float8 AS value
				FROM drives
				WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
			)
			SELECT b.label, COALESCE(COUNT(f.value), 0)::int
			FROM buckets b
			LEFT JOIN filtered f ON f.value >= b.min_value AND (b.max_value IS NULL OR f.value < b.max_value)
			GROUP BY b.ord, b.label
			ORDER BY b.ord`, carID, startUTC, endUTC, "drive_distance", "drive_distance", "count")
	case "drives:duration":
		return fetchNumericDistribution(`
			WITH buckets(ord, label, min_value, max_value) AS (
				VALUES (1, '0-10', 0.0, 10.0), (2, '10-20', 10.0, 20.0), (3, '20-40', 20.0, 40.0),
					(4, '40-60', 40.0, 60.0), (5, '60+', 60.0, NULL)
			),
			filtered AS (
				SELECT GREATEST(COALESCE(duration_min, 0), 0)::float8 AS value
				FROM drives
				WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
			)
			SELECT b.label, COALESCE(COUNT(f.value), 0)::int
			FROM buckets b
			LEFT JOIN filtered f ON f.value >= b.min_value AND (b.max_value IS NULL OR f.value < b.max_value)
			GROUP BY b.ord, b.label
			ORDER BY b.ord`, carID, startUTC, endUTC, "drive_duration", "drive_duration", "count")
	case "drives:speed":
		return fetchNumericDistribution(`
			WITH buckets(ord, label, min_value, max_value) AS (
				VALUES (1, '0-20', 0.0, 20.0), (2, '20-40', 20.0, 40.0), (3, '40-60', 40.0, 60.0),
					(4, '60-100', 60.0, 100.0), (5, '100+', 100.0, NULL)
			),
			filtered AS (
				SELECT COALESCE(speed_max, 0)::float8 AS value
				FROM drives
				WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
			)
			SELECT b.label, COALESCE(COUNT(f.value), 0)::int
			FROM buckets b
			LEFT JOIN filtered f ON f.value >= b.min_value AND (b.max_value IS NULL OR f.value < b.max_value)
			GROUP BY b.ord, b.label
			ORDER BY b.ord`, carID, startUTC, endUTC, "drive_speed", "drive_speed", "count")
	case "drives:efficiency":
		return fetchNumericDistribution(`
			WITH buckets(ord, label, min_value, max_value) AS (
				VALUES (1, '0-120', 0.0, 120.0), (2, '120-160', 120.0, 160.0), (3, '160-200', 160.0, 200.0),
					(4, '200-260', 200.0, 260.0), (5, '260+', 260.0, NULL)
			),
			filtered AS (
				SELECT CASE
					WHEN drives.distance > 0 AND (drives.start_rated_range_km - drives.end_rated_range_km) > 0
					THEN ((drives.start_rated_range_km - drives.end_rated_range_km) * cars.efficiency / drives.distance) * 1000.0
					ELSE NULL
				END::float8 AS value
				FROM drives
				LEFT JOIN cars ON cars.id = drives.car_id
				WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL AND drives.start_date >= $2 AND drives.end_date < $3
			)
			SELECT b.label, COALESCE(COUNT(f.value), 0)::int
			FROM buckets b
			LEFT JOIN filtered f ON f.value >= b.min_value AND (b.max_value IS NULL OR f.value < b.max_value)
			GROUP BY b.ord, b.label
			ORDER BY b.ord`, carID, startUTC, endUTC, "drive_efficiency", "drive_efficiency", "count")
	case "charges:start_hour":
		return fetchHourDistribution("charging_processes", "charging_processes.start_date", carID, startUTC, endUTC, "charge_start_hour")
	case "charges:weekday":
		return fetchWeekdayDistribution("charging_processes", "charging_processes.start_date", carID, startUTC, endUTC, "charge_weekday")
	case "charges:energy":
		return fetchNumericDistribution(`
			WITH buckets(ord, label, min_value, max_value) AS (
				VALUES (1, '0-10', 0.0, 10.0), (2, '10-20', 10.0, 20.0), (3, '20-40', 20.0, 40.0),
					(4, '40-70', 40.0, 70.0), (5, '70+', 70.0, NULL)
			),
			filtered AS (
				SELECT GREATEST(COALESCE(charge_energy_added, 0), 0)::float8 AS value
				FROM charging_processes
				WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
			)
			SELECT b.label, COALESCE(COUNT(f.value), 0)::int
			FROM buckets b
			LEFT JOIN filtered f ON f.value >= b.min_value AND (b.max_value IS NULL OR f.value < b.max_value)
			GROUP BY b.ord, b.label
			ORDER BY b.ord`, carID, startUTC, endUTC, "charge_energy", "charge_energy", "count")
	case "charges:duration":
		return fetchNumericDistribution(`
			WITH buckets(ord, label, min_value, max_value) AS (
				VALUES (1, '0-30', 0.0, 30.0), (2, '30-60', 30.0, 60.0), (3, '60-120', 60.0, 120.0),
					(4, '120-240', 120.0, 240.0), (5, '240+', 240.0, NULL)
			),
			filtered AS (
				SELECT GREATEST(COALESCE(duration_min, 0), 0)::float8 AS value
				FROM charging_processes
				WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
			)
			SELECT b.label, COALESCE(COUNT(f.value), 0)::int
			FROM buckets b
			LEFT JOIN filtered f ON f.value >= b.min_value AND (b.max_value IS NULL OR f.value < b.max_value)
			GROUP BY b.ord, b.label
			ORDER BY b.ord`, carID, startUTC, endUTC, "charge_duration", "charge_duration", "count")
	case "charges:power":
		return fetchNumericDistribution(`
			WITH buckets(ord, label, min_value, max_value) AS (
				VALUES (1, '0-4', 0.0, 4.0), (2, '4-8', 4.0, 8.0), (3, '8-12', 8.0, 12.0),
					(4, '12-50', 12.0, 50.0), (5, '50+', 50.0, NULL)
			),
			filtered AS (
				SELECT MAX(COALESCE(charges.charger_power, 0))::float8 AS value
				FROM charges
				INNER JOIN charging_processes ON charging_processes.id = charges.charging_process_id
				WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL
					AND charging_processes.start_date >= $2 AND charging_processes.end_date < $3
				GROUP BY charging_processes.id
			)
			SELECT b.label, COALESCE(COUNT(f.value), 0)::int
			FROM buckets b
			LEFT JOIN filtered f ON f.value >= b.min_value AND (b.max_value IS NULL OR f.value < b.max_value)
			GROUP BY b.ord, b.label
			ORDER BY b.ord`, carID, startUTC, endUTC, "charge_power", "charge_power", "count")
	case "charges:cost":
		return fetchNumericDistribution(`
			WITH buckets(ord, label, min_value, max_value) AS (
				VALUES (1, '0-5', 0.0, 5.0), (2, '5-10', 5.0, 10.0), (3, '10-20', 10.0, 20.0),
					(4, '20-50', 20.0, 50.0), (5, '50+', 50.0, NULL)
			),
			filtered AS (
				SELECT cost::float8 AS value
				FROM charging_processes
				WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3 AND cost IS NOT NULL AND cost >= 0
			)
			SELECT b.label, COALESCE(COUNT(f.value), 0)::int
			FROM buckets b
			LEFT JOIN filtered f ON f.value >= b.min_value AND (b.max_value IS NULL OR f.value < b.max_value)
			GROUP BY b.ord, b.label
			ORDER BY b.ord`, carID, startUTC, endUTC, "charge_cost", "charge_cost", "count")
	default:
		return nil, fmt.Errorf("unsupported %s distribution metric: %s", scope, metric)
	}
}

func fetchHourDistribution(table, dateExpr string, carID int, startUTC, endUTC, name string) (map[string]any, error) {
	query := fmt.Sprintf(`
		WITH buckets AS (
			SELECT generate_series(0, 23)::int AS hour
		),
		filtered AS (
			SELECT EXTRACT(HOUR FROM timezone($4, %s))::int AS hour
			FROM %s
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
		)
		SELECT b.hour, COALESCE(COUNT(f.hour), 0)::int
		FROM buckets b
		LEFT JOIN filtered f ON f.hour = b.hour
		GROUP BY b.hour
		ORDER BY b.hour`, dateExpr, table)
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	buckets := make([]any, 0, 24)
	for rows.Next() {
		var hour, count int
		if err := rows.Scan(&hour, &count); err != nil {
			return nil, err
		}
		buckets = append(buckets, map[string]any{"label": fmt.Sprintf("%02d:00-%02d:00", hour, (hour+1)%24), "from": hour, "to": hour + 1, "count": count, "value": count})
	}
	return map[string]any{"metric": name, "name": name, "unit": "count", "chart_type": "bar", "buckets": buckets}, rows.Err()
}

func fetchWeekdayDistribution(table, dateExpr string, carID int, startUTC, endUTC, name string) (map[string]any, error) {
	query := fmt.Sprintf(`
		WITH buckets(ord, label) AS (
			VALUES (1, 'Mon'), (2, 'Tue'), (3, 'Wed'), (4, 'Thu'), (5, 'Fri'), (6, 'Sat'), (7, 'Sun')
		),
		filtered AS (
			SELECT EXTRACT(ISODOW FROM timezone($4, %s))::int AS ord
			FROM %s
			WHERE car_id = $1 AND end_date IS NOT NULL AND start_date >= $2 AND end_date < $3
		)
		SELECT b.label, COALESCE(COUNT(f.ord), 0)::int
		FROM buckets b
		LEFT JOIN filtered f ON f.ord = b.ord
		GROUP BY b.ord, b.label
		ORDER BY b.ord`, dateExpr, table)
	return fetchNumericDistributionWithTimezone(query, carID, startUTC, endUTC, name, name, "count")
}

func fetchNumericDistributionWithTimezone(query string, carID int, startUTC, endUTC, metric, name, unit string) (map[string]any, error) {
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, carID, startUTC, endUTC, appUsersTimezone.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDistributionRows(rows, metric, name, unit)
}

func fetchNumericDistribution(query string, carID int, startUTC, endUTC, metric, name, unit string) (map[string]any, error) {
	queryCtx, cancel := newAggregateQueryContext()
	defer cancel()
	rows, err := db.QueryContext(queryCtx, query, carID, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDistributionRows(rows, metric, name, unit)
}

func scanDistributionRows(rows *sql.Rows, metric, name, unit string) (map[string]any, error) {
	buckets := make([]any, 0)
	for rows.Next() {
		var label string
		var count int
		if err := rows.Scan(&label, &count); err != nil {
			return nil, err
		}
		buckets = append(buckets, map[string]any{
			"label": label,
			"count": count,
			"value": count,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{
		"metric":     metric,
		"name":       name,
		"unit":       unit,
		"chart_type": "bar",
		"buckets":    buckets,
	}, nil
}
