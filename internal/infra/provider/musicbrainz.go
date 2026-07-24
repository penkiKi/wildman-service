package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	appprovider "wildman-service/internal/app/provider"
)

const (
	musicBrainzName           = "musicbrainz"
	musicBrainzAdapterVersion = "musicbrainz-v1"
	musicBrainzBaseURL        = "https://musicbrainz.org/ws/2/recording"
	maximumProviderResponse   = 1 << 20
	maximumProviderAttempts   = 3
)

type MusicBrainz struct {
	client       *http.Client
	userAgent    string
	requestMutex sync.Mutex
	nextRequest  time.Time
}

func NewMusicBrainz(serviceVersion, operatorContact string) (*MusicBrainz, error) {
	operatorContact = strings.TrimSpace(operatorContact)
	if operatorContact == "" {
		return nil, fmt.Errorf("MusicBrainz operator contact is required")
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConnsPerHost:   2,
	}
	return &MusicBrainz{
		client: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
			CheckRedirect: func(request *http.Request, via []*http.Request) error {
				if len(via) >= 3 || request.URL.Scheme != "https" || !strings.EqualFold(request.URL.Host, "musicbrainz.org") || request.URL.User != nil {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		userAgent: fmt.Sprintf("WildmanService/%s (%s)", serviceVersion, operatorContact),
	}, nil
}

func (provider *MusicBrainz) Name() string { return musicBrainzName }

func (provider *MusicBrainz) AdapterVersion() string { return musicBrainzAdapterVersion }

func (provider *MusicBrainz) SearchRecordings(ctx context.Context, query appprovider.RecordingQuery) ([]appprovider.RecordingCandidate, error) {
	searchURL, err := musicBrainzSearchURL(query)
	if err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: err}
	}
	var lastError error
	for attempt := 0; attempt < maximumProviderAttempts; attempt++ {
		if attempt > 0 {
			if err := waitForRetry(ctx, time.Duration(1<<(attempt-1))*250*time.Millisecond); err != nil {
				return nil, &appprovider.Error{Kind: appprovider.ErrorTimeout, Err: err}
			}
		}
		if err := provider.waitForRateLimit(ctx); err != nil {
			return nil, &appprovider.Error{Kind: appprovider.ErrorTimeout, Err: err}
		}
		candidates, err := provider.searchOnce(ctx, searchURL)
		if err == nil {
			return candidates, nil
		}
		lastError = err
		kind, _ := appprovider.ErrorKindOf(err)
		if kind != appprovider.ErrorUnavailable && kind != appprovider.ErrorTimeout {
			return nil, err
		}
	}
	return nil, lastError
}

func (provider *MusicBrainz) searchOnce(ctx context.Context, searchURL string) ([]appprovider.RecordingCandidate, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: err}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", provider.userAgent)
	response, err := provider.client.Do(request)
	if err != nil {
		kind := appprovider.ErrorUnavailable
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			kind = appprovider.ErrorTimeout
		}
		return nil, &appprovider.Error{Kind: kind, Err: err}
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, &appprovider.Error{Kind: appprovider.ErrorUnauthorized}
	case http.StatusForbidden:
		return nil, &appprovider.Error{Kind: appprovider.ErrorForbidden}
	case http.StatusTooManyRequests:
		return nil, &appprovider.Error{Kind: appprovider.ErrorRateLimited, RetryAfter: retryAfter(response.Header.Get("Retry-After"))}
	default:
		if response.StatusCode >= 500 {
			return nil, &appprovider.Error{Kind: appprovider.ErrorUnavailable, Err: fmt.Errorf("status %d", response.StatusCode)}
		}
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: fmt.Errorf("status %d", response.StatusCode)}
	}

	payload, err := io.ReadAll(io.LimitReader(response.Body, maximumProviderResponse+1))
	if err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorUnavailable, Err: err}
	}
	if len(payload) > maximumProviderResponse {
		return nil, &appprovider.Error{Kind: appprovider.ErrorResponseTooLarge}
	}
	var result musicBrainzSearchResponse
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: err}
	}
	return toRecordingCandidates(result.Recordings, payload), nil
}

func (provider *MusicBrainz) waitForRateLimit(ctx context.Context) error {
	provider.requestMutex.Lock()
	wait := time.Until(provider.nextRequest)
	if wait < 0 {
		wait = 0
	}
	provider.nextRequest = time.Now().Add(wait + time.Second)
	provider.requestMutex.Unlock()
	return waitForRetry(ctx, wait)
}

func waitForRetry(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func musicBrainzSearchURL(query appprovider.RecordingQuery) (string, error) {
	clauses := make([]string, 0, 3)
	if value := strings.TrimSpace(query.Title); value != "" {
		clauses = append(clauses, `recording:"`+escapeLucene(value)+`"`)
	}
	if len(query.Artists) > 0 && strings.TrimSpace(query.Artists[0]) != "" {
		clauses = append(clauses, `artist:"`+escapeLucene(strings.TrimSpace(query.Artists[0]))+`"`)
	}
	if value := strings.TrimSpace(query.Album); value != "" {
		clauses = append(clauses, `release:"`+escapeLucene(value)+`"`)
	}
	if len(clauses) == 0 {
		return "", fmt.Errorf("at least one search field is required")
	}
	values := url.Values{}
	values.Set("query", strings.Join(clauses, " AND "))
	values.Set("fmt", "json")
	values.Set("limit", "25")
	return musicBrainzBaseURL + "?" + values.Encode(), nil
}

func escapeLucene(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `:`, `\:`, `(`, `\(`, `)`, `\)`)
	return replacer.Replace(value)
}

func retryAfter(value string) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		return max(0, time.Until(at))
	}
	return time.Second
}

type musicBrainzRecording struct {
	ID           string                    `json:"id"`
	Title        string                    `json:"title"`
	Length       *int64                    `json:"length"`
	ISRCs        []string                  `json:"isrcs"`
	ArtistCredit []musicBrainzArtistCredit `json:"artist-credit"`
	Releases     []musicBrainzRelease      `json:"releases"`
}

type musicBrainzArtistCredit struct {
	Name string `json:"name"`
}

type musicBrainzRelease struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type musicBrainzSearchResponse struct {
	Recordings []musicBrainzRecording `json:"recordings"`
}

func toRecordingCandidates(recordings []musicBrainzRecording, payload []byte) []appprovider.RecordingCandidate {
	candidates := make([]appprovider.RecordingCandidate, 0, len(recordings))
	for _, recording := range recordings {
		candidate := appprovider.RecordingCandidate{ExternalID: recording.ID, Title: recording.Title, DurationMS: recording.Length, RawPayload: payload}
		for _, credit := range recording.ArtistCredit {
			candidate.Artists = append(candidate.Artists, credit.Name)
		}
		if len(recording.Releases) > 0 {
			candidate.ReleaseExternalID = recording.Releases[0].ID
			candidate.ReleaseTitle = recording.Releases[0].Title
		}
		if len(recording.ISRCs) > 0 {
			candidate.ISRC = recording.ISRCs[0]
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}
