package operations

import (
	"context"
	"fmt"
	"time"

	"wildman-service/internal/app/provider"
)

type ResolutionSummary struct {
	ID         string     `json:"id"`
	ClientName string     `json:"clientName"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

type Store interface {
	ListResolutionSummaries(ctx context.Context, limit int) ([]ResolutionSummary, error)
	PurgeExpiredData(ctx context.Context, actorUserID string, now time.Time) (RetentionResult, error)
	ProviderMetrics(ctx context.Context) (provider.MetricsSnapshot, error)
}

type RetentionResult struct {
	Resolutions        int64 `json:"resolutions"`
	Observations       int64 `json:"observations"`
	SourceObservations int64 `json:"sourceObservations"`
	AuditEvents        int64 `json:"auditEvents"`
}

type ProviderSummary struct {
	Name       string                   `json:"name"`
	Configured bool                     `json:"configured"`
	Metrics    provider.MetricsSnapshot `json:"metrics"`
}

type Service struct {
	store              Store
	providerConfigured bool
}

func NewService(store Store, providerConfigured bool, metrics *provider.Metrics) *Service {
	return &Service{store: store, providerConfigured: providerConfigured}
}

func (service *Service) Resolutions(ctx context.Context) ([]ResolutionSummary, error) {
	items, err := service.store.ListResolutionSummaries(ctx, 100)
	if err != nil {
		return nil, fmt.Errorf("list resolution summaries: %w", err)
	}
	return items, nil
}

func (service *Service) Provider(ctx context.Context) (ProviderSummary, error) {
	metrics, err := service.store.ProviderMetrics(ctx)
	if err != nil {
		return ProviderSummary{}, fmt.Errorf("read provider metrics: %w", err)
	}
	return ProviderSummary{Name: "musicbrainz + wikidata + acoustid", Configured: service.providerConfigured, Metrics: metrics}, nil
}

func (service *Service) Purge(ctx context.Context, actorUserID string) (RetentionResult, error) {
	result, err := service.store.PurgeExpiredData(ctx, actorUserID, time.Now().UTC())
	if err != nil {
		return RetentionResult{}, fmt.Errorf("purge expired data: %w", err)
	}
	return result, nil
}
