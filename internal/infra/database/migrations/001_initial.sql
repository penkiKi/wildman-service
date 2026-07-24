CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions(user_id);
CREATE INDEX sessions_expires_at_idx ON sessions(expires_at);

CREATE TABLE libraries (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL UNIQUE,
    mode TEXT NOT NULL CHECK (mode IN ('read_only', 'read_write')),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE media_files (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    relative_path TEXT NOT NULL,
    path_key TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    modified_at TEXT NOT NULL,
    format TEXT NOT NULL,
    duration_ms INTEGER,
    bit_rate INTEGER,
    sample_rate INTEGER,
    channels INTEGER,
    codec TEXT,
    scan_status TEXT NOT NULL,
    missing_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (library_id, path_key)
);

CREATE INDEX media_files_library_id_idx ON media_files(library_id);
CREATE INDEX media_files_scan_status_idx ON media_files(scan_status);

CREATE TABLE tag_snapshots (
    id TEXT PRIMARY KEY,
    media_file_id TEXT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('scan', 'before_write', 'after_write')),
    tags_json TEXT NOT NULL,
    parser_version TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX tag_snapshots_media_file_id_idx ON tag_snapshots(media_file_id);

CREATE TABLE library_issues (
    id TEXT PRIMARY KEY,
    media_file_id TEXT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    issue_type TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'error')),
    details_json TEXT NOT NULL DEFAULT '{}',
    resolved_at TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (media_file_id, issue_type)
);

CREATE INDEX library_issues_issue_type_idx ON library_issues(issue_type);
CREATE INDEX library_issues_resolved_at_idx ON library_issues(resolved_at);

CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'canceling', 'completed', 'failed', 'canceled')),
    input_json TEXT NOT NULL,
    progress_current INTEGER NOT NULL DEFAULT 0,
    progress_total INTEGER NOT NULL DEFAULT 0,
    cancel_requested_at TEXT,
    last_error_code TEXT,
    last_error_message TEXT,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT
);

CREATE INDEX jobs_status_idx ON jobs(status);
CREATE INDEX jobs_created_at_idx ON jobs(created_at);

CREATE TABLE job_items (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    item_key TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'failed', 'skipped')),
    error_code TEXT,
    error_message TEXT,
    created_at TEXT NOT NULL,
    finished_at TEXT,
    UNIQUE (job_id, item_key)
);

CREATE INDEX job_items_job_id_idx ON job_items(job_id);
