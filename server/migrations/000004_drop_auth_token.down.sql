-- Restore plaintext auth_token column (rollback)
ALTER TABLE users ADD COLUMN auth_token VARCHAR(128) NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS users_auth_token_idx ON users(auth_token);
