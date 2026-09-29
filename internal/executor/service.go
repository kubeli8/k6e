package executor

import (
	"context"
	"fmt"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
	"github.com/pyd-07/k6e/internal/store"
)

type Service struct {
	assignmentStore store.AssignmentStore
	workloadStore   store.WorkloadStore
	nodeStore       store.NodeStore
	executor        ContainerExecutor
}

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

func (s *Service) ExecuteAssignment(ctx context.Context, assignmentID string) (model.Assignment, error) {
	assignment, err := s.assignmentStore.GetAssignment(ctx, assignmentID)
	if err != nil {
		return model.Assignment{}, err
	}

	workload, err := s.workloadStore.Get(ctx, assignment.Workload)
	if err != nil {
		return model.Assignment{}, err
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
		Name:    container.Name,
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
