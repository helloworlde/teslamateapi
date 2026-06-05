package respond

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSafeRequestURIRedactsTokenQueryValues(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cars?token=secret&access_token=secret2&page=1", nil)
	c.Request = req

	got := SafeRequestURI(c)
	if got != "/api/v1/cars?access_token=%5BREDACTED%5D&page=1&token=%5BREDACTED%5D" {
		t.Fatalf("safeRequestURI() = %q", got)
	}
}
