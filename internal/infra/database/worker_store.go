package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	appprovider "wildman-service/internal/app/provider"
	"wildman-service/internal/domain/catalog"
)

type WorkerStore struct{ database *DB }

func (store *WorkerStore) SaveProviderMetrics(ctx context.Context, metrics appprovider.MetricsSnapshot, at time.Time) error {
	errorsJSON, err := json.Marshal(metrics.Errors)
	if err != nil {
		return err
	}
	_, err = store.database.ExecContext(ctx, `
		INSERT INTO provider_metrics (scope, cache_lookups, cache_hits, negative_hits, coalesced_requests, provider_requests, errors_json, updated_at)
		VALUES ('worker', ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (scope) DO UPDATE SET cache_lookups = excluded.cache_lookups, cache_hits = excluded.cache_hits,
			negative_hits = excluded.negative_hits, coalesced_requests = excluded.coalesced_requests,
			provider_requests = excluded.provider_requests, errors_json = excluded.errors_json, updated_at = excluded.updated_at
	`, metrics.CacheLookups, metrics.CacheHits, metrics.NegativeHits, metrics.CoalescedRequests, metrics.ProviderRequests, string(errorsJSON), at.Format(time.RFC3339Nano))
	return err
}

func NewWorkerStore(database *DB) *WorkerStore { return &WorkerStore{database: database} }

func (store *WorkerStore) ClaimResolution(ctx context.Context) (catalog.ResolutionRequest, catalog.StoredTrackObservation, bool, error) {
	if store.database.Dialect() != DialectPostgres {
		return catalog.ResolutionRequest{}, catalog.StoredTrackObservation{}, false, fmt.Errorf("independent worker requires PostgreSQL")
	}
	transaction, err := store.database.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return catalog.ResolutionRequest{}, catalog.StoredTrackObservation{}, false, err
	}
	defer transaction.Rollback()
	var requestID string
	err = transaction.QueryRowContext(ctx, `
		SELECT id FROM resolution_requests WHERE status = ?
		ORDER BY created_at, id FOR UPDATE SKIP LOCKED LIMIT 1
	`, catalog.ResolutionStatusQueued).Scan(&requestID)
	if err == sql.ErrNoRows {
		return catalog.ResolutionRequest{}, catalog.StoredTrackObservation{}, false, nil
	}
	if err != nil {
		return catalog.ResolutionRequest{}, catalog.StoredTrackObservation{}, false, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := transaction.ExecContext(ctx, `UPDATE resolution_requests SET status = ?, started_at = ? WHERE id = ?`, catalog.ResolutionStatusMatching, now, requestID); err != nil {
		return catalog.ResolutionRequest{}, catalog.StoredTrackObservation{}, false, err
	}
	request, err := scanResolutionRequest(transaction.QueryRowContext(ctx, `
		SELECT id, client_id, observation_id, idempotency_key, status, last_error_code, created_at, started_at, finished_at
		FROM resolution_requests WHERE id = ?
	`, requestID))
	if err != nil {
		return catalog.ResolutionRequest{}, catalog.StoredTrackObservation{}, false, err
	}
	observation, err := scanTrackObservation(transaction.QueryRowContext(ctx, `
		SELECT id, client_id, client_track_id, file_name, title, artists_json, album,
			disc_number, track_number, duration_ms, format, fingerprint, payload_hash, observed_at, created_at, updated_at
		FROM track_observations WHERE id = ? AND client_id = ?
	`, request.ObservationID, request.ClientID))
	if err != nil {
		return catalog.ResolutionRequest{}, catalog.StoredTrackObservation{}, false, err
	}
	if err := transaction.Commit(); err != nil {
		return catalog.ResolutionRequest{}, catalog.StoredTrackObservation{}, false, err
	}
	return request, observation, true, nil
}

func (store *WorkerStore) FailResolution(ctx context.Context, clientID, requestID, errorCode string) error {
	_, err := store.database.ExecContext(ctx, `
		UPDATE resolution_requests SET status = ?, last_error_code = ?, finished_at = ?
		WHERE id = ? AND client_id = ? AND status = ?
	`, catalog.ResolutionStatusFailed, errorCode, time.Now().UTC().Format(time.RFC3339Nano), requestID, clientID, catalog.ResolutionStatusMatching)
	return err
}
