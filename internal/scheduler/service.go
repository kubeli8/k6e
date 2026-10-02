package scheduler

import (
	"context"

	"github.com/google/uuid"
	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/store"
)

// Service coordinates the scheduling pipeline: it resolves the workload,
// retrieves the list of available nodes, delegates placement to a Scheduler,
// and persists the resulting Assignment.
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

// ScheduleWorkload creates a new Pending assignment for a single replica of
// the named workload. It captures the current template hash at the time of
// scheduling so that stale assignments can be detected if the workload
// template is later updated. Returns ErrNoReadyNodes if no node is available.
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
