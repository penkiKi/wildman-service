CREATE TABLE audit_events (
    id TEXT PRIMARY KEY,
    actor_user_id TEXT,
    action TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX audit_events_created_at_idx ON audit_events(created_at);
CREATE INDEX audit_events_subject_idx ON audit_events(subject_type, subject_id, created_at);
