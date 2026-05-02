package main

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRepairDateQueryParamOffsets(t *testing.T) {
	if got := repairDateQueryParam("2026-04-02T10:55:30 08:00"); got != "2026-04-02T10:55:30+08:00" {
		t.Fatalf("got %q", got)
	}
	if got := repairDateQueryParam("2026-04-02T10:55:30 -05:00"); got != "2026-04-02T10:55:30-05:00" {
		t.Fatalf("got %q", got)
	}
	if got := repairDateQueryParam("2026-04-02 10:55:30 +08:00"); got != "2026-04-02T10:55:30+08:00" {
		t.Fatalf("got %q", got)
	}
}

func TestParseCarID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "CarID", Value: "3"}}
	id, err := parseCarID(c)
	if err != nil || id != 3 {
		t.Fatalf("parseCarID valid: id=%d err=%v", id, err)
	}
	c.Params = gin.Params{{Key: "CarID", Value: "0"}}
	if _, err := parseCarID(c); err == nil {
		t.Fatal("parseCarID zero expected error")
	}
}

func TestParseDateParamLocalAndRFC3339(t *testing.T) {
	old := appUsersTimezone
	t.Cleanup(func() { appUsersTimezone = old })
	appUsersTimezone = time.FixedZone("Test", 8*3600)

	out, err := parseDateParam("2026-04-02 10:30:00")
	if err != nil || out == "" {
		t.Fatalf("local: %q err=%v", out, err)
	}
	out2, err := parseDateParam("2026-04-02T02:30:00Z")
	if err != nil || out2 == "" {
		t.Fatalf("z: %q err=%v", out2, err)
	}
	if _, err := parseDateParam("not-a-date"); err == nil {
		t.Fatal("expected invalid date error")
	}
}
