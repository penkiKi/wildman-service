package resolution

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"wildman-service/internal/domain/catalog"
)

const MaxIdempotencyKeyLength = 200

var (
	ErrClientIdentityRequired = errors.New("client identity is required")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyKeyInvalid  = errors.New("idempotency key is invalid")
	ErrObservationIDRequired  = errors.New("observation ID is required")
)

type Store interface {
	UpsertTrackObservation(
		ctx context.Context,
		observation catalog.StoredTrackObservation,
	) (catalog.StoredTrackObservation, error)
	CreateResolutionRequest(
		ctx context.Context,
		request catalog.ResolutionRequest,
	) (catalog.ResolutionRequest, bool, error)
	GetResolutionRequest(
		ctx context.Context,
		clientID string,
		requestID string,
	) (catalog.ResolutionRequest, bool, error)
	SaveResolutionCandidates(
		ctx context.Context,
		clientID string,
		requestID string,
		candidates []catalog.Candidate,
		finishedAt time.Time,
	) (bool, error)
	ListResolutionCandidates(
		ctx context.Context,
		clientID string,
		requestID string,
	) ([]catalog.Candidate, error)
	SaveResolutionReview(ctx context.Context, review catalog.ResolutionReview) (catalog.ResolutionReview, bool, error)
}

var ErrResolutionNotFound = errors.New("resolution request not found")
var ErrReviewInvalid = errors.New("resolution review is invalid")

type Result struct {
	Request    catalog.ResolutionRequest
	Candidates []catalog.Candidate
}

func (s *Service) GetRequest(
	ctx context.Context,
	clientID string,
	requestID string,
) (catalog.ResolutionRequest, error) {
	if clientID == "" {
		return catalog.ResolutionRequest{}, ErrClientIdentityRequired
	}
	if requestID == "" {
		return catalog.ResolutionRequest{}, ErrResolutionNotFound
	}
	request, found, err := s.store.GetResolutionRequest(ctx, clientID, requestID)
	if err != nil {
		return catalog.ResolutionRequest{}, fmt.Errorf("get resolution request: %w", err)
	}
	if !found {
		return catalog.ResolutionRequest{}, ErrResolutionNotFound
	}
	return request, nil
}

func (s *Service) CompleteRequest(
	ctx context.Context,
	clientID string,
	requestID string,
	candidates []catalog.Candidate,
) error {
	if clientID == "" {
		return ErrClientIdentityRequired
	}
	slices.SortStableFunc(candidates, func(left, right catalog.Candidate) int {
		if left.Score > right.Score {
			return -1
		}
		if left.Score < right.Score {
			return 1
		}
		return strings.Compare(left.Recording.ID, right.Recording.ID)
	})
	for index := range candidates {
		id, err := randomID()
		if err != nil {
			return fmt.Errorf("generate resolution candidate ID: %w", err)
		}
		candidates[index].ID = id
		candidates[index].Rank = index + 1
		if candidates[index].Source == "" {
			candidates[index].Source = "catalog"
		}
		if len(candidates[index].Sources) == 0 {
			candidates[index].Sources = []string{candidates[index].Source}
		}
	}
	found, err := s.store.SaveResolutionCandidates(ctx, clientID, requestID, candidates, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("save resolution candidates: %w", err)
	}
	if !found {
		return ErrResolutionNotFound
	}
	return nil
}

func (s *Service) GetResult(ctx context.Context, clientID, requestID string) (Result, error) {
	request, err := s.GetRequest(ctx, clientID, requestID)
	if err != nil {
		return Result{}, err
	}
	candidates, err := s.store.ListResolutionCandidates(ctx, clientID, requestID)
	if err != nil {
		return Result{}, fmt.Errorf("list resolution candidates: %w", err)
	}
	return Result{Request: request, Candidates: candidates}, nil
}

func (s *Service) ReportReview(
	ctx context.Context,
	clientID string,
	requestID string,
	decision catalog.ReviewDecision,
	recordingID string,
	writebackStatus catalog.WritebackStatus,
	writebackErrorCode string,
) (catalog.ResolutionReview, error) {
	if clientID == "" {
		return catalog.ResolutionReview{}, ErrClientIdentityRequired
	}
	if requestID == "" || (decision != catalog.ReviewAccepted && decision != catalog.ReviewRejected) ||
		(writebackStatus != catalog.WritebackNotAttempted && writebackStatus != catalog.WritebackSucceeded && writebackStatus != catalog.WritebackFailed) {
		return catalog.ResolutionReview{}, ErrReviewInvalid
	}
	if decision == catalog.ReviewAccepted && recordingID == "" {
		return catalog.ResolutionReview{}, ErrReviewInvalid
	}
	if decision == catalog.ReviewRejected && writebackStatus != catalog.WritebackNotAttempted {
		return catalog.ResolutionReview{}, ErrReviewInvalid
	}
	if writebackStatus == catalog.WritebackFailed {
		if writebackErrorCode == "" || len(writebackErrorCode) > 64 || !isPrintableASCII(writebackErrorCode) {
			return catalog.ResolutionReview{}, ErrReviewInvalid
		}
	} else if writebackErrorCode != "" {
		return catalog.ResolutionReview{}, ErrReviewInvalid
	}
	now := time.Now().UTC()
	var writebackAt *time.Time
	if writebackStatus != catalog.WritebackNotAttempted {
		writebackAt = &now
	}
	review, found, err := s.store.SaveResolutionReview(ctx, catalog.ResolutionReview{
		RequestID: requestID, ClientID: clientID, Decision: decision, RecordingID: recordingID,
		WritebackStatus: writebackStatus, WritebackErrorCode: writebackErrorCode, ReviewedAt: now, WritebackAt: writebackAt,
	})
	if err != nil {
		return catalog.ResolutionReview{}, fmt.Errorf("save resolution review: %w", err)
	}
	if !found {
		return catalog.ResolutionReview{}, ErrResolutionNotFound
	}
	return review, nil
}

func (s *Service) CreateRequest(
	ctx context.Context,
	clientID string,
	observationID string,
	idempotencyKey string,
) (catalog.ResolutionRequest, bool, error) {
	if clientID == "" {
		return catalog.ResolutionRequest{}, false, ErrClientIdentityRequired
	}
	if idempotencyKey == "" {
		return catalog.ResolutionRequest{}, false, ErrIdempotencyKeyRequired
	}
	if len(idempotencyKey) > MaxIdempotencyKeyLength || strings.TrimSpace(idempotencyKey) != idempotencyKey || !isPrintableASCII(idempotencyKey) {
		return catalog.ResolutionRequest{}, false, ErrIdempotencyKeyInvalid
	}
	if observationID == "" {
		return catalog.ResolutionRequest{}, false, ErrObservationIDRequired
	}
	id, err := randomID()
	if err != nil {
		return catalog.ResolutionRequest{}, false, fmt.Errorf("generate resolution request ID: %w", err)
	}
	request, created, err := s.store.CreateResolutionRequest(ctx, catalog.ResolutionRequest{
		ID:             id,
		ClientID:       clientID,
		ObservationID:  observationID,
		IdempotencyKey: idempotencyKey,
		Status:         catalog.ResolutionStatusQueued,
		CreatedAt:      time.Now().UTC(),
	})
	if err != nil {
		return catalog.ResolutionRequest{}, false, fmt.Errorf("create resolution request: %w", err)
	}
	return request, created, nil
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) UpsertObservation(
	ctx context.Context,
	clientID string,
	observation catalog.TrackObservation,
) (catalog.StoredTrackObservation, error) {
	if clientID == "" {
		return catalog.StoredTrackObservation{}, ErrClientIdentityRequired
	}
	if err := observation.Validate(); err != nil {
		return catalog.StoredTrackObservation{}, err
	}

	payloadHash, err := observationHash(observation)
	if err != nil {
		return catalog.StoredTrackObservation{}, fmt.Errorf("hash track observation: %w", err)
	}
	id, err := randomID()
	if err != nil {
		return catalog.StoredTrackObservation{}, fmt.Errorf("generate track observation ID: %w", err)
	}
	now := time.Now().UTC()
	record, err := s.store.UpsertTrackObservation(ctx, catalog.StoredTrackObservation{
		ID:          id,
		ClientID:    clientID,
		Observation: observation,
		PayloadHash: payloadHash,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return catalog.StoredTrackObservation{}, fmt.Errorf("upsert track observation: %w", err)
	}
	return record, nil
}

func observationHash(observation catalog.TrackObservation) (string, error) {
	artists := observation.Artists
	if artists == nil {
		artists = []string{}
	}
	payload, err := json.Marshal(struct {
		ClientTrackID string   `json:"clientTrackId"`
		FileName      string   `json:"fileName"`
		Title         string   `json:"title"`
		Artists       []string `json:"artists"`
		Album         string   `json:"album"`
		DiscNumber    *int     `json:"discNumber"`
		TrackNumber   *int     `json:"trackNumber"`
		DurationMS    *int64   `json:"durationMs"`
		Format        string   `json:"format"`
		Fingerprint   string   `json:"fingerprint"`
		ObservedAt    string   `json:"observedAt"`
	}{
		ClientTrackID: observation.ClientTrackID,
		FileName:      observation.FileName,
		Title:         observation.Title,
		Artists:       artists,
		Album:         observation.Album,
		DiscNumber:    observation.DiscNumber,
		TrackNumber:   observation.TrackNumber,
		DurationMS:    observation.DurationMS,
		Format:        observation.Format,
		Fingerprint:   observation.Fingerprint,
		ObservedAt:    observation.ObservedAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func isPrintableASCII(value string) bool {
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}
