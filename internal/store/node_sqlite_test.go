package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pyd-07/k6e/internal/model"
)

func TestSQLiteStore_RegisterNode(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close SQLiteStore: %v", err)
		}
	}()

	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusReady)
	err = store.RegisterNode(context.Background(), node)
	if err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}
}

func TestSQLiteStore_RegisterNodeUpdatesExistingNode(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close SQLiteStore: %v", err)
		}
	}()

	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusNotReady)
	err = store.RegisterNode(ctx, node)
	if err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}
	node.Address = "172.16.0.1:8080"
	node.Status = model.NodeStatusReady
	err = store.RegisterNode(ctx, node)
	if err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}

	retrievedNode, err := store.GetNode(ctx, node.ID)
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}

	if retrievedNode.Address != node.Address || retrievedNode.Status != node.Status {
		t.Errorf("GetNode() = %v, want %v", retrievedNode, node)
	}
}

func TestSQLiteStore_GetNode(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close SQLiteStore: %v", err)
		}
	}()

	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusReady)
	err = store.RegisterNode(ctx, node)
	if err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}

	retrievedNode, err := store.GetNode(ctx, node.ID)
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}

	if retrievedNode.Status != node.Status {
		t.Errorf("GetNode() = %v, want %v", retrievedNode, node)
	}
}

func TestSQLiteStore_GetNodeNotFound(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close SQLiteStore: %v", err)
		}
	}()

	_, err = store.GetNode(context.Background(), "non-existent-node")
	if err == nil {
		t.Errorf("expected error when getting non-existent node, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteStore_ListNodes(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close SQLiteStore: %v", err)
		}
	}()

	node1 := testNode("node-1", "127.0.0.1:8080", model.NodeStatusReady)
	node2 := testNode("node-2", "172.16.0.1:8080", model.NodeStatusNotReady)
	if err := store.RegisterNode(ctx, node1); err != nil {
		t.Fatalf("failed to register node1: %v", err)
	}
	if err := store.RegisterNode(ctx, node2); err != nil {
		t.Fatalf("failed to register node2: %v", err)
	}

	nodes, err := store.ListNodes(ctx)
	if err != nil {
		t.Fatalf("failed to list nodes: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestSQLiteStore_RemoveNode(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close SQLiteStore: %v", err)
		}
	}()

	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusReady)
	if err := store.RegisterNode(ctx, node); err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}
	if err := store.RemoveNode(ctx, node.ID); err != nil {
		t.Fatalf("RemoveNode() error = %v", err)
	}

	_, err = store.GetNode(ctx, node.ID)
	if err == nil {
		t.Errorf("expected error when getting removed node, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteStore_RemoveNodeNotFound(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close SQLiteStore: %v", err)
		}
	}()

	err = store.RemoveNode(context.Background(), "non-existent-node")
	if err == nil {
		t.Errorf("expected error when removing non-existent node, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteStore_UpdateHeartbeat(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close SQLiteStore: %v", err)
		}
	}()

	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusNotReady)
	if err := store.RegisterNode(ctx, node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	err = store.UpdateHeartbeat(ctx, node.ID, time.Now())
	if err != nil {
		t.Fatalf("failed to update heartbeat: %v", err)
	}

	retrievedNode, err := store.GetNode(ctx, node.ID)
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if retrievedNode.Status != model.NodeStatusReady {
		t.Errorf("expected node status to be Ready, got %v", retrievedNode.Status)
	}
}

func TestSQLiteStore_UpdateHeartbeatNotFound(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close SQLiteStore: %v", err)
		}
	}()

	err = store.UpdateHeartbeat(context.Background(), "non-existent-node", time.Now())
	if err == nil {
		t.Errorf("expected error when updating heartbeat for non-existent node, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
