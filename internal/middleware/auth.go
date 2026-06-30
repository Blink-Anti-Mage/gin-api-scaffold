package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/config"
	"github.com/example/gin-api-scaffold/pkg/response"
)

const (
	AuthClaimsKey  = "auth_claims"
	AuthSubjectKey = "auth_subject"
)

type JWTClaims struct {
	Subject   string
	Issuer    string
	Audience  []string
	ExpiresAt time.Time
	NotBefore *time.Time
	IssuedAt  *time.Time
	JWTID     string
	Roles     []string
	Scopes    []string
	Raw       map[string]any
}

type jwtTokenClaims struct {
	jwt.RegisteredClaims
	Roles  []string       `json:"roles"`
	Scopes []string       `json:"scopes"`
	Scope  string         `json:"scope"`
	Raw    map[string]any `json:"-"`
}

type jwtTokenClaimsAlias jwtTokenClaims

type TokenRevocationChecker interface {
	IsRevoked(jwtID string) bool
}

func (c *jwtTokenClaims) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var claims jwtTokenClaimsAlias
	if err := json.Unmarshal(data, &claims); err != nil {
		return err
	}

	*c = jwtTokenClaims(claims)
	c.Raw = raw
	return nil
}

func JWT(cfg config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			unauthorized(c, "missing_token", "missing bearer token")
			return
		}

		claims, err := parseAndValidateJWT(token, cfg, time.Now())
		if err != nil {
			unauthorized(c, "invalid_token", "invalid or expired token")
			return
		}

		c.Set(AuthClaimsKey, claims)
		c.Set(AuthSubjectKey, claims.Subject)
		c.Next()
	}
}

func RejectRevokedJWT(checker TokenRevocationChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checker == nil {
			c.Next()
			return
		}

		claims, ok := CurrentJWTClaims(c)
		if ok && checker.IsRevoked(claims.JWTID) {
			unauthorized(c, "revoked_token", "token has been revoked")
			return
		}

		c.Next()
	}
}

func CurrentJWTClaims(c *gin.Context) (JWTClaims, bool) {
	value, ok := c.Get(AuthClaimsKey)
	if !ok {
		return JWTClaims{}, false
	}

	claims, ok := value.(JWTClaims)
	return claims, ok
}

func CurrentSubject(c *gin.Context) (string, bool) {
	subject := c.GetString(AuthSubjectKey)
	return subject, subject != ""
}

func parseAndValidateJWT(token string, cfg config.AuthConfig, now time.Time) (JWTClaims, error) {
	parsed := &jwtTokenClaims{}
	jwtToken, err := jwt.ParseWithClaims(token, parsed, jwtKeyFunc(cfg), jwtParserOptions(cfg, now)...)
	if err != nil {
		return JWTClaims{}, err
	}
	if jwtToken == nil || !jwtToken.Valid {
		return JWTClaims{}, errors.New("jwt is invalid")
	}

	claims := jwtClaimsFromTokenClaims(parsed)
	if claims.Subject == "" {
		return JWTClaims{}, errors.New("jwt sub claim is required")
	}

	return claims, nil
}

func jwtParserOptions(cfg config.AuthConfig, now time.Time) []jwt.ParserOption {
	skew := cfg.ClockSkew
	if skew < 0 {
		skew = 0
	}

	options := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(skew),
		jwt.WithTimeFunc(func() time.Time {
			return now
		}),
	}
	if cfg.Issuer != "" {
		options = append(options, jwt.WithIssuer(cfg.Issuer))
	}
	if cfg.Audience != "" {
		options = append(options, jwt.WithAudience(cfg.Audience))
	}
	return options
}

func jwtKeyFunc(cfg config.AuthConfig) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if token.Method == nil {
			return nil, errors.New("missing jwt signing method")
		}
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected jwt signing method %s", token.Method.Alg())
		}
		return []byte(cfg.Secret), nil
	}
}

func jwtClaimsFromTokenClaims(parsed *jwtTokenClaims) JWTClaims {
	claims := JWTClaims{
		Subject:  strings.TrimSpace(parsed.Subject),
		Issuer:   parsed.Issuer,
		Audience: []string(parsed.Audience),
		JWTID:    parsed.ID,
		Roles:    parsed.Roles,
		Scopes:   normalizedScopes(parsed.Scopes, parsed.Scope),
		Raw:      parsed.Raw,
	}
	if parsed.ExpiresAt != nil {
		claims.ExpiresAt = parsed.ExpiresAt.Time
	}
	if parsed.NotBefore != nil {
		notBefore := parsed.NotBefore.Time
		claims.NotBefore = &notBefore
	}
	if parsed.IssuedAt != nil {
		issuedAt := parsed.IssuedAt.Time
		claims.IssuedAt = &issuedAt
	}
	return claims
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "

	header = strings.TrimSpace(header)
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}

	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

func normalizedScopes(scopes []string, scope string) []string {
	result := make([]string, 0, len(scopes)+4)
	for _, item := range scopes {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	for _, item := range strings.Fields(scope) {
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func unauthorized(c *gin.Context, code string, message string) {
	c.Header("WWW-Authenticate", `Bearer realm="api"`)
	response.Error(c, apperr.New(http.StatusUnauthorized, code, message))
}
