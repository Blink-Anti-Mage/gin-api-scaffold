package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/apperr"
)

func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			Error(c, apperr.PayloadTooLarge())
			return false
		}
		Error(c, apperr.BadRequest("invalid_request", err.Error()))
		return false
	}
	return true
}
