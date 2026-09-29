package controller

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
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
	observer        RuntimeObserver
}

func NewController(
	workloadStore store.WorkloadStore,
	assignmentStore store.AssignmentStore,
	scheduler WorkloadScheduler,
	observer RuntimeObserver,
) *Controller {
	return &Controller{
		workloadStore:   workloadStore,
		assignmentStore: assignmentStore,
		scheduler:       scheduler,
		observer:        observer,
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

	for i := range assignments {
		assignment := &assignments[i]
		if assignment.Status != model.AssignmentStatusRunning {
			continue
		}

		if assignment.ContainerID == "" {
			continue
		}

		info, err := c.observer.Inspect(ctx, *assignment)
		if err != nil {
			return err
		}

		observed := RuntimeStateStopped
		if info.Running {
			observed = RuntimeStateRunning
		}

		newStatus := DetermineAssignmentStatus(assignment.Status, observed)
		if newStatus == assignment.Status {
			continue
		}

		if err := c.assignmentStore.UpdateStatusAssignment(ctx, assignment.ID, newStatus); err != nil {
			return err
		}
		assignment.Status = newStatus
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

func (c *Controller) ObserveAssignment(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error) {
	return c.observer.Inspect(ctx, assignment)
}
