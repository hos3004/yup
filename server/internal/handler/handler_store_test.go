package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/yup/server/internal/service"
)

func newStoreServer(store service.DataStore) *Server {
	return New(store)
}

func runStoreTests(t *testing.T, store service.DataStore, label string) {
	t.Helper()

	s := newStoreServer(store)
	prefix := label

	t.Run(label+"/RegisterAndGetUser", func(t *testing.T) {
		user := prefix + "alice"
		token := registerUserStore(t, s, user)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user, nil)
		req.SetPathValue("username", user)
		w := httptest.NewRecorder()
		s.GetUser(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var result map[string]any
		json.Unmarshal(w.Body.Bytes(), &result)
		if result["username"] != user {
			t.Errorf("expected username %q, got %v", user, result["username"])
		}
		if _, exists := result["auth_token"]; exists {
			t.Errorf("auth_token should not be returned by GetUser")
		}
		_ = token
	})

	t.Run(label+"/TokenValidation", func(t *testing.T) {
		user := prefix + "tokenuser"
		token := registerUserStore(t, s, user)
		username, ok := store.ValidateToken(token)
		if !ok {
			t.Fatal("expected valid token")
		}
		if username != user {
			t.Errorf("expected username %q, got %q", user, username)
		}
		_, ok = store.ValidateToken("invalidtoken")
		if ok {
			t.Error("expected false for invalid token")
		}
	})

	t.Run(label+"/KeyBundleUploadAndFetch", func(t *testing.T) {
		user := prefix + "alicekey"
		aliceToken := registerUserStore(t, s, user)
		uploadKeysStore(t, s, aliceToken, user)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/keys/"+user, nil)
		req.SetPathValue("username", user)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w := httptest.NewRecorder()
		handler := s.AuthMiddleware(s.GetKeys)
		handler(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("fetch keys: got %d, want 200", w.Code)
		}
		var result map[string]any
		json.Unmarshal(w.Body.Bytes(), &result)
		if result["curve_key"] != "CurveKey"+user+"==" {
			t.Errorf("expected curve_key 'CurveKey%v==', got %v", user, result["curve_key"])
		}
	})

	t.Run(label+"/OTKConsumption", func(t *testing.T) {
		aliceToken := registerUserStore(t, s, prefix+"otkuser")
		bobToken := registerUserStore(t, s, prefix+"otkbob")

		bobBody := `{"curve_key":"BobCK","ed_key":"BobEK","one_time_keys":["OTKA","OTKB"]}`
		bobReq := httptest.NewRequest(http.MethodPut, "/api/v1/keys/"+prefix+"otkbob", strings.NewReader(bobBody))
		bobReq.Header.Set("Authorization", "Bearer "+bobToken)
		bobW := httptest.NewRecorder()
		bobHandler := s.AuthMiddleware(s.UploadKeys)
		bobHandler(bobW, bobReq)
		if bobW.Code != http.StatusOK {
			t.Fatalf("bob upload keys: got %d", bobW.Code)
		}

		// First fetch by alice
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/keys/"+prefix+"otkbob", nil)
		req1.SetPathValue("username", prefix+"otkbob")
		req1.Header.Set("Authorization", "Bearer "+aliceToken)
		w1 := httptest.NewRecorder()
		s.AuthMiddleware(s.GetKeys)(w1, req1)
		if w1.Code != http.StatusOK {
			t.Fatalf("first fetch: got %d", w1.Code)
		}
		var result1 map[string]any
		json.Unmarshal(w1.Body.Bytes(), &result1)
		otks1 := result1["one_time_keys"].([]any)
		if len(otks1) != 1 {
			t.Fatalf("expected 1 OTK on first fetch, got %d", len(otks1))
		}
		firstOTK := otks1[0].(string)

		// Second fetch should return a different OTK
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/keys/"+prefix+"otkbob", nil)
		req2.SetPathValue("username", prefix+"otkbob")
		req2.Header.Set("Authorization", "Bearer "+aliceToken)
		w2 := httptest.NewRecorder()
		s.AuthMiddleware(s.GetKeys)(w2, req2)
		if w2.Code != http.StatusOK {
			t.Fatalf("second fetch: got %d", w2.Code)
		}
		var result2 map[string]any
		json.Unmarshal(w2.Body.Bytes(), &result2)
		otks2 := result2["one_time_keys"].([]any)
		if len(otks2) != 1 {
			t.Fatalf("expected 1 OTK on second fetch, got %d", len(otks2))
		}
		secondOTK := otks2[0].(string)
		if firstOTK == secondOTK {
			t.Errorf("OTK was not consumed: same OTK returned twice: %s", firstOTK)
		}
	})

	t.Run(label+"/MessageLifecycle", func(t *testing.T) {
		aliceToken := registerUserStore(t, s, prefix+"msgalice")
		bobToken := registerUserStore(t, s, prefix+"msgbob")
		uploadKeysStore(t, s, aliceToken, prefix+"msgalice")

		// Alice sends to Bob
		body := `{"recipient":"` + prefix + `msgbob","ciphertext":"YWIxMg==","message_type":0}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w := httptest.NewRecorder()
		handler := s.AuthMiddleware(s.SendMessage)
		handler(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("send: got %d, want 201; body: %s", w.Code, w.Body.String())
		}
		var env map[string]any
		json.Unmarshal(w.Body.Bytes(), &env)
		msgID := env["id"].(string)
		if env["status"] != "pending" {
			t.Errorf("initial status should be 'pending', got %v", env["status"])
		}

		// Bob fetches pending messages
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
		req2.Header.Set("Authorization", "Bearer "+bobToken)
		w2 := httptest.NewRecorder()
		s.AuthMiddleware(s.GetMessages)(w2, req2)
		if w2.Code != http.StatusOK {
			t.Fatalf("fetch: got %d", w2.Code)
		}
		var envs []any
		json.Unmarshal(w2.Body.Bytes(), &envs)
		if len(envs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(envs))
		}

		// Bob ACKs
		req3 := httptest.NewRequest(http.MethodPost, "/api/v1/messages/"+msgID+"/ack", nil)
		req3.SetPathValue("messageID", msgID)
		req3.Header.Set("Authorization", "Bearer "+bobToken)
		w3 := httptest.NewRecorder()
		s.AuthMiddleware(s.AckMessage)(w3, req3)
		if w3.Code != http.StatusOK {
			t.Fatalf("ack: got %d, want 200", w3.Code)
		}

		// Verify sent messages
		req4 := httptest.NewRequest(http.MethodGet, "/api/v1/messages/sent", nil)
		req4.Header.Set("Authorization", "Bearer "+aliceToken)
		w4 := httptest.NewRecorder()
		s.AuthMiddleware(s.GetSentMessages)(w4, req4)
		if w4.Code != http.StatusOK {
			t.Fatalf("sent: got %d", w4.Code)
		}
		var sent []any
		json.Unmarshal(w4.Body.Bytes(), &sent)
		if len(sent) != 1 {
			t.Fatalf("expected 1 sent message, got %d", len(sent))
		}
		sentMap := sent[0].(map[string]any)
		if sentMap["status"] != "received" {
			t.Errorf("sent status should be 'received', got %v", sentMap["status"])
		}
		if sentMap["id"] != msgID {
			t.Errorf("expected message id %s, got %v", msgID, sentMap["id"])
		}
	})

	t.Run(label+"/GetSentMessages", func(t *testing.T) {
		aliceToken := registerUserStore(t, s, prefix+"sentalice")
		registerUserStore(t, s, prefix+"sentbob")
		uploadKeysStore(t, s, aliceToken, prefix+"sentalice")

		// Send 2 messages
		for i := 0; i < 2; i++ {
			body := `{"recipient":"` + prefix + `sentbob","ciphertext":"YWIxMg==","message_type":0}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+aliceToken)
			w := httptest.NewRecorder()
			s.AuthMiddleware(s.SendMessage)(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("send %d: got %d", i, w.Code)
			}
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/sent", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w := httptest.NewRecorder()
		s.AuthMiddleware(s.GetSentMessages)(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("sent: got %d", w.Code)
		}
		var sent []any
		json.Unmarshal(w.Body.Bytes(), &sent)
		if len(sent) != 2 {
			t.Errorf("expected 2 sent messages, got %d", len(sent))
		}
	})
}

func registerUserStore(t *testing.T, s *Server, username string) string {
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

func uploadKeysStore(t *testing.T, s *Server, token, username string) {
	t.Helper()
	curveKey := "CurveKey" + username + "=="
	edKey := "EdKey" + username + "=="
	body := `{"curve_key":"` + curveKey + `","ed_key":"` + edKey + `","one_time_keys":["otk1","otk2"]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/keys/"+username, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler := s.AuthMiddleware(s.UploadKeys)
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload keys for %s: got %d, want 200; body: %s", username, w.Code, w.Body.String())
	}
}

func TestStoreSuite_InMemory(t *testing.T) {
	store := service.NewInMemoryStore()
	runStoreTests(t, store, "InMemory")
}

func TestStoreSuite_Postgres(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("Skipping PostgresStore tests: set DATABASE_URL_TEST")
	}
	store, err := service.NewPostgresStore(dbURL)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	defer store.Close()

	// Clean all tables
	cleanCtx := context.Background()
	store.Pool().Exec(cleanCtx, `DELETE FROM messages`)
	store.Pool().Exec(cleanCtx, `DELETE FROM one_time_keys`)
	store.Pool().Exec(cleanCtx, `DELETE FROM key_bundles`)
	store.Pool().Exec(cleanCtx, `DELETE FROM users`)

	runStoreTests(t, store, "Postgres")
}
