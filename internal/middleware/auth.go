package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/config"
	"github.com/example/gin-api-scaffold/internal/httpx"
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

type rawJWTHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type rawJWTClaims struct {
	Subject   string       `json:"sub"`
	Issuer    string       `json:"iss"`
	Audience  audienceList `json:"aud"`
	ExpiresAt *numericDate `json:"exp"`
	NotBefore *numericDate `json:"nbf"`
	IssuedAt  *numericDate `json:"iat"`
	JWTID     string       `json:"jti"`
	Roles     []string     `json:"roles"`
	Scopes    []string     `json:"scopes"`
	Scope     string       `json:"scope"`
}

type numericDate struct {
	time.Time
}

func (d *numericDate) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var seconds json.Number
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&seconds); err != nil {
		return err
	}

	value, err := seconds.Int64()
	if err != nil {
		return err
	}
	d.Time = time.Unix(value, 0).UTC()
	return nil
}

type audienceList []string

func (a *audienceList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*a = audienceList{single}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*a = many
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
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return JWTClaims{}, errors.New("jwt must contain header, payload, and signature")
	}

	headerJSON, err := decodeJWTPart(parts[0])
	if err != nil {
		return JWTClaims{}, fmt.Errorf("decode jwt header: %w", err)
	}

	var header rawJWTHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return JWTClaims{}, fmt.Errorf("parse jwt header: %w", err)
	}
	if header.Algorithm != "HS256" {
		return JWTClaims{}, fmt.Errorf("unsupported jwt alg %q", header.Algorithm)
	}

	if !validHMACSHA256(parts[0]+"."+parts[1], parts[2], cfg.Secret) {
		return JWTClaims{}, errors.New("jwt signature mismatch")
	}

	payloadJSON, err := decodeJWTPart(parts[1])
	if err != nil {
		return JWTClaims{}, fmt.Errorf("decode jwt payload: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(payloadJSON, &raw); err != nil {
		return JWTClaims{}, fmt.Errorf("parse jwt raw claims: %w", err)
	}

	var parsed rawJWTClaims
	if err := json.Unmarshal(payloadJSON, &parsed); err != nil {
		return JWTClaims{}, fmt.Errorf("parse jwt claims: %w", err)
	}

	claims := JWTClaims{
		Subject:  strings.TrimSpace(parsed.Subject),
		Issuer:   parsed.Issuer,
		Audience: []string(parsed.Audience),
		JWTID:    parsed.JWTID,
		Roles:    parsed.Roles,
		Scopes:   normalizedScopes(parsed.Scopes, parsed.Scope),
		Raw:      raw,
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

	if err := validateJWTClaims(claims, cfg, now); err != nil {
		return JWTClaims{}, err
	}

	return claims, nil
}

func validateJWTClaims(claims JWTClaims, cfg config.AuthConfig, now time.Time) error {
	skew := cfg.ClockSkew
	if skew < 0 {
		skew = 0
	}

	if claims.Subject == "" {
		return errors.New("jwt sub claim is required")
	}
	if claims.ExpiresAt.IsZero() {
		return errors.New("jwt exp claim is required")
	}
	if now.After(claims.ExpiresAt.Add(skew)) {
		return errors.New("jwt is expired")
	}
	if claims.NotBefore != nil && now.Add(skew).Before(*claims.NotBefore) {
		return errors.New("jwt is not valid yet")
	}
	if claims.IssuedAt != nil && now.Add(skew).Before(*claims.IssuedAt) {
		return errors.New("jwt was issued in the future")
	}
	if cfg.Issuer != "" && claims.Issuer != cfg.Issuer {
		return errors.New("jwt issuer mismatch")
	}
	if cfg.Audience != "" && !containsString(claims.Audience, cfg.Audience) {
		return errors.New("jwt audience mismatch")
	}

	return nil
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

func validHMACSHA256(signingInput string, signature string, secret string) bool {
	decodedSignature, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSignature := mac.Sum(nil)

	return hmac.Equal(decodedSignature, expectedSignature)
}

func decodeJWTPart(part string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(part)
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

func containsString(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

func unauthorized(c *gin.Context, code string, message string) {
	c.Header("WWW-Authenticate", `Bearer realm="api"`)
	httpx.Error(c, apperr.New(http.StatusUnauthorized, code, message))
}
