package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/app/example/service"
	"github.com/example/gin-api-scaffold/app/example/types"
	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/httpx"
	"github.com/example/gin-api-scaffold/internal/middleware"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req types.LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	token, err := h.service.Login(c.Request.Context(), types.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, token)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	claims, ok := middleware.CurrentJWTClaims(c)
	if !ok {
		httpx.Error(c, apperr.New(http.StatusUnauthorized, "missing_token", "missing bearer token"))
		return
	}

	if err := h.service.Logout(c.Request.Context(), claims); err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.NoContent(c)
}

func (h *AuthHandler) Me(c *gin.Context) {
	claims, ok := middleware.CurrentJWTClaims(c)
	if !ok {
		httpx.Error(c, apperr.New(http.StatusUnauthorized, "missing_token", "missing bearer token"))
		return
	}

	httpx.OK(c, types.CurrentUserResponse{
		Subject:   claims.Subject,
		Email:     stringClaim(claims.Raw, "email"),
		Name:      stringClaim(claims.Raw, "name"),
		Roles:     claims.Roles,
		Scopes:    claims.Scopes,
		ExpiresAt: claims.ExpiresAt,
	})
}

func stringClaim(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return value
}
