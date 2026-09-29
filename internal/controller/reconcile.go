package controller

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/store"
)

type ReconcileDecision struct {
	Create int
}

type WorkloadScheduler interface {
	Schedule(ctx context.Context, ref model.WorkloadRef) (model.Assignment, error)
}

type Controller struct {
	workloadStore   store.WorkloadStore
	assignmentStore store.AssignmentStore
	scheduler       WorkloadScheduler
}

func NewController(
	workloadStore store.WorkloadStore,
	assignmentStore store.AssignmentStore,
	scheduler WorkloadScheduler,
) *Controller {
	return &Controller{
		workloadStore:   workloadStore,
		assignmentStore: assignmentStore,
		scheduler:       scheduler,
	}
}

func Decide(state ReplicaState) ReconcileDecision {
	decision := ReconcileDecision{
		Create: state.Missing(),
	}
	return decision
}

func (c *Controller) Reconcile(ctx context.Context, ref model.WorkloadRef) error {
	workload, err := c.workloadStore.Get(ctx, ref)
	if err != nil {
		return err
	}

	assignments, err := c.assignmentStore.ListAssignmentsForWorkload(ctx, ref)
	if err != nil {
		return err
	}

	state := CalaculateReplicaState(workload, assignments)
	decision := Decide(state)

	for i := 0; i < decision.Create; i++ {
		_, err := c.scheduler.Schedule(ctx, ref)
		if err != nil {
			return err
		}
	}

	return nil
}
