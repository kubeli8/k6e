// Command control-plane runs the k6e control plane: the REST API server,
// backed by a SQLite store, plus two background loops - the workload
// reconciliation controller and the node liveness checker - started as
// goroutines before ListenAndServe.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/kubeli8/k6e/internal/agentclient"
	"github.com/kubeli8/k6e/internal/api"
	"github.com/kubeli8/k6e/internal/controller"
	"github.com/kubeli8/k6e/internal/executor"
	"github.com/kubeli8/k6e/internal/scheduler"
	"github.com/kubeli8/k6e/internal/store"
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

	schedulerService := scheduler.NewService(store, store, store, &scheduler.SimpleScheduler{})
	workerClient := agentclient.New(nil)
	executorService := executor.NewService(store, store, store, workerClient)
	controllerService := controller.NewController(store, store, schedulerService, executorService, controller.NewHTTPRuntimeObserver(store, workerClient), "default", 10*time.Second)

	server := api.NewServer(store, store, store, schedulerService, executorService)
	livenessChecker := controller.NewLivenessChecker(store, 30*time.Second)

	ctx := context.Background()

	go func() {
		if err := livenessChecker.Start(ctx, 10*time.Second); err != nil {
			log.Fatalf("Liveness checker failed: %v", err)
		}
	}()
	go func() {
		if err := controllerService.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("controller stopped: %v", err)
		}
	}()

	log.Println("kubeli8 control plane listening on :8080")
	log.Printf("using database: %s", *dbPath)

	if err := http.ListenAndServe(":8080", server.Handler()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
