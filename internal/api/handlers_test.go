package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/store"
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
			Replicas: 3,
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

func testServer() *Server {
	workloadStore := store.NewMemoryStore()
	nodeStore := store.NewMemoryNodeStore()
	return NewServer(workloadStore, nodeStore)
}

func TestCreateWorkload(t *testing.T) {
	ctx := context.Background()
	server := testServer()

	body := `{
		"apiVersion": "k6e.io/v1alpha1",
		"kind": "Workload",
		"metadata": {
			"name": "test-workload",
			"namespace": "default"
		},
		"spec": {
			"replicas": 3,
			"template": {
				"containers": [
					{
						"name": "nginx",
						"image": "nginx:latest"
					}
				]
			}
		}
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/workloads",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	created, err := server.workloadStore.Get(ctx, "default", "test-workload")
	if err != nil {
		t.Fatalf("Failed to get created workload: %v", err)
	}

	if created.Spec.Replicas != 3 {
		t.Fatalf("Expected replicas %d, got %d", 3, created.Spec.Replicas)
	}
}

func TestCreateWorkloadInvalidJSON(t *testing.T) {
	server := testServer()

	body := `{
		"apiVersion": "k6e.io/v1alpha1",
		"kind": "Workload",
		"metadata": {
			"name": "test-workload",
			"namespace": "default"
		},
		"spec": {
			"replicas": 3,
			"template": {
				"containers": [
					{
						"name": "nginx",
						"image": "nginx:latest"
					}
				]
			}
	}` // Missing closing brace

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/workloads",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateWorkloadInvalidData(t *testing.T) {
	server := testServer()

	body := `{
		"apiVersion": "k6e.io/v1alpha1",
		"kind": "Workload",
		"metadata": {
			"name": "",
			"namespace": "default"
		},
		"spec": {
			"replicas": 3,
			"template": {
				"containers": [
					{
						"name": "nginx",
						"image": "nginx:latest"
					}
				]
			}
		}
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/workloads",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateWorkloadAlreadyExists(t *testing.T) {
	server := testServer()

	if err := server.workloadStore.Create(context.Background(), testWorkload("test-workload", "default")); err != nil {
		t.Fatalf("Failed to create initial workload: %v", err)
	}

	body := `{
		"apiVersion": "k6e.io/v1alpha1",
		"kind": "Workload",
		"metadata": {
			"name": "test-workload",
			"namespace": "default"
		},
		"spec": {
			"replicas": 3,
			"template": {
				"containers": [
					{
						"name": "nginx",
						"image": "nginx:latest"
					}
				]
			}
		}
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/workloads",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("Expected status code %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestGetWorkload(t *testing.T) {
	server := testServer()

	workload := testWorkload("test-workload", "default")

	if err := server.workloadStore.Create(context.Background(), workload); err != nil {
		t.Fatalf("Failed to create initial workload: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/workloads/default/test-workload",
		nil,
	)

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	var got model.Workload
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if got.Metadata.Name != workload.Metadata.Name {
		t.Fatalf(
			"Expected workload name %q, got %q",
			workload.Metadata.Name,
			got.Metadata.Name,
		)
	}
}

func TestGetWorkloadNotFound(t *testing.T) {
	server := testServer()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/workloads/default/nonexistent-workload",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestWorkloadList(t *testing.T) {
	server := testServer()

	workload1 := testWorkload("test-workload-1", "default")
	workload2 := testWorkload("test-workload-2", "default")
	workload3 := testWorkload("other-workload", "production")

	if err := server.workloadStore.Create(context.Background(), workload1); err != nil {
		t.Fatalf("Failed to create initial workload: %v", err)
	}
	if err := server.workloadStore.Create(context.Background(), workload2); err != nil {
		t.Fatalf("Failed to create initial workload: %v", err)
	}
	if err := server.workloadStore.Create(context.Background(), workload3); err != nil {
		t.Fatalf("Failed to create initial workload: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/workloads?namespace=default",
		nil,
	)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	var workloads []model.Workload
	if err := json.NewDecoder(rec.Body).Decode(&workloads); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(workloads) != 2 {
		t.Fatalf("Expected 2 workloads, got %d", len(workloads))
	}
}

func TestWorkloadListEmpty(t *testing.T) {
	server := testServer()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/workloads?namespace=empty",
		nil,
	)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	var workloads []model.Workload
	if err := json.NewDecoder(rec.Body).Decode(&workloads); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(workloads) != 0 {
		t.Fatalf("Expected 0 workloads, got %d", len(workloads))
	}
}

func TestWorkloadDelete(t *testing.T) {
	server := testServer()

	workload := testWorkload("test-workload", "default")
	if err := server.workloadStore.Create(context.Background(), workload); err != nil {
		t.Fatalf("Failed to create initial workload: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/workloads/default/test-workload",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("Expected status code %d, got %d", http.StatusNoContent, rec.Code)
	}

	_, err := server.workloadStore.Get(context.Background(), "default", "test-workload")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Expected workload to be deleted, but it still exists")
	}
}

func TestWorkloadDeleteNotFound(t *testing.T) {
	server := testServer()

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/workloads/default/nonexistent-workload",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestNodeRegistration(t *testing.T) {
	server := testServer()

	body := `{
		"id": "test-node",
		"address": "127.0.0.1:8080",
		"status": "Ready"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/nodes",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	node, err := server.nodeStore.GetNode(context.Background(), "test-node")
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	if node.ID != "test-node" {
		t.Fatalf("Expected node ID 'test-node', got '%s'", node.ID)
	}
	if node.Address != "127.0.0.1:8080" {
		t.Fatalf("Expected node address '127.0.0.1:8080', got '%s'", node.Address)
	}
	if node.Status != "Ready" {
		t.Fatalf("Expected node status 'Ready', got '%s'", node.Status)
	}
}

func TestNodeRegistrationWrongData(t *testing.T) {
	server := testServer()

	body := `{
		"id": "op",
		"address": "127.0.0.1:8080",
		"status": "not-valid"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/nodes",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNodeGet(t *testing.T) {
	server := testServer()
	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusReady)
	if err := server.nodeStore.RegisterNode(context.Background(), node); err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/nodes/test-node",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	var got model.Node
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if got.ID != node.ID {
		t.Fatalf("Expected node ID %q, got %q", node.ID, got.ID)
	}
	if got.Address != node.Address {
		t.Fatalf("Expected node address %q, got %q", node.Address, got.Address)
	}
	if got.Status != node.Status {
		t.Fatalf("Expected node status %q, got %q", node.Status, got.Status)
	}
}

func TestNodeGetNotFound(t *testing.T) {
	server := testServer()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/nodes/nonexistent-node",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestNodeList(t *testing.T) {
	server := testServer()
	node1 := testNode("node-1", "127.0.0.1:8080", model.NodeStatusReady)
	node2 := testNode("node-2", "172.16.0.1:8080", model.NodeStatusNotReady)
	if err := server.nodeStore.RegisterNode(context.Background(), node1); err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}
	if err := server.nodeStore.RegisterNode(context.Background(), node2); err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/nodes",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	var nodes []model.Node
	if err := json.NewDecoder(rec.Body).Decode(&nodes); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(nodes))
	}
}

func TestNodeRemove(t *testing.T) {
	server := testServer()
	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusReady)
	if err := server.nodeStore.RegisterNode(context.Background(), node); err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/nodes/test-node",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("Expected status code %d, got %d", http.StatusNoContent, rec.Code)
	}

	_, err := server.nodeStore.GetNode(context.Background(), "test-node")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Expected node to be removed, but it still exists")
	}
}

func TestNodeRemoveNotFound(t *testing.T) {
	server := testServer()

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/nodes/nonexistent-node",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestNodeUpdateHeartbeat(t *testing.T) {
	server := testServer()
	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusNotReady)
	if err := server.nodeStore.RegisterNode(context.Background(), node); err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/nodes/test-node/heartbeat",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("Expected status code %d, got %d", http.StatusNoContent, rec.Code)
	}

	retrievedNode, err := server.nodeStore.GetNode(context.Background(), "test-node")
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	if retrievedNode.Status != model.NodeStatusReady {
		t.Errorf("expected updated status 'Ready', got '%s'", retrievedNode.Status)
	}
}

func TestNodeUpdateHeartbeatNotFound(t *testing.T) {
	server := testServer()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/nodes/nonexistent-node/heartbeat",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}
