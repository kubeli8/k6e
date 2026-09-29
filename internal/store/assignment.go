package store

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

type AssignmentStore interface {
	CreateAssignment(ctx context.Context, assignment model.Assignment) error
	GetAssignment(ctx context.Context, id string) (model.Assignment, error)
	ListAssignments(ctx context.Context, namespace string) ([]model.Assignment, error)
	ListAssignmentsForWorkload(ctx context.Context, ref model.WorkloadRef) ([]model.Assignment, error)
	UpdateStatusAssignment(ctx context.Context, id string, status model.AssignmentStatus) error
	UpdateAssignmentExecution(ctx context.Context, id string, containerID runtime.ContainerID, status model.AssignmentStatus) error
	DeleteAssignment(ctx context.Context, id string) error
}
