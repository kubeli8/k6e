package main

import (
	"context"
	"flag"
	"fmt"
	"log"

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
		"localhost:8801",
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

	id, err := ag.Run(ctx, runtime.ContainerSpec{
		Name:    "kubelite-test",
		Image:   "alpine",
		Command: []string{"sleep", "60"},
	})
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	fmt.Println("Running Container: ", id)

	info, err := ag.Inspect(ctx, id)
	if err != nil {
		log.Fatalf("Failed to inspect container: %v", err)
	}

	fmt.Printf("Container state: %+v\n", info.State)

	if err := ag.Stop(ctx, id); err != nil {
		log.Fatalf("Failed to stop container: %v", err)
	}

	fmt.Println("Stopped Container: ", id)

	if err := ag.Remove(ctx, id); err != nil {
		log.Fatalf("Failed to remove container: %v", err)
	}

	fmt.Println("Removed Container: ", id)
}
