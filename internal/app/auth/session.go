package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	domainauth "wildman-service/internal/domain/auth"
)

const (
	sessionAbsoluteLifetime = 7 * 24 * time.Hour
	sessionIdleLifetime     = 24 * time.Hour
	sessionTouchInterval    = 5 * time.Minute
)

type LoginResult struct {
	User      domainauth.User
	Token     string
	ExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, username, password string) (LoginResult, error) {
	normalizedUsername, usernameErr := normalizeUsername(username)
	if usernameErr != nil {
		_, _ = verifyPassword(password, s.dummyPasswordHash)
		return LoginResult{}, ErrInvalidCredentials
	}

	user, found, err := s.store.FindUserByUsername(ctx, normalizedUsername)
	if err != nil {
		return LoginResult{}, fmt.Errorf("find user: %w", err)
	}
	passwordHash := s.dummyPasswordHash
	if found {
		passwordHash = user.PasswordHash
	}
	passwordMatches, err := verifyPassword(password, passwordHash)
	if err != nil {
		return LoginResult{}, fmt.Errorf("verify password: %w", err)
	}
	if !found || !passwordMatches || user.Status != domainauth.UserStatusActive {
		return LoginResult{}, ErrInvalidCredentials
	}

	token := randomToken(32)
	now := time.Now().UTC()
	session := domainauth.Session{
		ID:         randomID(),
		UserID:     user.ID,
		TokenHash:  hashToken(token),
		ExpiresAt:  now.Add(sessionAbsoluteLifetime),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	if err := s.store.CreateSession(ctx, session); err != nil {
		return LoginResult{}, fmt.Errorf("create session: %w", err)
	}

	return LoginResult{User: user, Token: token, ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (domainauth.User, error) {
	if token == "" {
		return domainauth.User{}, ErrUnauthenticated
	}

	session, user, found, err := s.store.FindSessionByTokenHash(ctx, hashToken(token))
	if err != nil {
		return domainauth.User{}, fmt.Errorf("find session: %w", err)
	}
	if !found || session.RevokedAt != nil || user.Status != domainauth.UserStatusActive {
		return domainauth.User{}, ErrUnauthenticated
	}

	now := time.Now().UTC()
	if !now.Before(session.ExpiresAt) || now.Sub(session.LastSeenAt) > sessionIdleLifetime {
		_ = s.store.RevokeSession(ctx, session.TokenHash, now)
		return domainauth.User{}, ErrUnauthenticated
	}
	if now.Sub(session.LastSeenAt) >= sessionTouchInterval {
		if err := s.store.TouchSession(ctx, session.ID, now); err != nil {
			return domainauth.User{}, fmt.Errorf("touch session: %w", err)
		}
	}

	return user, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	if err := s.store.RevokeSession(ctx, hashToken(token), time.Now().UTC()); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func randomToken(length int) string {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(value)
}

func hashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
