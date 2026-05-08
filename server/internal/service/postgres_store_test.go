package service

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yup/server/internal/model"
)

// getTestPool returns a pgxpool connected to the test database, or nil
// if DATABASE_URL_TEST is not set or connection fails.
func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("Skipping integration test: set DATABASE_URL_TEST")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test db: %v", err)
	}

	return pool
}

// setupTestStore creates a PostgresStore on a clean schema for testing.
func setupTestStore(t *testing.T) *PostgresStore {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("Skipping integration test: set DATABASE_URL_TEST")
	}

	store, err := NewPostgresStore(dbURL)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}

	// Clean all tables before each test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = store.pool.Exec(ctx, `DELETE FROM messages`)
	_, _ = store.pool.Exec(ctx, `DELETE FROM one_time_keys`)
	_, _ = store.pool.Exec(ctx, `DELETE FROM key_bundles`)
	_, _ = store.pool.Exec(ctx, `DELETE FROM users`)

	return store
}

func TestPostgresStore_RegisterUser(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	user, err := store.RegisterUser("alice")
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("got username %q, want %q", user.Username, "alice")
	}
	if user.AuthToken == "" {
		t.Errorf("expected non-empty auth token")
	}
	if user.CreatedAt.IsZero() {
		t.Errorf("expected non-zero CreatedAt")
	}
}

func TestPostgresStore_RegisterUser_Duplicate(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	_, err := store.RegisterUser("alice")
	if err != nil {
		t.Fatalf("first register: %v", err)
	}

	_, err = store.RegisterUser("alice")
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
	if !strings.Contains(err.Error(), "username already exists") {
		t.Errorf("got error %q, want 'username already exists'", err.Error())
	}
}

func TestPostgresStore_GetUser(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Non-existent user
	_, ok := store.GetUser("nonexistent")
	if ok {
		t.Error("expected false for nonexistent user")
	}

	// Register and fetch
	store.RegisterUser("alice")
	user, ok := store.GetUser("alice")
	if !ok {
		t.Fatal("expected true for existing user")
	}
	if user.Username != "alice" {
		t.Errorf("got username %q", user.Username)
	}
}

func TestPostgresStore_ValidateToken(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	user, err := store.RegisterUser("alice")
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	// Valid token
	username, ok := store.ValidateToken(user.AuthToken)
	if !ok {
		t.Fatal("expected valid token")
	}
	if username != "alice" {
		t.Errorf("got username %q", username)
	}

	// Invalid token
	_, ok = store.ValidateToken("invalidtoken")
	if ok {
		t.Error("expected false for invalid token")
	}

	// Empty token
	_, ok = store.ValidateToken("")
	if ok {
		t.Error("expected false for empty token")
	}
}

func TestPostgresStore_UploadKeyBundle(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")

	bundle, err := store.UploadKeyBundle("alice", &model.KeyBundle{
		CurveKey:    "base64curvekey==",
		EdKey:       "base64edkey==",
		OneTimeKeys: []string{"otk1", "otk2", "otk3"},
	})
	if err != nil {
		t.Fatalf("UploadKeyBundle: %v", err)
	}
	if bundle.DeviceID == "" {
		t.Error("expected non-empty DeviceID")
	}
	if bundle.CurveKey != "base64curvekey==" {
		t.Errorf("got CurveKey %q", bundle.CurveKey)
	}
}

func TestPostgresStore_UploadKeyBundle_UserNotFound(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	_, err := store.UploadKeyBundle("nonexistent", &model.KeyBundle{
		CurveKey: "ck", EdKey: "ek",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestPostgresStore_GetKeyBundle(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")
	store.UploadKeyBundle("alice", &model.KeyBundle{
		CurveKey:    "ck",
		EdKey:       "ek",
		OneTimeKeys: []string{"otk1", "otk2"},
	})

	// First fetch should get otk1
	bundle, ok, remaining := store.GetKeyBundle("alice")
	if !ok {
		t.Fatal("expected true for existing bundle")
	}
	if len(bundle.OneTimeKeys) != 1 {
		t.Errorf("expected 1 OTK, got %d", len(bundle.OneTimeKeys))
	}
	if remaining != "" {
		t.Errorf("expected no remaining flag, got %q", remaining)
	}
	firstOTK := bundle.OneTimeKeys[0]

	// Second fetch should get otk2
	bundle2, ok2, _ := store.GetKeyBundle("alice")
	if !ok2 {
		t.Fatal("expected true on second fetch")
	}
	if len(bundle2.OneTimeKeys) != 1 {
		t.Errorf("expected 1 OTK, got %d", len(bundle2.OneTimeKeys))
	}
	secondOTK := bundle2.OneTimeKeys[0]

	if firstOTK == secondOTK {
		t.Errorf("OTK was not consumed: got same OTK twice: %s", firstOTK)
	}
}

func TestPostgresStore_GetKeyBundle_NoOTK(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")
	store.UploadKeyBundle("alice", &model.KeyBundle{
		CurveKey:    "ck",
		EdKey:       "ek",
		OneTimeKeys: []string{"otk1"},
	})

	// Consume the only OTK
	store.GetKeyBundle("alice")

	// No OTKs left
	bundle, ok, remaining := store.GetKeyBundle("alice")
	if !ok {
		t.Fatal("expected bundle to still exist")
	}
	if remaining != "no_otk_available" {
		t.Errorf("expected 'no_otk_available', got %q", remaining)
	}
	if len(bundle.OneTimeKeys) != 0 {
		t.Errorf("expected 0 OTKs when exhausted, got %d", len(bundle.OneTimeKeys))
	}
}

func TestPostgresStore_GetKeyBundle_UserNotFound(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	_, ok, _ := store.GetKeyBundle("nonexistent")
	if ok {
		t.Error("expected false for nonexistent user")
	}
}

func TestPostgresStore_AvailableOTKCount(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")

	// No bundle yet
	if c := store.AvailableOTKCount("alice"); c != 0 {
		t.Errorf("expected 0, got %d", c)
	}

	store.UploadKeyBundle("alice", &model.KeyBundle{
		CurveKey: "ck", EdKey: "ek",
		OneTimeKeys: []string{"a", "b", "c"},
	})

	if c := store.AvailableOTKCount("alice"); c != 3 {
		t.Errorf("expected 3, got %d", c)
	}

	// Consume one
	store.GetKeyBundle("alice")
	if c := store.AvailableOTKCount("alice"); c != 2 {
		t.Errorf("expected 2 after consuming one, got %d", c)
	}
}

func TestPostgresStore_StoreMessage(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")
	store.RegisterUser("bob")
	store.UploadKeyBundle("bob", &model.KeyBundle{CurveKey: "ck", EdKey: "ek"})

	env, err := store.StoreMessage("alice", "bob", "YWIxMg==", 0, "alice_key")
	if err != nil {
		t.Fatalf("StoreMessage: %v", err)
	}
	if env.ID == "" {
		t.Error("expected non-empty ID")
	}
	if env.SenderUsername != "alice" {
		t.Errorf("got sender %q", env.SenderUsername)
	}
	if env.Status != "pending" {
		t.Errorf("got status %q, want 'pending'", env.Status)
	}
}

func TestPostgresStore_StoreMessage_RecipientNotFound(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")

	_, err := store.StoreMessage("alice", "nonexistent", "YWIxMg==", 0, "k")
	if err == nil {
		t.Fatal("expected error for nonexistent recipient")
	}
}

func TestPostgresStore_GetPendingEnvelopes(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")
	store.RegisterUser("bob")
	store.UploadKeyBundle("bob", &model.KeyBundle{CurveKey: "ck", EdKey: "ek"})

	// Alice sends to Bob
	store.StoreMessage("alice", "bob", "YWIxMg==", 0, "ak")

	// Bob fetches
	envs := store.GetPendingEnvelopes("bob")
	if len(envs) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(envs))
	}
	if envs[0].Status != "delivered" {
		t.Errorf("expected status 'delivered', got %q", envs[0].Status)
	}

	// Second fetch should be empty
	envs2 := store.GetPendingEnvelopes("bob")
	if len(envs2) != 0 {
		t.Errorf("expected 0 on second fetch, got %d", len(envs2))
	}
}

func TestPostgresStore_AckMessage(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")
	store.RegisterUser("bob")
	store.UploadKeyBundle("bob", &model.KeyBundle{CurveKey: "ck", EdKey: "ek"})

	env, _ := store.StoreMessage("alice", "bob", "YWIxMg==", 0, "ak")
	store.GetPendingEnvelopes("bob") // marks as delivered

	err := store.AckMessage(env.ID, "bob")
	if err != nil {
		t.Fatalf("AckMessage: %v", err)
	}

	// Verify via sent messages
	sent := store.GetSentMessages("alice")
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent, got %d", len(sent))
	}
	if sent[0].Status != "received" {
		t.Errorf("expected status 'received', got %q", sent[0].Status)
	}
}

func TestPostgresStore_AckMessage_WrongUser(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")
	store.RegisterUser("bob")
	store.RegisterUser("mallory")
	store.UploadKeyBundle("bob", &model.KeyBundle{CurveKey: "ck", EdKey: "ek"})

	env, _ := store.StoreMessage("alice", "bob", "YWIxMg==", 0, "ak")
	store.GetPendingEnvelopes("bob")

	err := store.AckMessage(env.ID, "mallory")
	if err == nil {
		t.Fatal("expected error for wrong user")
	}
}

func TestPostgresStore_AckMessage_NotFound(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("bob")
	err := store.AckMessage("nonexistent-id", "bob")
	if err == nil {
		t.Fatal("expected error for nonexistent message")
	}
}

func TestPostgresStore_GetSentMessages(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")
	store.RegisterUser("bob")
	store.UploadKeyBundle("bob", &model.KeyBundle{CurveKey: "ck", EdKey: "ek"})

	store.StoreMessage("alice", "bob", "msg1", 0, "ak")
	store.StoreMessage("alice", "bob", "msg2", 0, "ak")

	sent := store.GetSentMessages("alice")
	if len(sent) != 2 {
		t.Fatalf("expected 2 sent, got %d", len(sent))
	}
}

func TestPostgresStore_DeleteAllUserData(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")
	store.RegisterUser("bob")
	store.UploadKeyBundle("bob", &model.KeyBundle{CurveKey: "ck", EdKey: "ek", OneTimeKeys: []string{"otk1"}})
	store.StoreMessage("alice", "bob", "YWIxMg==", 0, "ak")

	err := store.DeleteAllUserData("bob")
	if err != nil {
		t.Fatalf("DeleteAllUserData: %v", err)
	}

	// Verify bob is gone
	_, ok := store.GetUser("bob")
	if ok {
		t.Error("expected bob to be deleted")
	}

	// Verify bob's bundle is gone
	_, ok, _ = store.GetKeyBundle("bob")
	if ok {
		t.Error("expected bob's key bundle to be deleted")
	}

	// Verify messages by/to bob are gone
	sent := store.GetSentMessages("alice")
	if len(sent) != 0 {
		t.Errorf("expected alice sent messages to bob to be deleted, got %d", len(sent))
	}
}

func TestPostgresStore_DeleteAllUserData_NotFound(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	err := store.DeleteAllUserData("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestPostgresStore_MessageLifecycle(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	store.RegisterUser("alice")
	store.RegisterUser("bob")
	store.UploadKeyBundle("bob", &model.KeyBundle{CurveKey: "ck", EdKey: "ek"})

	// Store message
	env, err := store.StoreMessage("alice", "bob", "YWIxMg==", 0, "ak")
	if err != nil {
		t.Fatalf("StoreMessage: %v", err)
	}
	if env.Status != "pending" {
		t.Errorf("initial status should be 'pending', got %s", env.Status)
	}

	// Fetch (pending -> delivered)
	envs := store.GetPendingEnvelopes("bob")
	if len(envs) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(envs))
	}
	if envs[0].Status != "delivered" {
		t.Errorf("status after fetch should be 'delivered', got %s", envs[0].Status)
	}

	// Ack (delivered -> received)
	err = store.AckMessage(env.ID, "bob")
	if err != nil {
		t.Fatalf("AckMessage: %v", err)
	}

	// Verify via sent messages
	sent := store.GetSentMessages("alice")
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sent))
	}
	if sent[0].Status != "received" {
		t.Errorf("status after ack should be 'received', got %s", sent[0].Status)
	}
}
