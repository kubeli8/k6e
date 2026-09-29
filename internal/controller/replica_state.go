package controller

import "github.com/pyd-07/k6e/internal/model"

type ReplicaState struct {
	Desired  int
	Assigned int
	Pending  int
	Running  int
	Failed   int
}

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

func (s ReplicaState) Missing() int {
	missing := s.Desired - (s.Running + s.Pending)
	if missing < 0 {
		return 0
	}
	return missing
}

func (s ReplicaState) Extra() int {
	extra := (s.Running + s.Pending) - s.Desired
	if extra < 0 {
		return 0
	}
	return extra
}
