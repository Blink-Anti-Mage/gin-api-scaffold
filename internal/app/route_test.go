package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/gin-api-scaffold/internal/config"
	authmodel "github.com/example/gin-api-scaffold/internal/models/auth"
	authservice "github.com/example/gin-api-scaffold/internal/services/auth"
)

type routeTestAuthRepository struct{}

func (routeTestAuthRepository) GetByEmail(ctx context.Context, email string) (authmodel.AuthUser, error) {
	return authmodel.AuthUser{}, nil
}

func TestAuthLoginRouteIsPublicWhenJWTEnabled(t *testing.T) {
	cfg := config.Default()
	cfg.App.Env = "test"
	cfg.Auth.Enabled = true
	cfg.Auth.Secret = "test-secret-for-route-auth-32-bytes"
	cfg.RateLimit.Enabled = false

	router := NewRouter(RouterDeps{
		Config:      cfg,
		AuthService: authservice.NewAuthService(cfg.Auth, routeTestAuthRepository{}),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected public login to return 400 for invalid body, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "missing_token") {
		t.Fatalf("login route should not require bearer token: %s", rec.Body.String())
	}
}

func TestProtectedAuthRouteRequiresJWT(t *testing.T) {
	cfg := config.Default()
	cfg.App.Env = "test"
	cfg.Auth.Enabled = true
	cfg.Auth.Secret = "test-secret-for-route-auth-32-bytes"
	cfg.RateLimit.Enabled = false

	router := NewRouter(RouterDeps{
		Config:      cfg,
		AuthService: authservice.NewAuthService(cfg.Auth, routeTestAuthRepository{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected protected route to return 401, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "missing_token") {
		t.Fatalf("expected missing_token response: %s", rec.Body.String())
	}
}
