package store

import (
	"context"
	"testing"

	"github.com/pyd-07/k6e/internal/model"
)

func testNode(id, address string, status model.NodeStatus) model.Node {
	return model.Node{
		ID:      id,
		Address: address,
		Status:  status,
	}
}

func TestNodeMemoryStoreRegister(t *testing.T) {
	store := NewMemoryNodeStore()
	node := testNode("node1", "addr:node1", model.NodeStatusReady)
	err := store.RegisterNode(context.Background(), node)
	if err != nil {
		t.Fatalf("register, got %v", err)
	}
}

func TestNodeMemoryStoreGetNotFound(t *testing.T) {
	store := NewMemoryNodeStore()
	_, err := store.GetNode(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestNodeMemoryStoreRegisterUpdatesExistingNode(t *testing.T) {
	store := NewMemoryNodeStore()
	node := testNode("node1", "addr:node1", model.NodeStatusReady)
	err := store.RegisterNode(context.Background(), node)
	if err != nil {
		t.Fatalf("register, got %v", err)
	}

	// Update the node
	node.Address = "addr:node1-updated"
	node.Status = model.NodeStatusNotReady
	err = store.RegisterNode(context.Background(), node)
	if err != nil {
		t.Fatalf("register, got %v", err)
	}

	retrievedNode, err := store.GetNode(context.Background(), "node1")
	if err != nil {
		t.Fatalf("get, got %v", err)
	}

	if retrievedNode.Address != "addr:node1-updated" {
		t.Errorf("expected updated address 'addr:node1-updated', got '%s'", retrievedNode.Address)
	}
	if retrievedNode.Status != model.NodeStatusNotReady {
		t.Errorf("expected updated status 'NotReady', got '%s'", retrievedNode.Status)
	}
}

func TestNodeMemoryStoreGet(t *testing.T) {
	store := NewMemoryNodeStore()
	node := testNode("node1", "addr:node1", model.NodeStatusReady)
	err := store.RegisterNode(context.Background(), node)
	if err != nil {
		t.Fatalf("register, got %v", err)
	}

	retrievedNode, err := store.GetNode(context.Background(), "node1")
	if err != nil {
		t.Fatalf("get, got %v", err)
	}

	if retrievedNode.ID != "node1" {
		t.Errorf("expected node ID 'node1', got '%s'", retrievedNode.ID)
	}
}

func TestNodeMemoryStoreList(t *testing.T) {
	store := NewMemoryNodeStore()
	nodes := []model.Node{
		testNode("node1", "addr:node1", model.NodeStatusReady),
		testNode("node2", "addr:node2", model.NodeStatusNotReady),
	}
	for _, node := range nodes {
		err := store.RegisterNode(context.Background(), node)
		if err != nil {
			t.Fatalf("register, got %v", err)
		}
	}
	retrievedNodes, err := store.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("list, got %v", err)
	}
	if len(retrievedNodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(retrievedNodes))
	}
}

func TestNodeMemoryStoreRemove(t *testing.T) {
	store := NewMemoryNodeStore()
	node := testNode("node1", "addr:node1", model.NodeStatusReady)
	err := store.RegisterNode(context.Background(), node)
	if err != nil {
		t.Fatalf("register, got %v", err)
	}
	err = store.RemoveNode(context.Background(), "node1")
	if err != nil {
		t.Fatalf("remove, got %v", err)
	}
}

func TestNodeMemoryStoreRemoveNotFound(t *testing.T) {
	store := NewMemoryNodeStore()
	err := store.RemoveNode(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
