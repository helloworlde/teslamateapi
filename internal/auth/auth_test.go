package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/config"
)

const testToken = "0123456789abcdef0123456789abcdef" // 32 chars

func ctxWithRequest(header, query string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	target := "/api/v1/cars"
	if query != "" {
		target += "?" + query
	}
	c.Request = httptest.NewRequest("GET", target, nil)
	if header != "" {
		c.Request.Header.Set("Authorization", header)
	}
	return c
}

func TestValidate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tok := New(config.Config{APIToken: testToken})

	tests := []struct {
		name   string
		header string
		query  string
		want   bool
	}{
		{"valid bearer header", "Bearer " + testToken, "", true},
		{"valid bearer no space", "Bearer" + testToken, "", true},
		{"valid bearer extra whitespace", "Bearer   " + testToken + "  ", "", true},
		{"wrong token", "Bearer wrong", "", false},
		{"empty bearer", "Bearer ", "", false},
		{"missing scheme", testToken, "", false},
		{"lowercase scheme rejected", "bearer " + testToken, "", false},
		{"scheme not at start rejected", "foo Bearer " + testToken, "", false},
		{"valid query token", "", "token=" + testToken, true},
		{"invalid query token", "", "token=wrong", false},
		{"no credentials", "", "", false},
		{"header takes precedence over query", "Bearer wrong", "token=" + testToken, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msg := tok.Validate(ctxWithRequest(tt.header, tt.query))
			if got != tt.want {
				t.Errorf("Validate() = %v (%q), want %v", got, msg, tt.want)
			}
		})
	}
}

func TestValidateDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tok := New(config.Config{APIToken: testToken, APITokenDisable: true})
	if ok, _ := tok.Validate(ctxWithRequest("", "")); !ok {
		t.Error("Validate() with API_TOKEN_DISABLE=true should always pass")
	}
}

func TestValidateUnsetTokenRejects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tok := New(config.Config{})
	if ok, _ := tok.Validate(ctxWithRequest("Bearer anything", "")); ok {
		t.Error("Validate() with empty API_TOKEN should reject bearer credentials")
	}
}

func TestPrivilegedAuditAction(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/cars/1/command/honk_horn", "command_exec"},
		{"/api/v1/cars/1/commands", "command_exec"},
		{"/api/v1/cars/1/wake_up", "command_exec"},
		{"/api/v1/cars/1/logging/resume", "logging_exec"},
		{"/api/v1/cars/1/charges", ""},
	}
	for _, tt := range tests {
		if got := privilegedAuditAction(tt.path); got != tt.want {
			t.Errorf("privilegedAuditAction(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
