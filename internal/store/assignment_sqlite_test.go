package store

import (
	"context"
	"errors"
	"testing"

	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/runtime"
	"github.com/kubeli8/k6e/internal/testutil"
)

func TestAssignmentSQLiteStoreCreate(t *testing.T) {
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

	assignment := testutil.Assignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

	err = store.CreateAssignment(ctx, assignment)
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

func TestAssignmentSQLiteStoreCreateWithoutID(t *testing.T) {
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
	assignment := testutil.Assignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)
	assignment.ID = ""

	err = store.CreateAssignment(ctx, assignment)
	if err == nil {
		t.Fatalf("expected error for missing ID")
	}
	if !errors.Is(err, ErrMissingID) {
		t.Fatalf("expected ErrMissingID, got %v", err)
	}
}

func TestAssignmentSQLiteStoreCreateDuplicate(t *testing.T) {
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
	assignment := testutil.Assignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

	err = store.CreateAssignment(ctx, assignment)
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

func TestAssignmentSQLiteStoreGet(t *testing.T) {
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
	assignment := testutil.Assignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

	err = store.CreateAssignment(ctx, assignment)
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

func TestAssignmentSQLiteStoreGetNotFound(t *testing.T) {
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
	_, err = store.GetAssignment(ctx, "nonexistent")
	if err == nil {
		t.Fatalf("expected error for non-existent assignment")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteStoreList(t *testing.T) {
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

	assignment1 := testutil.Assignment("test1", "default", "node1", "container1", model.AssignmentStatusRunning)
	assignment2 := testutil.Assignment("test2", "default", "node2", "container2", model.AssignmentStatusPending)
	assignment3 := testutil.Assignment("test3", "other", "node3", "container3", model.AssignmentStatusFailed)

	err = store.CreateAssignment(ctx, assignment1)
	if err != nil {
		t.Fatalf("create assignment1, got %v", err)
	}
	err = store.CreateAssignment(ctx, assignment2)
	if err != nil {
		t.Fatalf("create assignment2, got %v", err)
	}
	err = store.CreateAssignment(ctx, assignment3)
	if err != nil {
		t.Fatalf("create assignment3, got %v", err)
	}

	assignments, err := store.ListAssignments(ctx, "default")
	if err != nil {
		t.Fatalf("list, got %v", err)
	}

	if len(assignments) != 2 {
		t.Errorf("expected 2 assignments, got %d", len(assignments))
	}
}

func TestSQLiteStoreListForWorkload(t *testing.T) {
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

	assignment1 := testutil.Assignment("test1", "default", "node1", "container1", model.AssignmentStatusRunning)
	assignment2 := testutil.Assignment("test2", "default", "node2", "container2", model.AssignmentStatusPending)
	assignment3 := testutil.Assignment("test3", "other", "node3", "container3", model.AssignmentStatusFailed)

	err = store.CreateAssignment(ctx, assignment1)
	if err != nil {
		t.Fatalf("create assignment1, got %v", err)
	}
	err = store.CreateAssignment(ctx, assignment2)
	if err != nil {
		t.Fatalf("create assignment2, got %v", err)
	}
	err = store.CreateAssignment(ctx, assignment3)
	if err != nil {
		t.Fatalf("create assignment3, got %v", err)
	}

	assignments, err := store.ListAssignmentsForWorkload(ctx, model.WorkloadRef{Name: "test1", Namespace: "default"})
	if err != nil {
		t.Fatalf("list by workload, got %v", err)
	}

	if len(assignments) != 1 {
		t.Fatalf("expected 1 assignment for workload 'test1', got %d", len(assignments))
	}
	if assignments[0].ID != assignment1.ID {
		t.Errorf("expected assignment ID %s, got %s", assignment1.ID, assignments[0].ID)
	}
}

func TestSQLiteStoreListForWorkloadEmpty(t *testing.T) {
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

	assignments, err := store.ListAssignmentsForWorkload(ctx, model.WorkloadRef{Name: "nonexistent", Namespace: "default"})
	if err != nil {
		t.Fatalf("list by workload, got %v", err)
	}

	if len(assignments) != 0 {
		t.Fatalf("expected 0 assignments for non-existent workload, got %d", len(assignments))
	}
}

func TestSQLiteStoreUpdateStatus(t *testing.T) {
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
	assignment := testutil.Assignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

	err = store.CreateAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}

	err = store.UpdateStatusAssignment(ctx, assignment.ID, model.AssignmentStatusPending)
	if err != nil {
		t.Fatalf("update status, got %v", err)
	}

	retrieved, err := store.GetAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("get, got %v", err)
	}
	if retrieved.Status != model.AssignmentStatusPending {
		t.Errorf("expected status %s, got %s", model.AssignmentStatusPending, retrieved.Status)
	}
}

func TestSQLiteStoreUpdateStatusNotFound(t *testing.T) {
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
	err = store.UpdateStatusAssignment(ctx, "nonexistent", model.AssignmentStatusPending)
	if err == nil {
		t.Fatalf("expected error for non-existent assignment")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteStoreUpdateExecution(t *testing.T) {
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
	assignment := testutil.Assignment("test", "default", "node1", "", model.AssignmentStatusPending)

	err = store.CreateAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}

	containerID := "container123"
	err = store.UpdateAssignmentExecution(ctx, assignment.ID, runtime.ContainerID(containerID), model.AssignmentStatusRunning)
	if err != nil {
		t.Fatalf("update execution, got %v", err)
	}

	retrieved, err := store.GetAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("get, got %v", err)
	}
	if retrieved.ContainerID != containerID {
		t.Errorf("expected container ID %s, got %s", containerID, retrieved.ContainerID)
	}
	if retrieved.Status != model.AssignmentStatusRunning {
		t.Errorf("expected status %s, got %s", model.AssignmentStatusRunning, retrieved.Status)
	}
}

func TestSQLiteStoreUpdateExecutionNotFound(t *testing.T) {
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
	err = store.UpdateAssignmentExecution(ctx, "nonexistent", runtime.ContainerID("container123"), model.AssignmentStatusRunning)
	if err == nil {
		t.Fatalf("expected error for non-existent assignment")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteStoreDelete(t *testing.T) {
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
	assignment := testutil.Assignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

	err = store.CreateAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("create, got %v", err)
	}

	err = store.DeleteAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("delete, got %v", err)
	}

	_, err = store.GetAssignment(ctx, assignment.ID)
	if err == nil {
		t.Fatalf("expected error for deleted assignment")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteStoreDeleteNotFound(t *testing.T) {
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
	err = store.DeleteAssignment(ctx, "nonexistent")
	if err == nil {
		t.Fatalf("expected error for non-existent assignment")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
