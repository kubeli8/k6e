package controller

import (
	"context"
	"fmt"
	"time"

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

	namespace string
	interval  time.Duration
}

func NewController(
	workloadStore store.WorkloadStore,
	assignmentStore store.AssignmentStore,
	scheduler WorkloadScheduler,
	observer RuntimeObserver,
	namespace string,
	interval time.Duration,
) *Controller {
	return &Controller{
		workloadStore:   workloadStore,
		assignmentStore: assignmentStore,
		scheduler:       scheduler,
		observer:        observer,
		namespace:       namespace,
		interval:        interval,
	}
}

func Decide(state ReplicaState) ReconcileDecision {
	decision := ReconcileDecision{
		Create: state.Missing(),
	}
	return decision
}

func (c *Controller) Start(ctx context.Context) error {
	if c.interval <= 0 {
		return fmt.Errorf("controller reconciliation interval must be positive")
	}

	_ = c.ReconcileAll(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = c.ReconcileAll(ctx)
		}
	}
}

func (c *Controller) ReconcileAll(ctx context.Context) error {
	workloads, err := c.workloadStore.List(ctx, c.namespace)
	if err != nil {
		return err
	}

	var firstError error

	for _, workload := range workloads {
		if err := c.Reconcile(ctx, model.WorkloadRef{
			Name:      workload.Metadata.Name,
			Namespace: workload.Metadata.Namespace,
		}); err != nil {
			if firstError == nil {
				firstError = err
			}
			continue
		}
	}

	return firstError
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
