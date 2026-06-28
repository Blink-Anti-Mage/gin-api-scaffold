package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCursorPaginationMiddlewareStoresParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/users", CursorPagination(CursorPaginationConfig{
		DefaultLimit: 25,
		MaxLimit:     50,
	}), func(c *gin.Context) {
		params, ok := CurrentCursorPagination(c)
		if !ok {
			t.Fatal("expected cursor pagination params")
		}

		c.JSON(http.StatusOK, gin.H{
			"limit":  params.Limit,
			"cursor": params.Cursor,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/users?limit=100&cursor=%20abc%20", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"limit":50`) {
		t.Fatalf("expected capped limit in body: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"cursor":"abc"`) {
		t.Fatalf("expected trimmed cursor in body: %s", rec.Body.String())
	}
}

func TestCursorPaginationMiddlewareUsesDefaultLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/users", CursorPagination(CursorPaginationConfig{
		DefaultLimit: 25,
		MaxLimit:     50,
	}), func(c *gin.Context) {
		params, ok := CurrentCursorPagination(c)
		if !ok {
			t.Fatal("expected cursor pagination params")
		}

		c.JSON(http.StatusOK, gin.H{"limit": params.Limit})
	})

	req := httptest.NewRequest(http.MethodGet, "/users?limit=0", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"limit":25`) {
		t.Fatalf("expected default limit in body: %s", rec.Body.String())
	}
}

func TestCursorPaginationMiddlewareRejectsInvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/users", CursorPagination(CursorPaginationConfig{
		DefaultLimit: 25,
		MaxLimit:     50,
	}), func(c *gin.Context) {
		t.Fatal("handler should not be called for invalid limit")
	})

	req := httptest.NewRequest(http.MethodGet, "/users?limit=bad", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"invalid_query"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
