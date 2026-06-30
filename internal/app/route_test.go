package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/gin-api-scaffold/internal/config"
	"github.com/example/gin-api-scaffold/internal/models"
	"github.com/example/gin-api-scaffold/internal/services"
)

type routeTestAuthRepository struct{}

func (routeTestAuthRepository) GetByEmail(ctx context.Context, email string) (models.AuthUser, error) {
	return models.AuthUser{}, nil
}

type routeTestUsersRepository struct {
	create func(context.Context, models.User) (models.User, error)
}

func (r routeTestUsersRepository) List(ctx context.Context, filter models.ListUsersFilter) (models.UserList, error) {
	return models.UserList{}, nil
}

func (r routeTestUsersRepository) Get(ctx context.Context, id string) (models.User, error) {
	return models.User{}, nil
}

func (r routeTestUsersRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	if r.create == nil {
		return user, nil
	}
	return r.create(ctx, user)
}

func (r routeTestUsersRepository) Update(ctx context.Context, user models.User) (models.User, error) {
	return user, nil
}

func (r routeTestUsersRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (r routeTestUsersRepository) Stats(ctx context.Context) (models.UserStats, error) {
	return models.UserStats{}, nil
}

func TestAuthLoginRouteIsPublicWhenJWTEnabled(t *testing.T) {
	cfg := config.Default()
	cfg.App.Env = "test"
	cfg.Auth.Enabled = true
	cfg.Auth.Secret = "test-secret-for-route-auth-32-bytes"
	cfg.RateLimit.Enabled = false

	router := NewRouter(RouterDeps{
		Config:      cfg,
		AuthService: services.NewAuthService(cfg.Auth, routeTestAuthRepository{}),
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

func TestAuthRegisterRouteIsPublicWhenJWTEnabled(t *testing.T) {
	cfg := config.Default()
	cfg.App.Env = "test"
	cfg.Auth.Enabled = true
	cfg.Auth.Secret = "test-secret-for-route-auth-32-bytes"
	cfg.RateLimit.Enabled = false

	usersService := services.NewUsersService(routeTestUsersRepository{
		create: func(_ context.Context, user models.User) (models.User, error) {
			user.ID = "user-001"
			return user, nil
		},
	})
	router := NewRouter(RouterDeps{
		Config:       cfg,
		UsersService: usersService,
		AuthService:  services.NewAuthService(cfg.Auth, routeTestAuthRepository{}),
	})

	body := `{"name":"Ada Byron","email":"ada@example.com","password":"valid-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected public register to return 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "missing_token") {
		t.Fatalf("register route should not require bearer token: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"email":"ada@example.com"`) {
		t.Fatalf("expected created user response: %s", rec.Body.String())
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
		AuthService: services.NewAuthService(cfg.Auth, routeTestAuthRepository{}),
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
