package provider

import (
	"context"
	"encoding/json"
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

const acoustIDEndpoint = "https://api.acoustid.org/v2/lookup"

type AcoustID struct {
	client      *http.Client
	apiKey      string
	userAgent   string
	mutex       sync.Mutex
	nextRequest time.Time
}

func NewAcoustID(serviceVersion, operatorContact, apiKey string) (*AcoustID, error) {
	if strings.TrimSpace(operatorContact) == "" || strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("AcoustID contact and API key are required")
	}
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second}
	return &AcoustID{client: &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if len(via) > 0 || request.URL.Scheme != "https" || !strings.EqualFold(request.URL.Host, "api.acoustid.org") {
			return http.ErrUseLastResponse
		}
		return nil
	}}, apiKey: strings.TrimSpace(apiKey), userAgent: fmt.Sprintf("WildmanService/%s (%s)", serviceVersion, strings.TrimSpace(operatorContact))}, nil
}

func (provider *AcoustID) Name() string           { return "acoustid" }
func (provider *AcoustID) AdapterVersion() string { return "acoustid-v1" }

func (provider *AcoustID) SearchRecordings(ctx context.Context, query appprovider.RecordingQuery) ([]appprovider.RecordingCandidate, error) {
	if query.Fingerprint == "" || query.DurationMS == nil || *query.DurationMS <= 0 {
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: fmt.Errorf("fingerprint and duration are required")}
	}
	provider.mutex.Lock()
	wait := time.Until(provider.nextRequest)
	if wait < 0 {
		wait = 0
	}
	provider.nextRequest = time.Now().Add(wait + time.Second)
	provider.mutex.Unlock()
	if err := waitForRetry(ctx, wait); err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorTimeout, Err: err}
	}
	values := url.Values{}
	values.Set("client", provider.apiKey)
	values.Set("duration", strconv.FormatInt((*query.DurationMS+500)/1000, 10))
	values.Set("fingerprint", query.Fingerprint)
	values.Set("meta", "recordings+releasegroups+compress")
	values.Set("format", "json")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, acoustIDEndpoint+"?"+values.Encode(), nil)
	if err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: err}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", provider.userAgent)
	response, err := provider.client.Do(request)
	if err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorUnavailable, Err: err}
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, &appprovider.Error{Kind: appprovider.ErrorRateLimited, RetryAfter: retryAfter(response.Header.Get("Retry-After"))}
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, &appprovider.Error{Kind: appprovider.ErrorUnauthorized}
	}
	if response.StatusCode != http.StatusOK {
		return nil, &appprovider.Error{Kind: appprovider.ErrorUnavailable, Err: fmt.Errorf("status %d", response.StatusCode)}
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maximumProviderResponse+1))
	if err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorUnavailable, Err: err}
	}
	if len(payload) > maximumProviderResponse {
		return nil, &appprovider.Error{Kind: appprovider.ErrorResponseTooLarge}
	}
	var result acoustIDResponse
	if err := json.Unmarshal(payload, &result); err != nil || result.Status != "ok" {
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: err}
	}
	candidates := make([]appprovider.RecordingCandidate, 0)
	for _, match := range result.Results {
		for _, recording := range match.Recordings {
			candidate := appprovider.RecordingCandidate{ExternalID: recording.ID, Title: recording.Title, DurationMS: recording.DurationMS(), RawPayload: payload}
			for _, artist := range recording.Artists {
				candidate.Artists = append(candidate.Artists, artist.Name)
			}
			if len(recording.ReleaseGroups) > 0 {
				candidate.ReleaseExternalID = recording.ReleaseGroups[0].ID
				candidate.ReleaseTitle = recording.ReleaseGroups[0].Title
			}
			candidates = append(candidates, candidate)
		}
	}
	return candidates, nil
}

type acoustIDResponse struct {
	Status  string `json:"status"`
	Results []struct {
		Recordings []acoustIDRecording `json:"recordings"`
	} `json:"results"`
}

type acoustIDRecording struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
	Artists  []struct {
		Name string `json:"name"`
	} `json:"artists"`
	ReleaseGroups []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"releasegroups"`
}

func (recording acoustIDRecording) DurationMS() *int64 {
	if recording.Duration <= 0 {
		return nil
	}
	value := int64(recording.Duration * 1000)
	return &value
}
