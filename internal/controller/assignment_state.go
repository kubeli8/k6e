package controller

import "github.com/pyd-07/k6e/internal/model"

type RuntimeState string

const (
	RuntimeStateRunning RuntimeState = "running"
	RuntimeStateStopped RuntimeState = "stopped"
)

func DetermineAssignmentStatus(current model.AssignmentStatus, observed RuntimeState) model.AssignmentStatus {
	if current == model.AssignmentStatusRunning &&
		observed == RuntimeStateStopped {
		return model.AssignmentStatusFailed
	}

	return current
}
