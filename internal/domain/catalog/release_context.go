package catalog

import "math"

func ApplyReleaseContext(candidate Candidate, observation TrackObservation, track *Track) Candidate {
	if track == nil {
		return candidate
	}
	if candidate.Release != nil && track.ReleaseID != candidate.Release.ID {
		candidate.Conflicts = append(candidate.Conflicts, "候选曲目不属于候选发行")
		candidate.Score = math.Max(0, candidate.Score-0.10)
		return candidate
	}
	matchedPosition := false
	if observation.DiscNumber != nil {
		if *observation.DiscNumber == track.DiscNumber {
			matchedPosition = true
		} else {
			candidate.Conflicts = append(candidate.Conflicts, "碟号不一致")
			candidate.Score = math.Max(0, candidate.Score-0.05)
		}
	}
	if observation.TrackNumber != nil {
		if *observation.TrackNumber == track.TrackNumber {
			matchedPosition = true
		} else {
			candidate.Conflicts = append(candidate.Conflicts, "曲号不一致")
			candidate.Score = math.Max(0, candidate.Score-0.10)
		}
	}
	if matchedPosition {
		candidate.Evidence = append(candidate.Evidence, "发行内曲目顺序一致")
		candidate.Score = math.Min(1, candidate.Score+0.05)
	}
	return candidate
}
