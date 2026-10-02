package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeli8/k6e/internal/model"
)

func TestMemoryNodeStoreRegistrationInitializesHeartbeat(t *testing.T) {
	nodes := NewMemoryNodeStore()
	before := time.Now()
	if err := nodes.RegisterNode(context.Background(), model.Node{ID: "node", Address: "localhost", Status: model.NodeStatusReady}); err != nil {
		t.Fatal(err)
	}
	node, err := nodes.GetNode(context.Background(), "node")
	if err != nil {
		t.Fatal(err)
	}
	if node.LastHeartbeat.Before(before) {
		t.Fatalf("heartbeat was not initialized: %v", node.LastHeartbeat)
	}
}

func TestSQLiteNodeStoreRegistrationInitializesHeartbeat(t *testing.T) {
	nodes, err := NewSQLiteStore(filepath.Join(t.TempDir(), "nodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer nodes.Close()
	before := time.Now()
	if err := nodes.RegisterNode(context.Background(), model.Node{ID: "node", Address: "localhost", Status: model.NodeStatusReady}); err != nil {
		t.Fatal(err)
	}
	node, err := nodes.GetNode(context.Background(), "node")
	if err != nil {
		t.Fatal(err)
	}
	if node.LastHeartbeat.Before(before) {
		t.Fatalf("heartbeat was not initialized: %v", node.LastHeartbeat)
	}
}
