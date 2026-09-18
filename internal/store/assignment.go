package store

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
)

type AssignmentStore interface {
	CreateAssignment(ctx context.Context, assignment model.Assignment) error
	GetAssignment(ctx context.Context, id string) (model.Assignment, error)
	ListAssignments(ctx context.Context, namespace string) ([]model.Assignment, error)
	UpdateStatusAssignment(ctx context.Context, id string, status model.AssignmentStatus) error
	DeleteAssignment(ctx context.Context, id string) error
}
