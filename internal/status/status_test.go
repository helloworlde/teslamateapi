package status

import "testing"

type fakeMessage struct {
	topic   string
	payload []byte
}

func (m fakeMessage) Duplicate() bool   { return false }
func (m fakeMessage) Qos() byte         { return 0 }
func (m fakeMessage) Retained() bool    { return false }
func (m fakeMessage) Topic() string     { return m.topic }
func (m fakeMessage) MessageID() uint16 { return 0 }
func (m fakeMessage) Payload() []byte   { return m.payload }
func (m fakeMessage) Ack()              {}

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

func TestLockedStartsUnknown(t *testing.T) {
	c := &Cache{
		cache: map[int]*Info{
			1: {MQTTDataState: "offline"},
		},
	}

	got, ok := c.Snapshot(1)
	if !ok {
		t.Fatal("expected snapshot")
	}
	if got.MQTTDataLocked != nil {
		t.Fatalf("expected unknown lock state, got %v", *got.MQTTDataLocked)
	}
}

func TestLockedMessageParsesBool(t *testing.T) {
	c := &Cache{
		topicScan: "teslamate/cars/%d/%s",
		cache:     map[int]*Info{},
	}

	c.newMessage(nil, fakeMessage{
		topic:   "teslamate/cars/1/locked",
		payload: []byte("true"),
	})

	got, ok := c.Snapshot(1)
	if !ok {
		t.Fatal("expected snapshot")
	}
	if got.MQTTDataLocked == nil || *got.MQTTDataLocked != true {
		t.Fatalf("expected locked=true, got %#v", got.MQTTDataLocked)
	}
}

func TestInvalidLockedMessageDoesNotOverwritePreviousValue(t *testing.T) {
	locked := true
	c := &Cache{
		topicScan: "teslamate/cars/%d/%s",
		cache: map[int]*Info{
			1: {MQTTDataLocked: &locked},
		},
	}

	c.newMessage(nil, fakeMessage{
		topic:   "teslamate/cars/1/locked",
		payload: []byte(""),
	})

	got, ok := c.Snapshot(1)
	if !ok {
		t.Fatal("expected snapshot")
	}
	if got.MQTTDataLocked == nil || *got.MQTTDataLocked != true {
		t.Fatalf("invalid payload should preserve previous lock state, got %#v", got.MQTTDataLocked)
	}
}
