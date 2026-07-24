package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domaincatalog "wildman-service/internal/domain/catalog"
)

type CatalogStore struct {
	database *DB
}

func NewCatalogStore(database *DB) *CatalogStore {
	return &CatalogStore{database: database}
}

func (s *CatalogStore) SaveArtist(ctx context.Context, artist domaincatalog.Artist) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.database.ExecContext(ctx, `
		INSERT INTO catalog_artists (id, name, normalized_name, sort_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			name = excluded.name,
			normalized_name = excluded.normalized_name,
			sort_name = excluded.sort_name,
			updated_at = excluded.updated_at
	`, artist.ID, artist.Name, artist.NormalizedName, nullableString(artist.SortName), now, now)
	if err != nil {
		return fmt.Errorf("save catalog artist: %w", err)
	}
	return nil
}

func (s *CatalogStore) SaveRelease(ctx context.Context, release domaincatalog.Release) error {
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog release save: %w", err)
	}
	defer transaction.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = transaction.ExecContext(ctx, `
		INSERT INTO catalog_releases (id, title, normalized_title, release_date, barcode, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			title = excluded.title,
			normalized_title = excluded.normalized_title,
			release_date = excluded.release_date,
			barcode = excluded.barcode,
			updated_at = excluded.updated_at
	`, release.ID, release.Title, release.NormalizedTitle, nullableString(release.ReleaseDate), nullableString(release.Barcode), now, now)
	if err != nil {
		return fmt.Errorf("save catalog release: %w", err)
	}
	if err := replaceReleaseArtists(ctx, transaction, release.ID, release.Artists); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit catalog release save: %w", err)
	}
	return nil
}

func (s *CatalogStore) SaveRecording(ctx context.Context, recording domaincatalog.Recording) error {
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog recording save: %w", err)
	}
	defer transaction.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = transaction.ExecContext(ctx, `
		INSERT INTO catalog_recordings (id, title, normalized_title, duration_ms, isrc, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			title = excluded.title,
			normalized_title = excluded.normalized_title,
			duration_ms = excluded.duration_ms,
			isrc = excluded.isrc,
			updated_at = excluded.updated_at
	`, recording.ID, recording.Title, recording.NormalizedTitle, nullableInt64(recording.DurationMS), nullableString(recording.ISRC), now, now)
	if err != nil {
		return fmt.Errorf("save catalog recording: %w", err)
	}
	if err := replaceRecordingArtists(ctx, transaction, recording.ID, recording.Artists); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit catalog recording save: %w", err)
	}
	return nil
}

func (s *CatalogStore) SaveTrack(ctx context.Context, track domaincatalog.Track) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.database.ExecContext(ctx, `
		INSERT INTO catalog_tracks (
			id, release_id, recording_id, disc_number, track_number, title, duration_ms, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			release_id = excluded.release_id,
			recording_id = excluded.recording_id,
			disc_number = excluded.disc_number,
			track_number = excluded.track_number,
			title = excluded.title,
			duration_ms = excluded.duration_ms,
			updated_at = excluded.updated_at
	`, track.ID, track.ReleaseID, track.RecordingID, track.DiscNumber, track.TrackNumber, track.Title, nullableInt64(track.DurationMS), now, now)
	if err != nil {
		return fmt.Errorf("save catalog track: %w", err)
	}
	return nil
}

func (s *CatalogStore) GetArtist(ctx context.Context, id string) (domaincatalog.Artist, bool, error) {
	return scanArtistResult(s.database.QueryRowContext(ctx, `
		SELECT id, name, normalized_name, sort_name FROM catalog_artists WHERE id = ?
	`, id))
}

func (s *CatalogStore) GetRelease(ctx context.Context, id string) (domaincatalog.Release, bool, error) {
	release, found, err := scanReleaseResult(s.database.QueryRowContext(ctx, `
		SELECT id, title, normalized_title, release_date, barcode FROM catalog_releases WHERE id = ?
	`, id))
	if err != nil || !found {
		return release, found, err
	}
	release.Artists, err = s.releaseArtists(ctx, release.ID)
	return release, true, err
}

func (s *CatalogStore) GetRecording(ctx context.Context, id string) (domaincatalog.Recording, bool, error) {
	recording, found, err := scanRecordingResult(s.database.QueryRowContext(ctx, `
		SELECT id, title, normalized_title, duration_ms, isrc FROM catalog_recordings WHERE id = ?
	`, id))
	if err != nil || !found {
		return recording, found, err
	}
	recording.Artists, err = s.recordingArtists(ctx, recording.ID)
	return recording, true, err
}

func (s *CatalogStore) GetTrack(ctx context.Context, id string) (domaincatalog.Track, bool, error) {
	return scanTrackResult(s.database.QueryRowContext(ctx, `
		SELECT id, release_id, recording_id, disc_number, track_number, title, duration_ms
		FROM catalog_tracks WHERE id = ?
	`, id))
}

func (s *CatalogStore) FindArtists(ctx context.Context, normalizedName string) ([]domaincatalog.Artist, error) {
	rows, err := s.database.QueryContext(ctx, `
		SELECT id, name, normalized_name, sort_name FROM catalog_artists
		WHERE normalized_name = ? ORDER BY name, id
	`, normalizedName)
	if err != nil {
		return nil, fmt.Errorf("find catalog artists: %w", err)
	}
	defer rows.Close()
	result := make([]domaincatalog.Artist, 0)
	for rows.Next() {
		artist, err := scanArtist(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, artist)
	}
	return result, rows.Err()
}

func (s *CatalogStore) FindReleases(ctx context.Context, normalizedTitle string) ([]domaincatalog.Release, error) {
	rows, err := s.database.QueryContext(ctx, `
		SELECT id, title, normalized_title, release_date, barcode FROM catalog_releases
		WHERE normalized_title = ? ORDER BY title, id
	`, normalizedTitle)
	if err != nil {
		return nil, fmt.Errorf("find catalog releases: %w", err)
	}
	defer rows.Close()
	result := make([]domaincatalog.Release, 0)
	for rows.Next() {
		release, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, release)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range result {
		result[index].Artists, err = s.releaseArtists(ctx, result[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *CatalogStore) FindRecordings(ctx context.Context, normalizedTitle string) ([]domaincatalog.Recording, error) {
	rows, err := s.database.QueryContext(ctx, `
		SELECT id, title, normalized_title, duration_ms, isrc FROM catalog_recordings
		WHERE normalized_title = ? ORDER BY title, id
	`, normalizedTitle)
	if err != nil {
		return nil, fmt.Errorf("find catalog recordings: %w", err)
	}
	defer rows.Close()
	result := make([]domaincatalog.Recording, 0)
	for rows.Next() {
		recording, err := scanRecording(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, recording)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range result {
		result[index].Artists, err = s.recordingArtists(ctx, result[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func replaceReleaseArtists(ctx context.Context, tx *Tx, releaseID string, artists []domaincatalog.Artist) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM catalog_release_artists WHERE release_id = ?", releaseID); err != nil {
		return fmt.Errorf("clear catalog release artists: %w", err)
	}
	for position, artist := range artists {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_release_artists (release_id, artist_id, position) VALUES (?, ?, ?)
		`, releaseID, artist.ID, position); err != nil {
			return fmt.Errorf("save catalog release artist: %w", err)
		}
	}
	return nil
}

func replaceRecordingArtists(ctx context.Context, tx *Tx, recordingID string, artists []domaincatalog.Artist) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM catalog_recording_artists WHERE recording_id = ?", recordingID); err != nil {
		return fmt.Errorf("clear catalog recording artists: %w", err)
	}
	for position, artist := range artists {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_recording_artists (recording_id, artist_id, position) VALUES (?, ?, ?)
		`, recordingID, artist.ID, position); err != nil {
			return fmt.Errorf("save catalog recording artist: %w", err)
		}
	}
	return nil
}

func (s *CatalogStore) releaseArtists(ctx context.Context, releaseID string) ([]domaincatalog.Artist, error) {
	return s.relatedArtists(ctx, "catalog_release_artists", "release_id", releaseID)
}

func (s *CatalogStore) recordingArtists(ctx context.Context, recordingID string) ([]domaincatalog.Artist, error) {
	return s.relatedArtists(ctx, "catalog_recording_artists", "recording_id", recordingID)
}

func (s *CatalogStore) relatedArtists(ctx context.Context, table, foreignKey, entityID string) ([]domaincatalog.Artist, error) {
	query := fmt.Sprintf(`
		SELECT a.id, a.name, a.normalized_name, a.sort_name
		FROM %s AS relation
		JOIN catalog_artists AS a ON a.id = relation.artist_id
		WHERE relation.%s = ? ORDER BY relation.position
	`, table, foreignKey)
	rows, err := s.database.QueryContext(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("query related catalog artists: %w", err)
	}
	defer rows.Close()
	artists := make([]domaincatalog.Artist, 0)
	for rows.Next() {
		artist, err := scanArtist(rows)
		if err != nil {
			return nil, err
		}
		artists = append(artists, artist)
	}
	return artists, rows.Err()
}

func scanArtistResult(scanner rowScanner) (domaincatalog.Artist, bool, error) {
	artist, err := scanArtist(scanner)
	if err == sql.ErrNoRows {
		return domaincatalog.Artist{}, false, nil
	}
	return artist, err == nil, err
}

func scanArtist(scanner rowScanner) (domaincatalog.Artist, error) {
	var artist domaincatalog.Artist
	var sortName sql.NullString
	err := scanner.Scan(&artist.ID, &artist.Name, &artist.NormalizedName, &sortName)
	artist.SortName = sortName.String
	return artist, err
}

func scanReleaseResult(scanner rowScanner) (domaincatalog.Release, bool, error) {
	release, err := scanRelease(scanner)
	if err == sql.ErrNoRows {
		return domaincatalog.Release{}, false, nil
	}
	return release, err == nil, err
}

func scanRelease(scanner rowScanner) (domaincatalog.Release, error) {
	var release domaincatalog.Release
	var releaseDate, barcode sql.NullString
	err := scanner.Scan(&release.ID, &release.Title, &release.NormalizedTitle, &releaseDate, &barcode)
	release.ReleaseDate, release.Barcode = releaseDate.String, barcode.String
	return release, err
}

func scanRecordingResult(scanner rowScanner) (domaincatalog.Recording, bool, error) {
	recording, err := scanRecording(scanner)
	if err == sql.ErrNoRows {
		return domaincatalog.Recording{}, false, nil
	}
	return recording, err == nil, err
}

func scanRecording(scanner rowScanner) (domaincatalog.Recording, error) {
	var recording domaincatalog.Recording
	var duration sql.NullInt64
	var isrc sql.NullString
	err := scanner.Scan(&recording.ID, &recording.Title, &recording.NormalizedTitle, &duration, &isrc)
	if duration.Valid {
		recording.DurationMS = &duration.Int64
	}
	recording.ISRC = isrc.String
	return recording, err
}

func scanTrackResult(scanner rowScanner) (domaincatalog.Track, bool, error) {
	var track domaincatalog.Track
	var duration sql.NullInt64
	err := scanner.Scan(&track.ID, &track.ReleaseID, &track.RecordingID, &track.DiscNumber, &track.TrackNumber, &track.Title, &duration)
	if err == sql.ErrNoRows {
		return domaincatalog.Track{}, false, nil
	}
	if duration.Valid {
		track.DurationMS = &duration.Int64
	}
	return track, err == nil, err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
