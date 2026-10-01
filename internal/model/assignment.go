package model

// AssignmentStatus represents the lifecycle phase of an Assignment.
type AssignmentStatus string

const (
	// AssignmentStatusPending indicates the assignment has been created and
	// persisted but the container has not yet been started on the target node.
	AssignmentStatusPending AssignmentStatus = "Pending"
	// AssignmentStatusRunning indicates the container is running on the target node.
	AssignmentStatusRunning AssignmentStatus = "Running"
	// AssignmentStatusCompleted indicates the assignment was deliberately stopped
	// and removed, typically as part of a scale-down operation.
	AssignmentStatusCompleted AssignmentStatus = "Completed"
	// AssignmentStatusFailed indicates the container exited unexpectedly or could
	// not be started. Failed assignments are counted as missing replicas and will
	// be replaced by the next reconciliation pass.
	AssignmentStatusFailed AssignmentStatus = "Failed"
)

// Assignment records the scheduling decision that places one workload replica
// on a specific node. It tracks the container lifecycle from creation through
// running to termination.
type Assignment struct {
	ID          string           `json:"id"`
	Workload    WorkloadRef      `json:"workload"`
	NodeID      string           `json:"nodeId"`
	Status      AssignmentStatus `json:"status"`
	ContainerID string           `json:"containerId"`
	// TemplateHash is the hash of the PodTemplateSpec at the time the assignment
	// was created. It is compared against the current workload template to detect
	// stale assignments during rollout.
	TemplateHash string `json:"templateHash"`
}

// WorkloadRef identifies a Workload by its namespace and name.
type WorkloadRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
