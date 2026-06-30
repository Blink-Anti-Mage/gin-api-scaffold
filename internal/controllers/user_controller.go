package controllers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/middleware"
	"github.com/example/gin-api-scaffold/internal/models"
	"github.com/example/gin-api-scaffold/internal/services"
	"github.com/example/gin-api-scaffold/pkg/response"
)

type UsersHandler struct {
	service *services.UsersService
}

func NewUsersHandler(service *services.UsersService) *UsersHandler {
	return &UsersHandler{
		service: service,
	}
}

func (h *UsersHandler) List(c *gin.Context) {
	filter := listUsersFilter(c)
	users, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, users)
}

func (h *UsersHandler) Stats(c *gin.Context) {
	stats, err := h.service.Stats(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, stats)
}

func (h *UsersHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("missing_user_id", "missing user id"))
		return
	}

	user, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, user)
}

func (h *UsersHandler) Create(c *gin.Context) {
	var req models.CreateUserRequest
	if !response.BindJSON(c, &req) {
		return
	}

	user, err := h.service.Create(c.Request.Context(), models.CreateUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, user)
}

func (h *UsersHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("missing_user_id", "missing user id"))
		return
	}

	var req models.UpdateUserRequest
	if !response.BindJSON(c, &req) {
		return
	}

	user, err := h.service.Update(c.Request.Context(), models.UpdateUserInput{
		ID:    id,
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, user)
}

func (h *UsersHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("missing_user_id", "missing user id"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}

func listUsersFilter(c *gin.Context) models.ListUsersFilter {
	pagination, _ := middleware.CurrentCursorPagination(c)

	return models.ListUsersFilter{
		Search: strings.TrimSpace(c.Query("search")),
		Limit:  pagination.Limit,
		Cursor: pagination.Cursor,
	}
}
