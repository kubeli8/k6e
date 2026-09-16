package main

import (
	"net/http"
	"log"

	"github.com/pyd-07/k6e/internal/api"
	"github.com/pyd-07/k6e/internal/store"
)

func main() {
	workloadStore := store.NewMemoryStore()
	server := api.NewServer(workloadStore)

	log.Println("kubeli8 control plane listening on :8080")

	if err := http.ListenAndServe(":8080", server.Handler()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}