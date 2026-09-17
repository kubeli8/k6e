package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/pyd-07/k6e/internal/api"
	"github.com/pyd-07/k6e/internal/store"
)

func main() {
	dbPath := flag.String(
		"db",
		"k6e.db",
		"path to SQLite database",
	)
	flag.Parse()

	store, err := store.NewSQLiteStore(*dbPath)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	server := api.NewServer(store, store)

	log.Println("kubeli8 control plane listening on :8080")
	log.Printf("using database: %s", *dbPath)

	if err := http.ListenAndServe(":8080", server.Handler()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
