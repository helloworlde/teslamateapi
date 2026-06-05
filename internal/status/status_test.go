package status

import "testing"

func TestSnapshotReturnsCopy(t *testing.T) {
	c := &Cache{
		cache: map[int]*Info{
			1: {
				MQTTDataState:        "online",
				MQTTDataBatteryLevel: 80,
				MQTTDataLocation: InfoLocation{
					Latitude:  52.52,
					Longitude: 13.405,
				},
			},
		},
	}

	snapshot, ok := c.Snapshot(1)
	if !ok {
		t.Fatal("expected snapshot")
	}

	c.cache[1].MQTTDataState = "asleep"
	c.cache[1].MQTTDataBatteryLevel = 50
	c.cache[1].MQTTDataLocation.Latitude = 1

	if snapshot.MQTTDataState != "online" {
		t.Fatalf("snapshot shares string field with cache: %q", snapshot.MQTTDataState)
	}
	if snapshot.MQTTDataBatteryLevel != 80 {
		t.Fatalf("snapshot shares int field with cache: %d", snapshot.MQTTDataBatteryLevel)
	}
	if snapshot.MQTTDataLocation.Latitude != 52.52 {
		t.Fatalf("snapshot shares nested struct with cache: %f", snapshot.MQTTDataLocation.Latitude)
	}
}

func TestGetReturnsDetachedPointer(t *testing.T) {
	c := &Cache{
		cache: map[int]*Info{
			1: {MQTTDataState: "online"},
		},
	}

	got := c.Get(1)
	if got == nil {
		t.Fatal("expected info")
	}
	got.MQTTDataState = "mutated"

	if c.cache[1].MQTTDataState != "online" {
		t.Fatalf("Get returned live cache pointer")
	}
}
