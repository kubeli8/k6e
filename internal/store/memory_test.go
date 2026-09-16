package store

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

func TestMemoryStoreCreateAndGet(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	want := testWorkload("test-workload", "default")

	if err := store.Create(ctx, want); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.Get(ctx, want.Metadata.Namespace, want.Metadata.Name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Metadata.Name != want.Metadata.Name {
		t.Errorf("Get() got = %v, want = %v", got.Metadata.Name, want.Metadata.Name)
	}

	if got.Spec.Replicas != want.Spec.Replicas {
		t.Errorf("Get() got = %v, want = %v", got.Spec.Replicas, want.Spec.Replicas)
	}
}

func TestMemoryStoreDuplicateCreate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	workload := testWorkload("test-workload", "default")
	if err := store.Create(ctx, workload); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err := store.Create(ctx, workload)

	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("Second Create() expected error = %v, got = %v", ErrAlreadyExists, err)
	}
}

func TestMemoryStoreGetNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	_, err := store.Get(ctx, "default", "non-existent-workload")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() expected error = %v, got = %v", ErrNotFound, err)
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	workload := testWorkload("test-workload", "default")

	if err := store.Create(ctx, workload); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.Delete(ctx, workload.Metadata.Namespace, workload.Metadata.Name); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := store.Get(ctx, workload.Metadata.Namespace, workload.Metadata.Name)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() after Delete expected error = %v, got = %v", ErrNotFound, err)
	}
}

func TestMemoryStoreDeleteNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	err := store.Delete(ctx, "default", "non-existent-workload")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() expected error = %v, got = %v", ErrNotFound, err)
	}
}

func TestMemoryStoreList(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	workloads := []model.Workload{
		testWorkload("workload-1", "default"),
		testWorkload("workload-2", "default"),
		testWorkload("workload-3", "other-namespace"),
	}

	workloads[0].Metadata.Namespace = "default"
	workloads[1].Metadata.Namespace = "default"
	workloads[2].Metadata.Namespace = "other-namespace"

	for _, workload := range workloads {
		if err := store.Create(ctx, workload); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	got, err := store.List(ctx, "default")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List() got = %v, want = %v", len(got), 2)
	}
}
