package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
	"github.com/pyd-07/k6e/internal/store"
)

type fakeScheduler struct {
	calls       int
	assignments []model.Assignment
	err         error
}

func (s *fakeScheduler) Schedule(ctx context.Context, ref model.WorkloadRef) (model.Assignment, error) {
	s.calls++
	if s.err != nil {
		return model.Assignment{}, s.err
	}
	assignment := model.Assignment{
		ID:       uuid.NewString(),
		Workload: ref,
		Status:   model.AssignmentStatusPending,
	}
	s.assignments = append(s.assignments, assignment)
	return assignment, nil
}

type fakeObserver struct {
	calls int
	info  runtime.ContainerInfo
	err   error
}

func (o *fakeObserver) Inspect(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error) {
	o.calls++
	if o.err != nil {
		return runtime.ContainerInfo{}, o.err
	}
	return o.info, nil
}

func TestControllerReconcile(t *testing.T) {
	tests := []struct {
		name            string
		desired         int32
		running         int
		pending         int
		failed          int
		expectedCreates int
	}{
		{
			name:            "already converged",
			desired:         3,
			running:         3,
			pending:         0,
			failed:          0,
			expectedCreates: 0,
		},
		{
			name:            "one replica missing",
			desired:         3,
			running:         2,
			pending:         0,
			failed:          0,
			expectedCreates: 1,
		},
		{
			name:            "pending replica counts as in flight",
			desired:         3,
			running:         2,
			pending:         1,
			failed:          0,
			expectedCreates: 0,
		},
		{
			name:            "all replicas missing",
			desired:         3,
			running:         0,
			pending:         0,
			failed:          0,
			expectedCreates: 3,
		},
		{
			name:            "failed replicas do not count",
			desired:         3,
			running:         2,
			pending:         0,
			failed:          1,
			expectedCreates: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			workloadStore := store.NewMemoryStore()
			assignmentStore := store.NewMemoryAssignmentStore()
			scheduler := &fakeScheduler{}
			observer := &fakeObserver{}

			replicas := tt.desired

			workload := model.Workload{
				APIVersion: "k6e/v1",
				Kind:       "Workload",
				Metadata: model.ObjectMeta{
					Name:      "nginx",
					Namespace: "default",
				},
				Spec: model.WorkloadSpec{
					Replicas: replicas,
				},
			}

			err := workloadStore.Create(ctx, workload)
			if err != nil {
				t.Fatalf("failed to create workload: %v", err)
			}

			for i := 0; i < tt.running; i++ {
				err := assignmentStore.CreateAssignment(ctx, model.Assignment{
					ID: uuid.NewString(),
					Workload: model.WorkloadRef{
						Name:      "nginx",
						Namespace: "default",
					},
					Status: model.AssignmentStatusRunning,
				})
				if err != nil {
					t.Fatalf("failed to create running assignment: %v", err)
				}
			}

			for i := 0; i < tt.pending; i++ {
				err := assignmentStore.CreateAssignment(ctx, model.Assignment{
					ID: uuid.NewString(),
					Workload: model.WorkloadRef{
						Name:      "nginx",
						Namespace: "default",
					},
					Status: model.AssignmentStatusPending,
				})
				if err != nil {
					t.Fatalf("failed to create pending assignment: %v", err)
				}
			}

			for i := 0; i < tt.failed; i++ {
				err := assignmentStore.CreateAssignment(ctx, model.Assignment{
					ID: uuid.NewString(),
					Workload: model.WorkloadRef{
						Name:      "nginx",
						Namespace: "default",
					},
					Status: model.AssignmentStatusFailed,
				})
				if err != nil {
					t.Fatalf("failed to create failed assignment: %v", err)
				}
			}

			controller := NewController(
				workloadStore,
				assignmentStore,
				scheduler,
				observer,
			)

			ref := model.WorkloadRef{
				Name:      "nginx",
				Namespace: "default",
			}

			err = controller.Reconcile(ctx, ref)
			if err != nil {
				t.Fatalf("Reconcile() failed: %v", err)
			}

			if scheduler.calls != tt.expectedCreates {
				t.Fatalf(
					"expected scheduler to be called %d times, got %d",
					tt.expectedCreates,
					scheduler.calls,
				)
			}
		})
	}
}

func TestControllerReconcileSchedulerError(t *testing.T) {
	ctx := context.Background()

	workloadStore := store.NewMemoryStore()
	assignmentStore := store.NewMemoryAssignmentStore()

	replicas := int32(1)

	workload := model.Workload{
		Metadata: model.ObjectMeta{
			Name:      "nginx",
			Namespace: "default",
		},
		Spec: model.WorkloadSpec{
			Replicas: replicas,
		},
	}
	err := workloadStore.Create(ctx, workload)
	if err != nil {
		t.Fatalf("failed to create workload: %v", err)
	}

	schedulerErr := errors.New("no ready nodes")

	scheduler := &fakeScheduler{
		err: schedulerErr,
	}
	observer := &fakeObserver{}

	controller := NewController(
		workloadStore,
		assignmentStore,
		scheduler,
		observer,
	)

	ref := model.WorkloadRef{
		Name:      "nginx",
		Namespace: "default",
	}

	err = controller.Reconcile(ctx, ref)

	if !errors.Is(err, schedulerErr) {
		t.Fatalf("expected error %v, got %v", schedulerErr, err)
	}
}

func TestControllerObserveAssignment(t *testing.T) {
	ctx := context.Background()

	expected := runtime.ContainerInfo{
		ID:      "container-id",
		Running: true,
		State:   "running",
	}

	observer := &fakeObserver{
		info: expected,
	}
	controller := NewController(nil, nil, nil, observer)

	assignment := model.Assignment{
		ID:          "assignment-id",
		NodeID:      "node-1",
		ContainerID: "container-id",
		Status:      model.AssignmentStatusRunning,
	}

	actual, err := controller.ObserveAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("ObserveAssignment() failed: %v", err)
	}

	if actual.ID != expected.ID {
		t.Fatalf("expected %v, got %v", expected, actual)
	}

	if !actual.Running {
		t.Fatalf("expected container to be running")
	}

	if observer.calls != 1 {
		t.Fatalf("expected observer to be called once, got %d", observer.calls)
	}
}

func TestControllerObserverAssignmentError(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New("container inspection failed")
	observer := &fakeObserver{
		err: expectedErr,
	}
	controller := NewController(nil, nil, nil, observer)

	assignment := model.Assignment{
		ID:          "assignment-id",
		NodeID:      "node-1",
		ContainerID: "container-id",
		Status:      model.AssignmentStatusRunning,
	}

	_, err := controller.ObserveAssignment(ctx, assignment)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if observer.calls != 1 {
		t.Fatalf("expected observer to be called once, got %d", observer.calls)
	}
}

func TestControllerReconcileDeadContainer(t *testing.T) {
	ctx := context.Background()

	workloadStore := store.NewMemoryStore()
	assignmentStore := store.NewMemoryAssignmentStore()

	replicas := int32(1)

	workload := model.Workload{
		Metadata: model.ObjectMeta{
			Name:      "nginx",
			Namespace: "default",
		},
		Spec: model.WorkloadSpec{
			Replicas: replicas,
		},
	}

	if err := workloadStore.Create(ctx, workload); err != nil {
		t.Fatalf("failed to create workload: %v", err)
	}

	assignment := model.Assignment{
		ID: "assignment-1",
		Workload: model.WorkloadRef{
			Name:      "nginx",
			Namespace: "default",
		},
		NodeID:      "node-1",
		Status:      model.AssignmentStatusRunning,
		ContainerID: "container-1",
	}

	if err := assignmentStore.CreateAssignment(ctx, assignment); err != nil {
		t.Fatalf("failed to create assignment: %v", err)
	}

	observer := &fakeObserver{
		info: runtime.ContainerInfo{
			ID:      "container-1",
			State:   "exited",
			Running: false,
		},
	}

	scheduler := &fakeScheduler{}

	controller := NewController(
		workloadStore,
		assignmentStore,
		scheduler,
		observer,
	)

	ref := model.WorkloadRef{
		Name:      "nginx",
		Namespace: "default",
	}

	if err := controller.Reconcile(ctx, ref); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	updated, err := assignmentStore.GetAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("failed to get assignment: %v", err)
	}

	if updated.Status != model.AssignmentStatusFailed {
		t.Fatalf("expected assignment to be failed, got %v", updated.Status)
	}

	if scheduler.calls != 1 {
		t.Fatalf("expected scheduler to be called once, got %d", scheduler.calls)
	}

	if observer.calls != 1 {
		t.Fatalf("expected runtime observer to be called once, got %d", observer.calls)
	}
}
