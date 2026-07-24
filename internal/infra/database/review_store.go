package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wildman-service/internal/domain/catalog"
)

func (s *ObservationStore) SaveResolutionReview(ctx context.Context, review catalog.ResolutionReview) (catalog.ResolutionReview, bool, error) {
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return catalog.ResolutionReview{}, false, err
	}
	defer transaction.Rollback()
	var requestExists bool
	if err := transaction.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM resolution_requests WHERE id = ? AND client_id = ?)`, review.RequestID, review.ClientID).Scan(&requestExists); err != nil {
		return catalog.ResolutionReview{}, false, err
	}
	if !requestExists {
		return catalog.ResolutionReview{}, false, nil
	}
	if review.RecordingID != "" {
		var candidateExists bool
		if err := transaction.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM resolution_candidates WHERE request_id = ? AND recording_id = ?)`, review.RequestID, review.RecordingID).Scan(&candidateExists); err != nil {
			return catalog.ResolutionReview{}, false, err
		}
		if !candidateExists {
			return catalog.ResolutionReview{}, false, nil
		}
	}
	_, err = transaction.ExecContext(ctx, `
		INSERT INTO resolution_reviews (request_id, client_id, decision, recording_id, writeback_status, writeback_error_code, reviewed_at, writeback_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (request_id) DO UPDATE SET
			decision = excluded.decision, recording_id = excluded.recording_id,
			writeback_status = excluded.writeback_status, writeback_error_code = excluded.writeback_error_code,
			writeback_at = excluded.writeback_at
	`, review.RequestID, review.ClientID, review.Decision, nullableString(review.RecordingID), review.WritebackStatus,
		nullableString(review.WritebackErrorCode), review.ReviewedAt.Format(time.RFC3339Nano), nullableTime(review.WritebackAt))
	if err != nil {
		return catalog.ResolutionReview{}, false, fmt.Errorf("write resolution review: %w", err)
	}
	stored, err := scanResolutionReview(transaction.QueryRowContext(ctx, `
		SELECT request_id, client_id, decision, recording_id, writeback_status, writeback_error_code, reviewed_at, writeback_at
		FROM resolution_reviews WHERE request_id = ? AND client_id = ?
	`, review.RequestID, review.ClientID))
	if err != nil {
		return catalog.ResolutionReview{}, false, err
	}
	if err := transaction.Commit(); err != nil {
		return catalog.ResolutionReview{}, false, err
	}
	return stored, true, nil
}

func scanResolutionReview(scanner rowScanner) (catalog.ResolutionReview, error) {
	var review catalog.ResolutionReview
	var decision, status string
	var recordingID, errorCode, writebackAt sql.NullString
	var reviewedAt string
	if err := scanner.Scan(&review.RequestID, &review.ClientID, &decision, &recordingID, &status, &errorCode, &reviewedAt, &writebackAt); err != nil {
		return review, err
	}
	review.Decision, review.WritebackStatus = catalog.ReviewDecision(decision), catalog.WritebackStatus(status)
	review.RecordingID, review.WritebackErrorCode = recordingID.String, errorCode.String
	var err error
	if review.ReviewedAt, err = time.Parse(time.RFC3339Nano, reviewedAt); err != nil {
		return review, err
	}
	if review.WritebackAt, err = parseOptionalTime(writebackAt); err != nil {
		return review, err
	}
	return review, nil
}
