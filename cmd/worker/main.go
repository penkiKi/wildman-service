package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	appprovider "wildman-service/internal/app/provider"
	appresolution "wildman-service/internal/app/resolution"
	appworker "wildman-service/internal/app/worker"
	"wildman-service/internal/config"
	"wildman-service/internal/infra/database"
	infraprovider "wildman-service/internal/infra/provider"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel()}))
	if cfg.DatabaseURL == "" {
		logger.Error("worker requires WILDMAN_DATABASE_URL")
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("worker database initialization failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	observationStore := database.NewObservationStore(db)
	providers := make([]appprovider.Provider, 0)
	if cfg.ProviderContact != "" {
		if musicBrainz, err := infraprovider.NewMusicBrainz("0.1.0", cfg.ProviderContact); err == nil {
			providers = append(providers, appprovider.NewCachedProvider(musicBrainz, 0, 0))
		}
		if wikidata, err := infraprovider.NewWikidata("0.1.0", cfg.ProviderContact); err == nil {
			providers = append(providers, appprovider.NewCachedProvider(wikidata, 0, 0))
		}
		if cfg.AcoustIDAPIKey != "" {
			if acoustID, err := infraprovider.NewAcoustID("0.1.0", cfg.ProviderContact, cfg.AcoustIDAPIKey); err == nil {
				providers = append(providers, appprovider.NewCachedProvider(acoustID, 0, 0))
			}
		}
	}
	service := appworker.NewService(database.NewWorkerStore(db), database.NewCatalogStore(db), appresolution.NewService(observationStore), providers, appprovider.NewSourceRecorder(database.NewSourceObservationStore(db)))
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		processed, err := service.RunOnce(ctx)
		if err != nil {
			logger.Error("resolution processing failed", "error", err)
		}
		if processed {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
