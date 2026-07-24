package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wildman-service/internal/domain/catalog"
)

func (s *ObservationStore) SaveResolutionCandidates(
	ctx context.Context,
	clientID string,
	requestID string,
	candidates []catalog.Candidate,
	finishedAt time.Time,
) (bool, error) {
	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin resolution completion: %w", err)
	}
	defer transaction.Rollback()
	status := catalog.ResolutionStatusMatched
	if len(candidates) == 0 {
		status = catalog.ResolutionStatusNoMatch
	}
	result, err := transaction.ExecContext(ctx, `
		UPDATE resolution_requests
		SET status = ?, finished_at = ?, last_error_code = NULL
		WHERE id = ? AND client_id = ?
	`, status, finishedAt.UTC().Format(time.RFC3339Nano), requestID, clientID)
	if err != nil {
		return false, fmt.Errorf("complete resolution request: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read resolution completion result: %w", err)
	}
	if rowsAffected == 0 {
		return false, nil
	}
	if _, err := transaction.ExecContext(ctx, "DELETE FROM resolution_candidates WHERE request_id = ?", requestID); err != nil {
		return false, fmt.Errorf("clear resolution candidates: %w", err)
	}
	for _, candidate := range candidates {
		evidence, err := json.Marshal(candidate.Evidence)
		if err != nil {
			return false, fmt.Errorf("encode candidate evidence: %w", err)
		}
		conflicts, err := json.Marshal(candidate.Conflicts)
		if err != nil {
			return false, fmt.Errorf("encode candidate conflicts: %w", err)
		}
		tagPatch, err := json.Marshal(candidate.TagPatch)
		if err != nil {
			return false, fmt.Errorf("encode candidate tag patch: %w", err)
		}
		sources := candidate.Sources
		if len(sources) == 0 {
			sources = []string{candidate.Source}
		}
		sourcesJSON, err := json.Marshal(sources)
		if err != nil {
			return false, fmt.Errorf("encode candidate sources: %w", err)
		}
		var releaseID any
		if candidate.Release != nil {
			releaseID = candidate.Release.ID
		}
		if _, err := transaction.ExecContext(ctx, `
			INSERT INTO resolution_candidates (
				id, request_id, recording_id, rank, score, evidence_json, conflicts_json,
				created_at, release_id, source, source_observation_id, tag_patch_json, sources_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, candidate.ID, requestID, candidate.Recording.ID, candidate.Rank, candidate.Score,
			string(evidence), string(conflicts), finishedAt.UTC().Format(time.RFC3339Nano),
			releaseID, candidate.Source, nullableString(candidate.SourceObservationID), string(tagPatch), string(sourcesJSON)); err != nil {
			return false, fmt.Errorf("insert resolution candidate: %w", err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return false, fmt.Errorf("commit resolution completion: %w", err)
	}
	return true, nil
}

func (s *ObservationStore) ListResolutionCandidates(
	ctx context.Context,
	clientID string,
	requestID string,
) ([]catalog.Candidate, error) {
	rows, err := s.database.QueryContext(ctx, `
		SELECT c.id, c.rank, c.score, c.evidence_json, c.conflicts_json,
			c.source, c.source_observation_id, c.recording_id, c.release_id, c.tag_patch_json, c.sources_json
		FROM resolution_candidates AS c
		JOIN resolution_requests AS r ON r.id = c.request_id
		WHERE c.request_id = ? AND r.client_id = ?
		ORDER BY c.rank
	`, requestID, clientID)
	if err != nil {
		return nil, fmt.Errorf("query resolution candidates: %w", err)
	}
	type storedCandidate struct {
		candidate   catalog.Candidate
		recordingID string
		releaseID   sql.NullString
	}
	stored := make([]storedCandidate, 0)
	for rows.Next() {
		var item storedCandidate
		var evidence, conflicts, tagPatch, sourcesJSON string
		var sourceObservationID sql.NullString
		if err := rows.Scan(
			&item.candidate.ID,
			&item.candidate.Rank,
			&item.candidate.Score,
			&evidence,
			&conflicts,
			&item.candidate.Source,
			&sourceObservationID,
			&item.recordingID,
			&item.releaseID,
			&tagPatch,
			&sourcesJSON,
		); err != nil {
			rows.Close()
			return nil, err
		}
		item.candidate.SourceObservationID = sourceObservationID.String
		if err := json.Unmarshal([]byte(evidence), &item.candidate.Evidence); err != nil {
			rows.Close()
			return nil, fmt.Errorf("decode candidate evidence: %w", err)
		}
		if err := json.Unmarshal([]byte(conflicts), &item.candidate.Conflicts); err != nil {
			rows.Close()
			return nil, fmt.Errorf("decode candidate conflicts: %w", err)
		}
		if err := json.Unmarshal([]byte(tagPatch), &item.candidate.TagPatch); err != nil {
			rows.Close()
			return nil, fmt.Errorf("decode candidate tag patch: %w", err)
		}
		if err := json.Unmarshal([]byte(sourcesJSON), &item.candidate.Sources); err != nil {
			rows.Close()
			return nil, fmt.Errorf("decode candidate sources: %w", err)
		}
		stored = append(stored, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	catalogStore := NewCatalogStore(s.database)
	candidates := make([]catalog.Candidate, 0, len(stored))
	for _, item := range stored {
		recording, found, err := catalogStore.GetRecording(ctx, item.recordingID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("candidate recording not found")
		}
		item.candidate.Recording = recording
		if item.releaseID.Valid {
			release, found, err := catalogStore.GetRelease(ctx, item.releaseID.String)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, fmt.Errorf("candidate release not found")
			}
			item.candidate.Release = &release
		}
		candidates = append(candidates, item.candidate)
	}
	return candidates, nil
}
