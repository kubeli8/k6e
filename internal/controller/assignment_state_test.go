package controller

import (
	"testing"

	"github.com/kubeli8/k6e/internal/model"
)

func TestDetermineAssignmentStatusRunning(t *testing.T) {
	got := DetermineAssignmentStatus(model.AssignmentStatusRunning, RuntimeStateRunning)

	if got != model.AssignmentStatusRunning {
		t.Fatalf("expected running, got %v", got)
	}
}

func TestDetermineAssignmentStatusStopped(t *testing.T) {
	got := DetermineAssignmentStatus(model.AssignmentStatusRunning, RuntimeStateStopped)

	if got != model.AssignmentStatusFailed {
		t.Fatalf("expected failed, got %v", got)
	}
}

func TestDetermineAssignmentStatusPending(t *testing.T) {
	got := DetermineAssignmentStatus(model.AssignmentStatusPending, RuntimeStateStopped)

	if got != model.AssignmentStatusPending {
		t.Fatalf("expected pending, got %v", got)
	}
}

func TestDetermineAssignmentStatusFailed(t *testing.T) {
	got := DetermineAssignmentStatus(model.AssignmentStatusFailed, RuntimeStateStopped)

	if got != model.AssignmentStatusFailed {
		t.Fatalf("expected failed, got %v", got)
	}
}
