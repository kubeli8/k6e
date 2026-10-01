// Package model defines the core domain types for k6e.
package model

// Workload is the primary scheduling unit in k6e. It describes the desired
// number of running container replicas and the template used to create each one.
type Workload struct {
	APIVersion string       `json:"apiVersion"`
	Kind       string       `json:"kind"`
	Metadata   ObjectMeta   `json:"metadata"`
	Spec       WorkloadSpec `json:"spec"`
}

// ObjectMeta carries the identity fields common to all k6e resources.
type ObjectMeta struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// WorkloadSpec describes the desired state of a Workload: how many replicas
// should be running and the template used to create each container.
type WorkloadSpec struct {
	Replicas int32           `json:"replicas"`
	Template PodTemplateSpec `json:"template"`
}

// PodTemplateSpec holds the container specifications for a single replica.
// k6e currently requires exactly one container per replica.
type PodTemplateSpec struct {
	Containers []ContainerSpec `json:"containers"`
}

// ContainerSpec describes a single container to be run on a node.
type ContainerSpec struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}
