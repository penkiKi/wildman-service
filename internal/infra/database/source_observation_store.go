package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wildman-service/internal/domain/catalog"
)

type SourceObservationStore struct {
	database *DB
}

func NewSourceObservationStore(database *DB) *SourceObservationStore {
	return &SourceObservationStore{database: database}
}

func (store *SourceObservationStore) CreateSourceObservation(
	ctx context.Context,
	observation catalog.SourceObservation,
) (catalog.SourceObservation, bool, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return catalog.SourceObservation{}, false, fmt.Errorf("begin source observation creation: %w", err)
	}
	defer transaction.Rollback()
	result, err := transaction.ExecContext(ctx, `
		INSERT INTO source_observations (
			id, provider, entity_type, external_id, canonical_entity_id,
			payload_json, payload_hash, fetched_at, expires_at, adapter_version
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (provider, entity_type, external_id, payload_hash) DO NOTHING
	`,
		observation.ID,
		observation.Provider,
		observation.EntityType,
		observation.ExternalID,
		nullableString(observation.CanonicalEntityID),
		string(observation.PayloadJSON),
		observation.PayloadHash,
		observation.FetchedAt.UTC().Format(time.RFC3339Nano),
		nullableTime(observation.ExpiresAt),
		observation.AdapterVersion,
	)
	if err != nil {
		return catalog.SourceObservation{}, false, fmt.Errorf("insert source observation: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return catalog.SourceObservation{}, false, fmt.Errorf("read source observation insertion result: %w", err)
	}
	stored, err := scanSourceObservation(transaction.QueryRowContext(ctx, `
		SELECT id, provider, entity_type, external_id, canonical_entity_id,
			payload_json, payload_hash, fetched_at, expires_at, adapter_version
		FROM source_observations
		WHERE provider = ? AND entity_type = ? AND external_id = ? AND payload_hash = ?
	`, observation.Provider, observation.EntityType, observation.ExternalID, observation.PayloadHash))
	if err != nil {
		return catalog.SourceObservation{}, false, fmt.Errorf("query source observation: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return catalog.SourceObservation{}, false, fmt.Errorf("commit source observation creation: %w", err)
	}
	return stored, rowsAffected == 1, nil
}

func scanSourceObservation(scanner rowScanner) (catalog.SourceObservation, error) {
	var observation catalog.SourceObservation
	var entityType string
	var canonicalEntityID, expiresAt sql.NullString
	var payloadJSON, fetchedAt string
	if err := scanner.Scan(
		&observation.ID,
		&observation.Provider,
		&entityType,
		&observation.ExternalID,
		&canonicalEntityID,
		&payloadJSON,
		&observation.PayloadHash,
		&fetchedAt,
		&expiresAt,
		&observation.AdapterVersion,
	); err != nil {
		return catalog.SourceObservation{}, err
	}
	observation.EntityType = catalog.SourceEntityType(entityType)
	observation.CanonicalEntityID = canonicalEntityID.String
	observation.PayloadJSON = []byte(payloadJSON)
	var err error
	if observation.FetchedAt, err = time.Parse(time.RFC3339Nano, fetchedAt); err != nil {
		return catalog.SourceObservation{}, fmt.Errorf("parse source observation fetch time: %w", err)
	}
	if observation.ExpiresAt, err = parseOptionalTime(expiresAt); err != nil {
		return catalog.SourceObservation{}, fmt.Errorf("parse source observation expiration: %w", err)
	}
	return observation, nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
