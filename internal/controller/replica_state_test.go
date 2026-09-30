package controller

import (
	"testing"

	"github.com/pyd-07/k6e/internal/model"
)

func TestCalculateReplicaState(t *testing.T) {
	replicas := int32(3)

	workload := model.Workload{
		Spec: model.WorkloadSpec{
			Replicas: replicas,
		},
	}

	assignments := []model.Assignment{
		{
			Status: model.AssignmentStatusRunning,
		},
		{
			Status: model.AssignmentStatusRunning,
		},
		{
			Status: model.AssignmentStatusPending,
		},
		{
			Status: model.AssignmentStatusFailed,
		},
	}

	state := CalculateReplicaState(workload, assignments)

	if state.Desired != 3 {
		t.Fatalf("expected desired=3, got %d", state.Desired)
	}

	if state.Assigned != 4 {
		t.Fatalf("expected assigned=4, got %d", state.Assigned)
	}

	if state.Running != 2 {
		t.Fatalf("expected running=2, got %d", state.Running)
	}

	if state.Pending != 1 {
		t.Fatalf("expected pending=1, got %d", state.Pending)
	}

	if state.Failed != 1 {
		t.Fatalf("expected failed=1, got %d", state.Failed)
	}

	if state.Missing() != 0 {
		t.Fatalf("expected missing=0, got %d", state.Missing())
	}
}

func TestReplicaStateMissing(t *testing.T) {
	state := ReplicaState{
		Desired: 3,
		Running: 2,
	}

	if state.Missing() != 1 {
		t.Fatalf("expected missing=1, got %d", state.Missing())
	}
}

func TestReplicaStateMissingIgnoresFailed(t *testing.T) {
	state := ReplicaState{
		Desired: 3,
		Running: 2,
		Failed:  1,
	}

	if state.Missing() != 1 {
		t.Fatalf("expected missing=1, got %d", state.Missing())
	}
}

func TestReplicaStatePendingCountsAsInFlight(t *testing.T) {
	state := ReplicaState{
		Desired: 3,
		Running: 2,
		Pending: 1,
	}

	if state.Missing() != 0 {
		t.Fatalf("expected missing=0, got %d", state.Missing())
	}
}

func TestReplicaStateDelete(t *testing.T) {
	state := ReplicaState{
		Desired:  2,
		Running:  4,
		Pending:  1,
		Assigned: 5,
	}

	if state.Delete() != 2 {
		t.Fatalf("expected delete=2, got %d", state.Delete())
	}

	if state.Extra() != 3 {
		t.Fatalf("expected extra=3, got %d", state.Extra())
	}
}

func TestReplicaStateDeleteWithPending(t *testing.T) {
	state := ReplicaState{
		Desired: 1,
		Running: 2,
		Pending: 1,
	}

	if state.Delete() != 1 {
		t.Fatalf("expected delete=1, got %d", state.Delete())
	}
}

func TestReplicaStateDeleteAll(t *testing.T) {
	state := ReplicaState{
		Desired:  0,
		Running:  3,
		Pending:  1,
		Assigned: 4,
	}

	if state.Delete() != 3 {
		t.Fatalf("expected delete=3, got %d", state.Delete())
	}
	if state.Extra() != 4 {
		t.Fatalf("expected extra=4, got %d", state.Extra())
	}
}

func TestDecideScaleDown(t *testing.T) {
	state := ReplicaState{
		Desired:  2,
		Running:  3,
		Pending:  1,
		Assigned: 4,
	}

	decision := Decide(state)

	if decision.Create != 0 {
		t.Fatalf("expected create=0, got %d", decision.Create)
	}

	if decision.Delete != 1 {
		t.Fatalf("expected delete=1, got %d", decision.Delete)
	}
}
