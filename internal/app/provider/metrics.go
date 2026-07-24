package provider

import "sync"

type MetricsSnapshot struct {
	CacheLookups      uint64            `json:"cacheLookups"`
	CacheHits         uint64            `json:"cacheHits"`
	NegativeHits      uint64            `json:"negativeHits"`
	CoalescedRequests uint64            `json:"coalescedRequests"`
	ProviderRequests  uint64            `json:"providerRequests"`
	Errors            map[string]uint64 `json:"errors"`
}

type Metrics struct {
	mutex    sync.Mutex
	snapshot MetricsSnapshot
}

func NewMetrics() *Metrics {
	return &Metrics{snapshot: MetricsSnapshot{Errors: make(map[string]uint64)}}
}

func (metrics *Metrics) Snapshot() MetricsSnapshot {
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()
	snapshot := metrics.snapshot
	snapshot.Errors = make(map[string]uint64, len(metrics.snapshot.Errors))
	for kind, count := range metrics.snapshot.Errors {
		snapshot.Errors[kind] = count
	}
	return snapshot
}

func (metrics *Metrics) observeCacheLookup() {
	metrics.mutex.Lock()
	metrics.snapshot.CacheLookups++
	metrics.mutex.Unlock()
}

func (metrics *Metrics) observeCacheHit(negative bool) {
	metrics.mutex.Lock()
	metrics.snapshot.CacheHits++
	if negative {
		metrics.snapshot.NegativeHits++
	}
	metrics.mutex.Unlock()
}

func (metrics *Metrics) observeCoalesced() {
	metrics.mutex.Lock()
	metrics.snapshot.CoalescedRequests++
	metrics.mutex.Unlock()
}

func (metrics *Metrics) observeProviderRequest(err error) {
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()
	metrics.snapshot.ProviderRequests++
	if err == nil {
		return
	}
	kind := "unknown"
	if providerKind, ok := ErrorKindOf(err); ok {
		kind = string(providerKind)
	}
	metrics.snapshot.Errors[kind]++
}
