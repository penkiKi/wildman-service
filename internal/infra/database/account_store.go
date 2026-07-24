package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	appaccount "wildman-service/internal/app/account"
	domainaccount "wildman-service/internal/domain/account"
	domainclient "wildman-service/internal/domain/client"
)

type AccountStore struct{ database *DB }

func NewAccountStore(database *DB) *AccountStore { return &AccountStore{database: database} }

func (store *AccountStore) CreateAccount(ctx context.Context, account domainaccount.Account, subscription domainaccount.Subscription, session domainaccount.Session) (bool, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer transaction.Rollback()
	result, err := transaction.ExecContext(ctx, `
		INSERT INTO customer_accounts (id, email, password_hash, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (email) DO NOTHING
	`, account.ID, account.Email, account.PasswordHash, account.Status, account.CreatedAt.Format(time.RFC3339Nano), account.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return false, err
	}
	if _, err := transaction.ExecContext(ctx, `INSERT INTO subscriptions (account_id, plan, status, monthly_quota, updated_at) VALUES (?, ?, ?, ?, ?)`, subscription.AccountID, subscription.Plan, subscription.Status, subscription.MonthlyQuota, subscription.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		return false, err
	}
	if err := createAccountSession(ctx, transaction, session); err != nil {
		return false, err
	}
	if err := insertAuditEvent(ctx, transaction, "", "account.registered", "account", account.ID, account.CreatedAt); err != nil {
		return false, err
	}
	if err := transaction.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (store *AccountStore) FindAccountByEmail(ctx context.Context, email string) (domainaccount.Account, bool, error) {
	return scanAccountResult(store.database.QueryRowContext(ctx, `SELECT id, email, password_hash, status, created_at, updated_at FROM customer_accounts WHERE email = ?`, email))
}

func (store *AccountStore) CreateAccountSession(ctx context.Context, session domainaccount.Session) error {
	_, err := store.database.ExecContext(ctx, `INSERT INTO customer_sessions (id, account_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`, session.ID, session.AccountID, session.TokenHash, session.ExpiresAt.Format(time.RFC3339Nano), session.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func createAccountSession(ctx context.Context, transaction *Tx, session domainaccount.Session) error {
	_, err := transaction.ExecContext(ctx, `INSERT INTO customer_sessions (id, account_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`, session.ID, session.AccountID, session.TokenHash, session.ExpiresAt.Format(time.RFC3339Nano), session.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (store *AccountStore) FindAccountBySessionHash(ctx context.Context, tokenHash string, now time.Time) (domainaccount.Account, bool, error) {
	return scanAccountResult(store.database.QueryRowContext(ctx, `
		SELECT a.id, a.email, a.password_hash, a.status, a.created_at, a.updated_at
		FROM customer_sessions AS s JOIN customer_accounts AS a ON a.id = s.account_id
		WHERE s.token_hash = ? AND s.expires_at > ?
	`, tokenHash, now.Format(time.RFC3339Nano)))
}

func scanAccountResult(scanner rowScanner) (domainaccount.Account, bool, error) {
	var account domainaccount.Account
	var createdAt, updatedAt string
	err := scanner.Scan(&account.ID, &account.Email, &account.PasswordHash, &account.Status, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return account, false, nil
	}
	if err != nil {
		return account, false, err
	}
	if account.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return account, false, err
	}
	if account.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return account, false, err
	}
	return account, true, nil
}

func (store *AccountStore) CreateDeviceAuthorization(ctx context.Context, authorization domainaccount.DeviceAuthorization) error {
	_, err := store.database.ExecContext(ctx, `
		INSERT INTO device_authorizations (id, device_code_hash, user_code_hash, client_name, status, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, authorization.ID, authorization.DeviceCodeHash, authorization.UserCodeHash, authorization.ClientName, authorization.Status, authorization.ExpiresAt.Format(time.RFC3339Nano), authorization.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (store *AccountStore) ApproveDeviceAuthorization(ctx context.Context, userCodeHash, accountID string, now time.Time) (bool, error) {
	result, err := store.database.ExecContext(ctx, `
		UPDATE device_authorizations SET status = 'approved', account_id = ?, approved_at = ?
		WHERE user_code_hash = ? AND status = 'pending' AND expires_at > ?
	`, accountID, now.Format(time.RFC3339Nano), userCodeHash, now.Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (store *AccountStore) ConsumeDeviceAuthorization(ctx context.Context, deviceCodeHash string, installation domainclient.ClientInstallation, now time.Time) (string, bool, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return "", false, err
	}
	defer transaction.Rollback()
	lock := ""
	if store.database.Dialect() == DialectPostgres {
		lock = " FOR UPDATE"
	}
	var status, accountID, clientName, expiresAt string
	err = transaction.QueryRowContext(ctx, `SELECT status, COALESCE(account_id, ''), client_name, expires_at FROM device_authorizations WHERE device_code_hash = ?`+lock, deviceCodeHash).Scan(&status, &accountID, &clientName, &expiresAt)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	expires, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil || !now.Before(expires) {
		return "", false, nil
	}
	if status != "approved" {
		return status, true, nil
	}
	var operatorUserID string
	if err := transaction.QueryRowContext(ctx, `SELECT id FROM users ORDER BY created_at LIMIT 1`).Scan(&operatorUserID); err != nil {
		return "", false, fmt.Errorf("operator administrator is required before device authorization: %w", err)
	}
	installation.Name, installation.AccountID, installation.CreatedByUserID = clientName, accountID, operatorUserID
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO client_installations (id, name, token_prefix, token_hash, status, created_by_user_id, account_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, installation.ID, installation.Name, installation.TokenPrefix, installation.TokenHash, installation.Status, installation.CreatedByUserID, installation.AccountID, installation.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return "", false, err
	}
	if _, err := transaction.ExecContext(ctx, `UPDATE device_authorizations SET status = 'consumed', consumed_at = ? WHERE device_code_hash = ? AND status = 'approved'`, now.Format(time.RFC3339Nano), deviceCodeHash); err != nil {
		return "", false, err
	}
	if err := insertAuditEvent(ctx, transaction, "", "device.authorized", "client", installation.ID, now); err != nil {
		return "", false, err
	}
	if err := transaction.Commit(); err != nil {
		return "", false, err
	}
	return "approved", true, nil
}

func (store *AccountStore) ConsumeResolutionQuota(ctx context.Context, clientID, idempotencyKey, period string, now time.Time) (bool, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer transaction.Rollback()
	var accountID sql.NullString
	if err := transaction.QueryRowContext(ctx, `SELECT account_id FROM client_installations WHERE id = ?`, clientID).Scan(&accountID); err != nil {
		return false, err
	}
	if !accountID.Valid {
		return true, transaction.Commit()
	}
	var consumed bool
	if err := transaction.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM quota_consumptions WHERE client_id = ? AND idempotency_key = ?)`, clientID, idempotencyKey).Scan(&consumed); err != nil {
		return false, err
	}
	if consumed {
		return true, transaction.Commit()
	}
	var status string
	var quota int64
	if err := transaction.QueryRowContext(ctx, `SELECT status, monthly_quota FROM subscriptions WHERE account_id = ?`, accountID.String).Scan(&status, &quota); err != nil {
		return false, err
	}
	if status != "active" || quota <= 0 {
		return false, nil
	}
	consumptionResult, err := transaction.ExecContext(ctx, `INSERT INTO quota_consumptions (client_id, idempotency_key, account_id, period, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT (client_id, idempotency_key) DO NOTHING`, clientID, idempotencyKey, accountID.String, period, now.Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	consumptionRows, err := consumptionResult.RowsAffected()
	if err != nil {
		return false, err
	}
	if consumptionRows == 0 {
		return true, transaction.Commit()
	}
	result, err := transaction.ExecContext(ctx, `
		INSERT INTO monthly_usage (account_id, period, resolutions) VALUES (?, ?, 1)
		ON CONFLICT (account_id, period) DO UPDATE SET resolutions = monthly_usage.resolutions + 1
		WHERE monthly_usage.resolutions < ?
	`, accountID.String, period, quota)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return false, err
	}
	return true, transaction.Commit()
}

func (store *AccountStore) UpdateSubscription(ctx context.Context, accountID, plan, status string, quota int64, actorUserID string, now time.Time) (bool, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer transaction.Rollback()
	result, err := transaction.ExecContext(ctx, `UPDATE subscriptions SET plan = ?, status = ?, monthly_quota = ?, updated_at = ? WHERE account_id = ?`, plan, status, quota, now.Format(time.RFC3339Nano), accountID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return false, err
	}
	if err := insertAuditEvent(ctx, transaction, actorUserID, "subscription.updated", "account", accountID, now); err != nil {
		return false, err
	}
	return true, transaction.Commit()
}

func (store *AccountStore) ListAccounts(ctx context.Context, period string) ([]appaccount.AccountSummary, error) {
	rows, err := store.database.QueryContext(ctx, `
		SELECT a.id, a.email, a.status, s.plan, s.status, s.monthly_quota, COALESCE(u.resolutions, 0)
		FROM customer_accounts AS a JOIN subscriptions AS s ON s.account_id = a.id
		LEFT JOIN monthly_usage AS u ON u.account_id = a.id AND u.period = ?
		ORDER BY a.created_at DESC, a.id DESC
	`, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]appaccount.AccountSummary, 0)
	for rows.Next() {
		var item appaccount.AccountSummary
		if err := rows.Scan(&item.ID, &item.Email, &item.Status, &item.Plan, &item.SubscriptionStatus, &item.MonthlyQuota, &item.Usage); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
