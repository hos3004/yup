CREATE TABLE IF NOT EXISTS users (
    username    VARCHAR(64)  PRIMARY KEY,
    auth_token  VARCHAR(128) NOT NULL UNIQUE,
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS key_bundles (
    username    VARCHAR(64)  PRIMARY KEY REFERENCES users(username) ON DELETE CASCADE,
    device_id   VARCHAR(64)  NOT NULL,
    curve_key   VARCHAR(128) NOT NULL,
    ed_key      VARCHAR(128) NOT NULL,
    signature   VARCHAR(256) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS one_time_keys (
    id          BIGSERIAL   PRIMARY KEY,
    username    VARCHAR(64) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    key_value   VARCHAR(256) NOT NULL,
    consumed    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_otk_username_consumed ON one_time_keys(username, consumed);

CREATE TABLE IF NOT EXISTS messages (
    id                VARCHAR(64)  PRIMARY KEY,
    sender_username   VARCHAR(64)  NOT NULL,
    recipient_username VARCHAR(64) NOT NULL,
    ciphertext        TEXT         NOT NULL,
    message_type      INT          NOT NULL DEFAULT 0,
    sender_curve_key  VARCHAR(256) NOT NULL DEFAULT '',
    status            VARCHAR(32)  NOT NULL DEFAULT 'pending',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    delivered_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_messages_recipient_status ON messages(recipient_username, status);
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_username);
