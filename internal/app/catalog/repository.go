package catalog

import (
	"context"

	domaincatalog "wildman-service/internal/domain/catalog"
)

type Repository interface {
	SaveArtist(ctx context.Context, artist domaincatalog.Artist) error
	SaveRelease(ctx context.Context, release domaincatalog.Release) error
	SaveRecording(ctx context.Context, recording domaincatalog.Recording) error
	SaveTrack(ctx context.Context, track domaincatalog.Track) error
	GetArtist(ctx context.Context, id string) (domaincatalog.Artist, bool, error)
	GetRelease(ctx context.Context, id string) (domaincatalog.Release, bool, error)
	GetRecording(ctx context.Context, id string) (domaincatalog.Recording, bool, error)
	GetTrack(ctx context.Context, id string) (domaincatalog.Track, bool, error)
	FindArtists(ctx context.Context, normalizedName string) ([]domaincatalog.Artist, error)
	FindReleases(ctx context.Context, normalizedTitle string) ([]domaincatalog.Release, error)
	FindRecordings(ctx context.Context, normalizedTitle string) ([]domaincatalog.Recording, error)
}
