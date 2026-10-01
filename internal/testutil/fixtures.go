// Package testutil provides fresh domain fixtures for tests.
package testutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/pyd-07/k6e/internal/model"
)

// Workload returns a standard workload fixture with a fresh container slice.
func Workload(name, namespace string) model.Workload {
	return WorkloadWithReplicas(name, namespace, 2)
}

// WorkloadWithReplicas returns a standard workload fixture with the requested replica count.
func WorkloadWithReplicas(name, namespace string, replicas int32) model.Workload {
	return model.Workload{
		APIVersion: "k6e.io/v1alpha1",
		Kind:       "Workload",
		Metadata: model.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: model.WorkloadSpec{
			Replicas: replicas,
			Template: model.PodTemplateSpec{
				Containers: []model.ContainerSpec{{Name: "nginx", Image: "nginx:latest"}},
			},
		},
	}
}

// WorkloadRef returns a reference for a workload name and namespace.
func WorkloadRef(name, namespace string) model.WorkloadRef {
	return model.WorkloadRef{Name: name, Namespace: namespace}
}

// Node returns a node fixture without a heartbeat.
func Node(id, address string, status model.NodeStatus) model.Node {
	return model.Node{ID: id, Address: address, Status: status}
}

// NodeAt returns a node fixture with the supplied heartbeat time.
func NodeAt(id, address string, status model.NodeStatus, heartbeat time.Time) model.Node {
	node := Node(id, address, status)
	node.LastHeartbeat = heartbeat
	return node
}

// Assignment returns an assignment fixture for a workload reference.
func Assignment(name, namespace, nodeID, containerID string, status model.AssignmentStatus) model.Assignment {
	return model.Assignment{
		ID:          uuid.NewString(),
		Workload:    WorkloadRef(name, namespace),
		NodeID:      nodeID,
		Status:      status,
		ContainerID: containerID,
	}
}

// AssignmentForWorkload returns an assignment whose template hash matches workload.
func AssignmentForWorkload(workload model.Workload, nodeID, containerID string, status model.AssignmentStatus) model.Assignment {
	assignment := Assignment(workload.Metadata.Name, workload.Metadata.Namespace, nodeID, containerID, status)
	assignment.TemplateHash = model.TemplateHash(workload.Spec.Template)
	return assignment
}
