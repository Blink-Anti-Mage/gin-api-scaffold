package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/app/example/handler"
	"github.com/example/gin-api-scaffold/app/example/service"
	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/config"
	"github.com/example/gin-api-scaffold/internal/handlers"
	"github.com/example/gin-api-scaffold/internal/httpx"
	"github.com/example/gin-api-scaffold/internal/middleware"
)

type RouterDeps struct {
	Config        config.Config
	Logger        *slog.Logger
	Database      Pinger
	UsersService  *service.UsersService
	ReadinessName string
}

type RouteDeps struct {
	UsersService *service.UsersService
}

type Pinger interface {
	Ping(ctx context.Context) error
}

func NewRouter(deps RouterDeps) *gin.Engine {
	switch {
	case deps.Config.App.IsProduction():
		gin.SetMode(gin.ReleaseMode)
	case deps.Config.App.IsTest():
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.HandleMethodNotAllowed = true

	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(deps.Logger))
	router.Use(gin.Recovery())
	router.Use(middleware.CORS(deps.Config.CORS))
	router.Use(middleware.BodySizeLimit(deps.Config.HTTP.MaxBodyBytes))

	readinessChecks := map[string]handlers.ReadinessCheck{}
	if deps.Database != nil {
		name := deps.ReadinessName
		if name == "" {
			name = "database"
		}
		readinessChecks[name] = deps.Database.Ping
	}
	health := handlers.NewHealth(readinessChecks)
	router.GET("/healthz", health.Liveness)
	router.GET("/readyz", health.Readiness)

	v1 := router.Group("/api/v1")
	v1.Use(middleware.RateLimit(deps.Config.RateLimit))
	if deps.Config.Auth.Enabled {
		v1.Use(middleware.JWT(deps.Config.Auth))
	}
	RegisterRoutes(v1, RouteDeps{
		UsersService: deps.UsersService,
	})

	router.NoRoute(func(c *gin.Context) {
		httpx.Error(c, apperr.NotFound("route"))
	})
	router.NoMethod(func(c *gin.Context) {
		httpx.Error(c, apperr.New(http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed"))
	})

	return router
}

func RegisterRoutes(router *gin.RouterGroup, deps RouteDeps) {
	if deps.UsersService != nil {
		registerUserRoutes(router, deps.UsersService)
	}
}

func registerUserRoutes(router *gin.RouterGroup, usersService *service.UsersService) {
	usersHandler := handler.NewUsersHandler(usersService)

	users := router.Group("/users")
	users.GET("", usersHandler.List)
	users.POST("", usersHandler.Create)
	users.GET("/stats", usersHandler.Stats)
	users.GET("/:id", usersHandler.Get)
}
