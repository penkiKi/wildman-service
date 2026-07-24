CREATE TABLE customer_accounts (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE customer_sessions (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES customer_accounts(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE subscriptions (
    account_id TEXT PRIMARY KEY REFERENCES customer_accounts(id) ON DELETE CASCADE,
    plan TEXT NOT NULL CHECK (plan IN ('free', 'pro')),
    status TEXT NOT NULL CHECK (status IN ('active', 'past_due', 'canceled')),
    monthly_quota INTEGER NOT NULL CHECK (monthly_quota >= 0),
    updated_at TEXT NOT NULL
);

CREATE TABLE monthly_usage (
    account_id TEXT NOT NULL REFERENCES customer_accounts(id) ON DELETE CASCADE,
    period TEXT NOT NULL,
    resolutions INTEGER NOT NULL DEFAULT 0 CHECK (resolutions >= 0),
    PRIMARY KEY (account_id, period)
);

ALTER TABLE client_installations ADD COLUMN account_id TEXT REFERENCES customer_accounts(id);
CREATE INDEX client_installations_account_id_idx ON client_installations(account_id);

CREATE TABLE device_authorizations (
    id TEXT PRIMARY KEY,
    device_code_hash TEXT NOT NULL UNIQUE,
    user_code_hash TEXT NOT NULL UNIQUE,
    client_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'consumed')),
    account_id TEXT REFERENCES customer_accounts(id),
    expires_at TEXT NOT NULL,
    approved_at TEXT,
    consumed_at TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX device_authorizations_expires_at_idx ON device_authorizations(expires_at);

CREATE TABLE quota_consumptions (
    client_id TEXT NOT NULL REFERENCES client_installations(id) ON DELETE CASCADE,
    idempotency_key TEXT NOT NULL,
    account_id TEXT NOT NULL REFERENCES customer_accounts(id) ON DELETE CASCADE,
    period TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (client_id, idempotency_key)
);
