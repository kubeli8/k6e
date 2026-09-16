package runtime

import (
	"context"
	"testing"
)

func TestDockerRuntimeLifecycle(t *testing.T) {
	ctx := context.Background()
	rt, err := NewDockerRuntime()
	if err != nil {
		t.Fatalf("Failed to create Docker runtime: %v", err)
	}

	id, err := rt.Create(ctx, ContainerSpec{
		Name:    "kubelite-test",
		Image:   "alpine",
		Command: []string{"sleep", "60"},
	})
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	t.Cleanup(func() {
		_ = rt.Stop(ctx, id)
		_ = rt.Remove(ctx, id)
	})

	if err := rt.Start(ctx, id); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	info, err := rt.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	if !info.Running {
		t.Fatalf("Container is not running after start")
	}

	if err := rt.Stop(ctx, id); err != nil {
		t.Fatalf("Failed to stop container: %v", err)
	}

	info, err = rt.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Failed to inspect container after stop: %v", err)
	}
	if info.Running {
		t.Fatalf("Container is still running after stop")
	}

	if err := rt.Remove(ctx, id); err != nil {
		t.Fatalf("Failed to remove container: %v", err)
	}
}
