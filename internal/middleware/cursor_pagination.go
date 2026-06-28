package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/httpx"
)

const CursorPaginationKey = "cursor_pagination"

const defaultCursorPaginationLimit = 20

type CursorPaginationConfig struct {
	DefaultLimit int
	MaxLimit     int
}

type CursorPaginationParams struct {
	Limit  int
	Cursor string
}

func CursorPagination(cfg CursorPaginationConfig) gin.HandlerFunc {
	cfg = normalizeCursorPaginationConfig(cfg)

	return func(c *gin.Context) {
		limit, ok := cursorPaginationLimit(c, cfg.DefaultLimit)
		if !ok {
			return
		}
		if cfg.MaxLimit > 0 && limit > cfg.MaxLimit {
			limit = cfg.MaxLimit
		}

		c.Set(CursorPaginationKey, CursorPaginationParams{
			Limit:  limit,
			Cursor: strings.TrimSpace(c.Query("cursor")),
		})
		c.Next()
	}
}

func CurrentCursorPagination(c *gin.Context) (CursorPaginationParams, bool) {
	value, ok := c.Get(CursorPaginationKey)
	if !ok {
		return CursorPaginationParams{}, false
	}

	params, ok := value.(CursorPaginationParams)
	return params, ok
}

func normalizeCursorPaginationConfig(cfg CursorPaginationConfig) CursorPaginationConfig {
	if cfg.DefaultLimit <= 0 {
		cfg.DefaultLimit = defaultCursorPaginationLimit
	}
	if cfg.MaxLimit > 0 && cfg.DefaultLimit > cfg.MaxLimit {
		cfg.DefaultLimit = cfg.MaxLimit
	}
	return cfg
}

func cursorPaginationLimit(c *gin.Context, defaultLimit int) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return defaultLimit, true
	}

	limit, err := strconv.Atoi(raw)
	if err != nil {
		httpx.Error(c, apperr.BadRequest("invalid_query", "limit must be an integer"))
		return 0, false
	}
	if limit <= 0 {
		return defaultLimit, true
	}
	return limit, true
}
