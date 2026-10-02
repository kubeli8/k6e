// Package controller implements the k6e reconciliation loop. It periodically
// compares the desired state declared by Workload resources against the
// observed state of running containers, and schedules or deletes assignments
// to converge them.
//
// The reconciliation flow for each workload is:
//
//  1. Observe: inspect every Running assignment with a ContainerID via the
//     RuntimeObserver to detect containers that have stopped unexpectedly.
//  2. Calculate: derive a ReplicaState from the workload's desired replica
//     count and the current status of all its assignments.
//  3. Decide: produce a ReconcileDecision that specifies how many replicas
//     to create or delete.
//  4. Converge: call the WorkloadScheduler and AssignmentExecutor for each
//     missing replica; call AssignmentExecutor.DeleteAssignment for each
//     excess running assignment.
package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/runtime"
	"github.com/kubeli8/k6e/internal/store"
)

// ReconcileDecision is the output of Decide: the number of replicas to create
// and the number of running replicas to delete in a single reconciliation pass.
type ReconcileDecision struct {
	Create int
	Delete int
}

// WorkloadScheduler selects a node for a workload replica and persists a new
// Assignment in the Pending state.
type WorkloadScheduler interface {
	ScheduleWorkload(ctx context.Context, ref model.WorkloadRef) (model.Assignment, error)
}

// AssignmentExecutor carries out a scheduled assignment on its target node:
// ExecuteAssignment starts the container and records the result on the
// assignment; DeleteAssignment stops and removes the container for a
// scale-down.
type AssignmentExecutor interface {
	ExecuteAssignment(ctx context.Context, assignmentID string) (model.Assignment, error)
	DeleteAssignment(ctx context.Context, assignmentID string) error
}

// Controller drives the reconciliation loop for all workloads in a namespace.
// It runs on a fixed interval and processes each workload independently,
// logging but not aborting on per-workload errors.
type Controller struct {
	workloadStore   store.WorkloadStore
	assignmentStore store.AssignmentStore
	scheduler       WorkloadScheduler
	executor        AssignmentExecutor
	observer        RuntimeObserver

	namespace string
	interval  time.Duration
}

// NewController constructs a Controller. interval must be positive; Start
// will return an error immediately if it is not.
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

// Decide translates a ReplicaState into a ReconcileDecision. The number of
// replicas to create equals the number of missing replicas; the number to
// delete equals the number of running replicas above the desired count.
// Pending replicas are treated as in-flight and count toward the desired
// total, so they are neither re-created nor deleted.
func Decide(state ReplicaState) ReconcileDecision {
	decision := ReconcileDecision{
		Create: state.Missing(),
		Delete: state.Delete(),
	}
	return decision
}

// Start performs an immediate reconciliation pass and then reconciles on every
// tick of interval until ctx is cancelled. A failing reconciliation pass is
// logged but does not stop the loop. Start returns ctx.Err() when the context
// is cancelled, or an error if interval is non-positive.
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

// ReconcileAll reconciles every workload in the controller's namespace.
// It returns the first error encountered while still attempting to reconcile
// all remaining workloads.
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

// Reconcile drives a single workload toward its desired state. It:
//  1. Observes the runtime state of Running assignments and marks stopped
//     containers as Failed.
//  2. Removes Running assignments created from an obsolete workload template.
//  3. Recalculates the assignment state from the latest persisted snapshot.
//  4. Creates missing assignments and removes excess running assignments.
func (c *Controller) Reconcile(ctx context.Context, ref model.WorkloadRef) error {
	workload, err := c.workloadStore.Get(ctx, ref)
	if err != nil {
		return err
	}

	assignments, err := c.assignmentStore.ListAssignmentsForWorkload(ctx, ref)
	if err != nil {
		return err
	}

	if err := c.observeAssignments(ctx, assignments); err != nil {
		return err
	}

	assignments, err = c.assignmentStore.ListAssignmentsForWorkload(ctx, ref)
	if err != nil {
		return err
	}

	if err := c.reconcileObsoleteAssignments(ctx, workload, assignments); err != nil {
		return err
	}

	assignments, err = c.assignmentStore.ListAssignmentsForWorkload(ctx, ref)
	if err != nil {
		return err
	}

	if err := c.reconcileReplicas(ctx, workload, ref, assignments); err != nil {
		return err
	}

	return nil
}

// ObserveAssignment queries the runtime state of the container associated with
// assignment by forwarding the request to the RuntimeObserver.
func (c *Controller) ObserveAssignment(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error) {
	return c.observer.Inspect(ctx, assignment)
}

// returnFirstRunningAssignment returns the first running assignment in the list, or nil if none is found.
func returnFirstRunningAssignment(assignments []model.Assignment) *model.Assignment {
	for i := range assignments {
		if assignments[i].Status == model.AssignmentStatusRunning {
			return &assignments[i]
		}
	}
	return nil
}

// observeAssignments reconciles the observed runtime state of running
// assignments with their persisted status.
func (c *Controller) observeAssignments(ctx context.Context, assignments []model.Assignment) error {
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
	return nil
}

// obsoleteAssignments returns the assignments that IsAssignmentObsolete
// considers stale for currentTemplateHash. It returns nil when no assignment
// is obsolete.
func obsoleteAssignments(assignments []model.Assignment, currentTemplateHash string) []model.Assignment {
	var obsolete []model.Assignment

	for _, assignment := range assignments {
		if IsAssignmentObsolete(assignment, currentTemplateHash) {
			obsolete = append(obsolete, assignment)
		}
	}

	return obsolete
}

// reconcileObsoleteAssignments removes running assignments that no longer
// represent the current workload template.
func (c *Controller) reconcileObsoleteAssignments(ctx context.Context, workload model.Workload, assignments []model.Assignment) error {
	currentTemplateHash := model.TemplateHash(workload.Spec.Template)
	obsolete := obsoleteAssignments(assignments, currentTemplateHash)

	for _, assignment := range obsolete {
		if c.executor == nil {
			return fmt.Errorf("assignment executor not configured")
		}

		if err := c.executor.DeleteAssignment(ctx, assignment.ID); err != nil {
			return err
		}
	}

	return nil
}

// reconcileReplicas converges the workload's assigned replicas
// towards its desirec replica count by creating missing assignments
// and removing excess running assignments.
func (c *Controller) reconcileReplicas(ctx context.Context, workload model.Workload, ref model.WorkloadRef, assignments []model.Assignment) error {
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

	for i := 0; i < decision.Delete; i++ {
		assignments, err := c.assignmentStore.ListAssignmentsForWorkload(ctx, ref)
		if err != nil {
			return err
		}
		if len(assignments) == 0 {
			return fmt.Errorf("no assignments found for workload %s", ref.Name)
		}

		assignment := returnFirstRunningAssignment(assignments)
		if assignment == nil {
			return fmt.Errorf("no running assignments found for workload %s", ref.Name)
		}

		if c.executor == nil {
			return fmt.Errorf("assignment executor not configured")
		}
		if err := c.executor.DeleteAssignment(ctx, assignment.ID); err != nil {
			return err
		}
	}

	return nil
}
