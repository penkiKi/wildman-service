package auth

import "time"

const UserStatusActive = "active"

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
