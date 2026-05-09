package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yup/server/internal/middleware"
	"github.com/yup/server/internal/model"
	"github.com/yup/server/internal/notifier"
	"github.com/yup/server/internal/service"
)

type Server struct {
	store    service.DataStore
	rl       *middleware.RateLimiter
	notifier notifier.Notifier
}

func New(store service.DataStore) *Server {
	return &Server{
		store:    store,
		rl:       middleware.NewRateLimiter(30, 60*time.Second),
		notifier: notifier.New(),
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

// AuthMiddleware validates Bearer token and passes username to next handler.
// Derives username from token, NOT from path/body.
func (s *Server) AuthMiddleware(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		username, ok := s.store.ValidateToken(token)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		next(w, r, username)
	}
}

func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256)
	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if len(body.Username) < 3 || len(body.Username) > 32 {
		writeError(w, http.StatusBadRequest, "username must be 3-32 characters")
		return
	}
	for _, c := range body.Username {
		if !isValidUsernameChar(c) {
			writeError(w, http.StatusBadRequest, "username contains invalid characters")
			return
		}
	}
	user, err := s.store.RegisterUser(body.Username)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func isValidUsernameChar(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
}

func (s *Server) GetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	username := r.PathValue("username")
	user, ok := s.store.GetUser(username)
	if !ok {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	user.AuthToken = ""
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) UploadKeys(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max key bundle
	var bundle model.KeyBundle
	if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
		writeError(w, http.StatusBadRequest, "invalid key bundle")
		return
	}
	if !isValidBase64(bundle.CurveKey) || !isValidBase64(bundle.EdKey) {
		writeError(w, http.StatusBadRequest, "invalid key format (must be base64)")
		return
	}
	if len(bundle.CurveKey) > 64 || len(bundle.EdKey) > 64 {
		writeError(w, http.StatusBadRequest, "invalid key length")
		return
	}
	if len(bundle.OneTimeKeys) > 100 {
		writeError(w, http.StatusBadRequest, "too many one-time keys")
		return
	}
	for _, otk := range bundle.OneTimeKeys {
		if !isValidBase64(otk) || len(otk) > 64 {
			writeError(w, http.StatusBadRequest, "invalid one-time key format")
			return
		}
	}
	saved, err := s.store.UploadKeyBundle(username, &bundle)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) GetKeys(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	targetUser := r.PathValue("username")
	bundle, ok, remaining := s.store.GetKeyBundle(targetUser)
	if !ok {
		writeError(w, http.StatusNotFound, "keys not found for user")
		return
	}
	resp := map[string]any{
		"device_id":          bundle.DeviceID,
		"curve_key":          bundle.CurveKey,
		"ed_key":             bundle.EdKey,
		"one_time_keys":      bundle.OneTimeKeys,
	}
	if remaining != "" {
		resp["no_otk_available"] = true
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) SendMessage(w http.ResponseWriter, r *http.Request, sender string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<18) // 256KB max
	var body struct {
		Recipient  string `json:"recipient"`
		Ciphertext string `json:"ciphertext"`
		MsgType    int    `json:"message_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid message")
		return
	}
	if len(body.Recipient) < 3 || len(body.Recipient) > 32 {
		writeError(w, http.StatusBadRequest, "invalid recipient username")
		return
	}
	if len(body.Ciphertext) == 0 || len(body.Ciphertext) > 1<<17 {
		writeError(w, http.StatusBadRequest, "invalid ciphertext")
		return
	}
	if !isValidBase64(body.Ciphertext) {
		writeError(w, http.StatusBadRequest, "invalid ciphertext encoding")
		return
	}
	if body.MsgType < 0 || body.MsgType > 1 {
		writeError(w, http.StatusBadRequest, "invalid message type")
		return
	}
	// Derive sender_key from authenticated sender's registered curve key
	senderKey, ok := s.store.GetCurveKey(sender)
	if !ok {
		writeError(w, http.StatusBadRequest, "sender has not uploaded keys")
		return
	}
	env, err := s.store.StoreMessage(sender, body.Recipient, body.Ciphertext, body.MsgType, senderKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Send push notification to recipient asynchronously
	go func() {
		tokens, err := s.store.GetDeviceTokens(body.Recipient)
		if err != nil {
			log.Printf("push: get tokens for %s: %v", body.Recipient, err)
			return
		}
		if len(tokens) > 0 {
			data := map[string]string{
				"type":    "new_message",
				"sender":  sender,
			}
			if _, err := s.notifier.SendPush(context.Background(), tokens, data); err != nil {
				log.Printf("push: send to %s: %v", body.Recipient, err)
			}
		}
	}()

	writeJSON(w, http.StatusCreated, env)
}

func (s *Server) GetMessages(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	envs := s.store.GetPendingEnvelopes(username)
	writeJSON(w, http.StatusOK, envs)
}

func (s *Server) AckMessage(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	messageID := r.PathValue("messageID")
	if err := s.store.AckMessage(messageID, username); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

func (s *Server) GetSentMessages(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	envs := s.store.GetSentMessages(username)
	writeJSON(w, http.StatusOK, envs)
}

// Simple base64 validation — checks the string only contains base64 characters.
func isValidBase64(s string) bool {
	if s == "" {
		return true
	}
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' || c == '-') {
			return false
		}
	}
	return true
}

// RateLimit wraps a handler with IP-based rate limiting.
func (s *Server) RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return s.rl.Middleware(next, func(r *http.Request) string {
		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx >= 0 {
			ip = ip[:idx]
		}
		return ip
	})
}

// RateLimitAuth wraps an authenticated handler with user-based rate limiting.
func (s *Server) RateLimitAuth(next http.HandlerFunc) http.HandlerFunc {
	return s.rl.Middleware(next, func(r *http.Request) string {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx >= 0 {
			ip = ip[:idx]
		}
		return ip
	})
}
