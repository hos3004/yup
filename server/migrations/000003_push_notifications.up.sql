-- M10: Push Notifications (FCM)
-- Add device tokens table for push notification delivery

CREATE TABLE IF NOT EXISTS device_tokens (
    id         BIGSERIAL    PRIMARY KEY,
    username   VARCHAR(64)  NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    token      VARCHAR(512) NOT NULL,
    platform   VARCHAR(16)  NOT NULL DEFAULT 'android',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(username, token)
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_username ON device_tokens(username);
