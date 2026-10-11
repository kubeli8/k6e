package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/runtime"
	"github.com/kubeli8/k6e/internal/store"
	"github.com/kubeli8/k6e/internal/testutil"
)

type fakeScheduler struct {
	calls int
	err   error
	store *store.MemoryAssignmentStore
	refs  []model.WorkloadRef
}

func (s *fakeScheduler) ScheduleWorkload(_ context.Context, ref model.WorkloadRef) (model.Assignment, error) {
	s.calls++
	s.refs = append(s.refs, ref)
	if s.err != nil {
		return model.Assignment{}, s.err
	}
	assignment := model.Assignment{ID: fmt.Sprintf("assignment-%d", s.calls), Workload: ref, Status: model.AssignmentStatusPending}
	if s.store != nil {
		if err := s.store.CreateAssignment(context.Background(), assignment); err != nil {
			return model.Assignment{}, err
		}
	}
	return assignment, nil
}

type fakeAssignmentExecutor struct {
	calls              int
	deleteCalls        int
	deletedAssignments []model.Assignment
	err                error
	called             chan<- string
}

func (e *fakeAssignmentExecutor) ExecuteAssignment(_ context.Context, id string) (model.Assignment, error) {
	e.calls++
	if e.called != nil {
		e.called <- id
	}
	if e.err != nil {
		return model.Assignment{}, e.err
	}
	return model.Assignment{ID: id, Status: model.AssignmentStatusRunning}, nil
}

func (e *fakeAssignmentExecutor) DeleteAssignment(_ context.Context, id string) error {
	e.deleteCalls++
	e.deletedAssignments = append(e.deletedAssignments, model.Assignment{ID: id})
	return e.err
}

type fakeObserver struct {
	calls int
	info  runtime.ContainerInfo
	err   error

	inspect func(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error)
}

func (o *fakeObserver) Inspect(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error) {
	o.calls++
	if o.inspect != nil {
		return o.inspect(ctx, assignment)
	}
	if o.err != nil {
		return runtime.ContainerInfo{}, o.err
	}
	return o.info, nil
}

func newControllerFixture(t *testing.T, replicas int32, assignments []model.Assignment, scheduler *fakeScheduler, executor *fakeAssignmentExecutor, observer *fakeObserver, interval time.Duration) (*Controller, model.WorkloadRef, *store.MemoryAssignmentStore) {
	t.Helper()
	ctx := context.Background()
	workloads := store.NewMemoryStore()
	assignmentStore := store.NewMemoryAssignmentStore()
	ref := model.WorkloadRef{Name: "nginx", Namespace: "default"}
	workload := testutil.WorkloadWithReplicas(ref.Name, ref.Namespace, replicas)

	if err := workloads.Create(ctx, workload); err != nil {
		t.Fatal(err)
	}
	for _, assignment := range assignments {
		if err := assignmentStore.CreateAssignment(ctx, assignment); err != nil {
			t.Fatal(err)
		}
	}
	return NewController(workloads, assignmentStore, scheduler, executor, observer, ref.Namespace, interval), ref, assignmentStore
}

func TestControllerReconcileSchedulesAndExecutesMissingReplica(t *testing.T) {
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
	c, ref, _ := newControllerFixture(t, 1, nil, scheduler, executor, &fakeObserver{}, time.Second)
	if err := c.Reconcile(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if scheduler.calls != 1 || executor.calls != 1 {
		t.Fatalf("schedule=%d execute=%d, want 1 each", scheduler.calls, executor.calls)
	}
}

func TestControllerReconcileExecutorErrorIsPropagated(t *testing.T) {
	want := errors.New("agent unavailable")
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{err: want}
	c, ref, _ := newControllerFixture(t, 1, nil, scheduler, executor, &fakeObserver{}, time.Second)
	if err := c.Reconcile(context.Background(), ref); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestControllerReconcileSchedulerError(t *testing.T) {
	want := errors.New("no ready nodes")
	scheduler, executor := &fakeScheduler{err: want}, &fakeAssignmentExecutor{}
	c, ref, _ := newControllerFixture(t, 1, nil, scheduler, executor, &fakeObserver{}, time.Second)
	if err := c.Reconcile(context.Background(), ref); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	if executor.calls != 0 {
		t.Fatalf("executor called %d times after scheduler error", executor.calls)
	}
}

func TestControllerReconcile(t *testing.T) {
	tests := []struct {
		name                              string
		desired, running, pending, failed int
		expectedSchedules                 int
	}{
		{name: "already converged", desired: 3, running: 3, expectedSchedules: 0},
		{name: "one missing", desired: 3, running: 2, expectedSchedules: 1},
		{name: "pending counts as in flight", desired: 3, running: 2, pending: 1, expectedSchedules: 0},
		{name: "all missing", desired: 3, expectedSchedules: 3},
		{name: "failed does not count", desired: 3, running: 2, failed: 1, expectedSchedules: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
			ref := model.WorkloadRef{Name: "nginx", Namespace: "default"}
			assignments := make([]model.Assignment, 0, tt.running+tt.pending+tt.failed)
			for _, item := range []struct {
				status model.AssignmentStatus
				count  int
			}{{model.AssignmentStatusRunning, tt.running}, {model.AssignmentStatusPending, tt.pending}, {model.AssignmentStatusFailed, tt.failed}} {
				for i := 0; i < item.count; i++ {
					assignment := model.Assignment{ID: fmt.Sprintf("%s-%d", item.status, i), Workload: ref, Status: item.status}
					if item.status == model.AssignmentStatusRunning {
						assignment.ContainerID = fmt.Sprintf("container-%d", i)
					}
					assignments = append(assignments, assignment)
				}
			}
			c, _, _ := newControllerFixture(t, int32(tt.desired), assignments, scheduler, executor, &fakeObserver{info: runtime.ContainerInfo{Running: true}}, time.Second)
			if err := c.Reconcile(context.Background(), ref); err != nil {
				t.Fatal(err)
			}
			if scheduler.calls != tt.expectedSchedules || executor.calls != tt.expectedSchedules {
				t.Fatalf("schedule=%d execute=%d, want %d each", scheduler.calls, executor.calls, tt.expectedSchedules)
			}
		})
	}
}

func TestControllerReconcileReplacesAssignmentOnUnavailableNode(t *testing.T) {
	ctx := context.Background()
	replicas := int32(1)

	assgnments := []model.Assignment{
		{ID: "old", Workload: model.WorkloadRef{Name: "nginx", Namespace: "default"}, Status: model.AssignmentStatusRunning, ContainerID: "container-old", NodeID: "node-unavailable"},
	}

	scheduler := &fakeScheduler{
		calls: 0,
	}
	executor := &fakeAssignmentExecutor{}
	observer := &fakeObserver{
		inspect: func(context.Context, model.Assignment) (runtime.ContainerInfo, error) {
			return runtime.ContainerInfo{}, ErrNodeUnavailable
		},
	}
	c, ref, assignmentStore := newControllerFixture(t, replicas, assgnments, scheduler, executor, observer, time.Second)
	scheduler.store = assignmentStore
	if err := c.Reconcile(ctx, ref); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	got, err := assignmentStore.ListAssignmentsForWorkload(ctx, ref)
	if err != nil {
		t.Fatalf("ListAssignmentsForWorkload() error = %v", err)
	}

	var oldFailed, replacementCreated bool
	for _, a := range got {
		switch a.ID {
		case "old":
			oldFailed = a.Status == model.AssignmentStatusFailed
		case "assignment-1":
			replacementCreated = true
		}
	}
	if !oldFailed {
		t.Fatal("expected old assignment to be failed")
	}
	if !replacementCreated {
		t.Fatal("expected replacement assignment to be created")
	}
	if scheduler.calls != 1 {
		t.Fatalf("ScheduleWorkload() calls = %d, want 1", scheduler.calls)
	}
}

func TestControllerReconcileDoesNotFailAssignmentOnUnexpectedInspectionError(t *testing.T) {
	ctx := context.Background()
	replicas := int32(1)
	inspectErr := errors.New("unexpected error")

	assignments := []model.Assignment{
		{ID: "old", Workload: model.WorkloadRef{Name: "nginx", Namespace: "default"}, Status: model.AssignmentStatusRunning, ContainerID: "container-old", NodeID: "node-1"},
	}
	observer := &fakeObserver{
		err: inspectErr,
	}

	controller, ref, assignmentStore := newControllerFixture(t, replicas, assignments, &fakeScheduler{}, &fakeAssignmentExecutor{}, observer, time.Second)
	err := controller.Reconcile(ctx, ref)
	if !errors.Is(err, inspectErr) {
		t.Fatalf("Reconcile() error = %v want %v", err, inspectErr)
	}

	got, err := assignmentStore.ListAssignmentsForWorkload(ctx, ref)
	if err != nil {
		t.Fatalf("ListAssignmentsForWorkload() error = %v", err)
	}

	if got[0].Status != model.AssignmentStatusRunning {
		t.Fatalf("expected assignment status to remain running, got %s", got[0].Status)
	}
}

func TestControllerReconcileContinuesAfterUnavailableNode(t *testing.T) {
	ctx := context.Background()
	replicas := int32(2)
	assignments := []model.Assignment{
		{ID: "old-1", Workload: model.WorkloadRef{Name: "nginx", Namespace: "default"}, Status: model.AssignmentStatusRunning, ContainerID: "container-old-1", NodeID: "node-unavailable"},
		{ID: "old-2", Workload: model.WorkloadRef{Name: "nginx", Namespace: "default"}, Status: model.AssignmentStatusRunning, ContainerID: "container-old-2", NodeID: "node-available"},
	}

	scheduler := &fakeScheduler{}
	executor := &fakeAssignmentExecutor{}
	observer := &fakeObserver{
		inspect: func(_ context.Context, assignment model.Assignment) (runtime.ContainerInfo, error) {
			if assignment.NodeID == "node-unavailable" {
				return runtime.ContainerInfo{}, ErrNodeUnavailable
			}
			return runtime.ContainerInfo{Running: true}, nil
		},
	}

	c, ref, assignmentStore := newControllerFixture(t, replicas, assignments, scheduler, executor, observer, time.Second)
	scheduler.store = assignmentStore
	if err := c.Reconcile(ctx, ref); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	got, err := assignmentStore.ListAssignmentsForWorkload(ctx, ref)
	if err != nil {
		t.Fatalf("ListAssignmentsForWorkload() error = %v", err)
	}

	var oldFailed, replacementCreated bool
	for _, a := range got {
		switch a.ID {
		case "old-1":
			oldFailed = a.Status == model.AssignmentStatusFailed
		case "assignment-1":
			replacementCreated = a.Status == model.AssignmentStatusPending
		case "old-2":
			if a.Status != model.AssignmentStatusRunning {
				t.Fatalf("expected old-2 assignment to remain running, got %s", a.Status)
			}
		}
	}

	if !oldFailed {
		t.Fatal("expected old-1 assignment to be failed")
	}
	if !replacementCreated {
		t.Fatal("expected replacement assignment to be created")
	}
	if observer.calls != 2 {
		t.Fatalf("Inspect() calls = %d, want 2", observer.calls)
	}
	if scheduler.calls != 1 {
		t.Fatalf("ScheduleWorkload() calls = %d, want 1", scheduler.calls)
	}
	if executor.calls != 1 {
		t.Fatalf("ExecuteAssignment() calls = %d, want 1", executor.calls)
	}
}

func TestControllerReconcileDoesNotDuplicatePendingOrRunning(t *testing.T) {
	for _, status := range []model.AssignmentStatus{model.AssignmentStatusPending, model.AssignmentStatusRunning} {
		t.Run(string(status), func(t *testing.T) {
			scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
			assignment := model.Assignment{ID: "a", Workload: model.WorkloadRef{Name: "nginx", Namespace: "default"}, Status: status}
			if status == model.AssignmentStatusRunning {
				assignment.ContainerID = "container"
			}
			c, ref, _ := newControllerFixture(t, 1, []model.Assignment{assignment}, scheduler, executor, &fakeObserver{info: runtime.ContainerInfo{Running: true}}, time.Second)
			if err := c.Reconcile(context.Background(), ref); err != nil {
				t.Fatal(err)
			}
			if scheduler.calls != 0 || executor.calls != 0 {
				t.Fatalf("unexpected work: schedule=%d execute=%d", scheduler.calls, executor.calls)
			}
		})
	}
}

func TestControllerReconcileDeadContainerFailsAndReplaces(t *testing.T) {
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
	assignment := model.Assignment{ID: "old", Workload: model.WorkloadRef{Name: "nginx", Namespace: "default"}, Status: model.AssignmentStatusRunning, ContainerID: "dead"}
	c, ref, assignments := newControllerFixture(t, 1, []model.Assignment{assignment}, scheduler, executor, &fakeObserver{info: runtime.ContainerInfo{Running: false}}, time.Second)
	if err := c.Reconcile(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	updated, err := assignments.GetAssignment(context.Background(), "old")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != model.AssignmentStatusFailed {
		t.Fatalf("got %s", updated.Status)
	}
	if scheduler.calls != 1 || executor.calls != 1 {
		t.Fatalf("schedule=%d execute=%d", scheduler.calls, executor.calls)
	}
}

func TestControllerReconcileFailedReplicaIsReplaced(t *testing.T) {
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
	failed := model.Assignment{ID: "failed", Workload: model.WorkloadRef{Name: "nginx", Namespace: "default"}, Status: model.AssignmentStatusFailed}
	c, ref, _ := newControllerFixture(t, 1, []model.Assignment{failed}, scheduler, executor, &fakeObserver{}, time.Second)
	if err := c.Reconcile(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if scheduler.calls != 1 || executor.calls != 1 {
		t.Fatalf("schedule=%d execute=%d", scheduler.calls, executor.calls)
	}
}

func TestControllerReconcileIsIdempotent(t *testing.T) {
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
	c, ref, assignments := newControllerFixture(t, 1, nil, scheduler, executor, &fakeObserver{}, time.Second)
	scheduler.store = assignments
	if err := c.Reconcile(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if err := c.Reconcile(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if scheduler.calls != 1 || executor.calls != 1 {
		t.Fatalf("schedule=%d execute=%d, want one execution", scheduler.calls, executor.calls)
	}
}

func TestControllerReconcileObsoleteAssignments(t *testing.T) {
	ctx := context.Background()
	workload := testutil.Workload("nginx", "default")
	deleteErr := errors.New("delete failed")

	tests := []struct {
		name      string
		setup     func() ([]model.Assignment, []string)
		deleteErr error
		wantErr   error
	}{
		{name: "no obsolete assignments", setup: func() ([]model.Assignment, []string) {
			assignment := testutil.AssignmentForWorkload(workload, "node-1", "container-1", model.AssignmentStatusRunning)
			return []model.Assignment{assignment}, nil
		}, wantErr: nil},
		{name: "one obsolete assignment", setup: func() ([]model.Assignment, []string) {
			assignemt := testutil.ObsoleteAssignment(workload, "node-1", "container-1", model.AssignmentStatusRunning)
			return []model.Assignment{assignemt}, []string{assignemt.ID}
		}, wantErr: nil},
		{name: "multiple obsolete assignments", setup: func() ([]model.Assignment, []string) {
			assignemt1 := testutil.ObsoleteAssignment(workload, "node-1", "container-1", model.AssignmentStatusRunning)
			assignemt2 := testutil.ObsoleteAssignment(workload, "node-2", "container-2", model.AssignmentStatusRunning)
			return []model.Assignment{assignemt1, assignemt2}, []string{assignemt1.ID, assignemt2.ID}
		}, wantErr: nil},
		{name: "non-running assignments are ignored", setup: func() ([]model.Assignment, []string) {
			assignemt := testutil.ObsoleteAssignment(workload, "node-1", "container-1", model.AssignmentStatusPending)
			return []model.Assignment{assignemt}, nil
		}, deleteErr: nil, wantErr: nil},
		{name: "delete error is propagated", setup: func() ([]model.Assignment, []string) {
			assignemt := testutil.ObsoleteAssignment(workload, "node-1", "container-1", model.AssignmentStatusRunning)
			return []model.Assignment{assignemt}, []string{assignemt.ID}
		}, deleteErr: deleteErr, wantErr: deleteErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignments, wantDeleted := tt.setup()
			executor := &fakeAssignmentExecutor{
				err: tt.deleteErr,
			}
			c, _, _ := newControllerFixture(t, 1, assignments, &fakeScheduler{}, executor, &fakeObserver{}, time.Second)

			err := c.reconcileObsoleteAssignments(ctx, workload, assignments)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("reconcileObsoleteAssignments() error = %v, want %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(executor.deletedAssignments, func() []model.Assignment {
				var deleted []model.Assignment
				for _, id := range wantDeleted {
					deleted = append(deleted, model.Assignment{ID: id})
				}
				return deleted
			}()) {
				t.Fatalf("deleted assignments = %v, want %v", executor.deletedAssignments, wantDeleted)
			}
		})
	}
}

func TestControllerReconcileAll(t *testing.T) {
	ctx := context.Background()
	workloads, assignments := store.NewMemoryStore(), store.NewMemoryAssignmentStore()
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
	for _, workload := range []model.Workload{
		{Metadata: model.ObjectMeta{Name: "nginx", Namespace: "default"}, Spec: model.WorkloadSpec{Replicas: 1}},
		{Metadata: model.ObjectMeta{Name: "redis", Namespace: "default"}, Spec: model.WorkloadSpec{Replicas: 2}},
		{Metadata: model.ObjectMeta{Name: "other", Namespace: "other"}, Spec: model.WorkloadSpec{Replicas: 3}},
	} {
		if err := workloads.Create(ctx, workload); err != nil {
			t.Fatal(err)
		}
	}
	c := NewController(workloads, assignments, scheduler, executor, &fakeObserver{}, "default", time.Second)
	if err := c.ReconcileAll(ctx); err != nil {
		t.Fatal(err)
	}
	if scheduler.calls != 3 || executor.calls != 3 {
		t.Fatalf("schedule=%d execute=%d, want 3 each", scheduler.calls, executor.calls)
	}
	scheduled := map[string]int{}
	for _, ref := range scheduler.refs {
		scheduled[ref.Namespace+"/"+ref.Name]++
	}
	if scheduled["default/nginx"] != 1 || scheduled["default/redis"] != 2 || scheduled["other/other"] != 0 {
		t.Fatalf("unexpected per-workload scheduling: %#v", scheduled)
	}
}

func TestControllerStartImmediateReconcile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := make(chan string, 1)
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{called: called}
	c, _, _ := newControllerFixture(t, 1, nil, scheduler, executor, &fakeObserver{}, time.Hour)
	done := make(chan error, 1)
	go func() { done <- c.Start(ctx) }()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("controller did not reconcile immediately")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestControllerStartPeriodicReconcile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := make(chan string, 2)
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{called: called}
	c, _, _ := newControllerFixture(t, 1, nil, scheduler, executor, &fakeObserver{}, 5*time.Millisecond)
	done := make(chan error, 1)
	go func() { done <- c.Start(ctx) }()
	for i := 0; i < 2; i++ {
		select {
		case <-called:
		case <-time.After(time.Second):
			t.Fatalf("controller completed only %d reconciliation cycles", i)
		}
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestControllerStartReturnsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
	c, _, _ := newControllerFixture(t, 0, nil, scheduler, executor, &fakeObserver{}, time.Second)
	done := make(chan error, 1)
	go func() { done <- c.Start(ctx) }()
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestControllerReconcileScaleDown(t *testing.T) {
	tests := []struct {
		name            string
		desired         int
		running         int
		pending         int
		expectedDeletes int
	}{
		{name: "no scale down needed", desired: 3, running: 3, expectedDeletes: 0},
		{name: "one extra running", desired: 2, running: 3, expectedDeletes: 1},
		{name: "pending does not count for deletion", desired: 2, running: 2, pending: 1, expectedDeletes: 0},
		{name: "all extra", desired: 1, running: 2, pending: 2, expectedDeletes: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
			workload := testutil.WorkloadWithReplicas("nginx", "default", int32(tt.desired))
			ref := testutil.WorkloadRef(workload.Metadata.Name, workload.Metadata.Namespace)

			assignments := make([]model.Assignment, 0, tt.running+tt.pending)
			for _, item := range []struct {
				status model.AssignmentStatus
				count  int
			}{
				{model.AssignmentStatusRunning, tt.running},
				{model.AssignmentStatusPending, tt.pending},
			} {
				for i := 0; i < item.count; i++ {
					assignment := model.Assignment{
						ID:       fmt.Sprintf("%s-%d", item.status, i),
						Workload: ref,
						Status:   item.status,
					}
					if item.status == model.AssignmentStatusRunning {
						assignment.ContainerID = fmt.Sprintf("container-%d", i)
						assignment.TemplateHash = model.TemplateHash(workload.Spec.Template)
					}
					assignments = append(assignments, assignment)
				}
			}

			c, _, _ := newControllerFixture(t, int32(tt.desired), assignments, scheduler, executor, &fakeObserver{info: runtime.ContainerInfo{Running: true}}, time.Second)
			if err := c.Reconcile(context.Background(), ref); err != nil {
				t.Fatal(err)
			}
			if executor.deleteCalls != tt.expectedDeletes {
				t.Fatalf("executor called %d times, want %d", executor.deleteCalls, tt.expectedDeletes)
			}

			if scheduler.calls != 0 {
				t.Fatalf("scheduler called %d times, want 0", scheduler.calls)
			}
		})
	}
}

func TestControllerReconcileScaleDownDeletesRunningAssignments(t *testing.T) {
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{}
	workload := testutil.WorkloadWithReplicas("nginx", "default", 1)
	ref := testutil.WorkloadRef(workload.Metadata.Name, workload.Metadata.Namespace)
	templateHash := model.TemplateHash(workload.Spec.Template)
	assignments := []model.Assignment{
		{ID: "running-1", Workload: ref, Status: model.AssignmentStatusRunning, ContainerID: "container-1", TemplateHash: templateHash},
		{ID: "pending-1", Workload: ref, Status: model.AssignmentStatusPending, ContainerID: "container-3", TemplateHash: templateHash},
		{ID: "running-2", Workload: ref, Status: model.AssignmentStatusRunning, ContainerID: "container-2", TemplateHash: templateHash},
	}
	c, _, _ := newControllerFixture(t, 1, assignments, scheduler, executor, &fakeObserver{info: runtime.ContainerInfo{Running: true}}, time.Second)
	if err := c.Reconcile(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if executor.deleteCalls != 1 {
		t.Fatalf("executor called %d times, want 1", executor.deleteCalls)
	}
	if scheduler.calls != 0 {
		t.Fatalf("scheduler called %d times, want 0", scheduler.calls)
	}
}

func TestControllerReconcileScaleDownDeleteError(t *testing.T) {
	scheduler, executor := &fakeScheduler{}, &fakeAssignmentExecutor{err: errors.New("delete failed")}
	ref := model.WorkloadRef{Name: "nginx", Namespace: "default"}
	assignments := []model.Assignment{
		{ID: "running-1", Workload: ref, Status: model.AssignmentStatusRunning, ContainerID: "container-1"},
		{ID: "running-2", Workload: ref, Status: model.AssignmentStatusRunning, ContainerID: "container-2"},
	}
	c, _, _ := newControllerFixture(t, 1, assignments, scheduler, executor, &fakeObserver{info: runtime.ContainerInfo{Running: true}}, time.Second)
	err := c.Reconcile(context.Background(), ref)
	if err == nil || err.Error() != "delete failed" {
		t.Fatalf("got %v, want delete failed error", err)
	}
	if executor.deleteCalls != 1 {
		t.Fatalf("executor called %d times, want 1", executor.deleteCalls)
	}
	if scheduler.calls != 0 {
		t.Fatalf("scheduler called %d times, want 0", scheduler.calls)
	}
}
