package response

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type bindTestUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func TestBindJSONReturnsValidationDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/users", func(c *gin.Context) {
		c.Writer.Header().Set(RequestIDHeader, "req-123")

		var req bindTestUserRequest
		if !BindJSON(c, &req) {
			return
		}

		OK(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"email":"not-an-email"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":"validation_failed"`) {
		t.Fatalf("expected validation_failed body: %s", body)
	}
	if !strings.Contains(body, `"details"`) {
		t.Fatalf("expected details body: %s", body)
	}
	if !strings.Contains(body, `"field":"name"`) || !strings.Contains(body, `"reason":"is required"`) {
		t.Fatalf("expected name detail body: %s", body)
	}
	if !strings.Contains(body, `"field":"email"`) || !strings.Contains(body, `"reason":"invalid email"`) {
		t.Fatalf("expected email detail body: %s", body)
	}
	if !strings.Contains(body, `"request_id":"req-123"`) {
		t.Fatalf("expected request_id body: %s", body)
	}
}
