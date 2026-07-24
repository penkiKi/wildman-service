package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	domainclient "wildman-service/internal/domain/client"
)

type ClientStore struct {
	database *DB
}

func NewClientStore(database *DB) *ClientStore {
	return &ClientStore{database: database}
}

func (s *ClientStore) CreateClientInstallation(
	ctx context.Context,
	installation domainclient.ClientInstallation,
) error {
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	_, err = transaction.ExecContext(ctx, `
		INSERT INTO client_installations (
			id, name, token_prefix, token_hash, status, created_by_user_id, account_id, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		installation.ID,
		installation.Name,
		installation.TokenPrefix,
		installation.TokenHash,
		installation.Status,
		installation.CreatedByUserID,
		nullableString(installation.AccountID),
		installation.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert client installation: %w", err)
	}
	if err := insertAuditEvent(ctx, transaction, installation.CreatedByUserID, "client.created", "client", installation.ID, installation.CreatedAt); err != nil {
		return err
	}
	return transaction.Commit()
}

func (s *ClientStore) ListClientInstallations(ctx context.Context) ([]domainclient.ClientInstallation, error) {
	rows, err := s.database.QueryContext(ctx, `
		SELECT id, name, token_prefix, token_hash, status, created_by_user_id, account_id,
			last_seen_at, revoked_at, created_at
		FROM client_installations
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query client installations: %w", err)
	}
	defer rows.Close()

	installations := make([]domainclient.ClientInstallation, 0)
	for rows.Next() {
		installation, err := scanClientInstallation(rows)
		if err != nil {
			return nil, err
		}
		installations = append(installations, installation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate client installations: %w", err)
	}
	return installations, nil
}

func (s *ClientStore) RevokeClientInstallation(
	ctx context.Context,
	installationID string,
	actorUserID string,
	revokedAt time.Time,
) (domainclient.ClientInstallation, bool, error) {
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return domainclient.ClientInstallation{}, false, fmt.Errorf("begin client revocation: %w", err)
	}
	defer transaction.Rollback()

	result, err := transaction.ExecContext(ctx, `
		UPDATE client_installations
		SET status = ?, revoked_at = ?
		WHERE id = ? AND status = ? AND revoked_at IS NULL
	`,
		domainclient.InstallationStatusRevoked,
		revokedAt.UTC().Format(time.RFC3339Nano),
		installationID,
		domainclient.InstallationStatusActive,
	)
	if err != nil {
		return domainclient.ClientInstallation{}, false, fmt.Errorf("update client installation status: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domainclient.ClientInstallation{}, false, err
	}
	if rowsAffected == 1 {
		if err := insertAuditEvent(ctx, transaction, actorUserID, "client.revoked", "client", installationID, revokedAt); err != nil {
			return domainclient.ClientInstallation{}, false, err
		}
	}

	installation, err := scanClientInstallation(transaction.QueryRowContext(ctx, `
		SELECT id, name, token_prefix, token_hash, status, created_by_user_id, account_id,
			last_seen_at, revoked_at, created_at
		FROM client_installations
		WHERE id = ?
	`, installationID))
	if err == sql.ErrNoRows {
		return domainclient.ClientInstallation{}, false, nil
	}
	if err != nil {
		return domainclient.ClientInstallation{}, false, err
	}
	if err := transaction.Commit(); err != nil {
		return domainclient.ClientInstallation{}, false, fmt.Errorf("commit client revocation: %w", err)
	}
	return installation, true, nil
}

func (s *ClientStore) DeleteClientInstallation(ctx context.Context, installationID, expectedName, actorUserID string) (bool, bool, error) {
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return false, false, err
	}
	defer transaction.Rollback()
	var name, status string
	if err := transaction.QueryRowContext(ctx, "SELECT name, status FROM client_installations WHERE id = ?", installationID).Scan(&name, &status); err == sql.ErrNoRows {
		return false, false, nil
	} else if err != nil {
		return false, false, err
	}
	if status != string(domainclient.InstallationStatusRevoked) || name != expectedName {
		return true, false, nil
	}
	statements := []string{
		"DELETE FROM resolution_reviews WHERE client_id = ?",
		"DELETE FROM resolution_candidates WHERE request_id IN (SELECT id FROM resolution_requests WHERE client_id = ?)",
		"DELETE FROM resolution_requests WHERE client_id = ?",
		"DELETE FROM track_observations WHERE client_id = ?",
		"DELETE FROM client_installations WHERE id = ?",
	}
	for _, statement := range statements {
		if _, err := transaction.ExecContext(ctx, statement, installationID); err != nil {
			return true, false, err
		}
	}
	now := time.Now().UTC()
	if err := insertAuditEvent(ctx, transaction, actorUserID, "client.deleted", "client", installationID, now); err != nil {
		return true, false, err
	}
	if err := transaction.Commit(); err != nil {
		return true, false, err
	}
	return true, true, nil
}

func insertAuditEvent(ctx context.Context, transaction *Tx, actorUserID, action, subjectType, subjectID string, at time.Time) error {
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return fmt.Errorf("generate audit event ID: %w", err)
	}
	_, err := transaction.ExecContext(ctx, `
		INSERT INTO audit_events (id, actor_user_id, action, subject_type, subject_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, hex.EncodeToString(idBytes), nullableString(actorUserID), action, subjectType, subjectID, at.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func (s *ClientStore) FindClientInstallationByToken(
	ctx context.Context,
	tokenPrefix string,
	tokenHash string,
) (domainclient.ClientInstallation, bool, error) {
	installation, err := scanClientInstallation(s.database.QueryRowContext(ctx, `
		SELECT id, name, token_prefix, token_hash, status, created_by_user_id, account_id,
			last_seen_at, revoked_at, created_at
		FROM client_installations
		WHERE token_prefix = ? AND token_hash = ?
	`, tokenPrefix, tokenHash))
	if err == sql.ErrNoRows {
		return domainclient.ClientInstallation{}, false, nil
	}
	if err != nil {
		return domainclient.ClientInstallation{}, false, fmt.Errorf("query client installation by token: %w", err)
	}
	return installation, true, nil
}

func (s *ClientStore) TouchClientInstallation(
	ctx context.Context,
	installationID string,
	lastSeenAt time.Time,
) (bool, error) {
	formatted := lastSeenAt.UTC().Format(time.RFC3339Nano)
	result, err := s.database.ExecContext(ctx, `
		UPDATE client_installations
		SET last_seen_at = CASE
			WHEN last_seen_at IS NULL OR last_seen_at < ? THEN ?
			ELSE last_seen_at
		END
		WHERE id = ? AND status = ? AND revoked_at IS NULL
	`, formatted, formatted, installationID, domainclient.InstallationStatusActive)
	if err != nil {
		return false, fmt.Errorf("update client installation last seen: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read client installation touch result: %w", err)
	}
	return rowsAffected == 1, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanClientInstallation(scanner rowScanner) (domainclient.ClientInstallation, error) {
	var installation domainclient.ClientInstallation
	var status string
	var accountID sql.NullString
	var lastSeenAt, revokedAt sql.NullString
	var createdAt string
	err := scanner.Scan(
		&installation.ID,
		&installation.Name,
		&installation.TokenPrefix,
		&installation.TokenHash,
		&status,
		&installation.CreatedByUserID,
		&accountID,
		&lastSeenAt,
		&revokedAt,
		&createdAt,
	)
	if err != nil {
		return domainclient.ClientInstallation{}, err
	}

	installation.Status = domainclient.InstallationStatus(status)
	installation.AccountID = accountID.String
	if installation.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return domainclient.ClientInstallation{}, fmt.Errorf("parse client installation creation: %w", err)
	}
	if lastSeenAt.Valid {
		parsed, err := time.Parse(time.RFC3339Nano, lastSeenAt.String)
		if err != nil {
			return domainclient.ClientInstallation{}, fmt.Errorf("parse client installation last seen: %w", err)
		}
		installation.LastSeenAt = &parsed
	}
	if revokedAt.Valid {
		parsed, err := time.Parse(time.RFC3339Nano, revokedAt.String)
		if err != nil {
			return domainclient.ClientInstallation{}, fmt.Errorf("parse client installation revocation: %w", err)
		}
		installation.RevokedAt = &parsed
	}
	return installation, nil
}
