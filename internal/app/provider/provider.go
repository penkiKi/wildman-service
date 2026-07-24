package provider

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type RecordingQuery struct {
	Title       string
	Artists     []string
	Album       string
	DurationMS  *int64
	Fingerprint string
}

type RecordingCandidate struct {
	ExternalID        string
	Title             string
	Artists           []string
	ReleaseExternalID string
	ReleaseTitle      string
	DurationMS        *int64
	ISRC              string
	RawPayload        []byte
}

type Provider interface {
	Name() string
	AdapterVersion() string
	SearchRecordings(ctx context.Context, query RecordingQuery) ([]RecordingCandidate, error)
}

type ErrorKind string

const (
	ErrorUnauthorized     ErrorKind = "unauthorized"
	ErrorForbidden        ErrorKind = "forbidden"
	ErrorRateLimited      ErrorKind = "rate_limited"
	ErrorTimeout          ErrorKind = "timeout"
	ErrorResponseTooLarge ErrorKind = "response_too_large"
	ErrorInvalidResponse  ErrorKind = "invalid_response"
	ErrorUnavailable      ErrorKind = "unavailable"
)

var ErrProvider = errors.New("provider request failed")

type Error struct {
	Kind       ErrorKind
	RetryAfter time.Duration
	Err        error
}

func (err *Error) Error() string {
	if err.Err == nil {
		return fmt.Sprintf("provider request failed: %s", err.Kind)
	}
	return fmt.Sprintf("provider request failed: %s: %v", err.Kind, err.Err)
}

func (err *Error) Unwrap() error {
	return ErrProvider
}

func ErrorKindOf(err error) (ErrorKind, bool) {
	var providerError *Error
	if !errors.As(err, &providerError) {
		return "", false
	}
	return providerError.Kind, true
}
