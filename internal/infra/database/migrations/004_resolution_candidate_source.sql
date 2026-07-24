ALTER TABLE resolution_candidates ADD COLUMN release_id TEXT REFERENCES catalog_releases(id);
ALTER TABLE resolution_candidates ADD COLUMN source TEXT NOT NULL DEFAULT 'catalog';
ALTER TABLE resolution_candidates ADD COLUMN source_observation_id TEXT REFERENCES source_observations(id);
