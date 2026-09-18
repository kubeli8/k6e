package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/pyd-07/k6e/internal/api"
	"github.com/pyd-07/k6e/internal/controller"
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
	livenessChecker := controller.NewLivenessChecker(store, 60)

	ctx := context.Background()

	go func() {
		if err := livenessChecker.Start(ctx, 30); err != nil {
			log.Fatalf("Liveness checker failed: %v", err)
		}
	}()

	log.Println("kubeli8 control plane listening on :8080")
	log.Printf("using database: %s", *dbPath)

	if err := http.ListenAndServe(":8080", server.Handler()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
