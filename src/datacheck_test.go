package main

import (
	"math"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func datacheckEnabled() bool {
	return os.Getenv("TESLAMATEAPI_DATACHECK") == "1"
}

func referenceDriveCount(carID int, parsedStart, parsedEnd string) (int, error) {
	q := `SELECT COUNT(*)::int FROM drives WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL`
	params := []any{carID}
	idx := 2
	q, params, _ = appendSummaryDateFilters(q, params, idx, "drives", parsedStart, parsedEnd)
	var n int
	err := db.QueryRow(q, params...).Scan(&n)
	return n, err
}

func referenceDriveDistanceSumKm(carID int, parsedStart, parsedEnd string) (float64, error) {
	q := `SELECT COALESCE(SUM(GREATEST(COALESCE(drives.distance, 0), 0)), 0)::float8 FROM drives WHERE drives.car_id = $1 AND drives.end_date IS NOT NULL`
	params := []any{carID}
	idx := 2
	q, params, _ = appendSummaryDateFilters(q, params, idx, "drives", parsedStart, parsedEnd)
	var v float64
	err := db.QueryRow(q, params...).Scan(&v)
	return v, err
}

func referenceChargeCount(carID int, parsedStart, parsedEnd string) (int, error) {
	q := `SELECT COUNT(*)::int FROM charging_processes WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL`
	params := []any{carID}
	idx := 2
	q, params, _ = appendSummaryDateFilters(q, params, idx, "charging_processes", parsedStart, parsedEnd)
	var n int
	err := db.QueryRow(q, params...).Scan(&n)
	return n, err
}

func referenceChargeEnergyAddedSum(carID int, parsedStart, parsedEnd string) (float64, error) {
	q := `SELECT COALESCE(SUM(GREATEST(COALESCE(charging_processes.charge_energy_added, 0), 0)), 0)::float8 FROM charging_processes WHERE charging_processes.car_id = $1 AND charging_processes.end_date IS NOT NULL`
	params := []any{carID}
	idx := 2
	q, params, _ = appendSummaryDateFilters(q, params, idx, "charging_processes", parsedStart, parsedEnd)
	var v float64
	err := db.QueryRow(q, params...).Scan(&v)
	return v, err
}

// TestDatacheckSummaryVsReferenceSQL 需要真实 Postgres：设置 TESLAMATEAPI_DATACHECK=1 与 DATABASE_*、TZ、ENCRYPTION_KEY（与运行服务相同）。
// 可选 TESLAMATEAPI_DATACHECK_CAR_ID（默认 1）、TESLAMATEAPI_DATACHECK_START_DATE / END_DATE（RFC3339 或 parseDateParam 支持的格式）。
func TestDatacheckSummaryVsReferenceSQL(t *testing.T) {
	if !datacheckEnabled() {
		t.Skip("数据校验：设置 TESLAMATEAPI_DATACHECK=1 及 DATABASE_*、TZ 后运行 go test -run TestDatacheck")
	}

	var err error
	appUsersTimezone, err = time.LoadLocation(getEnv("TZ", "Europe/Berlin"))
	if err != nil {
		t.Fatal(err)
	}

	initDBconnection()
	t.Cleanup(func() {
		if db != nil {
			_ = db.Close()
			db = nil
		}
	})

	carID := getEnvAsInt("TESLAMATEAPI_DATACHECK_CAR_ID", 1)
	pStart, pEnd := "", ""
	if s := getEnv("TESLAMATEAPI_DATACHECK_START_DATE", ""); s != "" {
		pStart, err = parseDateParam(s)
		if err != nil {
			t.Fatal("TESLAMATEAPI_DATACHECK_START_DATE:", err)
		}
	}
	if s := getEnv("TESLAMATEAPI_DATACHECK_END_DATE", ""); s != "" {
		pEnd, err = parseDateParam(s)
		if err != nil {
			t.Fatal("TESLAMATEAPI_DATACHECK_END_DATE:", err)
		}
	}

	unitsLength, _, _, err := fetchSummaryMetadata(carID)
	if err != nil {
		t.Fatal("fetchSummaryMetadata:", err)
	}

	driveSummary, err := fetchDriveHistorySummary(carID, pStart, pEnd, unitsLength)
	if err != nil {
		t.Fatal("fetchDriveHistorySummary:", err)
	}
	chargeSummary, err := fetchChargeHistorySummary(carID, pStart, pEnd, unitsLength)
	if err != nil {
		t.Fatal("fetchChargeHistorySummary:", err)
	}

	refDrives, err := referenceDriveCount(carID, pStart, pEnd)
	if err != nil {
		t.Fatal("referenceDriveCount:", err)
	}
	if driveSummary.DriveCount != refDrives {
		t.Errorf("drive_count: API=%d 独立SQL=%d (car_id=%d start=%q end=%q)", driveSummary.DriveCount, refDrives, carID, pStart, pEnd)
	}

	refCharges, err := referenceChargeCount(carID, pStart, pEnd)
	if err != nil {
		t.Fatal("referenceChargeCount:", err)
	}
	if chargeSummary.ChargeCount != refCharges {
		t.Errorf("charge_count: API=%d 独立SQL=%d", chargeSummary.ChargeCount, refCharges)
	}

	sumKm, err := referenceDriveDistanceSumKm(carID, pStart, pEnd)
	if err != nil {
		t.Fatal("referenceDriveDistanceSumKm:", err)
	}
	wantDist := sumKm
	if unitsLength == "mi" {
		wantDist = kilometersToMiles(sumKm)
	}
	if driveSummary.DriveCount == 0 && sumKm == 0 {
		// ok
	} else if math.Abs(driveSummary.TotalDistance-wantDist) > 1e-3 {
		t.Errorf("total_distance: API=%g 独立SUM(km)换算后=%g units=%q", driveSummary.TotalDistance, wantDist, unitsLength)
	}

	sumEnergy, err := referenceChargeEnergyAddedSum(carID, pStart, pEnd)
	if err != nil {
		t.Fatal("referenceChargeEnergyAddedSum:", err)
	}
	if chargeSummary.ChargeCount == 0 && sumEnergy == 0 {
		return
	}
	if math.Abs(chargeSummary.TotalEnergyAdded-sumEnergy) > 1e-6 {
		t.Errorf("total_energy_added: API=%g 独立SUM=%g", chargeSummary.TotalEnergyAdded, sumEnergy)
	}
}
