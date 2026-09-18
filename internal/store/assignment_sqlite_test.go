package store

import (
	"context"
	"errors"
	"testing"

	"github.com/pyd-07/k6e/internal/model"
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

	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

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
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)
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
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

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
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

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

	assignment1 := testAssignment("test1", "default", "node1", "container1", model.AssignmentStatusRunning)
	assignment2 := testAssignment("test2", "default", "node2", "container2", model.AssignmentStatusPending)
	assignment3 := testAssignment("test3", "other", "node3", "container3", model.AssignmentStatusFailed)

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
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

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
	assignment := testAssignment("test", "default", "node1", "container1", model.AssignmentStatusRunning)

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
