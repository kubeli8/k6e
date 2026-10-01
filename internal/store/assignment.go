package store

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

// AssignmentStore is the persistence interface for Assignment records.
type AssignmentStore interface {
	// CreateAssignment persists a new assignment. Returns ErrMissingID if the
	// assignment has no ID, or ErrAlreadyExists if an assignment with the same
	// ID is already present.
	CreateAssignment(ctx context.Context, assignment model.Assignment) error
	GetAssignment(ctx context.Context, id string) (model.Assignment, error)
	// ListAssignments returns all assignments whose workload belongs to namespace.
	ListAssignments(ctx context.Context, namespace string) ([]model.Assignment, error)
	// ListAssignmentsForWorkload returns all assignments for the identified workload,
	// regardless of their current status.
	ListAssignmentsForWorkload(ctx context.Context, ref model.WorkloadRef) ([]model.Assignment, error)
	// UpdateStatusAssignment updates only the Status field of the assignment.
	UpdateStatusAssignment(ctx context.Context, id string, status model.AssignmentStatus) error
	// UpdateAssignmentExecution atomically records the containerID and status
	// that result from a successful or failed container start attempt.
	UpdateAssignmentExecution(ctx context.Context, id string, containerID runtime.ContainerID, status model.AssignmentStatus) error
	DeleteAssignment(ctx context.Context, id string) error
}
