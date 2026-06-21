package v2

import (
	"strings"
	"testing"
)

func TestStatsByGeofenceSQLKeepsUngroupedRows(t *testing.T) {
	query := statsByGeofenceSQL("", "", "")

	required := []string{
		"SELECT d.start_geofence_id AS gid",
		"SELECT d.end_geofence_id AS gid",
		"SELECT cp.geofence_id AS gid",
		"SELECT dp.park_geofence_id AS gid",
		"dep.gid IS NOT DISTINCT FROM k.gid",
		"arr.gid IS NOT DISTINCT FROM k.gid",
		"ch.gid  IS NOT DISTINCT FROM k.gid",
		"pk.gid  IS NOT DISTINCT FROM k.gid",
	}
	for _, want := range required {
		if !strings.Contains(query, want) {
			t.Fatalf("statsByGeofenceSQL() missing %q", want)
		}
	}

	forbidden := []string{
		"start_geofence_id IS NOT NULL",
		"end_geofence_id IS NOT NULL",
		"cp.geofence_id IS NOT NULL",
		"park_geofence_id IS NOT NULL",
		"k.gid IS NOT NULL",
	}
	for _, bad := range forbidden {
		if strings.Contains(query, bad) {
			t.Fatalf("statsByGeofenceSQL() must not drop ungrouped rows with %q", bad)
		}
	}
}
