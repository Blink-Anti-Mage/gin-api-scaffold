package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/example/gin-api-scaffold/app/example/repository"
	"github.com/example/gin-api-scaffold/app/example/service"
	"github.com/example/gin-api-scaffold/internal/config"
	"github.com/example/gin-api-scaffold/internal/platform/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	cfg        config.Config
	logger     *slog.Logger
	httpServer *http.Server
	postgres   *pgxpool.Pool
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = slog.Default()
	}

	repos, postgres, err := buildRepositories(cfg, logger)
	if err != nil {
		return nil, err
	}

	usersService := service.NewUsersService(repos.Users)
	router := NewRouter(RouterDeps{
		Config:        cfg,
		Logger:        logger,
		Database:      postgres,
		UsersService:  usersService,
		ReadinessName: "postgres",
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTP.Addr(),
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	return &App{
		cfg:        cfg,
		logger:     logger,
		httpServer: httpServer,
		postgres:   postgres,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.close()

	errCh := make(chan error, 1)

	go func() {
		a.logger.Info("http_server_starting", "addr", a.httpServer.Addr)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
		defer cancel()

		a.logger.Info("http_server_shutting_down")
		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return <-errCh
	}
}

func buildRepositories(cfg config.Config, logger *slog.Logger) (repository.Repositories, *pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Database.ConnectTimeout)
	defer cancel()

	pool, err := database.OpenPostgres(ctx, cfg.Database)
	if err != nil {
		return repository.Repositories{}, nil, err
	}
	logger.Info("postgres_connected")

	return repository.Repositories{
		Users: repository.NewPostgresUsersRepository(pool),
	}, pool, nil
}

func (a *App) close() {
	if a.postgres != nil {
		a.postgres.Close()
		a.logger.Info("postgres_pool_closed")
	}
}
