package handler

import (
	"encoding/json"
	"net/http"

	"github.com/yup/server/internal/model"
)

func (s *Server) RegisterDevice(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var req model.DeviceTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Token) == 0 || len(req.Token) > 512 {
		writeError(w, http.StatusBadRequest, "invalid token")
		return
	}
	if req.Platform == "" {
		req.Platform = "android"
	}
	if req.Platform != "android" && req.Platform != "ios" && req.Platform != "web" {
		writeError(w, http.StatusBadRequest, "invalid platform")
		return
	}

	if err := s.store.RegisterDeviceToken(username, req.Token, req.Platform); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}
