package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yup/server/internal/model"
)

// PostgresStore implements DataStore backed by PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

const migrationUpSQL = `
CREATE TABLE IF NOT EXISTS users (
    username     VARCHAR(64)  PRIMARY KEY,
    auth_token   VARCHAR(128) NOT NULL UNIQUE,
    token_hash   VARCHAR(64)  NOT NULL DEFAULT '',
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
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
    id           BIGSERIAL    PRIMARY KEY,
    username     VARCHAR(64)  NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    key_value    VARCHAR(256) NOT NULL,
    consumed     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_otk_username_consumed ON one_time_keys(username, consumed);

CREATE TABLE IF NOT EXISTS messages (
    id                 VARCHAR(64)  PRIMARY KEY,
    sender_username    VARCHAR(64)  NOT NULL,
    recipient_username VARCHAR(64)  NOT NULL,
    ciphertext         TEXT         NOT NULL,
    message_type       INT          NOT NULL DEFAULT 0,
    sender_curve_key   VARCHAR(256) NOT NULL DEFAULT '',
    status             VARCHAR(32)  NOT NULL DEFAULT 'pending',
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    delivered_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_messages_recipient_status ON messages(recipient_username, status);
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_username);

-- M8.1: Add columns, indexes, and constraints for existing databases
ALTER TABLE one_time_keys ADD COLUMN IF NOT EXISTS consumed_at TIMESTAMPTZ;
DROP INDEX IF EXISTS idx_otk_username_consumed;
CREATE INDEX IF NOT EXISTS idx_otk_username_consumed ON one_time_keys(username, consumed, consumed_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_otk_unique_username_key ON one_time_keys(username, key_value);
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_hash VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE messages DROP CONSTRAINT IF EXISTS fk_messages_sender;
ALTER TABLE messages ADD CONSTRAINT fk_messages_sender FOREIGN KEY (sender_username) REFERENCES users(username) ON DELETE CASCADE;
ALTER TABLE messages DROP CONSTRAINT IF EXISTS fk_messages_recipient;
ALTER TABLE messages ADD CONSTRAINT fk_messages_recipient FOREIGN KEY (recipient_username) REFERENCES users(username) ON DELETE CASCADE;

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

`

// NewPostgresStore creates a new PostgresStore, connects to the database,
// and runs schema migrations.
func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	if _, err := pool.Exec(ctx, migrationUpSQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: migration: %w", err)
	}

	// Reset delivered messages to pending on startup (retry after restart)
	if _, err := pool.Exec(ctx, `UPDATE messages SET status = 'pending', delivered_at = NULL WHERE status = 'delivered'`); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: reset delivered: %w", err)
	}

	return &PostgresStore{pool: pool}, nil
}

// Close shuts down the connection pool.
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// Pool returns the underlying connection pool (for testing).
func (s *PostgresStore) Pool() *pgxpool.Pool {
	return s.pool
}

// ─── User methods ──────────────────────────────────────────

func (s *PostgresStore) RegisterUser(username string) (*model.User, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	tokenHash := sha256Hex(token)
	now := time.Now().UTC()

	_, err = s.pool.Exec(context.Background(),
		`INSERT INTO users (username, auth_token, token_hash, created_at) VALUES ($1, $2, $3, $4)`,
		username, token, tokenHash, now,
	)
	if err != nil {
		if isPGDuplicate(err) {
			return nil, fmt.Errorf("username already exists")
		}
		return nil, fmt.Errorf("register user: %w", err)
	}

	return &model.User{
		Username:  username,
		AuthToken: token,
		CreatedAt: now,
	}, nil
}

func (s *PostgresStore) GetUser(username string) (*model.User, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u model.User
	err := s.pool.QueryRow(ctx,
		`SELECT username, auth_token, display_name, created_at FROM users WHERE username = $1`,
		username,
	).Scan(&u.Username, &u.AuthToken, &u.DisplayName, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false
		}
		return nil, false
	}
	return &u, true
}

func (s *PostgresStore) ValidateToken(token string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokenHash := sha256Hex(token)
	var username string
	err := s.pool.QueryRow(ctx,
		`SELECT username FROM users WHERE token_hash = $1`,
		tokenHash,
	).Scan(&username)
	if err != nil {
		return "", false
	}
	return username, true
}

// ─── Key Bundle methods ────────────────────────────────────

func (s *PostgresStore) UploadKeyBundle(username string, bundle *model.KeyBundle) (*model.KeyBundle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deviceID, err := generateID()
	if err != nil {
		return nil, err
	}
	bundle.DeviceID = deviceID
	now := time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("upload key bundle: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Upsert key bundle
	_, err = tx.Exec(ctx,
		`INSERT INTO key_bundles (username, device_id, curve_key, ed_key, signature, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)
		 ON CONFLICT (username) DO UPDATE SET
		   device_id = EXCLUDED.device_id,
		   curve_key = EXCLUDED.curve_key,
		   ed_key    = EXCLUDED.ed_key,
		   signature = EXCLUDED.signature,
		   updated_at = EXCLUDED.updated_at`,
		username, deviceID, bundle.CurveKey, bundle.EdKey, bundle.Signature, now,
	)
	if err != nil {
		return nil, fmt.Errorf("upload key bundle: upsert: %w", err)
	}

	// Delete old OTKs for this user
	if _, err := tx.Exec(ctx,
		`DELETE FROM one_time_keys WHERE username = $1`, username,
	); err != nil {
		return nil, fmt.Errorf("upload key bundle: delete old otks: %w", err)
	}

	// Insert new OTKs
	for _, otk := range bundle.OneTimeKeys {
		if _, err := tx.Exec(ctx,
			`INSERT INTO one_time_keys (username, key_value, created_at) VALUES ($1, $2, $3)`,
			username, otk, now,
		); err != nil {
			return nil, fmt.Errorf("upload key bundle: insert otk: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("upload key bundle: commit: %w", err)
	}

	return bundle, nil
}

func (s *PostgresStore) GetKeyBundle(username string) (*model.KeyBundle, bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get the key bundle
	var b model.KeyBundle
	err := s.pool.QueryRow(ctx,
		`SELECT device_id, curve_key, ed_key, signature FROM key_bundles WHERE username = $1`,
		username,
	).Scan(&b.DeviceID, &b.CurveKey, &b.EdKey, &b.Signature)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ""
		}
		return nil, false, ""
	}

	// Consume one OTK in a sub-transaction
	var otkID int64
	var chosenOTK string
	var remaining string

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, false, ""
	}
	defer tx.Rollback(ctx)

	// Find one unconsumed OTK by id
	err = tx.QueryRow(ctx,
		`SELECT id, key_value FROM one_time_keys
		 WHERE username = $1 AND consumed = FALSE
		 ORDER BY id ASC LIMIT 1
		 FOR UPDATE SKIP LOCKED`,
		username,
	).Scan(&otkID, &chosenOTK)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ""
		}
		remaining = "no_otk_available"
	}

	if chosenOTK != "" {
		_, err = tx.Exec(ctx,
			`UPDATE one_time_keys SET consumed = TRUE, consumed_at = NOW() WHERE id = $1`,
			otkID,
		)
		if err != nil {
			return nil, false, ""
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, ""
	}

	respBundle := &model.KeyBundle{
		DeviceID:    b.DeviceID,
		CurveKey:    b.CurveKey,
		EdKey:       b.EdKey,
		OneTimeKeys: []string{},
		Signature:   b.Signature,
	}
	if chosenOTK != "" {
		respBundle.OneTimeKeys = []string{chosenOTK}
	}

	return respBundle, true, remaining
}

func (s *PostgresStore) GetCurveKey(username string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var curveKey string
	err := s.pool.QueryRow(ctx,
		`SELECT curve_key FROM key_bundles WHERE username = $1`,
		username,
	).Scan(&curveKey)
	if err != nil {
		return "", false
	}
	return curveKey, true
}

func (s *PostgresStore) AvailableOTKCount(username string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM one_time_keys WHERE username = $1 AND consumed = FALSE`,
		username,
	).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// ─── Message methods ───────────────────────────────────────

func (s *PostgresStore) StoreMessage(sender, recipient, ciphertext string, msgType int, senderKey string) (*model.Envelope, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify recipient exists
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`, recipient,
	).Scan(&exists)
	if err != nil || !exists {
		return nil, fmt.Errorf("recipient not found")
	}

	msgID, err := generateID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	_, err = s.pool.Exec(ctx,
		`INSERT INTO messages (id, sender_username, recipient_username, ciphertext, message_type, sender_curve_key, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)`,
		msgID, sender, recipient, ciphertext, msgType, senderKey, now,
	)
	if err != nil {
		return nil, fmt.Errorf("store message: %w", err)
	}

	return &model.Envelope{
		ID:             msgID,
		SenderUsername: sender,
		Ciphertext:     ciphertext,
		MessageType:    msgType,
		SenderCurveKey: senderKey,
		Status:         "pending",
		CreatedAt:      now,
	}, nil
}

func (s *PostgresStore) GetPendingEnvelopes(username string) []*model.Envelope {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil
	}
	defer tx.Rollback(ctx)

	// Fetch pending messages
	rows, err := tx.Query(ctx,
		`SELECT id, sender_username, ciphertext, message_type, sender_curve_key, status, created_at
		 FROM messages
		 WHERE recipient_username = $1 AND status = 'pending'
		 ORDER BY created_at ASC`,
		username,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	envs := make([]*model.Envelope, 0)
	now := time.Now().UTC()
	var ids []string

	for rows.Next() {
		var env model.Envelope
		var status string
		if err := rows.Scan(&env.ID, &env.SenderUsername, &env.Ciphertext, &env.MessageType, &env.SenderCurveKey, &status, &env.CreatedAt); err != nil {
			return nil
		}
		env.Status = "delivered"
		envs = append(envs, &env)
		ids = append(ids, env.ID)
	}

	// Mark as delivered
	if len(ids) > 0 {
		_, err = tx.Exec(ctx,
			`UPDATE messages SET status = 'delivered', delivered_at = $1
			 WHERE id = ANY($2) AND status = 'pending'`,
			now, ids,
		)
		if err != nil {
			return nil
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil
	}

	return envs
}

func (s *PostgresStore) AckMessage(messageID, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()

	result, err := s.pool.Exec(ctx,
		`UPDATE messages
		 SET status = 'received', delivered_at = $1
		 WHERE id = $2 AND recipient_username = $3 AND status IN ('pending', 'delivered')`,
		now, messageID, username,
	)
	if err != nil {
		return fmt.Errorf("ack message: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if message exists at all
		var exists bool
		err = s.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1)`, messageID,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("ack message: check exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("message not found")
		}
		return fmt.Errorf("not the recipient of this message")
	}

	return nil
}

func (s *PostgresStore) GetSentMessages(username string) []*model.Envelope {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT id, sender_username, ciphertext, message_type, sender_curve_key, status, created_at
		 FROM messages
		 WHERE sender_username = $1
		 ORDER BY created_at DESC`,
		username,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	envs := make([]*model.Envelope, 0)
	for rows.Next() {
		var env model.Envelope
		if err := rows.Scan(&env.ID, &env.SenderUsername, &env.Ciphertext, &env.MessageType, &env.SenderCurveKey, &env.Status, &env.CreatedAt); err != nil {
			return nil
		}
		envs = append(envs, &env)
	}
	return envs
}

func (s *PostgresStore) DeleteAllUserData(username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("delete user data: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Verify user exists
	var exists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`, username,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("delete user data: %w", err)
	}
	if !exists {
		return fmt.Errorf("user not found")
	}

	// Delete messages where user is sender or recipient
	if _, err := tx.Exec(ctx,
		`DELETE FROM messages WHERE sender_username = $1 OR recipient_username = $1`, username,
	); err != nil {
		return fmt.Errorf("delete user data: messages: %w", err)
	}

	// CASCADE will handle key_bundles, one_time_keys
	if _, err := tx.Exec(ctx,
		`DELETE FROM users WHERE username = $1`, username,
	); err != nil {
		return fmt.Errorf("delete user data: user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("delete user data: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) PurgeExpiredMessages(maxAge time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx,
		`DELETE FROM messages WHERE created_at < NOW() - ($1 || ' seconds')::interval`,
		int(maxAge.Seconds()),
	)
	if err != nil {
		return fmt.Errorf("purge expired messages: %w", err)
	}
	return nil
}

// ─── Device Token methods ────────────────────────────────────

func (s *PostgresStore) RegisterDeviceToken(username, token, platform string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx,
		`INSERT INTO device_tokens (username, token, platform, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 ON CONFLICT (username, token) DO UPDATE SET
		   platform = EXCLUDED.platform,
		   updated_at = NOW()`,
		username, token, platform,
	)
	if err != nil {
		return fmt.Errorf("register device token: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetDeviceTokens(username string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT token FROM device_tokens WHERE username = $1`, username,
	)
	if err != nil {
		return nil, fmt.Errorf("get device tokens: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("get device tokens: scan: %w", err)
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// ─── Helpers ───────────────────────────────────────────────

func isPGDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
