package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wildman-service/internal/domain/catalog"
)

func (s *ObservationStore) CreateResolutionRequest(
	ctx context.Context,
	request catalog.ResolutionRequest,
) (catalog.ResolutionRequest, bool, error) {
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return catalog.ResolutionRequest{}, false, fmt.Errorf("begin resolution request creation: %w", err)
	}
	defer transaction.Rollback()

	result, err := transaction.ExecContext(ctx, `
		INSERT INTO resolution_requests (
			id, client_id, observation_id, idempotency_key, status, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (client_id, idempotency_key) DO NOTHING
	`,
		request.ID,
		request.ClientID,
		request.ObservationID,
		request.IdempotencyKey,
		request.Status,
		request.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return catalog.ResolutionRequest{}, false, fmt.Errorf("insert resolution request: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return catalog.ResolutionRequest{}, false, fmt.Errorf("read resolution insertion result: %w", err)
	}

	stored, err := scanResolutionRequest(transaction.QueryRowContext(ctx, `
		SELECT id, client_id, observation_id, idempotency_key, status,
			last_error_code, created_at, started_at, finished_at
		FROM resolution_requests
		WHERE client_id = ? AND idempotency_key = ?
	`, request.ClientID, request.IdempotencyKey))
	if err != nil {
		return catalog.ResolutionRequest{}, false, fmt.Errorf("query resolution request: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return catalog.ResolutionRequest{}, false, fmt.Errorf("commit resolution request creation: %w", err)
	}
	return stored, rowsAffected == 1, nil
}

func (s *ObservationStore) GetResolutionRequest(
	ctx context.Context,
	clientID string,
	requestID string,
) (catalog.ResolutionRequest, bool, error) {
	request, err := scanResolutionRequest(s.database.QueryRowContext(ctx, `
		SELECT id, client_id, observation_id, idempotency_key, status,
			last_error_code, created_at, started_at, finished_at
		FROM resolution_requests
		WHERE id = ? AND client_id = ?
	`, requestID, clientID))
	if err == sql.ErrNoRows {
		return catalog.ResolutionRequest{}, false, nil
	}
	if err != nil {
		return catalog.ResolutionRequest{}, false, fmt.Errorf("query resolution request by client: %w", err)
	}
	return request, true, nil
}

func scanResolutionRequest(scanner rowScanner) (catalog.ResolutionRequest, error) {
	var request catalog.ResolutionRequest
	var status string
	var lastErrorCode, startedAt, finishedAt sql.NullString
	var createdAt string
	if err := scanner.Scan(
		&request.ID,
		&request.ClientID,
		&request.ObservationID,
		&request.IdempotencyKey,
		&status,
		&lastErrorCode,
		&createdAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return catalog.ResolutionRequest{}, err
	}
	request.Status = catalog.ResolutionStatus(status)
	request.LastErrorCode = lastErrorCode.String
	var err error
	if request.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return catalog.ResolutionRequest{}, fmt.Errorf("parse resolution creation time: %w", err)
	}
	if request.StartedAt, err = parseOptionalTime(startedAt); err != nil {
		return catalog.ResolutionRequest{}, fmt.Errorf("parse resolution start time: %w", err)
	}
	if request.FinishedAt, err = parseOptionalTime(finishedAt); err != nil {
		return catalog.ResolutionRequest{}, fmt.Errorf("parse resolution finish time: %w", err)
	}
	return request, nil
}

func parseOptionalTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
