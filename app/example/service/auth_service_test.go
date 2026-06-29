package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/example/gin-api-scaffold/app/example/types"
	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/config"
	"github.com/example/gin-api-scaffold/internal/middleware"
)

type recordingAuthRepository struct {
	getByEmail func(context.Context, string) (types.AuthUser, error)
}

func (r *recordingAuthRepository) GetByEmail(ctx context.Context, email string) (types.AuthUser, error) {
	if r.getByEmail == nil {
		return types.AuthUser{}, nil
	}
	return r.getByEmail(ctx, email)
}

func TestAuthServiceLoginSignsJWTForUserPassword(t *testing.T) {
	passwordHash, err := hashPassword("valid-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	var capturedEmail string
	repo := &recordingAuthRepository{
		getByEmail: func(_ context.Context, email string) (types.AuthUser, error) {
			capturedEmail = email
			return types.AuthUser{
				ID:           "user-001",
				Name:         "Ada Byron",
				Email:        "ada@example.com",
				PasswordHash: passwordHash,
			}, nil
		},
	}
	cfg := config.AuthConfig{
		Enabled:  true,
		Secret:   "test-secret-for-example-auth-service-32-bytes",
		Issuer:   "gin-api",
		Audience: "api-clients",
	}
	auth := NewAuthService(cfg, repo)
	auth.now = func() time.Time {
		return time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	}

	resp, err := auth.Login(context.Background(), types.LoginInput{
		Email:    " ADA@EXAMPLE.COM ",
		Password: "valid-password",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if capturedEmail != "ada@example.com" {
		t.Fatalf("expected normalized email, got %q", capturedEmail)
	}
	if resp.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if resp.TokenType != "Bearer" {
		t.Fatalf("expected Bearer token type, got %q", resp.TokenType)
	}
	if resp.Subject != "user-001" {
		t.Fatalf("expected subject user-001, got %q", resp.Subject)
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(resp.AccessToken, claims, func(token *jwt.Token) (any, error) {
		return []byte(cfg.Secret), nil
	}, jwt.WithIssuer(cfg.Issuer), jwt.WithAudience(cfg.Audience), jwt.WithTimeFunc(auth.now))
	if err != nil {
		t.Fatalf("parse signed token: %v", err)
	}
	if token == nil || !token.Valid {
		t.Fatal("expected valid token")
	}
	if claims["sub"] != "user-001" {
		t.Fatalf("expected token subject user-001, got %v", claims["sub"])
	}
	if claims["email"] != "ada@example.com" {
		t.Fatalf("expected email claim, got %v", claims["email"])
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		t.Fatal("expected token id")
	}
}

func TestAuthServiceLoginRejectsInvalidPassword(t *testing.T) {
	passwordHash, err := hashPassword("valid-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	auth := NewAuthService(config.AuthConfig{
		Enabled: true,
		Secret:  "test-secret-for-example-auth-service-32-bytes",
	}, &recordingAuthRepository{
		getByEmail: func(context.Context, string) (types.AuthUser, error) {
			return types.AuthUser{
				ID:           "user-001",
				Email:        "ada@example.com",
				PasswordHash: passwordHash,
			}, nil
		},
	})

	_, err = auth.Login(context.Background(), types.LoginInput{
		Email:    "ada@example.com",
		Password: "wrong-password",
	})
	if err == nil {
		t.Fatal("expected invalid credentials")
	}

	appErr := apperr.From(err)
	if appErr.Code != "invalid_credentials" {
		t.Fatalf("expected invalid_credentials, got %q", appErr.Code)
	}
}

func TestAuthServiceLogoutRevokesTokenID(t *testing.T) {
	auth := NewAuthService(config.AuthConfig{Enabled: true, Secret: "test-secret"}, nil)
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	auth.now = func() time.Time {
		return now
	}

	claims := middleware.JWTClaims{
		Subject:   "user-001",
		JWTID:     "token-001",
		ExpiresAt: now.Add(time.Hour),
	}
	if auth.IsRevoked(claims) {
		t.Fatal("token should not start revoked")
	}
	if err := auth.Logout(context.Background(), claims); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if !auth.IsRevoked(claims) {
		t.Fatal("expected token to be revoked")
	}
}
