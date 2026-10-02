package controller

import (
	"testing"

	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/testutil"
)

func TestIsAssignmentObsolete(t *testing.T) {
	oldWorkload := testutil.Workload("nginx", "default")
	newWorkload := oldWorkload
	newWorkload.Spec.Template.Containers = append(
		[]model.ContainerSpec(nil),
		oldWorkload.Spec.Template.Containers...,
	)
	newWorkload.Spec.Template.Containers[0].Image = "nginx:1.21"

	currentHash := model.TemplateHash(newWorkload.Spec.Template)

	tests := []struct {
		name     string
		status   model.AssignmentStatus
		expected bool
	}{
		{name: "running assignment with changed template", status: model.AssignmentStatusRunning, expected: true},
		{name: "running assignment with same template", status: model.AssignmentStatusRunning, expected: false},
		{name: "pending assignment with changed template", status: model.AssignmentStatusPending, expected: false},
		{name: "failed assignment with changed template", status: model.AssignmentStatusFailed, expected: false},
		{name: "completed assignment with changed template", status: model.AssignmentStatusCompleted, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workload := oldWorkload

			assignment := testutil.AssignmentForWorkload(
				workload,
				"node-1",
				"container-1",
				tt.status,
			)

			hash := currentHash
			if tt.name == "running assignment with same template" {
				hash = model.TemplateHash(workload.Spec.Template)
			}

			if got := IsAssignmentObsolete(assignment, hash); got != tt.expected {
				t.Errorf("IsAssignmentObsolete() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
