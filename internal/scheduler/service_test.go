package scheduler

import (
	"context"
	"errors"
	"testing"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
	"github.com/pyd-07/k6e/internal/store"
)

type failingAssignmentStore struct {
	err error
}

func (f *failingAssignmentStore) CreateAssignment(ctx context.Context, assignment model.Assignment) error {
	return f.err
}

func (f *failingAssignmentStore) GetAssignment(ctx context.Context, id string) (model.Assignment, error) {
	return model.Assignment{}, f.err
}

func (f *failingAssignmentStore) ListAssignments(ctx context.Context, namespace string) ([]model.Assignment, error) {
	return nil, f.err
}

func (f *failingAssignmentStore) ListAssignmentsForWorkload(ctx context.Context, ref model.WorkloadRef) ([]model.Assignment, error) {
	return nil, f.err
}

func (f *failingAssignmentStore) UpdateStatusAssignment(ctx context.Context, id string, status model.AssignmentStatus) error {
	return f.err
}

func (f *failingAssignmentStore) UpdateAssignmentExecution(ctx context.Context, id string, containerID runtime.ContainerID, status model.AssignmentStatus) error {
	return f.err
}

func (f *failingAssignmentStore) DeleteAssignment(ctx context.Context, id string) error {
	return f.err
}

func TestServiceScheduleWorkload(t *testing.T) {
	ctx := context.Background()

	workloadStore := store.NewMemoryStore()
	nodeStore := store.NewMemoryNodeStore()
	assignmentStore := store.NewMemoryAssignmentStore()

	scheduler := &SimpleScheduler{}

	service := NewService(workloadStore, nodeStore, assignmentStore, scheduler)
	workload := testWorkload("test-workload", "default")

	if err := workloadStore.Create(ctx, workload); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	nodeID := "node-1"
	nodeAddr := "localhost:8081"

	if err := nodeStore.RegisterNode(ctx, model.Node{ID: nodeID, Address: nodeAddr, Status: model.NodeStatusReady}); err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}

	ref := model.WorkloadRef{Namespace: workload.Metadata.Namespace, Name: workload.Metadata.Name}

	assignment, err := service.ScheduleWorkload(ctx, ref)
	if err != nil {
		t.Fatalf("ScheduleWorkload() error = %v", err)
	}

	if assignment.Workload != ref {
		t.Errorf("ScheduleWorkload() got workload ref = %v, want = %v", assignment.Workload, ref)
	}

	if assignment.NodeID != nodeID {
		t.Errorf("ScheduleWorkload() got node ID = %v, want = %v", assignment.NodeID, nodeID)
	}

	if assignment.Status != model.AssignmentStatusPending {
		t.Errorf("ScheduleWorkload() got status = %v, want = %v", assignment.Status, model.AssignmentStatusPending)
	}

	if assignment.ID == "" {
		t.Errorf("ScheduleWorkload() got empty assignment ID")
	}

	stored, err := assignmentStore.GetAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("GetAssignment() error = %v", err)
	}

	if stored.ID != assignment.ID {
		t.Errorf("GetAssignment() got ID = %v, want = %v", stored.ID, assignment.ID)
	}

}

func TestServiceScheduleWorkload_NoReadyNodes(t *testing.T) {
	ctx := context.Background()

	workloadStore := store.NewMemoryStore()
	nodeStore := store.NewMemoryNodeStore()
	assignmentStore := store.NewMemoryAssignmentStore()

	scheduler := &SimpleScheduler{}

	service := NewService(workloadStore, nodeStore, assignmentStore, scheduler)
	workload := testWorkload("test-workload", "default")

	if err := workloadStore.Create(ctx, workload); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	nodeID := "node-1"
	nodeAddr := "localhost:8081"

	if err := nodeStore.RegisterNode(ctx, model.Node{ID: nodeID, Address: nodeAddr, Status: model.NodeStatusNotReady}); err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}

	ref := model.WorkloadRef{Namespace: workload.Metadata.Namespace, Name: workload.Metadata.Name}

	_, err := service.ScheduleWorkload(ctx, ref)
	if err == nil {
		t.Fatalf("ScheduleWorkload() error = nil, want = error")
	}
}

func TestServiceScheduleWorkload_WorkloadNotFound(t *testing.T) {
	ctx := context.Background()

	workloadStore := store.NewMemoryStore()
	nodeStore := store.NewMemoryNodeStore()
	assignmentStore := store.NewMemoryAssignmentStore()

	scheduler := &SimpleScheduler{}

	service := NewService(workloadStore, nodeStore, assignmentStore, scheduler)

	ref := model.WorkloadRef{Namespace: "default", Name: "non-existent-workload"}

	_, err := service.ScheduleWorkload(ctx, ref)
	if err == nil {
		t.Fatalf("ScheduleWorkload() error = nil, want = error")
	}

	if err != store.ErrNotFound {
		t.Fatalf("ScheduleWorkload() error = %v, want = %v", err, store.ErrNotFound)
	}
}

func TestServiceScheduleWorkload_AssignmentStoreError(t *testing.T) {
	ctx := context.Background()

	workloadStore := store.NewMemoryStore()
	nodeStore := store.NewMemoryNodeStore()

	expectedErr := errors.New("assignment store error")
	assignmentStore := &failingAssignmentStore{err: expectedErr}

	scheduler := &SimpleScheduler{}

	service := NewService(workloadStore, nodeStore, assignmentStore, scheduler)
	workload := testWorkload("test-workload", "default")

	if err := workloadStore.Create(ctx, workload); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	nodeID := "node-1"
	nodeAddr := "localhost:8081"
	if err := nodeStore.RegisterNode(ctx, model.Node{ID: nodeID, Address: nodeAddr, Status: model.NodeStatusReady}); err != nil {
		t.Fatalf("RegisterNode() error = %v", err)
	}

	_, err := service.ScheduleWorkload(ctx, model.WorkloadRef{Namespace: workload.Metadata.Namespace, Name: workload.Metadata.Name})
	if err == nil {
		t.Fatalf("ScheduleWorkload() error = nil, want = error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf("ScheduleWorkload() error = %v, want = %v", err, expectedErr)
	}
}
