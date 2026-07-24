package catalog

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	MaxClientTrackIDLength = 200
	MaxFileNameLength      = 255
	MaxTitleLength         = 500
	MaxArtistCount         = 32
	MaxArtistLength        = 200
	MaxAlbumLength         = 500
	MaxDurationMS          = int64(7 * 24 * 60 * 60 * 1000)
	MaxFormatLength        = 32
	MaxFingerprintLength   = 16 * 1024
)

var ErrObservationInvalid = errors.New("track observation is invalid")

type ObservationValidationError struct {
	Field  string
	Reason string
}

func (err *ObservationValidationError) Error() string {
	return fmt.Sprintf("%s: %s", err.Field, err.Reason)
}

func (err *ObservationValidationError) Unwrap() error {
	return ErrObservationInvalid
}

func (observation TrackObservation) Validate() error {
	switch {
	case strings.TrimSpace(observation.ClientTrackID) == "":
		return invalidObservationField("clientTrackId", "required")
	case runeLength(observation.ClientTrackID) > MaxClientTrackIDLength:
		return invalidObservationField("clientTrackId", "too_long")
	case runeLength(observation.FileName) > MaxFileNameLength:
		return invalidObservationField("fileName", "too_long")
	case observation.FileName == "." || observation.FileName == ".." || strings.ContainsAny(observation.FileName, `/\`):
		return invalidObservationField("fileName", "must_be_file_name")
	case runeLength(observation.Title) > MaxTitleLength:
		return invalidObservationField("title", "too_long")
	case len(observation.Artists) > MaxArtistCount:
		return invalidObservationField("artists", "too_many")
	}
	for _, artist := range observation.Artists {
		if strings.TrimSpace(artist) == "" {
			return invalidObservationField("artists", "contains_empty_value")
		}
		if runeLength(artist) > MaxArtistLength {
			return invalidObservationField("artists", "value_too_long")
		}
	}
	switch {
	case runeLength(observation.Album) > MaxAlbumLength:
		return invalidObservationField("album", "too_long")
	case observation.DiscNumber != nil && (*observation.DiscNumber < 1 || *observation.DiscNumber > 999):
		return invalidObservationField("discNumber", "out_of_range")
	case observation.TrackNumber != nil && (*observation.TrackNumber < 1 || *observation.TrackNumber > 999):
		return invalidObservationField("trackNumber", "out_of_range")
	case observation.DurationMS != nil && (*observation.DurationMS < 0 || *observation.DurationMS > MaxDurationMS):
		return invalidObservationField("durationMs", "out_of_range")
	case runeLength(observation.Format) > MaxFormatLength:
		return invalidObservationField("format", "too_long")
	case len(observation.Fingerprint) > MaxFingerprintLength:
		return invalidObservationField("fingerprint", "too_long")
	case observation.ObservedAt.IsZero():
		return invalidObservationField("observedAt", "required")
	default:
		return nil
	}
}

func invalidObservationField(field, reason string) error {
	return &ObservationValidationError{Field: field, Reason: reason}
}

func runeLength(value string) int {
	return utf8.RuneCountInString(value)
}
