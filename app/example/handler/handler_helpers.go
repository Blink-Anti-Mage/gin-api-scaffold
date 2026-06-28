package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/httpx"
)

func optionalQueryInt(c *gin.Context, key string) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, true
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		httpx.Error(c, apperr.BadRequest("invalid_query", key+" must be an integer"))
		return 0, false
	}
	return value, true
}
