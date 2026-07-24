package auth

import "time"

type Session struct {
	ID         string
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	LastSeenAt time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}
