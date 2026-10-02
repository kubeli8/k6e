package controller

import "github.com/kubeli8/k6e/internal/model"

// RuntimeState represents the observed liveness of a container as reported by
// the node agent.
type RuntimeState string

const (
	RuntimeStateRunning RuntimeState = "running"
	RuntimeStateStopped RuntimeState = "stopped"
)

// DetermineAssignmentStatus maps a (current status, observed runtime state)
// pair to the next assignment status. The only transition it implements is:
// Running + stopped → Failed. All other combinations leave the status unchanged.
func DetermineAssignmentStatus(current model.AssignmentStatus, observed RuntimeState) model.AssignmentStatus {
	if current == model.AssignmentStatusRunning &&
		observed == RuntimeStateStopped {
		return model.AssignmentStatusFailed
	}

	return current
}
