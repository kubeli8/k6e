package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/pyd-07/k6e/internal/model"
)

func testAssignment(name, namespace, nodeID, containerID string, status model.AssignmentStatus) model.Assignment {
	return model.Assignment{
		ID: uuid.NewString(),
		Workload: model.WorkloadRef{
			Name:      name,
			Namespace: namespace,
		},
		NodeID:      nodeID,
		Status:      status,
		ContainerID: containerID,
	}
}

func TestAssignmentMemoryStoreCreate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)
	err := store.CreateAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}
	retrieved, err := store.GetAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("get, got %v", err)
	}
	if retrieved.ID != assignment.ID {
		t.Errorf("expected ID %s, got %s", assignment.ID, retrieved.ID)
	}
}

func TestAssignmentMemoryStoreCreateWithoutID(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)
	assignment.ID = ""
	err := store.CreateAssignment(ctx, assignment)
	if err == nil {
		t.Fatalf("expected error for missing ID")
	}
	if !errors.Is(err, ErrMissingID) {
		t.Fatalf("expected ErrMissingID, got %v", err)
	}
}

func TestAssignmentMemoryStoreCreateDuplicate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)
	err := store.CreateAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}
	err = store.CreateAssignment(ctx, assignment)
	if err == nil {
		t.Fatalf("expected error for duplicate creation")
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestAssignmentMemoryStoreGet(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)
	err := store.CreateAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}

	retrieved, err := store.GetAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("get, got %v", err)
	}
	if retrieved.ID != assignment.ID {
		t.Errorf("expected ID %s, got %s", assignment.ID, retrieved.ID)
	}
}

func TestAssignmentMemoryStoreGetNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	_, err := store.GetAssignment(ctx, "nonexistent")
	if err == nil {
		t.Fatalf("expected error for non-existent assignment")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAssignmentMemoryStoreList(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	assignment1 := testAssignment("test1", "default", "node1", "container1", model.AssignmentStatusRunning)
	assignment2 := testAssignment("test2", "default", "node2", "container2", model.AssignmentStatusPending)
	assignment3 := testAssignment("test3", "other", "node3", "container3", model.AssignmentStatusCompleted)
	err := store.CreateAssignment(ctx, assignment1)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}
	err = store.CreateAssignment(ctx, assignment2)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}
	err = store.CreateAssignment(ctx, assignment3)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}

	listed, err := store.ListAssignments(ctx, "default")
	if err != nil {
		t.Fatalf("list, got %v", err)
	}
	if len(listed) != 2 {
		t.Errorf("expected 2 assignments, got %d", len(listed))
	}
}

func TestAssignmentMemoryStoreUpdateStatus(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)
	err := store.CreateAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}

	err = store.UpdateStatusAssignment(ctx, assignment.ID, model.AssignmentStatusCompleted)
	if err != nil {
		t.Fatalf("update status, got %v", err)
	}

	retrieved, err := store.GetAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("get, got %v", err)
	}
	if retrieved.Status != model.AssignmentStatusCompleted {
		t.Errorf("expected status %s, got %s", model.AssignmentStatusCompleted, retrieved.Status)
	}
}

func TestAssignmentMemoryStoreUpdateStatusNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	err := store.UpdateStatusAssignment(ctx, "nonexistent", model.AssignmentStatusCompleted)
	if err == nil {
		t.Fatalf("expected error for non-existent assignment")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAssignmentMemoryStoreDelete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)
	err := store.CreateAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}
	err = store.DeleteAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("delete, got %v", err)
	}
}

func TestAssignmentMemoryStoreDeleteNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryAssignmentStore()
	err := store.DeleteAssignment(ctx, "nonexistent")
	if err == nil {
		t.Fatalf("expected error for non-existent assignment")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
