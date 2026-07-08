package v2

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/config"
	"github.com/tobiasehlert/teslamateapi/internal/database"
)

type v2LifetimeCostResponse struct {
	Data struct {
		Drives struct {
			EstimatedUsageCost *float64 `json:"estimated_usage_cost"`
		} `json:"drives"`
		Charges struct {
			CostPerDistance *float64 `json:"cost_per_distance"`
		} `json:"charges"`
	} `json:"data"`
}

type v2SummaryCostResponse struct {
	Data struct {
		Buckets []v2SummaryCostBucket `json:"buckets"`
	} `json:"data"`
}

type v2SummaryCostBucket struct {
	DrivesEstimatedUsageCost                *float64 `json:"drives_estimated_usage_cost"`
	DrivesCostPerDistance                   *float64 `json:"drives_cost_per_distance"`
	DrivesDistanceLifetimeCumulative        float64  `json:"drives_distance_lifetime_cumulative"`
	ChargesEnergyAddedKWhLifetimeCumulative float64  `json:"charges_energy_added_kwh_lifetime_cumulative"`
	ChargesEnergyUsedKWhLifetimeCumulative  float64  `json:"charges_energy_used_kwh_lifetime_cumulative"`
	ChargesCostLifetimeCumulative           float64  `json:"charges_cost_lifetime_cumulative"`
	VampireDrainKWhLifetimeCumulative       float64  `json:"vampire_drain_kwh_lifetime_cumulative"`
}

type v2ConsumptionCostResponse struct {
	Data struct {
		OverallEstimatedUsageCost *float64 `json:"overall_estimated_usage_cost"`
		OverallCostPerDistance    *float64 `json:"overall_cost_per_distance"`
		Groups                    []struct {
			Key                string   `json:"key"`
			EstimatedUsageCost *float64 `json:"estimated_usage_cost"`
			CostPerDistance    *float64 `json:"cost_per_distance"`
		} `json:"groups"`
	} `json:"data"`
}

func TestV2DriveCostIntegrationNullsPartialSOC(t *testing.T) {
	if os.Getenv("TESLAMATEAPI_DB_INTEGRATION") != "1" {
		t.Skip("set TESLAMATEAPI_DB_INTEGRATION=1 with the dev Postgres container running")
	}

	db := openV2DriveCostIntegrationDB(t)
	t.Cleanup(func() { _ = db.Close() })
	resetV2DriveCostFixture(t, db)
	t.Cleanup(func() { resetV2DriveCostFixture(t, db) })
	insertV2DriveCostFixture(t, db)

	gin.SetMode(gin.TestMode)
	handler := New(Deps{
		DB: database.Wrap(db),
		TZ: time.FixedZone("UTC", 0),
	})

	completeLifetime := requestV2LifetimeForCost(t, handler, 93)
	if completeLifetime.Data.Drives.EstimatedUsageCost == nil || completeLifetime.Data.Charges.CostPerDistance == nil {
		t.Fatalf("complete lifetime cost = %v/%v; want non-null/non-null",
			completeLifetime.Data.Drives.EstimatedUsageCost,
			completeLifetime.Data.Charges.CostPerDistance)
	}

	partialLifetime := requestV2LifetimeForCost(t, handler, 94)
	if partialLifetime.Data.Drives.EstimatedUsageCost != nil || partialLifetime.Data.Charges.CostPerDistance != nil {
		t.Fatalf("partial lifetime cost = %v/%v; want null/null",
			partialLifetime.Data.Drives.EstimatedUsageCost,
			partialLifetime.Data.Charges.CostPerDistance)
	}

	completeSummary := requestV2SummaryForCost(t, handler, 93)
	completeBucket := mustFirstV2SummaryBucket(t, completeSummary)
	if completeBucket.DrivesEstimatedUsageCost == nil || completeBucket.DrivesCostPerDistance == nil {
		t.Fatalf("complete summary cost = %v/%v; want non-null/non-null",
			completeBucket.DrivesEstimatedUsageCost,
			completeBucket.DrivesCostPerDistance)
	}

	partialSummary := requestV2SummaryForCost(t, handler, 94)
	partialBucket := mustFirstV2SummaryBucket(t, partialSummary)
	if partialBucket.DrivesEstimatedUsageCost != nil || partialBucket.DrivesCostPerDistance != nil {
		t.Fatalf("partial summary cost = %v/%v; want null/null",
			partialBucket.DrivesEstimatedUsageCost,
			partialBucket.DrivesCostPerDistance)
	}

	completeConsumption := requestV2ConsumptionForCost(t, handler, 93)
	completeGroup := mustFirstV2ConsumptionGroup(t, completeConsumption)
	if completeConsumption.Data.OverallEstimatedUsageCost == nil ||
		completeConsumption.Data.OverallCostPerDistance == nil ||
		completeGroup.EstimatedUsageCost == nil ||
		completeGroup.CostPerDistance == nil {
		t.Fatalf("complete consumption cost should be valid")
	}

	partialConsumption := requestV2ConsumptionForCost(t, handler, 94)
	partialGroup := mustFirstV2ConsumptionGroup(t, partialConsumption)
	if partialConsumption.Data.OverallEstimatedUsageCost != nil ||
		partialConsumption.Data.OverallCostPerDistance != nil ||
		partialGroup.EstimatedUsageCost != nil ||
		partialGroup.CostPerDistance != nil {
		t.Fatalf("partial consumption cost should be null")
	}

	mixedConsumption := requestV2ConsumptionForCost(t, handler, 95)
	if len(mixedConsumption.Data.Groups) != 2 {
		t.Fatalf("mixed consumption groups = %d; want 2", len(mixedConsumption.Data.Groups))
	}
	if mixedConsumption.Data.Groups[0].EstimatedUsageCost == nil ||
		mixedConsumption.Data.Groups[0].CostPerDistance == nil {
		t.Fatalf("mixed complete group cost = %v/%v; want non-null/non-null",
			mixedConsumption.Data.Groups[0].EstimatedUsageCost,
			mixedConsumption.Data.Groups[0].CostPerDistance)
	}
	if mixedConsumption.Data.Groups[1].EstimatedUsageCost != nil ||
		mixedConsumption.Data.Groups[1].CostPerDistance != nil {
		t.Fatalf("mixed partial group cost = %v/%v; want null/null",
			mixedConsumption.Data.Groups[1].EstimatedUsageCost,
			mixedConsumption.Data.Groups[1].CostPerDistance)
	}
	if mixedConsumption.Data.OverallEstimatedUsageCost != nil ||
		mixedConsumption.Data.OverallCostPerDistance != nil {
		t.Fatalf("mixed overall consumption cost = %v/%v; want null/null",
			mixedConsumption.Data.OverallEstimatedUsageCost,
			mixedConsumption.Data.OverallCostPerDistance)
	}

	windowedMixedSummary := requestV2SummaryForCost(
		t,
		handler,
		95,
		"start_date=2026-07-01T08:00:00Z",
		"end_date=2026-07-01T08:00:00Z",
	)
	if len(windowedMixedSummary.Data.Buckets) != 1 {
		t.Fatalf("windowed mixed summary buckets = %d; want 1", len(windowedMixedSummary.Data.Buckets))
	}
	windowedBucket := windowedMixedSummary.Data.Buckets[0]
	if windowedBucket.DrivesDistanceLifetimeCumulative != 20 ||
		windowedBucket.ChargesEnergyAddedKWhLifetimeCumulative != 10 ||
		windowedBucket.ChargesEnergyUsedKWhLifetimeCumulative != 11 ||
		windowedBucket.ChargesCostLifetimeCumulative != 5 ||
		math.Abs(windowedBucket.VampireDrainKWhLifetimeCumulative-3) > 0.000001 {
		t.Fatalf("windowed mixed cumulative summary = %+v; want lifetime totals through returned bucket", windowedBucket)
	}

	windowedParkingSummary := requestV2SummaryForCost(
		t,
		handler,
		96,
		"start_date=2026-07-02T00:00:00Z",
		"end_date=2026-07-02T00:00:00Z",
	)
	if len(windowedParkingSummary.Data.Buckets) != 1 {
		t.Fatalf("windowed parking summary buckets = %d; want 1", len(windowedParkingSummary.Data.Buckets))
	}
	windowedParkingBucket := windowedParkingSummary.Data.Buckets[0]
	if math.Abs(windowedParkingBucket.VampireDrainKWhLifetimeCumulative-3) > 0.000001 {
		t.Fatalf("windowed parking cumulative drain = %v; want 3.0 including pre-window baseline",
			windowedParkingBucket.VampireDrainKWhLifetimeCumulative)
	}
}

func openV2DriveCostIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	cfg := config.Config{
		DBHost:       v2IntegrationEnv("DATABASE_HOST", "127.0.0.1"),
		DBUser:       v2IntegrationEnv("DATABASE_USER", "teslamate"),
		DBPass:       v2IntegrationEnv("DATABASE_PASS", "secret"),
		DBName:       v2IntegrationEnv("DATABASE_NAME", "teslamate"),
		DBTimeoutMS:  5000,
		DBSSLMode:    v2IntegrationEnv("DATABASE_SSL", "disable"),
		DBDisableJIT: true,
	}
	port, err := strconv.Atoi(v2IntegrationEnv("DATABASE_PORT", "55432"))
	if err != nil {
		t.Fatalf("DATABASE_PORT invalid: %v", err)
	}
	cfg.DBPort = port

	db, err := database.New(cfg)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	return db
}

func v2IntegrationEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func resetV2DriveCostFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	statements := []string{
		`DELETE FROM drives WHERE id IN (93001, 94001, 95001, 95002, 96001, 96002)`,
		`DELETE FROM charging_processes WHERE id IN (93001, 94001, 95001)`,
		`DELETE FROM positions WHERE id IN (93001, 93002, 94001, 94002, 95001, 95002, 95003, 95004, 96001, 96002, 96003, 96004)`,
		`DELETE FROM cars WHERE id IN (93, 94, 95, 96)`,
		`INSERT INTO settings (id, unit_of_length, unit_of_temperature, preferred_range)
		 VALUES (1, 'km', 'C', 'rated')
		 ON CONFLICT (id) DO NOTHING`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("reset fixture: %v\n%s", err, statement)
		}
	}
}

func insertV2DriveCostFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	statements := []string{
		`INSERT INTO cars (id, name, model, efficiency, vin) VALUES
			(93, 'Complete Cost Fixture', 'Y', 0.15, 'V2COMPLETECOST00093'),
			(94, 'Partial Cost Fixture', 'Y', 0.15, 'V2PARTIALCOST00094'),
			(95, 'Mixed Cost Fixture', 'Y', 0.15, 'V2MIXEDCOST000095'),
			(96, 'Windowed Parking Fixture', 'Y', 0.15, 'V2PARKINGCOST000096')`,
		`INSERT INTO positions (id, car_id, date, battery_level, usable_battery_level, rated_battery_range_km, ideal_battery_range_km, odometer, speed, power) VALUES
			(93001, 93, '2026-07-01 08:00:00', 60, 60, 300.0, 320.0, 1000.0, 0, 0),
			(93002, 93, '2026-07-01 08:20:00', 58, 58, 288.0, 308.0, 1010.0, 0, 0),
			(94001, 94, '2026-07-01 08:00:00', 60, 60, 300.0, 320.0, 2000.0, 0, 0),
			(94002, 94, '2026-07-01 08:20:00', NULL, NULL, 288.0, 308.0, 2010.0, 0, 0),
			(95001, 95, '2026-06-01 08:00:00', 60, 60, 300.0, 320.0, 3000.0, 0, 0),
			(95002, 95, '2026-06-01 08:20:00', 58, 58, 288.0, 308.0, 3010.0, 0, 0),
			(95003, 95, '2026-07-01 08:00:00', 57, 57, 268.0, 288.0, 3010.0, 0, 0),
			(95004, 95, '2026-07-01 08:20:00', NULL, NULL, 256.0, 276.0, 3020.0, 0, 0),
			(96001, 96, '2026-06-30 23:40:00', 62, 62, 312.0, 332.0, 4000.0, 0, 0),
			(96002, 96, '2026-07-01 00:00:00', 60, 60, 300.0, 320.0, 4010.0, 0, 0),
			(96003, 96, '2026-07-03 00:00:00', 56, 56, 280.0, 300.0, 4010.0, 0, 0),
			(96004, 96, '2026-07-03 00:20:00', 54, 54, 268.0, 288.0, 4020.0, 0, 0)`,
		`INSERT INTO drives (id, car_id, start_date, end_date, start_position_id, end_position_id, distance, duration_min, speed_max, power_max, power_min, start_km, end_km, start_ideal_range_km, end_ideal_range_km, start_rated_range_km, end_rated_range_km, outside_temp_avg, inside_temp_avg) VALUES
			(93001, 93, '2026-07-01 08:00:00', '2026-07-01 08:20:00', 93001, 93002, 10.0, 20, 80, 100, -10, 1000.0, 1010.0, 320.0, 308.0, 300.0, 288.0, 25.0, 21.0),
			(94001, 94, '2026-07-01 08:00:00', '2026-07-01 08:20:00', 94001, 94002, 10.0, 20, 80, 100, -10, 2000.0, 2010.0, 320.0, 308.0, 300.0, 288.0, 25.0, 21.0),
			(95001, 95, '2026-06-01 08:00:00', '2026-06-01 08:20:00', 95001, 95002, 10.0, 20, 80, 100, -10, 3000.0, 3010.0, 320.0, 308.0, 300.0, 288.0, 25.0, 21.0),
			(95002, 95, '2026-07-01 08:00:00', '2026-07-01 08:20:00', 95003, 95004, 10.0, 20, 80, 100, -10, 3010.0, 3020.0, 288.0, 276.0, 268.0, 256.0, 25.0, 21.0),
			(96001, 96, '2026-06-30 23:40:00', '2026-07-01 00:00:00', 96001, 96002, 10.0, 20, 80, 100, -10, 4000.0, 4010.0, 332.0, 320.0, 312.0, 300.0, 25.0, 21.0),
			(96002, 96, '2026-07-03 00:00:00', '2026-07-03 00:20:00', 96003, 96004, 10.0, 20, 80, 100, -10, 4010.0, 4020.0, 300.0, 288.0, 280.0, 268.0, 25.0, 21.0)`,
		`INSERT INTO charging_processes (id, car_id, start_date, end_date, position_id, charge_energy_added, charge_energy_used, cost, start_battery_level, end_battery_level, duration_min) VALUES
			(93001, 93, '2026-07-01 07:00:00', '2026-07-01 07:30:00', 93001, 10.0, 11.0, 5.0, 50, 60, 30),
			(94001, 94, '2026-07-01 07:00:00', '2026-07-01 07:30:00', 94001, 10.0, 11.0, 5.0, 50, 60, 30),
			(95001, 95, '2026-06-01 07:00:00', '2026-06-01 07:30:00', 95001, 10.0, 11.0, 5.0, 50, 60, 30)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("insert fixture: %v\n%s", err, statement)
		}
	}
}

func requestV2LifetimeForCost(t *testing.T, handler *Handler, carID int) v2LifetimeCostResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/cars/%d/stats/lifetime", carID), nil)
	context.Params = gin.Params{{Key: "CarID", Value: strconv.Itoa(carID)}}

	handler.StatsLifetime(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("lifetime status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response v2LifetimeCostResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode lifetime response: %v\n%s", err, recorder.Body.String())
	}
	return response
}

func requestV2SummaryForCost(t *testing.T, handler *Handler, carID int, extraQuery ...string) v2SummaryCostResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	path := fmt.Sprintf("/api/v2/cars/%d/stats/summary?period=day", carID)
	for _, query := range extraQuery {
		path += "&" + query
	}
	context.Request = httptest.NewRequest(http.MethodGet, path, nil)
	context.Params = gin.Params{{Key: "CarID", Value: strconv.Itoa(carID)}}

	handler.StatsSummary(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("summary status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response v2SummaryCostResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode summary response: %v\n%s", err, recorder.Body.String())
	}
	return response
}

func requestV2ConsumptionForCost(t *testing.T, handler *Handler, carID int) v2ConsumptionCostResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/cars/%d/stats/consumption?group_by=month", carID), nil)
	context.Params = gin.Params{{Key: "CarID", Value: strconv.Itoa(carID)}}

	handler.StatsConsumption(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("consumption status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response v2ConsumptionCostResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode consumption response: %v\n%s", err, recorder.Body.String())
	}
	return response
}

func mustFirstV2SummaryBucket(t *testing.T, response v2SummaryCostResponse) v2SummaryCostBucket {
	t.Helper()

	if len(response.Data.Buckets) == 0 {
		t.Fatalf("summary returned no buckets")
	}
	return response.Data.Buckets[0]
}

func mustFirstV2ConsumptionGroup(t *testing.T, response v2ConsumptionCostResponse) struct {
	Key                string   `json:"key"`
	EstimatedUsageCost *float64 `json:"estimated_usage_cost"`
	CostPerDistance    *float64 `json:"cost_per_distance"`
} {
	t.Helper()

	if len(response.Data.Groups) == 0 {
		t.Fatalf("consumption returned no groups")
	}
	return response.Data.Groups[0]
}
