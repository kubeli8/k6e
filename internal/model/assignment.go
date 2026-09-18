package model

type AssignmentStatus string

const (
	AssignmentStatusPending   AssignmentStatus = "Pending"
	AssignmentStatusRunning   AssignmentStatus = "Running"
	AssignmentStatusCompleted AssignmentStatus = "Completed"
	AssignmentStatusFailed    AssignmentStatus = "Failed"
)

type Assignment struct {
	ID          string           `json:"id"`
	Workload    WorkloadRef      `json:"workload"`
	NodeID      string           `json:"nodeId"`
	Status      AssignmentStatus `json:"status"`
	ContainerID string           `json:"containerId"`
}

type WorkloadRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
