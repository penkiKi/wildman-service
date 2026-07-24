CREATE TABLE provider_metrics (
    scope TEXT PRIMARY KEY,
    cache_lookups INTEGER NOT NULL,
    cache_hits INTEGER NOT NULL,
    negative_hits INTEGER NOT NULL,
    coalesced_requests INTEGER NOT NULL,
    provider_requests INTEGER NOT NULL,
    errors_json TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
