package controller

import "github.com/pyd-07/k6e/internal/model"

// ReplicaState summarizes the current assignment counts for a single workload.
// It is produced by CalculateReplicaState and consumed by Decide.
type ReplicaState struct {
	Desired  int
	Assigned int
	Pending  int
	Running  int
	Failed   int
}

// CalculateReplicaState derives the ReplicaState for workload from its
// complete list of assignments. Completed assignments are not counted
// in any of the Pending, Running, or Failed totals.
func CalculateReplicaState(workload model.Workload, assignments []model.Assignment) ReplicaState {
	state := ReplicaState{
		Desired:  int(workload.Spec.Replicas),
		Assigned: len(assignments),
	}
	for _, assignment := range assignments {
		switch assignment.Status {
		case model.AssignmentStatusPending:
			state.Pending++
		case model.AssignmentStatusRunning:
			state.Running++
		case model.AssignmentStatusFailed:
			state.Failed++
		}
	}
	return state
}

// Missing returns the number of additional replicas that need to be created.
// Both Running and Pending assignments count toward satisfying the desired
// count; Failed assignments do not.
func (s ReplicaState) Missing() int {
	missing := max(s.Desired-(s.Running+s.Pending), 0)
	return missing
}

// Extra returns the total number of active replicas (Running + Pending) above
// the desired count.
func (s ReplicaState) Extra() int {
	extra := max(0, (s.Running+s.Pending)-s.Desired)
	return extra
}

// Delete returns the number of running replicas that must be stopped to reach
// the desired count. Only Running assignments are deleted; Pending assignments
// are left to complete their startup sequence.
func (s ReplicaState) Delete() int {
	delete := max(0, s.Running-s.Desired)
	return delete
}
