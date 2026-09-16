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

func TestCreateWorkload(t *testing.T) {
	ctx := context.Background()
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

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

	created, err := workloadStore.Get(ctx, "default", "test-workload")
	if err != nil {
		t.Fatalf("Failed to get created workload: %v", err)
	}

	if created.Spec.Replicas != 3 {
		t.Fatalf("Expected replicas %d, got %d", 3, created.Spec.Replicas)
	}
}

func TestCreateWorkloadInvalidJSON(t *testing.T) {
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

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
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

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
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

	workload := testWorkload("test-workload", "default")

	if err := workloadStore.Create(context.Background(), workload); err != nil {
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
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

	workload := testWorkload("test-workload", "default")

	if err := workloadStore.Create(context.Background(), workload); err != nil {
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
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

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
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

	workload1 := testWorkload("test-workload-1", "default")
	workload2 := testWorkload("test-workload-2", "default")
	workload3 := testWorkload("other-workload", "production")

	if err := workloadStore.Create(context.Background(), workload1); err != nil {
		t.Fatalf("Failed to create initial workload: %v", err)
	}
	if err := workloadStore.Create(context.Background(), workload2); err != nil {
		t.Fatalf("Failed to create initial workload: %v", err)
	}
	if err := workloadStore.Create(context.Background(), workload3); err != nil {
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
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

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
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

	workload := testWorkload("test-workload", "default")
	if err := workloadStore.Create(context.Background(), workload); err != nil {
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

	_, err := workloadStore.Get(context.Background(), "default", "test-workload")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Expected workload to be deleted, but it still exists")
	}
}

func TestWorkloadDeleteNotFound(t *testing.T) {
	workloadStore := store.NewMemoryStore()
	server := NewServer(workloadStore)

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
