package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/config"
)

const testJWTSecret = "test-secret-for-jwt-middleware-32-bytes"

func TestJWTMiddlewareAcceptsValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.AuthConfig{
		Enabled:   true,
		Secret:    testJWTSecret,
		Issuer:    "gin-api",
		Audience:  "api-clients",
		ClockSkew: 30 * time.Second,
	}
	router := gin.New()
	router.Use(JWT(cfg))
	router.GET("/me", func(c *gin.Context) {
		claims, ok := CurrentJWTClaims(c)
		if !ok {
			t.Fatal("expected jwt claims")
		}
		subject, ok := CurrentSubject(c)
		if !ok {
			t.Fatal("expected auth subject")
		}

		c.JSON(http.StatusOK, gin.H{
			"subject": subject,
			"scopes":  claims.Scopes,
		})
	})

	token := signTestJWT(t, cfg.Secret, map[string]any{
		"sub":   "user-123",
		"iss":   "gin-api",
		"aud":   []string{"api-clients"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"scope": "users:read users:write",
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"subject":"user-123"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"users:read"`) {
		t.Fatalf("expected scope in body: %s", rec.Body.String())
	}
}

func TestJWTMiddlewareRequiresBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(JWT(config.AuthConfig{Secret: testJWTSecret}))
	router.GET("/me", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected WWW-Authenticate header")
	}
	if !strings.Contains(rec.Body.String(), `"code":"missing_token"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestJWTMiddlewareRejectsExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(JWT(config.AuthConfig{Secret: testJWTSecret, ClockSkew: time.Second}))
	router.GET("/me", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := signTestJWT(t, testJWTSecret, map[string]any{
		"sub": "user-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"invalid_token"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func signTestJWT(t *testing.T, secret string, payload map[string]any) string {
	t.Helper()

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	body := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := header + "." + body

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature
}
