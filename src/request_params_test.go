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

func TestParseChargeID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ChargeID", Value: "42"}}
	id, err := parseChargeID(c)
	if err != nil || id != 42 {
		t.Fatalf("parseChargeID valid: id=%d err=%v", id, err)
	}
	c.Params = gin.Params{{Key: "ChargeID", Value: "abc"}}
	if _, err := parseChargeID(c); err == nil {
		t.Fatal("parseChargeID invalid expected error")
	}
}

func TestParsePositiveIntQueryClamp(t *testing.T) {
	v, err := parsePositiveIntQuery("0", 10, 1, 100)
	if err != nil || v != 1 {
		t.Fatalf("clamp min: %d %v", v, err)
	}
	v, err = parsePositiveIntQuery("9999", 10, 1, 100)
	if err != nil || v != 100 {
		t.Fatalf("clamp max: %d %v", v, err)
	}
}

func TestParseSummaryIncludesInvalid(t *testing.T) {
	if _, err := parseSummaryIncludes("overview,not_a_real_include"); err == nil {
		t.Fatal("expected error for unknown include")
	}
}

func TestParseInsightTypesInvalid(t *testing.T) {
	if _, err := parseInsightTypes("harsh_brake,unknown_type"); err == nil {
		t.Fatal("expected error for unknown insight type")
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
