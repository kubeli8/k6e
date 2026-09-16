package store

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
)

type WorkloadStore interface {
	Create(ctx context.Context, workload model.Workload) error
	Get(ctx context.Context, namespace, name string) (model.Workload, error)
	List(ctx context.Context, namespace string) ([]model.Workload, error)
	Delete(ctx context.Context, namespace, name string) error
}
