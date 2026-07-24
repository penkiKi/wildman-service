package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domainauth "wildman-service/internal/domain/auth"
)

type AuthStore struct {
	database *DB
}

func NewAuthStore(database *DB) *AuthStore {
	return &AuthStore{database: database}
}

func (s *AuthStore) IsInitialized(ctx context.Context) (bool, error) {
	var initialized bool
	if err := s.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users)").Scan(&initialized); err != nil {
		return false, fmt.Errorf("query users: %w", err)
	}
	return initialized, nil
}

func (s *AuthStore) CreateInitialAdmin(ctx context.Context, user domainauth.User) (bool, error) {
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin setup transaction: %w", err)
	}
	defer transaction.Rollback()

	var initialized bool
	if err := transaction.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users)").Scan(&initialized); err != nil {
		return false, fmt.Errorf("check setup state: %w", err)
	}
	if initialized {
		return false, nil
	}

	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.Status,
		user.CreatedAt.Format(time.RFC3339Nano),
		user.UpdatedAt.Format(time.RFC3339Nano),
	); err != nil {
		return false, fmt.Errorf("insert initial admin: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return false, fmt.Errorf("commit setup transaction: %w", err)
	}
	return true, nil
}

func (s *AuthStore) FindUserByUsername(ctx context.Context, username string) (domainauth.User, bool, error) {
	var user domainauth.User
	var createdAt, updatedAt string
	err := s.database.QueryRowContext(ctx, `
		SELECT id, username, password_hash, status, created_at, updated_at
		FROM users
		WHERE username = ?
	`, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Status,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return domainauth.User{}, false, nil
	}
	if err != nil {
		return domainauth.User{}, false, fmt.Errorf("query user: %w", err)
	}

	user.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domainauth.User{}, false, fmt.Errorf("parse user created time: %w", err)
	}
	user.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return domainauth.User{}, false, fmt.Errorf("parse user updated time: %w", err)
	}
	return user, true, nil
}

func (s *AuthStore) CreateSession(ctx context.Context, session domainauth.Session) error {
	_, err := s.database.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, last_seen_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.ExpiresAt.Format(time.RFC3339Nano),
		session.LastSeenAt.Format(time.RFC3339Nano),
		session.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *AuthStore) FindSessionByTokenHash(
	ctx context.Context,
	tokenHash string,
) (domainauth.Session, domainauth.User, bool, error) {
	var session domainauth.Session
	var user domainauth.User
	var expiresAt, lastSeenAt, sessionCreatedAt string
	var userCreatedAt, userUpdatedAt string
	var revokedAt sql.NullString
	err := s.database.QueryRowContext(ctx, `
		SELECT
			s.id, s.user_id, s.token_hash, s.expires_at, s.last_seen_at, s.revoked_at, s.created_at,
			u.id, u.username, u.password_hash, u.status, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ?
	`, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&expiresAt,
		&lastSeenAt,
		&revokedAt,
		&sessionCreatedAt,
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Status,
		&userCreatedAt,
		&userUpdatedAt,
	)
	if err == sql.ErrNoRows {
		return domainauth.Session{}, domainauth.User{}, false, nil
	}
	if err != nil {
		return domainauth.Session{}, domainauth.User{}, false, fmt.Errorf("query session: %w", err)
	}

	if session.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt); err != nil {
		return domainauth.Session{}, domainauth.User{}, false, fmt.Errorf("parse session expiry: %w", err)
	}
	if session.LastSeenAt, err = time.Parse(time.RFC3339Nano, lastSeenAt); err != nil {
		return domainauth.Session{}, domainauth.User{}, false, fmt.Errorf("parse session last seen: %w", err)
	}
	if session.CreatedAt, err = time.Parse(time.RFC3339Nano, sessionCreatedAt); err != nil {
		return domainauth.Session{}, domainauth.User{}, false, fmt.Errorf("parse session creation: %w", err)
	}
	if revokedAt.Valid {
		parsed, err := time.Parse(time.RFC3339Nano, revokedAt.String)
		if err != nil {
			return domainauth.Session{}, domainauth.User{}, false, fmt.Errorf("parse session revocation: %w", err)
		}
		session.RevokedAt = &parsed
	}
	if user.CreatedAt, err = time.Parse(time.RFC3339Nano, userCreatedAt); err != nil {
		return domainauth.Session{}, domainauth.User{}, false, fmt.Errorf("parse user created time: %w", err)
	}
	if user.UpdatedAt, err = time.Parse(time.RFC3339Nano, userUpdatedAt); err != nil {
		return domainauth.Session{}, domainauth.User{}, false, fmt.Errorf("parse user updated time: %w", err)
	}

	return session, user, true, nil
}

func (s *AuthStore) TouchSession(ctx context.Context, sessionID string, lastSeenAt time.Time) error {
	if _, err := s.database.ExecContext(ctx, `
		UPDATE sessions SET last_seen_at = ? WHERE id = ? AND revoked_at IS NULL
	`, lastSeenAt.Format(time.RFC3339Nano), sessionID); err != nil {
		return fmt.Errorf("update session last seen: %w", err)
	}
	return nil
}

func (s *AuthStore) RevokeSession(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	if _, err := s.database.ExecContext(ctx, `
		UPDATE sessions SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL
	`, revokedAt.Format(time.RFC3339Nano), tokenHash); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
