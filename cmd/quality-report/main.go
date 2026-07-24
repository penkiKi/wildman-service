package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"wildman-service/internal/domain/catalog"
)

type sample struct {
	Name                string                   `json:"name"`
	Observation         catalog.TrackObservation `json:"observation"`
	Candidates          []sampleCandidate        `json:"candidates"`
	ExpectedRecordingID string                   `json:"expectedRecordingId"`
}

type sampleCandidate struct {
	Recording catalog.Recording `json:"recording"`
	Release   *catalog.Release  `json:"release"`
}

type sampleResult struct {
	Name                string  `json:"name"`
	ExpectedRecordingID string  `json:"expectedRecordingId"`
	ActualRecordingID   string  `json:"actualRecordingId"`
	Score               float64 `json:"score"`
	Correct             bool    `json:"correct"`
}

func main() {
	path := "docs/quality/samples.json"
	if len(os.Args) == 2 {
		path = os.Args[1]
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	var samples []sample
	if err := json.Unmarshal(payload, &samples); err != nil {
		fail(err)
	}
	results := make([]sampleResult, 0, len(samples))
	correct := 0
	correctScoreTotal := 0.0
	for _, item := range samples {
		scored := make([]catalog.Candidate, 0, len(item.Candidates))
		for _, candidate := range item.Candidates {
			scored = append(scored, catalog.ScoreCandidate(item.Observation, candidate.Recording, candidate.Release))
		}
		slices.SortStableFunc(scored, func(left, right catalog.Candidate) int {
			if left.Score > right.Score {
				return -1
			}
			if left.Score < right.Score {
				return 1
			}
			return 0
		})
		result := sampleResult{Name: item.Name, ExpectedRecordingID: item.ExpectedRecordingID}
		if len(scored) > 0 {
			result.ActualRecordingID = scored[0].Recording.ID
			result.Score = scored[0].Score
		}
		result.Correct = result.ActualRecordingID == result.ExpectedRecordingID
		if result.Correct {
			correct++
			correctScoreTotal += result.Score
		}
		results = append(results, result)
	}
	averageScore := 0.0
	if correct > 0 {
		averageScore = correctScoreTotal / float64(correct)
	}
	report := struct {
		SampleCount         int            `json:"sampleCount"`
		Top1Correct         int            `json:"top1Correct"`
		Top1Accuracy        float64        `json:"top1Accuracy"`
		AverageCorrectScore float64        `json:"averageCorrectScore"`
		Results             []sampleResult `json:"results"`
	}{SampleCount: len(samples), Top1Correct: correct, AverageCorrectScore: averageScore, Results: results}
	if len(samples) > 0 {
		report.Top1Accuracy = float64(correct) / float64(len(samples))
	}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
