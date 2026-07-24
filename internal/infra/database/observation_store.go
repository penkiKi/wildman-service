package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wildman-service/internal/domain/catalog"
)

type ObservationStore struct {
	database *DB
}

func NewObservationStore(database *DB) *ObservationStore {
	return &ObservationStore{database: database}
}

func (s *ObservationStore) UpsertTrackObservation(
	ctx context.Context,
	record catalog.StoredTrackObservation,
) (catalog.StoredTrackObservation, error) {
	artists := record.Observation.Artists
	if artists == nil {
		artists = []string{}
	}
	artistsJSON, err := json.Marshal(artists)
	if err != nil {
		return catalog.StoredTrackObservation{}, fmt.Errorf("encode observation artists: %w", err)
	}
	durationMS := nullableInt64(record.Observation.DurationMS)
	stored, err := scanTrackObservation(s.database.QueryRowContext(ctx, `
		INSERT INTO track_observations (
			id, client_id, client_track_id, file_name, title, artists_json, album,
			disc_number, track_number, duration_ms, format, fingerprint, payload_hash, observed_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (client_id, client_track_id) DO UPDATE SET
			file_name = excluded.file_name,
			title = excluded.title,
			artists_json = excluded.artists_json,
			album = excluded.album,
			disc_number = excluded.disc_number,
			track_number = excluded.track_number,
			duration_ms = excluded.duration_ms,
			format = excluded.format,
			fingerprint = excluded.fingerprint,
			payload_hash = excluded.payload_hash,
			observed_at = excluded.observed_at,
			updated_at = excluded.updated_at
		RETURNING id, client_id, client_track_id, file_name, title, artists_json,
			album, disc_number, track_number, duration_ms, format, fingerprint, payload_hash, observed_at, created_at, updated_at
	`,
		record.ID,
		record.ClientID,
		record.Observation.ClientTrackID,
		record.Observation.FileName,
		record.Observation.Title,
		string(artistsJSON),
		record.Observation.Album,
		nullableInt(record.Observation.DiscNumber),
		nullableInt(record.Observation.TrackNumber),
		durationMS,
		record.Observation.Format,
		record.Observation.Fingerprint,
		record.PayloadHash,
		record.Observation.ObservedAt.UTC().Format(time.RFC3339Nano),
		record.CreatedAt.UTC().Format(time.RFC3339Nano),
		record.UpdatedAt.UTC().Format(time.RFC3339Nano),
	))
	if err != nil {
		return catalog.StoredTrackObservation{}, fmt.Errorf("write track observation: %w", err)
	}
	return stored, nil
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func scanTrackObservation(scanner rowScanner) (catalog.StoredTrackObservation, error) {
	var record catalog.StoredTrackObservation
	var artistsJSON string
	var durationMS sql.NullInt64
	var discNumber, trackNumber sql.NullInt64
	var observedAt, createdAt, updatedAt string
	err := scanner.Scan(
		&record.ID,
		&record.ClientID,
		&record.Observation.ClientTrackID,
		&record.Observation.FileName,
		&record.Observation.Title,
		&artistsJSON,
		&record.Observation.Album,
		&discNumber,
		&trackNumber,
		&durationMS,
		&record.Observation.Format,
		&record.Observation.Fingerprint,
		&record.PayloadHash,
		&observedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return catalog.StoredTrackObservation{}, err
	}
	if err := json.Unmarshal([]byte(artistsJSON), &record.Observation.Artists); err != nil {
		return catalog.StoredTrackObservation{}, fmt.Errorf("decode observation artists: %w", err)
	}
	if durationMS.Valid {
		record.Observation.DurationMS = &durationMS.Int64
	}
	if discNumber.Valid {
		value := int(discNumber.Int64)
		record.Observation.DiscNumber = &value
	}
	if trackNumber.Valid {
		value := int(trackNumber.Int64)
		record.Observation.TrackNumber = &value
	}
	if record.Observation.ObservedAt, err = parseObservationTime(observedAt); err != nil {
		return catalog.StoredTrackObservation{}, err
	}
	if record.CreatedAt, err = parseObservationTime(createdAt); err != nil {
		return catalog.StoredTrackObservation{}, err
	}
	if record.UpdatedAt, err = parseObservationTime(updatedAt); err != nil {
		return catalog.StoredTrackObservation{}, err
	}
	return record, nil
}

func parseObservationTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse observation time: %w", err)
	}
	return parsed, nil
}
