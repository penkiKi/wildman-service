package catalog

import "time"

type ReviewDecision string
type WritebackStatus string

const (
	ReviewAccepted        ReviewDecision  = "accepted"
	ReviewRejected        ReviewDecision  = "rejected"
	WritebackNotAttempted WritebackStatus = "not_attempted"
	WritebackSucceeded    WritebackStatus = "succeeded"
	WritebackFailed       WritebackStatus = "failed"
)

type ResolutionReview struct {
	RequestID          string
	ClientID           string
	Decision           ReviewDecision
	RecordingID        string
	WritebackStatus    WritebackStatus
	WritebackErrorCode string
	ReviewedAt         time.Time
	WritebackAt        *time.Time
}
