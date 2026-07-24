package catalog

import "time"

type TrackObservation struct {
	ClientTrackID string
	FileName      string
	Title         string
	Artists       []string
	Album         string
	DiscNumber    *int
	TrackNumber   *int
	DurationMS    *int64
	Format        string
	Fingerprint   string
	ObservedAt    time.Time
}

type StoredTrackObservation struct {
	ID          string
	ClientID    string
	Observation TrackObservation
	PayloadHash string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ResolutionStatus string

const (
	ResolutionStatusQueued   ResolutionStatus = "queued"
	ResolutionStatusMatching ResolutionStatus = "matching"
	ResolutionStatusMatched  ResolutionStatus = "matched"
	ResolutionStatusNoMatch  ResolutionStatus = "no_match"
	ResolutionStatusFailed   ResolutionStatus = "failed"
)

type ResolutionRequest struct {
	ID             string
	ClientID       string
	ObservationID  string
	IdempotencyKey string
	Status         ResolutionStatus
	LastErrorCode  string
	CreatedAt      time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
}

type Artist struct {
	ID             string
	Name           string
	NormalizedName string
	SortName       string
}

type Recording struct {
	ID              string
	Title           string
	NormalizedTitle string
	Artists         []Artist
	DurationMS      *int64
	ISRC            string
}

type Release struct {
	ID              string
	Title           string
	NormalizedTitle string
	Artists         []Artist
	ReleaseDate     string
	Barcode         string
}

type Track struct {
	ID          string
	ReleaseID   string
	RecordingID string
	DiscNumber  int
	TrackNumber int
	Title       string
	DurationMS  *int64
}

type SourceEntityType string

const (
	SourceEntityArtist    SourceEntityType = "artist"
	SourceEntityRelease   SourceEntityType = "release"
	SourceEntityRecording SourceEntityType = "recording"
)

type SourceObservation struct {
	ID                string
	Provider          string
	EntityType        SourceEntityType
	ExternalID        string
	CanonicalEntityID string
	PayloadJSON       []byte
	PayloadHash       string
	FetchedAt         time.Time
	ExpiresAt         *time.Time
	AdapterVersion    string
}

type Candidate struct {
	ID                  string
	Rank                int
	Source              string
	Sources             []string
	SourceObservationID string
	Recording           Recording
	Release             *Release
	Score               float64
	Evidence            []string
	Conflicts           []string
	TagPatch            []TagPatchOperation
}
