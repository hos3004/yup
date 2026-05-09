-- M10: Push Notifications (FCM) — Rollback
DROP INDEX IF EXISTS idx_device_tokens_username;
DROP TABLE IF EXISTS device_tokens;
