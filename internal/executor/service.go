package executor

import (
	"context"
	"fmt"

	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/runtime"
	"github.com/kubeli8/k6e/internal/store"
)

// Service executes and deletes assignments on behalf of the controller and
// the HTTP API. It pairs assignment-store bookkeeping with container
// operations performed by a ContainerExecutor.
type Service struct {
	assignmentStore store.AssignmentStore
	workloadStore   store.WorkloadStore
	nodeStore       store.NodeStore
	executor        ContainerExecutor
}

// NewService constructs a Service. All three stores and the executor must be
// non-nil for the service to operate; ExecuteAssignment returns an error when
// a dependency fails to resolve a required record.
func NewService(
	assignmentStore store.AssignmentStore,
	workloadStore store.WorkloadStore,
	nodeStore store.NodeStore,
	executor ContainerExecutor,
) *Service {
	return &Service{
		assignmentStore: assignmentStore,
		workloadStore:   workloadStore,
		nodeStore:       nodeStore,
		executor:        executor,
	}
}

// ExecuteAssignment starts the container for a Pending assignment and records
// the outcome on the assignment.
//
// It enforces two safety checks before starting anything:
//
//   - Stale-template protection: the assignment's TemplateHash must match the
//     current hash of the workload's template. On mismatch the assignment is
//     marked Failed and an error is returned, preventing containers from being
//     started from an outdated template during a rollout.
//   - Single-container requirement: the workload template must contain exactly
//     one container, which is the current k6e limitation.
//
// The container is named "k6e-<container name>-<first 8 chars of the
// assignment ID>" to keep names unique across assignments.
//
// If the container fails to start, the assignment is marked Failed with no
// container ID and the error is returned alongside the failed assignment. On
// success the assignment is updated to Running with its container ID and
// returned.
func (s *Service) ExecuteAssignment(ctx context.Context, assignmentID string) (model.Assignment, error) {
	assignment, err := s.assignmentStore.GetAssignment(ctx, assignmentID)
	if err != nil {
		return model.Assignment{}, err
	}

	workload, err := s.workloadStore.Get(ctx, assignment.Workload)
	if err != nil {
		return model.Assignment{}, err
	}

	currentTemplateHash := model.TemplateHash(workload.Spec.Template)
	if assignment.TemplateHash != currentTemplateHash {
		s.assignmentStore.UpdateStatusAssignment(ctx, assignmentID, model.AssignmentStatusFailed)
		return model.Assignment{}, fmt.Errorf("template hash mismatch for assignment %s: expected %s, got %s", assignmentID, assignment.TemplateHash, currentTemplateHash)
	}

	node, err := s.nodeStore.GetNode(ctx, assignment.NodeID)
	if err != nil {
		return model.Assignment{}, err
	}

	if len(workload.Spec.Template.Containers) != 1 {
		return model.Assignment{}, fmt.Errorf("workload must contain exactly one container")
	}

	container := workload.Spec.Template.Containers[0]

	spec := runtime.ContainerSpec{
		Name:    "k6e-" + container.Name + "-" + assignmentID[:8],
		Image:   container.Image,
		Command: container.Command,
		Args:    container.Args,
	}

	containerID, err := s.executor.RunContainer(ctx, node, spec)
	if err != nil {
		_ = s.assignmentStore.UpdateAssignmentExecution(ctx, assignmentID, "", model.AssignmentStatusFailed)
		assignment.Status = model.AssignmentStatusFailed
		return assignment, err
	}

	err = s.assignmentStore.UpdateAssignmentExecution(ctx, assignmentID, containerID, model.AssignmentStatusRunning)
	if err != nil {
		return model.Assignment{}, err
	}

	assignment.Status = model.AssignmentStatusRunning
	assignment.ContainerID = string(containerID)
	return assignment, nil
}

// DeleteAssignment stops and removes the container backing an assignment and
// marks it Completed. It returns an error if the assignment has no container
// ID (it never started) or if the stop or remove operation fails; the status
// is only updated after the container has been successfully removed.
func (s *Service) DeleteAssignment(ctx context.Context, assignmentID string) error {
	assignment, err := s.assignmentStore.GetAssignment(ctx, assignmentID)
	if err != nil {
		return err
	}

	if assignment.ContainerID == "" {
		return fmt.Errorf("assignment %s has no container ID", assignmentID)
	}

	node, err := s.nodeStore.GetNode(ctx, assignment.NodeID)
	if err != nil {
		return err
	}

	err = s.executor.DeleteContainer(ctx, node, runtime.ContainerID(assignment.ContainerID))
	if err != nil {
		return err
	}

	err = s.assignmentStore.UpdateStatusAssignment(ctx, assignmentID, model.AssignmentStatusCompleted)
	if err != nil {
		return err
	}

	return nil
}
