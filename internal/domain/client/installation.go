package client

import "time"

type InstallationStatus string

const (
	InstallationStatusActive  InstallationStatus = "active"
	InstallationStatusRevoked InstallationStatus = "revoked"
)

type ClientInstallation struct {
	ID              string
	Name            string
	TokenPrefix     string
	TokenHash       string
	Status          InstallationStatus
	CreatedByUserID string
	AccountID       string
	LastSeenAt      *time.Time
	RevokedAt       *time.Time
	CreatedAt       time.Time
}

func (installation ClientInstallation) IsActive() bool {
	return installation.Status == InstallationStatusActive && installation.RevokedAt == nil
}

func (installation *ClientInstallation) Revoke(at time.Time) bool {
	if !installation.IsActive() {
		return false
	}

	revokedAt := at.UTC()
	installation.Status = InstallationStatusRevoked
	installation.RevokedAt = &revokedAt
	return true
}
