package app

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/config"
	authcontroller "github.com/example/gin-api-scaffold/internal/controllers/auth"
	"github.com/example/gin-api-scaffold/internal/controllers/home"
	usercontroller "github.com/example/gin-api-scaffold/internal/controllers/user"
	"github.com/example/gin-api-scaffold/internal/middleware"
	authservice "github.com/example/gin-api-scaffold/internal/services/auth"
	userservice "github.com/example/gin-api-scaffold/internal/services/user"
	"github.com/example/gin-api-scaffold/pkg/response"
)

type RouterDeps struct {
	Config        config.Config
	Logger        *slog.Logger
	Database      Pinger
	UsersService  *userservice.UsersService
	AuthService   *authservice.AuthService
	ReadinessName string
}

type RouteDeps struct {
	UsersService *userservice.UsersService
	AuthService  *authservice.AuthService
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

	readinessChecks := map[string]home.ReadinessCheck{}
	if deps.Database != nil {
		name := deps.ReadinessName
		if name == "" {
			name = "database"
		}
		readinessChecks[name] = deps.Database.Ping
	}
	health := home.NewHealth(readinessChecks)
	router.GET("/healthz", health.Liveness)
	router.GET("/readyz", health.Readiness)

	v1 := router.Group("/api/v1")
	v1.Use(middleware.RateLimit(deps.Config.RateLimit))

	if deps.AuthService != nil {
		registerPublicAuthRoutes(v1, deps.AuthService)
	}

	protected := v1.Group("")
	if deps.Config.Auth.Enabled {
		protected.Use(middleware.JWT(deps.Config.Auth))
		protected.Use(middleware.RejectRevokedJWT(deps.AuthService))
	}
	RegisterRoutes(protected, RouteDeps{
		UsersService: deps.UsersService,
		AuthService:  deps.AuthService,
	})

	router.NoRoute(func(c *gin.Context) {
		response.Error(c, apperr.NotFound("route"))
	})
	router.NoMethod(func(c *gin.Context) {
		response.Error(c, apperr.New(http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed"))
	})

	return router
}

func RegisterRoutes(router *gin.RouterGroup, deps RouteDeps) {
	if deps.AuthService != nil {
		registerProtectedAuthRoutes(router, deps.AuthService)
	}
	if deps.UsersService != nil {
		registerUserRoutes(router, deps.UsersService)
	}
}

func registerPublicAuthRoutes(router *gin.RouterGroup, authService *authservice.AuthService) {
	authHandler := authcontroller.NewAuthHandler(authService)

	auth := router.Group("/auth")
	auth.POST("/login", authHandler.Login)
}

func registerProtectedAuthRoutes(router *gin.RouterGroup, authService *authservice.AuthService) {
	authHandler := authcontroller.NewAuthHandler(authService)

	auth := router.Group("/auth")
	auth.POST("/logout", authHandler.Logout)
	auth.GET("/me", authHandler.Me)
}

func registerUserRoutes(router *gin.RouterGroup, usersService *userservice.UsersService) {
	usersHandler := usercontroller.NewUsersHandler(usersService)

	users := router.Group("/users")
	users.GET("", middleware.CursorPagination(middleware.CursorPaginationConfig{
		DefaultLimit: userservice.DefaultUsersListLimit,
		MaxLimit:     userservice.MaxUsersListLimit,
	}), usersHandler.List)
	users.POST("", usersHandler.Create)
	users.GET("/stats", usersHandler.Stats)
	users.GET("/:id", usersHandler.Get)
	users.PUT("/:id", usersHandler.Update)
	users.DELETE("/:id", usersHandler.Delete)
}
