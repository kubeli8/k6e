package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
	"github.com/pyd-07/k6e/internal/store"
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
	calls       int
	deleteCalls int
	err         error
	called      chan<- string
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
	return e.err
}

type fakeObserver struct {
	calls int
	info  runtime.ContainerInfo
	err   error
}

func (o *fakeObserver) Inspect(_ context.Context, _ model.Assignment) (runtime.ContainerInfo, error) {
	o.calls++
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
	if err := workloads.Create(ctx, model.Workload{Metadata: model.ObjectMeta{Name: ref.Name, Namespace: ref.Namespace}, Spec: model.WorkloadSpec{Replicas: replicas}}); err != nil {
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
			ref := model.WorkloadRef{Name: "nginx", Namespace: "default"}
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
	ref := model.WorkloadRef{Name: "nginx", Namespace: "default"}
	assignments := []model.Assignment{
		{ID: "running-1", Workload: ref, Status: model.AssignmentStatusRunning, ContainerID: "container-1"},
		{ID: "pending-1", Workload: ref, Status: model.AssignmentStatusPending},
		{ID: "running-2", Workload: ref, Status: model.AssignmentStatusRunning, ContainerID: "container-2"},
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
