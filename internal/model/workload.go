package model

type Workload struct {
	APIVersion string       `json:"apiVersion"`
	Kind       string       `json:"kind"`
	Metadata   ObjectMeta   `json:"metadata"`
	Spec       WorkloadSpec `json:"spec"`
}

type ObjectMeta struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type WorkloadSpec struct {
	Replicas int32           `json:"replicas"`
	Template PodTemplateSpec `json:"template"`
}

type PodTemplateSpec struct {
	Containers []ContainerSpec `json:"containers"`
}

type ContainerSpec struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}
