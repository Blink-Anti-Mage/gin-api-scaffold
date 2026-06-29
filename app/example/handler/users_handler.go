package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/app/example/service"
	"github.com/example/gin-api-scaffold/app/example/types"
	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/httpx"
	"github.com/example/gin-api-scaffold/internal/middleware"
)

type UsersHandler struct {
	service *service.UsersService
}

func NewUsersHandler(service *service.UsersService) *UsersHandler {
	return &UsersHandler{
		service: service,
	}
}

func (h *UsersHandler) List(c *gin.Context) {
	filter := listUsersFilter(c)
	users, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, users)
}

func (h *UsersHandler) Stats(c *gin.Context) {
	stats, err := h.service.Stats(c.Request.Context())
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, stats)
}

func (h *UsersHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.Error(c, apperr.BadRequest("missing_user_id", "missing user id"))
		return
	}

	user, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, user)
}

func (h *UsersHandler) Create(c *gin.Context) {
	var req types.CreateUserRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	user, err := h.service.Create(c.Request.Context(), types.CreateUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.Created(c, user)
}

func (h *UsersHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.Error(c, apperr.BadRequest("missing_user_id", "missing user id"))
		return
	}

	var req types.UpdateUserRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	user, err := h.service.Update(c.Request.Context(), types.UpdateUserInput{
		ID:    id,
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, user)
}

func (h *UsersHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.Error(c, apperr.BadRequest("missing_user_id", "missing user id"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.NoContent(c)
}

func listUsersFilter(c *gin.Context) types.ListUsersFilter {
	pagination, _ := middleware.CurrentCursorPagination(c)

	return types.ListUsersFilter{
		Search: strings.TrimSpace(c.Query("search")),
		Limit:  pagination.Limit,
		Cursor: pagination.Cursor,
	}
}
