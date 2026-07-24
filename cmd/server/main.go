package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appaccount "wildman-service/internal/app/account"
	appauth "wildman-service/internal/app/auth"
	appclient "wildman-service/internal/app/client"
	appoperations "wildman-service/internal/app/operations"
	appprovider "wildman-service/internal/app/provider"
	appresolution "wildman-service/internal/app/resolution"
	appsystem "wildman-service/internal/app/system"
	"wildman-service/internal/config"
	"wildman-service/internal/httpserver"
	"wildman-service/internal/infra/database"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel()}))
	slog.SetDefault(logger)
	if cfg.DatabaseURL == "" {
		logger.Error("server requires WILDMAN_DATABASE_URL")
		os.Exit(1)
	}

	startupContext, cancelStartup := context.WithTimeout(context.Background(), 15*time.Second)
	db, err := database.Open(startupContext, cfg.DatabaseURL)
	cancelStartup()
	if err != nil {
		logger.Error("database initialization failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	authService := appauth.NewService(database.NewAuthStore(db))
	clientService := appclient.NewService(database.NewClientStore(db))
	resolutionService := appresolution.NewService(database.NewObservationStore(db))
	operationsService := appoperations.NewService(database.NewOperationsStore(db), cfg.ProviderContact != "", appprovider.NewMetrics())
	accountService := appaccount.NewService(database.NewAccountStore(db))
	systemService := appsystem.NewService(db, cfg)
	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           httpserver.NewRouter(cfg, logger, systemService, authService, clientService, resolutionService, operationsService, accountService),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("wildman service started", "address", server.Addr, "environment", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownSignal.Done()
	stop()

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("wildman service stopped")
}
