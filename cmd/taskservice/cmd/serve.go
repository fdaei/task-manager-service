package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	nethttp "net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"taskservice/internal/config"
	delivery "taskservice/internal/delivery/http"
	"taskservice/internal/domain/task"
	"taskservice/internal/domain/user"
	"taskservice/internal/infrastructure/postgres"
	redisCache "taskservice/internal/infrastructure/redis"
	"taskservice/internal/observability"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the taskservice",
		RunE:  runServe,
	}
	skipMigrations bool
)

func init() {
	serveCmd.Flags().BoolVar(&skipMigrations, "skip-migrations", false, "Skip running database migrations on startup")
	rootCmd.AddCommand(serveCmd)
}

func runServe(_ *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	shutdownTracing, err := observability.SetupTracing(ctx, observability.TracingConfig{
		ServiceName: cfg.ServiceName,
	})
	if err != nil {
		return err
	}
	defer shutdownTracing(context.Background())

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	if err := maybeRunMigrations(ctx, pool, skipMigrations); err != nil {
		return err
	}

	cache, cacheCloser := setupCache(ctx, cfg)
	if cacheCloser != nil {
		defer cacheCloser()
	}

	userRepo := postgres.NewUserRepository(pool)
	validator := task.NewValidator(userRepo)
	taskRepo := postgres.NewTaskRepository(pool)

	metrics := observability.NewMetrics(nil)
	taskService := task.NewService(taskRepo, validator, cache)
	userService := user.NewService(userRepo)
	handler := delivery.NewHandler(taskService, userService, metrics)
	router := delivery.NewRouter(handler, cfg.EnablePprof)

	srv := &nethttp.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: router,
	}

	go gracefulShutdown(ctx, srv)

	log.Printf("taskservice listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
		return err
	}

	return nil
}

func setupCache(ctx context.Context, cfg config.Config) (task.Cache, func() error) {
	if cfg.RedisAddr == "" {
		return nil, nil
	}

	rc := redisCache.NewTaskCache(cfg.RedisAddr, 30*time.Second)
	if err := rc.WarmUp(ctx); err != nil {
		log.Printf("redis not available: %v (continuing without cache)", err)
		return nil, nil
	}

	return rc, rc.Close
}

func gracefulShutdown(ctx context.Context, srv *nethttp.Server) {
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
