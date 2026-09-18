package scheduler

import (
	"context"
	"errors"
	"testing"

	"github.com/pyd-07/k6e/internal/model"
)

func testWorkload(name, namespace string) model.Workload {
	return model.Workload{
		APIVersion: "k6e.io/v1alpha1",
		Kind:       "Workload",
		Metadata: model.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: model.WorkloadSpec{
			Replicas: 2,
			Template: model.PodTemplateSpec{
				Containers: []model.ContainerSpec{
					{
						Name:  "nginx",
						Image: "nginx:latest",
					},
				},
			},
		},
	}
}

func testNode(id, address string, status model.NodeStatus) model.Node {
	return model.Node{
		ID:      id,
		Address: address,
		Status:  status,
	}
}

func TestSimpleScheduler_SelectsReadyNode(t *testing.T) {
	scheduler := &SimpleScheduler{}
	workload := testWorkload("test-workload", "default")

	node1 := testNode("test-node-1", "127.0.0.1:8080", model.NodeStatusNotReady)
	node2 := testNode("test-node-2", "127.0.0.1:8081", model.NodeStatusReady)
	node3 := testNode("test-node-3", "127.0.0.1:8082", model.NodeStatusReady)

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
	workload := testWorkload("test-workload", "default")
	node1 := testNode("test-node-1", "127.0.0.1:8080", model.NodeStatusNotReady)
	node2 := testNode("test-node-2", "127.0.0.1:8081", model.NodeStatusNotReady)
	node3 := testNode("test-node-3", "127.0.0.1:8082", model.NodeStatusNotReady)
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
	workload := testWorkload("test-workload", "default")
	node1 := testNode("test-node-1", "127.0.0.1:8080", model.NodeStatusReady)
	node2 := testNode("test-node-2", "127.0.0.1:8081", model.NodeStatusReady)
	node3 := testNode("test-node-3", "127.0.0.1:8082", model.NodeStatusReady)
	nodes := []model.Node{node1, node2, node3}

	selectedNode, err := scheduler.Schedule(context.Background(), workload, nodes)
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	if selectedNode.ID != node1.ID {
		t.Errorf("Schedule() got = %v, want = %v", selectedNode.ID, node1.ID)
	}
}
