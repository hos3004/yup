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

	mux.HandleFunc("POST /api/v1/users", h.RegisterUser)
	mux.HandleFunc("GET /api/v1/users/{username}", h.GetUser)
	mux.HandleFunc("PUT /api/v1/keys/{username}", h.AuthMiddleware(h.UploadKeys))
	mux.HandleFunc("GET /api/v1/keys/{username}", h.GetKeys)
	mux.HandleFunc("POST /api/v1/messages", h.SendMessage)
	mux.HandleFunc("GET /api/v1/messages/{username}", h.GetMessages)
	mux.HandleFunc("POST /api/v1/messages/{messageID}/ack", h.AuthMiddleware(h.AckMessage))
	mux.HandleFunc("GET /api/v1/messages/{username}/sent", h.AuthMiddleware(h.GetSentMessages))

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
