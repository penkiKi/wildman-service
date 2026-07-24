CREATE TABLE client_installations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    token_prefix TEXT NOT NULL UNIQUE,
    token_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'revoked')),
    created_by_user_id TEXT NOT NULL REFERENCES users(id),
    last_seen_at TEXT,
    revoked_at TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE catalog_artists (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    sort_name TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX catalog_artists_normalized_name_idx ON catalog_artists(normalized_name);

CREATE TABLE catalog_releases (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    normalized_title TEXT NOT NULL,
    release_date TEXT,
    barcode TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX catalog_releases_normalized_title_idx ON catalog_releases(normalized_title);

CREATE TABLE catalog_recordings (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    normalized_title TEXT NOT NULL,
    duration_ms INTEGER,
    isrc TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX catalog_recordings_normalized_title_idx ON catalog_recordings(normalized_title);
CREATE INDEX catalog_recordings_isrc_idx ON catalog_recordings(isrc);

CREATE TABLE catalog_recording_artists (
    recording_id TEXT NOT NULL REFERENCES catalog_recordings(id) ON DELETE CASCADE,
    artist_id TEXT NOT NULL REFERENCES catalog_artists(id),
    position INTEGER NOT NULL,
    PRIMARY KEY (recording_id, artist_id),
    UNIQUE (recording_id, position)
);

CREATE TABLE catalog_release_artists (
    release_id TEXT NOT NULL REFERENCES catalog_releases(id) ON DELETE CASCADE,
    artist_id TEXT NOT NULL REFERENCES catalog_artists(id),
    position INTEGER NOT NULL,
    PRIMARY KEY (release_id, artist_id),
    UNIQUE (release_id, position)
);

CREATE TABLE catalog_tracks (
    id TEXT PRIMARY KEY,
    release_id TEXT NOT NULL REFERENCES catalog_releases(id) ON DELETE CASCADE,
    recording_id TEXT NOT NULL REFERENCES catalog_recordings(id),
    disc_number INTEGER NOT NULL DEFAULT 1,
    track_number INTEGER NOT NULL,
    title TEXT NOT NULL,
    duration_ms INTEGER,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (release_id, disc_number, track_number)
);

CREATE INDEX catalog_tracks_recording_id_idx ON catalog_tracks(recording_id);

CREATE TABLE source_observations (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('artist', 'release', 'recording')),
    external_id TEXT NOT NULL,
    canonical_entity_id TEXT,
    payload_json TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    expires_at TEXT,
    adapter_version TEXT NOT NULL,
    UNIQUE (provider, entity_type, external_id, payload_hash)
);

CREATE INDEX source_observations_lookup_idx ON source_observations(provider, entity_type, external_id);
CREATE INDEX source_observations_expires_at_idx ON source_observations(expires_at);

CREATE TABLE track_observations (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES client_installations(id),
    client_track_id TEXT NOT NULL,
    file_name TEXT,
    title TEXT,
    artists_json TEXT NOT NULL DEFAULT '[]',
    album TEXT,
    duration_ms INTEGER,
    format TEXT,
    fingerprint TEXT,
    payload_hash TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (client_id, client_track_id)
);

CREATE TABLE resolution_requests (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES client_installations(id),
    observation_id TEXT NOT NULL REFERENCES track_observations(id),
    idempotency_key TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'matching', 'matched', 'no_match', 'failed')),
    last_error_code TEXT,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    UNIQUE (client_id, idempotency_key)
);

CREATE INDEX resolution_requests_status_idx ON resolution_requests(status);

CREATE TABLE resolution_candidates (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL REFERENCES resolution_requests(id) ON DELETE CASCADE,
    recording_id TEXT NOT NULL REFERENCES catalog_recordings(id),
    rank INTEGER NOT NULL,
    score REAL NOT NULL CHECK (score >= 0 AND score <= 1),
    evidence_json TEXT NOT NULL DEFAULT '[]',
    conflicts_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    UNIQUE (request_id, rank),
    UNIQUE (request_id, recording_id)
);
