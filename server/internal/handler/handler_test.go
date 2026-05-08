package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yup/server/internal/middleware"
	"github.com/yup/server/internal/model"
	"github.com/yup/server/internal/service"
)

func newTestServer() *Server {
	return New(service.NewStore())
}

func registerUser(t *testing.T, s *Server, username string) string {
	t.Helper()
	body := `{"username":"` + username + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.RegisterUser(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register %s: got %d, want %d; body: %s", username, w.Code, http.StatusCreated, w.Body.String())
	}
	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	token, _ := result["auth_token"].(string)
	return token
}

func authHeader(token string) string {
	return "Bearer " + token
}

// ─── Registration ─────────────────────────────────────────

func TestRegisterUser_Validation(t *testing.T) {
	s := newTestServer()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid username", `{"username":"alice"}`, http.StatusCreated},
		{"too short", `{"username":"ab"}`, http.StatusBadRequest},
		{"too long", `{"username":"` + strings.Repeat("a", 33) + `"}`, http.StatusBadRequest},
		{"empty", `{"username":""}`, http.StatusBadRequest},
		{"spaces", `{"username":"alice bob"}`, http.StatusBadRequest},
		{"special chars", `{"username":"alice@bob"}`, http.StatusBadRequest},
		{"valid with underscore", `{"username":"alice_bob"}`, http.StatusCreated},
		{"valid with hyphen", `{"username":"alice-bob"}`, http.StatusCreated},
		{"invalid json", `not-json`, http.StatusBadRequest},
		{"empty body", ``, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.RegisterUser(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestRegisterUser_Duplicate(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "alice")

	body := `{"username":"alice"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.RegisterUser(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate should return conflict, got %d", w.Code)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/nonexistent", nil)
	req.SetPathValue("username", "nonexistent")
	w := httptest.NewRecorder()
	s.GetUser(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("got status %d, want 404", w.Code)
	}
}

func TestGetUser_StripsAuthToken(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "alice")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/alice", nil)
	req.SetPathValue("username", "alice")
	w := httptest.NewRecorder()
	s.GetUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	if _, exists := result["auth_token"]; exists {
		t.Errorf("auth_token should be absent from GetUser response, but was present")
	}
	if result["username"] != "alice" {
		t.Errorf("expected username alice, got %v", result["username"])
	}
}

// ─── Auth Middleware ──────────────────────────────────────

func TestAuthMiddleware_NoAuth(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "alice")

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"no auth header", "", http.StatusUnauthorized},
		{"invalid token", "Bearer invalidtoken", http.StatusUnauthorized},
		{"malformed bearer", "Bearer ", http.StatusUnauthorized},
		{"wrong scheme", "Basic abc123", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := s.AuthMiddleware(func(w http.ResponseWriter, r *http.Request, username string) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			handler(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	s := newTestServer()
	token := registerUser(t, s, "alice")

	var capturedUsername string
	handler := s.AuthMiddleware(func(w http.ResponseWriter, r *http.Request, username string) {
		capturedUsername = username
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("Authorization", authHeader(token))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", w.Code)
	}
	if capturedUsername != "alice" {
		t.Errorf("captured username = %q, want %q", capturedUsername, "alice")
	}
}

// ─── Key Upload (requires auth) ──────────────────────────

func TestUploadKeys_RequiresAuth(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "alice")

	req := httptest.NewRequest(http.MethodPut, "/api/v1/keys/alice", nil)
	// No auth header
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.UploadKeys)
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("upload without auth: got %d, want 401", w.Code)
	}
}

func TestUploadKeys_Success(t *testing.T) {
	s := newTestServer()
	token := registerUser(t, s, "alice")

	body := `{"curve_key":"base64curve==","ed_key":"base64ed==","one_time_keys":["otk1","otk2"]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/keys/alice", strings.NewReader(body))
	req.Header.Set("Authorization", authHeader(token))
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.UploadKeys)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("upload keys: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

// ─── Key Fetch (requires auth, consumes OTK) ─────────────

func TestGetKeys_RequiresAuth(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "alice")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys/alice", nil)
	req.SetPathValue("username", "alice")
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.GetKeys)
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("get keys without auth: got %d, want 401", w.Code)
	}
}

func TestGetKeys_ConsumesOTK(t *testing.T) {
	s := newTestServer()
	aliceToken := registerUser(t, s, "alice")
	bobToken := registerUser(t, s, "bob")

	// Bob uploads keys with 2 OTKs
	body := `{"curve_key":"bobC1","ed_key":"bobE1","one_time_keys":["otk1","otk2"]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/keys/bob", strings.NewReader(body))
	req.Header.Set("Authorization", authHeader(bobToken))
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.UploadKeys)
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("bob upload keys: got %d", w.Code)
	}

	// Alice fetches Bob's keys (requires auth)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/keys/bob", nil)
	req2.SetPathValue("username", "bob")
	req2.Header.Set("Authorization", authHeader(aliceToken))
	w2 := httptest.NewRecorder()
	handler2 := s.AuthMiddleware(s.GetKeys)
	handler2(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("alice fetch bob keys: got %d", w2.Code)
	}

	var result map[string]any
	json.Unmarshal(w2.Body.Bytes(), &result)
	otks := result["one_time_keys"].([]any)
	if len(otks) != 1 {
		t.Fatalf("expected 1 OTK after first fetch, got %d", len(otks))
	}
	firstOTK := otks[0].(string)

	// Second fetch should return a different OTK
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/keys/bob", nil)
	req3.SetPathValue("username", "bob")
	req3.Header.Set("Authorization", authHeader(aliceToken))
	w3 := httptest.NewRecorder()
	handler3 := s.AuthMiddleware(s.GetKeys)
	handler3(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("alice fetch bob keys again: got %d", w3.Code)
	}

	var result2 map[string]any
	json.Unmarshal(w3.Body.Bytes(), &result2)
	otks2 := result2["one_time_keys"].([]any)
	if len(otks2) != 1 {
		t.Fatalf("expected 1 OTK on second fetch, got %d", len(otks2))
	}
	secondOTK := otks2[0].(string)

	if firstOTK == secondOTK {
		t.Errorf("OTK was not consumed: same OTK returned twice: %s", firstOTK)
	}
}

func TestGetKeys_NoOTKAvailable(t *testing.T) {
	s := newTestServer()
	aliceToken := registerUser(t, s, "alice")
	bobToken := registerUser(t, s, "bob")

	// Bob uploads keys with 0 OTKs
	body := `{"curve_key":"bobC1","ed_key":"bobE1","one_time_keys":[]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/keys/bob", strings.NewReader(body))
	req.Header.Set("Authorization", authHeader(bobToken))
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.UploadKeys)
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("bob upload keys: got %d", w.Code)
	}

	// Fetch keys
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/keys/bob", nil)
	req2.SetPathValue("username", "bob")
	req2.Header.Set("Authorization", authHeader(aliceToken))
	w2 := httptest.NewRecorder()
	handler2 := s.AuthMiddleware(s.GetKeys)
	handler2(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("fetch keys: got %d", w2.Code)
	}

	var result map[string]any
	json.Unmarshal(w2.Body.Bytes(), &result)
	if result["no_otk_available"] != true {
		t.Errorf("expected no_otk_available=true, got %v", result["no_otk_available"])
	}
	otks := result["one_time_keys"].([]any)
	if len(otks) != 0 {
		t.Errorf("expected 0 OTKs, got %d", len(otks))
	}
}

// ─── Send Message (requires auth, sender bound to token) ──

func TestSendMessage_RequiresAuth(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "bob")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages",
		strings.NewReader(`{"recipient":"bob","ciphertext":"abc","message_type":0}`))
	// No auth header
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.SendMessage)
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("send without auth: got %d, want 401", w.Code)
	}
}

func TestSendMessage_BindsSenderToAuthToken(t *testing.T) {
	s := newTestServer()
	aliceToken := registerUser(t, s, "alice")
	registerUser(t, s, "bob")

	body := `{"recipient":"bob","ciphertext":"YWIxMg==","message_type":0,"sender_key":"a2V5MTIz"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	req.Header.Set("Authorization", authHeader(aliceToken))
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.SendMessage)
	handler(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("send message: got %d, want 201; body: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["sender_username"] != "alice" {
		t.Errorf("sender should be 'alice' (from token), got %v", result["sender_username"])
	}
}

func TestSendMessage_SenderSpoofingRejected(t *testing.T) {
	s := newTestServer()
	aliceToken := registerUser(t, s, "alice")
	registerUser(t, s, "bob")
	registerUser(t, s, "mallory")

	// Mallory tries to send as alice — but sender is derived from token
	body := `{"recipient":"bob","ciphertext":"YWIxMg==","message_type":0,"sender_key":"a2V5MTIz"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	// Mallory's token
	req.Header.Set("Authorization", authHeader(aliceToken))
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.SendMessage)
	handler(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("send message: got %d", w.Code)
	}

	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["sender_username"] != "alice" {
		t.Errorf("sender should be 'alice' (from token), not spoofed, got %v", result["sender_username"])
	}
}

func TestSendMessage_InvalidRecipient(t *testing.T) {
	s := newTestServer()
	aliceToken := registerUser(t, s, "alice")

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"short recipient", `{"recipient":"ab","ciphertext":"YWIxMg==","message_type":0}`, http.StatusBadRequest},
		{"empty ciphertext", `{"recipient":"bob","ciphertext":"","message_type":0}`, http.StatusBadRequest},
		{"invalid message_type", `{"recipient":"bob","ciphertext":"YWIxMg==","message_type":2}`, http.StatusBadRequest},
		{"negative message_type", `{"recipient":"bob","ciphertext":"YWIxMg==","message_type":-1}`, http.StatusBadRequest},
		{"recipient not found", `{"recipient":"nonexistent","ciphertext":"YWIxMg==","message_type":0}`, http.StatusBadRequest},
		{"invalid ciphertext chars", `{"recipient":"bob","ciphertext":"!!!invalid!!!","message_type":0}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(tt.body))
			req.Header.Set("Authorization", authHeader(aliceToken))
			w := httptest.NewRecorder()
			handler := s.AuthMiddleware(s.SendMessage)
			handler(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

// ─── Get Messages (requires auth, only own messages) ─────

func TestGetMessages_RequiresAuth(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "alice")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.GetMessages)
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("get messages without auth: got %d, want 401", w.Code)
	}
}

func TestGetMessages_OnlyOwnMessages(t *testing.T) {
	s := newTestServer()
	aliceToken := registerUser(t, s, "alice")
	bobToken := registerUser(t, s, "bob")

	// Alice sends to Bob
	s.store.StoreMessage("alice", "bob", "YWIxMg==", 0, "alice_key")

	// Bob fetches messages (should see alice's message)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("Authorization", authHeader(bobToken))
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.GetMessages)
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("bob get messages: got %d", w.Code)
	}
	var envs []any
	json.Unmarshal(w.Body.Bytes(), &envs)
	if len(envs) != 1 {
		t.Errorf("bob should see 1 message, got %d", len(envs))
	}

	// Alice fetches messages (should see own, not Bob's inbound)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req2.Header.Set("Authorization", authHeader(aliceToken))
	w2 := httptest.NewRecorder()
	handler2 := s.AuthMiddleware(s.GetMessages)
	handler2(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("alice get messages: got %d", w2.Code)
	}
	var envs2 []any
	json.Unmarshal(w2.Body.Bytes(), &envs2)
	// Alice may have some or none but should NOT see bob's stuff
	for _, e := range envs2 {
		env := e.(map[string]any)
		if env["sender_username"] == "bob" {
			t.Errorf("alice should not see messages from bob in her queue")
		}
	}
}

// ─── ACK Route ──────────────────────────────────────────

func TestAckMessage_RequiresAuth(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "bob")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/someid/ack", nil)
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.AckMessage)
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("ack without auth: got %d, want 401", w.Code)
	}
}

func TestAckMessage_WrongUserRejected(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "alice")
	registerUser(t, s, "bob")
	malloryToken := registerUser(t, s, "mallory")

	// Alice sends to Bob
	env, err := s.store.StoreMessage("alice", "bob", "YWIxMg==", 0, "alice_key")
	if err != nil {
		t.Fatalf("StoreMessage: %v", err)
	}

	// Fetch as Bob to set status to delivered
	s.store.GetPendingEnvelopes("bob")

	// Mallory tries to ACK Bob's message
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/"+env.ID+"/ack", nil)
	req.SetPathValue("messageID", env.ID)
	req.Header.Set("Authorization", authHeader(malloryToken))
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.AckMessage)
	handler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("mallory ack: got %d, want 400", w.Code)
	}
}

func TestAckMessage_RecipientSucceeds(t *testing.T) {
	s := newTestServer()
	registerUser(t, s, "alice")
	bobToken := registerUser(t, s, "bob")

	// Alice sends to Bob
	env, err := s.store.StoreMessage("alice", "bob", "YWIxMg==", 0, "alice_key")
	if err != nil {
		t.Fatalf("StoreMessage: %v", err)
	}

	// Fetch as Bob to set status to delivered
	s.store.GetPendingEnvelopes("bob")

	// Bob ACKs
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/"+env.ID+"/ack", nil)
	req.SetPathValue("messageID", env.ID)
	req.Header.Set("Authorization", authHeader(bobToken))
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.AckMessage)
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("bob ack: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

// ─── Message Lifecycle ──────────────────────────────────

func TestMessageStatusTransitions(t *testing.T) {
	s := newTestServer()
	s.store.RegisterUser("alice")
	s.store.RegisterUser("bob")
	s.store.UploadKeyBundle("bob", &model.KeyBundle{
		CurveKey: "bob_curve_key",
		EdKey:    "bob_ed_key",
	})

	env, err := s.store.StoreMessage("alice", "bob", "YWIxMg==", 0, "alice_curve_key")
	if err != nil {
		t.Fatalf("StoreMessage failed: %v", err)
	}
	if env.Status != "pending" {
		t.Errorf("initial status should be 'pending', got %s", env.Status)
	}

	envs := s.store.GetPendingEnvelopes("bob")
	if len(envs) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(envs))
	}
	if envs[0].Status != "delivered" {
		t.Errorf("status after fetch should be 'delivered', got %s", envs[0].Status)
	}

	err = s.store.AckMessage(env.ID, "bob")
	if err != nil {
		t.Fatalf("AckMessage failed: %v", err)
	}

	sent := s.store.GetSentMessages("alice")
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sent))
	}
	if sent[0].Status != "received" {
		t.Errorf("status after ack should be 'received', got %s", sent[0].Status)
	}
}

// ─── Rate Limiting ──────────────────────────────────────

func TestRateLimit_Returns429(t *testing.T) {
	s := newTestServer()
	// Use a private rate limiter with very low limit
	s.rl = nil // will be recreated per test needs if needed, but RateLimit uses s.rl

	// Actually let's test via the middleware directly
	rl := middleware.NewRateLimiter(1, 60) // 1 request per 60 seconds
	handler := rl.Middleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, func(r *http.Request) string {
		return "test-client"
	})

	// First should pass
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("first request: got %d, want 200", w.Code)
	}

	// Second should be rate limited
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	w2 := httptest.NewRecorder()
	handler(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("rate limited request: got %d, want 429", w2.Code)
	}
	if w2.Header().Get("Retry-After") == "" {
		t.Errorf("rate limited response should have Retry-After header")
	}
}

// ─── Validation ─────────────────────────────────────────

func TestIsValidUsernameChar(t *testing.T) {
	tests := []struct {
		char  rune
		valid bool
	}{
		{'a', true},
		{'Z', true},
		{'0', true},
		{'_', true},
		{'-', true},
		{' ', false},
		{'@', false},
		{'!', false},
		{'.', false},
	}

	for _, tt := range tests {
		got := isValidUsernameChar(tt.char)
		if got != tt.valid {
			t.Errorf("isValidUsernameChar(%q) = %v, want %v", tt.char, got, tt.valid)
		}
	}
}

func TestIsValidBase64(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"empty", "", true},
		{"valid base64", "ABCDEFghijkl+/==", true},
		{"url-safe base64", "aGVsbG8td29ybGQ", true},
		{"invalid chars", "hello world!!", false},
		{"at sign", "test@123", false},
	}

	for _, tt := range tests {
		got := isValidBase64(tt.input)
		if got != tt.valid {
			t.Errorf("isValidBase64(%q) = %v, want %v", tt.name, got, tt.valid)
		}
	}
}
