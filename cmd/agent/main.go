package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pyd-07/k6e/internal/agent"
	"github.com/pyd-07/k6e/internal/runtime"
)

func main() {
	ctx := context.Background()
	rt, err := runtime.NewDockerRuntime()
	if err != nil {
		log.Fatalf("Failed to create Docker runtime: %v", err)
	}

	ag := agent.New(rt)

	id, err := ag.Run(ctx, runtime.ContainerSpec{
		Name:    "kubelite-test",
		Image:   "alpine",
		Command: []string{"sleep", "60"},
	})
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	fmt.Println("Running: ", id)

	info, err := ag.Inspect(ctx, id)
	if err != nil {
		log.Fatalf("Failed to inspect container: %v", err)
	}

	fmt.Printf("Container state: %+v\n", info.State)

	if err := ag.Stop(ctx, id); err != nil {
		log.Fatalf("Failed to stop container: %v", err)
	}

	fmt.Println("Stopped: ", id)

	if err := ag.Remove(ctx, id); err != nil {
		log.Fatalf("Failed to remove container: %v", err)
	}

	fmt.Println("Removed: ", id)
}
