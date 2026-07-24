package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"

	domainauth "wildman-service/internal/domain/auth"
)

const (
	minimumUsernameLength = 3
	maximumUsernameLength = 32
	minimumPasswordLength = 6
	maximumPasswordBytes  = 1024
)

var (
	ErrSetupComplete      = errors.New("setup is already complete")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("authentication required")
)

type Store interface {
	IsInitialized(ctx context.Context) (bool, error)
	CreateInitialAdmin(ctx context.Context, user domainauth.User) (bool, error)
	FindUserByUsername(ctx context.Context, username string) (domainauth.User, bool, error)
	CreateSession(ctx context.Context, session domainauth.Session) error
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (domainauth.Session, domainauth.User, bool, error)
	TouchSession(ctx context.Context, sessionID string, lastSeenAt time.Time) error
	RevokeSession(ctx context.Context, tokenHash string, revokedAt time.Time) error
}

type SetupStatus struct {
	Required bool `json:"required"`
}

type Service struct {
	store             Store
	dummyPasswordHash string
}

func NewService(store Store) *Service {
	dummyPasswordHash, err := hashPassword("wildman-dummy-password")
	if err != nil {
		panic("create dummy password hash: " + err.Error())
	}
	return &Service{store: store, dummyPasswordHash: dummyPasswordHash}
}

func (s *Service) SetupStatus(ctx context.Context) (SetupStatus, error) {
	initialized, err := s.store.IsInitialized(ctx)
	if err != nil {
		return SetupStatus{}, fmt.Errorf("read setup status: %w", err)
	}
	return SetupStatus{Required: !initialized}, nil
}

func (s *Service) CreateInitialAdmin(ctx context.Context, username, password string) (domainauth.User, error) {
	normalizedUsername, err := normalizeUsername(username)
	if err != nil {
		return domainauth.User{}, err
	}
	if err := validatePassword(password); err != nil {
		return domainauth.User{}, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return domainauth.User{}, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := domainauth.User{
		ID:           randomID(),
		Username:     normalizedUsername,
		PasswordHash: passwordHash,
		Status:       domainauth.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := s.store.CreateInitialAdmin(ctx, user)
	if err != nil {
		return domainauth.User{}, fmt.Errorf("create initial admin: %w", err)
	}
	if !created {
		return domainauth.User{}, ErrSetupComplete
	}
	return user, nil
}

func normalizeUsername(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	length := utf8.RuneCountInString(value)
	if length < minimumUsernameLength || length > maximumUsernameLength {
		return "", ErrInvalidUsername
	}
	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || strings.ContainsRune("._-", character) {
			continue
		}
		return "", ErrInvalidUsername
	}
	return value, nil
}

func validatePassword(value string) error {
	length := utf8.RuneCountInString(value)
	if length < minimumPasswordLength || len(value) > maximumPasswordBytes {
		return ErrInvalidPassword
	}
	return nil
}

func ValidatePassword(value string) error { return validatePassword(value) }

func hashPassword(password string) (string, error) {
	const (
		memory      = 64 * 1024
		iterations  = 3
		parallelism = 2
		saltLength  = 16
		keyLength   = 32
	)

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func HashPassword(password string) (string, error) { return hashPassword(password) }

func randomID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(value)
}
