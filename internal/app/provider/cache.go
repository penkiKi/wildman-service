package provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"
)

const (
	DefaultSearchTTL   = 7 * 24 * time.Hour
	DefaultNegativeTTL = 6 * time.Hour
)

type cacheEntry struct {
	candidates []RecordingCandidate
	expiresAt  time.Time
	negative   bool
}

type providerCall struct {
	done       chan struct{}
	candidates []RecordingCandidate
	err        error
}

type CachedProvider struct {
	provider    Provider
	positiveTTL time.Duration
	negativeTTL time.Duration
	mutex       sync.Mutex
	entries     map[string]cacheEntry
	inFlight    map[string]*providerCall
	metrics     *Metrics
}

func NewCachedProvider(provider Provider, positiveTTL, negativeTTL time.Duration) *CachedProvider {
	if positiveTTL <= 0 {
		positiveTTL = DefaultSearchTTL
	}
	if negativeTTL <= 0 {
		negativeTTL = DefaultNegativeTTL
	}
	return &CachedProvider{
		provider: provider, positiveTTL: positiveTTL, negativeTTL: negativeTTL,
		entries: make(map[string]cacheEntry), inFlight: make(map[string]*providerCall),
		metrics: NewMetrics(),
	}
}

func (cached *CachedProvider) Name() string { return cached.provider.Name() }

func (cached *CachedProvider) AdapterVersion() string { return cached.provider.AdapterVersion() }

func (cached *CachedProvider) Metrics() MetricsSnapshot { return cached.metrics.Snapshot() }

func (cached *CachedProvider) SearchRecordings(ctx context.Context, query RecordingQuery) ([]RecordingCandidate, error) {
	key, err := cached.key(query)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	cached.metrics.observeCacheLookup()
	cached.mutex.Lock()
	if entry, found := cached.entries[key]; found {
		if now.Before(entry.expiresAt) {
			result := cloneCandidates(entry.candidates)
			cached.mutex.Unlock()
			cached.metrics.observeCacheHit(entry.negative)
			return result, nil
		}
		delete(cached.entries, key)
	}
	if call, found := cached.inFlight[key]; found {
		cached.mutex.Unlock()
		cached.metrics.observeCoalesced()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-call.done:
			return cloneCandidates(call.candidates), call.err
		}
	}
	call := &providerCall{done: make(chan struct{})}
	cached.inFlight[key] = call
	cached.mutex.Unlock()

	candidates, requestError := cached.provider.SearchRecordings(ctx, query)
	cached.metrics.observeProviderRequest(requestError)
	cached.mutex.Lock()
	call.candidates, call.err = cloneCandidates(candidates), requestError
	if requestError == nil {
		ttl := cached.positiveTTL
		if len(candidates) == 0 {
			ttl = cached.negativeTTL
		}
		cached.entries[key] = cacheEntry{candidates: cloneCandidates(candidates), expiresAt: time.Now().Add(ttl), negative: len(candidates) == 0}
	}
	delete(cached.inFlight, key)
	close(call.done)
	cached.mutex.Unlock()
	return candidates, requestError
}

func (cached *CachedProvider) key(query RecordingQuery) (string, error) {
	payload, err := json.Marshal(struct {
		Provider       string         `json:"provider"`
		AdapterVersion string         `json:"adapterVersion"`
		Query          RecordingQuery `json:"query"`
	}{cached.provider.Name(), cached.provider.AdapterVersion(), query})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func cloneCandidates(candidates []RecordingCandidate) []RecordingCandidate {
	if candidates == nil {
		return nil
	}
	cloned := make([]RecordingCandidate, len(candidates))
	for index, candidate := range candidates {
		cloned[index] = candidate
		cloned[index].Artists = append([]string(nil), candidate.Artists...)
		cloned[index].RawPayload = append([]byte(nil), candidate.RawPayload...)
	}
	return cloned
}
