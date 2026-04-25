package main

import "testing"

func TestMakeWeekdayBucketsStable(t *testing.T) {
	b := makeWeekdayBuckets("t", "weekday", "km")
	if len(b) != 7 {
		t.Fatalf("len %d", len(b))
	}
	if b[0].Value != 0 || b[0].Period != "weekday" || b[0].Unit != "km" {
		t.Fatalf("%+v", b[0])
	}
}

func TestMakeHourBucketsStable(t *testing.T) {
	b := makeHourBuckets("h", "hour", "sessions")
	if len(b) != 24 || b[0].Label != "00:00" {
		t.Fatalf("hour buckets %+v", b[0])
	}
}
