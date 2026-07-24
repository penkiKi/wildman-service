CREATE TABLE resolution_reviews (
    request_id TEXT PRIMARY KEY REFERENCES resolution_requests(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES client_installations(id),
    decision TEXT NOT NULL CHECK (decision IN ('accepted', 'rejected')),
    recording_id TEXT REFERENCES catalog_recordings(id),
    writeback_status TEXT NOT NULL CHECK (writeback_status IN ('not_attempted', 'succeeded', 'failed')),
    writeback_error_code TEXT,
    reviewed_at TEXT NOT NULL,
    writeback_at TEXT
);

CREATE INDEX resolution_reviews_client_id_idx ON resolution_reviews(client_id, reviewed_at);
