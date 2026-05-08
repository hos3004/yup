package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yup/server/internal/model"
	"github.com/yup/server/internal/service"
)

func newTestServer() *Server {
	return New(service.NewStore())
}

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
		{"turkish chars", `{"username":"kullanıcı"}`, http.StatusBadRequest},
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

	body := `{"username":"alice"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	s.RegisterUser(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first registration should succeed, got %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	s.RegisterUser(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Errorf("duplicate should return conflict, got %d", w2.Code)
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

func TestSendMessage_Validation(t *testing.T) {
	s := newTestServer()

	// Register a recipient first
	s.store.RegisterUser("bob")

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid message", `{"sender":"alice","recipient":"bob","ciphertext":"abc123","message_type":0,"sender_key":"key123"}`, http.StatusCreated},
		{"missing sender", `{"recipient":"bob","ciphertext":"abc","message_type":0,"sender_key":"key"}`, http.StatusBadRequest},
		{"empty ciphertext", `{"sender":"alice","recipient":"bob","ciphertext":"","message_type":0,"sender_key":"key"}`, http.StatusBadRequest},
		{"invalid message_type", `{"sender":"alice","recipient":"bob","ciphertext":"abc","message_type":2,"sender_key":"key"}`, http.StatusBadRequest},
		{"negative message_type", `{"sender":"alice","recipient":"bob","ciphertext":"abc","message_type":-1,"sender_key":"key"}`, http.StatusBadRequest},
		{"short sender", `{"sender":"ab","recipient":"bob","ciphertext":"abc","message_type":0,"sender_key":"key"}`, http.StatusBadRequest},
		{"short recipient", `{"sender":"alice","recipient":"bo","ciphertext":"abc","message_type":0,"sender_key":"key"}`, http.StatusBadRequest},
		{"recipient not found", `{"sender":"alice","recipient":"nonexistent","ciphertext":"abc","message_type":0,"sender_key":"key"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.SendMessage(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestMessageStatusTransitions(t *testing.T) {
	s := newTestServer()

	// Setup: register and create keys
	s.store.RegisterUser("alice")
	s.store.RegisterUser("bob")
	s.store.UploadKeyBundle("bob", &model.KeyBundle{
		CurveKey: "bob_curve_key",
		EdKey:    "bob_ed_key",
	})

	// Send a message
	env, err := s.store.StoreMessage("alice", "bob", "ciphertext_123", 0, "alice_curve_key")
	if err != nil {
		t.Fatalf("StoreMessage failed: %v", err)
	}
	if env.Status != "pending" {
		t.Errorf("initial status should be 'pending', got %s", env.Status)
	}

	// Fetch (mark delivered)
	envs := s.store.GetPendingEnvelopes("bob")
	if len(envs) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(envs))
	}
	if envs[0].Status != "delivered" {
		t.Errorf("status after fetch should be 'delivered', got %s", envs[0].Status)
	}

	// Ack (mark received)
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

func TestAuthMiddleware(t *testing.T) {
	s := newTestServer()
	s.store.RegisterUser("alice")

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"no auth header", "", http.StatusUnauthorized},
		{"invalid token", "Bearer invalidtoken", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := s.AuthMiddleware(func(w http.ResponseWriter, r *http.Request, username string) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodPut, "/api/v1/keys/alice", nil)
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
		{'\u0131', false}, // Turkish dotless i
	}

	for _, tt := range tests {
		got := isValidUsernameChar(tt.char)
		if got != tt.valid {
			t.Errorf("isValidUsernameChar(%q) = %v, want %v", tt.char, got, tt.valid)
		}
	}
}

func TestGetUserStripsAuthToken(t *testing.T) {
	s := newTestServer()
	s.store.RegisterUser("alice")

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
