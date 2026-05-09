-- M9/M10 Stabilization: Remove plaintext auth_token column
-- Token validation now uses sha256(token_hash) exclusively.
ALTER TABLE users DROP COLUMN IF EXISTS auth_token;
