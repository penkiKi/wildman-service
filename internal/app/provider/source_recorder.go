package provider

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"wildman-service/internal/domain/catalog"
)

var ErrSourceObservationInvalid = errors.New("source observation is invalid")

type SourceStore interface {
	CreateSourceObservation(
		ctx context.Context,
		observation catalog.SourceObservation,
	) (catalog.SourceObservation, bool, error)
}

type SourceRecorder struct {
	store SourceStore
}

func NewSourceRecorder(store SourceStore) *SourceRecorder {
	return &SourceRecorder{store: store}
}

func (recorder *SourceRecorder) Record(
	ctx context.Context,
	providerName string,
	adapterVersion string,
	entityType catalog.SourceEntityType,
	externalID string,
	canonicalEntityID string,
	payload []byte,
	ttl time.Duration,
) (catalog.SourceObservation, bool, error) {
	if strings.TrimSpace(providerName) == "" || strings.TrimSpace(adapterVersion) == "" || strings.TrimSpace(externalID) == "" || !validSourceEntityType(entityType) || !json.Valid(payload) || ttl < 0 {
		return catalog.SourceObservation{}, false, ErrSourceObservationInvalid
	}
	id, err := sourceObservationID()
	if err != nil {
		return catalog.SourceObservation{}, false, fmt.Errorf("generate source observation ID: %w", err)
	}
	digest := sha256.Sum256(payload)
	now := time.Now().UTC()
	var expiresAt *time.Time
	if ttl > 0 {
		expiration := now.Add(ttl)
		expiresAt = &expiration
	}
	observation, created, err := recorder.store.CreateSourceObservation(ctx, catalog.SourceObservation{
		ID:                id,
		Provider:          providerName,
		EntityType:        entityType,
		ExternalID:        externalID,
		CanonicalEntityID: canonicalEntityID,
		PayloadJSON:       append([]byte(nil), payload...),
		PayloadHash:       base64.RawURLEncoding.EncodeToString(digest[:]),
		FetchedAt:         now,
		ExpiresAt:         expiresAt,
		AdapterVersion:    adapterVersion,
	})
	if err != nil {
		return catalog.SourceObservation{}, false, fmt.Errorf("record source observation: %w", err)
	}
	return observation, created, nil
}

func validSourceEntityType(entityType catalog.SourceEntityType) bool {
	return entityType == catalog.SourceEntityArtist || entityType == catalog.SourceEntityRelease || entityType == catalog.SourceEntityRecording
}

func sourceObservationID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
