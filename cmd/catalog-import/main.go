package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"wildman-service/internal/domain/catalog"
	"wildman-service/internal/infra/database"
)

type importRecord struct {
	Type        string   `json:"type"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	SortName    string   `json:"sortName"`
	ArtistIDs   []string `json:"artistIds"`
	ReleaseID   string   `json:"releaseId"`
	RecordingID string   `json:"recordingId"`
	ReleaseDate string   `json:"releaseDate"`
	Barcode     string   `json:"barcode"`
	ISRC        string   `json:"isrc"`
	DurationMS  *int64   `json:"durationMs"`
	DiscNumber  int      `json:"discNumber"`
	TrackNumber int      `json:"trackNumber"`
}

func main() {
	dataDir := flag.String("data-dir", "./data", "service data directory")
	input := flag.String("input", "", "MusicBrainz-derived JSONL file")
	flag.Parse()
	if *input == "" {
		fail("-input is required")
	}
	file, err := os.Open(*input)
	if err != nil {
		fail(err.Error())
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	db, err := database.Open(ctx, *dataDir)
	if err != nil {
		fail(err.Error())
	}
	defer db.Close()
	store := database.NewCatalogStore(db)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	count := 0
	for scanner.Scan() {
		count++
		var record importRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			fail(fmt.Sprintf("line %d: %v", count, err))
		}
		if err := importOne(ctx, store, record); err != nil {
			fail(fmt.Sprintf("line %d: %v", count, err))
		}
	}
	if err := scanner.Err(); err != nil {
		fail(err.Error())
	}
	fmt.Printf("imported %d records\n", count)
}

func importOne(ctx context.Context, store *database.CatalogStore, record importRecord) error {
	if record.ID == "" || !strings.HasPrefix(record.ID, "musicbrainz:") {
		return fmt.Errorf("id must use musicbrainz: prefix")
	}
	artists := make([]catalog.Artist, 0, len(record.ArtistIDs))
	for _, id := range record.ArtistIDs {
		artists = append(artists, catalog.Artist{ID: id})
	}
	switch record.Type {
	case "artist":
		return store.SaveArtist(ctx, catalog.Artist{ID: record.ID, Name: record.Name, NormalizedName: catalog.NormalizeText(record.Name), SortName: record.SortName})
	case "release":
		return store.SaveRelease(ctx, catalog.Release{ID: record.ID, Title: record.Title, NormalizedTitle: catalog.NormalizeText(record.Title), Artists: artists, ReleaseDate: record.ReleaseDate, Barcode: record.Barcode})
	case "recording":
		return store.SaveRecording(ctx, catalog.Recording{ID: record.ID, Title: record.Title, NormalizedTitle: catalog.NormalizeText(record.Title), Artists: artists, DurationMS: record.DurationMS, ISRC: record.ISRC})
	case "track":
		if record.ReleaseID == "" || record.RecordingID == "" || record.DiscNumber < 1 || record.TrackNumber < 1 {
			return fmt.Errorf("track relationship and position are required")
		}
		return store.SaveTrack(ctx, catalog.Track{ID: record.ID, ReleaseID: record.ReleaseID, RecordingID: record.RecordingID, DiscNumber: record.DiscNumber, TrackNumber: record.TrackNumber, Title: record.Title, DurationMS: record.DurationMS})
	default:
		return fmt.Errorf("unsupported entity type %q", record.Type)
	}
}

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
