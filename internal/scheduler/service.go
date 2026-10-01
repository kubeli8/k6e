package scheduler

import (
	"context"

	"github.com/google/uuid"
	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/store"
)

type Service struct {
	workloadStore   store.WorkloadStore
	nodeStore       store.NodeStore
	assignmentStore store.AssignmentStore
	scheduler       Scheduler
}

func NewService(workloadStore store.WorkloadStore, nodeStore store.NodeStore, assignmentStore store.AssignmentStore, scheduler Scheduler) *Service {
	return &Service{
		workloadStore:   workloadStore,
		nodeStore:       nodeStore,
		assignmentStore: assignmentStore,
		scheduler:       scheduler,
	}
}

func (s *Service) ScheduleWorkload(ctx context.Context, ref model.WorkloadRef) (model.Assignment, error) {
	workload, err := s.workloadStore.Get(ctx, ref)
	if err != nil {
		return model.Assignment{}, err
	}

	nodes, err := s.nodeStore.ListNodes(ctx)
	if err != nil {
		return model.Assignment{}, err
	}

	node, err := s.scheduler.Schedule(ctx, workload, nodes)
	if err != nil {
		return model.Assignment{}, err
	}

	assignment := model.Assignment{
		ID:           uuid.NewString(),
		Workload:     ref,
		NodeID:       node.ID,
		Status:       model.AssignmentStatusPending,
		TemplateHash: model.TemplateHash(workload.Spec.Template),
	}

	if err := s.assignmentStore.CreateAssignment(ctx, assignment); err != nil {
		return model.Assignment{}, err
	}
	return assignment, nil
}
