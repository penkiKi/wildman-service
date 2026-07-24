package catalog

import (
	"strings"
	"unicode"
)

type NormalizedTrackFields struct {
	Title           string
	Artists         []string
	Album           string
	DurationMS      *int64
	DiscNumber      *int
	TrackNumber     *int
	VersionKeywords []string
}

var versionKeywords = []struct {
	canonical string
	aliases   []string
}{
	{"live", []string{"live", "现场", "演唱会"}},
	{"remaster", []string{"remaster", "remastered", "重制", "重制版"}},
	{"deluxe", []string{"deluxe", "豪华", "豪华版"}},
	{"acoustic", []string{"acoustic", "不插电"}},
	{"instrumental", []string{"instrumental", "伴奏", "纯音乐"}},
	{"karaoke", []string{"karaoke", "卡拉ok"}},
	{"mono", []string{"mono", "单声道"}},
	{"stereo", []string{"stereo", "立体声"}},
}

func NormalizeTrackObservation(observation TrackObservation) NormalizedTrackFields {
	artists := make([]string, 0, len(observation.Artists))
	for _, artist := range observation.Artists {
		if normalized := NormalizeText(artist); normalized != "" {
			artists = append(artists, normalized)
		}
	}
	var durationMS *int64
	if observation.DurationMS != nil {
		value := *observation.DurationMS
		durationMS = &value
	}
	var discNumber, trackNumber *int
	if observation.DiscNumber != nil {
		value := *observation.DiscNumber
		discNumber = &value
	}
	if observation.TrackNumber != nil {
		value := *observation.TrackNumber
		trackNumber = &value
	}
	return NormalizedTrackFields{
		Title:           NormalizeText(observation.Title),
		Artists:         artists,
		Album:           NormalizeText(observation.Album),
		DurationMS:      durationMS,
		DiscNumber:      discNumber,
		TrackNumber:     trackNumber,
		VersionKeywords: ExtractVersionKeywords(observation.Title),
	}
}

func NormalizeText(value string) string {
	var builder strings.Builder
	spacePending := false
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(character) || unicode.IsNumber(character) {
			if spacePending && builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteRune(character)
			spacePending = false
		} else if builder.Len() > 0 {
			spacePending = true
		}
	}
	return builder.String()
}

func ExtractVersionKeywords(title string) []string {
	normalized := NormalizeText(title)
	found := make([]string, 0)
	for _, definition := range versionKeywords {
		for _, alias := range definition.aliases {
			if containsNormalizedPhrase(normalized, NormalizeText(alias)) {
				found = append(found, definition.canonical)
				break
			}
		}
	}
	return found
}

func containsNormalizedPhrase(value, phrase string) bool {
	if value == phrase {
		return true
	}
	return strings.HasPrefix(value, phrase+" ") || strings.HasSuffix(value, " "+phrase) || strings.Contains(value, " "+phrase+" ")
}
