package scheduler

import (
	"context"
	"errors"
	"testing"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/testutil"
)

func TestSimpleScheduler_SelectsReadyNode(t *testing.T) {
	scheduler := &SimpleScheduler{}
	workload := testutil.Workload("test-workload", "default")

	node1 := testutil.Node("test-node-1", "127.0.0.1:8080", model.NodeStatusNotReady)
	node2 := testutil.Node("test-node-2", "127.0.0.1:8081", model.NodeStatusReady)
	node3 := testutil.Node("test-node-3", "127.0.0.1:8082", model.NodeStatusReady)

	nodes := []model.Node{node1, node2, node3}

	selectedNode, err := scheduler.Schedule(context.Background(), workload, nodes)
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	if selectedNode.ID != node2.ID {
		t.Errorf("Schedule() got = %v, want = %v", selectedNode.ID, node2.ID)
	}
}

func TestSimpleScheduler_NoReadyNodes(t *testing.T) {
	scheduler := &SimpleScheduler{}
	workload := testutil.Workload("test-workload", "default")
	node1 := testutil.Node("test-node-1", "127.0.0.1:8080", model.NodeStatusNotReady)
	node2 := testutil.Node("test-node-2", "127.0.0.1:8081", model.NodeStatusNotReady)
	node3 := testutil.Node("test-node-3", "127.0.0.1:8082", model.NodeStatusNotReady)
	nodes := []model.Node{node1, node2, node3}

	_, err := scheduler.Schedule(context.Background(), workload, nodes)
	if err == nil {
		t.Errorf("Schedule() error = nil, want = error")
	}

	if !errors.Is(err, ErrNoReadyNodes) {
		t.Errorf("Schedule() error = %v, want = %v", err, ErrNoReadyNodes)
	}
}

func TestSimpleScheduler_ReturnsFirstReadyNode(t *testing.T) {
	scheduler := &SimpleScheduler{}
	workload := testutil.Workload("test-workload", "default")
	node1 := testutil.Node("test-node-1", "127.0.0.1:8080", model.NodeStatusReady)
	node2 := testutil.Node("test-node-2", "127.0.0.1:8081", model.NodeStatusReady)
	node3 := testutil.Node("test-node-3", "127.0.0.1:8082", model.NodeStatusReady)
	nodes := []model.Node{node1, node2, node3}

	selectedNode, err := scheduler.Schedule(context.Background(), workload, nodes)
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	if selectedNode.ID != node1.ID {
		t.Errorf("Schedule() got = %v, want = %v", selectedNode.ID, node1.ID)
	}
}
