package catalog

import (
	"math"
	"slices"
	"strings"
)

func MergeCrossSourceCandidates(candidates []Candidate) []Candidate {
	merged := make([]Candidate, 0, len(candidates))
	indexes := make(map[string]int)
	for _, candidate := range candidates {
		if len(candidate.Sources) == 0 && candidate.Source != "" {
			candidate.Sources = []string{candidate.Source}
		}
		key := candidateMergeKey(candidate)
		index, found := indexes[key]
		if !found || key == "" {
			indexes[key] = len(merged)
			merged = append(merged, candidate)
			continue
		}
		existing := &merged[index]
		for _, source := range candidate.Sources {
			if !slices.Contains(existing.Sources, source) {
				existing.Sources = append(existing.Sources, source)
			}
		}
		if candidate.Score > existing.Score {
			existing.Score = candidate.Score
			existing.Evidence = append(existing.Evidence, candidate.Evidence...)
			existing.Conflicts = append(existing.Conflicts, candidate.Conflicts...)
		}
		if len(existing.Sources) > 1 && !slices.Contains(existing.Evidence, "多个独立来源一致") {
			existing.Evidence = append(existing.Evidence, "多个独立来源一致")
			existing.Score = math.Min(1, existing.Score+0.05)
		}
	}
	return merged
}

func candidateMergeKey(candidate Candidate) string {
	if candidate.Recording.ISRC != "" {
		return "isrc:" + strings.ToUpper(candidate.Recording.ISRC)
	}
	artists := make([]string, 0, len(candidate.Recording.Artists))
	for _, artist := range candidate.Recording.Artists {
		artists = append(artists, NormalizeText(artist.Name))
	}
	slices.Sort(artists)
	if candidate.Recording.Title == "" || len(artists) == 0 {
		return ""
	}
	return "metadata:" + NormalizeText(candidate.Recording.Title) + ":" + strings.Join(artists, "|")
}
