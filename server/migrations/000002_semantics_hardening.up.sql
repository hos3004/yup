-- M8.1: PostgreSQL Semantics Hardening
-- Add consumed_at to one_time_keys, unique constraint, FKs, token hashing

ALTER TABLE one_time_keys ADD COLUMN IF NOT EXISTS consumed_at TIMESTAMPTZ;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'otk_unique_username_key'
    ) THEN
        ALTER TABLE one_time_keys ADD CONSTRAINT otk_unique_username_key UNIQUE (username, key_value);
    END IF;
END $$;

-- Add foreign keys for messages
ALTER TABLE messages ADD CONSTRAINT fk_messages_sender
    FOREIGN KEY (sender_username) REFERENCES users(username) ON DELETE CASCADE;
ALTER TABLE messages ADD CONSTRAINT fk_messages_recipient
    FOREIGN KEY (recipient_username) REFERENCES users(username) ON DELETE CASCADE;

-- Add token_hash column (will be populated on next registration/login)
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_hash VARCHAR(64) NOT NULL DEFAULT '';

-- Update index to include consumed_at for ordering
DROP INDEX IF EXISTS idx_otk_username_consumed;
CREATE INDEX IF NOT EXISTS idx_otk_username_consumed ON one_time_keys(username, consumed, consumed_at);
