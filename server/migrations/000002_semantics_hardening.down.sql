DROP INDEX IF EXISTS idx_otk_username_consumed;
CREATE INDEX IF NOT EXISTS idx_otk_username_consumed ON one_time_keys(username, consumed);

ALTER TABLE messages DROP CONSTRAINT IF EXISTS fk_messages_recipient;
ALTER TABLE messages DROP CONSTRAINT IF EXISTS fk_messages_sender;

ALTER TABLE one_time_keys DROP CONSTRAINT IF EXISTS otk_unique_username_key;
ALTER TABLE one_time_keys DROP COLUMN IF EXISTS consumed_at;

ALTER TABLE users DROP COLUMN IF EXISTS token_hash;
