package v1

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/config"
	"github.com/tobiasehlert/teslamateapi/internal/database"
	"github.com/tobiasehlert/teslamateapi/pkg/dto"
)

func TestV1DriveCostIntegrationWithPostgres(t *testing.T) {
	if os.Getenv("TESLAMATEAPI_DB_INTEGRATION") != "1" {
		t.Skip("set TESLAMATEAPI_DB_INTEGRATION=1 with the dev Postgres container running")
	}

	db := openV1DriveCostIntegrationDB(t)
	t.Cleanup(func() { _ = db.Close() })
	resetV1DriveCostFixture(t, db)
	t.Cleanup(func() { resetV1DriveCostFixture(t, db) })
	insertV1DriveCostFixture(t, db)

	gin.SetMode(gin.TestMode)
	handler := New(Deps{
		DB: database.Wrap(db),
		TZ: time.FixedZone("UTC", 0),
	})

	list := requestV1DrivesForCost(t, handler, 91)
	driveA := mustFindV1DriveCostItem(t, list.Data.Drives, 91001)
	driveB := mustFindV1DriveCostItem(t, list.Data.Drives, 91002)
	driveNoEnergy := mustFindV1DriveCostItem(t, list.Data.Drives, 91003)

	assertFloatPtrClose(t, driveA.EnergyConsumedNet, 0.6345)
	assertFloatPtrClose(t, driveB.EnergyConsumedNet, 1.1415)
	assertFloatPtrClose(t, driveA.EstimatedUsageCost, 0.6345*0.647662)
	assertFloatPtrClose(t, driveB.EstimatedUsageCost, 1.1415*0.647662)
	if almostEqual(*driveA.EstimatedUsageCost, *driveB.EstimatedUsageCost) {
		t.Fatalf("same 1%% SOC drop produced equal costs: %f", *driveA.EstimatedUsageCost)
	}
	if driveNoEnergy.EnergyConsumedNet != nil || driveNoEnergy.EstimatedUsageCost != nil {
		t.Fatalf("non-positive rated-range drop energy/cost = %v/%v; want nil/nil", driveNoEnergy.EnergyConsumedNet, driveNoEnergy.EstimatedUsageCost)
	}

	detail := requestV1DriveDetailForCost(t, handler, 91, 91001)
	assertFloatPtrClose(t, detail.Data.Drive.EnergyConsumedNet, 0.6345)
	assertFloatPtrClose(t, detail.Data.Drive.EstimatedUsageCost, 0.6345*0.647662)

	noChargeDetail := requestV1DriveDetailForCost(t, handler, 92, 92001)
	assertFloatPtrClose(t, noChargeDetail.Data.Drive.EnergyConsumedNet, 0.75)
	if noChargeDetail.Data.Drive.EstimatedUsageCost != nil {
		t.Fatalf("drive without charge_energy_added cost = %f; want nil", *noChargeDetail.Data.Drive.EstimatedUsageCost)
	}
}

func openV1DriveCostIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	cfg := config.Config{
		DBHost:       integrationEnv("DATABASE_HOST", "127.0.0.1"),
		DBUser:       integrationEnv("DATABASE_USER", "teslamate"),
		DBPass:       integrationEnv("DATABASE_PASS", "secret"),
		DBName:       integrationEnv("DATABASE_NAME", "teslamate"),
		DBTimeoutMS:  5000,
		DBSSLMode:    integrationEnv("DATABASE_SSL", "disable"),
		DBDisableJIT: true,
	}
	port, err := strconv.Atoi(integrationEnv("DATABASE_PORT", "55432"))
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

func integrationEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func resetV1DriveCostFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	statements := []string{
		`DELETE FROM drives WHERE id IN (91001, 91002, 91003, 92001)`,
		`DELETE FROM charging_processes WHERE id IN (91001, 92001)`,
		`DELETE FROM positions WHERE id BETWEEN 91001 AND 92002`,
		`DELETE FROM cars WHERE id IN (91, 92)`,
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

func insertV1DriveCostFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	statements := []string{
		`INSERT INTO cars (id, name, model, efficiency) VALUES
			(91, 'Cost Fixture', 'Y', 0.15),
			(92, 'No Charge Fixture', '3', 0.15)`,
		`INSERT INTO positions (id, car_id, date, battery_level, usable_battery_level, rated_battery_range_km, ideal_battery_range_km, odometer, speed, power) VALUES
			(91001, 91, '2026-07-01 08:00:00', 65, 65, 156.23, 170.00, 1000.0, 0, 0),
			(91002, 91, '2026-07-01 08:10:00', 64, 64, 152.00, 165.77, 1003.0, 0, 0),
			(91003, 91, '2026-07-01 09:00:00', 63, 63, 180.61, 194.00, 1003.0, 0, 0),
			(91004, 91, '2026-07-01 09:15:00', 62, 62, 173.00, 186.39, 1009.0, 0, 0),
			(91005, 91, '2026-07-01 10:00:00', 61, 61, 150.00, 160.00, 1009.0, 0, 0),
			(91006, 91, '2026-07-01 10:05:00', 60, 60, 150.00, 160.00, 1010.0, 0, 0),
			(92001, 92, '2026-07-01 11:00:00', 70, 70, 100.00, 110.00, 2000.0, 0, 0),
			(92002, 92, '2026-07-01 11:10:00', 69, 69, 95.00, 105.00, 2005.0, 0, 0)`,
		`INSERT INTO drives (id, car_id, start_date, end_date, start_position_id, end_position_id, distance, duration_min, speed_max, power_max, power_min, start_km, end_km, start_ideal_range_km, end_ideal_range_km, start_rated_range_km, end_rated_range_km, outside_temp_avg, inside_temp_avg) VALUES
			(91001, 91, '2026-07-01 08:00:00', '2026-07-01 08:10:00', 91001, 91002, 3.0, 10, 50, 80, -10, 1000.0, 1003.0, 170.00, 165.77, 156.23, 152.00, 25.0, 21.0),
			(91002, 91, '2026-07-01 09:00:00', '2026-07-01 09:15:00', 91003, 91004, 6.0, 15, 55, 90, -12, 1003.0, 1009.0, 194.00, 186.39, 180.61, 173.00, 26.0, 21.0),
			(91003, 91, '2026-07-01 10:00:00', '2026-07-01 10:05:00', 91005, 91006, 1.0, 5, 30, 40, -5, 1009.0, 1010.0, 160.00, 160.00, 150.00, 150.00, 27.0, 21.0),
			(92001, 92, '2026-07-01 11:00:00', '2026-07-01 11:10:00', 92001, 92002, 5.0, 10, 45, 70, -8, 2000.0, 2005.0, 110.00, 105.00, 100.00, 95.00, 25.0, 21.0)`,
		`INSERT INTO charging_processes (id, car_id, start_date, end_date, position_id, charge_energy_added, charge_energy_used, cost, start_rated_range_km, end_rated_range_km, start_battery_level, end_battery_level, duration_min) VALUES
			(91001, 91, '2026-07-01 07:00:00', '2026-07-01 07:30:00', 91001, 10.0, 11.0, 6.47662, 100.0, 160.0, 50, 60, 30)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("insert fixture: %v\n%s", err, statement)
		}
	}
}

func requestV1DrivesForCost(t *testing.T, handler *Handler, carID int) dto.V1DrivesResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/cars/%d/drives?show=20", carID), nil)
	context.Params = gin.Params{{Key: "CarID", Value: strconv.Itoa(carID)}}

	handler.Drives(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("drives status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response dto.V1DrivesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode drives response: %v\n%s", err, recorder.Body.String())
	}
	return response
}

func requestV1DriveDetailForCost(t *testing.T, handler *Handler, carID, driveID int) dto.V1DriveDetailResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/cars/%d/drives/%d?include_route=false", carID, driveID), nil)
	context.Params = gin.Params{
		{Key: "CarID", Value: strconv.Itoa(carID)},
		{Key: "DriveID", Value: strconv.Itoa(driveID)},
	}

	handler.DrivesDetails(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("drive detail status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response dto.V1DriveDetailResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode drive detail response: %v\n%s", err, recorder.Body.String())
	}
	return response
}

func mustFindV1DriveCostItem(t *testing.T, drives []dto.V1DriveListItem, driveID int) dto.V1DriveListItem {
	t.Helper()

	for _, drive := range drives {
		if drive.DriveID == driveID {
			return drive
		}
	}
	t.Fatalf("drive_id %d not found in %d drives", driveID, len(drives))
	return dto.V1DriveListItem{}
}

func assertFloatPtrClose(t *testing.T, got *float64, want float64) {
	t.Helper()

	if got == nil {
		t.Fatalf("got nil; want %.12f", want)
	}
	if !almostEqual(*got, want) {
		t.Fatalf("got %.12f; want %.12f", *got, want)
	}
}
