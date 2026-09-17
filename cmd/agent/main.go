package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/pyd-07/k6e/internal/agent"
	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

func main() {

	controlPlane := flag.String(
		"control-plane",
		"http://localhost:8080",
		"control plane address",
	)
	nodeId := flag.String(
		"node-id",
		"node-1",
		"node ID",
	)
	address := flag.String(
		"address",
		"localhost:8081",
		"node agent address",
	)
	flag.Parse()

	ctx := context.Background()
	rt, err := runtime.NewDockerRuntime()
	if err != nil {
		log.Fatalf("Failed to create Docker runtime: %v", err)
	}

	registrar := agent.NewHTTPNodeRegistrar(*controlPlane)
	ag := agent.New(rt, registrar)

	err = ag.Register(ctx, model.Node{
		ID:      *nodeId,
		Address: *address,
		Status:  model.NodeStatusReady,
	})
	if err != nil {
		log.Fatalf("Failed to register node: %v", err)
	}

	fmt.Println("Node Registered:", *nodeId)

	agentServer := agent.NewServer(ag)
	log.Printf("Agent API listening on %s\n", *address)

	if err := http.ListenAndServe(*address, agentServer.Handler()); err != nil {
		log.Fatalf("Failed to start agent server: %v", err)
	}
}
