package store

import (
	"context"
	"errors"
	"testing"
)

func TestSQLiteStore(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close SQLiteStore: %v", err)
		}
	}()
}

func TestSQLiteStore_InitializeSchema(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close SQLiteStore: %v", err)
		}
	}()

	var tableName string
	err = store.db.QueryRow(`
		SELECT name
		FROM sqlite_master
		WHERE type='table' AND name='workloads'
	`).Scan(&tableName)
	if err != nil {
		t.Fatalf("failed to query table name: %v", err)
	}
	if tableName != "workloads" {
		t.Errorf("expected table name 'workloads', got '%s'", tableName)
	}
}

func TestSQLiteStore_Create(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close SQLiteStore: %v", err)
		}
	}()

	workload := testWorkload("test-workload", "default")

	err = store.Create(context.Background(), workload)
	if err != nil {
		t.Fatalf("failed to create workload: %v", err)
	}

	var specJSON string
	err = store.db.QueryRow(`
		SELECT spec_json
		FROM workloads
		WHERE namespace = ? AND name = ?
	`, workload.Metadata.Namespace, workload.Metadata.Name).Scan(&specJSON)
	if err != nil {
		t.Fatalf("failed to query workload: %v", err)
	}

	if specJSON == "" {
		t.Errorf("expected non-empty spec_json, got empty")
	}
}

func TestSQLiteStore_CreateDuplicate(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close SQLiteStore: %v", err)
		}
	}()

	workload := testWorkload("test-workload", "default")
	ctx := context.Background()

	if err := store.Create(ctx, workload); err != nil {
		t.Fatalf("failed to create workload: %v", err)
	}

	err = store.Create(ctx, workload)

	if err == nil {
		t.Fatalf("expected error when creating duplicate workload, got nil")
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got: %v", err)
	}
}

func TestSQLiteStore_Get(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close SQLiteStore: %v", err)
		}
	}()

	workload := testWorkload("test-workload", "default")
	ctx := context.Background()

	if err := store.Create(ctx, workload); err != nil {
		t.Fatalf("failed to create workload: %v", err)
	}

	retrievedWorkload, err := store.Get(ctx, workload.Metadata.Namespace, workload.Metadata.Name)
	if err != nil {
		t.Fatalf("failed to get workload: %v", err)
	}

	if retrievedWorkload.Metadata.Name != workload.Metadata.Name {
		t.Errorf("expected workload name '%s', got '%s'", workload.Metadata.Name, retrievedWorkload.Metadata.Name)
	}
	if retrievedWorkload.Metadata.Namespace != workload.Metadata.Namespace {
		t.Errorf("expected workload namespace '%s', got '%s'", workload.Metadata.Namespace, retrievedWorkload.Metadata.Namespace)
	}
	if retrievedWorkload.Spec.Replicas != workload.Spec.Replicas {
		t.Errorf("expected workload replicas '%d', got '%d'", workload.Spec.Replicas, retrievedWorkload.Spec.Replicas)
	}
}

func TestSQLiteStore_GetNotFound(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close SQLiteStore: %v", err)
		}
	}()

	ctx := context.Background()
	_, err = store.Get(ctx, "nonexistent-namespace", "nonexistent-name")
	if err == nil {
		t.Fatalf("expected error when getting non-existent workload, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteStore_List(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close SQLiteStore: %v", err)
		}
	}()

	ctx := context.Background()
	workload1 := testWorkload("workload-1", "default")
	workload2 := testWorkload("workload-2", "default")
	workload3 := testWorkload("workload-3", "other-namespace")

	if err := store.Create(ctx, workload1); err != nil {
		t.Fatalf("failed to create workload1: %v", err)
	}
	if err := store.Create(ctx, workload2); err != nil {
		t.Fatalf("failed to create workload2: %v", err)
	}
	if err := store.Create(ctx, workload3); err != nil {
		t.Fatalf("failed to create workload3: %v", err)
	}

	workloads, err := store.List(ctx, "default")
	if err != nil {
		t.Fatalf("failed to list workloads: %v", err)
	}

	if len(workloads) != 2 {
		t.Errorf("expected 2 workloads in 'default' namespace, got %d", len(workloads))
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close SQLiteStore: %v", err)
		}
	}()

	workload := testWorkload("test-workload", "default")
	ctx := context.Background()
	if err := store.Create(ctx, workload); err != nil {
		t.Fatalf("failed to create workload: %v", err)
	}

	if err := store.Delete(ctx, "default", "test-workload"); err != nil {
		t.Fatalf("failed to delete the workload: %v", err)
	}

	_, err = store.Get(ctx, "default", "test-workload")
	if err == nil {
		t.Fatalf("expected error when getting deleted workload, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteStore_DeleteNotFound(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create SQLiteStore: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close SQLiteStore: %v", err)
		}
	}()

	ctx := context.Background()
	err = store.Delete(ctx, "default", "nonexistent-workload")
	if err == nil {
		t.Fatalf("expected error when deleting non-existent workload, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}
