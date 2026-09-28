//go:build integration

package v2

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/tobiasehlert/teslamateapi/internal/database"
)

//go:embed testdata/stats_by_geofence.sql
var geofenceTestSchema string

// Only the isolated runner supplies this socket; no application database
// configuration is read. An explicitly requested integration run must not skip.
func openGeofenceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	socket := os.Getenv("TESLAMATEAPI_GEOFENCE_TEST_SOCKET")
	if !filepath.IsAbs(socket) || !strings.HasPrefix(filepath.Base(socket), "teslamateapi-geofence.") {
		t.Fatal("run make test-integration to create the dedicated test database")
	}
	connection := url.URL{
		Scheme: "postgres",
		User:   url.User("geofence_test"),
		Path:   "/teslamateapi_geofence_test",
		RawQuery: url.Values{
			"host": {socket}, "port": {"5432"}, "sslmode": {"disable"},
			"connect_timeout": {"5"}, "options": {"-c jit=off"},
		}.Encode(),
	}
	db, err := sql.Open("postgres", connection.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var name string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "teslamateapi_geofence_test" {
		t.Fatal("refusing to write fixtures outside the dedicated test database")
	}
	schema := fmt.Sprintf("geofence_test_%d", time.Now().UnixNano())
	execGeofenceFixture(t, db, "CREATE SCHEMA "+schema)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := db.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop owned test schema: %v", err)
		}
	})
	execGeofenceFixture(t, db, "SET search_path TO "+schema)
	execGeofenceFixture(t, db, geofenceTestSchema)
	execGeofenceFixture(t, db, `INSERT INTO settings (id) VALUES (1);
		INSERT INTO cars (id, name) VALUES (1, 'Geofence Test'), (2, 'Other Vehicle');
		INSERT INTO geofences (id, name) VALUES (1, 'Test Charger'), (2, 'Other Charger');`)
	return db
}

func execGeofenceFixture(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, query); err != nil {
		t.Fatal(err)
	}
}

type geofenceTestRow struct {
	ID              *int     `json:"geofence_id"`
	Name            string   `json:"geofence_name"`
	Count           int      `json:"charges_count"`
	Added           *float64 `json:"charges_energy_added_kwh"`
	Cost            *float64 `json:"charges_cost"`
	Used            *float64 `json:"charges_energy_used_kwh"`
	UsedCount       int      `json:"charges_energy_used_session_count"`
	Duration        *int64   `json:"charges_duration_min"`
	DurationCount   int      `json:"charges_duration_session_count"`
	UnitCost        *float64 `json:"charges_unit_cost_per_kwh"`
	UnitCostCount   int      `json:"charges_unit_cost_session_count"`
	UnitCostUsed    float64  `json:"charges_unit_cost_energy_used_kwh"`
	Efficiency      *float64 `json:"charges_efficiency_pct"`
	EfficiencyCount int      `json:"charges_efficiency_session_count"`
	EfficiencyUsed  float64  `json:"charges_efficiency_energy_used_kwh"`
	CostCount       int      `json:"charges_cost_session_count"`
	AddedCount      int      `json:"charges_energy_added_session_count"`
}

func requestGeofenceTest(t *testing.T, db *sql.DB, suffix string, status int) []geofenceTestRow {
	t.Helper()
	handler := New(Deps{DB: database.Wrap(db), TZ: time.UTC})
	router := gin.New()
	router.GET("/api/v2/cars/:CarID/stats/by-geofence", handler.StatsByGeofence)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/cars/"+suffix, nil)
	ctx, cancel := context.WithTimeout(request.Context(), 10*time.Second)
	defer cancel()
	router.ServeHTTP(recorder, request.WithContext(ctx))
	if recorder.Code != status {
		t.Fatalf("HTTP %d, want %d: %s", recorder.Code, status, recorder.Body.String())
	}
	if !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("unexpected content type %q", recorder.Header().Get("Content-Type"))
	}
	if status != http.StatusOK {
		return nil
	}
	var response struct {
		Data struct {
			Geofences []geofenceTestRow `json:"geofences"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.Data.Geofences
}

func TestStatsByGeofenceIntegrationMetrics(t *testing.T) {
	db := openGeofenceTestDB(t)
	gin.SetMode(gin.TestMode)
	f := func(v float64) *float64 { return &v }
	i := func(v int64) *int64 { return &v }
	cases := []struct {
		name       string
		values     string // recorded charger kWh, vehicle kWh, cost, minutes
		used       *float64
		duration   *int64
		unitCost   *float64
		efficiency *float64
	}{
		{"weighted totals", "(40, 30, 20, 60), (60, 60, 80, 90)", f(100), i(150), f(1), f(90)},
		{"decimal arithmetic", "(0.10, 0.09, 0.03, 1), (0.20, 0.18, 0.06, 2)", f(0.30), i(3), f(0.30), f(90)},
		{"stored decimal precision", "(0.105, 0.095, 0.035, 1)", f(0.11), i(1), f(4.0 / 11), f(10.0 / 11 * 100)},
		{"nan cost", "(40, 30, 'NaN'::numeric, 60)", f(40), i(60), nil, f(75)},
		{"free charging", "(100, 90, 0, 150)", f(100), i(150), f(0), f(90)},
		{"missing cost", "(40, 30, NULL, 60), (60, 60, 80, 90)", f(100), i(150), f(80.0 / 60), f(90)},
		{"all missing cost", "(100, 90, NULL, 150)", f(100), i(150), nil, f(90)},
		{"missing charger energy", "(NULL, 30, 20, 60), (60, 60, 80, 90)", f(60), i(150), f(80.0 / 60), f(100)},
		{"missing duration", "(40, 30, 20, NULL), (60, 60, 80, 90)", f(100), i(90), f(1), f(90)},
		{"missing vehicle energy", "(40, NULL, 20, 60), (60, 60, 80, 90)", f(100), i(150), f(1), f(100)},
		{"zero denominator", "(0, 0, 0, 0)", f(0), i(0), nil, nil},
		{"negative charger energy", "(-1, 0, 20, 60)", nil, i(60), nil, nil},
		{"negative cost", "(40, 30, -20, 60)", f(40), i(60), nil, f(75)},
		{"negative duration", "(40, 30, 20, -1)", f(40), nil, f(0.5), f(75)},
		{"invalid session efficiency", "(40, 45, 20, 60), (60, 45, 80, 90)", f(100), i(150), f(1), f(75)},
		{"nan charger energy", "('NaN'::numeric, 30, 20, 60)", nil, i(60), nil, nil},
		{"nan vehicle energy", "(40, 'NaN'::numeric, 20, 60)", f(40), i(60), f(0.5), nil},
		{"all missing", "(NULL::numeric, NULL::numeric, NULL::numeric, NULL::int)", nil, nil, nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			execGeofenceFixture(t, db, "DELETE FROM charging_processes")
			execGeofenceFixture(t, db, `INSERT INTO charging_processes
				(car_id, geofence_id, start_date, end_date, charge_energy_used, charge_energy_added, cost, duration_min)
				SELECT 1, 1, '2026-01-01'::timestamp, '2026-01-02'::timestamp, used::numeric, added::numeric, cost::numeric, duration::smallint
				FROM (VALUES `+tc.values+`) AS v(used, added, cost, duration)`)
			rows := requestGeofenceTest(t, db, "1/stats/by-geofence", http.StatusOK)
			if len(rows) != 1 {
				t.Fatalf("got %d locations, want 1", len(rows))
			}
			row := rows[0]
			for _, metric := range []struct {
				name      string
				got, want *float64
			}{
				{"used", row.Used, tc.used}, {"unit cost", row.UnitCost, tc.unitCost}, {"efficiency", row.Efficiency, tc.efficiency},
			} {
				if (metric.got == nil) != (metric.want == nil) {
					t.Errorf("%s nullability: got %v, want %v", metric.name, metric.got, metric.want)
				} else if metric.got != nil && (math.IsNaN(*metric.got) || math.Abs(*metric.got-*metric.want) > 1e-9) {
					t.Errorf("%s: got %g, want %g", metric.name, *metric.got, *metric.want)
				}
			}
			if (row.Duration == nil) != (tc.duration == nil) {
				t.Errorf("duration nullability: got %v, want %v", row.Duration, tc.duration)
			} else if row.Duration != nil && *row.Duration != *tc.duration {
				t.Errorf("duration: got %d, want %d", *row.Duration, *tc.duration)
			}
		})
	}
}

func TestStatsByGeofenceIntegrationPartialCoverage(t *testing.T) {
	db := openGeofenceTestDB(t)
	gin.SetMode(gin.TestMode)
	execGeofenceFixture(t, db, `INSERT INTO charging_processes
		(car_id, geofence_id, start_date, end_date, charge_energy_used, charge_energy_added, cost, duration_min) VALUES
		(1, 1, '2026-01-01', '2026-01-02', 40, 30, NULL, 60),
		(1, 1, '2026-01-03', '2026-01-04', 60, 70, 80, NULL),
		(1, 1, '2026-01-05', '2026-01-06', 100, 90, 50, 120);`)
	rows := requestGeofenceTest(t, db, "1/stats/by-geofence", http.StatusOK)
	if len(rows) != 1 {
		t.Fatalf("got %d locations, want 1", len(rows))
	}
	row := rows[0]
	if row.Count != 3 || row.Duration == nil || *row.Duration != 180 || row.DurationCount != 2 {
		t.Errorf("duration coverage: %+v", row)
	}
	if row.UnitCost == nil || math.Abs(*row.UnitCost-130.0/160) > 1e-9 ||
		row.UnitCostCount != 2 || math.Abs(row.UnitCostUsed-160) > 1e-9 {
		t.Errorf("unit cost coverage: %+v", row)
	}
	if row.Efficiency == nil || math.Abs(*row.Efficiency-120.0/140*100) > 1e-9 ||
		row.EfficiencyCount != 2 || math.Abs(row.EfficiencyUsed-140) > 1e-9 {
		t.Errorf("efficiency coverage: %+v", row)
	}
	if row.Cost == nil || *row.Cost != 130 || row.CostCount != 2 ||
		row.Added == nil || *row.Added != 190 || row.AddedCount != 3 ||
		row.Used == nil || *row.Used != 200 || row.UsedCount != 3 {
		t.Errorf("cost/energy coverage: %+v", row)
	}
}

func TestStatsByGeofenceIntegrationUnknownAndFreeCost(t *testing.T) {
	db := openGeofenceTestDB(t)
	gin.SetMode(gin.TestMode)
	execGeofenceFixture(t, db, `INSERT INTO charging_processes
		(car_id, geofence_id, start_date, end_date, charge_energy_used, charge_energy_added, cost, duration_min) VALUES
		(1, 1, '2026-01-01', '2026-01-02', 10, 9, NULL, 30),
		(1, 2, '2026-01-01', '2026-01-02', 10, 9, 0, 30);`)
	rows := requestGeofenceTest(t, db, "1/stats/by-geofence", http.StatusOK)
	if len(rows) != 2 {
		t.Fatalf("got %d locations, want 2", len(rows))
	}
	for _, row := range rows {
		if row.ID == nil {
			t.Fatalf("missing geofence id: %+v", row)
		}
		switch *row.ID {
		case 1:
			if row.Cost != nil || row.CostCount != 0 || row.UnitCost != nil {
				t.Errorf("unknown cost exposed as free: %+v", row)
			}
		case 2:
			if row.Cost == nil || *row.Cost != 0 || row.CostCount != 1 ||
				row.UnitCost == nil || *row.UnitCost != 0 {
				t.Errorf("free charging hidden as unknown: %+v", row)
			}
		default:
			t.Errorf("unexpected geofence: %+v", row)
		}
	}
}

func TestStatsByGeofenceIntegrationBoundaries(t *testing.T) {
	db := openGeofenceTestDB(t)
	gin.SetMode(gin.TestMode)
	execGeofenceFixture(t, db, `INSERT INTO charging_processes
		(car_id, geofence_id, start_date, end_date, charge_energy_used, charge_energy_added, cost, duration_min) VALUES
		(1, 1, '2026-01-01', '2026-01-02', 10, 9, 5, 20),
		(1, NULL, '2026-02-01', '2026-02-02', 40, 30, 20, 60),
		(1, 1, '2026-02-03', NULL, 1000, 900, 500, 600),
		(2, 2, '2026-02-01', '2026-02-02', 1000, 900, 500, 600);
		INSERT INTO drives (car_id, start_date, end_date, start_geofence_id, end_geofence_id)
		VALUES (1, '2026-02-01', '2026-02-02', 2, 2);`)
	rows := requestGeofenceTest(t, db, "1/stats/by-geofence?start_date=2026-02-01T00:00:00Z&end_date=2026-02-01T00:00:00Z", http.StatusOK)
	if len(rows) != 2 {
		t.Fatalf("got %d locations, want Other and drive-only location", len(rows))
	}
	for _, row := range rows {
		if row.ID == nil {
			if row.Name != "Other" || row.Count != 1 || row.Used == nil || *row.Used != 40 ||
				row.Cost == nil || *row.Cost != 20 || row.Added == nil || *row.Added != 30 {
				t.Errorf("incorrect Other/window aggregate: %+v", row)
			}
		} else if *row.ID != 2 || row.Count != 0 || row.Used != nil || row.Duration != nil ||
			row.Cost != nil || row.Added != nil || row.UnitCost != nil || row.Efficiency != nil {
			t.Errorf("incorrect no-charge aggregate: %+v", row)
		}
	}
	for _, suffix := range []string{"0/stats/by-geofence", "1/stats/by-geofence?start_date=invalid"} {
		requestGeofenceTest(t, db, suffix, http.StatusBadRequest)
	}
}

// Impossible stored values must be rejected by the real PostgreSQL types, not
// accepted by a wider float8/int fixture and then "handled" by the endpoint.
func TestStatsByGeofenceIntegrationStorageLimits(t *testing.T) {
	db := openGeofenceTestDB(t)
	for _, tc := range []struct{ name, values string }{
		{"infinite energy", "'Infinity'::numeric, 1, 1, 1"},
		{"infinite cost", "1, 1, 'Infinity'::numeric, 1"},
		{"energy precision overflow", "1000000, 1, 1, 1"},
		{"cost precision overflow", "1, 1, 10000, 1"},
		{"duration smallint overflow", "1, 1, 1, 32768"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := db.ExecContext(ctx, `INSERT INTO charging_processes
				(car_id, start_date, charge_energy_used, charge_energy_added, cost, duration_min)
				VALUES (1, '2026-01-01', `+tc.values+`)`)
			var pgErr *pq.Error
			if !errors.As(err, &pgErr) || pgErr.Code != "22003" {
				t.Fatalf("got %v, want PostgreSQL numeric_value_out_of_range", err)
			}
		})
	}
}

func TestStatsByGeofenceIntegrationLargeDuration(t *testing.T) {
	db := openGeofenceTestDB(t)
	// Each duration fits smallint; only the aggregate exceeds int32.
	execGeofenceFixture(t, db, `INSERT INTO charging_processes
		(car_id, geofence_id, start_date, end_date, charge_energy_used, charge_energy_added, cost, duration_min)
		SELECT 1, 1, '2026-01-01'::timestamp, '2026-01-01'::timestamp + interval '30000 minutes',
			1, 0.9, 0.5, 30000 FROM generate_series(1, 100000)`)
	rows := requestGeofenceTest(t, db, "1/stats/by-geofence", http.StatusOK)
	if len(rows) != 1 || rows[0].Duration == nil || *rows[0].Duration != 3000000000 {
		t.Fatalf("duration sum did not preserve int64: %+v", rows)
	}
}
