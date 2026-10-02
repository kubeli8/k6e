package store

import (
	"context"

	"github.com/kubeli8/k6e/internal/model"
)

// WorkloadStore is the persistence interface for Workload resources.
// All methods are scoped to a namespace/name key and return ErrNotFound
// when the workload does not exist.
type WorkloadStore interface {
	// Create persists a new workload. Returns ErrAlreadyExists if a workload
	// with the same namespace and name is already present.
	Create(ctx context.Context, workload model.Workload) error
	Get(ctx context.Context, ref model.WorkloadRef) (model.Workload, error)
	// List returns all workloads in the given namespace.
	List(ctx context.Context, namespace string) ([]model.Workload, error)
	// Update replaces the stored workload identified by ref. Returns ErrNotFound
	// if no matching workload exists.
	Update(ctx context.Context, ref model.WorkloadRef, workload model.Workload) error
	Delete(ctx context.Context, ref model.WorkloadRef) error
}
