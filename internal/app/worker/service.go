package worker

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	appcatalog "wildman-service/internal/app/catalog"
	appprovider "wildman-service/internal/app/provider"
	appresolution "wildman-service/internal/app/resolution"
	"wildman-service/internal/domain/catalog"
)

type Store interface {
	ClaimResolution(ctx context.Context) (catalog.ResolutionRequest, catalog.StoredTrackObservation, bool, error)
	FailResolution(ctx context.Context, clientID, requestID, errorCode string) error
	SaveProviderMetrics(ctx context.Context, metrics appprovider.MetricsSnapshot, at time.Time) error
}

type Service struct {
	store       Store
	catalog     appcatalog.Repository
	resolutions *appresolution.Service
	providers   []appprovider.Provider
	sources     *appprovider.SourceRecorder
}

func NewService(store Store, repository appcatalog.Repository, resolutions *appresolution.Service, providers []appprovider.Provider, sources *appprovider.SourceRecorder) *Service {
	return &Service{store: store, catalog: repository, resolutions: resolutions, providers: providers, sources: sources}
}

func (service *Service) RunOnce(ctx context.Context) (bool, error) {
	request, storedObservation, found, err := service.store.ClaimResolution(ctx)
	if err != nil || !found {
		return found, err
	}
	normalized := catalog.NormalizeTrackObservation(storedObservation.Observation)
	recordings, err := service.catalog.FindRecordings(ctx, normalized.Title)
	if err != nil {
		_ = service.store.FailResolution(ctx, request.ClientID, request.ID, "CATALOG_QUERY_FAILED")
		return true, fmt.Errorf("find catalog recordings: %w", err)
	}
	candidates := make([]catalog.Candidate, 0, len(recordings))
	for _, recording := range recordings {
		candidate := catalog.ScoreCandidate(storedObservation.Observation, recording, nil)
		if candidate.Score < 0.45 {
			continue
		}
		candidate.Source = "catalog"
		candidate.Sources = []string{"catalog"}
		candidate.TagPatch = catalog.GenerateTagPatch(storedObservation.Observation, candidate, nil)
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 && len(service.providers) > 0 {
		providerCandidates, successfulProvider, err := service.queryProviders(ctx, storedObservation.Observation)
		if err != nil {
			_ = service.store.FailResolution(ctx, request.ClientID, request.ID, "PROVIDER_QUERY_FAILED")
			return true, err
		}
		if err := service.store.SaveProviderMetrics(ctx, service.providerMetrics(), time.Now().UTC()); err != nil {
			return true, err
		}
		if !successfulProvider && len(providerCandidates) == 0 {
			_ = service.store.FailResolution(ctx, request.ClientID, request.ID, "PROVIDER_UNAVAILABLE")
			return true, fmt.Errorf("all configured providers failed")
		}
		candidates = catalog.MergeCrossSourceCandidates(providerCandidates)
	}
	if err := service.resolutions.CompleteRequest(ctx, request.ClientID, request.ID, candidates); err != nil {
		_ = service.store.FailResolution(ctx, request.ClientID, request.ID, "CANDIDATE_SAVE_FAILED")
		return true, err
	}
	return true, nil
}

func (service *Service) providerMetrics() appprovider.MetricsSnapshot {
	result := appprovider.MetricsSnapshot{Errors: make(map[string]uint64)}
	for _, provider := range service.providers {
		metricsProvider, ok := provider.(interface {
			Metrics() appprovider.MetricsSnapshot
		})
		if !ok {
			continue
		}
		snapshot := metricsProvider.Metrics()
		result.CacheLookups += snapshot.CacheLookups
		result.CacheHits += snapshot.CacheHits
		result.NegativeHits += snapshot.NegativeHits
		result.CoalescedRequests += snapshot.CoalescedRequests
		result.ProviderRequests += snapshot.ProviderRequests
		for kind, count := range snapshot.Errors {
			result.Errors[kind] += count
		}
	}
	return result
}

func (service *Service) queryProviders(ctx context.Context, observation catalog.TrackObservation) ([]catalog.Candidate, bool, error) {
	query := appprovider.RecordingQuery{Title: observation.Title, Artists: observation.Artists, Album: observation.Album, DurationMS: observation.DurationMS, Fingerprint: observation.Fingerprint}
	candidates := make([]catalog.Candidate, 0)
	successful := false
	for _, provider := range service.providers {
		if provider.Name() == "acoustid" && (query.Fingerprint == "" || query.DurationMS == nil) {
			continue
		}
		found, err := provider.SearchRecordings(ctx, query)
		if err != nil {
			continue
		}
		successful = true
		for _, sourceCandidate := range found {
			candidate, err := service.persistProviderCandidate(ctx, provider, observation, sourceCandidate)
			if err != nil {
				return nil, successful, err
			}
			if candidate.Score >= 0.45 {
				candidates = append(candidates, candidate)
			}
		}
	}
	return candidates, successful, nil
}

func (service *Service) persistProviderCandidate(ctx context.Context, provider appprovider.Provider, observation catalog.TrackObservation, source appprovider.RecordingCandidate) (catalog.Candidate, error) {
	artists := make([]catalog.Artist, 0, len(source.Artists))
	for _, name := range source.Artists {
		artist := catalog.Artist{ID: sourceEntityID(provider.Name(), "artist", name), Name: name, NormalizedName: catalog.NormalizeText(name)}
		if err := service.catalog.SaveArtist(ctx, artist); err != nil {
			return catalog.Candidate{}, err
		}
		artists = append(artists, artist)
	}
	recording := catalog.Recording{ID: sourceEntityID(provider.Name(), "recording", source.ExternalID), Title: source.Title, NormalizedTitle: catalog.NormalizeText(source.Title), Artists: artists, DurationMS: source.DurationMS, ISRC: source.ISRC}
	if err := service.catalog.SaveRecording(ctx, recording); err != nil {
		return catalog.Candidate{}, err
	}
	var release *catalog.Release
	if source.ReleaseExternalID != "" {
		value := catalog.Release{ID: sourceEntityID(provider.Name(), "release", source.ReleaseExternalID), Title: source.ReleaseTitle, NormalizedTitle: catalog.NormalizeText(source.ReleaseTitle), Artists: artists}
		if err := service.catalog.SaveRelease(ctx, value); err != nil {
			return catalog.Candidate{}, err
		}
		release = &value
	}
	sourceObservationID := ""
	if service.sources != nil && len(source.RawPayload) > 0 {
		stored, _, err := service.sources.Record(ctx, provider.Name(), provider.AdapterVersion(), catalog.SourceEntityRecording, source.ExternalID, recording.ID, source.RawPayload, 7*24*time.Hour)
		if err != nil {
			return catalog.Candidate{}, err
		}
		sourceObservationID = stored.ID
	}
	candidate := catalog.ScoreCandidate(observation, recording, release)
	candidate.Source, candidate.Sources, candidate.SourceObservationID = provider.Name(), []string{provider.Name()}, sourceObservationID
	candidate.TagPatch = catalog.GenerateTagPatch(observation, candidate, nil)
	return candidate, nil
}

func sourceEntityID(provider, entityType, externalID string) string {
	digest := sha256.Sum256([]byte(provider + ":" + entityType + ":" + externalID))
	return "source:" + base64.RawURLEncoding.EncodeToString(digest[:16])
}
