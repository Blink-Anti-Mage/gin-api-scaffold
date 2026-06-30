package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/pkg/response"
)

type ReadinessCheck func(ctx context.Context) error

type Health struct {
	readinessChecks map[string]ReadinessCheck
}

func NewHealth(readinessChecks map[string]ReadinessCheck) *Health {
	return &Health{
		readinessChecks: readinessChecks,
	}
}

func Liveness(c *gin.Context) {
	response.OK(c, gin.H{
		"status": "ok",
	})
}

func Readiness(c *gin.Context) {
	NewHealth(nil).Readiness(c)
}

func (h *Health) Liveness(c *gin.Context) {
	Liveness(c)
}

func (h *Health) Readiness(c *gin.Context) {
	if len(h.readinessChecks) == 0 {
		response.OK(c, gin.H{
			"status": "ready",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{}
	for name, check := range h.readinessChecks {
		if err := check(ctx); err != nil {
			response.Error(c, apperr.Wrap(err, http.StatusServiceUnavailable, "not_ready", "service is not ready"))
			return
		}
		checks[name] = "ok"
	}

	response.OK(c, gin.H{
		"status": "ready",
		"checks": checks,
	})
}
