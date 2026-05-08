package main

import (
	"log"
	"net/http"
	"os"

	"github.com/yup/server/internal/handler"
	"github.com/yup/server/internal/service"
)

func main() {
	store := service.NewStore()
	h := handler.New(store)

	mux := http.NewServeMux()

	// Public route with rate limiting
	mux.HandleFunc("POST /api/v1/users", h.RateLimit(h.RegisterUser))

	// Authenticated routes with rate limiting
	mux.HandleFunc("PUT /api/v1/keys/{username}", h.RateLimitAuth(h.AuthMiddleware(h.UploadKeys)))
	mux.HandleFunc("GET /api/v1/keys/{username}", h.RateLimitAuth(h.AuthMiddleware(h.GetKeys)))
	mux.HandleFunc("POST /api/v1/messages", h.RateLimitAuth(h.AuthMiddleware(h.SendMessage)))
	mux.HandleFunc("GET /api/v1/messages", h.RateLimitAuth(h.AuthMiddleware(h.GetMessages)))
	mux.HandleFunc("POST /api/v1/messages/{messageID}/ack", h.AuthMiddleware(h.AckMessage))
	mux.HandleFunc("GET /api/v1/messages/sent", h.AuthMiddleware(h.GetSentMessages))

	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("YUP server starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
