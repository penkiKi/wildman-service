package account

import "time"

type Account struct {
	ID           string
	Email        string
	PasswordHash string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID        string
	AccountID string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Subscription struct {
	AccountID    string
	Plan         string
	Status       string
	MonthlyQuota int64
	UpdatedAt    time.Time
}

type DeviceAuthorization struct {
	ID             string
	DeviceCodeHash string
	UserCodeHash   string
	ClientName     string
	Status         string
	AccountID      string
	ExpiresAt      time.Time
	ApprovedAt     *time.Time
	ConsumedAt     *time.Time
	CreatedAt      time.Time
}
