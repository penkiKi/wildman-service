package account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	appauth "wildman-service/internal/app/auth"
	domainaccount "wildman-service/internal/domain/account"
	domainclient "wildman-service/internal/domain/client"
)

const freeMonthlyQuota = 1000

var (
	ErrInvalidEmail                  = errors.New("invalid account email")
	ErrInvalidPassword               = errors.New("invalid account password")
	ErrEmailExists                   = errors.New("account email exists")
	ErrInvalidCredentials            = errors.New("invalid account credentials")
	ErrAccountAuthenticationRequired = errors.New("account authentication required")
	ErrDeviceCodeInvalid             = errors.New("device code invalid")
	ErrDeviceAuthorizationPending    = errors.New("device authorization pending")
	ErrDeviceAuthorizationConsumed   = errors.New("device authorization consumed")
	ErrQuotaExceeded                 = errors.New("account quota exceeded")
	ErrSubscriptionInvalid           = errors.New("subscription invalid")
)

type Store interface {
	CreateAccount(ctx context.Context, account domainaccount.Account, subscription domainaccount.Subscription, session domainaccount.Session) (bool, error)
	FindAccountByEmail(ctx context.Context, email string) (domainaccount.Account, bool, error)
	CreateAccountSession(ctx context.Context, session domainaccount.Session) error
	FindAccountBySessionHash(ctx context.Context, tokenHash string, now time.Time) (domainaccount.Account, bool, error)
	CreateDeviceAuthorization(ctx context.Context, authorization domainaccount.DeviceAuthorization) error
	ApproveDeviceAuthorization(ctx context.Context, userCodeHash, accountID string, now time.Time) (bool, error)
	ConsumeDeviceAuthorization(ctx context.Context, deviceCodeHash string, installation domainclient.ClientInstallation, now time.Time) (string, bool, error)
	ConsumeResolutionQuota(ctx context.Context, clientID, idempotencyKey, period string, now time.Time) (bool, error)
	UpdateSubscription(ctx context.Context, accountID, plan, status string, quota int64, actorUserID string, now time.Time) (bool, error)
	ListAccounts(ctx context.Context, period string) ([]AccountSummary, error)
}

type AccountSummary struct {
	ID                 string `json:"id"`
	Email              string `json:"email"`
	Status             string `json:"status"`
	Plan               string `json:"plan"`
	SubscriptionStatus string `json:"subscriptionStatus"`
	MonthlyQuota       int64  `json:"monthlyQuota"`
	Usage              int64  `json:"usage"`
}

type Service struct {
	store             Store
	dummyPasswordHash string
}

func NewService(store Store) *Service {
	dummy, err := appauth.HashPassword("wildman-account-dummy-password")
	if err != nil {
		panic(err)
	}
	return &Service{store: store, dummyPasswordHash: dummy}
}

type Authentication struct {
	Account domainaccount.Account
	Token   string
}

func (service *Service) Register(ctx context.Context, email, password string) (Authentication, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return Authentication{}, err
	}
	if err := appauth.ValidatePassword(password); err != nil {
		return Authentication{}, ErrInvalidPassword
	}
	passwordHash, err := appauth.HashPassword(password)
	if err != nil {
		return Authentication{}, err
	}
	now := time.Now().UTC()
	account := domainaccount.Account{ID: randomValue(16), Email: email, PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now}
	session, token := newSession(account.ID, now)
	created, err := service.store.CreateAccount(ctx, account, domainaccount.Subscription{AccountID: account.ID, Plan: "free", Status: "active", MonthlyQuota: freeMonthlyQuota, UpdatedAt: now}, session)
	if err != nil {
		return Authentication{}, fmt.Errorf("create account: %w", err)
	}
	if !created {
		return Authentication{}, ErrEmailExists
	}
	return Authentication{Account: account, Token: token}, nil
}

func (service *Service) Login(ctx context.Context, email, password string) (Authentication, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return Authentication{}, ErrInvalidCredentials
	}
	account, found, err := service.store.FindAccountByEmail(ctx, email)
	if err != nil {
		return Authentication{}, err
	}
	hash := account.PasswordHash
	if !found {
		hash = service.dummyPasswordHash
	}
	valid, err := appauth.VerifyPassword(password, hash)
	if err != nil || !valid {
		return Authentication{}, ErrInvalidCredentials
	}
	if account.Status != "active" {
		return Authentication{}, ErrInvalidCredentials
	}
	session, token := newSession(account.ID, time.Now().UTC())
	if err := service.store.CreateAccountSession(ctx, session); err != nil {
		return Authentication{}, err
	}
	return Authentication{Account: account, Token: token}, nil
}

func (service *Service) Authenticate(ctx context.Context, token string) (domainaccount.Account, error) {
	if !strings.HasPrefix(token, "wa_session_") {
		return domainaccount.Account{}, ErrAccountAuthenticationRequired
	}
	account, found, err := service.store.FindAccountBySessionHash(ctx, digest(token), time.Now().UTC())
	if err != nil {
		return domainaccount.Account{}, err
	}
	if !found || account.Status != "active" {
		return domainaccount.Account{}, ErrAccountAuthenticationRequired
	}
	return account, nil
}

type DeviceStart struct {
	DeviceCode string
	UserCode   string
	ExpiresIn  int
	Interval   int
}

func (service *Service) StartDeviceAuthorization(ctx context.Context, clientName string) (DeviceStart, error) {
	clientName = strings.TrimSpace(clientName)
	if clientName == "" || len([]rune(clientName)) > 100 {
		return DeviceStart{}, ErrDeviceCodeInvalid
	}
	now := time.Now().UTC()
	deviceCode := "wd_device_" + randomValue(32)
	userCode := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes(5)))
	authorization := domainaccount.DeviceAuthorization{ID: randomValue(16), DeviceCodeHash: digest(deviceCode), UserCodeHash: digest(userCode), ClientName: clientName, Status: "pending", ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now}
	if err := service.store.CreateDeviceAuthorization(ctx, authorization); err != nil {
		return DeviceStart{}, err
	}
	return DeviceStart{DeviceCode: deviceCode, UserCode: userCode, ExpiresIn: 600, Interval: 5}, nil
}

func (service *Service) ApproveDevice(ctx context.Context, accountID, userCode string) error {
	approved, err := service.store.ApproveDeviceAuthorization(ctx, digest(strings.ToUpper(strings.TrimSpace(userCode))), accountID, time.Now().UTC())
	if err != nil {
		return err
	}
	if !approved {
		return ErrDeviceCodeInvalid
	}
	return nil
}

func (service *Service) PollDevice(ctx context.Context, deviceCode string) (string, error) {
	if !strings.HasPrefix(deviceCode, "wd_device_") {
		return "", ErrDeviceCodeInvalid
	}
	issued, err := domainclient.IssueToken()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	installation := domainclient.ClientInstallation{ID: randomValue(16), TokenPrefix: issued.Prefix, TokenHash: issued.Hash, Status: domainclient.InstallationStatusActive, CreatedAt: now}
	status, found, err := service.store.ConsumeDeviceAuthorization(ctx, digest(deviceCode), installation, now)
	if err != nil {
		return "", err
	}
	if !found {
		return "", ErrDeviceCodeInvalid
	}
	switch status {
	case "pending":
		return "", ErrDeviceAuthorizationPending
	case "consumed":
		return "", ErrDeviceAuthorizationConsumed
	}
	return issued.Value, nil
}

func (service *Service) ConsumeQuota(ctx context.Context, clientID, idempotencyKey string) error {
	now := time.Now().UTC()
	allowed, err := service.store.ConsumeResolutionQuota(ctx, clientID, idempotencyKey, now.Format("2006-01"), now)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrQuotaExceeded
	}
	return nil
}

func (service *Service) UpdateSubscription(ctx context.Context, accountID, plan, status string, quota int64, actorUserID string) error {
	if (plan != "free" && plan != "pro") || (status != "active" && status != "past_due" && status != "canceled") || quota < 0 {
		return ErrSubscriptionInvalid
	}
	found, err := service.store.UpdateSubscription(ctx, accountID, plan, status, quota, actorUserID, time.Now().UTC())
	if err != nil {
		return err
	}
	if !found {
		return ErrSubscriptionInvalid
	}
	return nil
}

func (service *Service) Accounts(ctx context.Context) ([]AccountSummary, error) {
	return service.store.ListAccounts(ctx, time.Now().UTC().Format("2006-01"))
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || len(value) > 254 {
		return "", ErrInvalidEmail
	}
	return value, nil
}

func newSession(accountID string, now time.Time) (domainaccount.Session, string) {
	token := "wa_session_" + randomValue(32)
	return domainaccount.Session{ID: randomValue(16), AccountID: accountID, TokenHash: digest(token), ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now}, token
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
func randomBytes(length int) []byte {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	return value
}
func randomValue(length int) string { return base64.RawURLEncoding.EncodeToString(randomBytes(length)) }
