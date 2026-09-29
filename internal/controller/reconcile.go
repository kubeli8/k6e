package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
	"github.com/pyd-07/k6e/internal/store"
)

type ReconcileDecision struct {
	Create int
}

type WorkloadScheduler interface {
	ScheduleWorkload(ctx context.Context, ref model.WorkloadRef) (model.Assignment, error)
}

// AssignmentExecutor advances a scheduled assignment through its worker-side execution.
type AssignmentExecutor interface {
	ExecuteAssignment(ctx context.Context, assignmentID string) (model.Assignment, error)
}

type Controller struct {
	workloadStore   store.WorkloadStore
	assignmentStore store.AssignmentStore
	scheduler       WorkloadScheduler
	executor        AssignmentExecutor
	observer        RuntimeObserver

	namespace string
	interval  time.Duration
}

func NewController(
	workloadStore store.WorkloadStore,
	assignmentStore store.AssignmentStore,
	scheduler WorkloadScheduler,
	executor AssignmentExecutor,
	observer RuntimeObserver,
	namespace string,
	interval time.Duration,
) *Controller {
	return &Controller{
		workloadStore:   workloadStore,
		assignmentStore: assignmentStore,
		scheduler:       scheduler,
		executor:        executor,
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

	if err := c.ReconcileAll(ctx); err != nil {
		log.Printf("initial reconciliation failed: %v", err)
	}

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.ReconcileAll(ctx); err != nil {
				log.Printf("reconciliation failed: %v", err)
			}
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

	state := CalculateReplicaState(workload, assignments)
	decision := Decide(state)

	for i := 0; i < decision.Create; i++ {
		assignment, err := c.scheduler.ScheduleWorkload(ctx, ref)
		if err != nil {
			return err
		}
		if c.executor == nil {
			return fmt.Errorf("assignment executor not configured")
		}
		if _, err := c.executor.ExecuteAssignment(ctx, assignment.ID); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) ObserveAssignment(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error) {
	return c.observer.Inspect(ctx, assignment)
}
