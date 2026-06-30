package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/middleware"
	"github.com/example/gin-api-scaffold/internal/models"
	"github.com/example/gin-api-scaffold/internal/services"
	"github.com/example/gin-api-scaffold/pkg/response"
)

type AuthHandler struct {
	service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if !response.BindJSON(c, &req) {
		return
	}

	token, err := h.service.Login(c.Request.Context(), models.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, token)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	claims, ok := middleware.CurrentJWTClaims(c)
	if !ok {
		response.Error(c, apperr.New(http.StatusUnauthorized, "missing_token", "missing bearer token"))
		return
	}

	if err := h.service.Logout(c.Request.Context(), models.LogoutInput{
		JWTID:     claims.JWTID,
		ExpiresAt: claims.ExpiresAt,
	}); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}

func (h *AuthHandler) Me(c *gin.Context) {
	claims, ok := middleware.CurrentJWTClaims(c)
	if !ok {
		response.Error(c, apperr.New(http.StatusUnauthorized, "missing_token", "missing bearer token"))
		return
	}

	response.OK(c, models.CurrentUserResponse{
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
