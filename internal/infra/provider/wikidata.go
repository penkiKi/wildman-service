package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	appprovider "wildman-service/internal/app/provider"
)

const wikidataEndpoint = "https://query.wikidata.org/sparql"

type Wikidata struct {
	client      *http.Client
	userAgent   string
	mutex       sync.Mutex
	nextRequest time.Time
}

func NewWikidata(serviceVersion, operatorContact string) (*Wikidata, error) {
	if strings.TrimSpace(operatorContact) == "" {
		return nil, fmt.Errorf("Wikidata operator contact is required")
	}
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second}
	return &Wikidata{client: &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if len(via) >= 2 || request.URL.Scheme != "https" || !strings.EqualFold(request.URL.Host, "query.wikidata.org") {
			return http.ErrUseLastResponse
		}
		return nil
	}}, userAgent: fmt.Sprintf("WildmanService/%s (%s)", serviceVersion, strings.TrimSpace(operatorContact))}, nil
}

func (provider *Wikidata) Name() string           { return "wikidata" }
func (provider *Wikidata) AdapterVersion() string { return "wikidata-v1" }

func (provider *Wikidata) SearchRecordings(ctx context.Context, query appprovider.RecordingQuery) ([]appprovider.RecordingCandidate, error) {
	title := strings.TrimSpace(query.Title)
	if title == "" {
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: fmt.Errorf("title is required")}
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
	sparql := `SELECT ?work ?workLabel WHERE { ?work wdt:P31/wdt:P279* wd:Q7302866; rdfs:label ?workLabel. FILTER(LANG(?workLabel) = "en" || LANG(?workLabel) = "zh") FILTER(CONTAINS(LCASE(STR(?workLabel)), LCASE("` + escapeSPARQL(title) + `"))) } LIMIT 25`
	values := url.Values{"query": {sparql}, "format": {"json"}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, wikidataEndpoint+"?"+values.Encode(), nil)
	if err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: err}
	}
	request.Header.Set("Accept", "application/sparql-results+json")
	request.Header.Set("User-Agent", provider.userAgent)
	response, err := provider.client.Do(request)
	if err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorUnavailable, Err: err}
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, &appprovider.Error{Kind: appprovider.ErrorRateLimited, RetryAfter: retryAfter(response.Header.Get("Retry-After"))}
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
	var result struct {
		Results struct {
			Bindings []struct {
				Work, WorkLabel struct {
					Value string `json:"value"`
				}
			} `json:"bindings"`
		} `json:"results"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, &appprovider.Error{Kind: appprovider.ErrorInvalidResponse, Err: err}
	}
	candidates := make([]appprovider.RecordingCandidate, 0, len(result.Results.Bindings))
	for _, binding := range result.Results.Bindings {
		id := binding.Work.Value
		if index := strings.LastIndex(id, "/"); index >= 0 {
			id = id[index+1:]
		}
		candidates = append(candidates, appprovider.RecordingCandidate{ExternalID: id, Title: binding.WorkLabel.Value, RawPayload: payload})
	}
	return candidates, nil
}

func escapeSPARQL(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`).Replace(value)
}
