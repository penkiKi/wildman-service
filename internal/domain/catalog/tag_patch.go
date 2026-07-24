package catalog

type TagValue struct {
	Text   *string  `json:"text,omitempty"`
	Texts  []string `json:"texts,omitempty"`
	Number *int     `json:"number,omitempty"`
}

type TagPatchOperation struct {
	Field     string   `json:"field"`
	Current   TagValue `json:"current"`
	Suggested TagValue `json:"suggested"`
	Source    string   `json:"source"`
}

func GenerateTagPatch(observation TrackObservation, candidate Candidate, track *Track) []TagPatchOperation {
	patch := make([]TagPatchOperation, 0)
	if candidate.Recording.Title != "" && NormalizeText(observation.Title) != NormalizeText(candidate.Recording.Title) {
		patch = append(patch, textPatch("title", observation.Title, candidate.Recording.Title, candidate.Source))
	}
	artistNames := make([]string, 0, len(candidate.Recording.Artists))
	normalizedArtists := make([]string, 0, len(candidate.Recording.Artists))
	for _, artist := range candidate.Recording.Artists {
		artistNames = append(artistNames, artist.Name)
		normalizedArtists = append(normalizedArtists, NormalizeText(artist.Name))
	}
	if len(artistNames) > 0 && !sameStringSet(NormalizeTrackObservation(observation).Artists, normalizedArtists) {
		patch = append(patch, TagPatchOperation{Field: "artists", Current: TagValue{Texts: append([]string(nil), observation.Artists...)}, Suggested: TagValue{Texts: artistNames}, Source: candidate.Source})
	}
	if candidate.Release != nil && candidate.Release.Title != "" && NormalizeText(observation.Album) != NormalizeText(candidate.Release.Title) {
		patch = append(patch, textPatch("album", observation.Album, candidate.Release.Title, candidate.Source))
	}
	if track != nil {
		if observation.DiscNumber == nil || *observation.DiscNumber != track.DiscNumber {
			patch = append(patch, numberPatch("discNumber", observation.DiscNumber, track.DiscNumber, candidate.Source))
		}
		if observation.TrackNumber == nil || *observation.TrackNumber != track.TrackNumber {
			patch = append(patch, numberPatch("trackNumber", observation.TrackNumber, track.TrackNumber, candidate.Source))
		}
	}
	if candidate.Recording.ISRC != "" {
		patch = append(patch, textPatch("isrc", "", candidate.Recording.ISRC, candidate.Source))
	}
	return patch
}

func textPatch(field, current, suggested, source string) TagPatchOperation {
	return TagPatchOperation{Field: field, Current: TagValue{Text: &current}, Suggested: TagValue{Text: &suggested}, Source: source}
}

func numberPatch(field string, current *int, suggested int, source string) TagPatchOperation {
	return TagPatchOperation{Field: field, Current: TagValue{Number: current}, Suggested: TagValue{Number: &suggested}, Source: source}
}
