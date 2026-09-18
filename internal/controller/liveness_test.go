package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/store"
)

func testNode(id, address string, status model.NodeStatus, heartbeat time.Time) model.Node {
	return model.Node{
		ID:            id,
		Address:       address,
		Status:        status,
		LastHeartbeat: heartbeat,
	}
}

func TestLivenessCheckerMarksStaleNodeNotReady(t *testing.T) {
	ctx := context.Background()
	nodeStore := store.NewMemoryNodeStore()

	node := testNode(
		"test-node",
		"127.0.0.1:8801",
		model.NodeStatusReady,
		time.Now().Add(-10*time.Minute),
	)

	if err := nodeStore.RegisterNode(ctx, node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	checker := NewLivenessChecker(nodeStore, 5*time.Minute)

	if err := checker.CheckLiveness(ctx); err != nil {
		t.Fatalf("CheckLiveness() returned unexpected error: %v", err)
	}

	got, err := nodeStore.GetNode(ctx, "test-node")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if got.Status != model.NodeStatusNotReady {
		t.Errorf(
			"expected node status %q, got %q",
			model.NodeStatusNotReady,
			got.Status,
		)
	}
}

func TestLivenessCheckerLeavesFreshNodeAlone(t *testing.T) {
	ctx := context.Background()
	nodeStore := store.NewMemoryNodeStore()

	node := testNode(
		"test-node",
		"127.0.0.1:8801",
		model.NodeStatusReady,
		time.Now(),
	)

	if err := nodeStore.RegisterNode(ctx, node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	checker := NewLivenessChecker(nodeStore, 5*time.Minute)

	if err := checker.CheckLiveness(ctx); err != nil {
		t.Fatalf("CheckLiveness() returned unexpected error: %v", err)
	}

	got, err := nodeStore.GetNode(ctx, "test-node")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if got.Status != model.NodeStatusReady {
		t.Errorf(
			"expected node status %q, got %q",
			model.NodeStatusReady,
			got.Status,
		)
	}
}

func TestLivenessCheckerLeavesAlreadyNotReadyNodeAlone(t *testing.T) {
	ctx := context.Background()
	nodeStore := store.NewMemoryNodeStore()

	node := testNode(
		"test-node",
		"127.0.0.1:8801",
		model.NodeStatusNotReady,
		time.Now().Add(-10*time.Minute),
	)

	if err := nodeStore.RegisterNode(ctx, node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	checker := NewLivenessChecker(nodeStore, 5*time.Minute)

	if err := checker.CheckLiveness(ctx); err != nil {
		t.Fatalf("CheckLiveness() returned unexpected error: %v", err)
	}

	got, err := nodeStore.GetNode(ctx, "test-node")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if got.Status != model.NodeStatusNotReady {
		t.Errorf(
			"expected node status to remain %q, got %q",
			model.NodeStatusNotReady,
			got.Status,
		)
	}
}

func TestLivenessCheckerInvalidTimeout(t *testing.T) {
	ctx := context.Background()
	nodeStore := store.NewMemoryNodeStore()

	checker := NewLivenessChecker(nodeStore, 0)

	err := checker.CheckLiveness(ctx)
	if err == nil {
		t.Fatal("expected error for invalid liveness timeout")
	}
}

func TestLivenessCheckerWithoutStore(t *testing.T) {
	ctx := context.Background()

	checker := NewLivenessChecker(nil, 5*time.Minute)

	err := checker.CheckLiveness(ctx)
	if err == nil {
		t.Fatal("expected error when node store is not configured")
	}
}

type failingNodeStore struct {
	store.NodeStore
}

func (f *failingNodeStore) ListNodes(ctx context.Context) ([]model.Node, error) {
	return nil, errors.New("list nodes failed")
}

func TestLivenessCheckerStoreError(t *testing.T) {
	ctx := context.Background()

	nodeStore := &failingNodeStore{}

	checker := NewLivenessChecker(nodeStore, 5*time.Minute)

	err := checker.CheckLiveness(ctx)
	if err == nil {
		t.Fatal("expected store error")
	}

	if err.Error() != "list nodes failed" {
		t.Errorf("expected 'list nodes failed', got %v", err)
	}
}

func TestLivenessCheckerStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	nodeStore := store.NewMemoryNodeStore()

	node := testNode(
		"test-node",
		"127.0.0.1:8801",
		model.NodeStatusReady,
		time.Now().Add(-10*time.Minute),
	)

	if err := nodeStore.RegisterNode(ctx, node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	checker := NewLivenessChecker(nodeStore, 5*time.Minute)

	done := make(chan error, 1)

	go func() {
		done <- checker.Start(ctx, 10*time.Millisecond)
	}()

	select {
	case <-time.After(100 * time.Millisecond):
	case err := <-done:
		t.Fatalf("Start() stopped unexpectedly: %v", err)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start() returned unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start() did not stop after context cancellation")
	}

	got, err := nodeStore.GetNode(context.Background(), "test-node")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if got.Status != model.NodeStatusNotReady {
		t.Errorf(
			"expected node status %q, got %q",
			model.NodeStatusNotReady,
			got.Status,
		)
	}
}
