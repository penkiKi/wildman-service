package catalog

import (
	"math"
	"slices"
)

func ScoreCandidate(observation TrackObservation, recording Recording, release *Release) Candidate {
	normalized := NormalizeTrackObservation(observation)
	score := 0.0
	evidence := make([]string, 0)
	conflicts := make([]string, 0)

	titleSimilarity := textSimilarity(normalized.Title, NormalizeText(recording.Title))
	score += 0.40 * titleSimilarity
	if titleSimilarity == 1 {
		evidence = append(evidence, "标题完全一致")
	} else if titleSimilarity >= 0.75 {
		evidence = append(evidence, "标题高度相似")
	} else if normalized.Title != "" && recording.Title != "" {
		conflicts = append(conflicts, "标题不一致")
	}

	candidateArtists := make([]string, 0, len(recording.Artists))
	for _, artist := range recording.Artists {
		candidateArtists = append(candidateArtists, NormalizeText(artist.Name))
	}
	if len(normalized.Artists) > 0 && len(candidateArtists) > 0 {
		if sameStringSet(normalized.Artists, candidateArtists) {
			score += 0.30
			evidence = append(evidence, "歌手完全一致")
		} else if setsOverlap(normalized.Artists, candidateArtists) {
			score += 0.15
			evidence = append(evidence, "部分歌手一致")
		} else {
			conflicts = append(conflicts, "歌手不一致")
		}
	}

	if release != nil && normalized.Album != "" && release.Title != "" {
		albumSimilarity := textSimilarity(normalized.Album, NormalizeText(release.Title))
		if albumSimilarity == 1 {
			score += 0.15
			evidence = append(evidence, "专辑完全一致")
		} else if albumSimilarity >= 0.75 {
			score += 0.075
			evidence = append(evidence, "专辑高度相似")
		} else {
			conflicts = append(conflicts, "专辑不一致")
		}
	}

	if normalized.DurationMS != nil && recording.DurationMS != nil {
		difference := absInt64(*normalized.DurationMS - *recording.DurationMS)
		switch {
		case difference <= 2_000:
			score += 0.10
			evidence = append(evidence, "时长误差不超过 2 秒")
		case difference <= 10_000:
			score += 0.05
			evidence = append(evidence, "时长接近")
		default:
			conflicts = append(conflicts, "时长差异超过 10 秒")
		}
	}

	candidateVersions := ExtractVersionKeywords(recording.Title)
	if sameStringSet(normalized.VersionKeywords, candidateVersions) {
		score += 0.05
		if len(candidateVersions) > 0 {
			evidence = append(evidence, "版本关键词一致")
		}
	} else {
		conflicts = append(conflicts, "版本关键词不一致")
	}

	return Candidate{
		Recording: recording,
		Release:   release,
		Score:     math.Max(0, math.Min(1, score)),
		Evidence:  evidence,
		Conflicts: conflicts,
	}
}

func textSimilarity(left, right string) float64 {
	if left == right && left != "" {
		return 1
	}
	leftRunes, rightRunes := []rune(left), []rune(right)
	maximumLength := max(len(leftRunes), len(rightRunes))
	if maximumLength == 0 {
		return 0
	}
	return 1 - float64(editDistance(leftRunes, rightRunes))/float64(maximumLength)
}

func editDistance(left, right []rune) int {
	previous := make([]int, len(right)+1)
	for index := range previous {
		previous[index] = index
	}
	for leftIndex, leftRune := range left {
		current := make([]int, len(right)+1)
		current[0] = leftIndex + 1
		for rightIndex, rightRune := range right {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[rightIndex+1] = min(current[rightIndex]+1, previous[rightIndex+1]+1, previous[rightIndex]+cost)
		}
		previous = current
	}
	return previous[len(right)]
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy, rightCopy := append([]string(nil), left...), append([]string(nil), right...)
	slices.Sort(leftCopy)
	slices.Sort(rightCopy)
	return slices.Equal(leftCopy, rightCopy)
}

func setsOverlap(left, right []string) bool {
	for _, leftValue := range left {
		if slices.Contains(right, leftValue) {
			return true
		}
	}
	return false
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}
