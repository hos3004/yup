package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/yup/server/internal/model"
	"github.com/yup/server/internal/service"
)

type Server struct {
	store *service.Store
}

func New(store *service.Store) *Server {
	return &Server{store: store}
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

func (s *Server) AuthMiddleware(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		username := r.PathValue("username")
		if !s.store.ValidateToken(username, token) {
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
	if len(bundle.CurveKey) > 64 || len(bundle.EdKey) > 64 {
		writeError(w, http.StatusBadRequest, "invalid key format")
		return
	}
	if len(bundle.OneTimeKeys) > 100 {
		writeError(w, http.StatusBadRequest, "too many one-time keys")
		return
	}
	saved, err := s.store.UploadKeyBundle(username, &bundle)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) GetKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	username := r.PathValue("username")
	bundle, ok := s.store.GetKeyBundle(username)
	if !ok {
		writeError(w, http.StatusNotFound, "keys not found for user")
		return
	}
	writeJSON(w, http.StatusOK, bundle)
}

func (s *Server) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<18) // 256KB max message
	var body struct {
		Sender     string `json:"sender"`
		Recipient  string `json:"recipient"`
		Ciphertext string `json:"ciphertext"`
		MsgType    int    `json:"message_type"`
		SenderKey  string `json:"sender_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid message")
		return
	}
	if len(body.Sender) < 3 || len(body.Sender) > 32 ||
		len(body.Recipient) < 3 || len(body.Recipient) > 32 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}
	if len(body.Ciphertext) == 0 || len(body.Ciphertext) > 1<<17 {
		writeError(w, http.StatusBadRequest, "invalid ciphertext")
		return
	}
	if body.MsgType < 0 || body.MsgType > 1 {
		writeError(w, http.StatusBadRequest, "invalid message type")
		return
	}
	env, err := s.store.StoreMessage(body.Sender, body.Recipient, body.Ciphertext, body.MsgType, body.SenderKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, env)
}

func (s *Server) GetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	username := r.PathValue("username")
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
