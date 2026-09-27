package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"be-remote-device/internal/config"
	"be-remote-device/internal/database"
	deliveryHttp "be-remote-device/internal/delivery/http"
	"be-remote-device/internal/repository"
	"be-remote-device/internal/service"
	"be-remote-device/pkg/logger"
)

func main() {
	cfg := config.LoadConfig()
	logger.Init(cfg.LogLevel)

	slog.Info("Starting service...", "app_name", cfg.AppName, "environment", cfg.AppEnv)

	// Database Connection & Repositories
	var actionLogRepo repository.ActionLogRepository
	var userRepo repository.UserRepository

	db, err := database.InitDB(cfg)
	if err != nil {
		slog.Warn("Database initialization failed. Using in-memory repositories as fallback.", "error", err)
		actionLogRepo = repository.NewMemoryActionLogRepository()
		userRepo = repository.NewMemoryUserRepository()
	} else {
		defer db.Close()
		actionLogRepo = repository.NewPostgresActionLogRepository(db)
		userRepo = repository.NewPostgresUserRepository(db)
	}

	// Action Log Service
	actionLogSvc := service.NewActionLogService(actionLogRepo)
	defer actionLogSvc.Stop()

	// Auth Service
	authSvc := service.NewAuthService(cfg, userRepo)

	// Device Service & Repositories
	deviceRepo := repository.NewMemoryDeviceRepository()
	deviceSvc := service.NewDeviceService(deviceRepo)

	// HTTP Handlers & Router
	deviceHandler := deliveryHttp.NewDeviceHandler(deviceSvc)
	logHandler := deliveryHttp.NewLogHandler(actionLogSvc)
	authHandler := deliveryHttp.NewAuthHandler(authSvc)
	router := deliveryHttp.NewRouter(cfg, deviceHandler, logHandler, authHandler, actionLogSvc, authSvc, deviceSvc)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server run context for graceful shutdown
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, shutdownCancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer shutdownCancel()

		go func() {
			<-shutdownCtx.Done()
			if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				slog.Error("Graceful shutdown timed out.. forcing exit.")
				os.Exit(1)
			}
		}()

		slog.Info("Shutting down server gracefully...")
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Server shutdown failed", "error", err)
		}
		serverStopCtx()
	}()

	slog.Info(fmt.Sprintf("Server running on port %s (http://localhost:%s)", cfg.Port, cfg.Port))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}

	<-serverCtx.Done()
	slog.Info("Server stopped cleanly")
}
