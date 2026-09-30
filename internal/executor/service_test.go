package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

type fakeAssignmentStore struct {
	assignment       model.Assignment
	updateCalled     bool
	updatedID        string
	updatedContainer runtime.ContainerID
	updatedStatus    model.AssignmentStatus
	getErr           error
	updateErr        error
}

func testAssignment(name, namespace, nodeID, containerID string, status model.AssignmentStatus) model.Assignment {
	return model.Assignment{
		ID: uuid.NewString(),
		Workload: model.WorkloadRef{
			Name:      name,
			Namespace: namespace,
		},
		NodeID:      nodeID,
		Status:      status,
		ContainerID: containerID,
	}
}

func testWorkload(name, namespace string) model.Workload {
	return model.Workload{
		APIVersion: "k6e.io/v1alpha1",
		Kind:       "Workload",
		Metadata: model.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: model.WorkloadSpec{
			Replicas: 2,
			Template: model.PodTemplateSpec{
				Containers: []model.ContainerSpec{
					{
						Name:  "nginx",
						Image: "nginx:latest",
					},
				},
			},
		},
	}
}

func testNode(id, address string, status model.NodeStatus) model.Node {
	return model.Node{
		ID:            id,
		Address:       address,
		Status:        status,
		LastHeartbeat: time.Now(),
	}
}

func (f *fakeAssignmentStore) CreateAssignment(ctx context.Context, assignment model.Assignment) error {
	return nil
}

func (f *fakeAssignmentStore) GetAssignment(ctx context.Context, id string) (model.Assignment, error) {
	if f.getErr != nil {
		return model.Assignment{}, f.getErr
	}

	return f.assignment, nil
}

func (f *fakeAssignmentStore) ListAssignments(ctx context.Context, namespace string) ([]model.Assignment, error) {
	return nil, nil
}

func (f *fakeAssignmentStore) ListAssignmentsForWorkload(ctx context.Context, ref model.WorkloadRef) ([]model.Assignment, error) {
	return nil, nil
}

func (f *fakeAssignmentStore) UpdateStatusAssignment(ctx context.Context, id string, status model.AssignmentStatus) error {
	return nil
}

func (f *fakeAssignmentStore) UpdateAssignmentExecution(ctx context.Context, id string, containerID runtime.ContainerID, status model.AssignmentStatus) error {
	f.updateCalled = true
	f.updatedID = id
	f.updatedContainer = containerID
	f.updatedStatus = status

	return f.updateErr
}

func (f *fakeAssignmentStore) DeleteAssignment(ctx context.Context, id string) error {
	return nil
}

type fakeWorkloadStore struct {
	workload model.Workload
	getErr   error
}

func (f *fakeWorkloadStore) Create(ctx context.Context, workload model.Workload) error {
	return f.getErr
}

func (f *fakeWorkloadStore) Get(ctx context.Context, ref model.WorkloadRef) (model.Workload, error) {
	if f.getErr != nil {
		return model.Workload{}, f.getErr
	}

	return f.workload, nil
}

func (f *fakeWorkloadStore) List(ctx context.Context, namespace string) ([]model.Workload, error) {
	return nil, f.getErr
}

func (f *fakeWorkloadStore) Update(ctx context.Context, ref model.WorkloadRef, workload model.Workload) error {
	return f.getErr
}

func (f *fakeWorkloadStore) Delete(ctx context.Context, ref model.WorkloadRef) error {
	return f.getErr
}

type fakeNodeStore struct {
	node   model.Node
	getErr error
}

func (f *fakeNodeStore) RegisterNode(ctx context.Context, node model.Node) error {
	return nil
}

func (f *fakeNodeStore) GetNode(ctx context.Context, id string) (model.Node, error) {
	if f.getErr != nil {
		return model.Node{}, f.getErr
	}

	return f.node, nil
}

func (f *fakeNodeStore) ListNodes(ctx context.Context) ([]model.Node, error) {
	return nil, nil
}

func (f *fakeNodeStore) RemoveNode(ctx context.Context, id string) error {
	return nil
}

func (f *fakeNodeStore) UpdateHeartbeat(ctx context.Context, id string, time time.Time) error {
	return nil
}

func (f *fakeNodeStore) UpdateNodeStatus(ctx context.Context, id string, status model.NodeStatus) error {
	return nil
}

type fakeExecutor struct {
	containerID runtime.ContainerID
	err         error

	called bool
	node   model.Node
	spec   runtime.ContainerSpec
}

func (f *fakeExecutor) RunContainer(ctx context.Context, node model.Node, spec runtime.ContainerSpec) (runtime.ContainerID, error) {
	f.called = true
	f.node = node
	f.spec = spec

	if f.err != nil {
		return "", f.err
	}

	return f.containerID, nil
}

func (f *fakeExecutor) DeleteContainer(ctx context.Context, node model.Node, containerID runtime.ContainerID) error {
	f.called = true
	f.node = node
	f.containerID = containerID
	return f.err
}

func TestExecuteAssignment(t *testing.T) {
	ctx := context.Background()

	node := testNode("node-1", "127.0.0.1", model.NodeStatusReady)
	assignment := testAssignment("nginx", "default", node.ID, "", model.AssignmentStatusPending)
	workload := testWorkload("nginx", "default")

	assignmentStore := &fakeAssignmentStore{assignment: assignment}
	workloadStore := &fakeWorkloadStore{workload: workload}
	nodeStore := &fakeNodeStore{node: node}
	executor := &fakeExecutor{containerID: "container-123"}

	service := NewService(assignmentStore, workloadStore, nodeStore, executor)
	result, err := service.ExecuteAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("ExecuteAssignment returned error: %v", err)
	}

	if result.ContainerID != "container-123" {
		t.Fatalf("ExecuteAssignment returned unexpected container ID: %s", result)
	}

	if !executor.called {
		t.Fatal("expected executor to be called")
	}

	if executor.node.ID != node.ID {
		t.Fatalf("executor called with unexpected node ID: %s", executor.node.ID)
	}

	if executor.spec.Image != workload.Spec.Template.Containers[0].Image {
		t.Fatalf("executor called with unexpected container image: %s", executor.spec.Image)
	}

	if assignmentStore.updatedContainer != "container-123" {
		t.Fatalf("assignment store updated with unexpected container ID: %s", assignmentStore.updatedContainer)
	}

	if assignmentStore.updatedStatus != model.AssignmentStatusRunning {
		t.Fatalf("assignment store updated with unexpected status: %s", assignmentStore.updatedStatus)
	}
}

func TestExecuteAssignmentAssignmentNotFound(t *testing.T) {
	expectedErr := errors.New("assignment not found")

	assignmentStore := &fakeAssignmentStore{getErr: expectedErr}

	service := NewService(assignmentStore, &fakeWorkloadStore{}, &fakeNodeStore{}, &fakeExecutor{})

	_, err := service.ExecuteAssignment(context.Background(), "missing")

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected error %v, got %v",
			expectedErr,
			err,
		)
	}
}

func TestExecuteAssignmentWorkloadNotFound(t *testing.T) {
	expectedErr := errors.New("workload not found")

	assignmentStore := &fakeAssignmentStore{
		assignment: testAssignment("nginx", "default", "node-1", "", model.AssignmentStatusPending),
	}

	workloadStore := &fakeWorkloadStore{getErr: expectedErr}

	service := NewService(assignmentStore, workloadStore, &fakeNodeStore{}, &fakeExecutor{})

	_, err := service.ExecuteAssignment(context.Background(), "assignment-1")

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected error %v, got %v",
			expectedErr,
			err,
		)
	}
}

func TestExecuteAssignmentNodeNotFound(t *testing.T) {
	expectedErr := errors.New("node not found")

	assignmentStore := &fakeAssignmentStore{
		assignment: testAssignment("nginx", "default", "node-1", "", model.AssignmentStatusPending),
	}

	workloadStore := &fakeWorkloadStore{
		workload: testWorkload("nginx", "default"),
	}

	nodeStore := &fakeNodeStore{getErr: expectedErr}

	service := NewService(assignmentStore, workloadStore, nodeStore, &fakeExecutor{})

	_, err := service.ExecuteAssignment(context.Background(), "assignment-1")

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected error %v, got %v",
			expectedErr,
			err,
		)
	}
}

func TestExecuteAssignmentExecutorFailure(t *testing.T) {
	executionErr := errors.New("failed to execute container")
	workload := testWorkload("nginx", "default")

	node := testNode("node-1", "localhost:8081", model.NodeStatusReady)
	assignment := testAssignment("nginx", "default", node.ID, "", model.AssignmentStatusPending)

	assignmentStore := &fakeAssignmentStore{assignment: assignment}
	workloadStore := &fakeWorkloadStore{workload: workload}
	nodeStore := &fakeNodeStore{node: node}
	executor := &fakeExecutor{err: executionErr}
	service := NewService(assignmentStore, workloadStore, nodeStore, executor)

	result, err := service.ExecuteAssignment(context.Background(), assignment.ID)

	if !errors.Is(err, executionErr) {
		t.Fatalf(
			"expected error %v, got %v",
			executionErr,
			err,
		)
	}

	if result.Status != model.AssignmentStatusFailed {
		t.Fatalf(
			"expected status %q, got %q",
			model.AssignmentStatusFailed,
			result.Status,
		)
	}

	if !assignmentStore.updateCalled {
		t.Fatal("expected assignment execution update to be called")
	}

	if assignmentStore.updatedStatus != model.AssignmentStatusFailed {
		t.Fatalf(
			"expected persisted status %q, got %q",
			model.AssignmentStatusFailed,
			assignmentStore.updatedStatus,
		)
	}
}

func TestDeleteAssignment(t *testing.T) {
	ctx := context.Background()
	workload := testWorkload("nginx", "default")
	node := testNode("node-1", "localhost:8081", model.NodeStatusReady)
	assignment := testAssignment("nginx", "default", node.ID, "container-123", model.AssignmentStatusRunning)

	assignmentStore := &fakeAssignmentStore{assignment: assignment}
	workloadStore := &fakeWorkloadStore{workload: workload}
	nodeStore := &fakeNodeStore{node: node}
	executor := &fakeExecutor{}
	service := NewService(assignmentStore, workloadStore, nodeStore, executor)

	err := service.DeleteAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("DeleteAssignment returned error: %v", err)
	}

	if !executor.called {
		t.Fatal("expected executor to be called")
	}

	if executor.node.ID != node.ID {
		t.Fatalf("executor called with unexpected node ID: %s", executor.node.ID)
	}

	if executor.containerID != "container-123" {
		t.Fatalf("executor called with unexpected container ID: %s", executor.containerID)
	}
}

func TestDeleteAssignmentNoContainerID(t *testing.T) {
	ctx := context.Background()
	assignment := testAssignment("nginx", "default", "node-1", "", model.AssignmentStatusRunning)

	assignmentStore := &fakeAssignmentStore{assignment: assignment}
	service := NewService(assignmentStore, &fakeWorkloadStore{}, &fakeNodeStore{}, &fakeExecutor{})

	err := service.DeleteAssignment(ctx, assignment.ID)
	if err == nil {
		t.Fatal("expected error when deleting assignment with no container ID")
	}
}

func TestDeleteAssignmentDeleteContainerError(t *testing.T) {
	ctx := context.Background()
	workload := testWorkload("nginx", "default")
	node := testNode("node-1", "localhost:8081", model.NodeStatusReady)
	assignment := testAssignment("nginx", "default", node.ID, "container-123", model.AssignmentStatusRunning)

	assignmentStore := &fakeAssignmentStore{assignment: assignment}
	workloadStore := &fakeWorkloadStore{workload: workload}
	nodeStore := &fakeNodeStore{node: node}
	executorErr := errors.New("failed to delete container")
	executor := &fakeExecutor{err: executorErr}
	service := NewService(assignmentStore, workloadStore, nodeStore, executor)

	err := service.DeleteAssignment(ctx, assignment.ID)
	if !errors.Is(err, executorErr) {
		t.Fatalf(
			"expected error %v, got %v",
			executorErr,
			err,
		)
	}
}
