package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/config"
	"github.com/example/gin-api-scaffold/internal/models"
)

const (
	defaultAccessTokenTTL = time.Hour
)

var defaultAuthScopes = []string{"users:read", "users:write"}
var defaultAuthRoles = []string{"admin"}

type AuthRepository interface {
	GetByEmail(ctx context.Context, email string) (models.AuthUser, error)
}

type AuthService struct {
	cfg  config.AuthConfig
	repo AuthRepository
	now  func() time.Time

	mu      sync.Mutex
	revoked map[string]time.Time
}

func NewAuthService(cfg config.AuthConfig, repo AuthRepository) *AuthService {
	return &AuthService{
		cfg:     cfg,
		repo:    repo,
		now:     time.Now,
		revoked: make(map[string]time.Time),
	}
}

func (s *AuthService) Login(ctx context.Context, input models.LoginInput) (models.LoginResponse, error) {
	if !s.cfg.Enabled {
		return models.LoginResponse{}, apperr.BadRequest("auth_disabled", "auth is disabled")
	}
	if strings.TrimSpace(s.cfg.Secret) == "" {
		return models.LoginResponse{}, apperr.New(http.StatusServiceUnavailable, "auth_unavailable", "auth is unavailable")
	}
	if s.repo == nil {
		return models.LoginResponse{}, apperr.New(http.StatusServiceUnavailable, "auth_unavailable", "auth is unavailable")
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if !validEmail(email) || input.Password == "" {
		return models.LoginResponse{}, apperr.New(http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if isNotFound(err) {
			return models.LoginResponse{}, apperr.New(http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		}
		return models.LoginResponse{}, err
	}
	if !passwordMatches(user.PasswordHash, input.Password) {
		return models.LoginResponse{}, apperr.New(http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	}

	now := s.now().UTC()
	expiresAt := now.Add(defaultAccessTokenTTL)
	scopes := append([]string(nil), defaultAuthScopes...)
	roles := append([]string(nil), defaultAuthRoles...)

	accessToken, err := s.signAccessToken(user, now, expiresAt, roles, scopes)
	if err != nil {
		return models.LoginResponse{}, err
	}

	return models.LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(defaultAccessTokenTTL / time.Second),
		ExpiresAt:   expiresAt,
		Subject:     user.ID,
		Roles:       roles,
		Scopes:      scopes,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, input models.LogoutInput) error {
	_ = ctx

	if input.JWTID == "" {
		return apperr.BadRequest("missing_token_id", "token id is required")
	}
	if input.ExpiresAt.IsZero() {
		return apperr.BadRequest("missing_token_expiry", "token expiry is required")
	}

	now := s.now().UTC()
	if !input.ExpiresAt.After(now) {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupRevokedLocked(now)
	s.revoked[input.JWTID] = input.ExpiresAt
	return nil
}

func (s *AuthService) IsRevoked(jwtID string) bool {
	if jwtID == "" {
		return false
	}

	now := s.now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupRevokedLocked(now)
	expiresAt, ok := s.revoked[jwtID]
	if !ok {
		return false
	}
	if !expiresAt.After(now) {
		delete(s.revoked, jwtID)
		return false
	}
	return true
}

func (s *AuthService) signAccessToken(user models.AuthUser, now time.Time, expiresAt time.Time, roles []string, scopes []string) (string, error) {
	jti, err := newAuthTokenID()
	if err != nil {
		return "", apperr.Internal(err)
	}

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"exp":   expiresAt.Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"jti":   jti,
		"roles": roles,
		"scope": strings.Join(scopes, " "),
	}
	if s.cfg.Issuer != "" {
		claims["iss"] = s.cfg.Issuer
	}
	if s.cfg.Audience != "" {
		claims["aud"] = []string{s.cfg.Audience}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return "", apperr.Internal(err)
	}
	return signed, nil
}

func (s *AuthService) cleanupRevokedLocked(now time.Time) {
	for jti, expiresAt := range s.revoked {
		if !expiresAt.After(now) {
			delete(s.revoked, jti)
		}
	}
}

func newAuthTokenID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:]), nil
	}
	return strconv.FormatInt(time.Now().UnixNano(), 36), nil
}

func isNotFound(err error) bool {
	appErr := apperr.From(err)
	return appErr != nil && appErr.Status == http.StatusNotFound
}
