package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/pyd-07/k6e/internal/model"
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

	controller := NewController(
		workloadStore,
		assignmentStore,
		scheduler,
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
