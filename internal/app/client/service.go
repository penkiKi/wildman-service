package client

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	domainclient "wildman-service/internal/domain/client"
)

const (
	maximumInstallationNameLength = 100
	lastSeenTouchInterval         = 5 * time.Minute
)

var (
	ErrInvalidInstallationName  = errors.New("invalid client installation name")
	ErrInstallationNotFound     = errors.New("client installation not found")
	ErrAuthenticationRequired   = errors.New("client authentication required")
	ErrInstallationRevoked      = errors.New("client installation revoked")
	ErrClientDeletionNotAllowed = errors.New("client deletion is not allowed")
)

type Store interface {
	CreateClientInstallation(ctx context.Context, installation domainclient.ClientInstallation) error
	ListClientInstallations(ctx context.Context) ([]domainclient.ClientInstallation, error)
	RevokeClientInstallation(
		ctx context.Context,
		installationID string,
		actorUserID string,
		revokedAt time.Time,
	) (domainclient.ClientInstallation, bool, error)
	FindClientInstallationByToken(
		ctx context.Context,
		tokenPrefix string,
		tokenHash string,
	) (domainclient.ClientInstallation, bool, error)
	TouchClientInstallation(ctx context.Context, installationID string, lastSeenAt time.Time) (bool, error)
	DeleteClientInstallation(ctx context.Context, installationID, expectedName, actorUserID string) (bool, bool, error)
}

type CreateResult struct {
	Installation domainclient.ClientInstallation
	Token        string
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(
	ctx context.Context,
	name string,
	createdByUserID string,
) (CreateResult, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maximumInstallationNameLength {
		return CreateResult{}, ErrInvalidInstallationName
	}

	token, err := domainclient.IssueToken()
	if err != nil {
		return CreateResult{}, fmt.Errorf("issue client token: %w", err)
	}
	id, err := randomID()
	if err != nil {
		return CreateResult{}, fmt.Errorf("generate client installation ID: %w", err)
	}
	now := time.Now().UTC()
	installation := domainclient.ClientInstallation{
		ID:              id,
		Name:            name,
		TokenPrefix:     token.Prefix,
		TokenHash:       token.Hash,
		Status:          domainclient.InstallationStatusActive,
		CreatedByUserID: createdByUserID,
		CreatedAt:       now,
	}
	if err := s.store.CreateClientInstallation(ctx, installation); err != nil {
		return CreateResult{}, fmt.Errorf("create client installation: %w", err)
	}

	return CreateResult{Installation: installation, Token: token.Value}, nil
}

func (s *Service) List(ctx context.Context) ([]domainclient.ClientInstallation, error) {
	installations, err := s.store.ListClientInstallations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list client installations: %w", err)
	}
	return installations, nil
}

func (s *Service) Revoke(ctx context.Context, installationID, actorUserID string) (domainclient.ClientInstallation, error) {
	if installationID == "" {
		return domainclient.ClientInstallation{}, ErrInstallationNotFound
	}

	installation, found, err := s.store.RevokeClientInstallation(ctx, installationID, actorUserID, time.Now().UTC())
	if err != nil {
		return domainclient.ClientInstallation{}, fmt.Errorf("revoke client installation: %w", err)
	}
	if !found {
		return domainclient.ClientInstallation{}, ErrInstallationNotFound
	}
	return installation, nil
}

func (s *Service) Delete(ctx context.Context, installationID, expectedName, actorUserID string) error {
	if installationID == "" {
		return ErrInstallationNotFound
	}
	found, deleted, err := s.store.DeleteClientInstallation(ctx, installationID, strings.TrimSpace(expectedName), actorUserID)
	if err != nil {
		return fmt.Errorf("delete client installation: %w", err)
	}
	if !found {
		return ErrInstallationNotFound
	}
	if !deleted {
		return ErrClientDeletionNotAllowed
	}
	return nil
}

func (s *Service) Authenticate(ctx context.Context, tokenValue string) (domainclient.ClientInstallation, error) {
	tokenPrefix, err := domainclient.ParseToken(tokenValue)
	if err != nil {
		return domainclient.ClientInstallation{}, ErrAuthenticationRequired
	}
	tokenHash, err := domainclient.DigestToken(tokenValue)
	if err != nil {
		return domainclient.ClientInstallation{}, ErrAuthenticationRequired
	}

	installation, found, err := s.store.FindClientInstallationByToken(ctx, tokenPrefix, tokenHash)
	if err != nil {
		return domainclient.ClientInstallation{}, fmt.Errorf("find client installation by token: %w", err)
	}
	if !found {
		return domainclient.ClientInstallation{}, ErrAuthenticationRequired
	}
	if !installation.IsActive() {
		if installation.Status == domainclient.InstallationStatusRevoked || installation.RevokedAt != nil {
			return domainclient.ClientInstallation{}, ErrInstallationRevoked
		}
		return domainclient.ClientInstallation{}, ErrAuthenticationRequired
	}

	now := time.Now().UTC()
	if installation.LastSeenAt == nil || now.Sub(*installation.LastSeenAt) >= lastSeenTouchInterval {
		active, err := s.store.TouchClientInstallation(ctx, installation.ID, now)
		if err != nil {
			return domainclient.ClientInstallation{}, fmt.Errorf("touch client installation: %w", err)
		}
		if !active {
			return domainclient.ClientInstallation{}, ErrInstallationRevoked
		}
		installation.LastSeenAt = &now
	}

	return installation, nil
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
