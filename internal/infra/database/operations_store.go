package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wildman-service/internal/app/operations"
	"wildman-service/internal/app/provider"
)

type OperationsStore struct{ database *DB }

func NewOperationsStore(database *DB) *OperationsStore {
	return &OperationsStore{database: database}
}

func (store *OperationsStore) ProviderMetrics(ctx context.Context) (provider.MetricsSnapshot, error) {
	metrics := provider.MetricsSnapshot{Errors: make(map[string]uint64)}
	var errorsJSON string
	err := store.database.QueryRowContext(ctx, `SELECT cache_lookups, cache_hits, negative_hits, coalesced_requests, provider_requests, errors_json FROM provider_metrics WHERE scope = 'worker'`).Scan(&metrics.CacheLookups, &metrics.CacheHits, &metrics.NegativeHits, &metrics.CoalescedRequests, &metrics.ProviderRequests, &errorsJSON)
	if err == sql.ErrNoRows {
		return metrics, nil
	}
	if err != nil {
		return metrics, err
	}
	if err := json.Unmarshal([]byte(errorsJSON), &metrics.Errors); err != nil {
		return metrics, err
	}
	return metrics, nil
}

func (store *OperationsStore) ListResolutionSummaries(ctx context.Context, limit int) ([]operations.ResolutionSummary, error) {
	rows, err := store.database.QueryContext(ctx, `
		SELECT r.id, c.name, r.status, r.created_at, r.finished_at
		FROM resolution_requests AS r
		JOIN client_installations AS c ON c.id = r.client_id
		ORDER BY r.created_at DESC, r.id DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query resolution summaries: %w", err)
	}
	defer rows.Close()
	items := make([]operations.ResolutionSummary, 0)
	for rows.Next() {
		var item operations.ResolutionSummary
		var createdAt string
		var finishedAt sql.NullString
		if err := rows.Scan(&item.ID, &item.ClientName, &item.Status, &createdAt, &finishedAt); err != nil {
			return nil, err
		}
		if item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
			return nil, err
		}
		if item.FinishedAt, err = parseOptionalTime(finishedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (store *OperationsStore) PurgeExpiredData(ctx context.Context, actorUserID string, now time.Time) (operations.RetentionResult, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return operations.RetentionResult{}, err
	}
	defer transaction.Rollback()
	resolutionCutoff := now.AddDate(0, 0, -180).Format(time.RFC3339Nano)
	if _, err := transaction.ExecContext(ctx, `DELETE FROM resolution_reviews WHERE request_id IN (SELECT id FROM resolution_requests WHERE finished_at IS NOT NULL AND finished_at < ?)`, resolutionCutoff); err != nil {
		return operations.RetentionResult{}, err
	}
	if _, err := transaction.ExecContext(ctx, `DELETE FROM resolution_candidates WHERE request_id IN (SELECT id FROM resolution_requests WHERE finished_at IS NOT NULL AND finished_at < ?)`, resolutionCutoff); err != nil {
		return operations.RetentionResult{}, err
	}
	deletedRequests, err := transaction.ExecContext(ctx, `DELETE FROM resolution_requests WHERE finished_at IS NOT NULL AND finished_at < ?`, resolutionCutoff)
	if err != nil {
		return operations.RetentionResult{}, err
	}
	deletedObservations, err := transaction.ExecContext(ctx, `DELETE FROM track_observations WHERE updated_at < ? AND NOT EXISTS (SELECT 1 FROM resolution_requests WHERE observation_id = track_observations.id)`, resolutionCutoff)
	if err != nil {
		return operations.RetentionResult{}, err
	}
	deletedSources, err := transaction.ExecContext(ctx, `DELETE FROM source_observations WHERE expires_at IS NOT NULL AND expires_at < ? AND NOT EXISTS (SELECT 1 FROM resolution_candidates WHERE source_observation_id = source_observations.id)`, now.AddDate(0, 0, -30).Format(time.RFC3339Nano))
	if err != nil {
		return operations.RetentionResult{}, err
	}
	deletedAudits, err := transaction.ExecContext(ctx, `DELETE FROM audit_events WHERE created_at < ?`, now.AddDate(-1, 0, 0).Format(time.RFC3339Nano))
	if err != nil {
		return operations.RetentionResult{}, err
	}
	result := operations.RetentionResult{}
	result.Resolutions, _ = deletedRequests.RowsAffected()
	result.Observations, _ = deletedObservations.RowsAffected()
	result.SourceObservations, _ = deletedSources.RowsAffected()
	result.AuditEvents, _ = deletedAudits.RowsAffected()
	if err := insertAuditEvent(ctx, transaction, actorUserID, "retention.purged", "system", "retention", now); err != nil {
		return operations.RetentionResult{}, err
	}
	if err := transaction.Commit(); err != nil {
		return operations.RetentionResult{}, err
	}
	return result, nil
}
